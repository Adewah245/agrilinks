package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"backend/internal/models"
	"backend/internal/money"
)

type EscrowRepository struct{ db *sql.DB }

func NewEscrowRepository(db *sql.DB) *EscrowRepository { return &EscrowRepository{db: db} }

func holdOrderSale(tx *sql.Tx, orderID int, sellerType string, sellerID int, amount, currency string, bps int64, costs ...DepositCosts) error {
	gross, err := money.Positive(amount)
	if err != nil {
		return err
	}
	commission, err := money.BasisPoints(gross, bps)
	if err != nil {
		return err
	}
	var buyerID int
	var reference, provider string
	if err = tx.QueryRow(`SELECT buyer_id,payment_reference,payment_provider FROM transactions WHERE order_id=$1 AND status='successful' AND NOT review_required ORDER BY id LIMIT 1`, orderID).Scan(&buyerID, &reference, &provider); err != nil {
		return err
	}
	result, err := tx.Exec(`INSERT INTO order_escrows(order_id,buyer_id,seller_type,seller_id,payment_reference,currency,gross_kobo,commission_kobo,commission_bps) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(order_id) DO NOTHING`, orderID, buyerID, sellerType, sellerID, reference, currency, gross, commission, bps)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	escrowAccount := fmt.Sprintf("escrow:%d", orderID)
	if err = ensureAccount(tx, escrowAccount, "liability"); err != nil {
		return err
	}
	source := "provider_clearing"
	customerProvider := int64(0)
	var actual *int64
	if provider == "wallet" {
		var walletID int64
		if err = tx.QueryRow(`SELECT id FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency=$2`, buyerID, currency).Scan(&walletID); err != nil {
			return err
		}
		source = walletAccount(walletID)
		if err = ensureAccount(tx, source, "liability"); err != nil {
			return err
		}
	} else if len(costs) > 0 {
		actual = costs[0].ProviderKobo
		if costs[0].ChargedKobo != nil && *costs[0].ChargedKobo > gross {
			customerProvider = *costs[0].ChargedKobo - gross
		}
	}
	received, err := money.Add(gross, customerProvider)
	if err != nil {
		return err
	}
	if err = postJournal(tx, reference+":escrow", "purchase_escrow_hold", currency, buyerID, "{}", debit(source, received), credit(escrowAccount, gross), credit("provider_cost_recovery", customerProvider)); err != nil {
		return err
	}
	if actual != nil {
		if err = postJournal(tx, reference+":provider-cost", "provider_cost", currency, 0, `{"source":"verified_app_fee"}`, debit("provider_expense", *actual), credit("provider_clearing", *actual)); err != nil {
			return err
		}
	}

	if _, err = tx.Exec(`INSERT INTO escrow_events(order_id,actor_id,action,reason) VALUES($1,$2,'funded','Buyer payment verified; money held pending delivery, buyer confirmation and administrator review')`, orderID, buyerID); err != nil {
		return err
	}
	// Existing organization ownership is resolved by the same user mapping used
	// for organization products; notifications never credit a personal wallet.
	if sellerType == "user" {
		_, err = tx.Exec(`INSERT INTO notifications(user_id,notification_type,title,message,reference_type,reference_id) VALUES($1,'order','Order paid — prepare delivery',$2,'order',$3)`, sellerID, fmt.Sprintf("Order #%d is paid. %s %s is protected in escrow. Deliver the order and record delivery evidence.", orderID, currency, money.Format(gross)), orderID)
	}
	if sellerType == "organization" {
		_, err = tx.Exec(`INSERT INTO notifications(user_id,notification_type,title,message,reference_type,reference_id) SELECT user_id,'order','Order paid — prepare delivery',$2,'order',$3 FROM organization_members WHERE organization_id=$1`, sellerID, fmt.Sprintf("Order #%d is paid and held in purchase protection.", orderID), orderID)
	}
	return err
}

const escrowSelect = `e.order_id,e.buyer_id,e.seller_type,e.seller_id,e.payment_reference,e.currency,e.gross_kobo,e.commission_kobo,e.commission_bps,e.state,
 COALESCE((SELECT delivered_at FROM deliveries WHERE order_id=e.order_id AND status='delivered'),e.seller_delivered_at),e.buyer_confirmed_at,e.delivery_evidence,e.dispute_reason,e.decided_by,e.decided_at,e.decision_reason,e.created_at,
 b.full_name,COALESCE(s.full_name,o.name,''),COALESCE((SELECT string_agg(p.name,', ' ORDER BY i.id) FROM order_items i JOIN products p ON p.id=i.product_id WHERE i.order_id=e.order_id),'')`
