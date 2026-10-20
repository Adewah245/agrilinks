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
// Business decisions belong to NotificationService.
// ============================================================

type NotificationRepository struct {
	db *sql.DB
}

// ============================================================
// CREATE REPOSITORY
// ============================================================

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{
		db: db,
	}
}

// ============================================================
// CREATE NOTIFICATION
//
// Current notifications schema:
//
//	user_id
//	notification_type
//	title
//	message
//	reference_type
//	reference_id
//	is_read
//	created_at
//
// The database generates:
// - id
// - created_at
// ============================================================

func (r *NotificationRepository) Create(
	notification *models.Notification,
) error {

	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}

	if notification == nil {
		return sql.ErrNoRows
	}

	query := `
		INSERT INTO notifications (
			user_id,
			notification_type,
			title,
			message,
			reference_type,
			reference_id,
			is_read
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	return r.db.QueryRow(
		query,
		notification.UserID,
		notification.Type,
		notification.Title,
		notification.Message,
		notification.ReferenceType,
		notification.ReferenceID,
		notification.IsRead,
	).Scan(
		&notification.ID,
		&notification.CreatedAt,
	)
}

// ============================================================
// GET USER NOTIFICATIONS
//
// Retrieves all notifications belonging to one user.
//
// Newest notifications are returned first.
// ============================================================

func (r *NotificationRepository) GetUserNotifications(
	userID int,
) ([]models.Notification, error) {

	if r == nil || r.db == nil {
		return nil, sql.ErrConnDone
	}

	query := `
		SELECT
			id,
			user_id,
			notification_type,
			title,
			message,
			reference_type,
			reference_id,
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

	notifications := make([]models.Notification, 0)

	for rows.Next() {
		var notification models.Notification

		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Type,
			&notification.Title,
			&notification.Message,
			&notification.ReferenceType,
			&notification.ReferenceID,
			&notification.IsRead,
			&notification.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		notifications = append(
			notifications,
			notification,
		)
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
// ============================================================

func (r *NotificationRepository) GetUnreadCount(
	userID int,
) (int, error) {

	if r == nil || r.db == nil {
		return 0, sql.ErrConnDone
	}

	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE user_id = $1
		  AND is_read = FALSE
	`

	var count int

	err := r.db.QueryRow(
		query,
		userID,
	).Scan(&count)

	return count, err
}

// ============================================================
// MARK ONE NOTIFICATION AS READ
//
// Security:
//
// The notification must belong to the supplied user.
// ============================================================

func (r *NotificationRepository) MarkAsRead(
	notificationID int,
	userID int,
) error {

	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}

	query := `
		UPDATE notifications
		SET is_read = TRUE
		WHERE id = $1
		  AND user_id = $2
	`

	result, err := r.db.Exec(
		query,
		notificationID,
		userID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ============================================================
// MARK ALL NOTIFICATIONS AS READ
//
// Marks every unread notification belonging to the current
// user as read.
// ============================================================

func (r *NotificationRepository) MarkAllAsRead(
	userID int,
) error {

	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}

	query := `
		UPDATE notifications
		SET is_read = TRUE
		WHERE user_id = $1
		  AND is_read = FALSE
	`

	_, err := r.db.Exec(
		query,
		userID,
	)

	return err
}
