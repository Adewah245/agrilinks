# OTP verification with OTP.dev

Registration phone confirmation, password resets, wallet withdrawals, and wallet
transfers use OTP.dev SMS delivery. Select the provider with:

```dotenv
SMS_PROVIDER=otpdev
OTPDEV_API_KEY=your-api-key
OTPDEV_SENDER=your-sender
OTPDEV_TEMPLATE_ID=your-template-uuid
```

Set these in the server environment or an untracked `backend/.env` file when
starting the server from `backend/`. Use the sender and SMS template configured
in your OTP.dev account. The template must contain `{code}`. Keep the API key
on the backend.

`OTPDEV_BASE_URL` defaults to `https://api.otp.dev/v1`. An override is available
for testing. Startup rejects missing OTP.dev settings when `SMS_PROVIDER=otpdev`.

Agro-Shield generates a six-digit code and stores its bcrypt hash, account ID,
purpose, and expiry. It sends the same code to `POST /v1/verifications` with
`X-OTP-Key` authentication and a `data` object containing `channel`, `sender`,
`phone`, `template`, and `code`. Nigerian phone numbers are normalized to the
international digits-only format, such as `2348012345678`.

The request deliberately supplies `code` without `code_length`, as required by
the [OTP.dev SMS API](https://otp.dev/en/docs/sms-otp/). Verification happens
inside Agro-Shield against the stored hash and purpose; it does not use
OTP.dev's status lookup. This preserves the application's expiry, failed-attempt
limit, and atomic single-use checks. Codes cannot authorize a different purpose.

Optional settings:

| Variable | Default | Effect |
| --- | --- | --- |
| `OTP_TTL` | `10m` | How long Agro-Shield accepts a code |
| `OTP_MAX_ATTEMPTS` | `5` | Failed verification attempts allowed per code |
| `OTP_RESEND_INTERVAL` | `1m` | Minimum delay before issuing another code for the same account and purpose |

Avoid putting a different validity period in the SMS template. Agro-Shield's
`OTP_TTL` controls acceptance even if OTP.dev reports a later expiration.
Resending invalidates the previous code. Failed delivery removes the newly
created code, and provider response bodies are not exposed in errors.

Without `SMS_PROVIDER` configured, delivery remains disabled. Brevo remains
available for email delivery; OTP.dev SMS never falls back to email.
Signup requires a phone delivery provider; an email code cannot confirm a phone.
Identity/NIN checks are separate from phone confirmation.

## Registration

Registration now collects account details and a selfie, then requires the user
to request and enter an SMS confirmation code. Form details and the selected
photo remain in the browser if confirmation fails. Resending replaces the old
code, and changing the phone number requires a new code.

Apply migration `064_registration_phone_confirmation.sql` before starting the
updated application (`go run ./cmd/migrate` from `backend/`). It adds a separate
`registration_phone_codes` table because signup codes exist before a user ID.
The browser receives an HttpOnly, SameSite cookie; only its hash is stored, and
each challenge is tied to the normalized phone number. No account or wallet is
created until the code is confirmed.

The backend reserves each verification attempt atomically, checks the code,
then consumes it in the same transaction that creates the account. Failed
account inserts roll back consumption so the user can correct their details
and retry within the remaining attempt limit. The account stores
`phone_verified_at`; existing accounts keep their previous access and are not
retroactively marked as confirmed. Changing a stored phone clears its timestamp.

`POST /register/phone-code` issues or resends a code and is rate-limited by IP
and phone. `POST /register` requires both the browser challenge and the SMS code.
Expired signup challenges are pruned after a day when new codes are requested.

Provider contract, configuration, and service checks use simulated HTTP
responses and do not send real SMS messages:

```bash
cd backend
go test ./internal/services ./internal/config ./internal/app
```

A live delivery check requires OTP.dev credentials, a configured sender/template,
and an explicitly chosen recipient.
