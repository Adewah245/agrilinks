import { createVoice } from "../lib/voice.js";
import { addExampleButton } from "../lib/example-images.js";

export default function mount(scope, data) {
  const { document, window, fetch } = scope;
  scope.ready(() => {
    const form = document.getElementById("ai-form");
    if (!form) return;
    const send = form.querySelector(".send-button");
    const status = document.getElementById("submit-status");
    const imageInput = document.getElementById("image");
    const imagePreview = document.getElementById("image-preview");
    const imageWrap = document.getElementById("image-preview-wrap");
    const imageStatus = document.getElementById("image-status");
    const videoInput = document.getElementById("video");
    const videoFeedback = document.getElementById("video-feedback");
    const menu = document.getElementById("attachment-menu");
    const imageButton = document.getElementById("add-image");
    const resultPanel = document.getElementById("ai-result");
    const resultText = document.getElementById("ai-result-text");
    const resultTools = document.getElementById("ai-result-tools");
    const voice = createVoice(scope, { prefix: "diagnosis" });
    let active = true,
      pending = false,
      imageURL = "",
      controller = null,
      cancelExample = null;
    const audio = setupAudioRecorder(scope, voice, () => pending);

    function renderResult(answer, id, examples, read = false) {
      cancelExample?.();
      voice.stopReading();
      resultText.textContent = answer;
      resultTools.replaceChildren();
      resultPanel.hidden = !answer;
      const empty = document.getElementById("ai-result-empty");
      if (empty) empty.hidden = Boolean(answer);
      if (!answer) return;
      const listen = voice.addListenButton(resultTools, answer);
      if (examples && Number.isInteger(id) && id > 0) {
        cancelExample = addExampleButton(
          scope,
          resultTools,
          { diagnosis_id: id },
          "/ai-assistant/example",
        );
      }
      if (read) voice.autoSpeak(answer, listen);
    }
    renderResult(data?.Result || "", data?.ResultID, data?.ExamplesAvailable);

    function closeMenu() {
      menu.hidden = true;
      imageButton.setAttribute("aria-expanded", "false");
    }
    imageButton.addEventListener("click", (event) => {
      event.stopPropagation();
      menu.hidden = !menu.hidden;
      imageButton.setAttribute("aria-expanded", String(!menu.hidden));
    });
    document.addEventListener("click", (event) => {
      if (!menu.contains(event.target) && event.target !== imageButton)
        closeMenu();
    });
    for (const [button, input, capture] of [
      ["upload-photo", imageInput, false],
      ["take-photo", imageInput, true],
      ["upload-video", videoInput, false],
      ["record-video", videoInput, true],
    ]) {
      document.getElementById(button).addEventListener("click", () => {
        closeMenu();
        if (capture) input.setAttribute("capture", "environment");
        else input.removeAttribute("capture");
        input.click();
      });
    }
    function clearImage() {
      imageInput.value = "";
      if (imageURL) window.URL.revokeObjectURL(imageURL);
      imageURL = "";
      imagePreview.removeAttribute("src");
      imageWrap.hidden = true;
      imageStatus.textContent = "";
    }
    imageInput.addEventListener("change", () => {
      const file = imageInput.files?.[0];
      if (!file) return;
      if (!file.type.startsWith("image/") || file.size > 10 * 1024 * 1024) {
        clearImage();
        status.textContent = "Please choose a photo smaller than 10MB.";
        return;
      }
      if (imageURL) window.URL.revokeObjectURL(imageURL);
      imageURL = window.URL.createObjectURL(file);
      imagePreview.src = imageURL;
      imageStatus.textContent = file.name + " attached";
      imageWrap.hidden = false;
    });
    document
      .getElementById("remove-image")
      .addEventListener("click", clearImage);
    videoInput.addEventListener("change", () => {
      const file = videoInput.files?.[0];
      if (!file) return;
      if (!file.type.startsWith("video/") || file.size > 30 * 1024 * 1024) {
        videoInput.value = "";
        videoFeedback.hidden = true;
        status.textContent = "Please choose a video smaller than 30MB.";
        return;
      }
      document.getElementById("video-status").textContent =
        file.name + " attached";
      videoFeedback.hidden = false;
    });
    document.getElementById("remove-video").addEventListener("click", () => {
      videoInput.value = "";
      videoFeedback.hidden = true;
    });
    document
      .getElementById("think-toggle")
      .addEventListener("click", (event) => {
        const button = event.currentTarget;
        const enabled = button.getAttribute("aria-pressed") !== "true";
        button.setAttribute("aria-pressed", String(enabled));
        document.getElementById("thinking").value = String(enabled);
      });

    document.querySelectorAll("[data-assistant-question]").forEach((button) => {
      button.addEventListener("click", () => {
        if (pending) return;
        const category = document.getElementById("category");
        const question = document.getElementById("description");
        category.value = button.dataset.assistantCategory;
        question.value = button.dataset.assistantQuestion;
        question.focus();
        question.scrollIntoView({ block: "nearest", behavior: "smooth" });
      });
    });

    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      if (pending || !data?.Available) return;
      if (audio.busy) {
        status.textContent = "Finish recording before sending your question.";
        return;
      }
      const body = new window.FormData(form);
      // Keep the file ourselves: some browsers cannot assign input.files.
      if (audio.file) body.set("audio", audio.file, audio.file.name);
      if (
        !String(body.get("description") || "").trim() &&
        !["image", "video", "audio"].some((key) => body.get(key)?.size > 0)
      ) {
        status.textContent =
          "Describe what happened or attach a photo or recording.";
        return;
      }
      pending = true;
      send.disabled = true;
      audio.setBusy(true);
      voice.stopReading();
      status.textContent = "Checking your question… Please wait.";
      controller = new window.AbortController();
      const timer = scope.setTimeout(() => controller?.abort(), 95000);
      try {
        const response = await fetch(form.action, {
          method: "POST",
          body,
          headers: { Accept: "application/json" },
          credentials: "same-origin",
          signal: controller.signal,
        });
        if (!active) return;
        if (response.redirected) {
          window.location.assign(response.url);
          return;
        }
        const result = await response.json();
        if (
          !response.ok ||
          typeof result.result !== "string" ||
          !result.result.trim()
        )
          throw new Error(
            result.error ||
              "The farming assistant could not answer. Please try again.",
          );
        renderResult(
          result.result,
          result.diagnosis_id,
          result.examples_available,
          true,
        );
        status.textContent =
          "Your answer is ready. You can read it or listen to it.";
        resultPanel.scrollIntoView({ block: "nearest", behavior: "smooth" });
      } catch (error) {
        if (active)
          status.textContent =
            error.name === "AbortError"
              ? "The answer took too long. Your question is still here. Please try again."
              : error.message || "Please check your connection and try again.";
      } finally {
        window.clearTimeout(timer);
        controller = null;
        pending = false;
        if (active) {
          send.disabled = !data?.Available;
          audio.setBusy(false);
        }
      }
    });
    scope.cleanup(() => {
      active = false;
      controller?.abort();
      cancelExample?.();
      if (imageURL) window.URL.revokeObjectURL(imageURL);
    });
  });
}

