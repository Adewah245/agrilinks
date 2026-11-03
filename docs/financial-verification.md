# Financial implementation verification

Delivery date: 2 November 2026. This report covers the local working tree, including the earlier wallet safety, virtual-account and registration PIN work retained during the accounting implementation. Changes have not been committed or deployed. No production database or real payment was used.

## Delivered implementation

The existing `wallet_transactions` table was inspected and found to be single-entry activity history. Existing wallets and payment records were retained. The new `accounting_accounts`, `accounting_journals` and `accounting_postings` tables provide one double-entry accounting source linked to those records.

Implemented components include:

- Separate customer, escrow, seller, refund, provider, revenue, cash, statutory and reserve accounts; integer-kobo postings; database-enforced balancing and immutability; reversals and idempotency.
- Backend fee quotes and wallet confirmation/history integration. Funding credits the full requested amount, with the ₦50 service fee on top. Withdrawals pay the requested principal, with the approved service fee and disclosed provider charge on top.
- Escrow holds, delivery evidence, buyer confirmation, administrator review, disputes, releases, refunds and chargebacks integrated with the existing order/payment flow.
- Finance permissions and password/PIN/OTP checks; two-administrator company withdrawals and sensitive configuration approvals; independently approved business destinations; separate execution and settlement confirmation.
- Provider-independent treasury operations, normalized settlement imports, independent reconciliation, immutable evidence revisions, correction/reversal handling and a ledger-backed administrator dashboard.
- Date-filtered reporting and accounting CSV exports. Tax obligations have separate accounts and approved manual controls; no tax rate or tax compliance is assumed.

See [Financial operations](financial-operations.md) for the chart of accounts, transaction postings, backend entry points and operating controls. The complete [changed-file inventory](verification/2026-11-02/changed-files.txt) lists modified and added files, including preserved earlier work and verification artifacts.

## Migrations

The working tree adds migrations **067–078**:

| Migration | Purpose |
|---|---|
| `067_transaction_pins.sql` | Registration and transaction PIN controls |
| `068_payment_request_safety.sql` | Request idempotency and provider receipt ownership |
| `069_wallet_fees.sql` | Policies, quotes, fee breakdowns and funding intents |
| `070_purchase_escrow.sql` | Protected purchase states and decisions |
| `071_double_entry_accounting.sql` | Chart, journal, postings and preserved opening balances |
| `072_finance_controls.sql` | Roles, treasury, settlement staging and approvals |
| `073_accounting_integrity.sql` | Additional immutability and approval constraints |
| `074_company_settlement_evidence.sql` | Business-bank receipt evidence |
| `075_financial_reversals.sql` | Payout reversals and marketplace chargebacks |
| `076_funding_allocation.sql` | Explicit unmatched funding allocation and settlement revisions |
| `077_outbound_settlement.sql` | Payout and provider-payable reconciliation |
| `078_finance_authentication.sql` | Finance OTP purpose |

Tests apply the existing schema and these migrations to uniquely named disposable PostgreSQL databases. The test role uses a local server on port 55432. Tests do not reset an application database. Earlier migration numbers contain gaps, so the number of migration files differs from the highest version.

The historical migration fixture reconstructs the schema through 068, inspects the financial tables, seeds a ₦1,234.56 wallet balance and historical transaction, applies the remaining migrations, and checks that the balance remains exactly ₦1,234.56. Its opening journal has **123,456 kobo on each side**. Historical backing goes into unverified opening suspense and cannot be withdrawn as company cash without independent reconciliation.

## Actual test results

| Check | Result | Evidence |
|---|---|---|
| Complete backend suite, `go test ./... -count=1` | Passed; integration 88.086s, repository 17.753s | [Full suite log](verification/2026-11-02/backend-full.log) |
| Complete race suite, `go test -race ./... -count=1 -timeout 240s` | Passed; integration 177.309s, repository 23.385s; no race reports | [Final race log](verification/2026-11-02/backend-race.log) |
| Financial workflows and rollback checks | Passed; balanced postings and zero wallet projection mismatches | [Detailed workflow log](verification/2026-11-02/financial-workflows.log) |
| Simulated treasury provider execution and recovery | Passed, including ambiguous submission recovery without a second payout | [Treasury provider log](verification/2026-11-02/treasury-provider.log) |
| Historical migration preservation | Passed | [Schema inspection and migration log](verification/2026-11-02/migration-preservation.log) |
| Browser layout and interaction checks | 20/20 passed; no horizontal overflow or JavaScript exceptions | [Browser log](verification/2026-11-02/mobile-checks.log) |
| Working-tree whitespace check | `git diff --check` passed | Checked again after delivery documentation |

The full ordinary run preceded the final safety refinements. The complete final race run includes those refinements, including executor audit, destination verification/reuse and provider-fee display. The detailed workflow and migration logs are earlier diagnostic runs; the final race run also executes their current tests against the final migration set.

All database-backed runs used `TEST_DATABASE_URL`; they did not take the helper's no-database skip path. Final commands used:

```sh
cd backend
env GOCACHE=/tmp/agrilinks-go-build GOMODCACHE=/tmp/agrilinks-go-mod \
  TEST_DATABASE_URL='postgres://agrilinks_test@127.0.0.1:55432/postgres?sslmode=disable' \
  go test ./... -count=1

env GOCACHE=/tmp/agrilinks-go-build GOMODCACHE=/tmp/agrilinks-go-mod \
  TEST_DATABASE_URL='postgres://agrilinks_test@127.0.0.1:55432/postgres?sslmode=disable' \
  go test -race ./... -count=1 -timeout 240s
```

