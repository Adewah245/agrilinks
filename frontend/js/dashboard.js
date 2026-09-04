/**
 * Agro-Shield Dashboard Interactivity
 * Handles real-time clock, dynamic greeting, smooth counter animations,
 * and health progress bar expansions using modern Web APIs.
 */
document.addEventListener("DOMContentLoaded", () => {
    initClockAndGreeting();
    initAnimatedCounters();
    initProgressBars();
});

/**
 * 1. Clock & Dynamic Greeting
 * Updates time and sets contextual greeting based on local client hour.
 */
function initClockAndGreeting() {
    const timeEl = document.getElementById("currentTime");
    const greetingEl = document.getElementById("greeting");

    // Standardized cross-browser DateTime Formatter
    const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
        weekday: 'short',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        hour12: true
    });

    function updateTimeAndGreeting() {
        const now = new Date();

        if (timeEl) {
            timeEl.textContent = dateTimeFormatter.format(now);
        }

        if (greetingEl) {
            const hour = now.getHours();
            if (hour >= 5 && hour < 12) {
                greetingEl.textContent = "Good morning";
            } else if (hour >= 12 && hour < 17) {
                greetingEl.textContent = "Good afternoon";
            } else if (hour >= 17 && hour < 22) {
                greetingEl.textContent = "Good evening";
            } else {
                greetingEl.textContent = "Welcome back";
            }
        }
    }

    updateTimeAndGreeting();
    // Sync update accurately with the start of the next second
    const delayUntilNextSecond = 1000 - new Date().getMilliseconds();
    setTimeout(() => {
        updateTimeAndGreeting();
        setInterval(updateTimeAndGreeting, 1000);
    }, delayUntilNextSecond);
}

/**
 * 2. Animated Metric Counters
 * Animates numbers using requestAnimationFrame and cubic ease-out.
 * Triggers via IntersectionObserver when element scrolls into view.
 */
function initAnimatedCounters() {
    const counters = document.querySelectorAll(".counter[data-target]");
    if (!counters.length) return;

    const animateCounter = (counter) => {
        const rawTarget = counter.getAttribute("data-target");
        const target = parseFloat(rawTarget) || 0;
        
        if (target === 0) {
            counter.textContent = "0";
            return;
        }

        // Determine required decimal precision (e.g., 4.8 vs 100)
        const decimalPlaces = (rawTarget.split('.')[1] || '').length;
        const duration = 1200; // Total duration in ms
        let startTime = null;

        const step = (timestamp) => {
            if (!startTime) startTime = timestamp;
            const progress = Math.min((timestamp - startTime) / duration, 1);
            
            // Cubic Ease-Out formula for natural deceleration
            const easeOutProgress = 1 - Math.pow(1 - progress, 3);
            const currentVal = easeOutProgress * target;

            counter.textContent = currentVal.toLocaleString(undefined, {
                minimumFractionDigits: decimalPlaces,
                maximumFractionDigits: decimalPlaces
            });

            if (progress < 1) {
                requestAnimationFrame(step);
            } else {
                counter.textContent = target.toLocaleString(undefined, {
                    minimumFractionDigits: decimalPlaces,
                    maximumFractionDigits: decimalPlaces
                });
            }
        };

        requestAnimationFrame(step);
    };

    // Trigger animations only when visible
    if ('IntersectionObserver' in window) {
        const observer = new IntersectionObserver((entries, obs) => {
            entries.forEach((entry) => {
                if (entry.isIntersecting) {
                    animateCounter(entry.target);
                    obs.unobserve(entry.target);
                }
            });
        }, { threshold: 0.2 });

        counters.forEach((counter) => observer.observe(counter));
    } else {
        // Fallback for legacy environments
        counters.forEach(animateCounter);
    }
}

/**
 * 3. Health Score Progress Bar Animation
 * Expands progress width smoothly when visible in viewport.
 */
function initProgressBars() {
    const progressBars = document.querySelectorAll(".progress-value[data-width]");
    if (!progressBars.length) return;

    const triggerBar = (bar) => {
        const targetWidth = bar.getAttribute("data-width");
        if (targetWidth) {
            bar.style.width = targetWidth;
        }
    };

    if ('IntersectionObserver' in window) {
        const observer = new IntersectionObserver((entries, obs) => {
            entries.forEach((entry) => {
                if (entry.isIntersecting) {
                    triggerBar(entry.target);
                    obs.unobserve(entry.target);
                }
            });
        }, { threshold: 0.1 });

        progressBars.forEach((bar) => observer.observe(bar));
    } else {
        setTimeout(() => progressBars.forEach(triggerBar), 150);
    }
}