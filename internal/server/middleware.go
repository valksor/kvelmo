package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const (
	// userContextKey is the context key for the authenticated user.
	userContextKey contextKey = "user"

	// sessionCookieName is the name of the session cookie.
	sessionCookieName = "mehr_session"

	// sessionDuration is how long sessions last.
	sessionDuration = 7 * 24 * time.Hour

	// sessionCleanupInterval is how often expired sessions are cleaned up.
	sessionCleanupInterval = 1 * time.Hour
)

// sessionStore manages active sessions in memory with automatic cleanup.
type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*session
	stopCh   chan struct{}
	stopped  bool
}

// session represents an active user session.
type session struct {
	Token     string
	Username  string
	Role      storage.Role
	CreatedAt time.Time
	ExpiresAt time.Time
	CSRFToken string // CSRF protection token
}

// newSessionStore creates a new session store and starts the cleanup goroutine.
func newSessionStore() *sessionStore {
	s := &sessionStore{
		sessions: make(map[string]*session),
		stopCh:   make(chan struct{}),
	}
	go s.cleanupLoop()

	return s
}

// stop stops the cleanup goroutine. Call this when shutting down the server.
func (s *sessionStore) stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()

		return
	}
	s.stopped = true
	s.mu.Unlock()
	close(s.stopCh)
}

// cleanupLoop periodically removes expired sessions to prevent memory leaks.
func (s *sessionStore) cleanupLoop() {
	ticker := time.NewTicker(sessionCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.stopCh:
			return
		}
	}
}

// cleanup removes all expired sessions using a two-pass approach
// to minimize lock contention.
func (s *sessionStore) cleanup() {
	now := time.Now()
	expired := make([]string, 0)

	// First pass: identify expired sessions (read lock)
	s.mu.RLock()
	for token, sess := range s.sessions {
		if now.After(sess.ExpiresAt) {
			expired = append(expired, token)
		}
	}
	s.mu.RUnlock()

	// Second pass: delete expired sessions (write lock)
	if len(expired) > 0 {
		s.mu.Lock()
		for _, token := range expired {
			delete(s.sessions, token)
		}
		s.mu.Unlock()
	}
}

// create creates a new session for a user with their role.
func (s *sessionStore) create(username string, role storage.Role) (*session, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	csrfToken, err := generateToken()
	if err != nil {
		return nil, err
	}

	sess := &session{
		Token:     token,
		Username:  username,
		Role:      role,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sessionDuration),
		CSRFToken: csrfToken,
	}

	s.mu.Lock()
	s.sessions[token] = sess
	s.mu.Unlock()

	return sess, nil
}

// get retrieves a session by token.
func (s *sessionStore) get(token string) (*session, bool) {
	s.mu.RLock()
	sess, exists := s.sessions[token]
	s.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(sess.ExpiresAt) {
		s.delete(token)

		return nil, false
	}

	return sess, true
}

// delete removes a session.
func (s *sessionStore) delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// generateToken creates a cryptographically secure token.
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// authMiddleware wraps handlers with authentication.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if no auth store configured (localhost mode)
		if s.config.AuthStore == nil {
			next.ServeHTTP(w, r)

			return
		}

		// Skip auth for public endpoints
		if isPublicEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)

			return
		}

		// Check session cookie
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			s.redirectToLogin(w, r)

			return
		}

		sess, valid := s.sessions.get(cookie.Value)
		if !valid {
			s.redirectToLogin(w, r)

			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), userContextKey, sess.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// csrfMiddleware validates CSRF tokens on state-changing requests (POST/PUT/DELETE).
// Protects against cross-site request forgery when auth is enabled and requests use session cookies.
func (s *Server) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF for safe methods (GET, HEAD, OPTIONS)
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)

			return
		}

		// Skip CSRF if no auth store configured (localhost mode)
		if s.config.AuthStore == nil {
			next.ServeHTTP(w, r)

			return
		}

		// Skip CSRF for webhook endpoints (they use provider-specific auth)
		if strings.HasPrefix(r.URL.Path, "/api/v1/webhooks/") {
			next.ServeHTTP(w, r)

			return
		}

		// Skip CSRF for login endpoint (no session yet)
		if r.URL.Path == "/api/v1/auth/login" {
			next.ServeHTTP(w, r)

			return
		}

		// Validate CSRF token from header against session
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			// No session cookie = no CSRF needed (auth middleware will reject)
			next.ServeHTTP(w, r)

			return
		}

		sess, valid := s.sessions.get(cookie.Value)
		if !valid {
			next.ServeHTTP(w, r)

			return
		}

		csrfToken := r.Header.Get("X-Csrf-Token")
		if csrfToken == "" || csrfToken != sess.CSRFToken {
			s.writeError(w, http.StatusForbidden, "CSRF token invalid or missing")

			return
		}

		next.ServeHTTP(w, r)
	})
}