function setupAudioRecorder(scope, voice, isPending) {
  const { document, window } = scope;
  const start = document.getElementById("start-audio");
  const stop = document.getElementById("stop-audio");
  const status = document.getElementById("audio-status");
  const preview = document.getElementById("audio-preview");
  const remove = document.getElementById("remove-audio");
  const play = document.getElementById("play-audio");
  let active = true,
    locked = false,
    opening = false,
    finishing = false,
    recorder = null,
    stream = null;
  let file = null,
    previewURL = "",
    sequence = 0,
    timer = null;
  const supported =
    !!window.MediaRecorder && !!window.navigator.mediaDevices?.getUserMedia;
  const release = () => {
    stream?.getTracks().forEach((track) => track.stop());
    stream = null;
  };
  const update = () => {
    start.disabled = !supported || locked || opening || finishing;
    start.classList.toggle("recording", recorder?.state === "recording");
    start.setAttribute(
      "aria-label",
      recorder?.state === "recording"
        ? "Stop voice recording"
        : "Start voice recording",
    );
    stop.hidden = !recorder || recorder.state !== "recording";
    stop.disabled = stop.hidden;
    remove.disabled = locked || opening || finishing || !!recorder;
  };
  function clear() {
    file = null;
    document.getElementById("audio").value = "";
    preview.pause();
    if (previewURL) window.URL.revokeObjectURL(previewURL);
    previewURL = "";
    preview.removeAttribute("src");
    preview.load();
    preview.hidden = true;
    remove.hidden = true;
    play.hidden = true;
  }
  function finish() {
    if (!recorder || recorder.state === "inactive") return;
    finishing = true;
    recorder.stop();
    window.clearTimeout(timer);
    release();
    status.textContent = "Preparing your voice recording…";
    update();
  }
  start.addEventListener("click", async () => {
    if (!supported || !active || locked || opening || finishing || isPending())
      return;
    if (recorder?.state === "recording") {
      finish();
      return;
    }
    opening = true;
    const currentSequence = ++sequence;
    voice.stopReading();
    voice.enableRead();
    clear();
    update();
    status.textContent = "Allow your microphone, then ask your question.";
    try {
      const acquired = await window.navigator.mediaDevices.getUserMedia({
        audio: { echoCancellation: true, noiseSuppression: true },
      });
      if (!active || currentSequence !== sequence) {
        acquired.getTracks().forEach((track) => track.stop());
        return;
      }
      stream = acquired;
      const mime = [
        "audio/webm;codecs=opus",
        "audio/mp4",
        "audio/ogg;codecs=opus",
      ].find((type) => window.MediaRecorder.isTypeSupported(type));
      const current = mime
        ? new window.MediaRecorder(stream, { mimeType: mime })
        : new window.MediaRecorder(stream);
      recorder = current;
      const chunks = [];
      let size = 0;
      current.addEventListener("dataavailable", (event) => {
        if (event.data.size) {
          chunks.push(event.data);
          size += event.data.size;
          if (size > 10 * 1024 * 1024) finish();
        }
      });
      current.addEventListener("stop", () => {
        release();
        window.clearTimeout(timer);
        if (!active || currentSequence !== sequence) return;
        recorder = null;
        finishing = false;
        const blob = new window.Blob(chunks, {
          type: current.mimeType || "audio/webm",
        });
        if (!blob.size || blob.size > 10 * 1024 * 1024) {
          status.textContent =
            "Please record a shorter question and try again.";
          update();
          return;
        }
        const extension = blob.type.includes("mp4")
          ? "mp4"
          : blob.type.includes("ogg")
            ? "ogg"
            : "webm";
        file = new window.File([blob], "voice-recording." + extension, {
          type: blob.type,
        });
        previewURL = window.URL.createObjectURL(blob);
        preview.src = previewURL;
        preview.hidden = false;
        remove.hidden = false;
        play.hidden = false;
        status.textContent =
          "Your voice recording is ready. You can play it before sending.";
        update();
      });
      current.addEventListener("error", () => {
        sequence++;
        release();
        window.clearTimeout(timer);
        recorder = null;
        opening = false;
        finishing = false;
        status.textContent =
          "The microphone stopped. Please record your question again.";
        update();
      });
      current.start(500);
      status.textContent = "Ask your question, then tap Stop recording.";
      timer = scope.setTimeout(finish, 60000);
    } catch {
      release();
      if (active)
        status.textContent =
          "Please allow microphone access, or type your question.";
    } finally {
      if (currentSequence === sequence) {
        opening = false;
        update();
      }
    }
  });
  stop.addEventListener("click", finish);
  remove.addEventListener("click", () => {
    clear();
    status.textContent = "Voice recording removed.";
  });
  play.addEventListener("click", async () => {
    try {
      await preview.play();
    } catch {
      status.textContent = "This device could not play the recording.";
    }
  });
  const dispose = () => {
    active = false;
    sequence++;
    window.clearTimeout(timer);
    if (recorder?.state !== "inactive") {
      try {
        recorder?.stop();
      } catch {}
    }
    release();
    clear();
  };
  scope.cleanup(dispose);
  window.addEventListener("pagehide", dispose);
  if (!supported)
    status.textContent =
      "Voice recording is not available on this device. You can still type your question.";
  update();
  return {
    get file() {
      return file;
    },
    get busy() {
      return opening || finishing || !!recorder;
    },
    setBusy(value) {
      locked = value;
      update();
    },
  };
}
