package repository

import (
	"backend/internal/models"
	"database/sql"
	"encoding/json"
	"fmt"
)

func escrowLiability(e *models.Escrow) string {
	if e.State == "disputed" {
		return fmt.Sprintf("disputed-escrow:%d", e.OrderID)
	}
	return fmt.Sprintf("escrow:%d", e.OrderID)
}
func moveDisputedEscrow(tx *sql.Tx, e *models.Escrow, freeze bool, actorID int, reason string) error {
	held := fmt.Sprintf("escrow:%d", e.OrderID)
	disputed := fmt.Sprintf("disputed-escrow:%d", e.OrderID)
	if err := ensureAccount(tx, disputed, "liability"); err != nil {
		return err
	}
	source, dest, action := held, disputed, "freeze"
	if !freeze {
		source, dest, action = disputed, held, "unfreeze"
	}
	var sequence int64
	if err := tx.QueryRow(`SELECT count(*) FROM escrow_events WHERE order_id=$1`, e.OrderID).Scan(&sequence); err != nil {
		return err
	}
	evidence, _ := json.Marshal(map[string]any{"order_id": e.OrderID, "reason": reason})
	return postJournal(tx, fmt.Sprintf("escrow:%d:%s:%d", e.OrderID, action, sequence), "escrow_"+action, e.Currency, actorID, string(evidence), debit(source, e.GrossKobo), credit(dest, e.GrossKobo))
}

// Support ticket creation shares the order lock with delivery confirmation and
// release. A support administrator can freeze a disputed purchase, never pay it.
func freezeTicketEscrow(tx *sql.Tx, orderID, actorID int, reason string) error {
	var locked int
	if err := tx.QueryRow(`SELECT id FROM orders WHERE id=$1 FOR UPDATE`, orderID).Scan(&locked); err != nil {
		return err
	}
	e, err := scanEscrow(tx.QueryRow(`SELECT `+escrowSelect+escrowJoins+`WHERE e.order_id=$1 FOR UPDATE OF e`, orderID))
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if e.State != "held" {
		return nil
	}
	if err = moveDisputedEscrow(tx, e, true, actorID, reason); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE order_escrows SET state='disputed',dispute_reason=$2,updated_at=CURRENT_TIMESTAMP WHERE order_id=$1`, orderID, reason); err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO escrow_events(order_id,actor_id,action,reason) VALUES($1,$2,'support_dispute',$3)`, orderID, actorID, reason)
	return err
}
