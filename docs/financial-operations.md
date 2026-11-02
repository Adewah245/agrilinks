# AgriLinks financial operations

Implemented locally on 2 November 2026. Development and verification use disposable PostgreSQL databases and simulated provider responses. No production financial database was connected to or changed.

## Existing architecture and accounting boundary

Inspection of the Go repositories, migrations and a database built from migrations through 068 established that `wallet_transactions` was customer activity history: one amount and transaction type per row. It was not a double-entry ledger. `platform_revenue` was an order-commission reporting projection.

The existing wallets, deposit/withdrawal records, provider receipt ownership, order payments, PINs and recovery logic remain in use. `accounting_journals` and `accounting_postings` supply the single double-entry accounting ledger linked to those original references. There is no second wallet or parallel payment processor. `wallet_fee_records` and `platform_revenue` are transaction/reporting projections; accounting totals come from the journal.

Posted journals and postings cannot be updated or deleted. PostgreSQL deferred constraints enforce equal debits and credits per currency and at least two lines. A trigger prevents appending new lines to a journal in a later transaction. Corrections use new journals with stable references; whole-journal reversals link to the original. Unique references and canonical posting fingerprints reject reuse with different amounts. Entries include currency, destination account, source, transaction type, posting status, timestamp, actor and evidence.

All new money arithmetic uses integer kobo. Decimal strings are used only at UI, provider and existing PostgreSQL `NUMERIC` boundaries. Cached wallet balances are updated in the same database transaction as their journal and are reconciled in the dashboard. Debit operations refuse a cached balance that does not match the wallet liability account.

## Chart of accounts

| Purpose | Account codes | Category |
|---|---|---|
| Individual and organization wallets | `wallet:<wallet_id>` | Liability |
| Unconfirmed funding instructions | `pending_customer_deposits`, paired with `pending_provider_collections` | Memorandum liability/asset; excluded from cash |
| Unallocated incoming funding | `funding_unmatched` | Liability |
| Customer withdrawal payout and fees reserved | `withdrawal:<reference>`, `fees:<reference>` | Liability |
| Wallet transfers awaiting decision | `transfer:<reference>` | Liability |
| Customer refunds payable | `customer_refunds_payable` | Liability |
| Protected purchases | `escrow:<order_id>` | Liability |
| Disputed protected purchases | `disputed-escrow:<order_id>` | Liability |
| Seller and marketplace refunds pending credit | `seller_payouts_payable`, `marketplace_refunds_payable` | Liability |
| AgriLinks revenue | `funding_fee_revenue`, `withdrawal_fee_revenue`, `other_platform_revenue`, `marketplace_commission` | Revenue |
| Unsettled provider movements | `provider_clearing` | Asset/clearing |
| Reconciled cash | `provider_settled_cash`, `company-bank:<approved_account_id>` | Asset |
| Company transfer reservations | `company_transfer_reserved` | Restricted asset |
| Provider costs | `provider_expense`, `provider_cost_recovery` | Expense/contra-expense; never platform revenue |
| Provider obligations | `provider_payable` | Liability |
| Company and statutory costs | `company_operating_expense`, `statutory_expense` | Expense |
| Assessed statutory obligations | `tax_statutory_payable` | Liability |
| Reserves | `company_operating_reserve`, `chargeback_reserve`, `reserve_allocation` | Restricted equity allocations |
| Reversal recovery debt | `wallet-receivable:<wallet_id>` | Asset; never spendable customer money |
| Historical assets not yet externally verified | `opening_suspense` | Asset; excluded from available cash |

Migration 071 preserves existing wallet balances and pending reservations with balanced opening journals. The offset is unverified opening suspense, not a claim that a bank or provider holds that cash. Reconciliation of opening funds requires independent approval and statement evidence. No historical seller credit is retroactively confiscated.

## Approved fees and money movement

Funding service fee: **₦50 on top**. A requested wallet credit of ₦10,000 remains ₦10,000. With an actual disclosed provider charge of ₦140, the total payment is ₦10,190.

| Funding posting | Debit | Credit |
|---|---:|---:|
| Provider clearing | ₦10,190 | — |
| Customer wallet liability | — | ₦10,000 |
| AgriLinks funding revenue | — | ₦50 |
| Provider cost recovery | — | ₦140 |
| Provider expense, when verified | ₦140 | — |
| Provider clearing, actual cost | — | ₦140 |

The two journals balance independently. Net clearing is ₦10,050, covering ₦10,000 owed to the customer and ₦50 company revenue. The example provider fee is illustrative, not a configured rate.

| Requested withdrawal | AgriLinks fee |
|---|---:|
| ₦1–₦5,000 | ₦50 |
| ₦5,001–₦9,999 | ₦200 |
| ₦10,000+ | ₦300 |

