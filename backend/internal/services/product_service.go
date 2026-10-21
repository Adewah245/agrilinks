package services

import (
	"errors"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"
)

// ProductService contains the business logic for agricultural products.
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
// The service is responsible for:
//   - validating product input
//   - validating ownership
//   - enforcing marketplace rules
//   - coordinating product and image operations
type ProductService struct {
	repo *repository.ProductRepository
}

// NewProductService creates a new ProductService.
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

// AddProduct creates a new agricultural product.
//
// ownerType:
//   - "user"
//   - "organization"
//
// ownerID is the ID of the corresponding owner.
//
// A product can be created as:
//   - inactive: stored in inventory but not listed
//   - active: immediately available on marketplace
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
	communityID *int,
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

	// --------------------------------------------------------
	// OWNER VALIDATION
	// --------------------------------------------------------

	if err := validateProductOwner(ownerType, ownerID); err != nil {
		return err
	}

	// --------------------------------------------------------
	// BASIC PRODUCT VALIDATION
	// --------------------------------------------------------

	if name == "" {
		return errors.New("product name is required")
	}

	if category == "" {
		return errors.New("product category is required")
	}

	if unit == "" {
		return errors.New("Say how this product is sold, for example kg or bag.")
	}

	if quantity <= 0 {
		return errors.New("Enter how much you have.")
	}

	if price < 0 {
		return errors.New("price cannot be negative")
	}

	// --------------------------------------------------------
	// MARKETPLACE VALIDATION
	// --------------------------------------------------------

	if listForSale && price <= 0 {
		return errors.New(
			"price must be greater than zero to list for sale",
		)
	}

	// --------------------------------------------------------
	// IMAGE VALIDATION
	// --------------------------------------------------------

	validImages := cleanImageURLs(imageURLs)

	if len(validImages) == 0 {
		return errors.New(
			"at least one product picture is required",
		)
	}

	// --------------------------------------------------------
	// LOCATION VALIDATION
	// --------------------------------------------------------
	//
	// Location fields are optional in the database.
	// Therefore, we only validate values when coordinates
	// have actually been supplied.

	if err := validateProductLocation(
		latitude,
		longitude,
		accuracy,
	); err != nil {
		return err
	}

	// --------------------------------------------------------
	// STATUS
	// --------------------------------------------------------

	status := "inactive"

	if listForSale {
		status = "active"
	}

	// --------------------------------------------------------
	// BUILD PRODUCT MODEL
	// --------------------------------------------------------

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
		CommunityID:      communityID,
		Latitude:         latitude,
		Longitude:        longitude,
		LocationAccuracy: accuracy,
		Status:           status,
	}

	// --------------------------------------------------------
	// CREATE PRODUCT
	// --------------------------------------------------------

	if err := s.repo.Create(product); err != nil {
		return err
	}

	// --------------------------------------------------------
	// CREATE PRODUCT IMAGES
	// --------------------------------------------------------

	if err := s.repo.AddImages(
		product.ID,
		validImages,
	); err != nil {
		// Product creation succeeded but image creation failed.
		//
		// Future improvement:
		// Wrap product + image creation in one database
		// transaction so the operations become atomic.
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

// SearchAvailable returns available products whose product, seller, or
// location details match the buyer's search term.
func (s *ProductService) SearchAvailable(
	term string,
) ([]models.Product, error) {
	return s.repo.SearchAvailable(term)
}

// ============================================================
// GET PRODUCT
// ============================================================

// GetProduct retrieves a product by ID.
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

// UpdateProduct updates an existing product.
//
// Existing product images are preserved.
// New valid images are added to the existing images.
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
	communityID *int,
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

	// --------------------------------------------------------
	// VALIDATE OWNER
	// --------------------------------------------------------

	if err := validateProductOwner(
		ownerType,
		ownerID,
	); err != nil {
		return err
	}

	if productID <= 0 {
		return errors.New("invalid product ID")
	}

	// --------------------------------------------------------
	// VALIDATE PRODUCT DATA
	// --------------------------------------------------------

	if name == "" {
		return errors.New("product name is required")
	}

	if category == "" {
		return errors.New("product category is required")
	}

	if unit == "" {
		return errors.New("Say how this product is sold, for example kg or bag.")
	}

	if quantity <= 0 {
		return errors.New("Enter how much you have.")
	}

	if price < 0 {
		return errors.New("price cannot be negative")
	}

	if listForSale && price <= 0 {
		return errors.New(
			"price must be greater than zero to list for sale",
		)
	}

	// --------------------------------------------------------
	// VALIDATE LOCATION
	// --------------------------------------------------------

	if err := validateProductLocation(
		latitude,
		longitude,
		accuracy,
	); err != nil {
		return err
	}

	// --------------------------------------------------------
	// LOAD EXISTING PRODUCT
	// --------------------------------------------------------

	product, err := s.repo.GetByID(productID)
	if err != nil {
		return err
	}

	if product == nil {
		return errors.New("product not found")
	}

	// --------------------------------------------------------
	// OWNERSHIP CHECK
	// --------------------------------------------------------

	if product.OwnerType != ownerType ||
		product.OwnerID != ownerID {
		return errors.New(
			"you are not allowed to modify this product",
		)
	}

	// --------------------------------------------------------
	// STATUS
	// --------------------------------------------------------

	status := "inactive"

	if listForSale {
		status = "active"
	}

	// --------------------------------------------------------
	// UPDATE MODEL
	// --------------------------------------------------------

	product.Name = name
	product.Description = description
	product.Category = category
	product.Unit = unit
	product.Quantity = quantity
	product.Price = price
	product.StateID = stateID
	product.LGAID = lgaID
	product.CommunityID = communityID
	product.Latitude = latitude
	product.Longitude = longitude
	product.LocationAccuracy = accuracy
	product.Status = status

	// --------------------------------------------------------
	// SAVE PRODUCT
	// --------------------------------------------------------

	if err := s.repo.Update(product); err != nil {
		return err
	}

	// --------------------------------------------------------
	// ADD NEW IMAGES
	// --------------------------------------------------------

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

// UnlistProduct removes a product from the marketplace
// without deleting the product or its inventory.
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
// A product must have available quantity before it can be relisted.
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
			"There is nothing available to relist.",
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

// validateProductOwner validates the product owner.
func validateProductOwner(
	ownerType string,
	ownerID int,
) error {
	switch ownerType {
	case "user", "organization":
		// valid
	default:
		return errors.New(
			"invalid owner type: must be user or organization",
		)
	}

	if ownerID <= 0 {
		return errors.New("invalid product owner")
	}

	return nil
}

// ============================================================
// LOCATION VALIDATION
// ============================================================

// validateProductLocation validates GPS values when supplied.
//
// The database allows location coordinates to be NULL, so a product
// is allowed to have no GPS location.
//
// However, when coordinates are supplied:
//   - latitude must be between -90 and 90
//   - longitude must be between -180 and 180
//   - accuracy must be greater than zero
//
// The 0,0 coordinate is treated as "not supplied".
func validateProductLocation(
	latitude float64,
	longitude float64,
	accuracy float64,
) error {
	// No GPS location supplied.
	if latitude == 0 && longitude == 0 {
		return nil
	}

	if latitude < -90 || latitude > 90 {
		return errors.New("invalid product latitude")
	}

	if longitude < -180 || longitude > 180 {
		return errors.New("invalid product longitude")
	}

	if accuracy <= 0 {
		return errors.New(
			"valid product GPS accuracy is required when location is supplied",
		)
	}

	return nil
}

// ============================================================
// IMAGE VALIDATION
// ============================================================

// cleanImageURLs removes empty image URLs and trims whitespace.
//
// The original order is preserved.
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
