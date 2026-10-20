package repository

import (
	"backend/internal/models"
	"database/sql"
	"errors"
	"strings"
)

// ============================================================
// NEGOTIATION MESSAGE REPOSITORY
//
// Responsibility:
// - Communicates directly with negotiation_messages.
// - Stores normal negotiation messages.
// - Stores price offers.
// - Retrieves negotiation history.
// - Retrieves individual messages/offers.
// - Updates individual offer status.
//
// negotiation_messages supports two message types:
//
//   message → normal conversation message
//   offer   → price offer / counter-offer
//
// Offer status:
//
//   none
//   pending
//   accepted
//   rejected
//
// The database CHECK constraints remain the final authority.
// The repository validates common rules before reaching PostgreSQL.
// ============================================================

type NegotiationMessageRepository struct {
	db *sql.DB
}

// ============================================================
// CONSTRUCTOR
// ============================================================

func NewNegotiationMessageRepository(
	db *sql.DB,
) *NegotiationMessageRepository {
	return &NegotiationMessageRepository{
		db: db,
	}
}

// ============================================================
// CREATE
// ============================================================

// Create stores a normal negotiation message or price offer.
//
// Normal message:
//
//	message_type = message
//	message     = required
//	offer_price = NULL
//	offer_status = none
//
// Offer:
//
//	message_type = offer
//	message      = optional
//	offer_price  = required
//	offer_status = pending
func (r *NegotiationMessageRepository) Create(
	m *models.NegotiationMessage,
) error {

	if r == nil || r.db == nil {
		return errors.New(
			"negotiation message repository database is not configured",
		)
	}

	if m == nil {
		return errors.New(
			"negotiation message cannot be nil",
		)
	}

	if m.NegotiationID <= 0 {
		return errors.New(
			"invalid negotiation ID",
		)
	}

	if m.SenderID <= 0 {
		return errors.New(
			"invalid sender ID",
		)
	}

	m.MessageType = strings.ToLower(
		strings.TrimSpace(m.MessageType),
	)

	switch m.MessageType {
	case "message":
		return r.createNormalMessage(m)

	case "offer":
		return r.createOffer(m)

	default:
		return errors.New(
			"invalid negotiation message type",
		)
	}
}

// ============================================================
// CREATE NORMAL MESSAGE
// ============================================================

func (r *NegotiationMessageRepository) createNormalMessage(
	m *models.NegotiationMessage,
) error {

	if m.Message == nil ||
		strings.TrimSpace(*m.Message) == "" {
		return errors.New(
			"message cannot be empty",
		)
	}

	message := strings.TrimSpace(*m.Message)

	if len(message) > 5000 {
		return errors.New(
			"message is too long",
		)
	}

	m.Message = &message
	m.OfferPrice = nil
	m.OfferStatus = "none"

	const query = `
		INSERT INTO negotiation_messages (
			negotiation_id,
			sender_id,
			message_type,
			offer_price,
			message,
			offer_status
		)
		VALUES ($1, $2, 'message', NULL, $3, 'none')
		RETURNING id, created_at
	`

	return r.db.QueryRow(
		query,
		m.NegotiationID,
		m.SenderID,
		m.Message,
	).Scan(
		&m.ID,
		&m.CreatedAt,
	)
}

// ============================================================
// CREATE OFFER
// ============================================================

func (r *NegotiationMessageRepository) createOffer(
	m *models.NegotiationMessage,
) error {

	if m.OfferPrice == nil {
		return errors.New(
			"offer price is required",
		)
	}

	if *m.OfferPrice <= 0 {
		return errors.New(
			"offer price must be greater than zero",
		)
	}

	if m.Message != nil {
		message := strings.TrimSpace(*m.Message)

		if len(message) > 5000 {
			return errors.New(
				"message is too long",
			)
		}

		if message == "" {
			m.Message = nil
		} else {
			m.Message = &message
		}
	}

	m.OfferStatus = "pending"

	const query = `
		INSERT INTO negotiation_messages (
			negotiation_id,
			sender_id,
			message_type,
			offer_price,
			message,
			offer_status
		)
		VALUES ($1, $2, 'offer', $3, $4, 'pending')
		RETURNING id, created_at
	`

	return r.db.QueryRow(
		query,
		m.NegotiationID,
		m.SenderID,
		m.OfferPrice,
		m.Message,
	).Scan(
		&m.ID,
		&m.CreatedAt,
	)
}

