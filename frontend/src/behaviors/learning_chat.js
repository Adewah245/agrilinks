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
  (() => {
    const form = document.getElementById("learning-chat-form");
    if (!form) return;
    const input = document.getElementById("learning-question");
    const list = document.getElementById("chat-messages");
    const category = document.getElementById("farming-type");
    const custom = document.getElementById("custom-farming-type");
    const status = document.getElementById("learning-status");
    const send = document.getElementById("send-question");
    const reset = document.getElementById("clear-chat");
    const voice = createVoice(scope, {
      prefix: "learning",
      input,
      uploadAvailable: Boolean(data?.VoiceUploadAvailable),
    });
    const pictureCancels = [];
    const starters = [
      ...document.querySelectorAll(".learning-starters button"),
    ];
    const available = !send.disabled;
    const learningNote =
      "This chat is for learning. For a serious crop or animal problem, ask a local farm adviser or animal doctor for help.";
    let history = [],
      historyCategory = "",
      pending = false;
    const addMessage = (text, role, error = false, example = null) => {
      const message = document.createElement("article");
      message.className = `learning-message learning-message-${role}${error ? " learning-message-error" : ""}`;
      if (role === "assistant") {
        const label = document.createElement("span");
        label.className = "learning-message-label";
        label.textContent = "Agrilinks Nigeria learning guide";
        message.append(label);
      }
      const content = document.createElement("p");
      String(text)
        .split(/(\*\*[^*]+\*\*)/g)
        .forEach((part) => {
          if (part.startsWith("**") && part.endsWith("**") && part.length > 4) {
            const bold = document.createElement("strong");
            bold.textContent = part.slice(2, -2);
            content.append(bold);
          } else content.append(document.createTextNode(part));
        });
      message.append(content);
      if (role === "assistant" && !error) {
        voice.addListenButton(message, text);
        if (example && data?.ExamplesAvailable)
          pictureCancels.push(addExampleButton(scope, message, example));
      }
      list.append(message);
      list.scrollTop = list.scrollHeight;
      return message;
    };
    const newChat = () => {
      voice.stopReading();
      voice.stopListening(true);
      pictureCancels.splice(0).forEach((cancel) => cancel());
      history = [];
      historyCategory = "";
      list.replaceChildren();
      addMessage(
        "Hello! What would you like to learn about farming today?",
        "assistant",
      );
      status.textContent = learningNote;
    };
    category.addEventListener("change", () => {
      const other = category.value === "other";
      custom.hidden = !other;
      custom.disabled = !other;
      custom.required = other;
      document.getElementById("custom-farming-label").hidden = !other;
      if (history.length) newChat();
      status.textContent =
        "Choose a farming category and ask a question about it.";
      if (other) custom.focus();
    });
    const requestedTopic = new URL(window.location.href).searchParams
      .get("topic")
      ?.trim();
    if (
      requestedTopic &&
      Array.from(requestedTopic).length <= 80 &&
      !/[\u0000-\u001f\u007f]/.test(requestedTopic)
    ) {
      const existing = [...category.options].some(
        (option) => option.value === requestedTopic,
      );
      category.value = existing ? requestedTopic : "other";
      category.dispatchEvent(new Event("change"));
      if (!existing) custom.value = requestedTopic;
      input.value = `What should I know before getting started with ${requestedTopic.toLowerCase()}?`;
    }
    reset.addEventListener("click", () => {
      if (!pending) {
        newChat();
        input.focus();
      }
    });
    starters.forEach((button) =>
      button.addEventListener("click", () => {
        if (pending) return;
        category.value = button.dataset.category;
        category.dispatchEvent(new Event("change"));
        input.value = button.dataset.question;
        input.focus();
      }),
    );
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      if (pending || !available || !form.reportValidity()) return;
      if (voice.busy) {
        status.textContent = "Finish your voice question before sending.";
        return;
      }
      const topic =
        category.value === "other" ? custom.value.trim() : category.value;
      const question = input.value.trim();
      if (!topic || !question || Array.from(question).length > 1200) {
        status.textContent =
          "Choose a category and enter a question of up to 1,200 characters.";
        return;
      }
      if (historyCategory && historyCategory !== topic) newChat();
      const messages = [
        ...history.slice(-10).map((message) => ({
          role: message.role,
          content: Array.from(message.content).slice(0, 1100).join(""),
        })),
        { role: "user", content: question },
      ];
      pending = true;
      voice.stopReading();
      voice.setBusy(true);
      list.setAttribute("aria-busy", "true");
      [send, reset, input, category, custom, ...starters].forEach((element) => {
        element.disabled = true;
      });
      addMessage(question, "user");
      const typing = document.createElement("div");
      typing.className =
        "learning-message learning-message-assistant learning-typing";
      typing.setAttribute(
        "aria-label",
        "Agrilinks Nigeria is preparing a response",
      );
      const typingLabel = document.createElement("span");
      typingLabel.className = "learning-message-label";
      typingLabel.textContent = "Agrilinks Nigeria learning guide";
      const dots = document.createElement("span");
      dots.className = "learning-typing-dots";
      dots.setAttribute("aria-hidden", "true");
      for (let i = 0; i < 3; i++) dots.append(document.createElement("i"));
      const caption = document.createElement("span");
      caption.className = "learning-typing-caption";
      caption.textContent = "Putting together a helpful answer…";
      typing.append(typingLabel, dots, caption);
      list.append(typing);
      list.scrollTop = list.scrollHeight;
      status.textContent = "Your guide is working on your question…";
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 125000);
      try {
        const response = await fetch("/learning/chat", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
          },
          body: JSON.stringify({ farmingType: topic, messages }),
          signal: controller.signal,
        });
        if (!response.ok) {
          const result = await response.json().catch(() => null);
          throw new Error(
            result?.error ||
              "The guide could not answer right now. Please try again.",
          );
        }
        const result = await response.json();
        if (typeof result.answer !== "string" || !result.answer.trim())
          throw new Error(
            "The guide returned an empty answer. Please try again.",
          );
        typing.remove();
        const reply = addMessage(result.answer, "assistant", false, {
          topic,
          question,
          answer: result.answer,
        });
        voice.autoSpeak(
          result.answer,
          reply.querySelector(".ai-listen-button"),
        );
        history = [
          ...messages,
          { role: "assistant", content: result.answer },
        ].slice(-12);
        historyCategory = topic;
        input.value = "";
        status.textContent = learningNote;
      } catch (error) {
        typing.remove();
        addMessage(
          error.name === "AbortError"
            ? "The guide took too long to reply. Please try again."
            : error.message,
          "assistant",
          true,
        );
        status.textContent =
          "Your question is still in the box. You can try again.";
      } finally {
        clearTimeout(timeout);
        typing.remove();
        pending = false;
        voice.setBusy(false);
        list.removeAttribute("aria-busy");
        [send, reset, input, category, ...starters].forEach((element) => {
          element.disabled = false;
        });
        custom.disabled = category.value !== "other";
        input.focus();
      }
    });
  })();
}
