package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
)

// ErrNegotiationAccessDenied is returned when a user is not a buyer in
// the negotiation and cannot act for the product owner.
var ErrNegotiationAccessDenied = errors.New(
	"you're not part of this negotiation",
)

// NegotiationService handles negotiation business logic.
//
// It is responsible for:
//
//   - Starting negotiations.
//   - Sending normal negotiation messages.
//   - Sending offers and counter-offers.
//   - Accepting individual offers.
//   - Rejecting individual offers.
//   - Keeping negotiation history.
//   - Moving accepted deals into the buyer's cart.
//   - Sending negotiation notifications.
//   - Recording negotiation market events.
//
// The service sits between HTTP handlers and repositories:
//
// Handler
//
//	↓
//
// NegotiationService
//
//	↓
//
// Repositories
//
//	↓
//
// PostgreSQL
type NegotiationService struct {
	repo         *repository.NegotiationRepository
	msgRepo      *repository.NegotiationMessageRepository
	productRepo  *repository.ProductRepository
	cartService  *CartService
	notification *NotificationService
	eventRepo    *repository.MarketEventRepository
	maxRounds    int
	window       time.Duration
}

// NewNegotiationService creates a new NegotiationService connected
// to the negotiation, message, product, cart, notification, and
// market event dependencies.
func NewNegotiationService(
	repo *repository.NegotiationRepository,
	msgRepo *repository.NegotiationMessageRepository,
	productRepo *repository.ProductRepository,
	cartService *CartService,
	notification *NotificationService,
	eventRepo *repository.MarketEventRepository,
	maxRounds int,
	window time.Duration,
) *NegotiationService {
	return &NegotiationService{
		repo:         repo,
		msgRepo:      msgRepo,
		productRepo:  productRepo,
		cartService:  cartService,
		notification: notification,
		eventRepo:    eventRepo,
		maxRounds:    maxRounds,
		window:       window,
	}
}

// StartNegotiation creates a new negotiation for an available product.
//
// The buyer's first offer is stored as the first negotiation message.
// The negotiation remains open for the configured negotiation window.
func (s *NegotiationService) StartNegotiation(
	buyerID,
	productID int,
	quantity,
	offerPrice float64,
	message string,
) (*models.Negotiation, error) {
	if s.maxRounds <= 0 || s.window <= 0 {
		return nil, errors.New(
			"negotiation policy is not configured",
		)
	}

	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}

	if productID <= 0 {
		return nil, errors.New("invalid product")
	}

	if quantity <= 0 {
		return nil, errors.New(
			"Enter how much you want.",
		)
	}

	if offerPrice <= 0 {
		return nil, errors.New(
			"Enter an offer greater than zero.",
		)
	}

	message = strings.TrimSpace(message)

	product, err := s.productRepo.GetByID(productID)
	if err != nil {
		return nil, err
	}

	if product == nil {
		return nil, errors.New("product not found")
	}

	// Only active products with available quantity can be negotiated.
	if !product.IsAvailable() {
		return nil, errors.New(
			"this product is not available for negotiation",
		)
	}

	// A user cannot negotiate on their own product.
	if product.OwnerType == "user" &&
		product.OwnerID == buyerID {
		return nil, errors.New(
			"you can't negotiate on your own product",
		)
	}

	if product.OwnerType == "organization" {
		isMember, err := s.repo.IsOrganizationMember(
			product.OwnerID,
			buyerID,
		)
		if err != nil {
			return nil, err
		}

		if isMember {
			return nil, errors.New(
				"you can't negotiate on your own organization's product",
			)
		}
	}

	// Organization ownership is supported by the product model,
	// but organization-member authorization is handled by the
	// organization layer. We must not treat an organization ID
	// as if it were a user ID.
	if product.OwnerType != "user" &&
		product.OwnerType != "organization" {
		return nil, errors.New(
			"product has an invalid owner type",
		)
	}

	if quantity > product.Quantity {
		return nil, errors.New(
			"There is not enough available.",
		)
	}

	negotiation := &models.Negotiation{
		ProductID: productID,
		BuyerID:   buyerID,
		OwnerType: product.OwnerType,
		OwnerID:   product.OwnerID,
		Quantity:  quantity,
		Status:    "open",
		MaxRounds: s.maxRounds,
		ExpiresAt: time.Now().Add(s.window),
	}

	if err := s.repo.Create(negotiation); err != nil {
		return nil, err
	}

	// Create the buyer's first offer.
	firstOffer := &models.NegotiationMessage{
		NegotiationID: negotiation.ID,
		SenderID:      buyerID,
		MessageType:   "offer",
		OfferPrice:    float64Pointer(offerPrice),
		Message:       stringPointer(message),
		OfferStatus:   "pending",
	}

	if err := s.msgRepo.Create(firstOffer); err != nil {
		return nil, err
	}

	// The first offer consumes one negotiation round.
	if err := s.repo.IncrementRound(
		negotiation.ID,
	); err != nil {
		return nil, err
	}

	// Notify individual product owners.
	//
	// Organization notifications will require organization-member
	// notification routing.
	if s.notification != nil &&
		product.OwnerType == "user" {

		_ = s.notification.CreateNotification(
			product.OwnerID,
			"New negotiation",
			"A buyer has started a negotiation for your product.",
			"negotiation",
		)
	}

	return negotiation, nil
}

