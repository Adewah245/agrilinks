package repository

import (
	"backend/internal/models"
	"database/sql"
)

// ============================================================
// NOTIFICATION REPOSITORY
//
// Responsibility:
// - Communicates directly with the notifications table.
// - Creates notifications.
// - Retrieves notifications.
// - Counts unread notifications.
// - Marks notifications as read.
// - Marks all notifications as read.
//
// The repository only handles database operations.
// Business decisions belong to the NotificationService.
// ============================================================

// NotificationRepository handles database operations
// for user notifications.
type NotificationRepository struct {
	db *sql.DB
}

// ============================================================
// CREATE REPOSITORY
// ============================================================

// NewNotificationRepository creates a new NotificationRepository
// using the provided database connection.
func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{
		db: db,
	}
}

// ============================================================
// CREATE NOTIFICATION
//
// Saves a new notification in PostgreSQL.
// ============================================================

// Create stores a new notification for a user.
func (r *NotificationRepository) Create(
	notification *models.Notification,
) error {

	query := `
		INSERT INTO notifications (
			user_id,
			title,
			message,
			type,
			is_read
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	return r.db.QueryRow(
		query,
		notification.UserID,
		notification.Title,
		notification.Message,
		notification.Type,
		notification.IsRead,
	).Scan(&notification.ID)
}

// ============================================================
// GET USER NOTIFICATIONS
//
// Retrieves all notifications belonging to one user.
//
// Newest notifications are returned first.
// ============================================================

// GetUserNotifications retrieves all notifications belonging
// to the specified user, with the newest notifications first.
func (r *NotificationRepository) GetUserNotifications(
	userID int,
) ([]models.Notification, error) {

	query := `
		SELECT
			id,
			user_id,
			title,
			message,
			type,
			is_read,
			created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []models.Notification

	for rows.Next() {
		var notification models.Notification

		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Title,
			&notification.Message,
			&notification.Type,
			&notification.IsRead,
			&notification.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notifications, nil
}

// ============================================================
// GET UNREAD COUNT
//
// Returns the number of unread notifications belonging to
// one user.
//
// This will later be useful for the 🔔 notification badge.
// ============================================================

// GetUnreadCount returns the number of unread notifications
// belonging to the specified user.
func (r *NotificationRepository) GetUnreadCount(userID int) (int, error) {

	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE user_id = $1
		AND is_read = FALSE
	`

	var count int

	err := r.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// ============================================================
// MARK ONE NOTIFICATION AS READ
//
// Marks one notification as read.
//
// The userID condition is important because a user should not
// be able to mark another user's notification as read.
// ============================================================

// MarkAsRead marks one notification as read, provided that
// the notification belongs to the specified user.
func (r *NotificationRepository) MarkAsRead(
	notificationID int,
	userID int,
) error {

	query := `
		UPDATE notifications
		SET is_read = TRUE
		WHERE id = $1
		AND user_id = $2
	`

	_, err := r.db.Exec(
		query,
		notificationID,
		userID,
	)

	return err
}

// ============================================================
// MARK ALL NOTIFICATIONS AS READ
//
// Marks every notification belonging to the current user
// as read.
// ============================================================

// MarkAllAsRead marks all unread notifications belonging
// to the specified user as read.
func (r *NotificationRepository) MarkAllAsRead(userID int) error {

	query := `
		UPDATE notifications
		SET is_read = TRUE
		WHERE user_id = $1
		AND is_read = FALSE
	`

	_, err := r.db.Exec(query, userID)

	return err
}
