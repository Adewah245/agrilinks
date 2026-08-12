const registerForm = document.getElementById("register-form");

const firstName = document.getElementById("first-name");
const lastName = document.getElementById("last-name");
const email = document.getElementById("email");
const phone = document.getElementById("phone");
const password = document.getElementById("password");
const confirmPassword = document.getElementById("confirm-password");

const button = registerForm.querySelector("button");

// Message
const message = document.createElement("p");
message.className = "message";
registerForm.appendChild(message);

// Password strength
const strength = document.createElement("small");
strength.className = "password-strength";
password.parentNode.appendChild(strength);

// Show Password
const togglePassword = document.getElementById("toggle-password");

// Show Confirm Password
const toggleConfirm = document.getElementById("toggle-confirm-password");

togglePassword.addEventListener("click", () => {
  if (password.type === "password") {
    password.type = "text";
    togglePassword.classList.remove("fa-eye");
    togglePassword.classList.add("fa-eye-slash");
  } else {
    password.type = "password";
    togglePassword.classList.remove("fa-eye-slash");
    togglePassword.classList.add("fa-eye");
  }
});

toggleConfirm.addEventListener("click", () => {
  if (confirmPassword.type === "password") {
    confirmPassword.type = "text";
    toggleConfirm.classList.remove("fa-eye");
    toggleConfirm.classList.add("fa-eye-slash");
  } else {
    confirmPassword.type = "password";
    toggleConfirm.classList.remove("fa-eye-slash");
    toggleConfirm.classList.add("fa-eye");
  }
});

password.addEventListener("input", () => {

  if (password.value.length < 8) {
    strength.textContent = "Weak Password";
    strength.style.color = "red";
  }

  else if (password.value.length < 12) {
    strength.textContent = "Medium Password";
    strength.style.color = "orange";
  }

  else {
    strength.textContent = "Strong Password";
    strength.style.color = "green";
  }

});

registerForm.addEventListener("submit", function (event) {

  message.textContent = "";

  if (
    firstName.value === "" ||
    lastName.value === "" ||
    email.value === "" ||
    phone.value === ""
  ) {
    event.preventDefault();
    message.textContent = "Please complete all fields.";
    message.style.color = "red";
    return;
  }

  if (password.value !== confirmPassword.value) {
    event.preventDefault();
    message.textContent = "Passwords do not match.";
    message.style.color = "red";
    return;
  }

  if (password.value.length < 8) {
    event.preventDefault();
    message.textContent = "Password must be at least 8 characters.";
    message.style.color = "red";
    return;
  }

  // Validation passed — let the form submit for real to POST /register.
  button.disabled = true;
  button.textContent = "Creating Account...";
});

// ---------- Background Images ----------

const registerImages = [
  "/static/assets/images/login-image6.jpeg",
  "/static/assets/images/login-image7.jpeg",
  "/static/assets/images/login-image8.jpeg",
  "/static/assets/images/login-image8.jpeg",
  "/static/assets/images/un.jpeg"
];

const leftPanel = document.querySelector(".auth-left");

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
setInterval(changeBackground, 4000);