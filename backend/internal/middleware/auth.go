package middleware

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
)

type contextKey string

const userContextKey contextKey = "user"
const sessionCookieName = "agroshield_session"

// Presence is transient activity; authentication itself lives in PostgreSQL.
var presence = struct {
	sync.RWMutex
	lastSeen map[int]time.Time
}{lastSeen: make(map[int]time.Time)}

const onlineWindow = time.Minute

func MarkUserOnline(userID int) {
	if userID <= 0 {
		return
	}
	presence.Lock()
	defer presence.Unlock()
	now := time.Now()
	for id, seen := range presence.lastSeen {
		if now.Sub(seen) > onlineWindow {
			delete(presence.lastSeen, id)
		}
	}
	presence.lastSeen[userID] = now
}

func IsUserOnline(userID int) bool {
	presence.RLock()
	seen, ok := presence.lastSeen[userID]
	presence.RUnlock()
	return ok && time.Since(seen) <= onlineWindow
}

func SetSessionCookie(w http.ResponseWriter, r *http.Request, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", HttpOnly: true,
		Secure: RequestIsHTTPS(r), SameSite: http.SameSiteLaxMode,
		Expires: time.Now().Add(ttl), MaxAge: int(ttl.Seconds()),
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Expires: time.Unix(0, 0), MaxAge: -1,
	})
}

func SessionIDFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

func UserFromContext(r *http.Request) (*models.User, bool) {
	user, ok := r.Context().Value(userContextKey).(*models.User)
	return user, ok && user != nil
}

func RequireAuth(sessions *services.SessionService, repo *repository.UserRepository, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := SessionIDFromRequest(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		id, err := sessions.UserID(r.Context(), token)
		if err != nil {
			log.Printf("load session: %v", err)
			http.Error(w, "Authentication is temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if id == 0 {
			ClearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		user, err := repo.GetByID(id)
		if err != nil {
			log.Printf("load session user: %v", err)
			http.Error(w, "Authentication is temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if user == nil || user.AccountStatus == "suspended" || user.StaffDisabled {
			_ = sessions.Delete(r.Context(), token)
			ClearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		MarkUserOnline(id)
		w.Header().Set("Cache-Control", "no-store")
		next(w, r.WithContext(context.WithValue(r.Context(), userContextKey, user)))
	}
}
