package repository

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/money"
)

func (r *FinanceRepository) Page(ctx context.Context, user *models.User, from, to string) (*models.FinancePage, error) {
	if user == nil || !user.CanAdmin("revenue") {
		return nil, ErrAdminForbidden
	}
	start, end, err := reportDates(from, to)
	if err != nil {
		return nil, err
	}
	p := &models.FinancePage{User: user, CanRequest: user.CanAdmin("treasury"), CanApprove: user.CanAdmin("treasury_approve"), ReportFrom: start.Format("2006-01-02"), ReportTo: end.AddDate(0, 0, -1).Format("2006-01-02")}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT a.code,a.name,a.category,'NGN',COALESCE(SUM(p.debit_kobo::numeric),0)::bigint,COALESCE(SUM(p.credit_kobo::numeric),0)::bigint FROM accounting_accounts a LEFT JOIN accounting_postings p ON p.account_code=a.code AND p.currency='NGN' GROUP BY a.code,a.name,a.category ORDER BY a.category,a.code`)
	if err != nil {
		return nil, err
	}
	values := map[string]int64{}
	var revenue, expenses int64
	for rows.Next() {
		var a models.FinanceAccount
		if err = rows.Scan(&a.Code, &a.Name, &a.Category, &a.Currency, &a.DebitKobo, &a.CreditKobo); err != nil {
			rows.Close()
			return nil, err
		}
		a.BalanceKobo = a.DebitKobo - a.CreditKobo
		values[a.Code] = a.BalanceKobo
		if a.Category == "revenue" {
			revenue -= a.BalanceKobo
		}
		if a.Category == "expense" {
			expenses += a.BalanceKobo
		}
		p.Accounts = append(p.Accounts, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	available, provider, cash, obligations, reserves, err := availableCompany(ctx, tx)
	if err != nil {
		return nil, err
	}
	net := revenue - expenses
	backed := min(max(int64(0), net), max(int64(0), cash-obligations))
	metric := func(label string, n int64, detail string) {
		p.Metrics = append(p.Metrics, models.FinanceMetric{Label: label, Value: "₦" + money.Format(n), Detail: detail})
	}
	metric("Funding fees earned", -values["funding_fee_revenue"], "Successful funding, net of reversals")
	metric("Withdrawal fees earned", -values["withdrawal_fee_revenue"], "Successful customer payouts, net of reversals")
	metric("Other platform revenue", -values["other_platform_revenue"]-values["marketplace_commission"], "Includes released marketplace commission")
	metric("Gross revenue", revenue, "Excludes provider charge recovery and customer principal")
	metric("Provider expenses", values["provider_expense"], "Actual verified charges and settlement deductions")
	metric("Provider cost recovery", -values["provider_cost_recovery"], "Customer contributions offset expenses; never revenue")
	metric("Outstanding provider charges", -values["provider_payable"], "Unpaid provider obligations")
	metric("Expenses", expenses, "Includes operating and statutory expenses, net of provider recovery")
	metric("Net revenue", net, "Accounting revenue less recorded net expenses")
	metric("Settled revenue coverage", backed, "Net revenue backed by reconciled cash after liabilities; not a separate bank balance")
	metric("Revenue awaiting cash coverage", max(int64(0), net-backed), "Net earnings not yet backed by reconciled unrestricted cash")
	metric("Reconciled settled cash", cash, "Provider balance plus approved company bank accounts")
	metric("Available company funds", available, "Settled cash less liabilities, reserves and clearing deficits")
	metric("Available for provider transfer", provider, "Constrained to settled cash at the provider")
	metric("Outstanding liabilities", obligations, "Wallets, escrow, refunds, payouts, statutory and provider obligations")
	metric("Company operating reserves", -values["company_operating_reserve"], "Restricted by approved reserve allocations")
	metric("Refund and chargeback reserves", -values["chargeback_reserve"], "Restricted funds for reversals")
	metric("Total cash restrictions", reserves, "Reserves plus any negative provider clearing")
	metric("Provider settlement clearing", values["provider_clearing"], "Unsettled provider movements; excluded from available cash")
	metric("Unreconciled opening funds", values["opening_suspense"], "Historical opening assets require external evidence")
	var completed, pending, periodRevenue, periodExpense int64
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_kobo::numeric) FILTER(WHERE status='successful'),0)::bigint,COALESCE(SUM(total_reserved_kobo::numeric) FILTER(WHERE status IN ('pending','approved','processing')),0)::bigint FROM company_withdrawals`).Scan(&completed, &pending)
	if err != nil {
		return nil, err
	}
	metric("Completed company transfers", completed, "Transfers to the business bank; not operating expenses")
	metric("Pending company transfers", pending, "Reserved principal and quoted provider costs")
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(p.credit_kobo::numeric-p.debit_kobo::numeric) FILTER(WHERE a.category='revenue'),0)::bigint,COALESCE(SUM(p.debit_kobo::numeric-p.credit_kobo::numeric) FILTER(WHERE a.category='expense'),0)::bigint FROM accounting_postings p JOIN accounting_accounts a ON a.code=p.account_code JOIN accounting_journals j ON j.id=p.journal_id WHERE p.currency='NGN' AND j.created_at>=$1 AND j.created_at<$2`, start, end).Scan(&periodRevenue, &periodExpense)
	if err != nil {
		return nil, err
	}
	metric("Selected period revenue", periodRevenue, p.ReportFrom+" through "+p.ReportTo+" UTC")
	metric("Selected period expenses", periodExpense, "Use a single day or calendar month for daily/monthly reports")
	metric("Selected period net", periodRevenue-periodExpense, "Revenue less expenses in the selected period")
	var mismatches, unsettled int
	err = tx.QueryRowContext(ctx, `SELECT count(*) FROM wallets w WHERE (w.balance*100)::numeric<>COALESCE((SELECT SUM(p.credit_kobo::numeric-p.debit_kobo::numeric) FROM accounting_postings p WHERE p.account_code='wallet:'||w.id AND p.currency=w.currency),0)`).Scan(&mismatches)
	if err != nil {
		return nil, err
	}
	p.Metrics = append(p.Metrics, models.FinanceMetric{Label: "Wallet reconciliation exceptions", Value: strconv.Itoa(mismatches), Detail: "Cached wallet balances compared to ledger liabilities"})
	err = tx.QueryRowContext(ctx, `SELECT count(*) FROM accounting_journals j WHERE j.transaction_type IN ('wallet_funding','purchase_escrow_hold','unallocated_funding') AND EXISTS(SELECT 1 FROM accounting_postings p WHERE p.journal_id=j.id AND p.account_code='provider_clearing' AND p.debit_kobo>0) AND NOT EXISTS(SELECT 1 FROM provider_settlement_items i JOIN provider_settlement_batches b ON b.id=i.batch_id WHERE b.status='matched' AND (j.reference=i.reference||':receipt' OR j.reference=i.reference||':escrow'))`).Scan(&unsettled)
	if err != nil {
		return nil, err
	}
	p.Metrics = append(p.Metrics, models.FinanceMetric{Label: "Receipts awaiting reconciliation", Value: strconv.Itoa(unsettled), Detail: "Includes delayed, missing and mismatched settlement records"})
	rows, err = tx.QueryContext(ctx, `SELECT `+companyWithdrawalColumns+` FROM company_withdrawals ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		w, e := scanCompanyWithdrawal(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		p.Withdrawals = append(p.Withdrawals, *w)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT `+changeColumns+` FROM finance_change_requests ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		c, e := scanChange(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		p.Changes = append(p.Changes, *c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT `+settlementColumns+` FROM provider_settlement_batches ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		b, e := scanSettlement(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		p.Settlements = append(p.Settlements, *b)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT `+escrowSelect+escrowJoins+`WHERE e.state IN ('held','disputed') ORDER BY e.created_at LIMIT 100`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		e, scanErr := scanEscrow(rows)
		if scanErr != nil {
			rows.Close()
			return nil, scanErr
		}
		p.Escrows = append(p.Escrows, *e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	p.Policy, err = r.TransferPolicy(ctx)
	if err != nil {
		return nil, err
	}
	if p.Policy.CompanyAccountID.Valid {
		p.CompanyAccount, err = r.CompanyAccount(ctx, p.Policy.CompanyAccountID.Int64)
		if err != nil {
			return nil, err
		}
	}
	p.FeePolicy, err = NewWalletRepository(r.db, "NGN", 0).FeePolicy(ctx)
	return p, err
}
func reportDates(from, to string) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	if from == "" {
		from = now.Format("2006-01") + "-01"
	}
	if to == "" {
		to = now.Format("2006-01-02")
	}
	a, err := time.Parse("2006-01-02", from)
	if err != nil {
		return a, a, adminInvalid("choose a report start date")
	}
	b, err := time.Parse("2006-01-02", to)
	if err != nil || b.Before(a) || b.Sub(a) > 366*24*time.Hour {
		return a, b, adminInvalid("choose an end date within one year of the start")
	}
	return a, b.AddDate(0, 0, 1), nil
}
func (r *FinanceRepository) Export(ctx context.Context, user *models.User, out io.Writer, from, to string) error {
	if user == nil || !user.CanAdmin("reports") {
		return ErrAdminForbidden
	}
	a, b, err := reportDates(from, to)
	if err != nil {
		return err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT j.reference,j.transaction_type,j.created_at,j.status,COALESCE(j.actor_id::text,'system'),j.evidence::text,p.account_code,p.currency,p.debit_kobo,p.credit_kobo FROM accounting_journals j JOIN accounting_postings p ON p.journal_id=j.id WHERE j.created_at>=$1 AND j.created_at<$2 ORDER BY j.id,p.line`, a, b)
	if err != nil {
		return err
	}
	defer rows.Close()
	w := csv.NewWriter(out)
	if err = w.Write([]string{"reference", "type", "timestamp_utc", "status", "actor", "source_evidence", "destination_account", "currency", "debit_kobo", "credit_kobo"}); err != nil {
		return err
	}
	for rows.Next() {
		var reference, kind, status, actor, evidence, account, currency string
		var timestamp time.Time
		var d, c int64
		if err = rows.Scan(&reference, &kind, &timestamp, &status, &actor, &evidence, &account, &currency, &d, &c); err != nil {
			return err
		}
		line := []string{reference, kind, timestamp.UTC().Format(time.RFC3339Nano), status, actor, evidence, account, currency, fmt.Sprint(d), fmt.Sprint(c)}
		for i, v := range line {
			if strings.ContainsAny(strings.TrimLeft(v, " \t\r\n")[:min(1, len(strings.TrimLeft(v, " \t\r\n")))], "=+-@") {
				line[i] = "'" + v
			}
		}
		if err = w.Write(line); err != nil {
			return err
		}
	}
	w.Flush()
	if err = rows.Err(); err != nil {
		return err
	}
	return w.Error()
}
