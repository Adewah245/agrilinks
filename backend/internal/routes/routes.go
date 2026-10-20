package routes

import (
	"net/http"
	"os"

	"backend/internal/app"
	"backend/internal/handlers"
	"backend/internal/middleware"
	"backend/internal/services"
	"backend/internal/ussd"
)

func RegisterRoutes(container *app.Container) {

	// ========================================================
	// USSD
	// ========================================================

	// USSD gateways post menu selections to this
	// provider-neutral endpoint.
	ussdHandler := ussd.NewHandler(
		container.UserRepo,
		container.Product,
		services.NewWeatherService(),
		container.Order,
		container.Notification,
	)

	http.Handle("/ussd", ussdHandler)

	// ========================================================
	// STATIC FILES
	// ========================================================

	frontendDir := "../frontend"
	if _, err := os.Stat(frontendDir); err != nil {
		for _, candidate := range []string{"frontend", "../../../frontend"} {
			if _, candidateErr := os.Stat(candidate); candidateErr == nil {
				frontendDir = candidate
				break
			}
		}
	}

	http.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir(frontendDir)),
		),
	)

	// ========================================================
	// LOCATION API
	// ========================================================

	locationHandler := handlers.NewLocationHandler(
		container.LocationRepo,
	)

	http.HandleFunc(
		"/api/states",
		locationHandler.GetStatesHandler,
	)

	http.HandleFunc(
		"/api/states/",
		locationHandler.GetLGAsHandler,
	)

	// ========================================================
	// MARKET PRICE API
	// ========================================================

	priceHandler := handlers.NewPriceHandler(
		container.Price,
	)

	http.HandleFunc(
		"POST /api/v1/prices/report",
		priceHandler.HandleSubmitPrice,
	)

	http.HandleFunc(
		"GET /api/v1/prices/current",
		priceHandler.HandleGetCurrentPrice,
	)

	// ========================================================
	// HOME
	// ========================================================

	http.HandleFunc(
		"/",
		middleware.OnlyPath(
			"/",
			middleware.OnlyGet(
				handlers.IndexHandler,
			),
		),
	)

	// ========================================================
	// AUTHENTICATION
	// ========================================================

	registerHandler := handlers.NewRegisterHandler(
		container.Auth,
	)

	http.HandleFunc(
		"/register",
		middleware.OnlyPath(
			"/register",
			registerHandler.RegisterHandler,
		),
	)

	loginHandler := handlers.NewLoginHandler(
		container.Auth,
	)

	http.HandleFunc(
		"/login",
		middleware.OnlyPath(
			"/login",
			loginHandler.LoginHandler,
		),
	)

	http.HandleFunc(
		"/logout",
		middleware.OnlyPath(
			"/logout",
			middleware.OnlyGet(
				handlers.LogoutHandler,
			),
		),
	)

	forgotPasswordHandler := handlers.NewForgotPasswordHandler(
		container.Auth,
	)

	http.HandleFunc(
		"/forgot-password",
		middleware.OnlyPath(
			"/forgot-password",
			forgotPasswordHandler.Handler,
		),
	)

	// ========================================================
	// DASHBOARD
	// ========================================================

	dashboardHandler := handlers.NewDashboardHandler(
		container.Product,
		container.Order,
		container.Cart,
		container.AI,
		container.Negotiation,
		services.NewWeatherService(),
		container.Notification,
		container.MarketEvents,
	)

	http.HandleFunc(
		"/dashboard",
		middleware.OnlyPath(
			"/dashboard",
			middleware.OnlyGet(
				middleware.RequireAuth(
					container.UserRepo,
					dashboardHandler.DashBoard,
				),
			),
		),
	)

	http.HandleFunc(
		"/market-insights",
		middleware.OnlyPath(
			"/market-insights",
			middleware.OnlyGet(
				middleware.RequireAuth(
					container.UserRepo,
					dashboardHandler.MarketInsights,
				),
			),
		),
	)

	// ========================================================
	// LEARNING
	// ========================================================
	//
	// Learning is available to EVERY authenticated user.
	//
	// Users whose primary intent is learning use the learner role.
	//
	// Examples:
	//
	// Buyer       → can learn
	// Farmer      → can learn
	// Learner     → goes directly to learning
	// Organisation → can learn
	// Ministry    → can learn
	//
	// Learning journey:
	//
	// /learn
	//     ↓
	// Choose farming type
	//     ↓
	// /learn/start
	//     ↓
	// Learning Journey
	//     ↓
	// Lessons / Checkpoints / Messages / Progress
	//
	// ========================================================

	learningHandler := handlers.NewLearningHandler(
		container.Learning,
	)

	// --------------------------------------------------------
	// LEARNING HOME
	//
	// GET /learn
	//
	// Shows:
	//
	// - Available farming types
	// - User's farming interests
	// - Current learning journey
	// --------------------------------------------------------

	http.HandleFunc(
		"/learn",
		middleware.OnlyPath(
			"/learn",
			middleware.OnlyGet(
				middleware.RequireAuth(
					container.UserRepo,
					learningHandler.LearningHome,
				),
			),
		),
	)

	// --------------------------------------------------------
	// START LEARNING JOURNEY
	//
	// POST /learn/start
	//
	// Form:
	//
	// farming_type_id
	// experience_level
	// goal
	//
	// Example:
	//
	// User chooses Fish Farming
	//        ↓
	// POST /learn/start
	//        ↓
	// Learning Journey created
	// --------------------------------------------------------

	http.HandleFunc(
		"/learn/start",
		middleware.OnlyPath(
			"/learn/start",
			middleware.RequireAuth(
				container.UserRepo,
				learningHandler.StartLearning,
			),
		),
	)

	// --------------------------------------------------------
	// LEARNING JOURNEY
	//
	// GET /learn/journey?id=123
	//
	// Displays:
	//
	// - Farming type
	// - Current stage
	// - Progress
	// - Readiness score
	// - Checkpoints
	// - Learning messages
	// --------------------------------------------------------

	http.HandleFunc(
		"/learn/journey",
		middleware.OnlyPath(
			"/learn/journey",
			middleware.OnlyGet(
				middleware.RequireAuth(
					container.UserRepo,
					learningHandler.LearningJourney,
				),
			),
		),
	)

	// --------------------------------------------------------
	// UPDATE LEARNING JOURNEY
	//
	// POST /learn/journey/update
	//
	// Updates:
	//
	// - Current stage
	// - Status
	// - Readiness score
	//
	// --------------------------------------------------------

	http.HandleFunc(
		"/learn/journey/update",
		middleware.OnlyPath(
			"/learn/journey/update",
			middleware.RequireAuth(
				container.UserRepo,
				learningHandler.UpdateJourneyProgress,
			),
		),
	)

	// --------------------------------------------------------
	// SEND LEARNING MESSAGE
	//
	// POST /learn/message
	//
	// Used for:
	//
	// - User questions
	// - Learning discussions
	// - AI learning interaction later
	//
	// --------------------------------------------------------

	http.HandleFunc(
		"/learn/message",
		middleware.OnlyPath(
			"/learn/message",
			middleware.RequireAuth(
				container.UserRepo,
				learningHandler.SendMessage,
			),
		),
	)

	// --------------------------------------------------------
	// UPDATE LEARNING CHECKPOINT
	//
	// POST /learn/checkpoint/update
	//
	// Used to update:
	//
	// - Progress
	// - Status
	// - Notes
	//
	// --------------------------------------------------------

	http.HandleFunc(
		"/learn/checkpoint/update",
		middleware.OnlyPath(
			"/learn/checkpoint/update",
			middleware.RequireAuth(
				container.UserRepo,
				learningHandler.UpdateCheckpoint,
			),
		),
	)

	// ========================================================
	// PROFILE
	// ========================================================

	profileHandler := handlers.NewProfileHandler(
		container.Product,
		container.Order,
		container.UserRepo,
	)

	http.HandleFunc(
		"/profile",
		middleware.OnlyPath(
			"/profile",
			middleware.OnlyGet(
				middleware.RequireAuth(
					container.UserRepo,
					profileHandler.ProfileHandler,
				),
			),
		),
	)

	profileEditHandler := handlers.NewProfileEditHandler(
		container.Auth,
	)

	http.HandleFunc(
		"/profile/edit",
		middleware.OnlyPath(
			"/profile/edit",
			middleware.RequireAuth(
				container.UserRepo,
				profileEditHandler.Handler,
			),
		),
	)

	// ========================================================
	// STORAGE
	// ========================================================

	storageHandler := handlers.NewStorageHandler(
		container.Product,
	)

	salesHandler := handlers.NewSalesHandler(
		container.Order,
	)

	http.HandleFunc(
		"/sales",
		middleware.OnlyPath(
			"/sales",
			middleware.RequireAuth(
				container.UserRepo,
				salesHandler.Handler,
			),
		),
	)

	http.HandleFunc(
		"/storage",
		middleware.OnlyPath(
			"/storage",
			middleware.RequireAuth(
				container.UserRepo,
				storageHandler.StorageHandler,
			),
		),
	)

	// ========================================================
	// MARKETPLACE
	// ========================================================

	marketplaceHandler := handlers.NewMarketplaceHandler(
		container.Product,
		container.Order,
		container.MarketEvents,
	)

	http.HandleFunc(
		"/marketplace",
		middleware.OnlyPath(
			"/marketplace",
			middleware.RequireAuth(
				container.UserRepo,
				marketplaceHandler.MarketplaceHandler,
			),
		),
	)

	// ========================================================
	// PRODUCT DETAILS
	// ========================================================

	productHandler := handlers.NewProductHandler(
		container.Product,
		container.MarketEvents,
	)

	http.HandleFunc(
		"/product",
		middleware.OnlyPath(
			"/product",
			middleware.RequireAuth(
				container.UserRepo,
				productHandler.ProductDetailsHandler,
			),
		),
	)

	// ========================================================
	// CART
	// ========================================================

	cartHandler := handlers.NewCartHandler(
		container.Cart,
		container.MarketEvents,
	)

	http.HandleFunc(
		"/cart",
		middleware.OnlyPath(
			"/cart",
			middleware.RequireAuth(
				container.UserRepo,
				cartHandler.Handler,
			),
		),
	)

	// ========================================================
	// NOTIFICATIONS
	// ========================================================

	notificationHandler := handlers.NewNotificationHandler(
		container.Notification,
	)

	// --------------------------------------------------------
	// VIEW NOTIFICATIONS
	//
	// GET /notifications
	// --------------------------------------------------------

	http.HandleFunc(
		"/notifications",
		middleware.OnlyPath(
			"/notifications",
			middleware.RequireAuth(
				container.UserRepo,
				notificationHandler.NotificationsHandler,
			),
		),
	)

	// --------------------------------------------------------
	// MARK ONE NOTIFICATION AS READ
	//
	// POST /notifications/read?id=123
	// --------------------------------------------------------

	http.HandleFunc(
		"/notifications/read",
		middleware.OnlyPath(
			"/notifications/read",
			middleware.RequireAuth(
				container.UserRepo,
				notificationHandler.MarkAsReadHandler,
			),
		),
	)

	// --------------------------------------------------------
	// MARK ALL NOTIFICATIONS AS READ
	//
	// POST /notifications/read-all
	// --------------------------------------------------------

	http.HandleFunc(
		"/notifications/read-all",
		middleware.OnlyPath(
			"/notifications/read-all",
			middleware.RequireAuth(
				container.UserRepo,
				notificationHandler.MarkAllAsReadHandler,
			),
		),
	)

	// ========================================================
	// PAYMENTS
	// ========================================================

	paymentHandler := handlers.NewPaymentHandler(container.Payment)
	http.HandleFunc("/payments", middleware.OnlyPath("/payments", middleware.RequireAuth(container.UserRepo, paymentHandler.Handler)))
	deliveryHandler := handlers.NewDeliveryHandler(container.Delivery)
	http.HandleFunc("/deliveries", middleware.OnlyPath("/deliveries", middleware.OnlyGet(middleware.RequireAuth(container.UserRepo, deliveryHandler.Handler))))

	// ========================================================
	// AI ASSISTANT
	// ========================================================

	aiHandler := handlers.NewAIAssistantHandler(
		container.AI,
	)

	diagnosisHistoryHandler := handlers.NewAIDiagnosisHistoryHandler(
		container.AI,
	)

	http.HandleFunc(
		"/ai-assistant",
		middleware.OnlyPath(
			"/ai-assistant",
			middleware.RequireAuth(
				container.UserRepo,
				aiHandler.Handler,
			),
		),
	)

	http.HandleFunc(
		"/ai-diagnosis-history",
		middleware.OnlyPath(
			"/ai-diagnosis-history",
			middleware.RequireAuth(
				container.UserRepo,
				diagnosisHistoryHandler.Handler,
			),
		),
	)

	// ========================================================
	// NEGOTIATIONS
	// ========================================================

	negotiationHandler := handlers.NewNegotiationHandler(
		container.Negotiation,
		container.MarketEvents,
	)

	// --------------------------------------------------------
	// NEGOTIATION LIST
	//
	// GET /negotiations
	// --------------------------------------------------------

	http.HandleFunc(
		"/negotiations",
		middleware.OnlyPath(
			"/negotiations",
			middleware.OnlyGet(
				middleware.RequireAuth(
					container.UserRepo,
					negotiationHandler.ListHandler,
				),
			),
		),
	)

	// --------------------------------------------------------
	// NEGOTIATION CONVERSATION
	//
	// GET /negotiation?id=123
	// --------------------------------------------------------

	http.HandleFunc(
		"/negotiation",
		middleware.OnlyPath(
			"/negotiation",
			middleware.RequireAuth(
				container.UserRepo,
				negotiationHandler.ThreadHandler,
			),
		),
	)

	// --------------------------------------------------------
	// START NEGOTIATION
	//
	// POST /negotiation/start
	// --------------------------------------------------------

	http.HandleFunc(
		"/negotiation/start",
		middleware.OnlyPath(
			"/negotiation/start",
			middleware.RequireAuth(
				container.UserRepo,
				negotiationHandler.StartHandler,
			),
		),
	)

	// ========================================================
	// CHAT
	// ========================================================
	//
	// General Agro-Shield communication.
	//
	// Chat is different from negotiation.
	//
	// Negotiation:
	//
	// Buyer ↔ Seller
	// Price negotiation
	// Offers
	// Accept / Reject
	//
	// Chat:
	//
	// User ↔ User
	// User ↔ Group
	// Normal communication
	//
	// ========================================================

	chatHandler := handlers.NewChatHandler(
		container.Chat,
		container.UserRepo,
	)

	// --------------------------------------------------------
	// CHAT HOME
	//
	// GET /chat
	// --------------------------------------------------------

	http.HandleFunc(
		"/chat",
		middleware.OnlyPath(
			"/chat",
			middleware.OnlyGet(
				middleware.RequireAuth(
					container.UserRepo,
					chatHandler.ListHandler,
				),
			),
		),
	)

	// --------------------------------------------------------
	// VIEW CHAT
	//
	// GET /chat/view?id=123
	// --------------------------------------------------------

	http.HandleFunc(
		"/chat/view",
		middleware.OnlyPath(
			"/chat/view",
			middleware.OnlyGet(
				middleware.RequireAuth(
					container.UserRepo,
					chatHandler.ViewHandler,
				),
			),
		),
	)

	// --------------------------------------------------------
	// CHAT PRESENCE
	//
	// GET /chat/presence
	// --------------------------------------------------------

	http.HandleFunc(
		"/chat/presence",
		middleware.OnlyPath(
			"/chat/presence",
			middleware.OnlyGet(
				middleware.RequireAuth(
					container.UserRepo,
					chatHandler.PresenceHandler,
				),
			),
		),
	)

	// --------------------------------------------------------
	// SEND MESSAGE
	//
	// POST /chat/send
	// --------------------------------------------------------

	http.HandleFunc(
		"/chat/send",
		middleware.OnlyPath(
			"/chat/send",
			middleware.RequireAuth(
				container.UserRepo,
				chatHandler.SendMessageHandler,
			),
		),
	)

	// --------------------------------------------------------
	// CREATE GROUP
	//
	// POST /chat/group/create
	// --------------------------------------------------------

	http.HandleFunc(
		"/chat/group/create",
		middleware.OnlyPath(
			"/chat/group/create",
			middleware.RequireAuth(
				container.UserRepo,
				chatHandler.CreateGroupHandler,
			),
		),
	)

	// --------------------------------------------------------
	// CREATE PRIVATE CHAT
	//
	// POST /chat/private
	// --------------------------------------------------------

	http.HandleFunc(
		"/chat/private",
		middleware.OnlyPath(
			"/chat/private",
			middleware.RequireAuth(
				container.UserRepo,
				chatHandler.CreatePrivateChatHandler,
			),
		),
	)

	// --------------------------------------------------------
	// ADD MEMBER
	//
	// POST /chat/member/add
	// --------------------------------------------------------

	http.HandleFunc(
		"/chat/member/add",
		middleware.OnlyPath(
			"/chat/member/add",
			middleware.RequireAuth(
				container.UserRepo,
				chatHandler.AddMemberHandler,
			),
		),
	)

	// --------------------------------------------------------
	// REMOVE MEMBER
	//
	// POST /chat/member/remove
	// --------------------------------------------------------

	http.HandleFunc(
		"/chat/member/remove",
		middleware.OnlyPath(
			"/chat/member/remove",
			middleware.RequireAuth(
				container.UserRepo,
				chatHandler.RemoveMemberHandler,
			),
		),
	)

	// --------------------------------------------------------
	// LEAVE GROUP
	//
	// POST /chat/leave
	// --------------------------------------------------------

	http.HandleFunc(
		"/chat/leave",
		middleware.OnlyPath(
			"/chat/leave",
			middleware.RequireAuth(
				container.UserRepo,
				chatHandler.LeaveGroupHandler,
			),
		),
	)

	// --------------------------------------------------------
	// DELETE MESSAGE
	//
	// POST /chat/message/delete
	// --------------------------------------------------------

	http.HandleFunc(
		"/chat/message/delete",
		middleware.OnlyPath(
			"/chat/message/delete",
			middleware.RequireAuth(
				container.UserRepo,
				chatHandler.DeleteMessageHandler,
			),
		),
	)
}
