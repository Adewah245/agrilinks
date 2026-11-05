import Workspace from "../components/Workspace.jsx";
import { useState } from "react";

export default function AdminSecurityPage({ data }) {
  const [enabled, setEnabled] = useState(data.Enabled);
  const [verified, setVerified] = useState(data.Verified);
  const [secret, setSecret] = useState("");
  const [codes, setCodes] = useState([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [recovery, setRecovery] = useState(false);
  async function submit(event, action) {
    event.preventDefault();
    setBusy(true);
    setError("");
    const form = event.currentTarget;
    const body = new URLSearchParams(new FormData(form));
    body.set("action", action);
    body.set("csrf", data.CSRF);
    try {
      const response = await fetch("/admin/security", { method: "POST", body });
      const result = await response.json();
      if (!response.ok)
        throw new Error(result.error || "Verification failed. Try again.");
      if (result.secret) {
        setSecret(result.secret);
        setEnabled(false);
        setVerified(false);
        setCodes([]);
      }
      if (result.verified) {
        setEnabled(true);
        setVerified(true);
        setSecret("");
        setCodes(result.recoveryCodes || []);
      }
      form.reset();
    } catch (error) {
      setError(error.message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Workspace current="/admin/security" staff>
      <main className="security-page">
        <a href="/dashboard">← Back to your account</a>
        <h1>Administrator security</h1>
        <p>
          Use an authenticator app to protect access to private evidence and
          sensitive administrator actions.
        </p>
        {!data.Available ? (
          <p role="alert">
            Private security needs to be configured by the server operator
            before administrator access is available.
          </p>
        ) : (
          <>
            {error && (
              <p role="alert" className="security-error">
                {error}
              </p>
            )}
            {verified && (
              <p role="status">
                Administrator verification is active for 15 minutes on this
                session.
              </p>
            )}
            {codes.length > 0 && (
              <section>
                <h2>Save your recovery codes</h2>
                <p>
                  Store these somewhere private. Each code works once with your
                  password. They will not be displayed again.
                </p>
                <ul className="recovery-codes">
                  {codes.map((code) => (
                    <li key={code}>
                      <code>{code}</code>
                    </li>
                  ))}
                </ul>
              </section>
            )}
            {((!enabled && !secret) || verified) && (
              <form onSubmit={(e) => submit(e, enabled ? "replace" : "enroll")}>
                <h2>
                  {enabled
                    ? "Replace your authenticator"
                    : "Set up your authenticator"}
                </h2>
                {enabled && (
                  <p>
                    This replaces your current authenticator and recovery codes,
                    and signs out your other sessions.
                  </p>
                )}
                <label>
                  Current password
                  <input
                    name="password"
                    type="password"
                    autoComplete="current-password"
                    required
                  />
                </label>
                <button disabled={busy}>
                  {enabled ? "Replace authenticator" : "Start setup"}
                </button>
              </form>
            )}
            {secret && (
              <section>
                <h2>Add this account to your app</h2>
                <p>
                  Choose a time-based account in your authenticator app, named
                  Agrilinks Nigeria, and enter this setup key. Do not share it.
                </p>
                <code className="setup-secret">{secret}</code>
              </section>
            )}
            {!verified && (enabled || secret) && (
              <form
                onSubmit={(e) => submit(e, recovery ? "recover" : "verify")}
              >
                <h2>
                  {recovery
                    ? "Use a recovery code"
                    : "Verify your authenticator"}
                </h2>
                {recovery && (
                  <label>
                    Current password
                    <input
                      name="password"
                      type="password"
                      autoComplete="current-password"
                      required
                    />
                  </label>
                )}
                <label>
                  {recovery ? "Recovery code" : "Six-digit code"}
                  <input
                    key={String(recovery)}
                    name="code"
                    autoComplete="one-time-code"
                    inputMode={recovery ? "text" : "numeric"}
                    pattern={recovery ? "[a-fA-F0-9]{20}" : "[0-9]{6}"}
                    maxLength={recovery ? 20 : 6}
                    required
                  />
                </label>
                <button disabled={busy}>Verify</button>
                {enabled && (
                  <button type="button" onClick={() => setRecovery((v) => !v)}>
                    {recovery
                      ? "Use authenticator instead"
                      : "Use a recovery code"}
                  </button>
                )}
              </form>
            )}
            {verified && (
              <a
                className="security-primary"
                href={data.Destination || "/admin"}
              >
                Continue to administrator workspace
              </a>
            )}
          </>
        )}
      </main>
    </Workspace>
  );
}
