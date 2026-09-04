package services

import (
	"errors"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"
)

// OrderService handles business logic related to product orders.
//
// The service coordinates order creation, product availability,
// inventory reduction, purchases, and sales.
//
// The service does not communicate directly with PostgreSQL.
// Database operations are handled by the repositories.
type OrderService struct {
	orderRepo   *repository.OrderRepository
	productRepo *repository.ProductRepository
}

// NewOrderService creates a new OrderService using the provided
// order and product repositories.
func NewOrderService(
	orderRepo *repository.OrderRepository,
	productRepo *repository.ProductRepository,
) *OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
	}
}

// PlaceOrder lets a buyer purchase a quantity of an available product.
//
// The method verifies that:
//   - the buyer ID is valid,
//   - the product ID is valid,
//   - the requested quantity is greater than zero,
//   - the product exists and is available,
//   - a user does not purchase their own product,
//   - enough inventory is available.
//
// The order is currently completed immediately.
//
// NOTE:
// Order creation and inventory reduction are currently separate
// repository operations. The long-term implementation should wrap
// both operations in one PostgreSQL transaction so they either both
// succeed or both fail.
func (s *OrderService) PlaceOrder(
	buyerID int,
	productID int,
	quantity float64,
) error {
	if buyerID <= 0 {
		return errors.New("invalid buyer")
	}

	if productID <= 0 {
		return errors.New("invalid product")
	}

	if quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}

	product, err := s.productRepo.GetByID(productID)
	if err != nil {
		return err
	}

	if product == nil {
		return errors.New("product not found")
	}

	if !product.IsAvailable() {
		return errors.New(
			"this product is not available for purchase",
		)
	}

	// A user cannot purchase their own product.
	//
	// Organization-owned products are allowed because the buyer
	// is represented by a user account while the product belongs
	// to an organization.
	if product.OwnerType == "user" &&
		product.OwnerID == buyerID {
		return errors.New(
			"you can't buy your own product",
		)
	}

	if product.OwnerType != "user" &&
		product.OwnerType != "organization" {
		return errors.New("invalid product owner type")
	}

	if quantity > product.Quantity {
		return errors.New("not enough quantity available")
	}

	totalPrice := quantity * product.Price

	if totalPrice < 0 {
		return errors.New("invalid order total")
	}

	order := &models.Order{
		BuyerID:    buyerID,
		ProductID:  productID,
		Quantity:   quantity,
		TotalPrice: totalPrice,
		Status:     "completed",
	}

	// Create the order first.
	if err := s.orderRepo.Create(order); err != nil {
		return err
	}

	// Reduce the product inventory after the order has been created.
	//
	// ProductRepository.ReduceQuantity is responsible for updating
	// the product quantity and its inventory/status history.
	if err := s.productRepo.ReduceQuantity(
		productID,
		quantity,
	); err != nil {
		// IMPORTANT:
		// At this stage the order already exists.
		//
		// A database transaction should eventually be used here
		// so the order creation and inventory reduction happen
		// atomically.
		return err
	}

	return nil
}

// MyPurchases returns all orders placed by a buyer.
func (s *OrderService) MyPurchases(
	buyerID int,
) ([]models.Order, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}

	return s.orderRepo.ListByBuyer(buyerID)
}

// MySales returns all orders made from products owned by
// the specified seller.
//
// sellerType must be either:
//   - "user"
//   - "organization"
//
// sellerID is the ID of the corresponding user or organization.
func (s *OrderService) MySales(
	sellerType string,
	sellerID int,
) ([]models.Order, error) {
	sellerType = strings.ToLower(strings.TrimSpace(sellerType))

	if sellerType != "user" &&
		sellerType != "organization" {
		return nil, errors.New("invalid seller type")
	}

	if sellerID <= 0 {
		return nil, errors.New("invalid seller")
	}

	return s.orderRepo.ListSalesBySeller(
		sellerType,
		sellerID,
	)
}
