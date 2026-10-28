const registerForm = document.getElementById("register-form");

const firstName = document.getElementById("first-name");
const lastName = document.getElementById("last-name");
const email = document.getElementById("email");
const phone = document.getElementById("phone");

const country = document.getElementById("country");
const state = document.getElementById("state");
const lga = document.getElementById("lga");
const community = document.getElementById("community");

const password = document.getElementById("password");
const confirmPassword = document.getElementById("confirm-password");

const completeButton = document.getElementById("complete-registration");
const backButton = document.getElementById("back-to-details");
const profileSection = document.querySelector(".profile-section");
const cameraButton = document.getElementById("open-selfie-camera");
const captureButton = document.getElementById("take-selfie");
const retakeButton = document.getElementById("retake-selfie");
const selfieViewport = document.getElementById("selfie-viewport");
const selfieVideo = document.getElementById("selfie-video");
const selfiePreview = document.getElementById("selfie-preview");
const selfieStatus = document.getElementById("selfie-status");
const registerHeading = document.getElementById("register-heading");
const registrationSections = Array.from(
    registerForm.querySelectorAll(":scope > .form-section")
);
let selfieStep = false;
const detailsButton = document.getElementById("register-submit");
const sendCodeButton = document.getElementById("send-registration-code");
const registrationCode = document.getElementById("registration-code");
const confirmationPhone = document.getElementById("confirmation-phone");
const confirmationStatus = document.getElementById("confirmation-status");
let codePhone = "";
let resendAt = 0;
let sendingCode = false;
let submitting = false;
let selfieBlob = null;
let selfiePreviewURL = "";
let cameraStream = null;
let cameraRequest = 0;
let openingCamera = false;
let capturingSelfie = false;

function registrationLog(event, details = {}, level = "info") {
    console[level]("[registration]", event, details);
}

function cameraIsReady() {
    return cameraStream !== null && selfieVideo.readyState >= 2
        && selfieVideo.videoWidth > 0 && selfieVideo.videoHeight > 0;
}

function updateSelfieControls() {
    const hasSelfie = selfieBlob !== null;
    const hasCamera = cameraStream !== null;
    cameraButton.hidden = hasSelfie || hasCamera;
    cameraButton.disabled = openingCamera || submitting;
    captureButton.hidden = !hasCamera;
    captureButton.disabled = openingCamera || !cameraIsReady() || capturingSelfie || submitting;
    captureButton.textContent = capturingSelfie ? "Taking selfie…" : "Take selfie";
    retakeButton.hidden = !hasSelfie;
    retakeButton.disabled = submitting;
    selfieVideo.hidden = !hasCamera;
    selfiePreview.hidden = !hasSelfie;
    selfieViewport.hidden = !hasCamera && !hasSelfie;
}

function stopSelfieCamera(reason) {
    if (cameraStream || openingCamera || capturingSelfie) {
        registrationLog("selfie.camera.stopped", { reason, request: cameraRequest });
    }
    // Invalidate pending permission and capture requests before leaving the step.
    cameraRequest++;
    openingCamera = false;
    capturingSelfie = false;
    cameraButton.textContent = "Open camera";
    if (cameraStream) {
        cameraStream.getTracks().forEach(track => track.stop());
        cameraStream = null;
    }
    selfieVideo.srcObject = null;
    updateSelfieControls();
}

function cameraErrorMessage(error) {
    switch (error.name) {
        case "NotAllowedError":
        case "PermissionDeniedError":
        case "SecurityError":
            return "Camera access was blocked. Allow camera access for this site in your browser and device settings, then retry.";
        case "NotFoundError":
        case "DevicesNotFoundError":
            return "No camera was found. Connect a camera or use a device with a camera, then retry.";
        case "NotReadableError":
        case "TrackStartError":
            return "Your camera could not open. Close other apps using it and check your device camera permissions, then retry.";
        default:
            return "Could not open your camera. Check camera permissions and retry.";
    }
}

