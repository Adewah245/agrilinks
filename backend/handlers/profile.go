package handlers

import (
	"log"
	"net/http"

	"backend/internal/models"
	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

type Profile struct {
	crop  *services.CropService
	order *services.OrderService
}

func NewProfileHandler(crop *services.CropService, order *services.OrderService) *Profile {
	return &Profile{crop: crop, order: order}
}

// ProfilePageData embeds the farmer/buyer so the template can use .FullName,
// .Phone, .Location, .Role, .CreatedAt directly, plus role-specific data.
type ProfilePageData struct {
	*models.Farmer
	Crops     []models.Crop
	Purchases []models.Order
	Sales     []models.Order
}

func (h *Profile) ProfileHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("User Visited Profile")

	farmer, ok := middleware.FarmerFromContext(r)
	if !ok || farmer == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := ProfilePageData{Farmer: farmer}

	if farmer.IsBuyer() {
		if purchases, err := h.order.MyPurchases(farmer.ID); err == nil {
			data.Purchases = purchases
		} else {
			log.Println("failed to load purchases:", err)
		}
	} else {
		if crops, err := h.crop.MyCrops(farmer.ID); err == nil {
			data.Crops = crops
		} else {
			log.Println("failed to load crops:", err)
		}
		if sales, err := h.order.MySales(farmer.ID); err == nil {
			data.Sales = sales
		} else {
			log.Println("failed to load sales:", err)
		}
	}

	if err := render.RenderTemplates(w, "profile.html", data); err != nil {
		log.Println("err Render Problem", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
