document.addEventListener("DOMContentLoaded", () => {
    document.querySelectorAll(".navbar").forEach((navbar, index) => {
        const nav = navbar.querySelector("nav");
        if (!nav) return;
        const host = nav.parentElement;
        const toggle = navbar.querySelector(".nav-toggle") || document.createElement("button");
        const quickActions = document.createElement("div");
        quickActions.className = "nav-quick-actions";
        [["/cart", "Cart", "🛒"], ["/notifications", "Notifications", "🔔"]].forEach(([href, label, icon]) => {
            if (!nav.querySelector(`a[href="${href}"]`)) return;
            const link = document.createElement("a");
            link.href = href;
            link.setAttribute("aria-label", label);
            link.title = label;
            link.textContent = icon;
            quickActions.appendChild(link);
        });
        toggle.type = "button";
        toggle.className = "nav-toggle";
        nav.id ||= `main-navigation-${index}`;
        toggle.setAttribute("aria-controls", nav.id);
        if (quickActions.childElementCount && !navbar.querySelector(".nav-quick-actions")) host.insertBefore(quickActions, nav);
        host.insertBefore(toggle, nav);
        const setOpen = open => {
            navbar.classList.toggle("nav-open", open);
            toggle.setAttribute("aria-expanded", String(open));
            toggle.setAttribute("aria-label", open ? "Close navigation" : "Open navigation");
            toggle.innerHTML = `<span aria-hidden="true">${open ? "×" : "☰"}</span>`;
        };
        setOpen(false);
        navbar.classList.add("nav-enhanced");
        toggle.addEventListener("click", () => {
            setOpen(!navbar.classList.contains("nav-open"));
        });
        nav.addEventListener("click", event => {
            if (event.target.closest("a")) setOpen(false);
        });
        document.addEventListener("keydown", event => {
            if (event.key === "Escape" && navbar.classList.contains("nav-open")) {
                setOpen(false);
                toggle.focus();
            }
        });
        document.addEventListener("click", event => {
            if (!navbar.contains(event.target)) setOpen(false);
        });
        window.matchMedia("(min-width: 1101px)").addEventListener("change", () => setOpen(false));
        const updateHeight = () => document.documentElement.style.setProperty("--navbar-height", `${navbar.getBoundingClientRect().height}px`);
        updateHeight();
        if (typeof ResizeObserver !== "undefined") new ResizeObserver(updateHeight).observe(navbar);
    });
});
