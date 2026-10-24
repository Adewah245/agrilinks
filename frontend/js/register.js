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
const profilePicture = document.getElementById("profile-picture");
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

function updateConfirmationControls() {
    const wait = Math.max(0, Math.ceil((resendAt - Date.now()) / 1000));
    sendCodeButton.disabled = sendingCode || submitting || wait > 0;
    sendCodeButton.textContent = sendingCode ? "Sending code…" : wait > 0
        ? `Resend code in ${wait}s` : codePhone ? "Resend confirmation code" : "Send confirmation code";
    backButton.disabled = sendingCode || submitting;
    completeButton.disabled = submitting || sendingCode || profilePicture.files.length === 0
        || codePhone !== phone.value.trim() || !/^[0-9]{6}$/.test(registrationCode.value.trim());
}

sendCodeButton.addEventListener("click", async () => {
    if (sendingCode || Date.now() < resendAt) return;
    sendingCode = true;
    confirmationStatus.textContent = "";
    updateConfirmationControls();
    const targetPhone = phone.value.trim();
    try {
        const response = await fetch("/register/phone-code", {
            method: "POST",
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
            body: new URLSearchParams({ phone: targetPhone }),
        });
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
        registrationCode.focus();
    } catch (error) {
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
	// Keep form details and the selected photo in place if confirmation fails.
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
        registrationSections.forEach((section) => {
            section.hidden = section !== profileSection;
            section.style.display = section === profileSection ? "" : "none";
        });
        profilePicture.required = true;
        detailsButton.hidden = true;
        confirmationPhone.textContent = phone.value.trim();
        registrationCode.disabled = codePhone !== phone.value.trim();
        registrationCode.required = !registrationCode.disabled;
        updateConfirmationControls();
        registerHeading.textContent = "Confirm your phone and finish registration";
        profilePicture.focus();
        return;
    }

    if (codePhone !== phone.value.trim() || !/^[0-9]{6}$/.test(registrationCode.value.trim())) {
        message.textContent = "Request and enter the SMS code for your phone number.";
        message.style.color = "red";
        return;
    }
    submitting = true;
    updateConfirmationControls();
    completeButton.textContent = "Confirming and creating account…";
    try {
        const response = await fetch(registerForm.action, {
            method: "POST",
            body: new FormData(registerForm),
        });
        if (response.ok && response.redirected && new URL(response.url).pathname === "/login") {
            window.location.assign(response.url);
            return;
        }
        throw new Error(response.ok ? "Could not complete registration. Please try again." : await response.text());
    } catch (error) {
        message.textContent = error.message || "Could not complete registration. Please try again.";
        message.style.color = "red";
    } finally {
        submitting = false;
        completeButton.textContent = "Complete Registration";
        updateConfirmationControls();
    }

});

profilePicture.addEventListener("change", () => {
    updateConfirmationControls();
});

backButton.addEventListener("click", () => {
    selfieStep = false;
    profilePicture.required = false;
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
