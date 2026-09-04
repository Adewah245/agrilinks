package handlers

import (
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"
	"syscall"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

type Dashboard struct {
	product      *services.ProductService
	order        *services.OrderService
	cart         *services.CartService
	ai           *services.AIService
	negotiation  *services.NegotiationService
	weather      *services.WeatherService
	notification *services.NotificationService
	events       *repository.MarketEventRepository
}

func NewDashboardHandler(
	product *services.ProductService,
	order *services.OrderService,
	cart *services.CartService,
	ai *services.AIService,
	negotiation *services.NegotiationService,
	weather *services.WeatherService,
	notification *services.NotificationService,
	events *repository.MarketEventRepository,
) *Dashboard {
	return &Dashboard{
		product:      product,
		order:        order,
		cart:         cart,
		ai:           ai,
		negotiation:  negotiation,
		weather:      weather,
		notification: notification,
		events:       events,
	}
}

// DashboardData is what dashboard.html renders against.
// Farmer-only fields and buyer-only fields are both here;
// the template shows the right set based on IsBuyer.
type DashboardData struct {
	FullName string
	Role     string
	IsBuyer  bool
	PhotoURL string

	// Farmer stats.
	ProduceInStorage     int
	StorageCrops         []models.Product
	ListingsActive       int
	ListingsWithPhotos   int
	AIDiagnosesThisMonth int
	Revenue              float64
	Revenue30            float64
	CompletedSales       int

	// Buyer stats.
	PurchasesMade   int
	TotalSpent      float64
	AvailableProduce int
	CartItemCount   int
	CartTotal       float64
	HealthScore     int
	HealthStatus    string
	HealthMetrics   []HealthMetric

	UnreadNotifications int

	Weather         services.Weather
	LatestDiagnosis *models.Diagnosis

	MarketListings       int
	MarketPriceMin      float64
	MarketPriceMedian   float64
	MarketPriceMax      float64
	MarketHasPrices     bool
	MarketOrders30      int
	MarketNegotiations30 int

	Demand               models.ProduceDemandSummary
	ProduceDemands       []models.ProduceDemandSummary
	ProduceDemandPreview []models.ProduceDemandSummary

	Priorities      []DashboardPriority
	PrimaryPriority DashboardPriority
	RecentOrders    []models.Order
}

type HealthMetric struct {
	Name    string
	Percent int
}

type DashboardPriority struct {
	Tone, Label, Title, Detail, Action, Href string
}

func (h *Dashboard) DashBoard(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		log.Println("User Visited Dashboard")

		data := DashboardData{
			FullName: "user",
		}

		if h.events != nil {
			if err := h.events.ReconcileCheckoutAbandonments(24 * time.Hour); err != nil {
				log.Printf("dashboard checkout reconciliation failed: %v", err)
			}
		}

		user, ok := middleware.FarmerFromContext(r)
		if ok && user != nil {
			data.FullName = user.FullName
			data.Role = user.Role
			data.IsBuyer = user.IsBuyer()
			data.PhotoURL = user.PhotoURL

			if h.weather != nil {
				var err error

				data.Weather, err = h.weather.Current(
					r.Context(),
					user.Location,
				)

				if err != nil {
					if data.Weather.Available {
						log.Printf(
							"dashboard live weather refresh failed for %q; using cached conditions: %v",
							user.Location,
							err,
						)
					} else {
						log.Printf(
							"dashboard weather unavailable for %q: %v",
							user.Location,
							err,
						)
					}
				}
			}

			// -------------------------------------------------
			// BUYER DASHBOARD
			// -------------------------------------------------
			if user.IsBuyer() {
				if h.cart != nil {
					if items, err := h.cart.MyCart(user.ID); err != nil {
						log.Printf(
							"dashboard cart summary unavailable for buyer %d: %v",
							user.ID,
							err,
						)
					} else {
						data.CartItemCount, data.CartTotal = cartSummary(items)
					}
				}

				if h.product != nil {
					if available, err := h.product.AvailableProducts(); err == nil {
						data.AvailableProduce = len(available)
					}
				}

				if h.order != nil {
					if purchases, err := h.order.MyPurchases(user.ID); err == nil {
						data.PurchasesMade = len(purchases)
						data.RecentOrders = limitOrders(purchases, 5)

						for _, purchase := range purchases {
							data.TotalSpent += purchase.TotalPrice
						}
					}
				}

				data.Priorities = buyerPriorities(data)

			} else {
				// -------------------------------------------------
				// FARMER / SELLER DASHBOARD
				// -------------------------------------------------

				var userProducts []models.Product

				if h.product != nil {
					if got, err := h.product.MyProducts("user", user.ID); err == nil {
						userProducts = got

						data.StorageCrops = got
						data.ProduceInStorage = len(got)

						for _, product := range got {
							if product.Status == "active" {
								data.ListingsActive++

								if len(product.Images) > 0 {
									data.ListingsWithPhotos++
								}
							}
						}
					}
				}

				if h.events != nil && len(userProducts) > 0 {
					cutoff := time.Now().Add(-30 * 24 * time.Hour)
					previous := cutoff.Add(-30 * 24 * time.Hour)

					if summary, err := h.events.FarmerDemandSummary(
						user.ID,
						user.Location,
						cutoff,
						previous,
					); err != nil {
						log.Printf(
							"dashboard demand summary unavailable for farmer %d: %v",
							user.ID,
							err,
						)
					} else {
						data.Demand = summary
					}

					if summaries, err := h.events.GroupedDemandSummaries(
						user.ID,
						user.Location,
						cutoff,
						previous,
					); err != nil {
						log.Printf(
							"dashboard produce comparisons unavailable for farmer %d: %v",
							user.ID,
							err,
						)
					} else {
						data.ProduceDemands = summaries
						data.ProduceDemandPreview =
							prioritizeDemandSummaries(summaries, 3)
					}
				}

				// Marketplace statistics.
				if h.product != nil {
					if available, err := h.product.AvailableProducts(); err == nil {
						data.MarketListings = len(available)

						prices := make([]float64, 0, len(available))

						for _, listing := range available {
							if listing.Price <= 0 {
								continue
							}

							prices = append(prices, listing.Price)
						}

						if len(prices) > 0 {
							sort.Float64s(prices)

							data.MarketPriceMin = prices[0]
							data.MarketPriceMax = prices[len(prices)-1]

							middle := len(prices) / 2

							if len(prices)%2 == 0 {
								data.MarketPriceMedian =
									(prices[middle-1] + prices[middle]) / 2
							} else {
								data.MarketPriceMedian = prices[middle]
							}

							data.MarketHasPrices = true
						}
					}
				}

				// AI diagnosis statistics.
				if h.ai != nil {
					if count, err := h.ai.CountThisMonth(user.ID); err == nil {
						data.AIDiagnosesThisMonth = count
					}
				}

				// Sales statistics.
				if h.order != nil {
					if sales, err := h.order.MySales("user", user.ID); err == nil {
						data.RecentOrders = limitOrders(sales, 5)

						cutoff := time.Now().Add(-30 * 24 * time.Hour)

						for _, sale := range sales {
							data.Revenue += sale.TotalPrice

							if !sale.CreatedAt.Before(cutoff) {
								data.Revenue30 += sale.TotalPrice
								data.CompletedSales++
								data.MarketOrders30++
							}
						}
					}
				}

				// Negotiation statistics.
				if h.negotiation != nil {
					if negotiations, err := h.negotiation.MyNegotiations(user.ID); err == nil {
						cutoff := time.Now().Add(-30 * 24 * time.Hour)

						for _, negotiation := range negotiations {
							if !negotiation.CreatedAt.Before(cutoff) {
								data.MarketNegotiations30++
							}
						}
					}
				}

				// Product list for dashboard priorities.
				products := userProducts

				// AI diagnosis history.
				var diagnoses []models.Diagnosis

				if h.ai != nil {
					if got, err := h.ai.History(user.ID); err == nil {
						diagnoses = got

						if len(diagnoses) > 0 {
							latest := diagnoses[0]
							data.LatestDiagnosis = &latest
						}
					}
				}

				// Negotiations for dashboard priorities.
				var negotiations []models.Negotiation

				if h.negotiation != nil {
					if got, err := h.negotiation.MyNegotiations(user.ID); err == nil {
						negotiations = got
					}
				}

				data.Priorities = sellerPriorities(
					data,
					products,
					diagnoses,
					negotiations,
				)

				data.HealthMetrics,
					data.HealthScore,
					data.HealthStatus =
					farmHealth(data, diagnoses)
			}

			// Notifications are available to every user.
			if h.notification != nil {
				if count, err := h.notification.GetUnreadCount(user.ID); err == nil {
					data.UnreadNotifications = count
				}
			}

			if len(data.Priorities) > 0 {
				data.PrimaryPriority = data.Priorities[0]
			}
		}

		if err := render.RenderTemplates(
			w,
			"dashboard.html",
			data,
		); err != nil {
			if clientDisconnected(err) {
				log.Printf(
					"dashboard client disconnected while rendering: %v",
					err,
				)
				return
			}

			log.Println("render error", err)
			http.Error(
				w,
				"Internal Server Error",
				http.StatusInternalServerError,
			)
			return
		}

	case http.MethodPost:
		log.Println("user's Choices")
	}
}

