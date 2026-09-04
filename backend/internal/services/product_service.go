package services

import (
	"errors"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"
)

// ProductService contains the business logic for agricultural products.
//
// The service sits between handlers and the product repository:
//
// Handler
//
//	↓
//
// ProductService
//
//	↓
//
// ProductRepository
//
//	↓
//
// # PostgreSQL
//
// ProductService is responsible for validating product data,
// enforcing ownership rules, and coordinating product operations.
//
// Authorization for acting on behalf of an organization should be
// established by the caller or a dedicated organization membership
// service before an organization owner ID is supplied here.
type ProductService struct {
	repo *repository.ProductRepository
}

// NewProductService creates a new ProductService using the provided
// product repository.
func NewProductService(
	repo *repository.ProductRepository,
) *ProductService {
	return &ProductService{
		repo: repo,
	}
}

// ============================================================
// CREATE PRODUCT
// ============================================================

// AddProduct creates a new agricultural product listing.
//
// ownerType identifies who owns the product:
//   - "user"
//   - "organization"
//
// ownerID identifies the corresponding user or organization.
//
// Example:
//
//	ownerType = "user"
//	ownerID   = logged-in user's ID
//
// When listForSale is false, the product is stored as inactive.
// When listForSale is true, the product is immediately available
// on the marketplace.
//
// Product creation and image creation are currently separate
// repository operations. They should eventually be wrapped in
// one database transaction so a product cannot exist without
// its required images.
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
	ownerType = strings.ToLower(strings.TrimSpace(ownerType))
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	category = strings.TrimSpace(category)
	unit = strings.TrimSpace(unit)
	community = strings.TrimSpace(community)

	if err := validateProductOwner(ownerType, ownerID); err != nil {
		return err
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

	if listForSale {
		if quantity <= 0 {
			return errors.New(
				"quantity must be greater than zero to list for sale",
			)
		}

		if price <= 0 {
			return errors.New(
				"price must be greater than zero to list for sale",
			)
		}
	}

	// A new product must have at least one picture.
	validImages := cleanImageURLs(imageURLs)

	if len(validImages) == 0 {
		return errors.New(
			"at least one product picture is required",
		)
	}

	if err := validateProductLocation(
		latitude,
		longitude,
		accuracy,
	); err != nil {
		return err
	}

	status := "inactive"

	if listForSale {
		status = "active"
	}

	product := &models.Product{
		OwnerType:        ownerType,
		OwnerID:          ownerID,
		Name:             name,
		Description:      description,
		Category:         category,
		Quantity:         quantity,
		Unit:             unit,
		Price:            price,
		StateID:          stateID,
		LGAID:            lgaID,
		Community:        community,
		Latitude:         latitude,
		Longitude:        longitude,
		LocationAccuracy: accuracy,
		Status:           status,
	}

	if err := s.repo.Create(product); err != nil {
		return err
	}

	// Save all supplied product pictures.
	if err := s.repo.AddImages(
		product.ID,
		validImages,
	); err != nil {
		// Product creation succeeded but image creation failed.
		// The eventual transaction layer should make these operations
		// atomic.
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
//
// ownerType must be either "user" or "organization".
func (s *ProductService) MyProducts(
	ownerType string,
	ownerID int,
) ([]models.Product, error) {
	ownerType = strings.ToLower(strings.TrimSpace(ownerType))

	if err := validateProductOwner(
		ownerType,
		ownerID,
	); err != nil {
		return nil, err
	}

	return s.repo.ListByOwner(
		ownerType,
		ownerID,
	)
}

// ============================================================
// MARKETPLACE
// ============================================================

// AvailableProducts returns products currently available
// for buyers in the marketplace.
func (s *ProductService) AvailableProducts() (
	[]models.Product,
	error,
) {
	return s.repo.ListAvailable()
}

// ============================================================
// GET PRODUCT
// ============================================================

// GetProduct retrieves one product by its ID.
func (s *ProductService) GetProduct(
	productID int,
) (*models.Product, error) {
	if productID <= 0 {
		return nil, errors.New("invalid product ID")
	}

	product, err := s.repo.GetByID(productID)
	if err != nil {
		return nil, err
	}

	if product == nil {
		return nil, errors.New("product not found")
	}

	return product, nil
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
//
// If imageURLs contains valid images, those images are added to the
// existing product images. Existing images are not automatically deleted.
//
// The caller must already have authorization to act for the specified
// owner. In particular, organization ownership should be checked against
// organization membership before this method is called.
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
	ownerType = strings.ToLower(strings.TrimSpace(ownerType))
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	category = strings.TrimSpace(category)
	unit = strings.TrimSpace(unit)
	community = strings.TrimSpace(community)

	if err := validateProductOwner(
		ownerType,
		ownerID,
	); err != nil {
		return err
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

	if listForSale {
		if quantity <= 0 {
			return errors.New(
				"quantity must be greater than zero to list for sale",
			)
		}

		if price <= 0 {
			return errors.New(
				"price must be greater than zero to list for sale",
			)
		}
	}

	if err := validateProductLocation(
		latitude,
		longitude,
		accuracy,
	); err != nil {
		return err
	}

	product, err := s.repo.GetByID(productID)
	if err != nil {
		return err
	}

	if product == nil {
		return errors.New("product not found")
	}

	// Verify ownership before modifying the product.
	if product.OwnerType != ownerType ||
		product.OwnerID != ownerID {
		return errors.New(
			"you are not allowed to modify this product",
		)
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

	// Add new pictures only when valid images were supplied.
	//
	// Existing pictures remain untouched.
	validImages := cleanImageURLs(imageURLs)

	if len(validImages) > 0 {
		if err := s.repo.AddImages(
			product.ID,
			validImages,
		); err != nil {
			return err
		}
	}

	return nil
}

// ============================================================
// UNLIST PRODUCT
// ============================================================

// UnlistProduct removes an owner's product from the marketplace
// while keeping the product and its inventory in storage.
func (s *ProductService) UnlistProduct(
	ownerType string,
	ownerID int,
	productID int,
) error {
	ownerType = strings.ToLower(strings.TrimSpace(ownerType))

	if err := validateProductOwner(
		ownerType,
		ownerID,
	); err != nil {
		return err
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
		return errors.New(
			"you are not allowed to modify this product",
		)
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
//
// A product must have available inventory before it can be relisted.
func (s *ProductService) RelistProduct(
	ownerType string,
	ownerID int,
	productID int,
) error {
	ownerType = strings.ToLower(strings.TrimSpace(ownerType))

	if err := validateProductOwner(
		ownerType,
		ownerID,
	); err != nil {
		return err
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
		return errors.New(
			"you are not allowed to modify this product",
		)
	}

	if product.Quantity <= 0 {
		return errors.New(
			"cannot relist a product with no quantity available",
		)
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
//
// Product-related records such as images and product history are
// expected to be removed according to the database foreign-key
// rules defined by the product schema.
func (s *ProductService) DeleteProduct(
	ownerType string,
	ownerID int,
	productID int,
) error {
	ownerType = strings.ToLower(strings.TrimSpace(ownerType))

	if err := validateProductOwner(
		ownerType,
		ownerID,
	); err != nil {
		return err
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
		return errors.New(
			"you are not allowed to delete this product",
		)
	}

	return s.repo.Delete(
		productID,
		ownerType,
		ownerID,
	)
}

// ============================================================
// VALIDATION HELPERS
// ============================================================

// validateProductOwner validates the type and ID of a product owner.
//
// Valid owner types are "user" and "organization".
func validateProductOwner(
	ownerType string,
	ownerID int,
) error {
	if ownerType != "user" &&
		ownerType != "organization" {
		return errors.New("invalid owner type")
	}

	if ownerID <= 0 {
		return errors.New("invalid product owner")
	}

	return nil
}

// validateProductLocation validates the GPS coordinates and
// location accuracy supplied for a product.
func validateProductLocation(
	latitude float64,
	longitude float64,
	accuracy float64,
) error {
	if latitude < -90 ||
		latitude > 90 {
		return errors.New("invalid product latitude")
	}

	if longitude < -180 ||
		longitude > 180 {
		return errors.New("invalid product longitude")
	}

	// Coordinates of exactly 0,0 normally indicate that the
	// device did not provide a real location.
	if latitude == 0 && longitude == 0 {
		return errors.New("valid product GPS location is required")
	}

	if accuracy <= 0 {
		return errors.New(
			"valid product GPS accuracy is required",
		)
	}

	return nil
}

// ============================================================
// IMAGE VALIDATION
// ============================================================

// cleanImageURLs removes empty image URLs and trims whitespace.
//
// It returns only non-empty image URLs while preserving their
// original order.
func cleanImageURLs(
	imageURLs []string,
) []string {
	validImages := make(
		[]string,
		0,
		len(imageURLs),
	)

	for _, imageURL := range imageURLs {
		imageURL = strings.TrimSpace(imageURL)

		if imageURL == "" {
			continue
		}

		validImages = append(
			validImages,
			imageURL,
		)
	}

	return validImages
}
