package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestSessionStore_Create(t *testing.T) {
	store := newSessionStore()

	sess, err := store.create("admin", storage.RoleUser)
	require.NoError(t, err)

	assert.Equal(t, "admin", sess.Username)
	assert.NotEmpty(t, sess.Token)
	assert.False(t, sess.CreatedAt.IsZero())
	assert.False(t, sess.ExpiresAt.IsZero())
	assert.True(t, sess.ExpiresAt.After(sess.CreatedAt))
}

func TestSessionStore_Get(t *testing.T) {
	store := newSessionStore()

	// Create a session
	sess, err := store.create("admin", storage.RoleUser)
	require.NoError(t, err)

	// Get the session
	retrieved, exists := store.get(sess.Token)
	assert.True(t, exists)
	assert.Equal(t, "admin", retrieved.Username)

	// Get non-existent session
	_, exists = store.get("nonexistent")
	assert.False(t, exists)
}

func TestSessionStore_Delete(t *testing.T) {
	store := newSessionStore()

	sess, err := store.create("admin", storage.RoleUser)
	require.NoError(t, err)

	// Verify session exists
	_, exists := store.get(sess.Token)
	assert.True(t, exists)

	// Delete session
	store.delete(sess.Token)

	// Verify session no longer exists
	_, exists = store.get(sess.Token)
	assert.False(t, exists)
}

func TestSessionStore_Expiry(t *testing.T) {
	store := newSessionStore()

	// Create a session with short expiry for testing
	sess := &session{
		Token:     "test-token",
		Username:  "admin",
		Role:      storage.RoleUser,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(-time.Hour), // Already expired
	}

	store.mu.Lock()
	store.sessions[sess.Token] = sess
	store.mu.Unlock()

	// Get should return false for expired session
	_, exists := store.get(sess.Token)
	assert.False(t, exists)
}

func TestIsPublicEndpoint(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/login", true},
		{"/api/v1/auth/login", true},
		{"/health", true},
		{"/assets/index-C9-pDdfR.js", true},  // React SPA assets
		{"/assets/index-0FzapFyr.css", true}, // React SPA styles
		{"/vite.svg", true},                  // Vite favicon
		{"/", false},
		{"/api/v1/status", false},
		{"/api/v1/tasks", false},
		{"/api/v1/workflow/start", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isPublicEndpoint(tt.path)
			assert.Equal(t, tt.expected, result, "path: %s", tt.path)
		})
	}
}

func TestGenerateToken(t *testing.T) {
	token1, err := generateToken()
	require.NoError(t, err)
	assert.Len(t, token1, 64) // 32 bytes = 64 hex chars

	token2, err := generateToken()
	require.NoError(t, err)

	// Tokens should be unique
	assert.NotEqual(t, token1, token2)
}

func TestGetUserFromContext(t *testing.T) {
	// Context with user
	ctx := context.WithValue(context.Background(), userContextKey, "admin")
	user := GetUserFromContext(ctx)
	assert.Equal(t, "admin", user)

	// Context without user
	ctx = context.Background()
	user = GetUserFromContext(ctx)
	assert.Empty(t, user)
}

func TestAuthMiddleware_NoAuthStore(t *testing.T) {
	// Server without auth store should allow all requests
	cfg := Config{
		Port: 0,
		Mode: ModeProject,
		// AuthStore is nil
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	handler := srv.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_PublicEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "pass", storage.RoleUser))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	handler := srv.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Public endpoints should be accessible without auth
	publicPaths := []string{"/login", "/health", "/api/v1/auth/login"}
	for _, path := range publicPaths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "path: %s", path)
	}
}

func TestAuthMiddleware_ProtectedEndpoint_NoSession(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "pass", storage.RoleUser))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	handler := srv.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// API endpoint without session should return 401
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_ProtectedEndpoint_WithSession(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "pass", storage.RoleUser))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	// Create a session
	sess, err := srv.sessions.create("admin", storage.RoleUser)
	require.NoError(t, err)

	handler := srv.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify user is in context
		user := GetUserFromContext(r.Context())
		assert.Equal(t, "admin", user)
		w.WriteHeader(http.StatusOK)
	}))

	// Request with valid session cookie
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: sess.Token,
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_PageRedirect(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "pass", storage.RoleUser))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	handler := srv.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Non-API endpoint without session should redirect to login
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestServer_IsViewer_LocalhostMode(t *testing.T) {
	// Server without auth store (localhost mode)
	cfg := Config{
		Port: 0,
		Mode: ModeProject,
		// AuthStore is nil - localhost mode
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflow/plan", nil)

	// isViewer should always return false in localhost mode
	// This ensures localhost users have full access (no viewer restrictions)
	assert.False(t, srv.isViewer(req), "localhost mode should never be a viewer")
}

func TestServer_IsViewer_WithRole(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("viewer", "pass", storage.RoleViewer))
	require.NoError(t, authStore.AddUser("user", "pass", storage.RoleUser))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		name     string
		username string
		role     storage.Role
		expected bool
	}{
		{"viewer role returns true", "viewer", storage.RoleViewer, true},
		{"user role returns false", "user", storage.RoleUser, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a session for the user
			sess, err := srv.sessions.create(tt.username, tt.role)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/plan", nil)
			req.AddCookie(&http.Cookie{
				Name:  sessionCookieName,
				Value: sess.Token,
			})

			result := srv.isViewer(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServer_IsViewer_NoSession(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "pass", storage.RoleUser))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	// Request without session cookie
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/plan", nil)

	// isViewer should return false when no session exists
	// (the auth middleware would block unauthenticated requests anyway)
	assert.False(t, srv.isViewer(req), "no session should not be treated as viewer")
}
