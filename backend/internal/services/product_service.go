package services

import (
	"errors"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"
)

// ProductService contains the business logic for products.
type ProductService struct {
	repo *repository.ProductRepository
}

// NewProductService creates a new ProductService.
func NewProductService(repo *repository.ProductRepository) *ProductService {
	return &ProductService{
		repo: repo,
	}
}

// ============================================================
// CREATE PRODUCT
// ============================================================

// AddProduct creates a new product listing.
//
// ownerType identifies who owns the product.
// ownerID identifies the owner.
//
// Example:
// ownerType = "user"
// ownerID   = logged-in user's ID
func (s *ProductService) AddProduct(
	ownerType string,
	ownerID int,
	name string,
	description string,
	category string,
	unit string,
	quantity float64,
	price float64,
	stateID *int,
	lgaID *int,
	community string,
	imageURLs []string,
	latitude float64,
	longitude float64,
	accuracy float64,
	listForSale bool,
) error {

	ownerType = strings.TrimSpace(ownerType)
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	category = strings.TrimSpace(category)
	unit = strings.TrimSpace(unit)
	community = strings.TrimSpace(community)

	if ownerType == "" {
		return errors.New("owner type is required")
	}

	if ownerType != "user" && ownerType != "organization" {
		return errors.New("invalid owner type")
	}

	if ownerID <= 0 {
		return errors.New("invalid product owner")
	}

	if name == "" {
		return errors.New("product name is required")
	}

	if category == "" {
		return errors.New("product category is required")
	}

	if unit == "" {
		return errors.New("unit is required")
	}

	if quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}

	if price < 0 {
		return errors.New("price cannot be negative")
	}

	if listForSale && price <= 0 {
		return errors.New("price must be greater than zero to list for sale")
	}

	// A product must have at least one picture.
	validImages := cleanImageURLs(imageURLs)

	if len(validImages) == 0 {
		return errors.New("at least one product picture is required")
	}

	// Validate GPS location.
	if latitude < -90 ||
		latitude > 90 ||
		longitude < -180 ||
		longitude > 180 ||
		(latitude == 0 && longitude == 0) ||
		accuracy <= 0 {
		return errors.New("valid product GPS location is required")
	}

	status := "inactive"

	if listForSale {
		status = "active"
	}

	product := &models.Product{
		OwnerType:         ownerType,
		OwnerID:           ownerID,
		Name:              name,
		Description:       description,
		Category:          category,
		Quantity:          quantity,
		Unit:              unit,
		Price:             price,
		StateID:           stateID,
		LGAID:             lgaID,
		Community:         community,
		Latitude:          latitude,
		Longitude:         longitude,
		LocationAccuracy:  accuracy,
		Status:            status,
	}

	if err := s.repo.Create(product); err != nil {
		return err
	}

	// Save all product pictures.
	if err := s.repo.AddImages(product.ID, validImages); err != nil {
		return err
	}

	return nil
}

// ============================================================
// LIST OWNER PRODUCTS
// ============================================================

// MyProducts returns all products belonging to an owner.
//
// Both active and inactive products are returned.
func (s *ProductService) MyProducts(
	ownerType string,
	ownerID int,
) ([]models.Product, error) {

	ownerType = strings.TrimSpace(ownerType)

	if ownerType == "" {
		return nil, errors.New("owner type is required")
	}

	if ownerID <= 0 {
		return nil, errors.New("invalid product owner")
	}

	return s.repo.ListByOwner(ownerType, ownerID)
}

// ============================================================
// MARKETPLACE
// ============================================================

// AvailableProducts returns products currently available
// for buyers in the marketplace.
func (s *ProductService) AvailableProducts() ([]models.Product, error) {
	return s.repo.ListAvailable()
}

// ============================================================
// GET PRODUCT
// ============================================================

// GetProduct retrieves one product by ID.
func (s *ProductService) GetProduct(productID int) (*models.Product, error) {

	if productID <= 0 {
		return nil, errors.New("invalid product ID")
	}

	return s.repo.GetByID(productID)
}

// ============================================================
// PRODUCT IMAGES
// ============================================================

// GetProductImages retrieves all images belonging to a product.
func (s *ProductService) GetProductImages(
	productID int,
) ([]models.ProductImage, error) {

	if productID <= 0 {
		return nil, errors.New("invalid product ID")
	}

	return s.repo.ListImages(productID)
}

// ============================================================
// UPDATE PRODUCT
// ============================================================

