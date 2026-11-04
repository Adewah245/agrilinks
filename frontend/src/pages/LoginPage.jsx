import { useEffect, useRef, useState } from "react";
import PasswordField from "../components/PasswordField.jsx";
import FarmingPhotoPanel from "../components/FarmingPhotoPanel.jsx";

export default function LoginPage({ data }) {
  const [error, setError] = useState(data.Error || "");
  const [submitting, setSubmitting] = useState(false);
  const form = useRef(null);

  useEffect(() => {
    const restore = () => setSubmitting(false);
    window.addEventListener("pageshow", restore);
    return () => window.removeEventListener("pageshow", restore);
  }, []);

  function submit(event) {
    if (submitting) {
      event.preventDefault();
      return;
    }
    const fields = form.current.elements;
    const invalid = !fields.phone.value.trim()
      ? ["phone", "Phone number is required."]
      : fields.password.value.length < 8
        ? ["password", "Password must be at least 8 characters."]
        : null;
    if (invalid) {
      event.preventDefault();
      setError(invalid[1]);
      fields[invalid[0]].focus();
      return;
    }
    setError("");
    setSubmitting(true);
  }

  return (
    <main className="auth-container">
      <FarmingPhotoPanel imageId="auth-background">
        <div className="left-content">
          <h1>Welcome Back</h1>
          <p>
            Sign in to continue managing your farm, monitor agricultural
            products, connect with buyers, access the marketplace, and use
            AI-powered agricultural assistance.
          </p>
        </div>
      </FarmingPhotoPanel>
      <div className="auth-right">
        <div className="brand-area">
          <div className="brand-roof" aria-hidden="true" />
          <img
            className="auth-brand"
            src="/static/assets/logo/agrilinks-nigeria-logo.svg"
            alt="Agrilinks Nigeria logo"
          />
        </div>
        <form
          ref={form}
          id="login-form"
          method="POST"
          action="/login"
          noValidate
          onSubmit={submit}
        >
          <h2>Login</h2>
          {error && (
            <p className="message" role="alert">
              {error}
            </p>
          )}
          <label htmlFor="phone">Phone Number</label>
          <div className="input-wrapper">
            <i className="fa-solid fa-phone input-icon" aria-hidden="true" />
            <input
              type="tel"
              id="phone"
              name="phone"
              placeholder="Enter your phone number"
              autoComplete="tel"
              inputMode="tel"
              maxLength={20}
              required
            />
          </div>
          <PasswordField
            id="password"
            label="Password"
            toggleId="toggle-password"
            placeholder="Enter your password"
            autoComplete="current-password"
            minLength={8}
            maxLength={128}
            required
          />
          <button
            type="submit"
            disabled={submitting}
            aria-label="Sign in to Agrilinks Nigeria"
          >
            {submitting ? "Signing In…" : "Sign In"}
          </button>
          <p className="forgot-password">
            <a href="/forgot-password">Forgot password?</a>
          </p>
          <p className="register-link">
            Don't have an account? <a href="/register">Create one</a>
          </p>
        </form>
      </div>
    </main>
  );
}
