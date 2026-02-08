package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// csrfCookieName is the name of the CSRF cookie (used in localhost mode).
const csrfCookieName = "mehr_csrf"

// csrfMiddleware validates CSRF tokens on state-changing requests (POST/PUT/DELETE).
// Uses double-submit cookie pattern: the client reads the CSRF cookie and sends
// its value in the X-Csrf-Token header.
func (s *Server) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF for safe methods (GET, HEAD, OPTIONS)
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)

			return
		}

		// Get CSRF token from header
		csrfHeader := r.Header.Get("X-Csrf-Token")
		if csrfHeader == "" {
			s.writeError(w, http.StatusForbidden, "CSRF token missing")

			return
		}

		// Validate against CSRF cookie (double-submit pattern)
		cookie, err := r.Cookie(csrfCookieName)
		if err != nil || csrfHeader != cookie.Value {
			s.writeError(w, http.StatusForbidden, "CSRF token invalid")

			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleCSRFToken returns the CSRF token for the current session or generates one.
// Clients fetch this and include it as X-Csrf-Token on state-changing requests.
func (s *Server) handleCSRFToken(w http.ResponseWriter, r *http.Request) {
	// Check if we already have a CSRF cookie
	if cookie, err := r.Cookie(csrfCookieName); err == nil && cookie.Value != "" {
		s.writeJSON(w, http.StatusOK, map[string]string{
			"csrf_token": cookie.Value,
		})

		return
	}

	// Generate new CSRF token
	token, err := generateToken()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to generate CSRF token")

		return
	}

	// Set as cookie (accessible to JS for double-submit)
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Must be readable by JS
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400 * 7, // 7 days
	})

	s.writeJSON(w, http.StatusOK, map[string]string{
		"csrf_token": token,
	})
}

// generateToken creates a cryptographically secure token.
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