// A client may close the browser connection while a large dashboard is being
// written. The response is no longer writable in that case, so attempting to
// send a second HTTP error response causes a misleading superfluous-header log.
func clientDisconnected(err error) bool {
	return errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) ||
		strings.Contains(strings.ToLower(err.Error()), "broken pipe") ||
		strings.Contains(strings.ToLower(err.Error()), "connection reset")
}

func cartSummary(items []models.CartItem) (int, float64) {
	total := 0.0

	for _, item := range items {
		total += item.TotalPrice
	}

	return len(items), total
}

func (h *Dashboard) MarketInsights(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.FarmerFromContext(r)

	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	cutoff := time.Now().Add(-30 * 24 * time.Hour)

	summaries, err := h.events.GroupedDemandSummaries(
		user.ID,
		user.Location,
		cutoff,
		cutoff.Add(-30*24*time.Hour),
	)

	if err != nil {
		log.Printf(
			"market insights unavailable for farmer %d: %v",
			user.ID,
			err,
		)

		http.Error(
			w,
			"Unable to load market insights",
			http.StatusInternalServerError,
		)
		return
	}

	data := struct {
		FullName  string
		Summaries []models.ProduceDemandSummary
	}{
		user.FullName,
		prioritizeDemandSummaries(summaries, len(summaries)),
	}

	if err := render.RenderTemplates(
		w,
		"market-insights.html",
		data,
	); err != nil {
		log.Printf("market insights render error: %v", err)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
	}
}