// SendMessage sends a normal negotiation message.
//
// Normal messages do not consume negotiation rounds.
// They are separate from price offers.
func (s *NegotiationService) SendMessage(
	negotiationID,
	senderID int,
	message string,
) error {
	if negotiationID <= 0 {
		return errors.New("invalid negotiation ID")
	}

	if senderID <= 0 {
		return errors.New("invalid sender ID")
	}

	message = strings.TrimSpace(message)

	if message == "" {
		return errors.New(
			"message cannot be empty",
		)
	}

	if len(message) > 5000 {
		return errors.New(
			"message is too long",
		)
	}

	negotiation, err := s.repo.GetByID(
		negotiationID,
	)
	if err != nil {
		return err
	}

	if negotiation == nil {
		return errors.New(
			"negotiation not found",
		)
	}

	if err := s.requireParticipant(
		negotiation,
		senderID,
	); err != nil {
		return err
	}

	chatMessage := &models.NegotiationMessage{
		NegotiationID: negotiationID,
		SenderID:      senderID,
		MessageType:   "message",
		Message:       stringPointer(message),
		OfferStatus:   "none",
	}

	if err := s.msgRepo.Create(
		chatMessage,
	); err != nil {
		return err
	}

	// Notify the other participant when the owner is an individual user.
	if s.notification != nil {
		recipientID := s.otherParticipantID(
			negotiation,
			senderID,
		)

		if recipientID > 0 {
			_ = s.notification.CreateNotification(
				recipientID,
				"New negotiation message",
				"You received a new message in a negotiation.",
				"negotiation_message",
			)
		}
	}

	return nil
}

