"use strict";
document.addEventListener("click", async (event) => {
  const button = event.target.closest("[data-finance-code]");
  if (!button || button.disabled) return;
  const form = button.closest("form");
  const status = form.querySelector("[data-code-status]");
  button.disabled = true;
  try {
    const body = new URLSearchParams({ action: "send_code", csrf: form.elements.csrf.value });
    const response = await fetch("/admin/revenue", {
      method: "POST", credentials: "same-origin", headers: { Accept: "application/json" }, body
    });
    const result = await response.json();
    status.textContent = result.message || "Could not send your finance code.";
  } catch (_) { status.textContent = "Could not send your finance code. Please try again."; }
  finally { button.disabled = false; }
});
