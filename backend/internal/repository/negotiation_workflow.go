package repository

import (
	"database/sql"
	"fmt"
	"math"

	"backend/internal/models"
)

// StartWithOffer saves the negotiation and its first round together.
func (r *NegotiationRepository) StartWithOffer(n *models.Negotiation, offer *models.NegotiationMessage) error {
	if offer.OfferPrice == nil || *offer.OfferPrice <= 0 || math.IsNaN(*offer.OfferPrice) || math.IsInf(*offer.OfferPrice, 0) {
		return fmt.Errorf("enter a valid offer price")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = tx.QueryRow(`INSERT INTO negotiations(product_id,buyer_id,quantity,initial_offer_price,current_offer_price,round_count,max_rounds,status,expires_at)
		VALUES($1,$2,$3,$4,$4,1,$5,'open',$6) RETURNING id,created_at,updated_at`, n.ProductID, n.BuyerID, n.Quantity, *offer.OfferPrice, n.MaxRounds, n.ExpiresAt).
		Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return err
	}
	n.RoundCount = 1
	n.InitialOfferPrice = *offer.OfferPrice
	n.CurrentOfferPrice = *offer.OfferPrice
	offer.NegotiationID = n.ID
	if err = insertNegotiationOffer(tx, offer); err != nil {
		return err
	}
	return tx.Commit()
}

func insertNegotiationOffer(tx *sql.Tx, offer *models.NegotiationMessage) error {
	return tx.QueryRow(`INSERT INTO negotiation_messages(negotiation_id,sender_id,message_type,offer_price,message,offer_status)
		VALUES($1,$2,'offer',$3,$4,'pending') RETURNING id,created_at`, offer.NegotiationID, offer.SenderID, offer.OfferPrice, offer.Message).Scan(&offer.ID, &offer.CreatedAt)
}

// AddOffer changes the current price and round count with the saved message.
func (r *NegotiationRepository) AddOffer(offer *models.NegotiationMessage) error {
	if offer.OfferPrice == nil || *offer.OfferPrice <= 0 || math.IsNaN(*offer.OfferPrice) || math.IsInf(*offer.OfferPrice, 0) {
		return fmt.Errorf("enter a valid offer price")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE negotiations SET round_count=round_count+1,current_offer_price=$2,updated_at=CURRENT_TIMESTAMP
		WHERE id=$1 AND status='open' AND expires_at>CURRENT_TIMESTAMP AND round_count<max_rounds`, offer.NegotiationID, *offer.OfferPrice)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("this negotiation has closed, expired, or reached its offer limit")
	}
	if err = insertNegotiationOffer(tx, offer); err != nil {
		return err
	}
	return tx.Commit()
}

// AcceptOffer cannot leave an accepted offer without the matching cart item.
func (r *NegotiationRepository) AcceptOffer(negotiationID, offerID, accepterID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var buyerID, productID int
	var quantity float64
	var status string
	var expired bool
	if err = tx.QueryRow(`SELECT buyer_id FROM negotiations WHERE id=$1`, negotiationID).Scan(&buyerID); err != nil {
		return err
	}
	var lockedBuyer int
	if err = tx.QueryRow(`SELECT id FROM users WHERE id=$1 FOR UPDATE`, buyerID).Scan(&lockedBuyer); err != nil {
		return err
	}
	err = tx.QueryRow(`SELECT product_id,quantity,status,expires_at<=CURRENT_TIMESTAMP FROM negotiations WHERE id=$1 FOR UPDATE`, negotiationID).Scan(&productID, &quantity, &status, &expired)
	if err != nil {
		return err
	}
	if status != "open" || expired {
		return fmt.Errorf("this negotiation is no longer open")
	}
	var ownerType, productStatus string
	var ownerID int
	var available float64
	err = tx.QueryRow(`SELECT owner_type,owner_id,status,quantity FROM products WHERE id=$1 FOR UPDATE`, productID).Scan(&ownerType, &ownerID, &productStatus, &available)
	if err != nil {
		return err
	}
	if productStatus != "active" || quantity > available {
		return fmt.Errorf("the agreed amount is no longer available")
	}
	allowed := accepterID == buyerID || (ownerType == "user" && ownerID == accepterID)
	if !allowed && ownerType == "organization" {
		err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM organization_members WHERE organization_id=$1 AND user_id=$2)`, ownerID, accepterID).Scan(&allowed)
		if err != nil {
			return err
		}
	}
	if !allowed {
		return fmt.Errorf("you are not part of this negotiation")
	}
	var senderID int
	var price string
	err = tx.QueryRow(`SELECT sender_id,offer_price::text FROM negotiation_messages
		WHERE id=$1 AND negotiation_id=$2 AND message_type='offer' AND offer_status='pending' FOR UPDATE`, offerID, negotiationID).Scan(&senderID, &price)
	if err != nil {
		return err
	}
	if (senderID == buyerID) == (accepterID == buyerID) {
		return fmt.Errorf("you cannot accept your own side's offer")
	}
	_, err = tx.Exec(`INSERT INTO cart_items(buyer_id,product_id,negotiation_id,quantity,price_per_unit,total_price)
		VALUES($1,$2,$3,$4,$5::numeric,round($4::numeric*$5::numeric,2))`, buyerID, productID, negotiationID, quantity, price)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE negotiation_messages SET offer_status=CASE WHEN id=$2 THEN 'accepted' ELSE 'rejected' END
		WHERE negotiation_id=$1 AND message_type='offer' AND offer_status='pending'`, negotiationID, offerID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE negotiations SET status='accepted',current_offer_price=$2::numeric,accepted_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, negotiationID, price)
	if err != nil {
		return err
	}
	return tx.Commit()
}
