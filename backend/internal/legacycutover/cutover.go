// Package legacycutover transfers the original farmer/crop schema without deleting it.
package legacycutover

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"backend/internal/database"
	"backend/scripts"
)

type queryer interface {
	QueryRow(query string, args ...any) *sql.Row
}

// Run must be called while the application is stopped. A completed cutover is
// never repeated: its imported balances may have changed through real payments.
func Run(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtextextended('agro-shield:legacy-cutover',0))`); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(hashtextextended('agro-shield:legacy-cutover',0))`)
	var hasState bool
	if err = db.QueryRow(`SELECT to_regclass('legacy_agro_shield.cutover_state') IS NOT NULL`).Scan(&hasState); err != nil {
		return err
	}
	if hasState {
		var done bool
		if err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM legacy_agro_shield.cutover_state WHERE completed_at IS NOT NULL)`).Scan(&done); err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	if err = Preflight(db); err != nil {
		return err
	}
	var archived bool
	if err = db.QueryRow(`SELECT to_regclass('public.farmers') IS NULL`).Scan(&archived); err != nil {
		return err
	}
	if !archived {
		var current bool
		if err = db.QueryRow(`SELECT to_regclass('public.users') IS NOT NULL`).Scan(&current); err != nil {
			return err
		}
		if current {
			return fmt.Errorf("both old and new account tables exist; resolve this mixed database before transferring")
		}
		if err = runScript(db, "legacy_archive.sql"); err != nil {
			return err
		}
	}
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS legacy_agro_shield.cutover_state(singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK(singleton),started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,completed_at TIMESTAMPTZ);
        INSERT INTO legacy_agro_shield.cutover_state(singleton) VALUES(TRUE) ON CONFLICT DO NOTHING`); err != nil {
		return err
	}
	if err = database.RunMigration(db); err != nil {
		return err
	}
	script, err := scripts.Files.ReadFile("legacy_backfill.sql")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, string(script)); err != nil {
		return fmt.Errorf("backfill legacy data: %w", err)
	}
	if err = Verify(tx); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE legacy_agro_shield.cutover_state SET completed_at=CURRENT_TIMESTAMP WHERE singleton`); err != nil {
		return err
	}
	return tx.Commit()
}

func runScript(db *sql.DB, name string) error {
	script, err := scripts.Files.ReadFile(name)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(string(script)); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return tx.Commit()
}

func Preflight(db *sql.DB) error {
	schema, err := legacyDataSchema(db)
	if err != nil {
		return err
	}

	checks := []struct {
		name  string
		query string
	}{
		{
			"legacy phone numbers already used by another current user",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.farmers f
			 JOIN users u ON u.phone = f.phone AND u.id <> f.id
			 WHERE BTRIM(f.phone) <> ''
			   AND BTRIM(f.password_hash) <> ''`,
		},
		{
			"duplicate legacy phone numbers",
			`SELECT COUNT(*)
			 FROM (
			     SELECT phone
			     FROM legacy_agro_shield.farmers
			     WHERE BTRIM(phone) <> ''
			       AND BTRIM(password_hash) <> ''
			     GROUP BY phone
			     HAVING COUNT(*) > 1
			 ) duplicates`,
		},
		{
			"legacy email addresses already used by another current user",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.farmers f
			 JOIN users u ON LOWER(BTRIM(u.email)) = LOWER(BTRIM(f.email))
			 WHERE BTRIM(f.email) <> ''
			   AND u.id <> f.id`,
		},
		{
			"wallet transactions with an unsupported type",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.wallet_transactions
			 WHERE type NOT IN (
			     'credit', 'debit', 'deposit', 'deposit_reversal',
			     'withdrawal_reserved', 'withdrawal_refund',
			     'order_payment', 'sale_credit', 'transfer_out',
			     'transfer_in', 'transfer_refund'
			 )`,
		},
		{
			"wallet deposits with a blank reference",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.wallet_deposits
			 WHERE status IN ('pending', 'processing', 'successful', 'failed', 'expired', 'reversed')
			   AND (reference IS NULL OR BTRIM(reference) = '')`,
		},
		{
			"duplicate wallet deposit references",
			`SELECT COUNT(*)
			 FROM (
			     SELECT reference
			     FROM legacy_agro_shield.wallet_deposits
			     WHERE status IN ('pending', 'processing', 'successful', 'failed', 'expired', 'reversed')
			     GROUP BY reference
			     HAVING COUNT(*) > 1
			 ) duplicates`,
		},
		{
			"duplicate wallet provider transaction ids",
			`SELECT COUNT(*)
			 FROM (
			     SELECT provider, provider_transaction_id
			     FROM legacy_agro_shield.wallet_deposits
			     WHERE provider_transaction_id IS NOT NULL
			     GROUP BY provider, provider_transaction_id
			     HAVING COUNT(*) > 1
			 ) duplicates`,
		},
		{
			"withdrawals that cannot satisfy the current validation rules",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.wallet_withdrawals
			 WHERE amount <= 0
			    OR reference IS NULL
			    OR BTRIM(reference) = ''
			    OR bank_name IS NULL
			    OR BTRIM(bank_name) = ''
			    OR bank_code IS NULL
			    OR BTRIM(bank_code) = ''
			    OR account_name IS NULL
			    OR BTRIM(account_name) = ''
			    OR account_number IS NULL
			    OR account_number !~ '^[0-9]{10}$'`,
		},
		{
			"transfers that cannot satisfy the current validation rules",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.wallet_transfers
			 WHERE amount <= 0
			    OR sender_id = recipient_id
			    OR reference IS NULL
			    OR BTRIM(reference) = ''
			    OR recipient_phone IS NULL
			    OR BTRIM(recipient_phone) = ''`,
		},
		{
			"duplicate virtual-account references",
			`SELECT COUNT(*)
			 FROM (
			     SELECT tx_ref
			     FROM legacy_agro_shield.wallet_virtual_accounts
			     GROUP BY tx_ref
			     HAVING COUNT(*) > 1
			 ) duplicates`,
		},
		{
			"active virtual accounts with incomplete details",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.wallet_virtual_accounts
			 WHERE status = 'active'
			   AND (
			       account_number IS NULL
			       OR account_number !~ '^[0-9]{10}$'
			       OR bank_name IS NULL
			       OR BTRIM(bank_name) = ''
			       OR account_name IS NULL
			       OR BTRIM(account_name) = ''
			   )`,
		},
	}

	checks = append(checks, []struct{ name, query string }{
		{"accounts missing a name, phone, or password", `SELECT count(*) FROM legacy_agro_shield.farmers WHERE BTRIM(full_name)='' OR BTRIM(phone)='' OR BTRIM(password_hash)=''`},
		{"orders with missing products or invalid amounts", `SELECT count(*) FROM legacy_agro_shield.orders o LEFT JOIN legacy_agro_shield.crops p ON p.id=o.crop_id WHERE p.id IS NULL OR o.quantity<=0 OR o.total_price<0`},
		{"multiple deliveries for one order", `SELECT count(*) FROM (SELECT order_id FROM legacy_agro_shield.deliveries GROUP BY order_id HAVING count(*)>1) duplicates`},
		{"delivery progress missing its recorded date", `SELECT count(*) FROM legacy_agro_shield.deliveries WHERE (status IN ('picked_up','in_transit','delivered') AND picked_up_at IS NULL) OR (status='delivered' AND delivered_at IS NULL)`},
		{"payments with unsupported or incomplete values", `SELECT count(*) FROM legacy_agro_shield.payments WHERE status NOT IN ('pending','processing','successful','success','paid','failed','cancelled','refunded') OR BTRIM(reference)='' OR amount<0 OR BTRIM(currency)=''`},
		{"duplicate provider payment ids", `SELECT count(*) FROM (SELECT provider,transaction_id FROM legacy_agro_shield.payments WHERE transaction_id IS NOT NULL AND transaction_id<>'' GROUP BY provider,transaction_id HAVING count(*)>1) duplicates`},
	}...)

	var currentUsers bool
	if err := db.QueryRow(`SELECT to_regclass('public.users') IS NOT NULL`).Scan(&currentUsers); err != nil {
		return err
	}
	filtered := checks[:0]
	for _, check := range checks {
		if !currentUsers && strings.Contains(check.query, "JOIN users u") {
			continue
		}
		filtered = append(filtered, check)
	}
	checks = filtered
	for i := range checks {
		checks[i].query = strings.ReplaceAll(
			checks[i].query,
			"legacy_agro_shield.",
			schema+".",
		)
	}

	return runCutoverChecks(db, checks)
}

