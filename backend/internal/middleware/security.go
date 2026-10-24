package middleware

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"backend/internal/config"
)

type httpsContextKey struct{}

func RequestIsHTTPS(r *http.Request) bool {
	forwarded, _ := r.Context().Value(httpsContextKey{}).(bool)
	return r.TLS != nil || forwarded
}

// RequestGuard restores the old server's HTTP protections around the upgraded
// routes. Settings belong to this handler instance, not global mutable state.
func RequestGuard(next http.Handler, cfg config.Config) http.Handler {
	crossOrigin := http.NewCrossOriginProtection()
	crossOrigin.AddInsecureBypassPattern("/webhooks/flutterwave")
	crossOrigin.AddInsecureBypassPattern("/ussd")
	protected := crossOrigin.Handler(next)
	authLimit := newRequestLimiter(cfg.AuthRequestsPerMinute)
	aiLimit := newRequestLimiter(cfg.AIRequestsPerMinute)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer, _, _ := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(peer)
		trusted := false
		if cfg.TrustProxyHeaders {
			for _, network := range cfg.TrustedProxyCIDRs {
				if network.Contains(ip) {
					trusted = true
					break
				}
			}
		}
		if trusted {
			r = r.WithContext(context.WithValue(r.Context(), httpsContextKey{}, strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")))
			// The nearest untrusted hop identifies the caller. A client-supplied
			// leftmost address must not bypass throttling behind a proxy chain.
			chain := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
			for i := len(chain) - 1; i >= 0; i-- {
				candidate := net.ParseIP(strings.TrimSpace(chain[i]))
				if candidate == nil {
					break
				}
				peer = candidate.String()
				isProxy := false
				for _, network := range cfg.TrustedProxyCIDRs {
					if network.Contains(candidate) {
						isProxy = true
						break
					}
				}
				if !isProxy {
					break
				}
			}
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'")
		if RequestIsHTTPS(r) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		if r.URL.Path == "/ussd" {
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if cfg.USSDToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(cfg.USSDToken)) != 1 {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		if r.Method == http.MethodPost {
			var limiter *requestLimiter
			switch r.URL.Path {
			case "/login", "/register", "/register/phone-code", "/forgot-password":
				limiter = authLimit
			case "/learn/message", "/learning/chat", "/ai-assistant":
				limiter = aiLimit
			}
			if limiter != nil && !limiter.allow(peer) {
				w.Header().Set("Retry-After", strconv.Itoa(60))
				http.Error(w, "Too many requests. Please try again shortly.", http.StatusTooManyRequests)
				return
			}
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxRequestBytes)
		}
		protected.ServeHTTP(w, r)
	})
}

type requestWindow struct {
	count   int
	started time.Time
}
type requestLimiter struct {
	sync.Mutex
	limit   int
	windows map[string]requestWindow
	swept   time.Time
}

func newRequestLimiter(limit int) *requestLimiter {
	return &requestLimiter{limit: limit, windows: make(map[string]requestWindow)}
}

func (l *requestLimiter) allow(key string) bool {
	l.Lock()
	defer l.Unlock()
	now := time.Now()
	if now.Sub(l.swept) >= time.Minute {
		for k, window := range l.windows {
			if now.Sub(window.started) >= time.Minute {
				delete(l.windows, k)
			}
		}
		l.swept = now
	}
	window := l.windows[key]
	if now.Sub(window.started) >= time.Minute {
		window = requestWindow{started: now}
	}
	if window.count >= l.limit {
		return false
	}
	window.count++
	l.windows[key] = window
	return true
}
