package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
)

// contextKey is the private key type used for storing authenticated
// user information inside an HTTP request context.
type contextKey string

const userContextKey contextKey = "user"

const sessionCookieName = "agroshield_session"

// sessionStore keeps authenticated sessions in memory.
//
// The map stores sessionID -> userID.
//
// This implementation is suitable for a single-server deployment.
// If Agro-Shield is later deployed across multiple server instances,
// this store should be replaced with a shared session store such as Redis.
type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]int
	lastSeen map[int]time.Time
}

// store contains all active in-memory sessions and recent user activity.
var store = &sessionStore{
	sessions: make(map[string]int),
	lastSeen: make(map[int]time.Time),
}

// onlineWindow defines how long a user's recent activity keeps them
// considered online.
const onlineWindow = 60 * time.Second

// newSessionID generates a cryptographically random session identifier.
func newSessionID() string {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is extremely unusual. Returning an empty
		// session ID ensures a failed random read does not create a
		// predictable session identifier.
		return ""
	}

	return hex.EncodeToString(b)
}

// CreateSession creates a new authenticated session for a user
// and returns the generated session ID.
//
// A session ID is stored in memory and associated with the user's ID.
func CreateSession(userID int) string {
	if userID <= 0 {
		return ""
	}

	id := newSessionID()
	if id == "" {
		return ""
	}

	store.mu.Lock()
	store.sessions[id] = userID
	store.mu.Unlock()

	return id
}

// DeleteSession removes an authenticated session.
//
// If the user no longer has another active session, their online
// presence information is also removed.
func DeleteSession(sessionID string) {
	if sessionID == "" {
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	userID, exists := store.sessions[sessionID]

	delete(store.sessions, sessionID)

	if !exists {
		return
	}

	// Check whether the same user still has another active session.
	hasAnotherSession := false

	for _, sessionUserID := range store.sessions {
		if sessionUserID == userID {
			hasAnotherSession = true
			break
		}
	}

	if !hasAnotherSession {
		delete(store.lastSeen, userID)
	}
}

// MarkUserOnline records the user's most recent authenticated activity.
func MarkUserOnline(userID int) {
	if userID <= 0 {
		return
	}

	store.mu.Lock()
	store.lastSeen[userID] = time.Now()
	store.mu.Unlock()
}

// IsUserOnline reports whether a user has been active within the
// configured online presence window.
func IsUserOnline(userID int) bool {
	if userID <= 0 {
		return false
	}

	store.mu.RLock()
	lastSeen, ok := store.lastSeen[userID]
	store.mu.RUnlock()

	return ok && time.Since(lastSeen) <= onlineWindow
}

// getUserIDForSession retrieves the user ID associated with a session.
func getUserIDForSession(sessionID string) (int, bool) {
	if sessionID == "" {
		return 0, false
	}

	store.mu.RLock()
	defer store.mu.RUnlock()

	userID, ok := store.sessions[sessionID]

	return userID, ok
}

// SetSessionCookie writes the authenticated session ID into the
// user's HTTP-only session cookie.
func SetSessionCookie(w http.ResponseWriter, sessionID string) {
	if sessionID == "" {
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})
}

// ClearSessionCookie removes the authenticated session cookie.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

// SessionIDFromRequest reads the Agro-Shield session cookie from
// an incoming HTTP request.
func SessionIDFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}

	if cookie.Value == "" {
		return "", false
	}

	return cookie.Value, true
}

// UserFromContext retrieves the authenticated user attached to the
// request context by RequireAuth.
func UserFromContext(r *http.Request) (*models.User, bool) {
	user, ok := r.Context().Value(userContextKey).(*models.User)

	return user, ok
}

// RequireAuth protects a handler from unauthenticated access.
//
// It checks the session cookie, validates the session, loads the
// authenticated user from the database, records recent activity,
// and attaches the user to the request context.
//
// If authentication fails, the request is redirected to /login.
func RequireAuth(
	repo *repository.UserRepository,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := SessionIDFromRequest(r)

		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userID, ok := getUserIDForSession(sessionID)

		if !ok {
			ClearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user, err := repo.GetByID(userID)

		if err != nil || user == nil {
			DeleteSession(sessionID)
			ClearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		MarkUserOnline(userID)

		ctx := context.WithValue(
			r.Context(),
			userContextKey,
			user,
		)

		next(w, r.WithContext(ctx))
	}
}
