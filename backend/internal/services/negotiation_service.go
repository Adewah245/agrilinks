package services

import (
	"errors"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
)

const (
	MaxNegotiationRounds = 5
	NegotiationWindow    = 24 * time.Hour
)

type NegotiationService struct {
	repo      *repository.NegotiationRepository
	msgRepo   *repository.NegotiationMessageRepository
	cropRepo  *repository.CropRepository
	orderRepo *repository.OrderRepository
}

func NewNegotiationService(
	repo *repository.NegotiationRepository,
	msgRepo *repository.NegotiationMessageRepository,
	cropRepo *repository.CropRepository,
	orderRepo *repository.OrderRepository,
) *NegotiationService {
	return &NegotiationService{repo: repo, msgRepo: msgRepo, cropRepo: cropRepo, orderRepo: orderRepo}
}

// StartNegotiation lets a buyer open a price negotiation on a listed crop
// with an opening offer. The negotiation is capped at 5 rounds total and
// expires 24 hours after it's opened.
func (s *NegotiationService) StartNegotiation(buyerID, cropID int, quantity, offerPrice float64, message string) (*models.Negotiation, error) {
	crop, err := s.cropRepo.GetByID(cropID)
	if err != nil {
		return nil, err
	}
	if crop == nil || !crop.ListedForSale {
		return nil, errors.New("this produce is not available for negotiation")
	}
	if crop.FarmerID == buyerID {
		return nil, errors.New("you can't negotiate on your own produce")
	}
	if quantity <= 0 || quantity > crop.Quantity {
		return nil, errors.New("invalid quantity")
	}
	if offerPrice <= 0 {
		return nil, errors.New("offer price must be greater than zero")
	}

	negotiation := &models.Negotiation{
		CropID:    cropID,
		BuyerID:   buyerID,
		FarmerID:  crop.FarmerID,
		Quantity:  quantity,
		Status:    "open",
		MaxRounds: MaxNegotiationRounds,
		ExpiresAt: time.Now().Add(NegotiationWindow),
	}

	if err := s.repo.Create(negotiation); err != nil {
		return nil, err
	}

	if err := s.msgRepo.Create(&models.NegotiationMessage{
		NegotiationID: negotiation.ID,
		SenderID:      buyerID,
		OfferPrice:    offerPrice,
		Message:       message,
	}); err != nil {
		return nil, err
	}

	if err := s.repo.IncrementRound(negotiation.ID); err != nil {
		return nil, err
	}

	return negotiation, nil
}

// SendOffer lets either party post a counter-offer, as long as rounds and
// time remain on this negotiation.
func (s *NegotiationService) SendOffer(negotiationID, senderID int, offerPrice float64, message string) error {
	negotiation, err := s.repo.GetByID(negotiationID)
	if err != nil {
		return err
	}
	if negotiation == nil {
		return errors.New("negotiation not found")
	}
	if senderID != negotiation.BuyerID && senderID != negotiation.FarmerID {
		return errors.New("you're not part of this negotiation")
	}
	if negotiation.Status != "open" {
		return errors.New("this negotiation is closed")
	}
	if negotiation.IsExpired() {
		_ = s.repo.UpdateStatus(negotiationID, "expired")
		return errors.New("this negotiation has expired")
	}
	if negotiation.RoundCount >= negotiation.MaxRounds {
		return errors.New("you've reached the 5-round negotiation limit — accept, reject, or start a new negotiation")
	}
	if offerPrice <= 0 {
		return errors.New("offer price must be greater than zero")
	}

	if err := s.msgRepo.Create(&models.NegotiationMessage{
		NegotiationID: negotiationID,
		SenderID:      senderID,
		OfferPrice:    offerPrice,
		Message:       message,
	}); err != nil {
		return err
	}

	return s.repo.IncrementRound(negotiationID)
}

// Accept finalizes the negotiation at the most recently offered price by
// creating a real order, exactly like a normal marketplace purchase.
func (s *NegotiationService) Accept(negotiationID, accepterID int) error {
	negotiation, err := s.repo.GetByID(negotiationID)
	if err != nil {
		return err
	}
	if negotiation == nil {
		return errors.New("negotiation not found")
	}
	if accepterID != negotiation.BuyerID && accepterID != negotiation.FarmerID {
		return errors.New("you're not part of this negotiation")
	}
	if negotiation.Status != "open" {
		return errors.New("this negotiation is already closed")
	}

	lastOffer, err := s.msgRepo.LastOffer(negotiationID)
	if err != nil {
		return err
	}
	if lastOffer == nil {
		return errors.New("there's no offer to accept yet")
	}

	order := &models.Order{
		BuyerID:    negotiation.BuyerID,
		CropID:     negotiation.CropID,
		Quantity:   negotiation.Quantity,
		TotalPrice: lastOffer.OfferPrice * negotiation.Quantity,
		Status:     "completed",
	}

	if err := s.orderRepo.Create(order); err != nil {
		return err
	}
	if err := s.cropRepo.ReduceQuantity(negotiation.CropID, negotiation.Quantity); err != nil {
		return err
	}

	return s.repo.UpdateStatus(negotiationID, "accepted")
}

func (s *NegotiationService) Reject(negotiationID, rejecterID int) error {
	negotiation, err := s.repo.GetByID(negotiationID)
	if err != nil {
		return err
	}
	if negotiation == nil {
		return errors.New("negotiation not found")
	}
	if rejecterID != negotiation.BuyerID && rejecterID != negotiation.FarmerID {
		return errors.New("you're not part of this negotiation")
	}
	return s.repo.UpdateStatus(negotiationID, "rejected")
}

func (s *NegotiationService) Thread(negotiationID int) (*models.Negotiation, []models.NegotiationMessage, error) {
	negotiation, err := s.repo.GetByID(negotiationID)
	if err != nil || negotiation == nil {
		return negotiation, nil, err
	}
	messages, err := s.msgRepo.ListByNegotiation(negotiationID)
	return negotiation, messages, err
}

func (s *NegotiationService) MyNegotiations(userID int) ([]models.Negotiation, error) {
	return s.repo.ListForUser(userID)
}
