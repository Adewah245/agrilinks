import { createVoice } from "../lib/voice.js";
import { addExampleButton } from "../lib/example-images.js";

export default function mount(scope, data) {
  const {
    document,
    window,
    setTimeout,
    setInterval,
    requestAnimationFrame,
    fetch,
    ResizeObserver,
    IntersectionObserver,
  } = scope;
  ("use strict");
  const voice = createVoice(scope, {
    prefix: "journey",
    input: document.getElementById("learning-question"),
    uploadAvailable: Boolean(data?.VoiceUploadAvailable),
  });
  const messages = data?.Messages || [];
  document
    .querySelectorAll(".learning-message.ai")
    .forEach((element, index) => {
      const aiMessages = messages.filter(
        (message) => message.SenderType === "ai",
      );
      const message = aiMessages[index];
      if (!message) return;
      const listen = voice.addListenButton(element, message.Content);
      if (data?.ExamplesAvailable) {
        const position = messages.indexOf(message);
        const question =
          messages
            .slice(0, position)
            .reverse()
            .find((entry) => entry.SenderType === "user")?.Content ||
          data?.Journey?.Goal ||
          "";
        addExampleButton(scope, element, {
          topic: data?.Journey?.FarmingType?.Name,
          question,
          answer: message.Content,
        });
      }
      try {
        if (
          index === aiMessages.length - 1 &&
          window.sessionStorage.getItem("agrilinks-speak-journey") ===
            String(data?.Journey?.ID)
        ) {
          window.sessionStorage.removeItem("agrilinks-speak-journey");
          voice.autoSpeak(message.Content, listen);
        }
      } catch {}
    });

  document.querySelectorAll("[data-learning-form]").forEach((form) => {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      if (voice.busy) return;
      const button = form.querySelector('button[type="submit"]');
      const error = document.getElementById("learning-error");
      const status = form.querySelector(".form-status");
      button.disabled = true;
      voice.setBusy(true);
      error.hidden = true;
      if (status)
        status.textContent = "Your guide is working on your question…";
      try {
        const response = await fetch(form.action, {
          method: "POST",
          body: new URLSearchParams(new FormData(form)),
          headers: { Accept: "application/json" },
          credentials: "same-origin",
        });
        if (response.redirected) {
          window.location.assign(response.url);
          return;
        }
        const result = await response.json();
        if (!response.ok || !result.success)
          throw new Error(
            result.error || "We could not save that change. Please try again.",
          );
        if (new URL(form.action).pathname === "/learn/message") {
          try {
            window.sessionStorage.setItem(
              "agrilinks-speak-journey",
              String(data?.Journey?.ID),
            );
          } catch {}
        }
        window.location.reload();
      } catch (problem) {
        error.textContent =
          problem.message || "Please check your connection and try again.";
        error.hidden = false;
        error.scrollIntoView({ block: "nearest" });
      } finally {
        button.disabled = false;
        voice.setBusy(false);
        if (status) status.textContent = "";
      }
    });
  });
  const conversation = document.querySelector(".learning-messages");
  if (conversation) conversation.scrollTop = conversation.scrollHeight;
}
