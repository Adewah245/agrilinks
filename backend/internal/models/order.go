package models

import "time"

// Order represents a buyer's order.
//
// An order can contain multiple products. The individual products
// and their purchase details are stored in OrderItems.
type Order struct {
	ID int

	// Buyer who placed the order.
	BuyerID int

	// Order-level information.
	Status      string
	TotalAmount float64

	// Order items.
	Items []OrderItem

	// Display information populated by repository queries.
	BuyerName string

	// Timestamps.
	CreatedAt time.Time
	UpdatedAt time.Time
}

// OrderItem represents one product within an order.
//
// The unit price is captured at the time of purchase so that
// historical orders remain accurate even if the product price
// changes later.
type OrderItem struct {
	ID int

	// Parent order.
	OrderID int

	// Purchased product.
	ProductID int

	// Purchase details.
	Quantity   float64
	UnitPrice  float64
	TotalPrice float64

	// Display information populated by repository queries.
	ProductName string
	SellerName  string

	// Timestamp.
	CreatedAt time.Time
}
