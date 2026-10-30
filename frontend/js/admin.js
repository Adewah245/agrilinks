document.addEventListener("DOMContentLoaded", () => {
    const icon = name => {
        const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        const use = document.createElementNS("http://www.w3.org/2000/svg", "use");
        svg.classList.add("a-icon");
        svg.setAttribute("aria-hidden", "true");
        use.setAttribute("href", `#a-${name}`);
        svg.append(use);
        return svg;
    };
    const initials = name => {
        const names = name.trim().split(/\s+/);
        return [names[0], ...(names.length > 1 ? [names[names.length - 1]] : [])].map(part => Array.from(part)[0] || "").join("").toUpperCase();
    };
    const owner = document.querySelector(".sidebar-member strong")?.textContent.trim();
    if (owner) document.querySelectorAll("[data-admin-initials]").forEach(avatar => { avatar.textContent = initials(owner); });
    const today = new Date();
    const date = document.getElementById("admin-date");
    if (date) {
        date.dateTime = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}`;
        date.textContent = today.toLocaleDateString("en-GB", { weekday: "short", day: "numeric", month: "short", year: "numeric" });
    }
    const greeting = document.getElementById("admin-greeting");
    if (greeting) greeting.textContent = today.getHours() < 12 ? "Good morning" : today.getHours() < 18 ? "Good afternoon" : "Good evening";
    const groupDecimal = value => {
        const [whole, fraction] = value.split(".");
        return whole.replace(/\B(?=(\d{3})+(?!\d))/g, ",") + (fraction === undefined ? "" : `.${fraction}`);
    };
    document.querySelectorAll("[data-metric-value]").forEach(element => {
        const value = element.textContent.trim();
        element.textContent = /^\d+(?:\.\d+)?$/.test(value) ? groupDecimal(value) : value.replace(/\b([A-Z]{3}) (\d+(?:\.\d+)?)/g, (_, currency, amount) => `${currency} ${groupDecimal(amount)}`);
    });
    document.querySelectorAll(".attention-item").forEach(item => {
        item.classList.toggle("is-clear", item.querySelector("[data-metric-value]")?.textContent.trim() === "0");
    });

    const sidebar = document.querySelector(".admin-sidebar");
    const menuToggle = document.querySelector(".admin-menu-toggle");
    if (sidebar && menuToggle) {
        const setOpen = open => {
            sidebar.classList.toggle("admin-nav-open", open);
            menuToggle.setAttribute("aria-expanded", String(open));
            menuToggle.setAttribute("aria-label", open ? "Close navigation" : "Open navigation");
        };
        sidebar.classList.add("admin-nav-enhanced");
        menuToggle.addEventListener("click", () => setOpen(menuToggle.getAttribute("aria-expanded") !== "true"));
        sidebar.querySelector("nav")?.addEventListener("click", event => { if (event.target.closest("a")) setOpen(false); });
        document.addEventListener("click", event => { if (!sidebar.contains(event.target)) setOpen(false); });
        document.addEventListener("keydown", event => {
            if (event.key === "Escape" && menuToggle.getAttribute("aria-expanded") === "true") {
                setOpen(false);
                menuToggle.focus();
            }
        });
        window.matchMedia("(min-width: 901px)").addEventListener("change", () => setOpen(false));
    }

    const finder = document.getElementById("admin-workspace-finder");
    const finderInput = document.getElementById("workspace-finder-search");
    const finderTrigger = document.querySelector(".workspace-finder-trigger");
    if (finder && finderInput && finderTrigger && typeof finder.showModal === "function") {
        const links = [...finder.querySelectorAll(".finder-results a")];
        const filterWorkspaces = () => {
            const query = finderInput.value.trim().toLocaleLowerCase();
            links.forEach(link => { link.hidden = !link.textContent.toLocaleLowerCase().includes(query); });
            finder.querySelector(".finder-empty").hidden = links.some(link => !link.hidden);
        };
        const openFinder = () => {
            if (finder.open) return;
            finderInput.value = "";
            filterWorkspaces();
            finder.showModal();
            finderInput.focus();
        };
        const closeFinder = () => finder.close();
        finderTrigger.addEventListener("click", openFinder);
        finder.querySelector(".finder-close").addEventListener("click", closeFinder);
        finderInput.addEventListener("input", filterWorkspaces);
        finder.addEventListener("click", event => {
            const bounds = finder.getBoundingClientRect();
            if (event.target === finder && (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom)) closeFinder();
        });
        document.addEventListener("keydown", event => {
            if ((event.metaKey || event.ctrlKey) && !event.altKey && event.key.toLowerCase() === "k") {
                event.preventDefault();
                if (finder.open) closeFinder(); else openFinder();
            }
        });
        finder.addEventListener("keydown", event => {
            const visible = links.filter(link => !link.hidden);
            if (event.key === "ArrowDown" || event.key === "ArrowUp") {
                event.preventDefault();
                const current = visible.indexOf(document.activeElement);
                if (event.key === "ArrowDown") visible[(current + 1) % visible.length]?.focus();
                else if (current <= 0) finderInput.focus(); else visible[current - 1]?.focus();
            } else if (event.key === "Enter" && document.activeElement === finderInput && visible.length) {
                event.preventDefault();
                visible[0].click();
            }
        });
        if (/Mac|iPhone|iPad/.test(navigator.platform)) finderTrigger.querySelector("[data-shortcut-label]").textContent = "⌘ K";
    } else if (finderTrigger) {
        finderTrigger.remove();
    }

    // Sorting and filtering are explicitly scoped to the server-rendered page.
    const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: "base" });
    const compareValues = (a, b) => {
        const numeric = /^-?\d[\d,]*(?:\.\d+)?$/;
        if (numeric.test(a) && numeric.test(b)) {
            const cleanA = a.replaceAll(",", "");
            const cleanB = b.replaceAll(",", "");
            const precision = Math.max((cleanA.split(".")[1] || "").length, (cleanB.split(".")[1] || "").length);
            const exact = value => {
                const [whole, fraction = ""] = value.split(".");
                const minor = BigInt(`${whole.replace("-", "")}${fraction.padEnd(precision, "0")}`);
                return whole.startsWith("-") ? -minor : minor;
            };
            const first = exact(cleanA), second = exact(cleanB);
            return first < second ? -1 : first > second ? 1 : 0;
        }
        return collator.compare(a, b);
    };
    const badgeColumns = new Set(["status", "account status", "verification", "access", "priority", "result", "delivery status"]);
    const personColumns = new Set(["name", "member", "buyer", "seller", "owner", "sender", "recipient", "actor", "staff", "staff member", "assigned staff", "reviewed by", "approved by"]);
    const numberColumns = new Set(["balance", "amount", "unit price", "gross payment", "commission", "successful sales", "recorded refunds", "new members", "farmers", "buyers", "listings", "active", "active listings", "sold out", "tickets", "open", "resolved / closed", "deliveries", "payment attempts", "failed attempts"]);
    document.querySelectorAll(".records-panel").forEach(panel => {
        const table = panel.querySelector("table");
        if (!table) return;
        const body = table.tBodies[0];
        const rows = [...body.rows];
        const headers = [...table.querySelectorAll("thead th[data-column]")];
        rows.forEach(row => headers.forEach(header => {
            const column = Number(header.dataset.column);
            const cell = row.cells[column];
            if (!cell) return;
            const value = cell.textContent.trim();
            cell.dataset.sortValue = value;
            const name = header.textContent.trim().toLowerCase();
            if (badgeColumns.has(name) && value && value !== "—") {
                const badge = document.createElement("span");
                badge.className = "status-badge";
                badge.dataset.value = value.toLowerCase().replaceAll(" ", "_");
                badge.textContent = value.replaceAll("_", " ").replace(/^./, letter => letter.toUpperCase());
                cell.replaceChildren(badge);
            } else if (personColumns.has(name) && value && !/^(unassigned|awaiting approval|organization #|—)/i.test(value)) {
                const person = document.createElement("span");
                const avatar = document.createElement("span");
                const text = document.createElement("span");
                person.className = "table-person";
                avatar.className = "table-person-avatar";
                avatar.setAttribute("aria-hidden", "true");
                avatar.textContent = initials(value.replace(/\s*\(#\d+\)$/, ""));
                text.textContent = value;
                person.append(avatar, text);
                cell.replaceChildren(person);
            } else if (numberColumns.has(name) && /^-?\d+(?:\.\d+)?$/.test(value)) {
                cell.textContent = groupDecimal(value);
            }
        }));
        headers.forEach(header => {
            const label = header.textContent.trim();
            const button = document.createElement("button");
            const text = document.createElement("span");
            button.type = "button";
            button.className = "table-sort";
            button.title = `Sort this page by ${label}`;
            text.textContent = label;
            button.append(text, icon("sort"));
            header.replaceChildren(button);
            header.setAttribute("aria-sort", "none");
            button.addEventListener("click", () => {
                const descending = header.getAttribute("aria-sort") === "ascending";
                headers.forEach(other => other.setAttribute("aria-sort", "none"));
                header.setAttribute("aria-sort", descending ? "descending" : "ascending");
                const column = Number(header.dataset.column);
                const sorted = [...body.rows].sort((a, b) => {
                    const order = compareValues(a.cells[column]?.dataset.sortValue || "", b.cells[column]?.dataset.sortValue || "");
                    return descending ? -order : order;
                });
                // Move nodes so open row forms, tokens, and unsaved review notes survive sorting.
                sorted.forEach(row => body.append(row));
            });
        });
        const statuses = [...new Set(rows.map(row => row.dataset.status.trim()).filter(Boolean))].sort(collator.compare);
        const filter = panel.querySelector("[data-table-status]");
        if (filter && statuses.length) {
            statuses.forEach(status => filter.add(new Option(status.replaceAll("_", " ").replace(/^./, c => c.toUpperCase()), status)));
            const applyFilter = () => {
                let visible = 0;
                rows.forEach(row => {
                    row.hidden = Boolean(filter.value && row.dataset.status.trim() !== filter.value);
                    if (!row.hidden) visible++;
                });
                panel.querySelector(".record-count").textContent = filter.value ? `${visible} of ${rows.length} on this page` : `${rows.length} records on this page`;
                panel.querySelector(".filtered-empty").hidden = visible !== 0;
                panel.querySelector(".table-scroll").hidden = visible === 0;
            };
            filter.addEventListener("change", applyFilter);
            panel.querySelector("[data-reset-table]")?.addEventListener("click", () => { filter.value = ""; applyFilter(); });
        } else {
            filter?.closest(".record-status-filter")?.remove();
        }
    });

    const jump = document.getElementById("admin-table-jump");
    jump?.addEventListener("change", () => {
        const panel = document.getElementById(jump.value);
        if (!panel) return;
        panel.setAttribute("tabindex", "-1");
        panel.scrollIntoView({ block: "start", behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "instant" : "smooth" });
        panel.focus({ preventScroll: true });
    });
    document.body.classList.add("admin-enhanced");
});