// UpdateProduct updates an owner's product.
//
// Existing pictures are preserved when no new pictures are supplied.
func (s *ProductService) UpdateProduct(
	ownerType string,
	ownerID int,
	productID int,
	name string,
	description string,
	category string,
	unit string,
	quantity float64,
	price float64,
	stateID *int,
	lgaID *int,
	community string,
	imageURLs []string,
	latitude float64,
	longitude float64,
	accuracy float64,
	listForSale bool,
) error {

	ownerType = strings.TrimSpace(ownerType)
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	category = strings.TrimSpace(category)
	unit = strings.TrimSpace(unit)
	community = strings.TrimSpace(community)

	if ownerType != "user" && ownerType != "organization" {
		return errors.New("invalid owner type")
	}

	if ownerID <= 0 {
		return errors.New("invalid product owner")
	}

	if productID <= 0 {
		return errors.New("invalid product ID")
	}

	if name == "" {
		return errors.New("product name is required")
	}

	if category == "" {
		return errors.New("product category is required")
	}

	if unit == "" {
		return errors.New("unit is required")
	}

	if quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}

	if price < 0 {
		return errors.New("price cannot be negative")
	}

	if listForSale && price <= 0 {
		return errors.New("price must be greater than zero to list for sale")
	}

	if latitude < -90 ||
		latitude > 90 ||
		longitude < -180 ||
		longitude > 180 ||
		(latitude == 0 && longitude == 0) ||
		accuracy <= 0 {
		return errors.New("valid product GPS location is required")
	}

	product, err := s.repo.GetByID(productID)
	if err != nil {
		return err
	}

	if product == nil {
		return errors.New("product not found")
	}

	if product.OwnerType != ownerType ||
		product.OwnerID != ownerID {
		return errors.New("you are not allowed to modify this product")
	}

	status := "inactive"

	if listForSale {
		status = "active"
	}

	product.Name = name
	product.Description = description
	product.Category = category
	product.Unit = unit
	product.Quantity = quantity
	product.Price = price
	product.StateID = stateID
	product.LGAID = lgaID
	product.Community = community
	product.Latitude = latitude
	product.Longitude = longitude
	product.LocationAccuracy = accuracy
	product.Status = status

	if err := s.repo.Update(product); err != nil {
		return err
	}

	// Add new pictures only when the caller supplied them.
	validImages := cleanImageURLs(imageURLs)

	if len(validImages) > 0 {
		if err := s.repo.AddImages(product.ID, validImages); err != nil {
			return err
		}
	}

	return nil
}

// ============================================================
// UNLIST PRODUCT
// ============================================================

// UnlistProduct removes an owner's product from the marketplace
// while keeping it in storage.
func (s *ProductService) UnlistProduct(
	ownerType string,
	ownerID int,
	productID int,
) error {

	if ownerType != "user" && ownerType != "organization" {
		return errors.New("invalid owner type")
	}

	if ownerID <= 0 {
		return errors.New("invalid product owner")
	}

	if productID <= 0 {
		return errors.New("invalid product ID")
	}

	product, err := s.repo.GetByID(productID)
	if err != nil {
		return err
	}

	if product == nil {
		return errors.New("product not found")
	}

	if product.OwnerType != ownerType ||
		product.OwnerID != ownerID {
		return errors.New("you are not allowed to modify this product")
	}

	if product.Status != "active" {
		return errors.New("product is already unlisted")
	}

	return s.repo.Unlist(
		productID,
		ownerType,
		ownerID,
	)
}

// ============================================================
// RELIST PRODUCT
// ============================================================

// RelistProduct puts an existing product back on the marketplace.
func (s *ProductService) RelistProduct(
	ownerType string,
	ownerID int,
	productID int,
) error {

	if ownerType != "user" && ownerType != "organization" {
		return errors.New("invalid owner type")
	}

	if ownerID <= 0 {
		return errors.New("invalid product owner")
	}

	if productID <= 0 {
		return errors.New("invalid product ID")
	}

	product, err := s.repo.GetByID(productID)
	if err != nil {
		return err
	}

	if product == nil {
		return errors.New("product not found")
	}

	if product.OwnerType != ownerType ||
		product.OwnerID != ownerID {
		return errors.New("you are not allowed to modify this product")
	}

	if product.Quantity <= 0 {
		return errors.New("cannot relist a product with no quantity available")
	}

	if product.Status == "active" {
		return errors.New("product is already listed")
	}

	return s.repo.Relist(
		productID,
		ownerType,
		ownerID,
	)
}

// ============================================================
// DELETE PRODUCT
// ============================================================

// DeleteProduct permanently deletes an owner's product.
func (s *ProductService) DeleteProduct(
	ownerType string,
	ownerID int,
	productID int,
) error {

	if ownerType != "user" && ownerType != "organization" {
		return errors.New("invalid owner type")
	}

	if ownerID <= 0 {
		return errors.New("invalid product owner")
	}

	if productID <= 0 {
		return errors.New("invalid product ID")
	}

	product, err := s.repo.GetByID(productID)
	if err != nil {
		return err
	}

	if product == nil {
		return errors.New("product not found")
	}

	if product.OwnerType != ownerType ||
		product.OwnerID != ownerID {
		return errors.New("you are not allowed to delete this product")
	}

	return s.repo.Delete(
		productID,
		ownerType,
		ownerID,
	)
}

// ============================================================
// IMAGE VALIDATION
// ============================================================

// cleanImageURLs removes empty image URLs and trims whitespace.
func cleanImageURLs(imageURLs []string) []string {

	validImages := make([]string, 0, len(imageURLs))

	for _, imageURL := range imageURLs {
		imageURL = strings.TrimSpace(imageURL)

		if imageURL != "" {
			validImages = append(validImages, imageURL)
		}
	}

	return validImages
}