// ============================================================
// LIST BY NEGOTIATION
// ============================================================

// ListByNegotiation returns the complete negotiation history.
//
// History includes:
//
//	message
//	pending offer
//	accepted offer
//	rejected offer
func (r *NegotiationMessageRepository) ListByNegotiation(
	negotiationID int,
) ([]models.NegotiationMessage, error) {

	if r == nil || r.db == nil {
		return nil, errors.New(
			"negotiation message repository database is not configured",
		)
	}

	if negotiationID <= 0 {
		return nil, errors.New(
			"invalid negotiation ID",
		)
	}

	const query = `
		SELECT
			id,
			negotiation_id,
			sender_id,
			message_type,
			offer_price,
			message,
			offer_status,
			created_at
		FROM negotiation_messages
		WHERE negotiation_id = $1
		ORDER BY created_at ASC, id ASC
	`

	rows, err := r.db.Query(
		query,
		negotiationID,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	messages := make(
		[]models.NegotiationMessage,
		0,
	)

	for rows.Next() {

		var m models.NegotiationMessage

		if err := rows.Scan(
			&m.ID,
			&m.NegotiationID,
			&m.SenderID,
			&m.MessageType,
			&m.OfferPrice,
			&m.Message,
			&m.OfferStatus,
			&m.CreatedAt,
		); err != nil {
			return nil, err
		}

		messages = append(
			messages,
			m,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// ============================================================
// GET BY ID
// ============================================================

// GetByID retrieves one negotiation message.
//
// It returns nil, nil when the message does not exist.
func (r *NegotiationMessageRepository) GetByID(
	messageID int,
) (*models.NegotiationMessage, error) {

	if r == nil || r.db == nil {
		return nil, errors.New(
			"negotiation message repository database is not configured",
		)
	}

	if messageID <= 0 {
		return nil, errors.New(
			"invalid message ID",
		)
	}

	const query = `
		SELECT
			id,
			negotiation_id,
			sender_id,
			message_type,
			offer_price,
			message,
			offer_status,
			created_at
		FROM negotiation_messages
		WHERE id = $1
	`

	var m models.NegotiationMessage

	err := r.db.QueryRow(
		query,
		messageID,
	).Scan(
		&m.ID,
		&m.NegotiationID,
		&m.SenderID,
		&m.MessageType,
		&m.OfferPrice,
		&m.Message,
		&m.OfferStatus,
		&m.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &m, nil
}

// ============================================================
// UPDATE OFFER STATUS
// ============================================================

// UpdateOfferStatus changes one specific offer.
//
// Valid transitions supported by the application:
//
//	pending → accepted
//	pending → rejected
//
// A normal message can never be updated as an offer.
//
// The negotiation itself is NOT changed here.
func (r *NegotiationMessageRepository) UpdateOfferStatus(
	offerID int,
	status string,
) error {

	if r == nil || r.db == nil {
		return errors.New(
			"negotiation message repository database is not configured",
		)
	}

	if offerID <= 0 {
		return errors.New(
			"invalid offer ID",
		)
	}

	status = strings.ToLower(
		strings.TrimSpace(status),
	)

	switch status {
	case "accepted", "rejected":
		// valid
	default:
		return errors.New(
			"invalid offer status",
		)
	}

	const query = `
		UPDATE negotiation_messages
		SET offer_status = $1
		WHERE id = $2
		  AND message_type = 'offer'
		  AND offer_status = 'pending'
	`

	result, err := r.db.Exec(
		query,
		status,
		offerID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
