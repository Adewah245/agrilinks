package services

import (
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

type NotificationService struct {
	repo *repository.NotificationRepository
}

// ============================================================
// CREATE NOTIFICATION SERVICE
// ============================================================

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

func (s *NotificationService) CreateNotification(
	userID int,
	title string,
	message string,
	notificationType string,
) error {

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

func (s *NotificationService) GetUserNotifications(
	userID int,
) ([]models.Notification, error) {

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

func (s *NotificationService) GetUnreadCount(
	userID int,
) (int, error) {

	return s.repo.GetUnreadCount(userID)
}

// ============================================================
// MARK ONE NOTIFICATION AS READ
//
// Marks a specific notification as read.
// ============================================================

func (s *NotificationService) MarkAsRead(
	notificationID int,
	userID int,
) error {

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

func (s *NotificationService) MarkAllAsRead(
	userID int,
) error {

	return s.repo.MarkAllAsRead(userID)
}
