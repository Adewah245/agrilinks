package app

import (
	"backend/internal/config"
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
//      Repository
//          ↓
//      Service
//          ↓
//      Container
//          ↓
//      Handler
//          ↓
//      Route
// ============================================================

type Container struct {
	Admin    *repository.AdminRepository
	Wallet   *services.WalletService
	Config   config.Config
	Sessions *services.SessionService
	OTP      *services.OTPService
	Weather  *services.WeatherService
	// --------------------------------------------------------
	// APPLICATION SERVICES
	// --------------------------------------------------------

	Auth         *services.AuthService
	Product      *services.ProductService
	Order        *services.OrderService
	Cart         *services.CartService
	AI           *services.AIService
	Negotiation  *services.NegotiationService
	Notification *services.NotificationService
	Chat         *services.ChatService
	Learning     *services.LearningService
	Payment      *services.PaymentService
	Delivery     *services.DeliveryService
	Price        *services.PriceService
	Review       *services.ReviewService

	// --------------------------------------------------------
	// REPOSITORIES USED DIRECTLY BY THE APPLICATION
	// --------------------------------------------------------

	MarketEvents *repository.MarketEventRepository
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
	learningRepo *repository.LearningRepository,
	paymentRepo *repository.PaymentRepository,
	deliveryRepo *repository.DeliveryRepository,
	deliveryPartnerRepo *repository.DeliveryPartnerRepository,
	reviewRepo *repository.ReviewRepository,

	// ========================================================
	// CHAT REPOSITORIES
	// ========================================================

	conversationRepo *repository.ConversationRepository,
	memberRepo *repository.ConversationMemberRepository,
	chatMessageRepo *repository.ChatMessageRepository,

	aiProvider services.AIProvider,
	priceService *services.PriceService,
	negotiationPolicy config.NegotiationPolicy,
	paymentGateway services.PaymentGateway,
	paymentWebhookHash string,
	paymentCurrency string,
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
		negotiationPolicy.MaxRounds,
		negotiationPolicy.Window,
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
	// Accept / Reject
	//
	// Normal Chat:
	//
	// User ↔ User
	// Group
	//     ↓
	// Normal conversation
	//
	// Negotiation expiration does NOT affect ChatService.
	// ========================================================

	chatService := services.NewChatService(
		conversationRepo,
		memberRepo,
		chatMessageRepo,
		notificationService,
	)

	// ========================================================
	// 5. LEARNING SERVICE
	//
	// Learning is available to every authenticated user.
	//
	// Repository:
	//     LearningRepository
	//            ↓
	// Service:
	//     LearningService
	// ========================================================

	learningService := services.NewLearningService(
		learningRepo,
	)

	// ========================================================
	// 6. RETURN APPLICATION CONTAINER
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
			cartRepo,
			paymentCurrency,
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

		Chat: chatService,

		// ----------------------------------------------------
		// LEARNING
		// ----------------------------------------------------

		Learning: learningService,
		Payment: services.NewPaymentService(
			paymentRepo,
			orderRepo,
			userRepo,
			paymentGateway,
			notificationService,
			paymentWebhookHash,
			paymentCurrency,
		),
		Delivery: services.NewDeliveryService(deliveryRepo, deliveryPartnerRepo),
		Price:    priceService,
		Review:   services.NewReviewService(reviewRepo),

		// ----------------------------------------------------
		// MARKET EVENTS
		// ----------------------------------------------------

		MarketEvents: marketEventRepo,

		// ----------------------------------------------------
		// USER REPOSITORY
		// ----------------------------------------------------

		UserRepo: userRepo,

		// ----------------------------------------------------
		// LOCATION REPOSITORY
		// ----------------------------------------------------

		LocationRepo: locationRepo,
	}
}