func prioritizeDemandSummaries(
	items []models.ProduceDemandSummary,
	limit int,
) []models.ProduceDemandSummary {
	if limit <= 0 || len(items) == 0 {
		return nil
	}

	result := append(
		[]models.ProduceDemandSummary(nil),
		items...,
	)

	sort.SliceStable(result, func(i, j int) bool {
		rank := func(s models.ProduceDemandSummary) int {
			if s.PriceDifferencePercent > 15 && s.Orders == 0 {
				return 5
			}

			if s.CartAdds > 0 && s.Orders == 0 {
				return 4
			}

			if s.Views+s.CartAdds+s.Orders+s.Negotiations == 0 {
				return 3
			}

			if s.Orders > 0 {
				return 1
			}

			return 2
		}

		return rank(result[i]) > rank(result[j])
	})

	if len(result) > limit {
		result = result[:limit]
	}

	return result
}

func limitOrders(orders []models.Order, max int) []models.Order {
	if len(orders) <= max {
		return orders
	}

	return orders[:max]
}

func buyerPriorities(data DashboardData) []DashboardPriority {
	if data.PurchasesMade == 0 {
		return []DashboardPriority{
			{
				Tone:   "info",
				Label:  "Get started",
				Title:  "Find your first produce listing",
				Detail: "Browse fresh produce from local farmers and place an order.",
				Action: "Browse marketplace",
				Href:   "/marketplace",
			},
		}
	}

	return []DashboardPriority{
		{
			Tone:   "success",
			Label:  "On track",
			Title:  "Your buying activity is up to date",
			Detail: "Review recent orders or discover something new in the marketplace.",
			Action: "View marketplace",
			Href:   "/marketplace",
		},
	}
}