// SendOffer sends a new offer or counter-offer.
//
// Only offers consume negotiation rounds.
// Normal messages do not consume rounds.
func (s *NegotiationService) SendOffer(
	negotiationID,
	senderID int,
	offerPrice float64,
	message string,
) error {
	if negotiationID <= 0 {
		return errors.New("invalid negotiation ID")
	}

	if senderID <= 0 {
		return errors.New("invalid sender ID")
	}

	if offerPrice <= 0 {
		return errors.New(
			"Enter an offer greater than zero.",
		)
	}

	message = strings.TrimSpace(message)

	negotiation, err := s.repo.GetByID(
		negotiationID,
	)
	if err != nil {
		return err
	}

	if negotiation == nil {
		return errors.New(
			"negotiation not found",
		)
	}

	if err := s.requireParticipant(
		negotiation,
		senderID,
	); err != nil {
		return err
	}

	if negotiation.Status != "open" {
		return errors.New(
			"this negotiation is no longer open",
		)
	}

	if negotiation.IsExpired() {
		_ = s.repo.UpdateStatus(
			negotiationID,
			"expired",
		)

		return errors.New(
			"this negotiation has expired",
		)
	}

	if negotiation.RoundCount >= negotiation.MaxRounds {
		return errors.New(
			"you've reached the negotiation round limit — no more offers can be made",
		)
	}

	newOffer := &models.NegotiationMessage{
		NegotiationID: negotiationID,
		SenderID:      senderID,
		MessageType:   "offer",
		OfferPrice:    float64Pointer(offerPrice),
		Message:       stringPointer(message),
		OfferStatus:   "pending",
	}

	if err := s.msgRepo.Create(
		newOffer,
	); err != nil {
		return err
	}

	if err := s.repo.IncrementRound(
		negotiationID,
	); err != nil {
		return err
	}

	if s.notification != nil {
		recipientID := s.otherParticipantID(
			negotiation,
			senderID,
		)

		if recipientID > 0 {
			_ = s.notification.CreateNotification(
				recipientID,
				"New price offer",
				"You received a new price offer.",
				"negotiation_offer",
			)
		}
	}

	return nil
}

// Accept accepts one specific pending offer.
//
// The accepted offer finalizes the negotiation and moves
// the agreed deal into the buyer's cart.
func (s *NegotiationService) Accept(
	negotiationID,
	offerID,
	accepterID int,
) error {
	if negotiationID <= 0 {
		return errors.New("invalid negotiation ID")
	}

	if offerID <= 0 {
		return errors.New("invalid offer ID")
	}

	if accepterID <= 0 {
		return errors.New("invalid accepter ID")
	}

	negotiation, err := s.repo.GetByID(
		negotiationID,
	)
	if err != nil {
		return err
	}

	if negotiation == nil {
		return errors.New(
			"negotiation not found",
		)
	}

	if err := s.requireParticipant(
		negotiation,
		accepterID,
	); err != nil {
		return err
	}

	if negotiation.Status != "open" {
		return errors.New(
			"this negotiation is already finalized",
		)
	}

	if negotiation.IsExpired() {
		_ = s.repo.UpdateStatus(
			negotiationID,
			"expired",
		)

		return errors.New(
			"this negotiation has expired",
		)
	}

	offer, err := s.msgRepo.GetByID(
		offerID,
	)
	if err != nil {
		return err
	}

	if offer == nil {
		return errors.New(
			"offer not found",
		)
	}

	if offer.NegotiationID != negotiationID {
		return errors.New(
			"this offer does not belong to this negotiation",
		)
	}

	if offer.MessageType != "offer" {
		return errors.New(
			"this message is not an offer",
		)
	}

	if offer.OfferStatus != "pending" {
		return errors.New(
			"this offer is no longer pending",
		)
	}

	if offer.SenderID == accepterID {
		return errors.New(
			"you cannot accept your own offer",
		)
	}

	product, err := s.productRepo.GetByID(
		negotiation.ProductID,
	)
	if err != nil {
		return err
	}

	if product == nil {
		return errors.New(
			"product no longer exists",
		)
	}

	if !product.IsAvailable() {
		return errors.New(
			"this product is no longer available for sale",
		)
	}

	if negotiation.Quantity > product.Quantity {
		return errors.New(
			"The amount you asked for is no longer available.",
		)
	}

	if s.cartService == nil {
		return errors.New(
			"cart service is not configured",
		)
	}

	// Add the agreed deal to the buyer's cart before finalizing
	// the negotiation. This prevents an accepted negotiation from
	// being finalized when the cart operation cannot succeed.
	if offer.OfferPrice == nil {
		return errors.New("This offer has no price.")
	}

	if err := s.cartService.addFromNegotiation(
		negotiation,
		*offer.OfferPrice,
	); err != nil {
		return err
	}

	if err := s.msgRepo.UpdateOfferStatus(
		offerID,
		"accepted",
	); err != nil {
		return err
	}

	if err := s.repo.UpdateStatus(
		negotiationID,
		"accepted",
	); err != nil {
		return err
	}

	// Record the negotiation event if event tracking is enabled.
	if s.eventRepo != nil {
		userID := negotiation.BuyerID
		productID := negotiation.ProductID

		_ = s.eventRepo.Record(
			&models.MarketEvent{
				EventType: "negotiation_accepted",
				ProductID: &productID,
				UserID:    &userID,
				Metadata: json.RawMessage(fmt.Sprintf(
					`{"quantity":%g}`,
					negotiation.Quantity,
				)),
			},
		)
	}

	if s.notification != nil {
		recipientID := s.otherParticipantID(
			negotiation,
			accepterID,
		)

		if recipientID > 0 {
			_ = s.notification.CreateNotification(
				recipientID,
				"Negotiation accepted",
				"An offer in your negotiation has been accepted.",
				"negotiation_accepted",
			)
		}
	}

	return nil
}