async function openSelfieCamera() {
    if (!selfieStep || openingCamera || capturingSelfie || submitting || cameraStream || selfieBlob) return;
    if (!window.isSecureContext) {
        registrationLog("selfie.camera.unavailable", { reason: "insecure_connection" }, "warn");
        selfieStatus.textContent = "Camera access requires a secure connection. Open this site using HTTPS, then retry.";
        cameraButton.textContent = "Retry camera";
        return;
    }
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
        registrationLog("selfie.camera.unavailable", { reason: "unsupported_browser" }, "warn");
        selfieStatus.textContent = "This browser cannot access your camera. Use a current browser on a device with a camera to finish registration.";
        cameraButton.textContent = "Retry camera";
        return;
    }

    const request = ++cameraRequest;
    registrationLog("selfie.camera.requested", { request });
    openingCamera = true;
    selfieStatus.textContent = "Allow camera access when prompted. Opening your camera…";
    cameraButton.textContent = "Opening camera…";
    updateConfirmationControls();
    try {
        const stream = await navigator.mediaDevices.getUserMedia({
            video: { facingMode: { ideal: "user" } },
            audio: false,
        });
        if (request !== cameraRequest || !selfieStep) {
            stream.getTracks().forEach(track => track.stop());
            registrationLog("selfie.camera.stale_request", { request });
            return;
        }
        cameraStream = stream;
        selfieVideo.srcObject = stream;
        updateSelfieControls();
        stream.getVideoTracks().forEach(track => track.addEventListener("ended", () => {
            if (cameraStream !== stream) return;
            registrationLog("selfie.camera.interrupted", { request }, "warn");
            stopSelfieCamera("camera_ended");
            selfieStatus.textContent = "Camera access ended. Check your camera permissions and retry.";
            cameraButton.textContent = "Retry camera";
            updateConfirmationControls();
        }, { once: true }));
        await selfieVideo.play();
        if (request !== cameraRequest || !selfieStep) return;
        openingCamera = false;
        registrationLog("selfie.camera.ready", {
            request, width: selfieVideo.videoWidth, height: selfieVideo.videoHeight,
        });
        selfieStatus.textContent = "Position your face in the preview, then take your selfie.";
        updateSelfieControls();
        captureButton.focus();
    } catch (error) {
        if (request !== cameraRequest || !selfieStep) return;
        registrationLog("selfie.camera.failed", { request, error: error.name }, "error");
        stopSelfieCamera("camera_error");
        selfieStatus.textContent = cameraErrorMessage(error);
        cameraButton.textContent = "Retry camera";
        updateConfirmationControls();
    } finally {
        if (request === cameraRequest) {
            openingCamera = false;
            cameraButton.textContent = "Open camera";
            updateConfirmationControls();
        }
    }
}

async function takeSelfie() {
    if (!selfieStep || openingCamera || !cameraIsReady() || capturingSelfie || submitting) return;
    const request = cameraRequest;
    registrationLog("selfie.capture.started", { request });
    capturingSelfie = true;
    updateConfirmationControls();
    try {
        const canvas = document.createElement("canvas");
        const scale = Math.min(1, 1024 / Math.max(selfieVideo.videoWidth, selfieVideo.videoHeight));
        canvas.width = Math.max(1, Math.round(selfieVideo.videoWidth * scale));
        canvas.height = Math.max(1, Math.round(selfieVideo.videoHeight * scale));
        const context = canvas.getContext("2d");
        if (!context) throw new Error("Camera capture is unavailable in this browser. Try another browser.");
        context.drawImage(selfieVideo, 0, 0, canvas.width, canvas.height);
        const blob = await new Promise(resolve => canvas.toBlob(resolve, "image/jpeg", 0.9));
        if (request !== cameraRequest || !selfieStep) {
            registrationLog("selfie.capture.discarded", { request });
            return;
        }
        if (!blob || blob.size === 0 || blob.size > 10 * 1024 * 1024) {
            throw new Error("Could not save your selfie. Please take it again.");
        }
        const previewURL = URL.createObjectURL(blob);
        if (selfiePreviewURL) URL.revokeObjectURL(selfiePreviewURL);
        selfieBlob = blob;
        selfiePreviewURL = previewURL;
        selfiePreview.src = previewURL;
        registrationLog("selfie.capture.completed", {
            bytes: blob.size, type: blob.type, width: canvas.width, height: canvas.height,
        });
        stopSelfieCamera("selfie_captured");
        message.textContent = "";
        selfieStatus.textContent = "Selfie captured. This will be your profile picture when registration is complete.";
        updateConfirmationControls();
        retakeButton.focus();
    } catch (error) {
        if (request !== cameraRequest || !selfieStep) return;
        registrationLog("selfie.capture.failed", { request, error: error.name }, "error");
        selfieStatus.textContent = error.message || "Could not take your selfie. Please try again.";
    } finally {
        if (request === cameraRequest) {
            capturingSelfie = false;
            updateConfirmationControls();
        }
    }
}

