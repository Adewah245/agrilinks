# Private verification and financial security

These changes require migrations **080–084**, a stable encryption key, and administrator authenticator enrollment. The implementation and verification are recorded in the security reports.

## Deployment

1. Back up the database and existing uploads. Generate a 32-byte key with `openssl rand -base64 32` and store it as `SECURITY_ENCRYPTION_KEY` in the deployment secret manager. Use the same key on every application replica. Back up the key separately from the database; losing it makes verification photographs and authenticator secrets unreadable. Changing it requires a planned re-encryption migration, not simply replacing the environment value.
2. Build the React frontend with `npm run build`. Apply pending database migrations from `backend/` with `go run ./cmd/migrate`, then restart the backend with the configured key. Migration 079 is the preceding Nigeria branding migration; the security migrations are 080 (private evidence), 081 (MFA), 082 (limits and events), 083 (account-photo review and photo-access auditing), and 084 (photo lookup index and pending-review refresh). Migration 083 must retain its already-applied contents; it originally cleared existing photo-view acknowledgments. Migration 084 resets only pending, unsubmitted reviews, which administrators must reopen and load again. It preserves existing completed-review timestamps, photographs, MFA enrollment, withdrawal records, and reserved funds. Never edit an applied migration or replace its stored checksum.
3. Serve the application over HTTPS for camera, location, and secure sessions. If a proxy serves static files directly, apply the application's allowlist there too: only `assets/`, `css/`, `dist/`, `uploads/products/`, and `uploads/profiles/` are public. Block **all** historical `uploads/users/` URLs and the frontend source tree. Purge historical verification images from CDN/proxy caches. Previously downloaded public copies cannot be recalled. Keep the configured `FRONTEND_DIR/uploads` on durable storage shared by application replicas; registration stages and publishes its public picture on that filesystem.
4. Check startup logs and `legacy_private_photo_paths` for incomplete imports. Startup encrypts and sanitizes historical JPEG/PNG images up to 10 MB, then removes their old files; it can safely resume after interruption. Invalid, oversized-dimension, or WebP images stay quarantined and require an operator to convert/import them privately. Missing originals cannot be recovered by this migration. Nonregular files, symlinks, files above 10 MB, and storage errors stop startup for operator intervention.
5. Have every staff administrator sign in and enroll at `/admin/security`. Store recovery codes privately before leaving setup. Confirm that an ordinary member cannot retrieve another member's verification photograph, and that finance review works with the enrolled account.

Without the key, private capture/registration and privileged administrator access fail closed. An invalid key prevents startup. Ordinary public browsing remains available with an absent key. These behaviors make missing configuration visible; do not work around them by making verification files public.

## Withdrawal review

The member obtains a bank/fee quote, consents to private verification, opens the camera, and captures a selfie with fresh device GPS. Capture challenges last at most 10 minutes and are bound to the member, login session, quote, and withdrawal reference. Images are limited to 2 MB, decoded with pixel limits, resized, re-encoded as JPEG to remove metadata, and encrypted with AES-GCM. Device capture time must be recent and reported accuracy no worse than 1,000 metres.

The existing PIN, phone code, account-name, balance, and fee checks still apply. Submission atomically attaches evidence and reserves the complete debit. The request stays **pending** and makes no payout submission, including on a repeated member request.

In **Finance and payments**, a different finance administrator opens **Review evidence**, compares the withdrawal selfie with the **Original registration selfie**, checks the location/account/destination, records notes, and explicitly acknowledges the review. The backend requires an active MFA-verified session, current finance permission, and successful access to both available private photographs after the review was opened in that same session. Reopening the review resets those access checks. Replacing private registration evidence requires another review of that image; editing a public profile picture does not change the comparison original. Approval atomically claims the payout before contacting the provider. Repeated approvals cannot submit it twice. Rejection/cancellation returns reserved funds under the existing ledger rules; uncertain provider submissions retain the existing reconciliation safeguards.

Historical pending requests without evidence cannot be approved. Reject/refund requests that were never submitted and ask the member to submit again. Do not treat an already submitted or uncertain payout as safely refundable without provider reconciliation.

