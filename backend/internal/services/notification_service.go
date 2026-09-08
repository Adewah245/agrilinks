package services

import (
	"errors"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"
)

// ============================================================
// NOTIFICATION SERVICE
//
// Responsibility:
// - Contains notification business logic.
// - Acts as the middle layer between handlers/other services
//   and the notification repository.
// - Validates notification input.
// - Provides operations for creating and managing
//   notifications.
//
// The service does NOT communicate directly with PostgreSQL.
// The repository handles database communication.
// ============================================================

type NotificationService struct {
	repo *repository.NotificationRepository
}

// ============================================================
// CREATE SERVICE
// ============================================================

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	repo *repository.NotificationRepository,
) *NotificationService {
	return &NotificationService{
		repo: repo,
	}
}

// ============================================================
// CREATE NOTIFICATION
//
// Creates a new unread notification for a user.
//
// Current notification architecture:
//
//	UserID
//	NotificationType
//	Title
//	Message
//	ReferenceType
//	ReferenceID
//	IsRead
//	CreatedAt
//
// This method intentionally keeps the existing four-argument
// API so existing services do not need to change.
//
// Notifications that need a reference can use
// CreateNotificationWithReference below.
// ============================================================

func (s *NotificationService) CreateNotification(
	userID int,
	title string,
	message string,
	notificationType string,
) error {

	if s == nil || s.repo == nil {
		return errors.New("notification service is not initialized")
	}

	if userID <= 0 {
		return errors.New("invalid user ID")
	}

	title = strings.TrimSpace(title)
	message = strings.TrimSpace(message)
	notificationType = strings.TrimSpace(notificationType)

	if title == "" {
		return errors.New("notification title is required")
	}

	if message == "" {
		return errors.New("notification message is required")
	}

	if notificationType == "" {
		return errors.New("notification type is required")
	}

	notification := &models.Notification{
		UserID:           userID,
		NotificationType: notificationType,
		Title:            title,
		Message:          message,
		IsRead:           false,
	}

	return s.repo.Create(notification)
}

// ============================================================
// CREATE NOTIFICATION WITH REFERENCE
//
// Creates a notification connected to another Agro-Shield
// resource.
//
// Examples:
//
//	order
//	product
//	negotiation
//	message
//	training
//
// ReferenceType and ReferenceID allow the frontend to know
// what resource the notification belongs to.
// ============================================================

func (s *NotificationService) CreateNotificationWithReference(
	userID int,
	title string,
	message string,
	notificationType string,
	referenceType *string,
	referenceID *int,
) error {

	if s == nil || s.repo == nil {
		return errors.New("notification service is not initialized")
	}

	if userID <= 0 {
		return errors.New("invalid user ID")
	}

	title = strings.TrimSpace(title)
	message = strings.TrimSpace(message)
	notificationType = strings.TrimSpace(notificationType)

	if title == "" {
		return errors.New("notification title is required")
	}

	if message == "" {
		return errors.New("notification message is required")
	}

	if notificationType == "" {
		return errors.New("notification type is required")
	}

	if referenceType != nil {
		value := strings.TrimSpace(*referenceType)

		if value == "" {
			referenceType = nil
		} else {
			referenceType = &value
		}
	}

	if referenceID != nil && *referenceID <= 0 {
		return errors.New("invalid reference ID")
	}

	notification := &models.Notification{
		UserID:           userID,
		NotificationType: notificationType,
		Title:            title,
		Message:          message,
		ReferenceType:    referenceType,
		ReferenceID:      referenceID,
		IsRead:           false,
	}

	return s.repo.Create(notification)
}

// ============================================================
// GET USER NOTIFICATIONS
// ============================================================

// GetUserNotifications returns all notifications belonging
// to the specified user.
func (s *NotificationService) GetUserNotifications(
	userID int,
) ([]models.Notification, error) {

	if s == nil || s.repo == nil {
		return nil, errors.New("notification service is not initialized")
	}

	if userID <= 0 {
		return nil, errors.New("invalid user ID")
	}

	return s.repo.GetUserNotifications(userID)
}

// ============================================================
// GET UNREAD COUNT
//
// Used by the notification badge.
//
// Example:
//
//	🔔 3
// ============================================================

func (s *NotificationService) GetUnreadCount(
	userID int,
) (int, error) {

	if s == nil || s.repo == nil {
		return 0, errors.New("notification service is not initialized")
	}

	if userID <= 0 {
		return 0, errors.New("invalid user ID")
	}

	return s.repo.GetUnreadCount(userID)
}

// ============================================================
// MARK ONE NOTIFICATION AS READ
//
// The repository also verifies that the notification belongs
// to the supplied user.
// ============================================================

func (s *NotificationService) MarkAsRead(
	notificationID int,
	userID int,
) error {

	if s == nil || s.repo == nil {
		return errors.New("notification service is not initialized")
	}

	if notificationID <= 0 {
		return errors.New("invalid notification ID")
	}

	if userID <= 0 {
		return errors.New("invalid user ID")
	}

	return s.repo.MarkAsRead(
		notificationID,
		userID,
	)
}

// ============================================================
// MARK ALL NOTIFICATIONS AS READ
// ============================================================

func (s *NotificationService) MarkAllAsRead(
	userID int,
) error {

	if s == nil || s.repo == nil {
		return errors.New("notification service is not initialized")
	}

	if userID <= 0 {
		return errors.New("invalid user ID")
	}

	return s.repo.MarkAllAsRead(userID)
}