cameraButton.addEventListener("click", openSelfieCamera);
captureButton.addEventListener("click", takeSelfie);
retakeButton.addEventListener("click", () => {
    if (!selfieStep || submitting) return;
    registrationLog("selfie.retake");
    selfieBlob = null;
    if (selfiePreviewURL) URL.revokeObjectURL(selfiePreviewURL);
    selfiePreviewURL = "";
    selfiePreview.removeAttribute("src");
    updateConfirmationControls();
    openSelfieCamera();
});
selfieVideo.addEventListener("loadeddata", updateSelfieControls);
selfieVideo.addEventListener("resize", updateSelfieControls);
window.addEventListener("pagehide", () => stopSelfieCamera("page_hidden"));
document.addEventListener("visibilitychange", () => {
    if (document.visibilityState !== "hidden") return;
    const hadCamera = cameraStream !== null || openingCamera;
    stopSelfieCamera("tab_hidden");
    if (hadCamera) selfieStatus.textContent = "Camera paused. Open it again to take your selfie.";
    updateConfirmationControls();
});

function updateConfirmationControls() {
    const wait = Math.max(0, Math.ceil((resendAt - Date.now()) / 1000));
    sendCodeButton.disabled = sendingCode || submitting || wait > 0;
    sendCodeButton.textContent = sendingCode ? "Sending code…" : wait > 0
        ? `Resend code in ${wait}s` : codePhone ? "Resend confirmation code" : "Send confirmation code";
    backButton.disabled = sendingCode || submitting;
    completeButton.disabled = submitting || sendingCode || openingCamera || capturingSelfie || !selfieBlob
        || codePhone !== phone.value.trim() || !/^[0-9]{6}$/.test(registrationCode.value.trim());
    updateSelfieControls();
}

updateConfirmationControls();
registrationLog("selfie.initialized", {
    secureContext: window.isSecureContext,
    cameraSupported: Boolean(navigator.mediaDevices && navigator.mediaDevices.getUserMedia),
});

sendCodeButton.addEventListener("click", async () => {
    if (sendingCode || Date.now() < resendAt) return;
    sendingCode = true;
    confirmationStatus.textContent = "";
    updateConfirmationControls();
    const targetPhone = phone.value.trim();
    registrationLog("phone.code.requested");
    try {
        const response = await fetch("/register/phone-code", {
            method: "POST",
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
            body: new URLSearchParams({ phone: targetPhone }),
        });
        registrationLog("phone.code.response", { status: response.status });
        if (!response.ok) {
            const retry = Number(response.headers.get("Retry-After"));
            if (retry > 0) resendAt = Date.now() + retry * 1000;
            throw new Error(await response.text());
        }
        const data = await response.json();
        codePhone = targetPhone;
        registrationCode.value = "";
        registrationCode.disabled = false;
        registrationCode.required = true;
        resendAt = Date.now() + (Number(data.retry_after) || 0) * 1000;
        confirmationStatus.textContent = data.message;
        registrationLog("phone.code.sent");
        registrationCode.focus();
    } catch (error) {
        registrationLog("phone.code.failed", { error: error.name }, "error");
        confirmationStatus.textContent = error.message || "Could not send the code. Please try again.";
    } finally {
        sendingCode = false;
        updateConfirmationControls();
    }
});

registrationCode.addEventListener("input", updateConfirmationControls);
phone.addEventListener("input", () => {
    codePhone = "";
    registrationCode.value = "";
    registrationCode.disabled = true;
    registrationCode.required = false;
    confirmationStatus.textContent = "";
    updateConfirmationControls();
});
setInterval(updateConfirmationControls, 1000);


// ===============================
// MESSAGE AREA
// ===============================

const message = document.createElement("p");

message.className = "message";
message.setAttribute("aria-live", "polite");

registerForm.appendChild(message);


// ===============================
// PASSWORD STRENGTH
// ===============================

const strength = document.createElement("small");

strength.className = "password-strength";

password.parentNode.appendChild(strength);