const escrowJoins = ` FROM order_escrows e JOIN users b ON b.id=e.buyer_id LEFT JOIN users s ON e.seller_type='user' AND s.id=e.seller_id LEFT JOIN organizations o ON e.seller_type='organization' AND o.id=e.seller_id `

func scanEscrow(row interface{ Scan(...any) error }) (*models.Escrow, error) {
	e := &models.Escrow{}
	err := row.Scan(&e.OrderID, &e.BuyerID, &e.SellerType, &e.SellerID, &e.PaymentReference, &e.Currency, &e.GrossKobo, &e.CommissionKobo, &e.CommissionBPS, &e.State, &e.DeliveredAt, &e.BuyerConfirmedAt, &e.DeliveryEvidence, &e.DisputeReason, &e.DecidedBy, &e.DecidedAt, &e.DecisionReason, &e.CreatedAt, &e.BuyerName, &e.SellerName, &e.Products)
	return e, err
}
func (r *EscrowRepository) List(ctx context.Context, userID int) ([]models.Escrow, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+escrowSelect+escrowJoins+`WHERE e.buyer_id=$1 OR (e.seller_type='user' AND e.seller_id=$1) OR (e.seller_type='organization' AND EXISTS(SELECT 1 FROM organization_members m WHERE m.organization_id=e.seller_id AND m.user_id=$1 AND m.member_role IN ('owner','admin','manager'))) ORDER BY e.created_at DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Escrow{}
	for rows.Next() {
		e, err := scanEscrow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *e)
	}
	return items, rows.Err()
}
func (r *EscrowRepository) Page(ctx context.Context, user *models.User, orderID int) (*models.EscrowPage, error) {
	e, err := scanEscrow(r.db.QueryRowContext(ctx, `SELECT `+escrowSelect+escrowJoins+`WHERE e.order_id=$1`, orderID))
	if err != nil {
		return nil, err
	}
	p := &models.EscrowPage{User: user, Escrow: e, IsBuyer: user.ID == e.BuyerID, IsSeller: e.SellerType == "user" && e.SellerID == user.ID, CanReview: user.CanAdmin("finance")}
	if e.SellerType == "organization" {
		if err = r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM organization_members WHERE organization_id=$1 AND user_id=$2 AND member_role IN ('owner','admin','manager'))`, e.SellerID, user.ID).Scan(&p.IsSeller); err != nil {
			return nil, err
		}
	}
	if !p.IsBuyer && !p.IsSeller && !p.CanReview {
		return nil, ErrAdminForbidden
	}
	rows, err := r.db.QueryContext(ctx, `SELECT p.name,i.quantity::text,i.unit_price::text,i.total_price::text FROM order_items i JOIN products p ON p.id=i.product_id WHERE i.order_id=$1 ORDER BY i.id`, orderID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item models.EscrowItem
		if err = rows.Scan(&item.Product, &item.Quantity, &item.UnitPrice, &item.Total); err != nil {
			rows.Close()
			return nil, err
		}
		p.Items = append(p.Items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = r.db.QueryContext(ctx, `SELECT e.actor_id,COALESCE(u.full_name,'System'),e.action,e.reason,e.created_at FROM escrow_events e LEFT JOIN users u ON u.id=e.actor_id WHERE e.order_id=$1 ORDER BY e.id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var event models.EscrowEvent
		if err = rows.Scan(&event.ActorID, &event.ActorName, &event.Action, &event.Reason, &event.CreatedAt); err != nil {
			return nil, err
		}
		p.Events = append(p.Events, event)
	}
	return p, rows.Err()
}

func (r *EscrowRepository) Act(ctx context.Context, user *models.User, orderID int, action, reason string) error {
	if user == nil {
		return ErrAdminForbidden
	}
	reason = strings.TrimSpace(reason)
	if len(reason) < 5 || len(reason) > 2000 {
		return adminInvalid("record a reason or delivery evidence between 5 and 2,000 characters")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return err
	}
	var orderStatus, orderAmount, orderCurrency string
	var orderBuyer int
	if err = tx.QueryRowContext(ctx, `SELECT status,total_amount::text,currency,buyer_id FROM orders WHERE id=$1 FOR UPDATE`, orderID).Scan(&orderStatus, &orderAmount, &orderCurrency, &orderBuyer); err != nil {
		return err
	}
	e, err := scanEscrow(tx.QueryRowContext(ctx, `SELECT `+escrowSelect+escrowJoins+`WHERE e.order_id=$1 FOR UPDATE OF e`, orderID))
	if err != nil {
		return err
	}
	buyer := user.ID == e.BuyerID
	seller := e.SellerType == "user" && e.SellerID == user.ID
	if e.SellerType == "organization" {
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM organization_members WHERE organization_id=$1 AND user_id=$2 AND member_role IN ('owner','admin','manager'))`, e.SellerID, user.ID).Scan(&seller); err != nil {
			return err
		}
	}
	adminAction := action == "release" || action == "refund" || action == "resolve_dispute"
	if adminAction {
		user, err = financeActor(ctx, tx, user.ID, "finance")
		if err != nil {
			return err
		}
	}
	if adminAction && (!user.CanAdmin("finance") || buyer || seller) {
		return ErrAdminForbidden
	}
	if e.State == "released" || e.State == "refunded" || e.State == "charged_back" {
		if (action == "release" && e.State == "released") || (action == "refund" && e.State == "refunded") {
			return nil
		}
		return adminInvalid("this purchase has already been settled")
	}
	switch action {
	case "seller_delivered":
		if !seller {
			return ErrAdminForbidden
		}
		if e.State == "disputed" {
			return adminInvalid("resolve the dispute before changing delivery")
		}
		var activeDelivery bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM deliveries WHERE order_id=$1 AND (status<>'delivered' OR delivered_at IS NULL))`, orderID).Scan(&activeDelivery); err != nil {
			return err
		}
		if activeDelivery {
			return adminInvalid("the delivery partner must complete the tracked delivery first")
		}
		if _, err = tx.ExecContext(ctx, `UPDATE order_escrows SET seller_delivered_at=COALESCE(seller_delivered_at,CURRENT_TIMESTAMP),delivery_evidence=$2,updated_at=CURRENT_TIMESTAMP WHERE order_id=$1`, orderID, reason); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE orders SET status='delivered',updated_at=CURRENT_TIMESTAMP WHERE id=$1 AND status<>'cancelled'`, orderID); err != nil {
			return err
		}
	case "confirm_delivery":
		if !buyer {
			return ErrAdminForbidden
		}
		if e.State != "held" || !e.DeliveredAt.Valid || orderStatus != "delivered" {
			return adminInvalid("delivery must be recorded and all disputes resolved before you confirm receipt")
		}
		if e.BuyerConfirmedAt.Valid {
			return nil
		}
		_, err = tx.ExecContext(ctx, `UPDATE order_escrows SET buyer_confirmed_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE order_id=$1`, orderID)
	case "dispute":
		if !buyer && !seller {
			return ErrAdminForbidden
		}
		if e.State == "disputed" {
			return nil
		}
		if err = moveDisputedEscrow(tx, e, true, user.ID, reason); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE order_escrows SET state='disputed',dispute_reason=$2,updated_at=CURRENT_TIMESTAMP WHERE order_id=$1`, orderID, reason)
	case "resolve_dispute":
		if e.State != "disputed" {
			return adminInvalid("this order does not have an unresolved escrow dispute")
		}
		if err = moveDisputedEscrow(tx, e, false, user.ID, reason); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE order_escrows SET state='held',buyer_confirmed_at=NULL,dispute_reason='',updated_at=CURRENT_TIMESTAMP WHERE order_id=$1`, orderID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `WITH changed AS (UPDATE admin_support_tickets SET status='resolved',resolution=$2,updated_at=CURRENT_TIMESTAMP WHERE order_id=$1 AND kind IN ('dispute','refund') AND status NOT IN ('resolved','closed') RETURNING id) INSERT INTO admin_ticket_history(ticket_id,actor_id,status,note) SELECT id,$3,'resolved',$2 FROM changed`, orderID, reason, user.ID)
	case "release":
		if e.State != "held" || !e.DeliveredAt.Valid || !e.BuyerConfirmedAt.Valid || orderStatus != "delivered" {
			return adminInvalid("release requires recorded delivery, buyer confirmation and no unresolved dispute")
		}
		var openDispute bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM admin_support_tickets WHERE order_id=$1 AND kind IN ('dispute','refund') AND status NOT IN ('resolved','closed'))`, orderID).Scan(&openDispute); err != nil {
			return err
		}
		if openDispute {
			return adminInvalid("an open dispute or refund request blocks release")
		}
		total, err := money.Positive(orderAmount)
		if err != nil {
			return err
		}
		var itemsTotal string
		var paymentOK bool
		if err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(total_price),0)::text FROM order_items WHERE order_id=$1`, orderID).Scan(&itemsTotal); err != nil {
			return err
		}
		itemAmount, err := money.Positive(itemsTotal)
		if err != nil {
			return err
		}
		if total != e.GrossKobo || itemAmount != e.GrossKobo || orderBuyer != e.BuyerID || orderCurrency != e.Currency {
			return adminInvalid("the saved order agreement no longer matches the protected payment")
		}
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM transactions WHERE payment_reference=$1 AND order_id=$2 AND buyer_id=$3 AND seller_type=$4 AND seller_id=$5 AND amount=$6::numeric/100 AND currency=$7 AND status='successful' AND NOT review_required)`, e.PaymentReference, orderID, e.BuyerID, e.SellerType, e.SellerID, e.GrossKobo, e.Currency).Scan(&paymentOK); err != nil {
			return err
		}
		if !paymentOK {
			return adminInvalid("the payment is not verified for this order and seller")
		}
		if err = creditOrderSale(tx, orderID, e.SellerType, e.SellerID, money.Format(e.GrossKobo), e.Currency, e.CommissionBPS, user.ID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE order_escrows SET state='released',decided_by=$2,decided_at=CURRENT_TIMESTAMP,decision_reason=$3,updated_at=CURRENT_TIMESTAMP WHERE order_id=$1`, orderID, user.ID, reason)
	case "refund":
		var walletID int64
		if err = tx.QueryRowContext(ctx, `INSERT INTO wallets(owner_type,owner_id,currency) VALUES('user',$1,$2) ON CONFLICT(owner_type,owner_id,currency) DO UPDATE SET owner_id=EXCLUDED.owner_id RETURNING id`, e.BuyerID, e.Currency).Scan(&walletID); err != nil {
			return err
		}
		if err = ensureAccount(tx, walletAccount(walletID), "liability"); err != nil {
			return err
		}
		if err = postJournal(tx, e.PaymentReference+":escrow-refund", "purchase_refund", e.Currency, user.ID, "{}", debit(escrowLiability(e), e.GrossKobo), credit("marketplace_refunds_payable", e.GrossKobo)); err != nil {
			return err
		}
		if err = postJournal(tx, e.PaymentReference+":refund-credit", "marketplace_refund_credit", e.Currency, user.ID, "{}", debit("marketplace_refunds_payable", e.GrossKobo), credit(walletAccount(walletID), e.GrossKobo)); err != nil {
			return err
		}
		if err = creditWalletKobo(tx, walletID, e.GrossKobo, e.PaymentReference+":refund"); err != nil {
			return err
		}
		if err = walletEntry(tx, walletID, "escrow_refund", e.GrossKobo, e.PaymentReference+":refund"); err != nil {
			return err
		}

		if _, err = tx.ExecContext(ctx, `UPDATE orders SET status='cancelled',updated_at=CURRENT_TIMESTAMP WHERE id=$1`, orderID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE order_escrows SET state='refunded',decided_by=$2,decided_at=CURRENT_TIMESTAMP,decision_reason=$3,updated_at=CURRENT_TIMESTAMP WHERE order_id=$1`, orderID, user.ID, reason)
	default:
		return adminInvalid("choose a purchase protection action")
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO escrow_events(order_id,actor_id,action,reason) VALUES($1,$2,$3,$4)`, orderID, user.ID, action, reason); err != nil {
		return err
	}
	if adminAction {
		before, _ := json.Marshal(map[string]any{"state": e.State})
		after, _ := json.Marshal(map[string]any{"action": action, "payment_reference": e.PaymentReference})
		if _, err = tx.ExecContext(ctx, `INSERT INTO admin_audit_logs(actor_id,actor_name,actor_role,action,target,before_value,after_value,succeeded,detail) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,true,$8)`, user.ID, user.FullName, user.AdminRole(), "escrow_"+action, fmt.Sprintf("order:%d", orderID), string(before), string(after), reason); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO notifications(user_id,notification_type,title,message,reference_type,reference_id) VALUES($1,'order','Purchase protection updated',$2,'order',$3)`, e.BuyerID, fmt.Sprintf("Order #%d: %s. %s", orderID, strings.ReplaceAll(action, "_", " "), reason), orderID); err != nil {
		return err
	}
	return tx.Commit()
}