Withdrawals currently accept whole naira starting at ₦1. A ₦10,000 request pays the bank ₦10,000. The wallet reserves ₦10,300 plus the separately disclosed provider charge. Reservation debits the wallet liability and credits payout and deferred-fee liabilities. Verified success clears the payout, recognizes service revenue, and posts actual provider cost separately. A lower actual provider charge is refunded to the customer. An undisclosed excess is borne by the company, never silently charged to the wallet.

Failed withdrawals reverse the complete reservation. Any actual failed-transfer cost remains a company expense. Verified payout reversals return the customer's payout and customer fees using reversal postings. Repeated or out-of-order outcomes cannot create a second credit or fee.

Funding is recognized only on verified success. Pending/failed attempts earn no fee. A verified funding reversal reverses the receipt and fee revenue. If the credited money was spent, the system records a recovery receivable, sets wallet recovery debt and blocks new debits. Later credits first repay that debt. Provider charges remain expenses unless there is evidence that they were returned.

Permanent-account users confirm a funding instruction before sending the exact amount. Unmatched incoming funds stay in `funding_unmatched`, with no service revenue. The owner can review and explicitly accept a new allocation quote before wallet credit and a service fee are posted. Deposits too small to cover the fee remain held for return/investigation; automatic external refunds to the originating account are not implemented by this allocation path.

## Purchase protection

Existing wallet and hosted checkout payments post into escrow, notify the seller, and leave seller spendable funds unchanged. Seller delivery evidence or completed tracked delivery is required. The buyer must confirm receipt. A finance-authorized administrator then checks the stored buyer, seller, order items, agreed total, currency, verified payment, delivery, confirmation and support disputes before release. An administrator cannot approve their own purchase/sale.

Release posts escrow to seller payouts payable and the agreed commission, then clears the seller payable into the existing seller wallet. Refunds use marketplace refunds payable and credit the buyer without a new funding fee. Disputes reclassify the protected amount into a separate frozen liability. Resolving a dispute requires fresh buyer confirmation. Support disputes share the order lock with releases. Repeated releases/refunds are idempotent; decisions record actor, time, reason and order/payment references.

Provider chargebacks remove held liabilities or recover previously released/refunded wallet credits. Spent amounts become recovery debt. A chargeback cannot leave both a wallet refund and an external refund available to the buyer.

## Company cash and two-administrator controls

Available company cash is reconciled provider/business-bank cash less positive customer/seller/escrow/refund/provider/statutory liabilities, approved reserves and any provider-clearing deficit. Unsettled clearing, recovery receivables and opening suspense are excluded. Company transfer reservations immediately reduce available settled cash.

Transferable provider cash is constrained further: all obligations and reserves remain covered at the provider source. Previously transferred business-bank cash cannot be used to justify draining customer-backed provider funds. This is intentionally conservative when funds are held across several bank accounts.

Company withdrawals follow:

1. Live provider quote and resolved approved business destination.
2. Initiator authenticates with current password, transaction PIN and one-time finance code.
3. Atomic reservation, limits and available-cash checks; status `pending`.
4. A different senior finance/super administrator reviews and records a reason; status `approved`.
5. Separate execution action rechecks roles, current transfer policy, destination and obligations, durably claims submission, and records the executor.
6. Provider acceptance becomes `processing`. A timeout stays reserved and is recovered by reference; it never authorizes a second submission.
7. Provider identity, reference, amount, currency and bank destination are verified. A separate administrator confirms the exact business-bank receipt with a unique statement reference before status becomes `successful`.
8. Principal moves from reserved cash to company-bank cash. Provider expense is separate. Failed/reversed transfers create appropriate reversal postings.

Company transfers move assets; their principal is not an operating expense. The company bank must resolve to the legal company name and have business ownership evidence. A second super administrator approves destination changes. No personal account is hardcoded. Reusable approved destinations remain immutable, and new policy records retain their approval history.

Support staff have no treasury access. Finance administrators can initiate; senior finance/super administrators approve. Thresholds can require a super administrator. Fee policies, provider fee treatment, transfer limits, bank destinations, reserves, opening reconciliation and statutory obligations use pending change requests and a distinct approver. Effective-dated fee policies are immutable after creation. No tax rates or tax-compliance claim are supplied.

## Settlement and reporting

The `SettlementProvider` interface isolates transfer quotes, bank resolution, execution and verification from treasury accounting. Flutterwave is the configured adapter; tests inject independent simulated responses.

Normalized statement import supports collections, customer payouts and provider payables. It stores provider references, gross amounts, actual deductions, net movements, destination and bank/provider statement evidence. Every import revision is retained. A separate approver compares constituent records to the ledger. Duplicate references, missing receipts, reversed payments and mismatched amounts cannot create available cash. Corrected unposted mismatches require a new independent review. Matched settlements are reversed with new journals.

`/admin/revenue` reads accounting and settlement records. It reports fee income, other revenue, provider expenses/recovery/payables, gross/net revenue, cash coverage, available company funds, reserves, pending/completed company transfers, wallet mismatches and unreconciled receipts. **Settled revenue coverage** is cash-backed net income after obligations; it is not presented as a separate bank balance. Daily/monthly reporting uses date filters. CSV exports contain complete journal lines for accounting review, including tax, refunds, chargebacks and escrow accounts.

