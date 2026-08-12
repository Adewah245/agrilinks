package handlers

import (
	"log"
	"net/http"
	"strconv"

	"backend/internal/models"
	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

type Storage struct {
	crop *services.CropService
}

func NewStorageHandler(crop *services.CropService) *Storage {
	return &Storage{crop: crop}
}

type StoragePageData struct {
	FullName string
	Crops    []models.Crop
	Error    string
}

func (h *Storage) StorageHandler(w http.ResponseWriter, r *http.Request) {
	farmer, ok := middleware.FarmerFromContext(r)
	if !ok || farmer == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Buyers don't have produce to store — send them to the marketplace instead.
	if farmer.IsBuyer() {
		http.Redirect(w, r, "/marketplace", http.StatusSeeOther)
		return
	}

	switch r.Method {
	case http.MethodGet:
		log.Println("User Visited Storage")
		h.render(w, farmer.ID, farmer.FullName, "")

	case http.MethodPost:
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			h.render(w, farmer.ID, farmer.FullName, "Couldn't process the form")
			return
		}

		name := r.FormValue("produce")
		unit := r.FormValue("unit")
		location := r.FormValue("location")
		quantity, _ := strconv.ParseFloat(r.FormValue("quantity"), 64)
		price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
		listForSale := r.FormValue("list_for_sale") == "on"

		imageURL, err := saveUploadedFile(r, "produce_image", "crops")
		if err != nil {
			log.Println("produce image upload failed:", err)
		}

		if err := h.crop.AddCrop(farmer.ID, name, unit, location, quantity, price, listForSale, imageURL); err != nil {
			h.render(w, farmer.ID, farmer.FullName, err.Error())
			return
		}

		http.Redirect(w, r, "/storage", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Storage) render(w http.ResponseWriter, farmerID int, fullName, errMsg string) {
	crops, err := h.crop.MyCrops(farmerID)
	if err != nil {
		log.Println("failed to load crops:", err)
	}

	data := StoragePageData{
		FullName: fullName,
		Crops:    crops,
		Error:    errMsg,
	}

	if err := render.RenderTemplates(w, "storage.html", data); err != nil {
		log.Println("render error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
