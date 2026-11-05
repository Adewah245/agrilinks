const preferenceKey = "agrilinks-read-replies";
const plainSpeech = (text) =>
  String(text)
    .replace(/[*#_`]/g, "")
    .replace(/https?:\/\/\S+/g, "")
    .trim();

export function createVoice(scope, { prefix, input, uploadAvailable = false }) {
  const { document, window, fetch, setTimeout } = scope;
  const start = document.getElementById(`${prefix}-voice-start`);
  const finish = document.getElementById(`${prefix}-voice-stop`);
  const stopSpeech = document.getElementById(`${prefix}-speech-stop`);
  const read = document.getElementById(`${prefix}-read-aloud`);
  const status = document.getElementById(`${prefix}-voice-status`);
  const Recognition =
    window.SpeechRecognition || window.webkitSpeechRecognition;
  const synthesis = window.speechSynthesis;
  const playback = !!(synthesis && window.SpeechSynthesisUtterance);
  const recorderAvailable =
    uploadAvailable &&
    !!window.MediaRecorder &&
    !!window.navigator.mediaDevices?.getUserMedia;
  let active = true,
    locked = false,
    listening = false,
    opening = false,
    transcribing = false;
  let recognition = null,
    recorder = null,
    stream = null,
    discardRecording = false;
  let recordingTimer = null,
    requestSequence = 0,
    speechSequence = 0,
    speakingButton = null;
  let transcriptController = null,
    recognitionFailed = false;
  const sayStatus = (message) => {
    if (active && status) status.textContent = message;
  };
  const update = () => {
    if (start) {
      start.disabled =
        locked ||
        opening ||
        transcribing ||
        (!Recognition && !recorderAvailable);
      start.setAttribute("aria-pressed", String(listening));
    }
    if (finish) {
      finish.hidden = !listening && !opening;
      finish.disabled = locked;
    }
  };
  try {
    if (read)
      read.checked = window.localStorage.getItem(preferenceKey) === "true";
  } catch {}
  if (read) read.disabled = !playback;
  if (!Recognition && !recorderAvailable && start)
    sayStatus(
      "Voice questions are not available on this device. You can still type your question.",
    );
  if (!playback)
    sayStatus(
      "Reading aloud is not available on this device. The answer will still appear as text.",
    );

  function rememberRead() {
    try {
      window.localStorage.setItem(preferenceKey, String(read?.checked));
    } catch {}
  }
  function enableRead() {
    if (read && playback) {
      read.checked = true;
      rememberRead();
    }
  }
  function stopReading() {
    speechSequence++;
    synthesis?.cancel();
    if (speakingButton) {
      speakingButton.textContent = "Listen to answer";
      speakingButton.setAttribute("aria-pressed", "false");
    }
    speakingButton = null;
    if (stopSpeech) stopSpeech.hidden = true;
  }
  function speak(text, button) {
    if (!playback) {
      sayStatus("Reading aloud is not available on this device.");
      return;
    }
    if (button && speakingButton === button) {
      stopReading();
      return;
    }
    stopReading();
    const words = plainSpeech(text).split(/\s+/).filter(Boolean);
    const chunks = [];
    let chunk = "";
    for (const word of words) {
      if ((chunk + " " + word).length > 220 && chunk) {
        chunks.push(chunk);
        chunk = "";
      }
      chunk += (chunk ? " " : "") + word;
    }
    if (chunk) chunks.push(chunk);
    if (!chunks.length) return;
    const sequence = speechSequence;
    const voices = synthesis.getVoices();
    const selectedVoice =
      voices.find((voice) => /en[-_]NG/i.test(voice.lang)) ||
      voices.find((voice) => /^en/i.test(voice.lang));
    speakingButton = button || null;
    if (button) {
      button.textContent = "Stop reading";
      button.setAttribute("aria-pressed", "true");
    }
    if (stopSpeech) stopSpeech.hidden = false;
    chunks.forEach((content, index) => {
      const utterance = new window.SpeechSynthesisUtterance(content);
      utterance.lang = selectedVoice?.lang || "en-NG";
      if (selectedVoice) utterance.voice = selectedVoice;
      utterance.rate = 0.9;
      utterance.onend = () => {
        if (
          active &&
          sequence === speechSequence &&
          index === chunks.length - 1
        )
          stopReading();
      };
      utterance.onerror = (event) => {
        if (
          active &&
          sequence === speechSequence &&
          !["canceled", "interrupted"].includes(event.error)
        ) {
          stopReading();
          sayStatus("The voice could not play. You can still read the answer.");
        }
      };
      synthesis.speak(utterance);
    });
  }
  function addListenButton(container, text) {
    if (!playback) return null;
    const button = document.createElement("button");
    button.type = "button";
    button.className = "ai-listen-button";
    button.textContent = "Listen to answer";
    button.setAttribute("aria-pressed", "false");
    button.addEventListener("click", () => speak(text, button));
    container.append(button);
    return button;
  }
  function releaseStream() {
    stream?.getTracks().forEach((track) => track.stop());
    stream = null;
  }
  function setQuestion(text) {
    if (!active || !input) return;
    const limit = input.maxLength > 0 ? input.maxLength : 1200;
    input.value = Array.from(text).slice(0, limit).join("");
    input.dispatchEvent(new window.Event("input", { bubbles: true }));
  }
  function stopListening(discard = false) {
    if (discard || opening) requestSequence++;
    opening = false;
    discardRecording = discard;
    if (discard) {
      transcriptController?.abort();
      transcriptController = null;
      transcribing = false;
    }
    try {
      if (discard) recognition?.abort();
      else recognition?.stop();
    } catch {}
    if (recorder && recorder.state !== "inactive") {
      transcribing = !discard;
      try {
        recorder.stop();
      } catch {
        transcribing = false;
      }
    }
    if (recordingTimer) window.clearTimeout(recordingTimer);
    recordingTimer = null;
    listening = false;
    releaseStream();
    update();
  }
  async function startRecording() {
    opening = true;
    const sequence = ++requestSequence;
    update();
    sayStatus("Allow your microphone, then ask your question.");
    try {
      const acquired = await window.navigator.mediaDevices.getUserMedia({
        audio: { echoCancellation: true, noiseSuppression: true },
      });
      if (!active || sequence !== requestSequence) {
        acquired.getTracks().forEach((track) => track.stop());
        return;
      }
      stream = acquired;
      const mime = [
        "audio/webm;codecs=opus",
        "audio/mp4",
        "audio/ogg;codecs=opus",
      ].find((value) => window.MediaRecorder.isTypeSupported(value));
      const current = mime
        ? new window.MediaRecorder(stream, { mimeType: mime })
        : new window.MediaRecorder(stream);
      recorder = current;
      discardRecording = false;
      const chunks = [];
      let size = 0;
      current.addEventListener("dataavailable", (event) => {
        if (event.data.size) {
          size += event.data.size;
          chunks.push(event.data);
          if (size > 2 * 1024 * 1024) stopListening();
        }
      });
      current.addEventListener("stop", async () => {
        if (recorder !== current) return;
        releaseStream();
        recorder = null;
        listening = false;
        if (recordingTimer) window.clearTimeout(recordingTimer);
        if (!active || discardRecording || sequence !== requestSequence) {
          transcribing = false;
          update();
          return;
        }
        const blob = new window.Blob(chunks, {
          type: current.mimeType || "audio/webm",
        });
        if (!blob.size || blob.size > 2 * 1024 * 1024) {
          transcribing = false;
          sayStatus("Please record a shorter question and try again.");
          update();
          return;
        }
        transcribing = true;
        update();
        sayStatus("Listening to your question…");
        const body = new window.FormData();
        body.set(
          "audio",
          blob,
          "question." +
            (blob.type.includes("mp4")
              ? "mp4"
              : blob.type.includes("ogg")
                ? "ogg"
                : "webm"),
        );
        const controller = new window.AbortController();
        transcriptController = controller;
        try {
          const response = await fetch("/learning/transcribe", {
            method: "POST",
            body,
            headers: { Accept: "application/json" },
            signal: controller.signal,
          });
          const result = await response.json();
          if (
            !active ||
            sequence !== requestSequence ||
            controller.signal.aborted
          )
            return;
          if (
            !response.ok ||
            typeof result.text !== "string" ||
            !result.text.trim()
          )
            throw new Error(
              result.error ||
                "We could not hear your question. Please try again.",
            );
          setQuestion(result.text);
          sayStatus(
            "Your question is ready. You can change it before sending.",
          );
        } catch (error) {
          if (
            active &&
            sequence === requestSequence &&
            error.name !== "AbortError"
          )
            sayStatus(
              error.message ||
                "We could not hear your question. Please try again.",
            );
        } finally {
          if (sequence === requestSequence) {
            transcribing = false;
            transcriptController = null;
            update();
          }
        }
      });
      current.addEventListener("error", () => {
        stopListening(true);
        sayStatus("The microphone stopped. Please try again.");
      });
      current.start(500);
      listening = true;
      sayStatus("Ask your question, then tap Finish recording.");
      recordingTimer = setTimeout(() => stopListening(), 60000);
    } catch {
      if (active && sequence === requestSequence) {
        releaseStream();
        sayStatus("Please allow microphone access, or type your question.");
      }
    } finally {
      if (sequence === requestSequence) {
        opening = false;
        update();
      }
    }
  }
  function startRecognition() {
    const sequence = ++requestSequence;
    recognition = new Recognition();
    recognition.lang = "en-NG";
    recognition.interimResults = true;
    recognition.continuous = false;
    recognition.onresult = (event) => {
      if (!active || sequence !== requestSequence) return;
      const words = Array.from(event.results)
        .map((result) => result[0].transcript)
        .join(" ");
      setQuestion(words.trim());
    };
    recognition.onerror = (event) => {
      if (!active || sequence !== requestSequence) return;
      listening = false;
      if (event.error === "network") recognitionFailed = true;
      sayStatus(
        event.error === "not-allowed"
          ? "Please allow microphone access, or type your question."
          : event.error === "no-speech"
            ? "I could not hear you. Please try again."
            : "Voice typing did not work. Try voice again or type your question.",
      );
      update();
    };
    recognition.onend = () => {
      if (active && sequence === requestSequence) {
        listening = false;
        update();
      }
    };
    try {
      recognition.start();
      listening = true;
      sayStatus("Listening… Ask your question, then tap Finish recording.");
    } catch {
      listening = false;
      sayStatus("The microphone could not start. Please try again.");
    }
    update();
  }
  start?.addEventListener("click", () => {
    if (locked || transcribing || !input) return;
    if (listening || opening) {
      stopListening();
      return;
    }
    stopReading();
    enableRead();
    if (Recognition && !recognitionFailed) startRecognition();
    else if (recorderAvailable) startRecording();
  });
  finish?.addEventListener("click", () => stopListening());
  stopSpeech?.addEventListener("click", stopReading);
  read?.addEventListener("change", () => {
    rememberRead();
    if (!read.checked) stopReading();
  });
  update();
  const dispose = () => {
    active = false;
    stopListening(true);
    stopReading();
  };
  scope.cleanup(dispose);
  window.addEventListener("pagehide", dispose);
  return {
    speak,
    addListenButton,
    stopReading,
    stopListening,
    enableRead,
    get busy() {
      return opening || listening || transcribing;
    },
    setBusy(value) {
      locked = value;
      update();
    },
    autoSpeak(text, button) {
      if (read?.checked) speak(text, button);
    },
  };
}
