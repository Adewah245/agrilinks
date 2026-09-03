package routes

import (
	"backend/handlers"
	app "backend/internal"
	"backend/internal/services"
	"backend/middleware"
	"backend/ussd"
	"net/http"
)

func RegisterRoutes(container *app.Container) {
	// USSD gateways post menu selections to this provider-neutral endpoint.
	ussdHandler := ussd.NewHandler(container.UserRepo, container.Crop, services.NewWeatherService(), container.Order, container.Notification)
	http.Handle("/ussd", ussdHandler)

	// ========================================================
	// STATIC FILES
	// ========================================================

	http.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir("../frontend")),
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
		container.Crop,
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
			middleware.OnlyGet(middleware.RequireAuth(container.UserRepo, dashboardHandler.MarketInsights)),
		),
	)

	// ========================================================
	// PROFILE
	// ========================================================

	profileHandler := handlers.NewProfileHandler(
		container.Crop,
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
		container.Crop,
	)
	salesHandler := handlers.NewSalesHandler(container.Order)
	http.HandleFunc("/sales", middleware.OnlyPath("/sales", middleware.RequireAuth(container.UserRepo, salesHandler.Handler)))

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
		container.Crop,
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
		container.Crop,
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

	// View notifications
	//
	// GET /notifications

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

	// Mark one notification as read
	//
	// POST /notifications/read?id=123

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

	// Mark all notifications as read
	//
	// POST /notifications/read-all

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

	// Negotiation list
	//
	// GET /negotiations

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

	// Negotiation conversation
	//
	// GET /negotiation?id=123
	//
	// The handler also handles POST actions.

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

	// Start negotiation
	//
	// POST /negotiation/start

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
	// Examples:
	//
	// 🌽 Maize Farmers Association
	// 🛠️ Farm Tools Sellers
	// 👨🏽‍🌾 Benue Farmers
	// 👤 Private conversation with another user
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
	//
	// Shows all conversations belonging to the user.
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
	//
	// Displays:
	//
	// - Conversation
	// - Messages
	// - Members
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
	//
	// Example:
	//
	// Maize Farmers Association
	//
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
	//
	// Form:
	//
	// user_id
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
