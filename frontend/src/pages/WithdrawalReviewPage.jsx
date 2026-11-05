import Workspace from "../components/Workspace.jsx";
import { useState } from "react";

export default function WithdrawalReviewPage({ data }) {
  const withdrawal = data.Withdrawal;
  const evidence = data.Evidence;
  const [photoLoaded, setPhotoLoaded] = useState(false);
  const [accountPhotoLoaded, setAccountPhotoLoaded] = useState(
    !data.RegistrationPhotoID,
  );
  const [photoError, setPhotoError] = useState(false);
  const [decision, setDecision] = useState("rejected");
  const canApprove =
    evidence && !data.Error && photoLoaded && accountPhotoLoaded && !photoError;
  return (
    <Workspace current="/admin/finance" staff>
      <main className="security-page security-review">
        <a href="/admin/finance">← Finance and payments</a>
        <h1>Review withdrawal #{withdrawal.ID}</h1>
        <dl>
          <dt>Member</dt>
          <dd>{withdrawal.FullName}</dd>
          <dt>Bank payout</dt>
          <dd>
            {withdrawal.Currency} {withdrawal.Amount}
          </dd>
          <dt>Destination</dt>
          <dd>
            {withdrawal.BankName} · {withdrawal.MaskedAccountNumber} ·{" "}
            {withdrawal.AccountName}
          </dd>
          <dt>Status</dt>
          <dd>{withdrawal.Status}</dd>
        </dl>
        {data.Error && (
          <p role="alert" className="security-error">
            {data.Error}
          </p>
        )}
        {evidence && (
          <>
            <div className="verification-photos">
              <figure>
                <img
                  src={`/private/verification/${evidence.PhotoID}`}
                  alt="Private selfie captured for this withdrawal"
                  onLoad={() => setPhotoLoaded(true)}
                  onError={() => setPhotoError(true)}
                />
                <figcaption>Withdrawal selfie</figcaption>
              </figure>
              {data.RegistrationPhotoID > 0 ? (
                <figure>
                  <img
                    src={`/private/verification/${data.RegistrationPhotoID}`}
                    alt="Private original registration selfie"
                    onLoad={() => setAccountPhotoLoaded(true)}
                    onError={() => setPhotoError(true)}
                  />
                  <figcaption>Original registration selfie</figcaption>
                </figure>
              ) : (
                <p>
                  No earlier verification photograph is available. Follow the
                  verification procedure before approving.
                </p>
              )}
            </div>
            {photoError && (
              <p role="alert" className="security-error">
                A private photograph could not be loaded. Reload this page and
                verify your authenticator if prompted before approving.
              </p>
            )}
            <h2>Location evidence</h2>
            <dl>
              <dt>Coordinates</dt>
              <dd>
                {evidence.Latitude}, {evidence.Longitude}
              </dd>
              <dt>Reported accuracy</dt>
              <dd>{Math.round(evidence.AccuracyM)} metres</dd>
              <dt>Device capture time</dt>
              <dd>{new Date(evidence.CapturedAt).toLocaleString()}</dd>
              <dt>Received by server</dt>
              <dd>{new Date(evidence.ReceivedAt).toLocaleString()}</dd>
            </dl>
            <p>
              Camera and location data are supplied by the member's device.
              Compare the photographs and account details; this evidence alone
              does not prove identity, liveness, or location.
            </p>
          </>
        )}
        {withdrawal.Status === "pending" && (
          <form method="post" action="/admin/action">
            <input type="hidden" name="action" value="wallet_withdrawal" />
            <input type="hidden" name="id" value={withdrawal.ID} />
            <input type="hidden" name="csrf" value={data.CSRF} />
            <input type="hidden" name="return_to" value="/admin/finance" />
            <label>
              Decision
              <select
                name="value"
                value={decision}
                onChange={(event) => setDecision(event.target.value)}
              >
                <option value="rejected">
                  Reject and return all reserved funds
                </option>
                <option value="approve" disabled={!canApprove}>
                  Approve and submit payout
                </option>
              </select>
            </label>
            <label>
              <input
                type="checkbox"
                name="evidence_reviewed"
                value="yes"
                required={decision === "approve"}
              />{" "}
              I reviewed the private evidence, account holder, and payout
              destination.
            </label>
            <label>
              Review notes
              <textarea name="reason" required minLength={5} maxLength={2000} />
            </label>
            <button disabled={decision === "approve" && !canApprove}>
              Record decision
            </button>
          </form>
        )}
      </main>
    </Workspace>
  );
}
