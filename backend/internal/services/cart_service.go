package services

import (
	"encoding/json"
	"errors"
	"fmt"

	"backend/internal/models"
	"backend/internal/repository"
)

// CartService handles business logic related to a buyer's cart.
type CartService struct {
	cartRepo    *repository.CartRepository
	productRepo *repository.ProductRepository
	eventRepo   *repository.MarketEventRepository
}

// NewCartService creates a new CartService connected to the cart,
// product, and market event repositories.
func NewCartService(
	cartRepo *repository.CartRepository,
	productRepo *repository.ProductRepository,
	eventRepo *repository.MarketEventRepository,
) *CartService {
	return &CartService{
		cartRepo:    cartRepo,
		productRepo: productRepo,
		eventRepo:   eventRepo,
	}
}

// AddFromNegotiation adds an accepted negotiation to the buyer's cart.
//
// The method verifies that the negotiation was accepted, that the
// negotiated quantity is valid and still available, and that the
// agreed price is valid. The negotiated price is preserved in the
// cart so that future product price changes do not affect the deal.
func (s *CartService) AddFromNegotiation(
	negotiation *models.Negotiation,
	agreedPrice float64,
) error {
	if negotiation == nil {
		return errors.New("negotiation is required")
	}

	if negotiation.ID <= 0 {
		return errors.New("invalid negotiation")
	}

	if negotiation.BuyerID <= 0 {
		return errors.New("invalid buyer")
	}

	if negotiation.ProductID <= 0 {
		return errors.New("invalid product")
	}

	if negotiation.Status != "accepted" {
		return errors.New(
			"only accepted negotiations can be added to cart",
		)
	}

	if negotiation.Quantity <= 0 {
		return errors.New("invalid negotiation quantity")
	}

	if agreedPrice <= 0 {
		return errors.New("agreed price must be greater than zero")
	}

	// Verify that the product still exists.
	product, err := s.productRepo.GetByID(
		negotiation.ProductID,
	)
	if err != nil {
		return err
	}

	if product == nil {
		return errors.New("product no longer exists")
	}

	// A product that is no longer active cannot be added to the cart.
	if !product.IsAvailable() {
		return errors.New("product is no longer available")
	}

	// Verify that the negotiated quantity is still available.
	if negotiation.Quantity > product.Quantity {
		return errors.New(
			"the requested quantity is no longer available",
		)
	}

	item := &models.CartItem{
		BuyerID:       negotiation.BuyerID,
		ProductID:     negotiation.ProductID,
		NegotiationID: &negotiation.ID,
		Quantity:      negotiation.Quantity,
		PricePerUnit:  agreedPrice,
		TotalPrice:    negotiation.Quantity * agreedPrice,
	}

	if err := s.cartRepo.Create(item); err != nil {
		return err
	}

	// Record the cart activity for marketplace analytics.
	if s.eventRepo != nil {
		userID := item.BuyerID
		productID := item.ProductID

		err := s.eventRepo.Record(
			&models.MarketEvent{
				EventType: "cart_item_added",
				ProductID: &productID,
				UserID:    &userID,
				Metadata: json.RawMessage(fmt.Sprintf(
					`{"quantity":%g}`,
					item.Quantity,
				)),
			},
		)

		// Analytics must not prevent a successful cart operation.
		if err != nil {
			// Intentionally ignored.
			// The cart item has already been created successfully.
		}
	}

	return nil
}

// MyCart returns all cart items belonging to a buyer.
func (s *CartService) MyCart(
	buyerID int,
) ([]models.CartItem, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}

	return s.cartRepo.ListByBuyer(buyerID)
}

// GetCartItem retrieves a specific cart item belonging to a buyer.
//
// The repository performs the ownership check using both the cart
// item ID and buyer ID.
func (s *CartService) GetCartItem(
	buyerID,
	cartID int,
) (*models.CartItem, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}

	if cartID <= 0 {
		return nil, errors.New("invalid cart item")
	}

	item, err := s.cartRepo.GetByID(
		cartID,
		buyerID,
	)
	if err != nil {
		return nil, err
	}

	if item == nil {
		return nil, errors.New("cart item not found")
	}

	return item, nil
}

// RemoveFromCart removes a cart item belonging to the buyer.
//
// The method verifies that the cart item belongs to the buyer
// before allowing it to be deleted.
func (s *CartService) RemoveFromCart(
	buyerID,
	cartID int,
) error {
	if buyerID <= 0 {
		return errors.New("invalid buyer")
	}

	if cartID <= 0 {
		return errors.New("invalid cart item")
	}

	item, err := s.cartRepo.GetByID(
		cartID,
		buyerID,
	)
	if err != nil {
		return err
	}

	if item == nil {
		return errors.New("cart item not found")
	}

	return s.cartRepo.Delete(
		cartID,
		buyerID,
	)
}
