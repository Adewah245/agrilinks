package services

import (
	"errors"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"
)

// OrderService handles business logic related to product orders.
//
// Handler
//	↓
// OrderService
//	↓
// OrderRepository + ProductRepository
//	↓
// PostgreSQL
//
// The service is responsible for:
//   - validating buyers
//   - validating products
//   - checking product availability
//   - preventing self-purchases
//   - validating inventory
//   - calculating order totals
//   - creating orders
//   - reducing inventory
type OrderService struct {
	orderRepo   *repository.OrderRepository
	productRepo *repository.ProductRepository
}

// NewOrderService creates a new OrderService.
func NewOrderService(
	orderRepo *repository.OrderRepository,
	productRepo *repository.ProductRepository,
) *OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
	}
}

// ============================================================
// PLACE ORDER
// ============================================================

// PlaceOrder lets a buyer purchase a quantity of an available
// product.
//
// The current MVP supports one product per checkout request.
//
// Internally, however, the database uses:
//
//	orders
//	    ↓
//	order_items
//	    ↓
//	products
//
// Therefore, the service creates one Order containing one
// OrderItem.
//
// Successful orders use the database-supported "confirmed"
// status.
//
// NOTE:
// Order creation and inventory reduction are currently performed
// by separate repository operations. A future transaction-aware
// implementation should make both operations atomic.
func (s *OrderService) PlaceOrder(
	buyerID int,
	productID int,
	quantity float64,
) error {
	// --------------------------------------------------------
	// BASIC VALIDATION
	// --------------------------------------------------------

	if buyerID <= 0 {
		return errors.New("invalid buyer")
	}

	if productID <= 0 {
		return errors.New("invalid product")
	}

	if quantity <= 0 {
		return errors.New(
			"quantity must be greater than zero",
		)
	}

	// --------------------------------------------------------
	// LOAD PRODUCT
	// --------------------------------------------------------

	product, err := s.productRepo.GetByID(productID)
	if err != nil {
		return err
	}

	if product == nil {
		return errors.New("product not found")
	}

	// --------------------------------------------------------
	// PRODUCT AVAILABILITY
	// --------------------------------------------------------

	if !product.IsAvailable() {
		return errors.New(
			"this product is not available for purchase",
		)
	}

	// --------------------------------------------------------
	// OWNER VALIDATION
	// --------------------------------------------------------

	if product.OwnerType != "user" &&
		product.OwnerType != "organization" {
		return errors.New(
			"invalid product owner type",
		)
	}

	// --------------------------------------------------------
	// PREVENT SELF-PURCHASE
	// --------------------------------------------------------

	// A user cannot purchase their own product.
	//
	// Organization-owned products are allowed because the buyer
	// is represented by a user account.
	if product.OwnerType == "user" &&
		product.OwnerID == buyerID {
		return errors.New(
			"you can't buy your own product",
		)
	}

	// --------------------------------------------------------
	// INVENTORY VALIDATION
	// --------------------------------------------------------

	if quantity > product.Quantity {
		return errors.New(
			"not enough quantity available",
		)
	}

	// --------------------------------------------------------
	// CALCULATE TOTAL
	// --------------------------------------------------------

	totalPrice := quantity * product.Price

	if totalPrice < 0 {
		return errors.New("invalid order total")
	}

	// --------------------------------------------------------
	// BUILD ORDER
	// --------------------------------------------------------

	order := &models.Order{
		BuyerID: buyerID,
		Status:  "confirmed",
		TotalAmount: totalPrice,
		Items: []models.OrderItem{
			{
				ProductID: productID,
				Quantity:  quantity,
				UnitPrice: product.Price,
				TotalPrice: totalPrice,
				ProductName: product.Name,
				SellerName:  product.SellerName,
			},
		},
	}

	// --------------------------------------------------------
	// CREATE ORDER
	// --------------------------------------------------------

	if err := s.orderRepo.Create(order); err != nil {
		return err
	}

	// --------------------------------------------------------
	// REDUCE INVENTORY
	// --------------------------------------------------------

	if err := s.productRepo.ReduceQuantity(
		productID,
		quantity,
	); err != nil {
		// IMPORTANT:
		//
		// The order has already been created at this point.
		// Therefore, an inventory failure can leave an order
		// without the corresponding inventory reduction.
		//
		// The long-term solution is a transaction-aware service
		// that creates the order and reduces inventory atomically.
		return err
	}

	return nil
}

// ============================================================
// MY PURCHASES
// ============================================================

// MyPurchases returns all orders placed by a buyer.
func (s *OrderService) MyPurchases(
	buyerID int,
) ([]models.Order, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}

	return s.orderRepo.ListByBuyer(buyerID)
}

// ============================================================
// MY SALES
// ============================================================

// MySales returns orders containing products owned by the
// specified seller.
//
// sellerType:
//   - "user"
//   - "organization"
//
// sellerID identifies the corresponding seller.
func (s *OrderService) MySales(
	sellerType string,
	sellerID int,
) ([]models.Order, error) {
	sellerType = strings.ToLower(
		strings.TrimSpace(sellerType),
	)

	if sellerType != "user" &&
		sellerType != "organization" {
		return nil, errors.New(
			"invalid seller type",
		)
	}

	if sellerID <= 0 {
		return nil, errors.New(
			"invalid seller",
		)
	}

	return s.orderRepo.ListSalesBySeller(
		sellerType,
		sellerID,
	)
}