func float64Pointer(value float64) *float64 { return &value }

// Reject rejects one specific pending offer.
//
// Rejecting an offer does not close the negotiation.
// Other offers and normal messages can continue afterward.
func (s *NegotiationService) Reject(
	negotiationID,
	offerID,
	rejecterID int,
) error {
	if negotiationID <= 0 {
		return errors.New("invalid negotiation ID")
	}

	if offerID <= 0 {
		return errors.New("invalid offer ID")
	}

	if rejecterID <= 0 {
		return errors.New("invalid rejecter ID")
	}

	negotiation, err := s.repo.GetByID(
		negotiationID,
	)
	if err != nil {
		return err
	}

	if negotiation == nil {
		return errors.New(
			"negotiation not found",
		)
	}

	if err := s.requireParticipant(
		negotiation,
		rejecterID,
	); err != nil {
		return err
	}

	if negotiation.Status != "open" {
		return errors.New(
			"this negotiation is no longer open",
		)
	}

	if negotiation.IsExpired() {
		_ = s.repo.UpdateStatus(
			negotiationID,
			"expired",
		)

		return errors.New(
			"this negotiation has expired",
		)
	}

	offer, err := s.msgRepo.GetByID(
		offerID,
	)
	if err != nil {
		return err
	}

	if offer == nil {
		return errors.New(
			"offer not found",
		)
	}

	if offer.NegotiationID != negotiationID {
		return errors.New(
			"this offer does not belong to this negotiation",
		)
	}

	if offer.MessageType != "offer" {
		return errors.New(
			"this message is not an offer",
		)
	}

	if offer.OfferStatus != "pending" {
		return errors.New(
			"this offer is no longer pending",
		)
	}

	if offer.SenderID == rejecterID {
		return errors.New(
			"you cannot reject your own offer",
		)
	}

	if err := s.msgRepo.UpdateOfferStatus(
		offerID,
		"rejected",
	); err != nil {
		return err
	}

	// Record the negotiation event if event tracking is enabled.
	if s.eventRepo != nil {
		userID := negotiation.BuyerID
		productID := negotiation.ProductID

		_ = s.eventRepo.Record(
			&models.MarketEvent{
				EventType: "negotiation_rejected",
				ProductID: &productID,
				UserID:    &userID,
			},
		)
	}

	if s.notification != nil {
		recipientID := s.otherParticipantID(
			negotiation,
			rejecterID,
		)

		if recipientID > 0 {
			_ = s.notification.CreateNotification(
				recipientID,
				"Offer rejected",
				"An offer in your negotiation was rejected.",
				"negotiation_rejected",
			)
		}
	}

	return nil
}

