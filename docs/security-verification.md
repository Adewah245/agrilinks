# Security and financial regression verification

This report covers the local security implementation on 4 November 2026. It adds private withdrawal evidence, mandatory review before payout submission, administrator MFA, private/public photo separation, persistent rate limits, and financial security events. Deployment requirements and limitations are in [security operations](security-operations.md).

## Results

| Check | Result |
| --- | --- |
| Complete backend suite, `go test ./... -count=1 -timeout 360s` | Passed; integration 158.405s, repository 29.401s |
| Complete race-detector suite | Pending final result |
| Frontend production build, React test renderer, and frontend tests | Passed |
| `go vet ./...` | Passed |
| Financial/security browser checks | Passed: 28 layouts across 320, 390, 768, and 1440 px, plus security interactions |
| All-page React browser regression | Pending final result |

The PostgreSQL-backed suites used a temporary local PostgreSQL 16 server on `127.0.0.1:55439`. Each test created and dropped its own uniquely named database; database tests were not skipped. No production database, real identity image, or live payout was used. Flutterwave and phone-code responses were simulated.

## Coverage

- AES-GCM randomized nonces, ciphertext tamper rejection and authenticated context; image validation, bounded decoding, metadata/payload removal, and private encryption.
- Withdrawal challenge ownership/session/quote binding, recent GPS and accuracy checks, capture-token reuse rejection, private-image access restrictions, expiry cleanup, and migration of older public photographs.
- Zero provider submissions while awaiting review; acknowledgment and successful private-image access before approval; concurrent claims, replay/idempotency, and existing failed/uncertain/reversed payout handling.
- Authenticator standard test vectors; password-protected enrollment; session binding, concurrent verification and replay rejection, single-use recovery, MFA expiry, and protection during interrupted authenticator replacement. Replacement invalidates prior recovery codes and other sessions only after confirmation.
- Persistent rate limits under concurrent requests and application restart, plus repeated financial-failure alerts.
- The existing financial suite: balanced/immutable double-entry postings, fee tiers, complete debits, rollback, duplicate and concurrent withdrawals, ledger/wallet consistency, escrow, refunds/reversals, treasury authorization and settlement/reconciliation.
- Browser capture with a fake camera and simulated device GPS; permission denial, upload failure, consent revocation, token gating, camera shutdown, authenticator enrollment/invalid-code/recovery forms, and approval disabled when private images fail. Browser MFA responses are mocked; real TOTP/recovery enforcement is covered by backend integration tests.

## Reproduce

Build the browser and React test-renderer bundles **before** starting backend tests; rebuilding them concurrently temporarily removes files that backend rendering tests need.

```sh
npm --prefix frontend test

cd backend
# Use an isolated PostgreSQL server and a role permitted to create test databases.
export TEST_DATABASE_URL='postgres://student@127.0.0.1:55439/postgres?sslmode=disable'
go test ./... -count=1 -timeout 360s
REACT_PREVIEW_DIR=/tmp/agrilinks-react-preview go test -race ./... -count=1 -timeout 360s
go vet ./...
go run ./tools/finance-preview
cd ..
node scripts/check-react.mjs
node scripts/check-finance-mobile.mjs
git diff --check
```

The local run used `GOMODCACHE=/tmp/agrilinks-go-mod` and `GOCACHE=/tmp/agrilinks-go-build`. Browser scripts default to `/usr/bin/google-chrome`; set `CHROME` to override. The finance preview tool generates synthetic pages and placeholder images without connecting to a database or payment provider. Screenshots are written to the preview directory.

## Limits

This is automated regression testing and code review, not an independent penetration test or a live-provider certification. Browser tests use desktop Chrome emulation; physical camera/location permissions on members' devices still depend on browser support and HTTPS. Device evidence is not spoof-proof and the implementation does not perform biometric/liveness verification. Production key management, migrations, cache quarantine, staff enrollment, and ongoing alert review remain deployment/operator work.