Camera/GPS readings are supplied by the device. They are review signals, not proof of identity, liveness, or unspoofed physical location. The system does not perform biometric matching.

## MFA and photo access

All staff administrator workspaces and sensitive administrator actions require an authenticator or single-use recovery code. This also applies to staff using the shared purchase-protection, delivery, and delivery-partner pages, and to trust-badge and partner-review actions. Ordinary members retain access through the existing ownership checks. TOTP uses six digits, 30-second steps, a one-step clock tolerance, and an atomic replay check. Verification is valid for 15 minutes in one login session and never beyond that session's expiry. Keep server clocks synchronized.

Enrollment needs the current password and is bound to its initiating session. Ten recovery codes are displayed once and stored only as hashes; recovery needs both password and an unused code. Replacing an authenticator needs fresh MFA plus password. The existing factor and recovery codes stay active until the new factor is verified. Confirmation revokes the old recovery codes and other login sessions. Abandoning setup does not turn off existing MFA. There is no password-only MFA reset endpoint.

Registration requires a camera selfie and explains both uses before submission. The server validates and re-encodes the image once, publishes a sanitized JPEG as the initial public profile picture under `uploads/profiles/`, and stores a separate AES-GCM-encrypted original with the account. Phone confirmation, the account, transaction PIN, and private original are committed together. File-storage failure prevents account creation; a failed account transaction removes its public upload. Temporary files are staged outside the public directories. Public profile edits leave the encrypted original intact for withdrawal comparisons. Registration capture and the server both enforce a 2 MB limit.

Private originals and withdrawal selfies are served through authenticated `/private/verification/{id}` requests with `no-store` and same-origin response headers. Access belongs to the owner or a finance administrator with fresh MFA. Withdrawal selfies and GPS evidence remain private. Because older images shared one directory, the migration clears those historical public profile references and any matching organization logo references; existing members may need to upload a new public photo/logo. Existing private photographs are not automatically republished.

## Monitoring and limits

Persistent database counters apply across restarts and replicas:

| Action | Limit |
| --- | --- |
| Sign-in per normalized phone | 10 per 15 minutes |
| MFA setup per account | 5 per 15 minutes |
| MFA verify/recovery per account | 5 per 5 minutes |
| Withdrawal challenge per account | 10 per hour |
| Evidence upload per account | 12 per hour |
| Private photograph requests per account | 60 per minute |
| Wallet actions per account | 30 per minute |
| Workspace administrator / finance actions per account and scope | 20 per minute |
| Purchase protection, trust, delivery, and partner actions per account (shared finance scope) | 20 per minute |

IP limits additionally cover authentication, AI, and financial POST requests (financial: 60 per minute per application instance). Set trusted proxy networks explicitly; arbitrary forwarded addresses are not trusted. Existing transaction PIN and OTP attempt limits remain in place. Database limiter failures block the action.

The finance workspace displays warning/high events from `security_events`. Events cover failed wallet actions, bank changes, MFA failures/recovery, account rate limits, and large withdrawals. Five failed wallet actions in 15 minutes produce a repeated-failure alert; concurrent failures are serialized per account so they retain all failure records and create one alert per 15-minute interval. A withdrawal of at least **₦500,000** produces a large-withdrawal alert. Structured server logs also record security activity. Successful private-photo access records the photo, owner, viewer, hashed session, purpose, and time in `private_photo_access_logs`; the image is withheld if this audit cannot be saved. Failed authorization or decryption does not count as successful access. Monitoring provides reviewable records; it does not send external notifications or automatically declare fraud. Operators should review alerts regularly and route logs to their existing monitoring system.

## Retention

Hourly cleanup deletes unused evidence after expiry plus a one-hour grace period. Finalized withdrawal evidence is removed after 90 days from the withdrawal's last update; pending and uncertain records remain available. Associated orphaned withdrawal photos are removed. Photo-access records survive image deletion with a null photo reference; they contain no images, coordinates, or raw session tokens. Registration/legacy verification photos and audit events have no automatic deletion in this implementation. Database backups and previously exported copies have separate retention managed by the operator. The 90-day period is an implementation setting, not a statement of legal requirements.

See [security verification](security-verification.md) for the regression commands and their limits.
