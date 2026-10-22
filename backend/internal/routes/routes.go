package routes

import (
	"net/http"
	"os"

	"backend/internal/app"
	"backend/internal/handlers"
	"backend/internal/middleware"
	"backend/internal/ussd"
)

func NewHandler(container *app.Container) http.Handler {
	mux := http.NewServeMux()

	// ========================================================
	// USSD
	// ========================================================

	// USSD gateways post menu selections to this
	// provider-neutral endpoint.
	ussdHandler := ussd.NewHandler(
		container.UserRepo,
		container.Product,
		container.Weather,
		container.Order,
		container.Notification,
	)

	mux.Handle("/ussd", ussdHandler)

	// ========================================================
	// STATIC FILES
	// ========================================================

	frontendDir := container.Config.FrontendDir
	if frontendDir == "" {
		frontendDir = "../frontend"
	}
	if _, err := os.Stat(frontendDir); err != nil {
		for _, candidate := range []string{"frontend", "../../../frontend"} {
			if _, candidateErr := os.Stat(candidate); candidateErr == nil {
				frontendDir = candidate
				break
			}
		}
	}

	mux.Handle(
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

	mux.HandleFunc(
		"/api/states",
		locationHandler.GetStatesHandler,
	)

	mux.HandleFunc(
		"/api/states/",
		locationHandler.GetLGAsHandler,
	)

	// ========================================================
	// MARKET PRICE API
	// ========================================================

	priceHandler := handlers.NewPriceHandler(
		container.Price,
	)

	mux.HandleFunc(
		"POST /api/v1/prices/report",
		middleware.RequireAuth(container.Sessions, container.UserRepo, priceHandler.HandleSubmitPrice),
	)

	mux.HandleFunc(
		"GET /api/v1/prices/current",
		priceHandler.HandleGetCurrentPrice,
	)

	// ========================================================
	// HOME
	// ========================================================

	mux.HandleFunc(
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
		container.Learning,
	)

	mux.HandleFunc(
		"/register",
		middleware.OnlyPath(
			"/register",
			registerHandler.RegisterHandler,
		),
	)

	loginHandler := handlers.NewLoginHandler(
		container.Auth, container.Sessions,
	)

	mux.HandleFunc(
		"/login",
		middleware.OnlyPath(
			"/login",
			loginHandler.LoginHandler,
		),
	)

	mux.HandleFunc(
		"/logout",
		middleware.OnlyPath(
			"/logout",
			middleware.OnlyGet(
				handlers.NewLogoutHandler(container.Sessions),
			),
		),
	)

	forgotPasswordHandler := handlers.NewForgotPasswordHandler(
		container.Auth,
	)

	mux.HandleFunc(
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
		container.Weather,
		container.Notification,
		container.MarketEvents,
	)

	mux.HandleFunc(
		"/dashboard",
		middleware.OnlyPath(
			"/dashboard",
			middleware.OnlyGet(
				middleware.RequireAuth(container.Sessions,
					container.UserRepo,
					dashboardHandler.DashBoard,
				),
			),
		),
	)

	mux.HandleFunc(
		"/market-insights",
		middleware.OnlyPath(
			"/market-insights",
			middleware.OnlyGet(
				middleware.RequireAuth(container.Sessions,
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
	mux.HandleFunc("/learn/checkpoint/create", middleware.OnlyPath("/learn/checkpoint/create", middleware.RequireAuth(container.Sessions, container.UserRepo, learningHandler.CreateCheckpoint)))
	mux.HandleFunc("GET /learn/rooms", middleware.RequireAuth(container.Sessions, container.UserRepo, learningHandler.Rooms))
	mux.HandleFunc("POST /learn/rooms/create", middleware.RequireAuth(container.Sessions, container.UserRepo, learningHandler.CreateRoom))
	mux.HandleFunc("POST /learn/rooms/join", middleware.RequireAuth(container.Sessions, container.UserRepo, learningHandler.JoinRoom))

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

	mux.HandleFunc(
		"/learn",
		middleware.OnlyPath(
			"/learn",
			middleware.OnlyGet(
				middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/learn/start",
		middleware.OnlyPath(
			"/learn/start",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/learn/journey",
		middleware.OnlyPath(
			"/learn/journey",
			middleware.OnlyGet(
				middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/learn/journey/update",
		middleware.OnlyPath(
			"/learn/journey/update",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/learn/message",
		middleware.OnlyPath(
			"/learn/message",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/learn/checkpoint/update",
		middleware.OnlyPath(
			"/learn/checkpoint/update",
			middleware.RequireAuth(container.Sessions,
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
		container.Review,
		container.UserRepo,
	)

	mux.HandleFunc(
		"/profile",
		middleware.OnlyPath(
			"/profile",
			middleware.OnlyGet(
				middleware.RequireAuth(container.Sessions,
					container.UserRepo,
					profileHandler.ProfileHandler,
				),
			),
		),
	)

	profileEditHandler := handlers.NewProfileEditHandler(
		container.Auth,
	)

	mux.HandleFunc(
		"/profile/edit",
		middleware.OnlyPath(
			"/profile/edit",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/sales",
		middleware.OnlyPath(
			"/sales",
			middleware.RequireAuth(container.Sessions,
				container.UserRepo,
				salesHandler.Handler,
			),
		),
	)

	mux.HandleFunc(
		"/storage",
		middleware.OnlyPath(
			"/storage",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/marketplace",
		middleware.OnlyPath(
			"/marketplace",
			middleware.RequireAuth(container.Sessions,
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
		container.Review,
		container.MarketEvents,
		container.Notification,
	)

	mux.HandleFunc(
		"/product",
		middleware.OnlyPath(
			"/product",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/cart",
		middleware.OnlyPath(
			"/cart",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/notifications",
		middleware.OnlyPath(
			"/notifications",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/notifications/read",
		middleware.OnlyPath(
			"/notifications/read",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/notifications/read-all",
		middleware.OnlyPath(
			"/notifications/read-all",
			middleware.RequireAuth(container.Sessions,
				container.UserRepo,
				notificationHandler.MarkAllAsReadHandler,
			),
		),
	)

	// ========================================================
	// PAYMENTS
	// ========================================================

	orderHandler := handlers.NewOrderHandler(container.Order)
	mux.HandleFunc(
		"/orders",
		middleware.OnlyPath(
			"/orders",
			middleware.OnlyGet(
				middleware.RequireAuth(container.Sessions,
					container.UserRepo,
					orderHandler.ListHandler,
				),
			),
		),
	)
	mux.HandleFunc(
		"/orders/place",
		middleware.OnlyPath(
			"/orders/place",
			middleware.RequireAuth(container.Sessions,
				container.UserRepo,
				orderHandler.PlaceHandler,
			),
		),
	)

	reviewHandler := handlers.NewReviewHandler(container.Review)
	mux.HandleFunc(
		"/reviews",
		middleware.OnlyPath(
			"/reviews",
			middleware.RequireAuth(container.Sessions,
				container.UserRepo,
				reviewHandler.CreateHandler,
			),
		),
	)
	mux.HandleFunc(
		"/trust",
		middleware.OnlyPath(
			"/trust",
			middleware.RequireAuth(container.Sessions,
				container.UserRepo,
				reviewHandler.TrustHandler,
			),
		),
	)

	paymentHandler := handlers.NewPaymentHandler(container.Payment)
	mux.HandleFunc(
		"/payments",
		middleware.OnlyPath(
			"/payments",
			middleware.RequireAuth(container.Sessions,
				container.UserRepo,
				paymentHandler.Handler,
			),
		),
	)

	// The provider redirect and webhook are deliberately unauthenticated:
	// Flutterwave cannot hold an Agro-Shield session. Both are authenticated
	// by provider verification, and the webhook additionally uses verif-hash.
	mux.HandleFunc(
		"/payments/callback",
		middleware.OnlyPath(
			"/payments/callback",
			paymentHandler.CallbackHandler,
		),
	)
	mux.HandleFunc(
		"/webhooks/flutterwave",
		middleware.OnlyPath(
			"/webhooks/flutterwave",
			paymentHandler.WebhookHandler,
		),
	)

	deliveryHandler := handlers.NewDeliveryHandler(container.Delivery)
	mux.HandleFunc("/deliveries", middleware.OnlyPath("/deliveries", middleware.OnlyGet(middleware.RequireAuth(container.Sessions, container.UserRepo, deliveryHandler.Handler))))
	mux.HandleFunc("/deliveries/request", middleware.OnlyPath("/deliveries/request", middleware.RequireAuth(container.Sessions, container.UserRepo, deliveryHandler.RequestDeliveryHandler)))
	mux.HandleFunc("/deliveries/assign", middleware.OnlyPath("/deliveries/assign", middleware.RequireAuth(container.Sessions, container.UserRepo, deliveryHandler.AssignPartnerHandler)))
	mux.HandleFunc("/deliveries/status", middleware.OnlyPath("/deliveries/status", middleware.RequireAuth(container.Sessions, container.UserRepo, deliveryHandler.UpdateStatusHandler)))
	mux.HandleFunc("/deliveries/tracking", middleware.OnlyPath("/deliveries/tracking", middleware.RequireAuth(container.Sessions, container.UserRepo, deliveryHandler.TrackingHandler)))
	mux.HandleFunc("/delivery-partner", middleware.OnlyPath("/delivery-partner", middleware.RequireAuth(container.Sessions, container.UserRepo, deliveryHandler.PartnerPageHandler)))
	mux.HandleFunc("/delivery-partner/review", middleware.OnlyPath("/delivery-partner/review", middleware.RequireAuth(container.Sessions, container.UserRepo, deliveryHandler.PartnerReviewHandler)))

	// ========================================================
	// AI ASSISTANT
	// ========================================================

	aiHandler := handlers.NewAIAssistantHandler(
		container.AI,
	)

	diagnosisHistoryHandler := handlers.NewAIDiagnosisHistoryHandler(
		container.AI,
	)

	mux.HandleFunc(
		"/ai-assistant",
		middleware.OnlyPath(
			"/ai-assistant",
			middleware.RequireAuth(container.Sessions,
				container.UserRepo,
				aiHandler.Handler,
			),
		),
	)

	mux.HandleFunc(
		"/ai-diagnosis-history",
		middleware.OnlyPath(
			"/ai-diagnosis-history",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/negotiations",
		middleware.OnlyPath(
			"/negotiations",
			middleware.OnlyGet(
				middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/negotiation",
		middleware.OnlyPath(
			"/negotiation",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/negotiation/start",
		middleware.OnlyPath(
			"/negotiation/start",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/chat",
		middleware.OnlyPath(
			"/chat",
			middleware.OnlyGet(
				middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/chat/view",
		middleware.OnlyPath(
			"/chat/view",
			middleware.OnlyGet(
				middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/chat/presence",
		middleware.OnlyPath(
			"/chat/presence",
			middleware.OnlyGet(
				middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/chat/send",
		middleware.OnlyPath(
			"/chat/send",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/chat/group/create",
		middleware.OnlyPath(
			"/chat/group/create",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/chat/private",
		middleware.OnlyPath(
			"/chat/private",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/chat/member/add",
		middleware.OnlyPath(
			"/chat/member/add",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/chat/member/remove",
		middleware.OnlyPath(
			"/chat/member/remove",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/chat/leave",
		middleware.OnlyPath(
			"/chat/leave",
			middleware.RequireAuth(container.Sessions,
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

	mux.HandleFunc(
		"/chat/message/delete",
		middleware.OnlyPath(
			"/chat/message/delete",
			middleware.RequireAuth(container.Sessions,
				container.UserRepo,
				chatHandler.DeleteMessageHandler,
			),
		),
	)
	walletHandler := handlers.NewWalletHandler(container.Wallet)
	mux.HandleFunc("/wallet", middleware.OnlyPath("/wallet", middleware.RequireAuth(container.Sessions, container.UserRepo, walletHandler.Handler)))
	mux.HandleFunc("GET /wallet/banks", middleware.RequireAuth(container.Sessions, container.UserRepo, walletHandler.Banks))
	mux.HandleFunc("/wallet/admin", middleware.OnlyPath("/wallet/admin", middleware.RequireAuth(container.Sessions, container.UserRepo, walletHandler.Admin)))
	return middleware.RequestGuard(mux, container.Config)
}