password.addEventListener("input", () => {

    if (password.value.length < 8) {

        strength.textContent = "Weak Password";
        strength.style.color = "red";

    } else if (password.value.length < 12) {

        strength.textContent = "Medium Password";
        strength.style.color = "orange";

    } else {

        strength.textContent = "Strong Password";
        strength.style.color = "green";

    }

});


// ===============================
// SHOW / HIDE PASSWORD
// ===============================

const togglePassword =
    document.getElementById("toggle-password");

const toggleConfirm =
    document.getElementById("toggle-confirm-password");


togglePassword.addEventListener("click", () => {

    if (password.type === "password") {

        password.type = "text";

        togglePassword.classList.replace(
            "fa-eye",
            "fa-eye-slash"
        );

    } else {

        password.type = "password";

        togglePassword.classList.replace(
            "fa-eye-slash",
            "fa-eye"
        );

    }

});


toggleConfirm.addEventListener("click", () => {

    if (confirmPassword.type === "password") {

        confirmPassword.type = "text";

        toggleConfirm.classList.replace(
            "fa-eye",
            "fa-eye-slash"
        );

    } else {

        confirmPassword.type = "password";

        toggleConfirm.classList.replace(
            "fa-eye-slash",
            "fa-eye"
        );

    }

});


// ===============================
// LOAD STATES
// ===============================

async function loadStates() {

    try {

        const response = await fetch("/api/states");

        if (!response.ok) {
            throw new Error("Failed to load states");
        }

        const states = await response.json();

        state.innerHTML =
            '<option value="">Select your state</option>';

        states.forEach(item => {

            const option = document.createElement("option");

            option.value = item.name;

            option.textContent = item.name;

            option.dataset.stateId = item.id;

            state.appendChild(option);

        });

    } catch (error) {

        console.error("State loading error:", error);

        message.textContent =
            "Unable to load states. Please refresh the page.";

        message.style.color = "red";
    }
}


// Load states when page opens.
loadStates();


// ===============================
// LOAD LGAs
// ===============================

state.addEventListener("change", async () => {

    const selectedOption =
        state.options[state.selectedIndex];

    const stateID =
        selectedOption.dataset.stateId;

    // Reset LGA.
    lga.innerHTML =
        '<option value="">Select your LGA</option>';

    lga.disabled = true;

    if (!stateID) {
        return;
    }

    try {

        const response =
            await fetch(`/api/states/${stateID}/lgas`);

        if (!response.ok) {
            throw new Error("Failed to load LGAs");
        }

        const lgas =
            await response.json();

        lgas.forEach(item => {

            const option =
                document.createElement("option");

            option.value = item.name;

            option.textContent = item.name;

            option.dataset.lgaId = item.id;

            lga.appendChild(option);

        });

        lga.disabled = false;

    } catch (error) {

        console.error("LGA loading error:", error);

        message.textContent =
            "Unable to load LGAs. Please try again.";

        message.style.color = "red";
    }

});


// ===============================
// FORM VALIDATION
// ===============================

