import { useEffect, useState } from "react";

export default function WalletRecipientFields({
  currency,
  onVerified,
  submitting,
}) {
  const [input, setInput] = useState("");
  const [verification, setVerification] = useState(null);
  const [status, setStatus] = useState("idle");
  const [error, setError] = useState("");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    setVerification(null);
    onVerified(null);
    setError("");
    setStatus("idle");
    const recipient = input.trim();
    if (!recipient) return;

    let active = true;
    let timeout;
    const controller = new AbortController();
    setStatus("checking");
    const delay = setTimeout(async () => {
      timeout = setTimeout(() => controller.abort(), 12000);
      try {
        const response = await fetch("/wallet/recipient", {
          method: "POST",
          credentials: "same-origin",
          cache: "no-store",
          headers: { Accept: "application/json" },
          body: new URLSearchParams({ recipient }),
          signal: controller.signal,
        });
        if (response.redirected || response.status === 401)
          throw new Error(
            "Your session has expired. Sign in again to check the recipient.",
          );
        const account = await response.json().catch(() => null);
        if (!response.ok || !account)
          throw new Error(
            typeof account?.error === "string"
              ? account.error
              : "We couldn't check this recipient. Please try again.",
          );
        if (!active) return;
        if (controller.signal.aborted)
          throw new Error("Recipient lookup timed out.");
        if (
          typeof account.full_name !== "string" ||
          !account.full_name.trim() ||
          typeof account.wallet_number !== "string" ||
          !account.wallet_number.trim() ||
          account.currency !== currency
        )
          throw new Error(
            "We couldn't confirm this wallet. Check the recipient and try again.",
          );
        setVerification({ input, account });
        onVerified(account);
        setStatus("verified");
      } catch (error) {
        if (!active) return;
        setStatus("error");
        setError(
          controller.signal.aborted
            ? "The recipient check took too long. Please try again."
            : error.message ||
                "We couldn't check this recipient. Please try again.",
        );
      } finally {
        clearTimeout(timeout);
      }
    }, 400);
    return () => {
      active = false;
      clearTimeout(delay);
      clearTimeout(timeout);
      controller.abort();
    };
  }, [input, currency, attempt, onVerified]);

  useEffect(() => {
    if (!verification || submitting) return;
    const timer = setTimeout(
      () => {
        setVerification(null);
        onVerified(null);
        setStatus("error");
        setError(
          "Your recipient check has expired. Check the account again before sending.",
        );
      },
      5 * 60 * 1000,
    );
    return () => clearTimeout(timer);
  }, [verification, submitting, onVerified]);

  useEffect(() => {
    const restore = (event) => {
      if (!event.persisted) return;
      setVerification(null);
      onVerified(null);
      setAttempt((value) => value + 1);
    };
    window.addEventListener("pageshow", restore);
    return () => window.removeEventListener("pageshow", restore);
  }, [onVerified]);

  const account = verification?.input === input ? verification.account : null;
  return (
    <>
      <label htmlFor="wallet-recipient">Recipient wallet number</label>
      <input
        id="wallet-recipient"
        placeholder="Enter their wallet number"
        autoComplete="off"
        maxLength={64}
        required
        readOnly={submitting}
        value={input}
        aria-describedby="wallet-recipient-status wallet-recipient-error"
        onChange={(event) => {
          setInput(event.target.value);
          setVerification(null);
          onVerified(null);
          setError("");
        }}
      />
      <input
        type="hidden"
        name="recipient"
        value={account?.wallet_number || ""}
      />
      <div
        className="wallet-bank-verification"
        data-state={account ? "verified" : "unverified"}
        aria-busy={status === "checking"}
      >
        {account && (
          <>
            <label htmlFor="wallet-recipient-name">Account name</label>
            <output
              id="wallet-recipient-name"
              className="wallet-verified-name"
              htmlFor="wallet-recipient"
            >
              {account.full_name}
            </output>
          </>
        )}
        <p
          id="wallet-recipient-status"
          role="status"
          aria-live="polite"
          aria-atomic="true"
        >
          {status === "checking"
            ? "Checking recipient account…"
            : account
              ? "Account verified. Check the name before sending."
              : "Enter the recipient's wallet number to see their verified account name."}
        </p>
      </div>
      <p
        id="wallet-recipient-error"
        className="wallet-field-error"
        role="alert"
      >
        {error}
      </p>
      {status === "error" && (
        <button
          type="button"
          className="wallet-bank-retry"
          onClick={() => setAttempt((value) => value + 1)}
        >
          Check recipient again
        </button>
      )}
    </>
  );
}
