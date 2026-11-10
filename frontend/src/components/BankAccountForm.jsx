import { useEffect, useState } from "react";

async function readResponse(response, fallback) {
  if (response.redirected || response.status === 401) {
    throw new Error(
      "Your session has expired. Sign in again to verify your bank account.",
    );
  }
  const result = await response.json().catch(() => null);
  if (!response.ok || !result) {
    throw new Error(
      typeof result?.error === "string" ? result.error : fallback,
    );
  }
  return result;
}

export default function BankAccountForm({ hasTransactionPIN, registeredName }) {
  const [directory, setDirectory] = useState({
    status: "loading",
    banks: [],
    error: "",
  });
  const [directoryAttempt, setDirectoryAttempt] = useState(0);
  const [bankCode, setBankCode] = useState("");
  const [number, setNumber] = useState("");
  const [verification, setVerification] = useState(null);
  const [lookupStatus, setLookupStatus] = useState("idle");
  const [lookupError, setLookupError] = useState("");
  const [lookupAttempt, setLookupAttempt] = useState(0);
  const [pin, setPin] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    let active = true;
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 12000);
    setDirectory({ status: "loading", banks: [], error: "" });
    async function loadBanks() {
      try {
        const banks = await readResponse(
          await fetch("/wallet/banks", {
            credentials: "same-origin",
            cache: "no-store",
            headers: { Accept: "application/json" },
            signal: controller.signal,
          }),
          "Could not load the bank list. Please try again.",
        );
        if (!active) return;
        if (controller.signal.aborted)
          throw new Error("Loading banks timed out.");
        if (
          !Array.isArray(banks) ||
          !banks.length ||
          banks.some(
            (bank) =>
              typeof bank?.name !== "string" ||
              !bank.name.trim() ||
              typeof bank?.code !== "string" ||
              !bank.code.trim(),
          )
        )
          throw new Error("The bank list is unavailable. Please try again.");
        setDirectory({ status: "ready", banks, error: "" });
      } catch (error) {
        if (active)
          setDirectory({
            status: "error",
            banks: [],
            error: controller.signal.aborted
              ? "Loading banks took too long. Please try again."
              : error.message ||
                "Could not load the bank list. Please try again.",
          });
      } finally {
        clearTimeout(timeout);
      }
    }
    loadBanks();
    return () => {
      active = false;
      clearTimeout(timeout);
      controller.abort();
    };
  }, [directoryAttempt]);

  useEffect(() => {
    setVerification(null);
    setLookupError("");
    setLookupStatus("idle");
    setPin("");
    if (
      directory.status !== "ready" ||
      !bankCode ||
      !/^[0-9]{10}$/.test(number)
    )
      return;

    let active = true;
    let timeout;
    const controller = new AbortController();
    setLookupStatus("checking");
    // Wait for typing to settle and cancel older requests when either detail changes.
    const delay = setTimeout(async () => {
      timeout = setTimeout(() => controller.abort(), 25000);
      try {
        const account = await readResponse(
          await fetch("/wallet/resolve-account", {
            method: "POST",
            credentials: "same-origin",
            cache: "no-store",
            headers: { Accept: "application/json" },
            body: new URLSearchParams({
              bank_code: bankCode,
              account_number: number,
            }),
            signal: controller.signal,
          }),
          "We couldn't check your account name. Please try again.",
        );
        if (!active) return;
        if (controller.signal.aborted)
          throw new Error("Account lookup timed out.");
        if (
          account.bank_code !== bankCode ||
          account.account_number !== number ||
          typeof account.account_name !== "string" ||
          !account.account_name.trim() ||
          typeof account.matches_profile !== "boolean"
        ) {
          throw new Error(
            "The bank did not confirm these account details. Please check again.",
          );
        }
        setVerification(account);
        setLookupStatus("resolved");
      } catch (error) {
        if (!active) return;
        setLookupStatus("error");
        setLookupError(
          controller.signal.aborted
            ? "The account check took too long. Please try again."
            : error.message ||
                "We couldn't check your account name. Please try again.",
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
  }, [bankCode, number, directory.status, lookupAttempt]);

  useEffect(() => {
    if (!verification) return;
    const timeout = setTimeout(
      () => {
        setVerification(null);
        setPin("");
        setLookupStatus("error");
        setLookupError(
          "Your account check has expired. Check the name again before saving.",
        );
      },
      5 * 60 * 1000,
    );
    return () => clearTimeout(timeout);
  }, [verification]);

  useEffect(() => {
    const restore = (event) => {
      if (!event.persisted) return;
      setVerification(null);
      setPin("");
      setSubmitting(false);
      setLookupAttempt((attempt) => attempt + 1);
    };
    window.addEventListener("pageshow", restore);
    return () => window.removeEventListener("pageshow", restore);
  }, []);

  const invalidate = () => {
    setVerification(null);
    setLookupError("");
    setPin("");
  };
  const account =
    verification?.bank_code === bankCode &&
    verification?.account_number === number
      ? verification
      : null;
  const verified = account?.matches_profile === true;
  const canSave = verified && hasTransactionPIN && !submitting;
  const error =
    directory.error ||
    lookupError ||
    (account && !verified
      ? `This account name does not match your registered name${registeredName ? ` (${registeredName})` : ""}. Use an account in your name or correct your profile.`
      : "");

  return (
    <form
      method="POST"
      action="/wallet"
      className="wallet-form"
      data-bank-account-form
      onSubmitCapture={(event) => {
        if (!canSave) {
          event.preventDefault();
          setLookupError("Verify an account in your name before saving.");
          return;
        }
        setSubmitting(true);
      }}
    >
      <input type="hidden" name="action" value="save_bank" />
      <label htmlFor="wallet-bank">Bank</label>
      <select
        name="bank_code"
        id="wallet-bank"
        required
        value={bankCode}
        disabled={directory.status !== "ready"}
        aria-describedby="bank-error"
        onChange={(event) => {
          if (!submitting) {
            setBankCode(event.target.value);
            invalidate();
          }
        }}
      >
        <option value="">
          {directory.status === "loading"
            ? "Loading banks…"
            : directory.status === "error"
              ? "Bank list unavailable"
              : "Choose your bank"}
        </option>
        {directory.banks.map((bank) => (
          <option key={bank.code} value={bank.code}>
            {bank.name}
          </option>
        ))}
      </select>
      {directory.status === "error" && (
        <button
          type="button"
          className="wallet-bank-retry"
          onClick={() => setDirectoryAttempt((attempt) => attempt + 1)}
        >
          Retry loading banks
        </button>
      )}
      <label htmlFor="wallet-account-number">Account number</label>
      <input
        id="wallet-account-number"
        name="account_number"
        inputMode="numeric"
        pattern="[0-9]{10}"
        maxLength="10"
        placeholder="10-digit account number"
        autoComplete="off"
        required
        value={number}
        readOnly={submitting}
        aria-describedby="wallet-bank-status bank-error"
        onChange={(event) => {
          setNumber(event.target.value);
          invalidate();
        }}
        onPaste={(event) => {
          const pasted = event.clipboardData.getData("text").replace(/\s/g, "");
          if (/^[0-9]{10}$/.test(pasted)) {
            event.preventDefault();
            setNumber(pasted);
            invalidate();
          }
        }}
      />
      <div
        className="wallet-bank-verification"
        data-state={verified ? "verified" : "unverified"}
        aria-busy={lookupStatus === "checking"}
      >
        {account && (
          <>
            <label htmlFor="wallet-account-name">Account name</label>
            <output
              id="wallet-account-name"
              className="wallet-verified-name"
              htmlFor="wallet-bank wallet-account-number"
            >
              {account.account_name}
            </output>
          </>
        )}
        <input
          type="hidden"
          name="account_name"
          value={account?.account_name || ""}
        />
        <p
          id="wallet-bank-status"
          role="status"
          aria-live="polite"
          aria-atomic="true"
        >
          {lookupStatus === "checking"
            ? "Checking account name with your bank…"
            : verified
              ? "Account name verified. Confirm it is yours before saving."
              : account
                ? "Account name returned by your bank."
                : "Choose your bank and enter all 10 digits to check the name."}
        </p>
      </div>
      <p id="bank-error" className="wallet-field-error" role="alert">
        {error}
      </p>
      {lookupStatus === "error" && (
        <button
          type="button"
          className="wallet-bank-retry"
          onClick={() => setLookupAttempt((attempt) => attempt + 1)}
        >
          Check account again
        </button>
      )}
      {!hasTransactionPIN && (
        <p className="wallet-field-error">
          <a href="/profile">Create your transaction PIN in your profile</a>{" "}
          before saving your bank account.
        </p>
      )}
      <label htmlFor="wallet-bank-pin">Transaction PIN</label>
      <input
        id="wallet-bank-pin"
        name="transaction_pin"
        type="password"
        inputMode="numeric"
        pattern="[0-9]{6}"
        maxLength="6"
        autoComplete="off"
        required
        disabled={!verified || !hasTransactionPIN}
        readOnly={submitting}
        value={pin}
        onChange={(event) => setPin(event.target.value)}
      />
      <button
        type="submit"
        className="wallet-submit"
        id="wallet-save-bank"
        disabled={!canSave}
      >
        {submitting ? "Saving account…" : "Save verified account"}
        <svg className="w-icon" aria-hidden="true">
          <use href="#w-check" />
        </svg>
      </button>
    </form>
  );
}