// End closes an open negotiation without accepting an offer.
//
// Only a participant in the negotiation can end it.
func (s *NegotiationService) End(
	negotiationID,
	userID int,
) error {
	if negotiationID <= 0 {
		return errors.New("invalid negotiation ID")
	}

	if userID <= 0 {
		return errors.New("invalid user ID")
	}

	negotiation, err := s.repo.GetByID(
		negotiationID,
	)
	if err != nil {
		return err
	}

	if negotiation == nil {
		return errors.New(
			"negotiation not found",
		)
	}

	if err := s.requireParticipant(
		negotiation,
		userID,
	); err != nil {
		return err
	}

	if negotiation.Status != "open" {
		return errors.New(
			"this negotiation is already finalized",
		)
	}

	if err := s.repo.UpdateStatus(
		negotiationID,
		"closed",
	); err != nil {
		return err
	}

	if s.notification != nil {
		recipientID := s.otherParticipantID(
			negotiation,
			userID,
		)

		if recipientID > 0 {
			_ = s.notification.CreateNotification(
				recipientID,
				"Negotiation ended",
				"The other participant ended the negotiation.",
				"negotiation_ended",
			)
		}
	}

	return nil
}

// Thread retrieves a negotiation and its complete conversation.
//
// The returned messages include normal messages, offers,
// rejected offers, and accepted offers.
func (s *NegotiationService) Thread(
	negotiationID,
	userID int,
) (*models.Negotiation, []models.NegotiationMessage, error) {
	if negotiationID <= 0 {
		return nil, nil, errors.New(
			"invalid negotiation ID",
		)
	}

	negotiation, err := s.repo.GetByID(
		negotiationID,
	)
	if err != nil {
		return nil, nil, err
	}

	if negotiation == nil {
		return nil, nil, errors.New(
			"negotiation not found",
		)
	}

	if err := s.requireParticipant(
		negotiation,
		userID,
	); err != nil {
		return nil, nil, err
	}

	messages, err := s.msgRepo.ListByNegotiation(
		negotiationID,
	)
	if err != nil {
		return negotiation, nil, err
	}

	return negotiation, messages, nil
}

// MyNegotiations retrieves all negotiations involving a user.
func (s *NegotiationService) MyNegotiations(
	userID int,
) ([]models.Negotiation, error) {
	if userID <= 0 {
		return nil, errors.New(
			"invalid user",
		)
	}

	return s.repo.ListForUser(userID)
}

// isNegotiationParticipant determines whether a user may act in a
// negotiation as either the buyer or a representative of the product owner.
func (s *NegotiationService) isNegotiationParticipant(
	negotiation *models.Negotiation,
	userID int,
) (bool, error) {
	if negotiation == nil || userID <= 0 {
		return false, nil
	}

	if negotiation.BuyerID == userID {
		return true, nil
	}

	if negotiation.OwnerType == "user" {
		return negotiation.OwnerID == userID, nil
	}

	if negotiation.OwnerType == "organization" {
		return s.repo.IsOrganizationMember(
			negotiation.OwnerID,
			userID,
		)
	}

	return false, nil
}

func (s *NegotiationService) requireParticipant(
	negotiation *models.Negotiation,
	userID int,
) error {
	allowed, err := s.isNegotiationParticipant(
		negotiation,
		userID,
	)
	if err != nil {
		return err
	}

	if !allowed {
		return ErrNegotiationAccessDenied
	}

	return nil
}

// otherParticipantID returns the user ID of the other participant
// when the negotiation belongs to an individual product owner.
//
// For organization-owned products, zero is returned because the
// organization itself is not represented by a user ID.
func (s *NegotiationService) otherParticipantID(
	negotiation *models.Negotiation,
	senderID int,
) int {
	if negotiation == nil {
		return 0
	}

	if senderID == negotiation.BuyerID &&
		negotiation.OwnerType == "user" {
		return negotiation.OwnerID
	}

	if negotiation.OwnerType == "user" &&
		senderID == negotiation.OwnerID {
		return negotiation.BuyerID
	}

	return 0
}
