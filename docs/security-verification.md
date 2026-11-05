# Security and financial regression verification

This report covers local security and financial verification on 5 November 2026: private withdrawal evidence, mandatory review before payout submission, administrator MFA, private/public photo separation, persistent rate limits, and financial security events. The review closed MFA gaps on shared member/staff routes, enforced access to the current account photograph before withdrawal approval, and added detailed private-photo auditing. Deployment requirements and limitations are in [security operations](security-operations.md).

## Results

| Check | Result |
| --- | --- |
| Complete backend suite, `go test ./... -count=1 -timeout 360s` | Passed; integration 233.429s, repository 37.794s |
| Final complete race-detector suite, `go test -race ./... -count=1 -timeout 360s` | Passed across all packages; integration 224.002s, repository 22.551s; no race reports |
| Frontend production build, React test renderer, and frontend tests | Passed |
| `go vet ./...` and `git diff --check` | Passed |
| Financial/security browser checks | Passed: 28 layouts across 320, 390, 768, and 1440 px, plus security interactions |
| All-page React browser regression | Passed: 70 layouts (35 pages at mobile/desktop widths), navigation, registration, and existing interactive controls |

The PostgreSQL-backed suites used a temporary local PostgreSQL 16.15 server on `127.0.0.1:55439`. Each test created and dropped its own uniquely named database; database tests were not skipped. The run used Go 1.25.0, Node 26.13.0, and Chrome 145. No production database, real identity image, or live payout was used. Flutterwave and phone-code responses were simulated.

Recorded evidence: [backend suite](verification/2026-11-05/backend-full.log), [final complete race suite](verification/2026-11-05/backend-race-final.log), [static analysis](verification/2026-11-05/backend-vet.log), [financial/security browser checks](verification/2026-11-05/financial-security-browser.log), and [all-page browser checks](verification/2026-11-05/react-browser.log). Mobile screenshots show [MFA setup](verification/2026-11-05/admin-security-390.png), [private review with synthetic placeholders](verification/2026-11-05/withdrawal-review-390.png), and [withdrawal verification](verification/2026-11-05/withdrawal-390.png).

The new boundary regressions first reproduced missing MFA enforcement, approval without opening the account photograph, and missing persistent action limits. They passed after the fixes. The final full race run includes the last migration refinement: preserving completed withdrawals' historical photo-access timestamps while resetting only pending, unsubmitted reviews. The earlier full race run is also retained in [the initial race log](verification/2026-11-05/backend-race.log).

## Coverage

- AES-GCM randomized nonces, ciphertext tamper rejection and authenticated context; image validation, bounded decoding, metadata/payload removal, and private encryption.
- Withdrawal challenge ownership/session/quote binding, recent GPS and accuracy checks, capture-token reuse rejection, private-image access restrictions, expiry cleanup, and migration of older public photographs.
- Zero provider submissions while awaiting review; acknowledgment and successful private-image access before approval; concurrent claims, replay/idempotency, and existing failed/uncertain/reversed payout handling.
- Both available private photographs must be accessed after opening the review, in that same session. A changed account photograph invalidates the old review. Payout claims recheck current finance permission, including explicit roles overriding legacy configuration grants.
- Authenticator standard test vectors; password-protected enrollment; session binding, concurrent verification and replay rejection, single-use recovery, MFA expiry, and protection during interrupted authenticator replacement. Replacement invalidates prior recovery codes and other sessions only after confirmation.
- Staff MFA enforcement on purchase protection, delivery administration, partner review, and trust badges. A real escrow refund is blocked without fresh MFA and credits the buyer exactly once after verification; ordinary member access continues through ownership checks.
- Persistent rate limits under concurrent requests and application restart, including private-photo requests and shared financial action limits. Concurrent financial failures retain all underlying events and create one alert per detection interval.
- Private-photo audit records identify the owner, viewer, hashed session, purpose, and time. Audit storage failure withholds the image, unauthorized or undecryptable reads do not count as successful access, and audit records survive image retention cleanup.
- Migration 083 is pinned to its already-applied checksum. Migration 084 applies safely and repeatedly over schema 083, preserving reserved funds, pending withdrawals, private evidence, MFA sessions, and existing completed-withdrawal photo-access history. Pending, unsubmitted reviews require fresh image access. See the [migration recovery report](verification/2026-11-05/migration-083-recovery.md).
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

The local run used `GOMODCACHE=/tmp/agrilinks-go-mod`, `GOCACHE=/tmp/agrilinks-go-build`, and `GOPATH=/tmp/agrilinks-go`. Browser scripts default to `/usr/bin/google-chrome`; set `CHROME` to override. The finance preview tool generates synthetic pages and placeholder images without connecting to a database or payment provider. Screenshots are written to the preview directory. Install frontend dependencies with `npm run setup` before the first run.

## Limits

This is automated regression testing and code review, not an independent penetration test or a live-provider certification. Browser tests use desktop Chrome emulation; physical camera/location permissions on members' devices still depend on browser support and HTTPS. Device evidence is not spoof-proof and the implementation does not perform biometric/liveness verification. Production key management, migrations, cache quarantine, staff enrollment, and ongoing alert review remain deployment/operator work.
