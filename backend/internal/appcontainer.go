package app

import (
	"backend/internal/repository"
	"backend/internal/services"
)

// ============================================================
// APPLICATION CONTAINER
//
// Container holds the main services used throughout Agro-Shield.
//
// Dependency flow:
//
//	Repository
//	    ↓
//	Service
//	    ↓
//	Container
//	    ↓
//	Handler
//	    ↓
//	Route
// ============================================================

type Container struct {
	Auth         *services.AuthService
	Product      *services.ProductService
	Order        *services.OrderService
	Cart         *services.CartService
	AI           *services.AIService
	Negotiation  *services.NegotiationService
	Notification *services.NotificationService
	Chat         *services.ChatService
	MarketEvents *repository.MarketEventRepository

	// Used by authentication middleware.
	UserRepo     *repository.UserRepository
	LocationRepo *repository.LocationRepository
}

// ============================================================
// CREATE APPLICATION CONTAINER
// ============================================================

func NewContainer(
	userRepo *repository.UserRepository,
	locationRepo *repository.LocationRepository,
	productRepo *repository.ProductRepository,
	orderRepo *repository.OrderRepository,
	cartRepo *repository.CartRepository,
	diagnosisRepo *repository.DiagnosisRepository,
	negotiationRepo *repository.NegotiationRepository,
	negotiationMsgRepo *repository.NegotiationMessageRepository,
	notificationRepo *repository.NotificationRepository,
	marketEventRepo *repository.MarketEventRepository,

	// ========================================================
	// CHAT REPOSITORIES
	// ========================================================

	conversationRepo *repository.ConversationRepository,
	memberRepo *repository.ConversationMemberRepository,
	chatMessageRepo *repository.ChatMessageRepository,

	aiProvider services.AIProvider,
) *Container {

	// ========================================================
	// 1. CART SERVICE
	// ========================================================

	cartService := services.NewCartService(
		cartRepo,
		productRepo,
		marketEventRepo,
	)

	// ========================================================
	// 2. NOTIFICATION SERVICE
	// ========================================================

	notificationService := services.NewNotificationService(
		notificationRepo,
	)

	// ========================================================
	// 3. NEGOTIATION SERVICE
	// ========================================================

	negotiationService := services.NewNegotiationService(
		negotiationRepo,
		negotiationMsgRepo,
		productRepo,
		cartService,
		notificationService,
		marketEventRepo,
	)

	// ========================================================
	// 4. CHAT SERVICE
	//
	// Chat is completely separate from negotiation.
	//
	// Negotiation:
	//
	// Buyer ↔ Seller
	//     ↓
	// Offers
	//     ↓
	// Accept/Reject
	//
	// Normal Chat:
	//
	// User ↔ User
	// Group
	//     ↓
	// Unlimited normal conversation
	//
	// The negotiation expiration time does NOT affect ChatService.
	// ========================================================

	chatService := services.NewChatService(
		conversationRepo,
		memberRepo,
		chatMessageRepo,
		notificationService,
	)

	// ========================================================
	// 5. RETURN APPLICATION CONTAINER
	// ========================================================

	return &Container{

		// ----------------------------------------------------
		// AUTH
		// ----------------------------------------------------

		Auth: services.NewAuthService(
			userRepo,
		),

		// ----------------------------------------------------
		// PRODUCTS
		// ----------------------------------------------------

		Product: services.NewProductService(
			productRepo,
		),

		// ----------------------------------------------------
		// ORDERS
		// ----------------------------------------------------

		Order: services.NewOrderService(
			orderRepo,
			productRepo,
		),

		// ----------------------------------------------------
		// CART
		// ----------------------------------------------------

		Cart: cartService,

		// ----------------------------------------------------
		// AI
		// ----------------------------------------------------

		AI: services.NewAIService(
			diagnosisRepo,
			aiProvider,
		),

		// ----------------------------------------------------
		// NEGOTIATION
		// ----------------------------------------------------

		Negotiation: negotiationService,

		// ----------------------------------------------------
		// NOTIFICATION
		// ----------------------------------------------------

		Notification: notificationService,

		// ----------------------------------------------------
		// CHAT
		// ----------------------------------------------------

		Chat:         chatService,
		MarketEvents: marketEventRepo,

		// ----------------------------------------------------
		// USER REPOSITORY
		// ----------------------------------------------------

		UserRepo:     userRepo,
		LocationRepo: locationRepo,
	}
}
