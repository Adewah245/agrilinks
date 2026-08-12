package handlers

import (
	"log"
	"net/http"

	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

type Dashboard struct {
	crop  *services.CropService
	order *services.OrderService
	ai    *services.AIService
}

func NewDashboardHandler(crop *services.CropService, order *services.OrderService, ai *services.AIService) *Dashboard {
	return &Dashboard{crop: crop, order: order, ai: ai}
}

// DashboardData is what dashboard.html renders against. Farmer-only fields
// and buyer-only fields are both here; the template shows the right set
// based on IsBuyer.
type DashboardData struct {
	FullName string
	Role     string
	IsBuyer  bool

	// Farmer stats
	ProduceInStorage    int
	ListingsActive      int
	AIDiagnosesThisMonth int
	Revenue              float64

	// Buyer stats
	PurchasesMade int
	TotalSpent    float64
}

func (h *Dashboard) DashBoard(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		log.Println("User Visited Dashboard")

		data := DashboardData{FullName: "user"}

		farmer, ok := middleware.FarmerFromContext(r)
		if ok && farmer != nil {
			data.FullName = farmer.FullName
			data.Role = farmer.Role
			data.IsBuyer = farmer.IsBuyer()

			if farmer.IsBuyer() {
				if purchases, err := h.order.MyPurchases(farmer.ID); err == nil {
					data.PurchasesMade = len(purchases)
					for _, p := range purchases {
						data.TotalSpent += p.TotalPrice
					}
				}
			} else {
				if crops, err := h.crop.MyCrops(farmer.ID); err == nil {
					for _, c := range crops {
						data.ProduceInStorage += int(c.Quantity)
						if c.ListedForSale {
							data.ListingsActive++
						}
					}
				}
				if count, err := h.ai.CountThisMonth(farmer.ID); err == nil {
					data.AIDiagnosesThisMonth = count
				}
				if sales, err := h.order.MySales(farmer.ID); err == nil {
					for _, s := range sales {
						data.Revenue += s.TotalPrice
					}
				}
			}
		}

		if err := render.RenderTemplates(w, "dashboard.html", data); err != nil {
			log.Println("render error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	case http.MethodPost:
		log.Println("user's Choices")
	}
}
