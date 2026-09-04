document.addEventListener("DOMContentLoaded", () => {
    /* ============================================================
       CONFIG & LIMITS
       ============================================================ */

    // 50 MB limit for videos, 10 MB for images
    const MAX_IMAGE_SIZE_BYTES = 10 * 1024 * 1024;
    const MAX_VIDEO_SIZE_BYTES = 50 * 1024 * 1024;


    /* ============================================================
       ELEMENTS
       ============================================================ */

    const form = document.getElementById("ai-form");
    const submitStatus = document.getElementById("submit-status");

    const imageButton = document.getElementById("add-image");
    const imageInput = document.getElementById("image");
    const imagePreview = document.getElementById("image-preview");
    const imageWrap = document.getElementById("image-preview-wrap");
    const imageStatus = document.getElementById("image-status");
    const removeImage = document.getElementById("remove-image");

    const videoInput = document.getElementById("video");

    const attachmentMenu = document.getElementById("attachment-menu");
    const uploadPhoto = document.getElementById("upload-photo");
    const takePhoto = document.getElementById("take-photo");
    const uploadVideo = document.getElementById("upload-video");
    const recordVideo = document.getElementById("record-video");

    const thinkToggle = document.getElementById("think-toggle");
    const thinkingInput = document.getElementById("thinking");


    /* ============================================================
       ATTACHMENT MENU
       ============================================================ */

    if (imageButton && attachmentMenu) {
        imageButton.addEventListener("click", (event) => {
            event.stopPropagation();
            attachmentMenu.hidden = !attachmentMenu.hidden;
        });

        document.addEventListener("click", (event) => {
            if (
                !attachmentMenu.contains(event.target) &&
                event.target !== imageButton
            ) {
                attachmentMenu.hidden = true;
            }
        });
    }


    /* ============================================================
       UPLOAD PHOTO
       ============================================================ */

    uploadPhoto?.addEventListener("click", () => {
        attachmentMenu.hidden = true;

        if (!imageInput) {
            return;
        }

        imageInput.removeAttribute("capture");
        imageInput.click();
    });


    /* ============================================================
       TAKE PHOTO
       ============================================================ */

    takePhoto?.addEventListener("click", () => {
        attachmentMenu.hidden = true;

        if (!imageInput) {
            return;
        }

        imageInput.setAttribute("capture", "environment");
        imageInput.click();
    });


    /* ============================================================
       UPLOAD VIDEO
       ============================================================ */

    uploadVideo?.addEventListener("click", () => {
        attachmentMenu.hidden = true;

        if (!videoInput) {
            return;
        }

        videoInput.removeAttribute("capture");
        videoInput.click();
    });


    /* ============================================================
       RECORD VIDEO
       ============================================================ */

    recordVideo?.addEventListener("click", () => {
        attachmentMenu.hidden = true;

        if (!videoInput) {
            return;
        }

        videoInput.setAttribute("capture", "environment");
        videoInput.click();
    });


    /* ============================================================
       IMAGE SELECTION & VALIDATION
       ============================================================ */

    imageInput?.addEventListener("change", () => {
        if (!imageInput.files || imageInput.files.length === 0) {
            return;
        }

        const file = imageInput.files[0];

        if (!file.type.startsWith("image/")) {
            clearImageSelection();
            if (imageStatus) {
                imageStatus.textContent = "Please select a valid image.";
            }
            return;
        }

        if (file.size > MAX_IMAGE_SIZE_BYTES) {
            clearImageSelection();
            if (imageStatus) {
                imageStatus.textContent = "Image must be smaller than 10MB.";
            }
            return;
        }

        if (imagePreview) {
            imagePreview.src = URL.createObjectURL(file);
        }

        if (imageStatus) {
            imageStatus.textContent = `${file.name} attached`;
        }

        if (imageWrap) {
            imageWrap.hidden = false;
        }
    });


    /* ============================================================
       REMOVE IMAGE
       ============================================================ */

    function clearImageSelection() {
        if (imageInput) {
            imageInput.value = "";
        }

        if (imagePreview) {
            if (imagePreview.src) {
                URL.revokeObjectURL(imagePreview.src);
            }
            imagePreview.removeAttribute("src");
        }

        if (imageStatus) {
            imageStatus.textContent = "";
        }

        if (imageWrap) {
            imageWrap.hidden = true;
        }
    }

    removeImage?.addEventListener("click", clearImageSelection);


    /* ============================================================
       VIDEO SELECTION & VALIDATION
       ============================================================ */

    videoInput?.addEventListener("change", () => {
        if (!videoInput.files || videoInput.files.length === 0) {
            return;
        }

        const file = videoInput.files[0];

        if (!file.type.startsWith("video/")) {
            videoInput.value = "";
            removeVideoAttachment();
            return;
        }

        if (file.size > MAX_VIDEO_SIZE_BYTES) {
            videoInput.value = "";
            removeVideoAttachment();
            alert("Video file size must be less than 50MB.");
            return;
        }

        showVideoAttachment(file);
    });


    /* ============================================================
       VIDEO ATTACHMENT DISPLAY & REMOVAL
       ============================================================ */

    function showVideoAttachment(file) {
        let existingContainer = document.getElementById("video-attachment-container");

        if (!existingContainer) {
            existingContainer = document.createElement("div");
            existingContainer.id = "video-attachment-container";
            existingContainer.className = "attachment-status-container";

            const statusText = document.createElement("span");
            statusText.id = "video-attachment-status";
            statusText.setAttribute("aria-live", "polite");

            const removeBtn = document.createElement("button");
            removeBtn.type = "button";
            removeBtn.id = "remove-video";
            removeBtn.textContent = "✕";
            removeBtn.setAttribute("aria-label", "Remove video attachment");
            removeBtn.addEventListener("click", removeVideoAttachment);

            existingContainer.appendChild(statusText);
            existingContainer.appendChild(removeBtn);

            const composerFeedback = document.querySelector(".composer-feedback");
            if (composerFeedback) {
                composerFeedback.appendChild(existingContainer);
            }
        }

        const statusText = document.getElementById("video-attachment-status");
        if (statusText) {
            statusText.textContent = `🎥 ${file.name} attached`;
        }
    }

    function removeVideoAttachment() {
        if (videoInput) {
            videoInput.value = "";
        }

        const container = document.getElementById("video-attachment-container");
        if (container) {
            container.remove();
        }
    }


    /* ============================================================
       THINK TOGGLE
       ============================================================ */

    thinkToggle?.addEventListener("click", () => {
        const currentlyPressed = thinkToggle.getAttribute("aria-pressed") === "true";
        const enabled = !currentlyPressed;

        thinkToggle.setAttribute("aria-pressed", String(enabled));

        if (thinkingInput) {
            thinkingInput.value = String(enabled);
        }
    });


    /* ============================================================
       AUDIO RECORDER
       ============================================================ */

    setupAudioRecorder();


    /* ============================================================
       FORM SUBMISSION
       ============================================================ */

    form?.addEventListener("submit", () => {
        const sendButton = form.querySelector(".send-button");

        if (sendButton) {
            sendButton.disabled = true;
        }

        if (submitStatus) {
            submitStatus.textContent = "Analysing your farming issue… Please wait.";
        }
    });
});