Live statement retrieval is not enabled. The implemented import and independent reconciliation workflow is available when API statement retrieval or bank feeds are unavailable.

## Backend entry points and interfaces

| Route | Operations |
|---|---|
| `GET/POST /wallet` | Server fee quotes, funding confirmation/allocation, own-bank withdrawal, PIN and existing wallet operations |
| `GET /wallet/status` | Existing committed-state polling |
| `GET/POST /orders/protection?order_id=…` | Delivery, buyer confirmation, dispute, administrator release/refund and decision history |
| `GET/POST /admin/revenue` | Accounting dashboard, company quotes/requests/approvals/execution/receipt confirmation, controlled changes, statement imports/reconciliation |
| `GET /admin/revenue?export=ledger&from=…&to=…` | Accounting CSV export |
| `POST /webhooks/flutterwave` | Authenticated provider event routing to existing wallet/order flow or company transfer verification |

Core implementation: `repository/accounting.go`, `wallet_fees.go`, `wallet_allocation.go`, `financial_reversals.go`, `escrow.go`, `escrow_disputes.go`, `finance_repository.go`, `finance_controls.go`, `finance_settlements.go`, `finance_reports.go`; `services/wallet_fees.go`, `finance_service.go`; `handlers/finance.go`, `escrow.go`, and the extended existing wallet/payment handlers/services.

## Migrations

All are additive extensions of the existing financial structures, with constraint replacements where required. Run only after reviewing and backing up the target environment through its normal deployment process. Development executed them only on disposable databases.

| Migration | Purpose |
|---|---|
| 067 | Transaction PINs and registration confirmation |
| 068 | Payment request safety and shared provider receipt ownership |
| 069 | Fee policies, quotes, breakdowns, funding intents and recovery debt |
| 070 | Purchase escrow and immutable decision events |
| 071 | Chart of accounts, immutable balanced journal, company accounts and preserved opening balances |
| 072 | Finance roles, two-person changes, transfer policies, company withdrawals and settlement staging |
| 073 | Journal/quote/account immutability and company approval integrity |
| 074 | Unique, immutable business-bank receipt confirmations |
| 075 | Withdrawal reversals and marketplace chargeback records |
| 076 | Explicit funding allocation and immutable settlement import revisions |
| 077 | Outbound settlement and provider-payable reconciliation |
| 078 | Finance OTP purpose in the existing OTP system |

## Provider access and pricing evidence

Reviewed official sources:

- https://developer.flutterwave.com/reference/get-transfer-fee
- https://developer.flutterwave.com/reference/get-transaction-fee
- https://flutterwave.com/ng/pricing

The adapter requests `/transfers/fee` and `/transactions/fee`, validates currency and fee shape, and saves the quoted evidence. Provider verification supplies actual transfer/collection fees; statements reconcile deductions. Public pricing is not hardcoded as merchant-specific cost.

No application database connection, live Flutterwave credentials, approved company business destination or bank feed was available in this development environment. Consequently **actual merchant-account pricing, VAT/stamp-duty treatment, fee bearer settings, business-account enablement and live settlements are not confirmed**. They require the company's provider access and independent bank evidence. Unknown or conflicting outcomes stay pending/restricted. Testing uses explicitly simulated fees and receipts, not real bank transfers.

## Verification

Use a disposable PostgreSQL server and a role with `CREATEDB`. Tests create and drop unique `agrilinks_check_*` databases; they do not reset an existing application database.

```sh
cd backend
TEST_DATABASE_URL='postgres://test_role@127.0.0.1:55432/postgres?sslmode=disable' go test ./... -count=1
TEST_DATABASE_URL='postgres://test_role@127.0.0.1:55432/postgres?sslmode=disable' go test -race ./... -count=1
go run ./tools/finance-preview
cd ..
node scripts/check-finance-mobile.mjs
```

Financial tests cover approved fee boundaries, full credit/payout, insufficient complete debit, lower actual provider cost, duplicate/concurrent requests, repeated and out-of-order outcomes, failed/reversed withdrawals, spent deposit reversals, debt recovery, journal immutability/balance checks, database rollback, escrow/disputes/refunds/chargebacks, company permissions and two-person decisions, uncertain provider submission recovery, bank receipt verification, reserves, settlement deductions, corrections, outbound reconciliation and wallet-ledger consistency.

The migration test reconstructs the pre-accounting schema, logs existing financial columns, seeds historical funds and verifies they remain exactly ₦1,234.56 after migration, backed by equal opening debits/credits of 123,456 kobo.

The negotiation-settings failure was resolved separately: the configured fallback is five rounds, matching the documented policy and existing test. Existing financial tests were updated for the approved fee/escrow behavior without removing their prior replay and permission checks.

See `financial-verification.md` for final run results and artifact locations.