// rateLimiter provides per-IP rate limiting using a token bucket approach.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	limit   int           // Requests per window
	window  time.Duration // Time window for rate limit
}

// tokenBucket tracks request count for a single IP.
type tokenBucket struct {
	tokens    int
	lastReset time.Time
}

// newRateLimiter creates a rate limiter with the given limit per window.
func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		buckets: make(map[string]*tokenBucket),
		limit:   limit,
		window:  window,
	}
}

// maxBuckets is the maximum number of tracked IPs to prevent unbounded memory growth.
const maxBuckets = 10_000

// allow checks if a request from the given IP is allowed.
func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[ip]

	if !exists || now.Sub(bucket.lastReset) >= rl.window {
		// If at capacity, evict the oldest bucket before adding a new one.
		if !exists && len(rl.buckets) >= maxBuckets {
			var oldestIP string
			var oldestTime time.Time
			for k, v := range rl.buckets {
				if oldestIP == "" || v.lastReset.Before(oldestTime) {
					oldestIP = k
					oldestTime = v.lastReset
				}
			}
			delete(rl.buckets, oldestIP)
		}

		rl.buckets[ip] = &tokenBucket{tokens: 1, lastReset: now}

		return true
	}

	if bucket.tokens >= rl.limit {
		return false
	}

	bucket.tokens++

	return true
}

// cleanup removes stale entries older than 2x the window.
func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-2 * rl.window)
	for ip, bucket := range rl.buckets {
		if bucket.lastReset.Before(cutoff) {
			delete(rl.buckets, ip)
		}
	}
}

// rateLimitMiddleware provides per-IP rate limiting for API endpoints.
func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	// General API: 120 requests per minute
	generalLimiter := newRateLimiter(120, time.Minute)
	// Auth endpoints: 10 attempts per minute (brute-force protection)
	authLimiter := newRateLimiter(10, time.Minute)

	// Periodic cleanup of stale entries
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			generalLimiter.cleanup()
			authLimiter.cleanup()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP
		ip := extractClientIP(r)

		// Use stricter limiter for auth endpoints
		if r.URL.Path == "/api/v1/auth/login" {
			if !authLimiter.allow(ip) {
				w.Header().Set("Retry-After", "60")
				s.writeError(w, http.StatusTooManyRequests, "rate limit exceeded, try again later")

				return
			}
		}

		// General rate limit for all API endpoints
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if !generalLimiter.allow(ip) {
				w.Header().Set("Retry-After", "60")
				s.writeError(w, http.StatusTooManyRequests, "rate limit exceeded, try again later")

				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// extractClientIP extracts the client IP from the request.
func extractClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

// isPublicEndpoint returns true if the path doesn't require authentication.
func isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/login",
		"/api/v1/auth/login",
		"/health",
	}

	for _, p := range publicPaths {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}

	// Static assets
	if strings.HasPrefix(path, "/static/") {
		return true
	}

	return false
}

// redirectToLogin redirects the user to the login page.
func (s *Server) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	// For API requests, return 401
	if strings.HasPrefix(r.URL.Path, "/api/") {
		s.writeError(w, http.StatusUnauthorized, "authentication required")

		return
	}

	// For page requests, redirect to login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handleCSRFToken returns the CSRF token for the current session.
// Clients fetch this after login and include it as X-CSRF-Token on state-changing requests.
func (s *Server) handleCSRFToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "no session")

		return
	}

	sess, valid := s.sessions.get(cookie.Value)
	if !valid {
		s.writeError(w, http.StatusUnauthorized, "invalid session")

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"csrf_token": sess.CSRFToken,
	})
}

// GetUserFromContext extracts the username from the request context.
// This is exported for use by handlers that need to identify the logged-in user.
func GetUserFromContext(ctx context.Context) string {
	user, _ := ctx.Value(userContextKey).(string)

	return user
}

// isViewer returns true if the current request is from a viewer (read-only) user.
// Returns false for localhost mode (no auth) or regular users.
func (s *Server) isViewer(r *http.Request) bool {
	// Localhost mode - full access
	if s.config.AuthStore == nil {
		return false
	}

	sess := s.getSessionFromRequest(r)
	if sess == nil {
		return false
	}

	return sess.Role == storage.RoleViewer
}

// getSessionFromRequest retrieves the session from the request.
// Returns nil if no valid session exists.
func (s *Server) getSessionFromRequest(r *http.Request) *session {
	// Localhost mode - no session
	if s.config.AuthStore == nil {
		return nil
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}

	sess, valid := s.sessions.get(cookie.Value)
	if !valid {
		return nil
	}

	return sess
}
