package handlers

import (
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"
	"syscall"
	"time"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/repository"
	"backend/internal/services"
)

// Dashboard handles the Agro-Shield user dashboard.
//
// The dashboard provides different information depending on the
// authenticated user's role. Farmers and other sellers see storage,
// listings, sales, AI diagnosis, and negotiation information, while
// buyers see purchases, cart activity, spending, and marketplace
// information.
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

// NewDashboardHandler creates a new Dashboard handler with the
// services and repositories required by the dashboard.
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

// DashboardData contains all information rendered by dashboard.html.
//
// The structure contains both seller-oriented and buyer-oriented
// fields. The template decides which sections to display based on
// the authenticated user's role.
type DashboardData struct {
	FullName string
	Role     string
	IsBuyer  bool
	PhotoURL string

	// FarmingTypes contains the authenticated user's selected farming
	// activities. It is used by the seller dashboard to personalise the
	// farming activity sections.
	FarmingTypes []string

	// Seller statistics.
	ProduceInStorage     int
	StorageCrops         []models.Product
	ListingsActive       int
	ListingsWithPhotos   int
	AIDiagnosesThisMonth int
	Revenue              float64
	Revenue30            float64
	CompletedSales       int

	// Buyer statistics.
	PurchasesMade    int
	TotalSpent       float64
	AvailableProduce int
	CartItemCount    int
	CartTotal        float64
	HealthScore      int
	HealthStatus     string
	HealthMetrics    []HealthMetric

	UnreadNotifications int

	Weather         models.Weather
	LatestDiagnosis *models.Diagnosis

	MarketListings       int
	MarketPriceMin       float64
	MarketPriceMedian    float64
	MarketPriceMax       float64
	MarketHasPrices      bool
	MarketOrders30       int
	MarketNegotiations30 int

	Demand               models.ProduceDemandSummary
	ProduceDemands       []models.ProduceDemandSummary
	ProduceDemandPreview []models.ProduceDemandSummary

	Priorities      []DashboardPriority
	PrimaryPriority DashboardPriority
	RecentOrders    []models.Order
}

// HealthMetric represents one component of the seller dashboard
// health score.
type HealthMetric struct {
	Name    string
	Percent int
}

// DashboardPriority represents an action that should receive the
// user's attention on the dashboard.
type DashboardPriority struct {
	Tone   string
	Label  string
	Title  string
	Detail string
	Action string
	Href   string
}

