package services

import (
	"errors"

	"backend/internal/models"
	"backend/internal/repository"
)

type OrderService struct {
	orderRepo   *repository.OrderRepository
	productRepo *repository.ProductRepository
}

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
// The purchase is completed immediately in the current version.
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

	if product == nil || !product.IsAvailable() {
		return errors.New("this product is not available for purchase")
	}

	// A user cannot purchase their own product.
	// Organization-owned products are allowed because the buyer
	// is represented by a user account.
	if product.OwnerType == "user" && product.OwnerID == buyerID {
		return errors.New("you can't buy your own product")
	}

	if quantity > product.Quantity {
		return errors.New("not enough quantity available")
	}

	order := &models.Order{
		BuyerID:    buyerID,
		ProductID:  productID,
		Quantity:   quantity,
		TotalPrice: quantity * product.Price,
		Status:     "completed",
	}

	if err := s.orderRepo.Create(order); err != nil {
		return err
	}

	if err := s.productRepo.ReduceQuantity(productID, quantity); err != nil {
		return err
	}

	return nil
}

// MyPurchases returns all orders placed by a buyer.
func (s *OrderService) MyPurchases(buyerID int) ([]models.Order, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}

	return s.orderRepo.ListByBuyer(buyerID)
}

// MySales returns all orders made from a seller's products.
func (s *OrderService) MySales(
	sellerType string,
	sellerID int,
) ([]models.Order, error) {
	if sellerType != "user" && sellerType != "organization" {
		return nil, errors.New("invalid seller type")
	}

	if sellerID <= 0 {
		return nil, errors.New("invalid seller")
	}

	return s.orderRepo.ListSalesBySeller(sellerType, sellerID)
}
