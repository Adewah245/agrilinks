package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/models"
	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

// Storage handles product storage operations for authenticated users.
//
// Storage is responsible for creating, updating, listing, unlisting,
// relisting, and deleting agricultural products owned by the authenticated
// user.
//
// The storage feature is designed to support different agricultural
// activities, including crops, livestock, poultry, fisheries, beekeeping,
// and other product categories supported by Agro-Shield.
//
// Weather-based storage monitoring and agricultural alerts can be added
// later without changing the core product CRUD operations.
type Storage struct {
	product *services.ProductService
}

// communityID accepts the numeric community identifier used by the current
// schema. Older forms may submit a name; in that case no community is stored.
func communityID(value string) *int {
	id, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}

// NewStorageHandler creates a new Storage handler.
func NewStorageHandler(product *services.ProductService) *Storage {
	return &Storage{
		product: product,
	}
}

// StoragePageData contains the data required to render the storage page.
//
// Products contains the authenticated user's products, while EditProduct
// contains the optional product selected for editing. Error contains a
// user-facing message when a storage operation fails.
type StoragePageData struct {
	FullName    string
	Products    []models.Product
	EditProduct *models.Product
	Error       string
}

// StorageHandler handles authenticated user storage requests.
//
// Buyers currently do not have personal product storage and are redirected
// to the marketplace. Other authenticated users can create, update,
// unlist, relist, and delete products they own.
func (h *Storage) StorageHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// ========================================================
	// 1. GET AUTHENTICATED USER
	// ========================================================

	user, ok := middleware.UserFromContext(r)

	if !ok || user == nil {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	// ========================================================
	// 2. BUYER CHECK
	// ========================================================

	// Buyers currently do not have personal product storage.
	if user.IsBuyer() {
		http.Redirect(
			w,
			r,
			"/marketplace",
			http.StatusSeeOther,
		)
		return
	}

	// ========================================================
	// 3. HANDLE REQUEST METHOD
	// ========================================================

	switch r.Method {

	case http.MethodGet:
		// ====================================================
		// VIEW STORAGE
		// ====================================================

		log.Println("User visited storage")

		var editProduct *models.Product

		editID := strings.TrimSpace(
			r.URL.Query().Get("edit"),
		)

		if editID != "" {
			productID, err := strconv.Atoi(editID)

			if err == nil && productID > 0 {
				candidate, getErr := h.product.GetProduct(
					productID,
				)

				if getErr == nil &&
					candidate != nil &&
					candidate.OwnerType == "user" &&
					candidate.OwnerID == user.ID {

					editProduct = candidate
				}
			}
		}

		h.renderPage(
			w,
			user.ID,
			user.FullName,
			"",
			editProduct,
		)

	case http.MethodPost:
		// ====================================================
		// PROCESS STORAGE ACTION
		// ====================================================

		contentType := strings.ToLower(
			r.Header.Get("Content-Type"),
		)

		if strings.HasPrefix(
			contentType,
			"multipart/form-data",
		) {
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				h.render(
					w,
					user.ID,
					user.FullName,
					"Couldn't process the form",
				)
				return
			}
		} else {
			if err := r.ParseForm(); err != nil {
				h.render(
					w,
					user.ID,
					user.FullName,
					"Couldn't process the form",
				)
				return
			}
		}

		action := strings.TrimSpace(
			r.FormValue("action"),
		)

		switch action {

		case "create":
			h.createProduct(
				w,
				r,
				user.ID,
				user.FullName,
			)

		case "update":
			h.updateProduct(
				w,
				r,
				user.ID,
				user.FullName,
			)

		case "unlist":
			h.unlistProduct(
				w,
				r,
				user.ID,
				user.FullName,
			)

		case "relist":
			h.relistProduct(
				w,
				r,
				user.ID,
				user.FullName,
			)

		case "delete":
			h.deleteProduct(
				w,
				r,
				user.ID,
				user.FullName,
			)

		default:
			h.render(
				w,
				user.ID,
				user.FullName,
				"Invalid product action",
			)
		}

	default:
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
	}

}

