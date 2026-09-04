package main

import (
	"log"
	"net/http"
	"os"

	app "backend/internal"
	"backend/internal/database"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/routes"
)

func main() {

	// ============================================================
	// 1. CONNECT TO DATABASE
	//
	// Connects Agro-Shield to PostgreSQL.
	//
	// Flow:
	//
	// Agro-Shield
	//      ↓
	// ConnectDB()
	//      ↓
	// PostgreSQL
	// ============================================================

	db, err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	// ============================================================
	// 2. RUN DATABASE MIGRATIONS
	//
	// Creates the database structure required by Agro-Shield.
	//
	// Migration files are responsible for defining:
	//
	// - Users
	// - User verification
	// - Locations
	// - Farming
	// - Products
	// - Marketplace
	// - Orders
	// - Negotiations
	// - Cart
	// - Chat
	// - Organisations
	// - Events and news
	// - Logistics and delivery
	// - Training
	// - Notifications
	// - Payments
	// - Reviews
	//
	// Flow:
	//
	// main.go
	//     ↓
	// RunMigration()
	//     ↓
	// Migration files
	//     ↓
	// PostgreSQL
	// ============================================================

	if err := database.RunMigration(db); err != nil {
		log.Fatal(err)
	}

	// ============================================================
	// 3. CREATE REPOSITORIES
	//
	// Repositories communicate directly with PostgreSQL.
	//
	// Service
	//    ↓
	// Repository
	//    ↓
	// PostgreSQL
	// ============================================================

	// ------------------------------------------------------------
	// USER REPOSITORY
	//
	// Handles user account database operations.
	//
	// Users may have different roles such as:
	//
	// - Farmer
	// - Buyer
	// - Ministry
	// - Organisation
	// ------------------------------------------------------------

	userRepo := repository.NewUserRepository(db)

	// ------------------------------------------------------------
	// LOCATION REPOSITORY
	//
	// Handles:
	//
	// - States
	// - LGAs
	// - Communities
	// ------------------------------------------------------------

	locationRepo := repository.NewLocationRepository(db)

	// ------------------------------------------------------------
	// PRODUCT REPOSITORY
	//
	// Handles agricultural product listings.
	//
	// Product ownership is represented by:
	//
	// owner_type
	// owner_id
	//
	// Products can therefore belong to:
	//
	// User
	// Organisation
	// ------------------------------------------------------------

	productRepo := repository.NewProductRepository(db)

	// ------------------------------------------------------------
	// ORDER REPOSITORY
	//
	// Handles marketplace orders.
	// ------------------------------------------------------------

	orderRepo := repository.NewOrderRepository(db)

	// ------------------------------------------------------------
	// DIAGNOSIS REPOSITORY
	//
	// Handles AI diagnosis history.
	// ------------------------------------------------------------

	diagnosisRepo := repository.NewDiagnosisRepository(db)

	// ------------------------------------------------------------
	// NEGOTIATION REPOSITORY
	//
	// Handles negotiations between buyers and sellers.
	// ------------------------------------------------------------

	negotiationRepo := repository.NewNegotiationRepository(db)

	// ------------------------------------------------------------
	// NEGOTIATION MESSAGE REPOSITORY
	//
	// Handles messages and offers inside negotiations.
	// ------------------------------------------------------------

	negotiationMsgRepo :=
		repository.NewNegotiationMessageRepository(db)

	// ------------------------------------------------------------
	// CART REPOSITORY
	//
	// Handles products added to the buyer's cart.
	//
	// Example:
	//
	// Negotiation accepted
	//        ↓
	// Product added to cart
	//        ↓
	// Buyer sees product in cart
	// ------------------------------------------------------------

	cartRepo := repository.NewCartRepository(db)

	// ------------------------------------------------------------
	// NOTIFICATION REPOSITORY
	//
	// Handles notifications stored in PostgreSQL.
	//
	// Examples:
	//
	// - New negotiation offer
	// - Offer accepted
	// - Offer rejected
	// - Order created
	// ------------------------------------------------------------

	notificationRepo :=
		repository.NewNotificationRepository(db)

	marketEventRepo :=
		repository.NewMarketEventRepository(db)

	// ============================================================
	// CHAT REPOSITORIES
	//
	// Agro-Shield chat has three database areas:
	//
	// 1. Conversations
	// 2. Conversation members
	// 3. Chat messages
	//
	// Flow:
	//
	// ChatService
	//      ↓
	// Chat Repositories
	//      ↓
	// PostgreSQL
	// ============================================================

	// ------------------------------------------------------------
	// CONVERSATION REPOSITORY
	//
	// Handles chat conversations.
	// ------------------------------------------------------------

	conversationRepo :=
		repository.NewConversationRepository(db)

	// ------------------------------------------------------------
	// CONVERSATION MEMBER REPOSITORY
	//
	// Handles users who belong to conversations.
	// ------------------------------------------------------------

	conversationMemberRepo :=
		repository.NewConversationMemberRepository(db)

	// ------------------------------------------------------------
	// CHAT MESSAGE REPOSITORY
	//
	// Handles normal chat messages.
	// ------------------------------------------------------------

	chatMessageRepo :=
		repository.NewChatMessageRepository(db)

	// ============================================================
	// 4. CREATE AI PROVIDER
	//
	// GeminiProvider handles communication between Agro-Shield
	// and the Gemini AI service.
	//
	// API key:
	//
	// GEMINI_API_KEY
	// ============================================================

	aiProvider, err := services.NewGeminiProvider(
		os.Getenv("GEMINI_API_KEY"),
	)

	if err != nil {
		log.Fatal(err)
	}

	// ============================================================
	// 5. CREATE APPLICATION CONTAINER
	//
	// Dependency flow:
	//
	// PostgreSQL
	//     ↓
	// Repositories
	//     ↓
	// Services
	//     ↓
	// Container
	//     ↓
	// Routes / Handlers
	// ============================================================

	container := app.NewContainer(
		userRepo,
		locationRepo,
		productRepo,
		orderRepo,
		cartRepo,
		diagnosisRepo,
		negotiationRepo,
		negotiationMsgRepo,
		notificationRepo,
		marketEventRepo,

		// --------------------------------------------------------
		// CHAT REPOSITORIES
		// --------------------------------------------------------

		conversationRepo,
		conversationMemberRepo,
		chatMessageRepo,

		// --------------------------------------------------------
		// AI PROVIDER
		// --------------------------------------------------------

		aiProvider,
	)

	// ============================================================
	// 6. REGISTER ROUTES
	//
	// Connects URLs to their handlers.
	// ============================================================

	routes.RegisterRoutes(container)

	// ============================================================
	// 7. START HTTP SERVER
	//
	// Starts Agro-Shield on port 8080 by default.
	// ============================================================

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server starting on port:", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Println(err)
	}
}
