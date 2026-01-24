package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"
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
)

// sessionStore manages active sessions in memory.
type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*session
}

// session represents an active user session.
type session struct {
	Token     string
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// newSessionStore creates a new session store.
func newSessionStore() *sessionStore {
	return &sessionStore{
		sessions: make(map[string]*session),
	}
}

// create creates a new session for a user.
func (s *sessionStore) create(username string) (*session, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	sess := &session{
		Token:     token,
		Username:  username,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sessionDuration),
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

// GetUserFromContext extracts the username from the request context.
// This is exported for use by handlers that need to identify the logged-in user.
func GetUserFromContext(ctx context.Context) string {
	user, _ := ctx.Value(userContextKey).(string)

	return user
}