// DashBoard handles GET requests for the Agro-Shield dashboard.
//
// The dashboard loads user-specific marketplace, storage, order,
// negotiation, AI, notification, weather, and market insight data.
// POST requests are currently reserved for future dashboard actions.
func (h *Dashboard) DashBoard(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		log.Println("User Visited Dashboard")

		data := DashboardData{
			FullName: "user",
		}

		if h.events != nil {
			if err := h.events.ReconcileCheckoutAbandonments(24 * time.Hour); err != nil {
				log.Printf(
					"dashboard checkout reconciliation failed: %v",
					err,
				)
			}
		}

		user, ok := middleware.UserFromContext(r)

		if ok && user != nil {
			if user.IsLearner() {
				http.Redirect(
					w,
					r,
					"/learn",
					http.StatusSeeOther,
				)
				return
			}

			data.FullName = user.FullName
			data.Role = user.Role
			data.IsBuyer = user.IsBuyer()
			data.PhotoURL = user.PhotoURL
			data.FarmingTypes = user.FarmingTypes

			// -------------------------------------------------
			// WEATHER
			// -------------------------------------------------
			if h.weather != nil {
				var err error

				data.Weather, err = h.weather.CurrentForLocation(
					r.Context(),
					user.Location,
					user.LGA,
					user.State,
					user.Country,
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
						data.CartItemCount, data.CartTotal =
							cartSummary(items)
					}
				}

				if h.product != nil {
					if available, err := h.product.AvailableProducts(); err == nil {
						data.AvailableProduce = len(available)
					} else {
						log.Printf(
							"dashboard available products unavailable for buyer %d: %v",
							user.ID,
							err,
						)
					}
				}

				if h.order != nil {
					if purchases, err := h.order.MyPurchases(user.ID); err == nil {
						data.PurchasesMade = len(purchases)
						data.RecentOrders = limitOrders(purchases, 5)

						for _, purchase := range purchases {
							data.TotalSpent += purchase.TotalPrice
						}
					} else {
						log.Printf(
							"dashboard purchases unavailable for buyer %d: %v",
							user.ID,
							err,
						)
					}
				}

				data.Priorities = buyerPriorities(data)

			} else {
				// -------------------------------------------------
				// SELLER DASHBOARD
				// -------------------------------------------------

				var userProducts []models.Product

				if h.product != nil {
					if got, err := h.product.MyProducts(
						"user",
						user.ID,
					); err == nil {
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
					} else {
						log.Printf(
							"dashboard products unavailable for user %d: %v",
							user.ID,
							err,
						)
					}
				}

				// -------------------------------------------------
				// MARKET DEMAND
				// -------------------------------------------------
				if h.events != nil && len(userProducts) > 0 {
					cutoff := time.Now().Add(-30 * 24 * time.Hour)
					previous := cutoff.Add(-30 * 24 * time.Hour)

					if summary, err := h.events.UserDemandSummary(
						user.ID,
						user.Location,
						cutoff,
						previous,
					); err != nil {
						log.Printf(
							"dashboard demand summary unavailable for user %d: %v",
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
							"dashboard product comparisons unavailable for user %d: %v",
							user.ID,
							err,
						)
					} else {
						data.ProduceDemands = summaries
						data.ProduceDemandPreview =
							prioritizeDemandSummaries(
								summaries,
								3,
							)
					}
				}

				// -------------------------------------------------
				// MARKETPLACE STATISTICS
				// -------------------------------------------------
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
								data.MarketPriceMedian =
									prices[middle]
							}

							data.MarketHasPrices = true
						}
					} else {
						log.Printf(
							"dashboard marketplace statistics unavailable for user %d: %v",
							user.ID,
							err,
						)
					}
				}

				// -------------------------------------------------
				// AI DIAGNOSIS STATISTICS
				// -------------------------------------------------
				if h.ai != nil {
					if count, err := h.ai.CountThisMonth(
						user.ID,
					); err == nil {
						data.AIDiagnosesThisMonth = count
					} else {
						log.Printf(
							"dashboard AI diagnosis count unavailable for user %d: %v",
							user.ID,
							err,
						)
					}
				}

				// -------------------------------------------------
				// SALES STATISTICS
				// -------------------------------------------------
				if h.order != nil {
					if sales, err := h.order.MySales(
						"user",
						user.ID,
					); err == nil {
						data.RecentOrders =
							limitOrders(sales, 5)

						cutoff := time.Now().Add(
							-30 * 24 * time.Hour,
						)

						for _, sale := range sales {
							data.Revenue += sale.TotalPrice

							if !sale.CreatedAt.Before(cutoff) {
								data.Revenue30 += sale.TotalPrice
								data.CompletedSales++
								data.MarketOrders30++
							}
						}
					} else {
						log.Printf(
							"dashboard sales unavailable for user %d: %v",
							user.ID,
							err,
						)
					}
				}

				// -------------------------------------------------
				// NEGOTIATION STATISTICS
				// -------------------------------------------------
				if h.negotiation != nil {
					if negotiations, err := h.negotiation.MyNegotiations(
						user.ID,
					); err == nil {
						cutoff := time.Now().Add(
							-30 * 24 * time.Hour,
						)

						for _, negotiation := range negotiations {
							if !negotiation.CreatedAt.Before(cutoff) {
								data.MarketNegotiations30++
							}
						}
					} else {
						log.Printf(
							"dashboard negotiations unavailable for user %d: %v",
							user.ID,
							err,
						)
					}
				}

				// -------------------------------------------------
				// AI DIAGNOSIS HISTORY
				// -------------------------------------------------
				var diagnoses []models.Diagnosis

				if h.ai != nil {
					if got, err := h.ai.History(
						user.ID,
					); err == nil {
						diagnoses = got

						if len(diagnoses) > 0 {
							latest := diagnoses[0]
							data.LatestDiagnosis = &latest
						}
					} else {
						log.Printf(
							"dashboard diagnosis history unavailable for user %d: %v",
							user.ID,
							err,
						)
					}
				}

				// -------------------------------------------------
				// NEGOTIATIONS FOR PRIORITIES
				// -------------------------------------------------
				var negotiations []models.Negotiation

				if h.negotiation != nil {
					if got, err := h.negotiation.MyNegotiations(
						user.ID,
					); err == nil {
						negotiations = got
					} else {
						log.Printf(
							"dashboard negotiation priorities unavailable for user %d: %v",
							user.ID,
							err,
						)
					}
				}

				data.Priorities = sellerPriorities(
					data,
					userProducts,
					diagnoses,
					negotiations,
				)

				data.HealthMetrics,
					data.HealthScore,
					data.HealthStatus =
					farmHealth(data, diagnoses)
			}

			// -------------------------------------------------
			// NOTIFICATIONS
			// -------------------------------------------------
			if h.notification != nil {
				if count, err := h.notification.GetUnreadCount(
					user.ID,
				); err == nil {
					data.UnreadNotifications = count
				} else {
					log.Printf(
						"dashboard notification count unavailable for user %d: %v",
						user.ID,
						err,
					)
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
		log.Println("User dashboard POST request received")

		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)

	default:
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
	}
}

// clientDisconnected reports whether an HTTP response failed because
// the client closed the connection while the server was writing.
func clientDisconnected(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) ||
		strings.Contains(
			strings.ToLower(err.Error()),
			"broken pipe",
		) ||
		strings.Contains(
			strings.ToLower(err.Error()),
			"connection reset",
		)
}

