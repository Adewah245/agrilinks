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

registerForm.addEventListener("submit", (event) => {

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
        completeButton.disabled = profilePicture.files.length === 0;
        registerHeading.textContent = "Complete your registration";
        profilePicture.focus();
        return;
    }

    completeButton.disabled = true;
    completeButton.textContent = "Completing registration...";

});

profilePicture.addEventListener("change", () => {
    completeButton.disabled = profilePicture.files.length === 0;
});

backButton.addEventListener("click", () => {
    selfieStep = false;
    profilePicture.required = false;
    registrationSections.forEach((section) => {
        section.hidden = false;
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