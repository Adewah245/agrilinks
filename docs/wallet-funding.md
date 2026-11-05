# Wallet funding accounts

Agrilinks Nigeria adapts the permanent NGN account workflow from
[Agro-Shield](https://github.com/etoosammy01/Agro-Shield/tree/de67bc3bd405),
using the upgraded `users`, `wallets`, ledger and staff permission model.

## User flow

The wallet page separates the permanent **funding account** from the saved
**withdrawal account**. Users with a full name and valid email can submit an
11-digit BVN or NIN and explicit consent. The identity number goes to
Flutterwave and is never stored in the database, returned in a page, put in a
URL, or written to browser storage. Format validation is not a claim of full
identity verification.

The page displays setup, pending, failed, unavailable and active states. Pending
accounts expose a status check rather than another creation form. Active bank
details can be copied. The browser checks `/wallet/status` every 15 seconds
while visible and reloads when committed state changes. If a form has been
edited, it offers a refresh instead of discarding the user's input.

## Configuration

Use the existing `FLUTTERWAVE_SECRET_KEY`, `FLUTTERWAVE_SECRET_HASH`,
`FLUTTERWAVE_BASE_URL`, `PAYMENT_REDIRECT_URL` and `PAYMENT_CURRENCY=NGN` settings.
The merchant account must support Flutterwave permanent virtual accounts.
Configure Flutterwave to deliver authenticated `charge.completed` events to
`POST /webhooks/flutterwave` using the configured verification hash.

If startup logs `wallet funding reconciliation: funding account ... verification:
the payment provider did not permit the request`, Flutterwave returned HTTP
401 or 403 while the worker checked a payment. Check the API secret key, its
test/live environment, and merchant API permissions. The
`FLUTTERWAVE_SECRET_HASH` is for webhook checks and is not the API secret key.
Update server configuration and restart the backend after correcting it.
This warning does not stop browsing; refused verification cannot credit a
wallet, and pending funding remains available for later checks.

Migrations 043–046, 062 and 066 provide wallet funding and recovery fields.
Apply migrations 067 and 068 before running the updated application; they add
transaction PINs, verified registration challenges, durable checkout claims,
and shared provider receipts. These migrations preserve existing balances.

## Verification and recovery

- Setup is claimed in a database transaction before calling the provider.
  Concurrent requests reuse one account; retries have a two-minute cooldown
  measured by the database clock. Manual/background account checks share a
  30-second lease.
- A lost creation response remains pending. Lookup uses the saved provider
  `order_ref` when available. A missing result for the application's `tx_ref`
  alone does not authorize another creation: those references are not
  interchangeable in the provider API.
- Partial responses retain the provider order reference for later checks.
  Mismatched references, unsupported response codes, and inactive accounts
  are never displayed as ready funding accounts.
- Every credit uses a server-verified transaction ID, stored account reference,
  currency and decimal amount. Each bank transfer has a separate `VA-<id>`
  deposit/ledger reference. Balance and ledger writes are atomic, and repeated
  delivery cannot create a second credit.
- A verified payment can arrive before account setup finishes locally. The
  stored unique account reference establishes ownership in that case.
- Pending transactions retain their provider IDs. A pending verification does
  not consume the webhook event permanently; later successful delivery can
  settle it.
- A worker runs on startup and every minute, with a 45-second pass timeout and
  bounded batches. It refreshes pending accounts, rechecks pending deposits,
  and rotates active accounts through provider reference verification.

Flutterwave reference verification returns one transaction, so it cannot
guarantee recovery of every historical transfer when several webhooks were
missed. In **Admin → Finance → Check a missing bank transfer**, authorized
finance staff can enter the provider transaction ID and investigation notes.
The server verifies ownership and payment state before applying anything.
Attempts and outcomes are recorded in the existing staff audit log.

Finance also lists funding account states and setup references. If creation
remains pending without an order reference, investigate that reference with
Flutterwave before retrying or changing the stored account. Never request a
user's BVN/NIN through a support message.

Verified funding reversals now create balanced reversal entries. Available funds
are removed from the wallet; spent amounts become recovery debt and block further
debits. See [Financial operations](financial-operations.md) for the fee, accounting,
settlement and escrow workflows.

## Transaction PINs and request safety

Registration now requires a successful `/register/phone-verify` response before
PIN creation. Final account creation consumes the verified, unexpired challenge
and stores the six-digit PIN hash in one database transaction. Resending the
phone code invalidates the old proof. A PIN mismatch does not consume the proof.
The PIN and confirmation are never included in logs, URLs, or browser storage.

Existing members create their first PIN in Wallet using their current account
password. Wallet payments, transfers, withdrawals, and bank-account updates
require the PIN. Transfers and withdrawals also retain their existing one-time
code requirement. Five wrong PIN attempts lock verification for 15 minutes;
attempts are serialized in PostgreSQL across server instances. This change
does not introduce a PIN reset or replacement endpoint.

Withdrawal and transfer keys bind the operation to its original owner, amount,
and destination. Retries reuse the saved operation; changed details are rejected.
The reservation, operation, and ledger commit together. A pending or processing
withdrawal blocks another withdrawal even with a different key. A completed
wallet order payment can be retried without another debit.

Online top-ups and order checkouts durably claim initialization before contacting
Flutterwave. Another online top-up cannot be started while one remains pending;
its saved checkout link appears in funding history. A lost initialization
response remains unresolved and must be reconciled by provider reference. A
timeout or a provider lookup returning no result does not authorize resubmission.
Do not clear checkout claims simply because they are old.

Payout submission is durably marked `unknown` before the provider call and
`accepted` only after its transfer ID is saved. The transfer ID cannot be
replaced, and uncertain payouts cannot be refunded without verified failure.
Shared provider receipt ownership prevents the same provider transaction from
being applied to both an order and a wallet deposit. Existing cross-domain
conflicts are quarantined and flagged for finance review by migration 068.

Funding now charges the approved ₦50 service fee on top of the requested wallet
credit. Withdrawals charge the approved ₦50/₦200/₦300 tiers on top of the payout.
Server-generated quotes show the complete breakdown. Own-bank withdrawals require
a private selfie and current GPS evidence, followed by review by a different finance
administrator with fresh MFA before payout submission. Purchases remain protected
in escrow until delivery, buyer confirmation and administrator release. Apply migrations
069–078 for the financial foundation and 080–084 for private verification and staff
security. See [Financial operations](financial-operations.md) and [Security operations](security-operations.md).

## Checks

Run from `backend/` with `TEST_DATABASE_URL` pointing at a disposable PostgreSQL
server whose role can create test databases:

```sh
go test ./internal/services ./internal/handlers ./internal/integration -run 'TestFlutterwave|TestWallet|TestRegistration|TestTransactionPIN|TestConcurrentCheckout'
go test -race ./internal/services ./internal/integration -run 'TestFlutterwave|TestWallet|TestRegistration|TestTransactionPIN|TestConcurrentCheckout'
```

The integration tests exercise the real database with a simulated Flutterwave
server, including creation concurrency, consent/privacy, account response
validation, delayed activation, uncertain creation, pending-to-paid delivery,
multiple transfers to one account, wrong currency, background recovery,
finance permissions, duplicate recovery and reversal review. Merchant
enablement and real bank settlement still require validation in the configured
Flutterwave environment.