registerForm.addEventListener("submit", async (event) => {
	// Keep form details and the captured selfie in place if confirmation fails.
    event.preventDefault();
    if (submitting || sendingCode) return;

    message.textContent = "";

    // -------------------------------
    // PURPOSE
    // -------------------------------

    const selectedPurposes =
        document.querySelectorAll(
            'input[name="purpose"]:checked'
        );

    if (selectedPurposes.length === 0) {

        event.preventDefault();

        message.textContent =
            "Please select your purpose.";

        message.style.color = "red";

        return;
    }


    // -------------------------------
    // FARMING TYPE
    // -------------------------------

    const wantsToSell =
        Array.from(selectedPurposes)
            .some(input => input.value === "sell");

    const farmTypes =
        document.querySelectorAll(
            'input[name="farm_type"]:checked'
        );

    if (wantsToSell && farmTypes.length === 0) {

        event.preventDefault();

        message.textContent =
            "Please select at least one farming type.";

        message.style.color = "red";

        return;
    }


    // -------------------------------
    // BASIC INFORMATION
    // -------------------------------

    if (
        firstName.value.trim() === "" ||
        lastName.value.trim() === "" ||
        phone.value.trim() === "" ||
        email.value.trim() === ""
    ) {

        event.preventDefault();

        message.textContent =
            "Please complete all required personal information.";

        message.style.color = "red";

        return;
    }


    // -------------------------------
    // LOCATION
    // -------------------------------

    if (
        country.value === "" ||
        state.value === "" ||
        lga.value === "" ||
        community.value.trim() === ""
    ) {

        event.preventDefault();

        message.textContent =
            "Please complete your location information.";

        message.style.color = "red";

        return;
    }


    // -------------------------------
    // PASSWORD
    // -------------------------------

    if (password.value !== confirmPassword.value) {

        event.preventDefault();

        message.textContent =
            "Passwords do not match.";

        message.style.color = "red";

        return;
    }


    if (password.value.length < 8) {

        event.preventDefault();

        message.textContent =
            "Password must be at least 8 characters.";

        message.style.color = "red";

        return;
    }


    if (!selfieStep) {
        event.preventDefault();
        selfieStep = true;
        registrationLog("selfie.step.opened", { hasSelfie: Boolean(selfieBlob) });
        registrationSections.forEach((section) => {
            section.hidden = section !== profileSection;
            section.style.display = section === profileSection ? "" : "none";
        });
        detailsButton.hidden = true;
        confirmationPhone.textContent = phone.value.trim();
        registrationCode.disabled = codePhone !== phone.value.trim();
        registrationCode.required = !registrationCode.disabled;
        updateConfirmationControls();
        registerHeading.textContent = "Confirm your phone and finish registration";
        (selfieBlob ? retakeButton : cameraButton).focus();
        return;
    }

    if (!selfieBlob || openingCamera || capturingSelfie) {
        registrationLog("submission.blocked", { reason: "selfie_required" }, "warn");
        message.textContent = "Take a selfie with your camera before completing registration.";
        message.style.color = "red";
        return;
    }
    if (codePhone !== phone.value.trim() || !/^[0-9]{6}$/.test(registrationCode.value.trim())) {
        registrationLog("submission.blocked", { reason: "phone_confirmation_required" }, "warn");
        message.textContent = "Request and enter the SMS code for your phone number.";
        message.style.color = "red";
        return;
    }
    submitting = true;
    registrationLog("submission.started", { selfieBytes: selfieBlob.size, selfieType: selfieBlob.type });
    updateConfirmationControls();
    completeButton.textContent = "Confirming and creating account…";
    try {
        const formData = new FormData(registerForm);
        formData.set("profile_picture", selfieBlob, "selfie.jpg");
        const response = await fetch(registerForm.action, {
            method: "POST",
            body: formData,
        });
        registrationLog("submission.response", { status: response.status, redirected: response.redirected });
        if (response.ok && response.redirected && new URL(response.url).pathname === "/login") {
            registrationLog("submission.completed");
            window.location.assign(response.url);
            return;
        }
        throw new Error(response.ok ? "Could not complete registration. Please try again." : await response.text());
    } catch (error) {
        registrationLog("submission.failed", { error: error.name }, "error");
        message.textContent = error.message || "Could not complete registration. Please try again.";
        message.style.color = "red";
    } finally {
        submitting = false;
        completeButton.textContent = "Complete Registration";
        updateConfirmationControls();
    }

});

backButton.addEventListener("click", () => {
    if (sendingCode || submitting) return;
    selfieStep = false;
    registrationLog("selfie.step.left", { hasSelfie: Boolean(selfieBlob) });
    stopSelfieCamera("back_to_details");
    if (!selfieBlob) selfieStatus.textContent = "Open your camera to take a selfie.";
    registrationCode.required = false;
    registrationCode.disabled = true;
    detailsButton.hidden = false;
    registrationSections.forEach((section) => {
        section.hidden = section === profileSection;
        section.style.display = "";
    });
    registerHeading.textContent = "Create Your Account";
    confirmPassword.focus();
});


// ===============================
// BACKGROUND IMAGE ROTATION
// ===============================

const registerImages = [

    "/static/assets/images/login-image6.jpeg",

    "/static/assets/images/login-image7.jpeg",

    "/static/assets/images/login-image8.jpeg",

    "/static/assets/images/un.jpeg"

];


const leftPanel =
    document.querySelector(".auth-left");

let current = 0;


function changeBackground() {

    current++;

    if (current >= registerImages.length) {
        current = 0;
    }

    leftPanel.style.backgroundImage = `

        linear-gradient(
            rgba(0,100,0,.82),
            rgba(0,100,0,.82)
        ),

        url('${registerImages[current]}')

    `;
}


changeBackground();

setInterval(
    changeBackground,
    4000
);
