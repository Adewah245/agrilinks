package routes

import (
	"backend/handlers"
	app "backend/internal"
	"backend/middleware"
	"net/http"
)

func RegisterRoutes(container *app.Container) {
	http.Handle(
		"/static/", http.StripPrefix("/static/",
			http.FileServer(http.Dir("../frontend")),
		),
	)

	http.HandleFunc("/", middleware.OnlyPath("/", middleware.OnlyGet(handlers.IndexHandler)))

	registerHandler := handlers.NewRegisterHandler(container.Auth)
	http.HandleFunc("/register", middleware.OnlyPath("/register", registerHandler.RegisterHandler))

	// Login handles its own GET/POST switch internally, so it isn't
	// wrapped in OnlyGet (that was blocking POST from ever reaching it).
	loginHandler := handlers.NewLoginHandler(container.Auth)
	http.HandleFunc("/login", middleware.OnlyPath("/login", loginHandler.LoginHandler))

	http.HandleFunc("/logout", middleware.OnlyPath("/logout", middleware.OnlyGet(handlers.LogoutHandler)))

	forgotPasswordHandler := handlers.NewForgotPasswordHandler(container.Auth)
	http.HandleFunc("/forgot-password", middleware.OnlyPath("/forgot-password", forgotPasswordHandler.Handler))

	// Everything below is protected: RequireAuth loads the logged-in
	// farmer/buyer and attaches it to the request before the handler runs.

	dashboardHandler := handlers.NewDashboardHandler(container.Crop, container.Order, container.AI)
	http.HandleFunc(
		"/dashboard",
		middleware.OnlyPath("/dashboard", middleware.OnlyGet(middleware.RequireAuth(container.FarmerRepo, dashboardHandler.DashBoard))),
	)

	profileHandler := handlers.NewProfileHandler(container.Crop, container.Order)
	http.HandleFunc(
		"/profile",
		middleware.OnlyPath("/profile", middleware.OnlyGet(middleware.RequireAuth(container.FarmerRepo, profileHandler.ProfileHandler))),
	)

	profileEditHandler := handlers.NewProfileEditHandler(container.Auth)
	http.HandleFunc(
		"/profile/edit",
		middleware.OnlyPath("/profile/edit", middleware.RequireAuth(container.FarmerRepo, profileEditHandler.Handler)),
	)

	// Storage handles its own GET/POST switch (farmers register crops here).
	storageHandler := handlers.NewStorageHandler(container.Crop)
	http.HandleFunc(
		"/storage",
		middleware.OnlyPath("/storage", middleware.RequireAuth(container.FarmerRepo, storageHandler.StorageHandler)),
	)

	// Marketplace handles its own GET/POST switch (buyers place orders here).
	marketplaceHandler := handlers.NewMarketplaceHandler(container.Crop, container.Order)
	http.HandleFunc(
		"/marketplace",
		middleware.OnlyPath("/marketplace", middleware.RequireAuth(container.FarmerRepo, marketplaceHandler.MarketplaceHandler)),
	)

	// AI Assistant handles its own GET/POST switch (image upload).
	aiHandler := handlers.NewAIAssistantHandler(container.AI)
	http.HandleFunc(
		"/ai-assistant",
		middleware.OnlyPath("/ai-assistant", middleware.RequireAuth(container.FarmerRepo, aiHandler.Handler)),
	)

	// Negotiations: list, thread (chat + offer/accept/reject), and starting
	// a new one from the Marketplace page.
	negotiationHandler := handlers.NewNegotiationHandler(container.Negotiation)
	http.HandleFunc(
		"/negotiations",
		middleware.OnlyPath("/negotiations", middleware.OnlyGet(middleware.RequireAuth(container.FarmerRepo, negotiationHandler.ListHandler))),
	)
	http.HandleFunc(
		"/negotiation",
		middleware.OnlyPath("/negotiation", middleware.RequireAuth(container.FarmerRepo, negotiationHandler.ThreadHandler)),
	)
	http.HandleFunc(
		"/negotiation/start",
		middleware.OnlyPath("/negotiation/start", middleware.RequireAuth(container.FarmerRepo, negotiationHandler.StartHandler)),
	)
}
