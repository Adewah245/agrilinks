"use strict";
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
    const starters = [...document.querySelectorAll(".learning-starters button")];
    const available = !send.disabled;
    let history = [], historyCategory = "", pending = false;
    const addMessage = (text, role, error = false) => {
        const message = document.createElement("article");
        message.className = `public-learning-message ${role}${error ? " error" : ""}`;
        const label = document.createElement("span");
        label.textContent = role === "user" ? "You" : "Agro-Shield guide";
        const content = document.createElement("p");
        String(text).split(/(\*\*[^*]+\*\*)/g).forEach(part => {
            if (part.startsWith("**") && part.endsWith("**") && part.length > 4) {
                const bold = document.createElement("strong");
                bold.textContent = part.slice(2, -2); content.append(bold);
            } else content.append(document.createTextNode(part));
        });
        message.append(label, content); list.append(message); list.scrollTop = list.scrollHeight;
        return message;
    };
    const newChat = () => {
        history = []; historyCategory = ""; list.replaceChildren();
        addMessage("Hello! What would you like to learn about farming today?", "assistant");
        status.textContent = "Choose a category to begin. Your conversation stays in this tab.";
    };
    category.addEventListener("change", () => {
        const other = category.value === "other";
        custom.hidden = !other; custom.disabled = !other; custom.required = other;
        document.getElementById("custom-farming-label").hidden = !other;
        if (history.length) newChat();
        if (other) custom.focus();
    });
    reset.addEventListener("click", () => { if (!pending) { newChat(); input.focus(); } });
    starters.forEach(button => button.addEventListener("click", () => {
        if (pending) return;
        category.value = button.dataset.category; category.dispatchEvent(new Event("change"));
        input.value = button.dataset.question; input.focus();
    }));
    form.addEventListener("submit", async event => {
        event.preventDefault();
        if (pending || !available || !form.reportValidity()) return;
        const topic = category.value === "other" ? custom.value.trim() : category.value;
        const question = input.value.trim();
        if (!topic || !question || Array.from(question).length > 1200) { status.textContent = "Choose a category and enter a question of up to 1,200 characters."; return; }
        if (historyCategory && historyCategory !== topic) newChat();
        const messages = [...history.slice(-10).map(message => ({role:message.role,content:Array.from(message.content).slice(0,1100).join("")})), {role:"user",content:question}];
        pending = true;
        [send, reset, input, category, custom, ...starters].forEach(element => { element.disabled = true; });
        addMessage(question, "user");
        const typing = document.createElement("div"); typing.className = "public-learning-message typing"; typing.textContent = "Your guide is preparing an answer…"; list.append(typing);
        status.textContent = "Your guide is working on your question…";
        const controller = new AbortController();
        const timeout = setTimeout(() => controller.abort(), 125000);
        try {
            const response = await fetch("/learning/chat", {method:"POST",headers:{"Content-Type":"application/json",Accept:"application/json"},body:JSON.stringify({farmingType:topic,messages}),signal:controller.signal});
            if (!response.ok) { const result = await response.json().catch(() => null); throw new Error(result?.error || "The guide could not answer right now. Please try again."); }
            const result = await response.json();
            if (typeof result.answer !== "string" || !result.answer.trim()) throw new Error("The guide returned an empty answer. Please try again.");
            typing.remove(); addMessage(result.answer, "assistant");
            history = [...messages,{role:"assistant",content:result.answer}].slice(-12); historyCategory = topic;
            input.value = ""; status.textContent = "Your conversation stays in this tab. Start a new chat whenever you like.";
        } catch (error) {
            typing.remove(); addMessage(error.name === "AbortError" ? "The guide took too long to reply. Please try again." : error.message, "assistant", true);
            status.textContent = "Your question is still in the box. You can try again.";
        } finally {
            clearTimeout(timeout); typing.remove(); pending = false;
            [send, reset, input, category, ...starters].forEach(element => { element.disabled = false; });
            custom.disabled = category.value !== "other"; input.focus();
        }
    });
})();
