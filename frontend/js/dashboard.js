/* ============================================================
   AGRO-SHIELD DASHBOARD JAVASCRIPT
   ============================================================ */

document.addEventListener("DOMContentLoaded", () => {

    /* ========================================================
       ELEMENT REFERENCES
    ======================================================== */

    const todayElement = document.getElementById("today");
    const currentTimeElement = document.getElementById("currentTime");
    const greetingElement = document.getElementById("greeting");

    const counters = document.querySelectorAll(".counter");
    const progressBars = document.querySelectorAll(".progress-value");


    /* ========================================================
       DATE + TIME
    ======================================================== */

    function updateDateTime() {

        const now = new Date();

        /* -----------------------------------------------
           Greeting
        ------------------------------------------------ */

        if (greetingElement) {

            const hour = now.getHours();

            let greeting = "Welcome back";

            if (hour >= 5 && hour < 12) {
                greeting = "Good morning";
            } else if (hour >= 12 && hour < 17) {
                greeting = "Good afternoon";
            } else if (hour >= 17 && hour < 21) {
                greeting = "Good evening";
            } else {
                greeting = "Good night";
            }

            greetingElement.textContent = greeting;
        }


        /* -----------------------------------------------
           Date
        ------------------------------------------------ */

        if (todayElement) {

            const dateText = now.toLocaleDateString("en-NG", {
                weekday: "long",
                day: "numeric",
                month: "long",
                year: "numeric"
            });

            todayElement.textContent = dateText;
        }


        /* -----------------------------------------------
           Current Time
        ------------------------------------------------ */

        if (currentTimeElement) {

            const timeText = now.toLocaleTimeString("en-NG", {
                hour: "numeric",
                minute: "2-digit",
                second: "2-digit",
                hour12: true
            });

            currentTimeElement.textContent = timeText;
        }
    }


    /* Run immediately */
    updateDateTime();


    /* Update every second */
    setInterval(updateDateTime, 1000);


    /* ========================================================
       COUNTER ANIMATION
    ======================================================== */

    function animateCounter(element) {

        const target = Number(element.dataset.target);

        if (!Number.isFinite(target)) {
            return;
        }

        /* Don't animate negative values */
        const finalValue = Math.max(0, target);

        /* Already at target */
        if (finalValue === 0) {
            element.textContent = "0";
            return;
        }

        const duration = 1000;
        const startTime = performance.now();

        function updateCounter(currentTime) {

            const elapsed = currentTime - startTime;

            const progress = Math.min(
                elapsed / duration,
                1
            );

            /*
             * Ease-out animation.
             * Starts fast and slows down near the final number.
             */
            const easedProgress =
                1 - Math.pow(1 - progress, 3);

            const currentValue = Math.floor(
                easedProgress * finalValue
            );

            element.textContent =
                currentValue.toLocaleString("en-NG");

            if (progress < 1) {

                requestAnimationFrame(updateCounter);

            } else {

                element.textContent =
                    finalValue.toLocaleString("en-NG");
            }
        }

        requestAnimationFrame(updateCounter);
    }


    /* ========================================================
       START COUNTERS WHEN VISIBLE
    ======================================================== */

    if (counters.length > 0) {

        /*
         * IntersectionObserver prevents animations from
         * running unnecessarily before the cards are visible.
         */
        if ("IntersectionObserver" in window) {

            const counterObserver =
                new IntersectionObserver(
                    (entries, observer) => {

                        entries.forEach(entry => {

                            if (!entry.isIntersecting) {
                                return;
                            }

                            animateCounter(entry.target);

                            observer.unobserve(entry.target);
                        });

                    },
                    {
                        threshold: 0.25
                    }
                );


            counters.forEach(counter => {
                counterObserver.observe(counter);
            });

        } else {

            /* Fallback for older browsers */
            counters.forEach(counter => {
                animateCounter(counter);
            });
        }
    }


    /* ========================================================
       HEALTH PROGRESS BARS
    ======================================================== */

    function initializeProgressBars() {

        progressBars.forEach(bar => {

            const width = bar.dataset.width;

            if (!width) {
                return;
            }

            /*
             * Start from zero so the bar can animate.
             */
            bar.style.width = "0";


            /*
             * Small delay allows the browser to render
             * the initial zero width before animating.
             */
            requestAnimationFrame(() => {

                requestAnimationFrame(() => {

                    bar.style.width = width;

                });

            });

        });
    }


    initializeProgressBars();


    /* ========================================================
       SMOOTH INTERNAL LINKS
    ======================================================== */

    const internalLinks =
        document.querySelectorAll(
            'a[href^="#"]'
        );

    internalLinks.forEach(link => {

        link.addEventListener("click", event => {

            const targetID =
                link.getAttribute("href");

            if (
                !targetID ||
                targetID === "#"
            ) {
                return;
            }

            const target =
                document.querySelector(targetID);

            if (!target) {
                return;
            }

            event.preventDefault();

            target.scrollIntoView({
                behavior: "smooth",
                block: "start"
            });
        });

    });


    /* ========================================================
       DASHBOARD CARD INTERACTION
    ======================================================== */

    const interactiveCards =
        document.querySelectorAll(
            ".metric-card, .action-card"
        );

    interactiveCards.forEach(card => {

        card.addEventListener("keydown", event => {

            /*
             * Allows keyboard users to activate
             * card links naturally.
             */
            if (
                event.key === "Enter" ||
                event.key === " "
            ) {

                /*
                 * Don't interfere with actual buttons
                 * or links inside the card.
                 */
                if (
                    event.target.tagName === "A" ||
                    event.target.tagName === "BUTTON"
                ) {
                    return;
                }

                event.preventDefault();

                const link =
                    card.querySelector("a");

                if (link) {
                    link.click();
                }
            }

        });

    });


    /* ========================================================
       WEATHER CARD
    ======================================================== */

    const weatherCard =
        document.querySelector(".weather-card");

    if (weatherCard) {

        /*
         * Add a small visual state when weather
         * information is unavailable.
         */
        const weatherMain =
            weatherCard.querySelector(".weather-main");

        const unavailableText =
            weatherCard.textContent
                .toLowerCase()
                .includes("weather unavailable");

        if (
            unavailableText &&
            weatherMain
        ) {
            weatherCard.classList.add(
                "weather-unavailable"
            );
        }
    }


    /* ========================================================
       RESPONSIVE DASHBOARD
    ======================================================== */

    function handleResponsiveDashboard() {

        const width = window.innerWidth;

        /*
         * Store current device mode on the document.
         * CSS can use this if needed later.
         */
        if (width <= 600) {

            document.documentElement.dataset.device =
                "mobile";

        } else if (width <= 900) {

            document.documentElement.dataset.device =
                "tablet";

        } else {

            document.documentElement.dataset.device =
                "desktop";
        }
    }


    handleResponsiveDashboard();

    window.addEventListener(
        "resize",
        handleResponsiveDashboard
    );


    /* ========================================================
       PAGE VISIBILITY
    ======================================================== */

    /*
     * When the user leaves the dashboard tab, we don't
     * need to keep doing heavy work.
     *
     * The clock will still work because updateDateTime()
     * is lightweight, but this gives us a clean hook for
     * future dashboard live updates.
     */
    document.addEventListener(
        "visibilitychange",
        () => {

            if (!document.hidden) {
                updateDateTime();
            }

        }
    );


    /* ========================================================
       ACCESSIBILITY
    ======================================================== */

    /*
     * Make clickable cards keyboard-friendly where
     * appropriate.
     */
    interactiveCards.forEach(card => {

        if (
            card.tagName !== "A" &&
            card.getAttribute("role") !== "button"
        ) {
            return;
        }

        if (!card.hasAttribute("tabindex")) {
            card.setAttribute("tabindex", "0");
        }

    });


    /* ========================================================
       FARMING TYPE SUPPORT
    ======================================================== */

    /*
     * Agro-Shield is NOT crop-only.
     *
     * The backend will eventually provide the farmer's
     * selected farming types, for example:
     *
     * - Crop
     * - Livestock
     * - Poultry
     * - Fishery
     * - Pig Farming
     * - Goat Farming
     * - Cattle Farming
     * - Beekeeping
     * - Mixed Farming
     *
     * We intentionally do not hard-code those values here.
     *
     * The dashboard HTML can later receive something like:
     *
     * data-farming-types="crop,livestock,poultry"
     *
     * and render the appropriate dashboard sections.
     */

    const dashboard =
        document.querySelector(".dashboard");

    if (dashboard) {

        const farmingTypes =
            dashboard.dataset.farmingTypes;

        if (farmingTypes) {

            const types =
                farmingTypes
                    .split(",")
                    .map(type => type.trim().toLowerCase())
                    .filter(Boolean);

            /*
             * Store the selected farming types on the
             * document for future dashboard components.
             */
            document.documentElement.dataset.farmingTypes =
                types.join(",");
        }
    }


    /* ========================================================
       INITIALIZATION COMPLETE
    ======================================================== */

    document.documentElement.classList.add(
        "dashboard-ready"
    );

});