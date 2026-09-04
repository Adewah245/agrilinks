document.addEventListener("DOMContentLoaded", () => {
  const loginForm = document.getElementById("login-form");
  if (!loginForm) return;

  const phone = document.getElementById("phone");
  const password = document.getElementById("password");
  const button = loginForm.querySelector("button[type='submit']");

  // Create message container and place it right above the inputs for better visibility
  const message = document.createElement("p");
  message.className = "message";
  message.setAttribute("aria-live", "polite"); // Screen reader support for dynamic error messages
  
  const formHeading = loginForm.querySelector("h2");
  if (formHeading && formHeading.nextSibling) {
    loginForm.insertBefore(message, formHeading.nextSibling);
  } else {
    loginForm.prepend(message);
  }

  // ---------- Password Visibility Toggle ----------

  const toggle = document.getElementById("toggle-password");

  if (toggle && password) {
    toggle.addEventListener("click", () => {
      const isPassword = password.type === "password";
      
      // Toggle field type
      password.type = isPassword ? "text" : "password";
      
      // Toggle icon classes
      const icon = toggle.querySelector("i") || toggle;
      icon.classList.toggle("fa-eye", !isPassword);
      icon.classList.toggle("fa-eye-slash", isPassword);

      // Update accessibility state
      toggle.setAttribute("aria-pressed", isPassword ? "true" : "false");
      toggle.setAttribute(
        "aria-label",
        isPassword ? "Hide password" : "Show password"
      );
    });
  }

  // ---------- Form Submission & Validation ----------

  loginForm.addEventListener("submit", function (event) {
    message.textContent = "";

    // Basic Phone Validation
    if (!phone.value.trim()) {
      event.preventDefault();
      message.textContent = "Phone number is required.";
      phone.focus();
      return;
    }

    // Basic Password Length Validation
    if (password.value.length < 8) {
      event.preventDefault();
      message.textContent = "Password must be at least 8 characters.";
      password.focus();
      return;
    }

    // Validation passed — lock button to prevent double submissions
    button.disabled = true;
    button.textContent = "Signing In...";
  });

  // ---------- Background Image Rotator (with Preloading) ----------

  const loginImages = [
    "/static/assets/images/login-image1.jpeg",
    "/static/assets/images/login-image2.jpeg",
    "/static/assets/images/login-image3.jpeg",
    "/static/assets/images/login-image4.jpeg",
    "/static/assets/images/login-image5.jpeg"
  ];

  const background = document.getElementById("auth-background");

  if (background && loginImages.length > 1) {
    // Preload images into browser memory to eliminate flash on slide change
    loginImages.forEach((src) => {
      const img = new Image();
      img.src = src;
    });

    let current = 0;

    function changeBackground() {
      current = (current + 1) % loginImages.length;
      background.src = loginImages[current];
    }

    setInterval(changeBackground, 4000);
  }
});