/* ================================================================
   AUDIO RECORDER
   ================================================================ */

function setupAudioRecorder() {
    const startButton = document.getElementById("start-audio");
    const stopButton = document.getElementById("stop-audio");
    const status = document.getElementById("audio-status");
    const preview = document.getElementById("audio-preview");
    const removeButton = document.getElementById("remove-audio");
    const playButton = document.getElementById("play-audio");
    const audioInput = document.getElementById("audio");

    if (!startButton || !stopButton || !status || !preview) {
        return;
    }

    let recorder = null;
    let stream = null;
    let chunks = [];
    let previewURL = "";


    /* ============================================================
       SUPPORTED AUDIO FORMAT
       ============================================================ */

    function getSupportedAudioMimeType() {
        if (!window.MediaRecorder) {
            return "";
        }

        const types = [
            "audio/webm;codecs=opus",
            "audio/ogg;codecs=opus",
            "audio/webm",
            "audio/mp4"
        ];

        for (const type of types) {
            if (MediaRecorder.isTypeSupported(type)) {
                return type;
            }
        }

        return "";
    }


    /* ============================================================
       RELEASE MICROPHONE & LIFECYCLE CLEANUP
       ============================================================ */

    function releaseStream() {
        if (!stream) {
            return;
        }

        stream.getTracks().forEach((track) => {
            track.stop();
        });

        stream = null;
    }

    // Release microphone hardware if user navigates away or refreshes
    window.addEventListener("pagehide", releaseStream);


    /* ============================================================
       CLEAR AUDIO PREVIEW
       ============================================================ */

    function clearPreview() {
        if (previewURL) {
            URL.revokeObjectURL(previewURL);
            previewURL = "";
        }

        preview.pause();
        preview.removeAttribute("src");
        preview.load();

        preview.hidden = true;

        if (playButton) {
            playButton.hidden = true;
        }
    }


    /* ============================================================
       PUT RECORDING INTO FILE INPUT (SAFELY)
       ============================================================ */

    function putAudioIntoInput(blob) {
        if (!audioInput) {
            return;
        }

        let extension = "webm";

        if (blob.type.includes("ogg")) {
            extension = "ogg";
        } else if (blob.type.includes("mp4")) {
            extension = "mp4";
        }

        const file = new File(
            [blob],
            `voice-recording.${extension}`,
            {
                type: blob.type,
                lastModified: Date.now()
            }
        );

        try {
            const dataTransfer = new DataTransfer();
            dataTransfer.items.add(file);
            audioInput.files = dataTransfer.files;
        } catch (error) {
            console.warn("Direct input files assignment failed on this browser:", error);
        }
    }


    /* ============================================================
       START RECORDING
       ============================================================ */

    startButton.addEventListener("click", async () => {
        if (recorder && recorder.state === "recording") {
            stopRecording();
            return;
        }

        if (
            !navigator.mediaDevices ||
            !navigator.mediaDevices.getUserMedia ||
            !window.MediaRecorder
        ) {
            status.textContent = "Audio recording is not supported by this browser.";
            return;
        }

        try {
            clearPreview();

            if (removeButton) {
                removeButton.hidden = true;
            }

            stream = await navigator.mediaDevices.getUserMedia({
                audio: {
                    echoCancellation: true,
                    noiseSuppression: true,
                    autoGainControl: true
                }
            });

            chunks = [];
            const mimeType = getSupportedAudioMimeType();

            recorder = mimeType
                ? new MediaRecorder(stream, { mimeType: mimeType })
                : new MediaRecorder(stream);

            recorder.addEventListener("dataavailable", (event) => {
                if (event.data && event.data.size > 0) {
                    chunks.push(event.data);
                }
            });

            recorder.addEventListener("stop", () => {
                const blob = new Blob(chunks, {
                    type: recorder.mimeType || "audio/webm"
                });

                if (blob.size === 0) {
                    status.textContent = "No voice audio was captured. Please try again.";
                    releaseStream();
                    recorder = null;
                    return;
                }

                putAudioIntoInput(blob);

                previewURL = URL.createObjectURL(blob);
                preview.src = previewURL;
                preview.hidden = false;
                preview.controls = true;

                if (playButton) {
                    playButton.hidden = false;
                }

                if (removeButton) {
                    removeButton.hidden = false;
                }

                status.textContent = "Voice recording is ready. Play it before sending.";

                releaseStream();
                chunks = [];
                recorder = null;

                startButton.classList.remove("recording");
                startButton.setAttribute("aria-label", "Start voice recording");

                stopButton.hidden = true;
                stopButton.disabled = true;
            });

            recorder.addEventListener("error", (event) => {
                console.error("Audio recorder error:", event.error);
                status.textContent = "An error occurred while recording audio.";
                releaseStream();
                recorder = null;

                startButton.classList.remove("recording");
                startButton.setAttribute("aria-label", "Start voice recording");

                stopButton.hidden = true;
                stopButton.disabled = true;
            });

            recorder.start(1000);

            startButton.classList.add("recording");
            startButton.setAttribute("aria-label", "Stop voice recording");

            stopButton.hidden = false;
            stopButton.disabled = false;

            status.textContent = "Recording voice...";

        } catch (error) {
            console.error("Microphone error:", error);
            status.textContent = "Microphone permission was denied or unavailable.";
            releaseStream();
        }
    });


    /* ============================================================
       STOP RECORDING
       ============================================================ */

    function stopRecording() {
        if (!recorder || recorder.state === "inactive") {
            return;
        }

        recorder.stop();

        startButton.classList.remove("recording");
        startButton.setAttribute("aria-label", "Start voice recording");

        stopButton.disabled = true;
        stopButton.hidden = true;

        status.textContent = "Preparing voice recording...";
    }

    stopButton.addEventListener("click", stopRecording);


    /* ============================================================
       PLAY RECORDING
       ============================================================ */

    playButton?.addEventListener("click", async () => {
        try {
            await preview.play();
            status.textContent = "Playing your voice recording.";
        } catch (error) {
            console.error("Audio playback error:", error);
            status.textContent = "The recording cannot be played in this browser.";
        }
    });


    /* ============================================================
       REMOVE AUDIO
       ============================================================ */

    removeButton?.addEventListener("click", () => {
        if (audioInput) {
            audioInput.value = "";
        }

        clearPreview();

        removeButton.hidden = true;

        if (playButton) {
            playButton.hidden = true;
        }

        status.textContent = "Voice recording removed.";
    });
}