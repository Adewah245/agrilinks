package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"backend/internal/services"
	"backend/middleware"
)

// NotificationHandler handles notification-related HTTP requests.
//
// The handler receives requests, identifies the authenticated user,
// calls NotificationService, and sends notification data to the
// frontend. It does not communicate directly with PostgreSQL.
type NotificationHandler struct {
	notificationService *services.NotificationService
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(
	notificationService *services.NotificationService,
) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// NotificationsHandler displays all notifications belonging to
// the currently authenticated user.
//
// GET /notifications
func (h *NotificationHandler) NotificationsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	notifications, err := h.notificationService.GetUserNotifications(
		user.ID,
	)
	if err != nil {
		http.Error(
			w,
			"Failed to load notifications",
			http.StatusInternalServerError,
		)
		return
	}

	unreadCount, err := h.notificationService.GetUnreadCount(
		user.ID,
	)
	if err != nil {
		http.Error(
			w,
			"Failed to load notification count",
			http.StatusInternalServerError,
		)
		return
	}

	data := struct {
		User          interface{}
		Notifications interface{}
		UnreadCount   int
	}{
		User:          user,
		Notifications: notifications,
		UnreadCount:   unreadCount,
	}

	tmpl, err := template.ParseFiles(
		"../frontend/pages/notification.html",
	)
	if err != nil {
		http.Error(
			w,
			"Failed to load notifications page",
			http.StatusInternalServerError,
		)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		// The response may already have been partially written,
		// so we do not attempt another http.Error response here.
		return
	}
}

// MarkAsReadHandler marks one notification as read.
//
// POST /notifications/read?id=123
//
// The notification service receives the authenticated user's ID,
// ensuring that a user cannot mark another user's notification
// as read.
func (h *NotificationHandler) MarkAsReadHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	notificationIDString := r.URL.Query().Get("id")

	notificationID, err := strconv.Atoi(
		notificationIDString,
	)
	if err != nil || notificationID <= 0 {
		http.Error(
			w,
			"Invalid notification ID",
			http.StatusBadRequest,
		)
		return
	}

	if err := h.notificationService.MarkAsRead(
		notificationID,
		user.ID,
	); err != nil {
		http.Error(
			w,
			"Failed to mark notification as read",
			http.StatusInternalServerError,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/notifications",
		http.StatusSeeOther,
	)
}

// MarkAllAsReadHandler marks all notifications belonging to the
// authenticated user as read.
//
// POST /notifications/read-all
func (h *NotificationHandler) MarkAllAsReadHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	if err := h.notificationService.MarkAllAsRead(
		user.ID,
	); err != nil {
		http.Error(
			w,
			"Failed to mark notifications as read",
			http.StatusInternalServerError,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/notifications",
		http.StatusSeeOther,
	)
}
