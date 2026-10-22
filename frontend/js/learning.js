"use strict";

document.querySelectorAll("[data-learning-form]").forEach((form) => {
    form.addEventListener("submit", async (event) => {
        event.preventDefault();
        const button = form.querySelector('button[type="submit"]');
        const error = document.getElementById("learning-error");
        const status = form.querySelector(".form-status");
        button.disabled = true;
        error.hidden = true;
        if (status) status.textContent = "Your guide is working on your question…";
        try {
            const response = await fetch(form.action, {
                method: "POST", body: new URLSearchParams(new FormData(form)),
                headers: { Accept: "application/json" }, credentials: "same-origin"
            });
            if (response.redirected) { window.location.assign(response.url); return; }
            const result = await response.json();
            if (!response.ok || !result.success) throw new Error(result.error || "We could not save that change. Please try again.");
            window.location.reload();
        } catch (problem) {
            error.textContent = problem.message || "Please check your connection and try again.";
            error.hidden = false;
            error.scrollIntoView({ block: "nearest" });
        } finally {
            button.disabled = false;
            if (status) status.textContent = "";
        }
    });
});
const conversation = document.querySelector(".learning-messages");
if (conversation) conversation.scrollTop = conversation.scrollHeight;
