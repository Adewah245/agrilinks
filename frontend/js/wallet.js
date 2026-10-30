document.addEventListener("DOMContentLoaded", () => {
    document.body.classList.add("wallet-enhanced");

    const toast = document.getElementById("wallet-toast");
    let toastTimer;
    const announce = message => {
        if (!toast) return;
        clearTimeout(toastTimer);
        toast.textContent = message;
        toast.classList.add("is-visible");
        toastTimer = setTimeout(() => toast.classList.remove("is-visible"), 3500);
    };
    const readPreference = key => {
        try { return localStorage.getItem(key); } catch { return null; }
    };
    const savePreference = (key, value) => {
        try { localStorage.setItem(key, value); } catch { /* Personalization also works without storage. */ }
    };

    const date = document.getElementById("wallet-date");
    if (date) {
        const today = new Date();
        date.dateTime = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}`;
        date.textContent = today.toLocaleDateString("en-GB", { weekday: "short", day: "numeric", month: "short", year: "numeric" });
    }
    const owner = document.querySelector(".wallet-card-owner strong")?.textContent.trim();
    if (owner) {
        const names = owner.split(/\s+/);
        const initials = [names[0], ...(names.length > 1 ? [names[names.length - 1]] : [])].map(name => Array.from(name)[0]).join("").toUpperCase();
        document.querySelectorAll("[data-initials]").forEach(avatar => {
            if (!avatar.querySelector("img")) avatar.textContent = initials;
        });
    }

    // Group decimal strings without rounding or converting financial values to floats.
    const formatAmount = value => {
        if (!/^-?\d+(?:\.\d{1,2})?$/.test(value)) return value;
        const [whole, fraction = "00"] = value.split(".");
        return `${whole.replace(/\B(?=(\d{3})+(?!\d))/g, ",")}.${fraction.padEnd(2, "0")}`;
    };
    const formatMoney = (value, currency) => `${currency === "NGN" ? "₦" : `${currency} `}${formatAmount(value)}`;
    document.querySelectorAll("[data-money]").forEach(element => {
        element.textContent = formatMoney(element.dataset.money, element.dataset.moneyCurrency);
    });
    const balance = document.getElementById("wallet-balance");
    const balanceValue = document.getElementById("wallet-balance-value");
    const visibilityToggle = document.getElementById("wallet-balance-toggle");
    const setBalanceHidden = hidden => {
        if (!balance || !balanceValue || !visibilityToggle) return;
        balanceValue.textContent = hidden ? "••••••" : formatAmount(balance.dataset.balance);
        balance.setAttribute("aria-label", hidden ? "Balance hidden" : `Available balance: ${formatMoney(balance.dataset.balance, balance.dataset.currency)}`);
        visibilityToggle.setAttribute("aria-pressed", String(hidden));
        visibilityToggle.setAttribute("aria-label", hidden ? "Show balance" : "Hide balance");
        visibilityToggle.querySelector("use")?.setAttribute("href", hidden ? "#w-eye-off" : "#w-eye");
        document.querySelectorAll(".wallet-history-money").forEach(element => {
            element.textContent = hidden ? "••••" : formatMoney(element.dataset.money, element.dataset.moneyCurrency);
        });
    };
    setBalanceHidden(readPreference("agro-shield-wallet-balance-hidden") === "true");
    visibilityToggle?.addEventListener("click", () => {
        const hidden = visibilityToggle.getAttribute("aria-pressed") !== "true";
        setBalanceHidden(hidden);
        savePreference("agro-shield-wallet-balance-hidden", String(hidden));
    });

    const card = document.getElementById("wallet-metal-card");
    const finishLabel = document.getElementById("wallet-card-finish");
    const finishButtons = document.querySelectorAll("[data-wallet-finish]");
    const setFinish = finish => {
        if (!card || !["gold", "silver", "bronze"].includes(finish)) return;
        card.dataset.finish = finish;
        if (finishLabel) finishLabel.textContent = `${finish.toUpperCase()} EDITION`;
        finishButtons.forEach(button => button.setAttribute("aria-pressed", String(button.dataset.walletFinish === finish)));
    };
    setFinish(readPreference("agro-shield-wallet-finish") || "gold");
    finishButtons.forEach(button => button.addEventListener("click", () => {
        const finish = button.dataset.walletFinish;
        setFinish(finish);
        savePreference("agro-shield-wallet-finish", finish);
        announce(`${finish.charAt(0).toUpperCase() + finish.slice(1)} finish selected.`);
    }));

    document.querySelectorAll("[data-copy-target]").forEach(button => button.addEventListener("click", async () => {
        const target = document.getElementById(button.dataset.copyTarget);
        if (!target) return;
        try {
            if (!navigator.clipboard?.writeText) throw new Error("Clipboard unavailable");
            await navigator.clipboard.writeText(target.textContent.trim());
            announce(button.dataset.copyTarget === "wallet-number" ? "Wallet number copied." : "Bank account number copied.");
        } catch {
            const selection = window.getSelection();
            const range = document.createRange();
            range.selectNodeContents(target);
            selection?.removeAllRanges();
            selection?.addRange(range);
            announce("Number selected. Press and hold, or use your browser's copy command.");
        }
    }));

    const filters = document.querySelectorAll("[data-history-filter]");
    const historySections = document.querySelectorAll("[data-history]");
    const showHistory = category => {
        filters.forEach(button => button.setAttribute("aria-pressed", String(button.dataset.historyFilter === category)));
        historySections.forEach(section => { section.hidden = section.dataset.history !== category; });
    };
    if (filters.length) {
        showHistory("balance");
        filters.forEach(button => button.addEventListener("click", () => showHistory(button.dataset.historyFilter)));
    }

    const openMove = id => {
        const panel = document.getElementById(id);
        if (!(panel instanceof HTMLDetailsElement)) return null;
        panel.open = true;
        return panel;
    };
    document.querySelectorAll('a[href^="#wallet-"]').forEach(link => link.addEventListener("click", () => {
        const panel = openMove(link.hash.slice(1));
        if (panel) panel.querySelector("summary")?.focus({ preventScroll: true });
    }));
    openMove(window.location.hash.slice(1));
    window.addEventListener("hashchange", () => openMove(window.location.hash.slice(1)));

    // Keep the selected form and history category through the server's POST redirect.
    const movePanels = document.querySelectorAll(".wallet-move-panel");
    const actionCategories = { deposit: "funding", transfer: "transfers", withdraw: "withdrawals", cancel_transfer: "transfers", cancel_withdrawal: "withdrawals" };
    const storedActionKey = "agro-shield-wallet-last-action";
    document.querySelectorAll('form[action="/wallet"]').forEach(form => form.addEventListener("submit", () => {
        try {
            sessionStorage.setItem(storedActionKey, JSON.stringify({ panel: form.closest(".wallet-move-panel")?.id, category: actionCategories[form.elements.namedItem("action")?.value], time: Date.now() }));
        } catch { /* Forms still submit normally without storage. */ }
    }));
    try {
        const action = JSON.parse(sessionStorage.getItem(storedActionKey) || "null");
        sessionStorage.removeItem(storedActionKey);
        const notice = new URLSearchParams(window.location.search).get("notice");
        if (action && Date.now() - action.time < 15 * 60 * 1000 && (notice || document.querySelector(".wallet-alert-error"))) {
            const panel = openMove(action.panel);
            if (action.category) showHistory(action.category);
            if (panel) {
                movePanels.forEach(other => { if (other !== panel) other.open = false; });
                const code = panel.querySelector('input[autocomplete="one-time-code"]');
                if (notice === "code_sent" && code) {
                    panel.scrollIntoView({ block: "start" });
                    code.focus({ preventScroll: true });
                }
            }
        }
    } catch { /* A disabled or stale storage entry must not interrupt the wallet. */ }

    const depositInput = document.getElementById("wallet-deposit-amount");
    document.querySelectorAll("[data-deposit-amount]").forEach(button => button.addEventListener("click", () => {
        if (!depositInput) return;
        depositInput.value = button.dataset.depositAmount;
        depositInput.focus({ preventScroll: true });
    }));

    const sidebar = document.querySelector(".wallet-sidebar");
    const menuToggle = document.querySelector(".wallet-menu-toggle");
    if (sidebar && menuToggle) {
        sidebar.classList.add("wallet-nav-enhanced");
        const setMenuOpen = open => {
            sidebar.classList.toggle("wallet-nav-open", open);
            menuToggle.setAttribute("aria-expanded", String(open));
            menuToggle.setAttribute("aria-label", open ? "Close navigation" : "Open navigation");
        };
        menuToggle.addEventListener("click", () => setMenuOpen(menuToggle.getAttribute("aria-expanded") !== "true"));
        sidebar.querySelector("nav")?.addEventListener("click", event => {
            if (event.target.closest("a")) setMenuOpen(false);
        });
        document.addEventListener("click", event => { if (!sidebar.contains(event.target)) setMenuOpen(false); });
        document.addEventListener("keydown", event => {
            if (event.key === "Escape" && menuToggle.getAttribute("aria-expanded") === "true") {
                setMenuOpen(false);
                menuToggle.focus();
            }
        });
        window.matchMedia("(min-width: 1001px)").addEventListener("change", () => setMenuOpen(false));
    }

    const select = document.getElementById("wallet-bank");
    if (select) {
        const saveBank = document.getElementById("wallet-save-bank");
        const loadBanks = async () => {
            select.disabled = true;
            if (saveBank) saveBank.disabled = true;
            try {
                const response = await fetch("/wallet/banks", { headers: { Accept: "application/json" } });
                if (!response.ok || response.redirected) throw new Error("Could not load the bank list. Please reload the page.");
                const banks = await response.json();
                if (!Array.isArray(banks) || !banks.length || banks.some(bank => typeof bank.name !== "string" || typeof bank.code !== "string")) throw new Error("The bank list is unavailable. Please reload the page.");
                select.replaceChildren(new Option("Choose your bank", ""));
                banks.forEach(bank => select.add(new Option(bank.name, bank.code)));
                select.disabled = false;
                if (saveBank) saveBank.disabled = false;
            } catch (error) {
                select.replaceChildren(new Option("Bank list unavailable", ""));
                const errorMessage = document.getElementById("bank-error");
                if (errorMessage) errorMessage.textContent = error.message || "Could not load the bank list. Please reload the page.";
            }
        };
        loadBanks();
    }
});