func sellerPriorities(
	data DashboardData,
	products []models.Product,
	diagnoses []models.Diagnosis,
	negotiations []models.Negotiation,
) []DashboardPriority {
	var priorities []DashboardPriority

	for _, diagnosis := range diagnoses {
		text := strings.ToLower(
			diagnosis.Result + " " + diagnosis.Description,
		)

		if strings.Contains(text, "severe") ||
			strings.Contains(text, "urgent") ||
			strings.Contains(text, "critical") {

			priorities = append(
				priorities,
				DashboardPriority{
					Tone:   "danger",
					Label:  "Urgent",
					Title:  "Review a crop health alert",
					Detail: "Open your latest urgent AI diagnosis and follow its treatment guidance.",
					Action: "Review diagnosis",
					Href:   "/ai-diagnosis-history",
				},
			)

			break
		}
	}

	open := 0

	for _, negotiation := range negotiations {
		if negotiation.Status == "open" &&
			!negotiation.IsExpired() {
			open++
		}
	}

	if open > 0 {
		priorities = append(
			priorities,
			DashboardPriority{
				Tone:   "warning",
				Label:  "Response needed",
				Title:  "Reply to a buyer negotiation",
				Detail: "A buyer is waiting for your response.",
				Action: "View negotiations",
				Href:   "/negotiations",
			},
		)
	}

	aging := 0

	for _, product := range products {
		if product.Quantity > 0 &&
			product.Status != "active" &&
			product.Status != "sold_out" &&
			!product.CreatedAt.IsZero() &&
			time.Since(product.CreatedAt) >= 7*24*time.Hour {

			aging++
		}
	}

	if aging > 0 {
		priorities = append(
			priorities,
			DashboardPriority{
				Tone:   "warning",
				Label:  "Action needed",
				Title:  "List aging produce",
				Detail: "Some stored produce has been waiting for a buyer for more than a week.",
				Action: "Manage storage",
				Href:   "/storage",
			},
		)
	}

	if data.ProduceInStorage == 0 {
		priorities = append(
			priorities,
			DashboardPriority{
				Tone:   "warning",
				Label:  "Action needed",
				Title:  "Add produce to your storage",
				Detail: "Create a product record so you can track stock and list it for sale.",
				Action: "Open storage",
				Href:   "/storage",
			},
		)
	}

	if data.ProduceInStorage > 0 &&
		data.ListingsActive == 0 {

		priorities = append(
			priorities,
			DashboardPriority{
				Tone:   "warning",
				Label:  "Action needed",
				Title:  "Publish a listing",
				Detail: "Make your available produce visible to buyers in the marketplace.",
				Action: "Manage storage",
				Href:   "/storage",
			},
		)
	}

	if len(priorities) == 0 {
		priorities = append(
			priorities,
			DashboardPriority{
				Tone:   "success",
				Label:  "On track",
				Title:  "Your farm is ready for business",
				Detail: "Keep stock current and use the AI assistant whenever a crop needs attention.",
				Action: "View listings",
				Href:   "/marketplace",
			},
		)
	}

	sort.SliceStable(priorities, func(i, j int) bool {
		rank := map[string]int{
			"danger":  3,
			"warning": 2,
			"info":    1,
			"success": 0,
		}

		return rank[priorities[i].Tone] > rank[priorities[j].Tone]
	})

	if len(priorities) > 3 {
		priorities = priorities[:3]
	}

	return priorities
}

func farmHealth(
	data DashboardData,
	diagnoses []models.Diagnosis,
) ([]HealthMetric, int, string) {
	storage, listings, sales, health := 0, 0, 0, 100

	if data.ProduceInStorage > 0 {
		storage = 100

		listings =
			data.ListingsActive * 100 /
				data.ProduceInStorage

		if listings > 100 {
			listings = 100
		}
	}

	if data.Revenue > 0 {
		sales = 100
	}

	for _, diagnosis := range diagnoses {
		text := strings.ToLower(
			diagnosis.Result + " " + diagnosis.Description,
		)

		if strings.Contains(text, "severe") ||
			strings.Contains(text, "urgent") ||
			strings.Contains(text, "critical") {

			health = 40
			break
		}

		if health > 80 {
			health = 85
		}
	}

	score := (storage + listings + sales + health) / 4

	status := "Needs attention"

	if score >= 80 {
		status = "Healthy"
	} else if score >= 60 {
		status = "Fair"
	}

	return []HealthMetric{
		{"Storage", storage},
		{"Listings", listings},
		{"Produce health", health},
		{"Sales activity", sales},
	}, score, status
}
