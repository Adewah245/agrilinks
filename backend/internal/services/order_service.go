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
//
//	↓
//
// OrderService
//
//	↓
//
// OrderRepository + ProductRepository
//
//	↓
//
// # PostgreSQL
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
	cartRepo    *repository.CartRepository
}

// NewOrderService creates a new OrderService.
func NewOrderService(
	orderRepo *repository.OrderRepository,
	productRepo *repository.ProductRepository,
	cartRepo *repository.CartRepository,
) *OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
		cartRepo:    cartRepo,
	}
}

// Checkout converts the buyer's cart into one order.
//
// A cart may contain several products from the same seller. Checkout keeps
// them in one order and uses the prices stored in the cart, preserving any
// negotiated agreement. Separate sellers must be checked out separately so a
// payment can be attributed to exactly one seller.
func (s *OrderService) Checkout(buyerID int) (*models.Order, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}

	if s.cartRepo == nil || s.orderRepo == nil {
		return nil, errors.New("checkout is not configured")
	}

	items, err := s.cartRepo.ListByBuyer(buyerID)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, errors.New("your cart is empty")
	}

	order, cartIDs, err := buildOrderFromCart(buyerID, items)
	if err != nil {
		return nil, err
	}

	if err := s.orderRepo.CreateFromCart(order, cartIDs); err != nil {
		return nil, err
	}

	return order, nil
}

func buildOrderFromCart(
	buyerID int,
	items []models.CartItem,
) (*models.Order, []int, error) {
	if buyerID <= 0 {
		return nil, nil, errors.New("invalid buyer")
	}

	if len(items) == 0 {
		return nil, nil, errors.New("your cart is empty")
	}

	order := &models.Order{
		BuyerID: buyerID,
		Status:  "pending",
		Items:   make([]models.OrderItem, 0, len(items)),
	}
	cartIDs := make([]int, 0, len(items))

	var sellerType string
	var sellerID int

	for i := range items {
		item := &items[i]

		if item.ID <= 0 || item.ProductID <= 0 {
			return nil, nil, errors.New("cart contains an invalid item")
		}
		if item.Quantity <= 0 {
			return nil, nil, errors.New("The cart contains an invalid amount.")
		}
		if item.PricePerUnit < 0 {
			return nil, nil, errors.New("cart item price cannot be negative")
		}
		if item.SellerType != "user" && item.SellerType != "organization" {
			return nil, nil, errors.New("product has an invalid owner type")
		}
		if item.SellerID <= 0 {
			return nil, nil, errors.New("product seller is invalid")
		}
		if item.SellerType == "user" && item.SellerID == buyerID {
			return nil, nil, errors.New("you cannot buy your own product")
		}

		if i == 0 {
			sellerType = item.SellerType
			sellerID = item.SellerID
		} else if item.SellerType != sellerType || item.SellerID != sellerID {
			return nil, nil, errors.New(
				"check out products from one seller at a time",
			)
		}

		totalPrice := item.Quantity * item.PricePerUnit
		order.TotalAmount += totalPrice
		order.Items = append(order.Items, models.OrderItem{
			ProductID:   item.ProductID,
			Quantity:    item.Quantity,
			UnitPrice:   item.PricePerUnit,
			TotalPrice:  totalPrice,
			ProductName: item.ProductName,
			SellerName:  item.SellerName,
		})
		cartIDs = append(cartIDs, item.ID)
	}

	return order, cartIDs, nil
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
			"Enter how much you want.",
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
			"There is not enough available.",
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
		BuyerID:     buyerID,
		Status:      "confirmed",
		TotalAmount: totalPrice,
		Items: []models.OrderItem{
			{
				ProductID:   productID,
				Quantity:    quantity,
				UnitPrice:   product.Price,
				TotalPrice:  totalPrice,
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
