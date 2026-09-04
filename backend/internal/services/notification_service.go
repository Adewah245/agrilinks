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
// - Provides operations for creating and managing
//   notifications.
//
// The service does NOT communicate directly with PostgreSQL.
// The repository handles database communication.
// ============================================================

// NotificationService handles business logic related to
// user notifications.
type NotificationService struct {
	repo *repository.NotificationRepository
}

// ============================================================
// CREATE NOTIFICATION SERVICE
// ============================================================

// NewNotificationService creates a new NotificationService
// using the provided notification repository.
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
// ============================================================

// CreateNotification creates a new unread notification for
// the specified user.
func (s *NotificationService) CreateNotification(
	userID int,
	title string,
	message string,
	notificationType string,
) error {
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
		UserID:  userID,
		Title:   title,
		Message: message,
		Type:    notificationType,
		IsRead:  false,
	}

	return s.repo.Create(notification)
}

// ============================================================
// GET USER NOTIFICATIONS
//
// Retrieves all notifications belonging to a user.
// ============================================================

// GetUserNotifications returns all notifications belonging
// to the specified user.
func (s *NotificationService) GetUserNotifications(
	userID int,
) ([]models.Notification, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user ID")
	}

	return s.repo.GetUserNotifications(userID)
}

// ============================================================
// GET UNREAD COUNT
//
// Returns the number of unread notifications for a user.
//
// This will be used for the notification badge.
// Example:
//
// 🔔 3
// ============================================================

// GetUnreadCount returns the number of unread notifications
// belonging to the specified user.
func (s *NotificationService) GetUnreadCount(
	userID int,
) (int, error) {
	if userID <= 0 {
		return 0, errors.New("invalid user ID")
	}

	return s.repo.GetUnreadCount(userID)
}

// ============================================================
// MARK ONE NOTIFICATION AS READ
//
// Marks a specific notification as read.
// ============================================================

// MarkAsRead marks a specific notification as read, provided
// that it belongs to the specified user.
func (s *NotificationService) MarkAsRead(
	notificationID int,
	userID int,
) error {
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
//
// Marks every notification belonging to the user as read.
// ============================================================

// MarkAllAsRead marks every notification belonging to the
// specified user as read.
func (s *NotificationService) MarkAllAsRead(
	userID int,
) error {
	if userID <= 0 {
		return errors.New("invalid user ID")
	}

	return s.repo.MarkAllAsRead(userID)
}
