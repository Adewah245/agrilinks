package repository

import (
	"backend/internal/models"
	"database/sql"
)

type NegotiationMessageRepository struct {
	db *sql.DB
}

func NewNegotiationMessageRepository(db *sql.DB) *NegotiationMessageRepository {
	return &NegotiationMessageRepository{db: db}
}

func (r *NegotiationMessageRepository) Create(m *models.NegotiationMessage) error {
	query := `
	INSERT INTO negotiation_messages (negotiation_id, sender_id, offer_price, message)
    VALUES ($1, $2, $3, $4)	`
	_, err := r.db.Exec(query, m.NegotiationID, m.SenderID, m.OfferPrice, m.Message)
	return err
}

func (r *NegotiationMessageRepository) ListByNegotiation(negotiationID int) ([]models.NegotiationMessage, error) {
	query := `
	SELECT id, negotiation_id, sender_id, offer_price, message, created_at
	FROM negotiation_messages
	WHERE negotiation_id = $1
	ORDER BY created_at ASC
	`
	rows, err := r.db.Query(query, negotiationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.NegotiationMessage
	for rows.Next() {
		var m models.NegotiationMessage
		if err := rows.Scan(&m.ID, &m.NegotiationID, &m.SenderID, &m.OfferPrice, &m.Message, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// LastOffer returns the most recent offer in the thread — this is the
// price that gets accepted when either party clicks "Accept."
func (r *NegotiationMessageRepository) LastOffer(negotiationID int) (*models.NegotiationMessage, error) {
	query := `
	SELECT id, negotiation_id, sender_id, offer_price, message, created_at
	FROM negotiation_messages
	WHERE negotiation_id = $1
	ORDER BY created_at DESC
	LIMIT 1
	`
	var m models.NegotiationMessage
	err := r.db.QueryRow(query, negotiationID).Scan(&m.ID, &m.NegotiationID, &m.SenderID, &m.OfferPrice, &m.Message, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
