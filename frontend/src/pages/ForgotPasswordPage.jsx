import PasswordField from "../components/PasswordField.jsx";

export default function ForgotPasswordPage({ data }) {
  return (
    <main className="auth-container">
      <div className="auth-left">
        <h1>Reset Password</h1>
        <p>
          Enter the phone number on your account, then confirm the code sent to
          your phone or email before choosing a new password.
        </p>
      </div>
      <div className="auth-right">
        <form id="forgot-form" method="POST" action="/forgot-password">
          <h2>Forgot Password</h2>
          {data.Error && (
            <p className="message" style={{ color: "red" }} role="alert">
              {data.Error}
            </p>
          )}
          {data.Success && (
            <p className="message" style={{ color: "green" }} role="status">
              {data.Success}
            </p>
          )}
          <label htmlFor="phone">Phone Number</label>
          <input
            id="phone"
            type="tel"
            name="phone"
            defaultValue={data.Phone || ""}
            placeholder="Enter your phone number"
            autoComplete="tel"
            required
          />
          <button type="submit" name="action" value="send_code" formNoValidate>
            {data.CodeSent ? "Resend code" : "Send verification code"}
          </button>
          {data.CodeSent && (
            <>
              <label htmlFor="code">Verification Code</label>
              <input
                type="text"
                id="code"
                name="code"
                inputMode="numeric"
                pattern="[0-9]{6}"
                maxLength={6}
                autoComplete="one-time-code"
                placeholder="6-digit code"
                required
              />
              <PasswordField
                id="new_password"
                label="New Password"
                placeholder="At least 8 characters"
                autoComplete="new-password"
                minLength={8}
                required
              />
              <PasswordField
                id="confirm_password"
                label="Confirm New Password"
                placeholder="Confirm new password"
                autoComplete="new-password"
                minLength={8}
                required
              />
              <button type="submit">Reset Password</button>
            </>
          )}
          <p>
            <a href="/login">Back to login</a>
          </p>
        </form>
      </div>
    </main>
  );
}