func legacyDataSchema(db *sql.DB) (string, error) {
	var schema string
	err := db.QueryRow(`
		SELECT CASE
			WHEN to_regclass('public.farmers') IS NOT NULL THEN 'public'
			WHEN to_regclass('legacy_agro_shield.farmers') IS NOT NULL THEN 'legacy_agro_shield'
			ELSE ''
		END
	`).Scan(&schema)
	if err != nil {
		return "", fmt.Errorf("locate legacy data: %w", err)
	}
	if schema == "" {
		return "", fmt.Errorf("legacy Agro-Shield data was not found")
	}
	return schema, nil
}

func runCutoverChecks(db queryer, checks []struct {
	name  string
	query string
}) error {
	var failures []string
	for _, check := range checks {
		var count int
		if err := db.QueryRow(check.query).Scan(&count); err != nil {
			return fmt.Errorf("check %s: %w", check.name, err)
		}
		if count > 0 {
			failures = append(failures, fmt.Sprintf("%s: %d", check.name, count))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("legacy preflight failed: %s", strings.Join(failures, ", "))
	}

	log.Println("Legacy preflight passed")
	return nil
}

func Verify(db queryer) error {
	checks := []struct {
		name  string
		query string
	}{
		{
			"legacy farmer accounts",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.farmers f
			 LEFT JOIN users u ON u.id = f.id
			 WHERE BTRIM(f.phone) <> ''
			   AND BTRIM(f.password_hash) <> ''
			   AND u.id IS NULL`,
		},
		{
			"legacy products",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.crops c
			 JOIN users u ON u.id = c.farmer_id
			 LEFT JOIN products p ON p.id = c.id
			 WHERE p.id IS NULL`,
		},
		{
			"legacy product images",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.crop_images ci
			 JOIN products p ON p.id = ci.crop_id
			 WHERE BTRIM(ci.image_url) <> ''
			   AND NOT EXISTS (
			       SELECT 1 FROM product_images pi
			       WHERE pi.product_id = ci.crop_id
			         AND pi.image_url = ci.image_url
			   )`,
		},
		{
			"legacy orders",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.orders o
			 JOIN users u ON u.id = o.buyer_id
			 JOIN products p ON p.id = o.crop_id
			 LEFT JOIN orders current_order ON current_order.id = o.id
			 WHERE current_order.id IS NULL`,
		},
		{
			"legacy negotiations",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.negotiations n
			 JOIN products p ON p.id = n.crop_id
			 JOIN users u ON u.id = n.buyer_id
			 LEFT JOIN negotiations current_negotiation ON current_negotiation.id = n.id
			 WHERE current_negotiation.id IS NULL`,
		},
		{
			"legacy negotiation messages",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.negotiation_messages m
			 JOIN negotiations n ON n.id = m.negotiation_id
			 JOIN users u ON u.id = m.sender_id
			 WHERE NOT EXISTS (
			     SELECT 1 FROM negotiation_messages current_message
			     WHERE current_message.negotiation_id = m.negotiation_id
			       AND current_message.sender_id = m.sender_id
			       AND current_message.created_at = m.created_at
			 )`,
		},
		{
			"legacy cart items",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.cart_items ci
			 JOIN users u ON u.id = ci.buyer_id
			 JOIN products p ON p.id = ci.crop_id
			 WHERE NOT EXISTS (
			     SELECT 1 FROM cart_items current_item
			     WHERE current_item.buyer_id = ci.buyer_id
			       AND current_item.product_id = ci.crop_id
			       AND current_item.created_at = ci.created_at
			 )`,
		},
		{
			"legacy conversations",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.conversations c
			 LEFT JOIN conversations current_conversation ON current_conversation.id = c.id
			 WHERE current_conversation.id IS NULL`,
		},
		{
			"legacy conversation members",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.conversation_members cm
			 JOIN conversations c ON c.id = cm.conversation_id
			 JOIN users u ON u.id = cm.user_id
			 WHERE NOT EXISTS (
			     SELECT 1 FROM conversation_members current_member
			     WHERE current_member.conversation_id = cm.conversation_id
			       AND current_member.user_id = cm.user_id
			 )`,
		},
		{
			"legacy chat messages",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.chat_messages cm
			 JOIN conversations c ON c.id = cm.conversation_id
			 JOIN users u ON u.id = cm.sender_id
			 WHERE NOT EXISTS (
			     SELECT 1 FROM chat_messages current_message
			     WHERE current_message.conversation_id = cm.conversation_id
			       AND current_message.sender_id = cm.sender_id
			       AND current_message.created_at = cm.created_at
			 )`,
		},
		{
			"legacy notifications",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.notifications n
			 JOIN users u ON u.id = n.user_id
			 WHERE NOT EXISTS (
			     SELECT 1 FROM notifications current_notification
			     WHERE current_notification.user_id = n.user_id
			       AND current_notification.title = COALESCE(NULLIF(BTRIM(n.title), ''), 'Agro-Shield update')
			       AND current_notification.created_at = n.created_at
			 )`,
		},
		{
			"legacy diagnoses",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.diagnoses d
			 JOIN users u ON u.id = d.farmer_id
			 LEFT JOIN diagnoses current_diagnosis ON current_diagnosis.id = d.id
			 WHERE current_diagnosis.id IS NULL`,
		},
		{
			"legacy market events",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.market_events me
			 WHERE NOT EXISTS (
			     SELECT 1 FROM market_events current_event
			     WHERE current_event.event_type = COALESCE(NULLIF(BTRIM(me.event_type), ''), 'legacy_event')
			       AND current_event.created_at = me.created_at
			       AND current_event.session_id IS NOT DISTINCT FROM me.session_id
			 )`,
		},
		{
			"legacy wallets",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.wallets lw
			 JOIN users u ON u.id = lw.farmer_id
			 WHERE NOT EXISTS (
			     SELECT 1 FROM wallets w
			     WHERE w.owner_type = 'user'
			       AND w.owner_id = lw.farmer_id
			       AND w.currency = COALESCE(NULLIF(BTRIM(lw.currency), ''), 'NGN')
			 )`,
		},
		{
			"legacy wallet transactions",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.wallet_transactions lt
			 WHERE NOT EXISTS (
			     SELECT 1 FROM wallet_transactions current_transaction
			     WHERE current_transaction.reference = COALESCE(NULLIF(BTRIM(lt.reference), ''), 'legacy-wallet-txn-' || lt.id)
			 )`,
		},
		{
			"legacy wallet deposits",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.wallet_deposits ld
			 WHERE ld.status IN ('pending', 'processing', 'successful', 'failed', 'expired', 'reversed')
			   AND NOT EXISTS (
			       SELECT 1 FROM wallet_deposits current_deposit
			       WHERE current_deposit.reference = ld.reference
			   )`,
		},
		{
			"legacy wallet withdrawals",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.wallet_withdrawals lww
			 WHERE NOT EXISTS (
			     SELECT 1 FROM wallet_withdrawals current_withdrawal
			     WHERE current_withdrawal.reference = lww.reference
			 )`,
		},
		{
			"legacy wallet transfers",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.wallet_transfers lwt
			 WHERE NOT EXISTS (
			     SELECT 1 FROM wallet_transfers current_transfer
			     WHERE current_transfer.reference = lwt.reference
			 )`,
		},
		{
			"legacy bank account lookups",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.bank_account_lookups bal
			 WHERE bal.account_number ~ '^[0-9]{10}$'
			   AND NOT EXISTS (
			       SELECT 1 FROM bank_account_lookups current_lookup
			       WHERE current_lookup.account_number = bal.account_number
			         AND current_lookup.bank_code = bal.bank_code
			   )`,
		},
		{
			"legacy platform revenue",
			`SELECT COUNT(*)
			 FROM legacy_agro_shield.platform_revenue pr
			 JOIN orders o ON o.id = pr.order_id
			 WHERE pr.gross_amount > 0
			   AND pr.amount > 0
			   AND pr.amount <= pr.gross_amount
			   AND NOT EXISTS (
			       SELECT 1 FROM platform_revenue current_revenue
			       WHERE current_revenue.reference = pr.reference
			   )`,
		},
	}

	checks = append(checks, []struct{ name, query string }{
		{"wallet balance or number changed during transfer", `SELECT count(*) FROM legacy_agro_shield.wallets old JOIN legacy_agro_shield.farmers f ON f.id=old.farmer_id LEFT JOIN wallets w ON w.owner_type='user' AND w.owner_id=old.farmer_id AND w.currency=old.currency WHERE w.id IS NULL OR w.balance<>old.balance OR (f.wallet_number IS NOT NULL AND f.wallet_number<>'' AND w.wallet_number<>f.wallet_number)`},
		{"wallet ledger amount or owner changed", `SELECT count(*) FROM legacy_agro_shield.wallet_transactions old JOIN legacy_agro_shield.wallets ow ON ow.id=old.wallet_id LEFT JOIN wallet_transactions t ON t.reference=old.reference LEFT JOIN wallets w ON w.id=t.wallet_id WHERE t.id IS NULL OR t.amount<>old.amount OR w.owner_type<>'user' OR w.owner_id<>ow.farmer_id`},
		{"payments missing or changed", `SELECT count(*) FROM legacy_agro_shield.payments old LEFT JOIN transactions t ON t.payment_reference=old.reference WHERE t.id IS NULL OR t.order_id<>old.order_id OR t.amount<>old.amount OR t.currency<>old.currency`},
		{"deliveries missing", `SELECT count(*) FROM legacy_agro_shield.deliveries old LEFT JOIN deliveries d ON d.id=old.id AND d.order_id=old.order_id WHERE d.id IS NULL`},
		{"chat message text missing or changed", `SELECT count(*) FROM legacy_agro_shield.chat_messages old LEFT JOIN chat_messages m ON m.id=old.id WHERE m.id IS NULL OR (BTRIM(old.message)<>'' AND m.message<>BTRIM(old.message))`},
		{"negotiation prices changed", `SELECT count(*) FROM legacy_agro_shield.negotiations old LEFT JOIN negotiations n ON n.id=old.id WHERE n.id IS NULL OR n.initial_offer_price<>old.initial_offer_price OR n.current_offer_price<>old.current_offer_price`},
		{"virtual accounts missing", `SELECT count(*) FROM legacy_agro_shield.wallet_virtual_accounts old LEFT JOIN wallet_virtual_accounts a ON a.tx_ref=old.tx_ref WHERE a.id IS NULL OR a.user_id<>old.farmer_id OR a.account_number<>old.account_number`},
	}...)

	return runCutoverChecks(db, checks)
}

func legacySchemaPresent(db *sql.DB) (bool, error) {
	var present bool
	err := db.QueryRow(`
		SELECT to_regclass('public.farmers') IS NOT NULL
		    OR to_regclass('public.crops') IS NOT NULL
		    OR to_regclass('legacy_agro_shield.farmers') IS NOT NULL
		    OR to_regclass('legacy_agro_shield.crops') IS NOT NULL
	`).Scan(&present)
	return present, err
}