// cartSummary calculates the number of items and total value of
// the products currently in a user's cart.
func cartSummary(items []models.CartItem) (int, float64) {
	total := 0.0

	for _, item := range items {
		total += item.TotalPrice
	}

	return len(items), total
}

// MarketInsights renders market demand and pricing insights for
// the authenticated user.
func (h *Dashboard) MarketInsights(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	if h.events == nil {
		http.Error(
			w,
			"Market insights are unavailable",
			http.StatusInternalServerError,
		)
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
			"market insights unavailable for user %d: %v",
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
		FullName: user.FullName,
		Summaries: prioritizeDemandSummaries(
			summaries,
			len(summaries),
		),
	}

	if err := render.RenderTemplates(
		w,
		"market-insights.html",
		data,
	); err != nil {
		if clientDisconnected(err) {
			log.Printf(
				"market insights client disconnected while rendering: %v",
				err,
			)
			return
		}

		log.Printf(
			"market insights render error: %v",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
	}
}

// prioritizeDemandSummaries sorts demand summaries so that potentially
// actionable market opportunities appear first.
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

	sort.SliceStable(
		result,
		func(i, j int) bool {
			rank := func(
				s models.ProduceDemandSummary,
			) int {
				if s.PriceDifferencePercent > 15 &&
					s.Orders == 0 {
					return 5
				}

				if s.CartAdds > 0 &&
					s.Orders == 0 {
					return 4
				}

				if s.Views+
					s.CartAdds+
					s.Orders+
					s.Negotiations == 0 {
					return 3
				}

				if s.Orders > 0 {
					return 1
				}

				return 2
			}

			return rank(result[i]) > rank(result[j])
		},
	)

	if len(result) > limit {
		result = result[:limit]
	}

	return result
}

// limitOrders limits the number of orders returned for dashboard
// display without modifying the original order slice.
func limitOrders(
	orders []models.Order,
	max int,
) []models.Order {
	if max <= 0 {
		return nil
	}

	if len(orders) <= max {
		return orders
	}

	return orders[:max]
}

// buyerPriorities creates dashboard priorities for buyers based on
// their purchasing activity.
func buyerPriorities(
	data DashboardData,
) []DashboardPriority {
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

// sellerPriorities creates dashboard priorities for users who sell
// agricultural products.
func sellerPriorities(
	data DashboardData,
	products []models.Product,
	diagnoses []models.Diagnosis,
	negotiations []models.Negotiation,
) []DashboardPriority {
	var priorities []DashboardPriority

	for _, diagnosis := range diagnoses {
		text := strings.ToLower(diagnosisString(diagnosis.Result) + " " + diagnosisString(diagnosis.Symptoms))

		if strings.Contains(text, "severe") ||
			strings.Contains(text, "urgent") ||
			strings.Contains(text, "critical") {

			priorities = append(
				priorities,
				DashboardPriority{
					Tone:   "danger",
					Label:  "Urgent",
					Title:  "Review an agricultural health alert",
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
				Title:  "Add products to your storage",
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
				Detail: "Make your available products visible to buyers in the marketplace.",
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
				Detail: "Keep stock current and use the AI assistant whenever an agricultural activity needs attention.",
				Action: "View listings",
				Href:   "/marketplace",
			},
		)
	}

	sort.SliceStable(
		priorities,
		func(i, j int) bool {
			rank := map[string]int{
				"danger":  3,
				"warning": 2,
				"info":    1,
				"success": 0,
			}

			return rank[priorities[i].Tone] >
				rank[priorities[j].Tone]
		},
	)

	if len(priorities) > 3 {
		priorities = priorities[:3]
	}

	return priorities
}

// farmHealth calculates a simple seller dashboard health score
// from storage, listing, sales, and agricultural health activity.
func farmHealth(
	data DashboardData,
	diagnoses []models.Diagnosis,
) ([]HealthMetric, int, string) {
	storage, listings, sales, health := 0, 0, 0, 100

	if data.ProduceInStorage > 0 {
		storage = 100

		listings =
			data.ListingsActive *
				100 /
				data.ProduceInStorage

		if listings > 100 {
			listings = 100
		}
	}

	if data.Revenue > 0 {
		sales = 100
	}

	for _, diagnosis := range diagnoses {
		text := strings.ToLower(diagnosisString(diagnosis.Result) + " " + diagnosisString(diagnosis.Symptoms))

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

	score := (storage +
		listings +
		sales +
		health) / 4

	status := "Needs attention"

	if score >= 80 {
		status = "Healthy"
	} else if score >= 60 {
		status = "Fair"
	}

	return []HealthMetric{
		{
			Name:    "Storage",
			Percent: storage,
		},
		{
			Name:    "Listings",
			Percent: listings,
		},
		{
			Name:    "Produce health",
			Percent: health,
		},
		{
			Name:    "Sales activity",
			Percent: sales,
		},
	}, score, status
}

func diagnosisString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
