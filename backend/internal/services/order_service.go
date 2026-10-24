package services

import (
	"errors"
	"math"
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
	currency    string
}

// NewOrderService creates a new OrderService.
func NewOrderService(
	orderRepo *repository.OrderRepository,
	productRepo *repository.ProductRepository,
	cartRepo *repository.CartRepository,
	currency string,
) *OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
		cartRepo:    cartRepo,
		currency:    currency,
	}
}

// Checkout converts the buyer's cart into one order.
//
// A cart may contain several products from the same seller. Checkout keeps
// them in one order and uses the prices stored in the cart, preserving any
// negotiated agreement. Separate sellers must be checked out separately so a
// payment can be attributed to exactly one seller.
func (s *OrderService) Checkout(buyerID int, requestKey ...string) (*models.Order, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}
	order := &models.Order{BuyerID: buyerID, Status: "pending", Currency: s.currency}
	if len(requestKey) > 0 {
		order.CheckoutKey = requestKey[0]
	}
	if len(order.CheckoutKey) > 100 {
		return nil, errors.New("invalid checkout request")
	}
	if err := s.orderRepo.CreateFromCart(order, []int{}); err != nil {
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
			return nil, nil, errors.New("the cart contains an invalid amount")
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

// PlaceOrder creates an unpaid order and reserves its stock atomically.
func (s *OrderService) PlaceOrder(buyerID, productID int, quantity float64, requestKey ...string) error {
	if buyerID <= 0 || productID <= 0 || quantity <= 0 || math.IsNaN(quantity) || math.IsInf(quantity, 0) {
		return errors.New("choose a product and enter a valid amount")
	}
	order := &models.Order{BuyerID: buyerID, Currency: s.currency, Status: "pending", Items: []models.OrderItem{{ProductID: productID, Quantity: quantity}}}
	if len(requestKey) > 0 {
		order.CheckoutKey = requestKey[0]
	}
	return s.orderRepo.Create(order)
}

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