// createProduct creates a new product owned by the authenticated user.
//
// Product pictures are required when creating a product. Uploaded pictures
// are stored under the products upload directory.
func (h *Storage) createProduct(
	w http.ResponseWriter,
	r *http.Request,
	userID int,
	fullName string,
) {
	name := strings.TrimSpace(
		r.FormValue("produce"),
	)

	description := strings.TrimSpace(
		r.FormValue("description"),
	)

	category := strings.TrimSpace(
		r.FormValue("category"),
	)

	unit := strings.TrimSpace(
		r.FormValue("unit"),
	)

	community := strings.TrimSpace(
		r.FormValue("community"),
	)

	// The location name remains available to the UI.
	//
	// The current ProductService contract stores geographic
	// coordinates separately and does not accept a standalone
	// location-name parameter.
	_ = strings.TrimSpace(
		r.FormValue("location"),
	)

	latitude, _ := strconv.ParseFloat(
		r.FormValue("latitude"),
		64,
	)

	longitude, _ := strconv.ParseFloat(
		r.FormValue("longitude"),
		64,
	)

	accuracy, _ := strconv.ParseFloat(
		r.FormValue("location_accuracy"),
		64,
	)

	quantity, err := strconv.ParseFloat(
		strings.TrimSpace(
			r.FormValue("quantity"),
		),
		64,
	)

	if err != nil {
		h.render(
			w,
			userID,
			fullName,
			"Invalid quantity",
		)
		return
	}

	price, err := strconv.ParseFloat(
		strings.TrimSpace(
			r.FormValue("price"),
		),
		64,
	)

	if err != nil {
		h.render(
			w,
			userID,
			fullName,
			"Invalid price",
		)
		return
	}

	listForSale := r.FormValue(
		"list_for_sale",
	) == "on"

	// Product pictures are compulsory when creating a product.
	imageURLs, err := saveUploadedFiles(
		r,
		"produce_image",
		"products",
	)

	if err != nil {
		log.Println(
			"product image upload failed:",
			err,
		)

		h.render(
			w,
			userID,
			fullName,
			"Failed to upload product picture",
		)
		return
	}

	if len(imageURLs) == 0 {
		h.render(
			w,
			userID,
			fullName,
			"At least one product picture is required",
		)
		return
	}

	// ProductService accepts GPS information after image URLs.
	if err := h.product.AddProduct(
		"user",
		userID,
		name,
		description,
		category,
		unit,
		quantity,
		price,
		nil,
		nil,
		communityID(community),
		imageURLs,
		latitude,
		longitude,
		accuracy,
		listForSale,
	); err != nil {

		h.render(
			w,
			userID,
			fullName,
			err.Error(),
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/storage",
		http.StatusSeeOther,
	)

}

// updateProduct updates an existing product owned by the authenticated user.
//
// New product pictures are optional during an update. When no new picture
// is supplied, ProductService preserves the existing product images.
func (h *Storage) updateProduct(
	w http.ResponseWriter,
	r *http.Request,
	userID int,
	fullName string,
) {
	productID, err := strconv.Atoi(
		strings.TrimSpace(
			r.FormValue("product_id"),
		),
	)

	if err != nil || productID <= 0 {
		h.render(
			w,
			userID,
			fullName,
			"Invalid product ID",
		)
		return
	}

	name := strings.TrimSpace(
		r.FormValue("produce"),
	)

	description := strings.TrimSpace(
		r.FormValue("description"),
	)

	category := strings.TrimSpace(
		r.FormValue("category"),
	)

	unit := strings.TrimSpace(
		r.FormValue("unit"),
	)

	community := strings.TrimSpace(
		r.FormValue("community"),
	)

	// The location name remains available to the UI.
	_ = strings.TrimSpace(
		r.FormValue("location"),
	)

	latitude, _ := strconv.ParseFloat(
		r.FormValue("latitude"),
		64,
	)

	longitude, _ := strconv.ParseFloat(
		r.FormValue("longitude"),
		64,
	)

	accuracy, _ := strconv.ParseFloat(
		r.FormValue("location_accuracy"),
		64,
	)

	quantity, err := strconv.ParseFloat(
		strings.TrimSpace(
			r.FormValue("quantity"),
		),
		64,
	)

	if err != nil {
		h.render(
			w,
			userID,
			fullName,
			"Invalid quantity",
		)
		return
	}

	price, err := strconv.ParseFloat(
		strings.TrimSpace(
			r.FormValue("price"),
		),
		64,
	)

	if err != nil {
		h.render(
			w,
			userID,
			fullName,
			"Invalid price",
		)
		return
	}

	listForSale := r.FormValue(
		"list_for_sale",
	) == "on"

	// New pictures are optional during an update.
	imageURLs, uploadErr := saveUploadedFiles(
		r,
		"produce_image",
		"products",
	)

	if uploadErr != nil {
		log.Println(
			"updated product image upload failed:",
			uploadErr,
		)

		h.render(
			w,
			userID,
			fullName,
			"Failed to upload product picture",
		)
		return
	}

	if err := h.product.UpdateProduct(
		"user",
		userID,
		productID,
		name,
		description,
		category,
		unit,
		quantity,
		price,
		nil,
		nil,
		communityID(community),
		imageURLs,
		latitude,
		longitude,
		accuracy,
		listForSale,
	); err != nil {

		h.render(
			w,
			userID,
			fullName,
			err.Error(),
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/storage",
		http.StatusSeeOther,
	)

}

// unlistProduct removes a product from the marketplace while keeping
// the product in the user's storage.
func (h *Storage) unlistProduct(
	w http.ResponseWriter,
	r *http.Request,
	userID int,
	fullName string,
) {
	productID, err := strconv.Atoi(
		strings.TrimSpace(
			r.FormValue("product_id"),
		),
	)

	if err != nil || productID <= 0 {
		h.render(
			w,
			userID,
			fullName,
			"Invalid product ID",
		)
		return
	}

	if err := h.product.UnlistProduct(
		"user",
		userID,
		productID,
	); err != nil {

		h.render(
			w,
			userID,
			fullName,
			err.Error(),
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/storage",
		http.StatusSeeOther,
	)

}

// relistProduct puts an existing product back on the marketplace.
func (h *Storage) relistProduct(
	w http.ResponseWriter,
	r *http.Request,
	userID int,
	fullName string,
) {
	productID, err := strconv.Atoi(
		strings.TrimSpace(
			r.FormValue("product_id"),
		),
	)

	if err != nil || productID <= 0 {
		h.render(
			w,
			userID,
			fullName,
			"Invalid product ID",
		)
		return
	}

	if err := h.product.RelistProduct(
		"user",
		userID,
		productID,
	); err != nil {

		h.render(
			w,
			userID,
			fullName,
			err.Error(),
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/storage",
		http.StatusSeeOther,
	)

}

// deleteProduct permanently deletes a product owned by the authenticated
// user.
func (h *Storage) deleteProduct(
	w http.ResponseWriter,
	r *http.Request,
	userID int,
	fullName string,
) {
	productID, err := strconv.Atoi(
		strings.TrimSpace(
			r.FormValue("product_id"),
		),
	)

	if err != nil || productID <= 0 {
		h.render(
			w,
			userID,
			fullName,
			"Invalid product ID",
		)
		return
	}

	if err := h.product.DeleteProduct(
		"user",
		userID,
		productID,
	); err != nil {

		h.render(
			w,
			userID,
			fullName,
			err.Error(),
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/storage",
		http.StatusSeeOther,
	)

}

// render renders the storage page with an optional error message.
func (h *Storage) render(
	w http.ResponseWriter,
	userID int,
	fullName string,
	errMsg string,
) {
	h.renderPage(
		w,
		userID,
		fullName,
		errMsg,
		nil,
	)
}

// renderPage loads the authenticated user's products and their images,
// optionally loads the product being edited, and renders storage.html.
//
// This method is also the appropriate place to add future weather,
// storage-monitoring, or agricultural alert data without coupling those
// concerns to the product CRUD methods.
func (h *Storage) renderPage(
	w http.ResponseWriter,
	userID int,
	fullName string,
	errMsg string,
	editProduct *models.Product,
) {
	products, err := h.product.MyProducts(
		"user",
		userID,
	)

	if err != nil {
		log.Println(
			"failed to load products:",
			err,
		)

		http.Error(
			w,
			"Unable to load storage",
			http.StatusInternalServerError,
		)

		return
	}

	// Load images for every product.
	for i := range products {
		images, imageErr := h.product.GetProductImages(
			products[i].ID,
		)

		if imageErr != nil {
			log.Println(
				"failed to load product images:",
				imageErr,
			)
			continue
		}

		products[i].Images = images
	}

	// Load images for the product currently being edited.
	if editProduct != nil {
		images, imageErr := h.product.GetProductImages(
			editProduct.ID,
		)

		if imageErr != nil {
			log.Println(
				"failed to load edit product images:",
				imageErr,
			)
		} else {
			editProduct.Images = images
		}
	}

	data := StoragePageData{
		FullName:    fullName,
		Products:    products,
		EditProduct: editProduct,
		Error:       errMsg,
	}

	if err := render.RenderTemplates(
		w,
		"storage.html",
		data,
	); err != nil {
		log.Println(
			"storage render error:",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
	}

}