### Fee boundaries and financial coverage

| Requested payout | Verified Agrilinks Nigeria fee | Debit before any provider charge |
|---|---:|---:|
| ₦1 | ₦50 | ₦51 |
| ₦5,000 | ₦50 | ₦5,050 |
| ₦5,001 | ₦200 | ₦5,201 |
| ₦9,999 | ₦200 | ₦10,199 |
| ₦10,000 | ₦300 | ₦10,300 |

Withdrawal inputs currently accept whole naira, starting at ₦1. All internal calculations use integer kobo. Provider charges in tests are explicit simulations, not claims about the company's merchant pricing.

The suite covers funding/full requested credit, successful and failed payouts, insufficient complete debit, duplicate and concurrent requests, repeated/out-of-order provider events, reversals after spending, recovery debt, provider failures, actual-versus-quoted fees, escrow/disputes/releases/refunds/chargebacks, unauthorized and self-approvals, company cash restrictions, settlement deductions/corrections, database rollback, journal immutability and balancing.

Demonstrated cumulative postings:

| Workflow | Debits (kobo) | Credits (kobo) |
|---|---:|---:|
| Funding, withdrawals and reversals | 35,571,250 | 35,571,250 |
| Simulated company treasury | 8,055,000 | 8,055,000 |
| Historical opening balance | 123,456 | 123,456 |

These are total postings in their respective test fixtures, not actual company cash balances.

### Earlier failures and their resolution

- The negotiation-settings failure was resolved separately: the fallback and example configuration now use five rounds, matching the existing test and documented policy. The final full/race suites include that test.
- Existing wallet and commerce tests initially failed where assertions assumed no service fee or immediate seller credit. Expectations were updated for the approved fee and escrow behavior while retaining replay, balance and authorization assertions.
- A treasury authentication test exposed the missing finance OTP purpose constraint; migration 078 fixes it.
- Treasury tests exposed a cash-availability issue: funds already moved to the business bank could otherwise justify another transfer from customer-backed provider cash. Transferable provider cash now retains coverage for obligations and reserves at the provider source.
- An intermediate race attempt compiled during an executor-signature edit and failed. The subsequent complete race run above passed against the finished code.

No known test failures remain in the final recorded run.

## Browser evidence and its limits

The browser script renders the real templates with synthetic fixtures and exercises funding, withdrawal, PIN enrollment, platform revenue and escrow at widths **320, 390, 768 and 1440**. It checks fee breakdowns, PIN fields, the finance OTP control and JavaScript errors. Fetch responses are simulated. This is responsive/browser verification; it is not a live bank transfer or a physical-device test.

Screenshots:

- [Funding on mobile](verification/2026-11-02/funding-390.png)
- [Withdrawal on mobile](verification/2026-11-02/withdrawal-390.png)
- [PIN enrollment on mobile](verification/2026-11-02/enrollment-390.png)
- [Platform revenue on mobile](verification/2026-11-02/revenue-390.png)
- [Platform revenue on desktop](verification/2026-11-02/revenue-1440.png)
- [Purchase protection on mobile](verification/2026-11-02/escrow-390.png)

The production dashboard handler obtains figures from accounting postings, settlement records and withdrawal records. Browser fixtures are confined to the preview tool.

## How company money becomes withdrawable

A successful fee-bearing transaction posts service revenue separately from customer money and provider cost. Revenue recognition alone does not make cash withdrawable. Provider/bank settlement evidence must be independently reconciled into settled cash. Outstanding customer, escrow, seller, refund, provider and statutory liabilities, reserves and clearing deficits reduce availability. Treasury checks the provider source's own coverage and reserves the full transfer debit atomically.

A different authorized administrator approves the request. Execution is a separate action. Provider acceptance leaves the transfer processing; verified provider details and independently recorded business-bank receipt evidence are required for success. Principal moves between company asset accounts, while actual provider cost posts separately as expense. Approval alone never constitutes payment or settlement.

## External limitations and remaining operational work

- Flutterwave's official [transfer-fee](https://developer.flutterwave.com/reference/get-transfer-fee), [transaction-fee](https://developer.flutterwave.com/reference/get-transaction-fee) and [pricing](https://flutterwave.com/ng/pricing) documentation was reviewed. The code requests and validates provider fee quotes; it does not hardcode public rates as merchant-specific charges. **Actual company merchant fees, VAT/stamp-duty treatment, fee bearer settings and enabled payment features still require provider credentials and account verification.**
- No approved Agrilinks Nigeria business bank destination or real bank statement/feed was available. Company transfer tests use synthetic accounts and simulated provider responses. Live settlement and ownership evidence must be supplied through the implemented approval workflow.
- Automatic live provider/bank statement retrieval is not enabled. Audited normalized imports with independent reconciliation are implemented.
- Unmatched deposits that cannot cover an accepted funding allocation remain held liabilities. Automatic external refund initiation to the originating account is not implemented in that allocation path; provider-verified refund/reversal events are accounted for.
- Production deployment and migration execution remain outside this local delivery. No production funds were moved or modified.
