package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestHandler_LoginPage(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "password123"))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/login")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Mehrhof Login")
	assert.Contains(t, string(body), "Username")
	assert.Contains(t, string(body), "Password")
}

func TestHandler_Login_FormSubmit_Success(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "password123"))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	// Submit login form
	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "password123")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL()+"/api/v1/auth/login", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should redirect to home
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Equal(t, "/", resp.Header.Get("Location"))

	// Should set session cookie
	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			sessionCookie = c

			break
		}
	}
	require.NotNil(t, sessionCookie, "session cookie should be set")
	assert.NotEmpty(t, sessionCookie.Value)
	assert.True(t, sessionCookie.HttpOnly)
}

func TestHandler_Login_FormSubmit_InvalidCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "password123"))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Submit login form with wrong password
	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "wrongpassword")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL()+"/api/v1/auth/login", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should show login page with error
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Invalid username or password")
}

func TestHandler_Login_JSON_Success(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "password123"))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Submit login as JSON
	body := bytes.NewBufferString(`{"username":"admin","password":"password123"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/auth/login", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Equal(t, "ok", result["status"])

	// Should set session cookie
	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			sessionCookie = c

			break
		}
	}
	require.NotNil(t, sessionCookie, "session cookie should be set")
}

func TestHandler_Login_EmptyCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "password123"))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Submit login with empty credentials
	form := url.Values{}
	form.Set("username", "")
	form.Set("password", "")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL()+"/api/v1/auth/login", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Username and password are required")
}

func TestHandler_Logout(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "password123"))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// First login to get a session
	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "password123")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL()+"/api/v1/auth/login", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()

	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookieName {
			sessionCookie = c

			break
		}
	}
	require.NotNil(t, sessionCookie)

	// Now logout
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, srv.URL()+"/logout", nil)
	require.NoError(t, err)
	req.AddCookie(sessionCookie)

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should redirect to login
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Equal(t, "/login", resp.Header.Get("Location"))

	// Cookie should be cleared (MaxAge = -1)
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookieName {
			assert.Equal(t, -1, c.MaxAge)

			break
		}
	}
}

func TestHandler_ProtectedEndpoint_RequiresAuth(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "password123"))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Try to access protected endpoint without auth
	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/status")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestHandler_ProtectedEndpoint_WithAuth(t *testing.T) {
	tmpDir := t.TempDir()
	authStore, err := storage.LoadAuthStoreFromPath(tmpDir + "/auth.yaml")
	require.NoError(t, err)
	require.NoError(t, authStore.AddUser("admin", "password123"))

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		AuthStore: authStore,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// First login to get session
	body := bytes.NewBufferString(`{"username":"admin","password":"password123"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/auth/login", body)
	require.NoError(t, err)
	_ = resp.Body.Close()

	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookieName {
			sessionCookie = c

			break
		}
	}
	require.NotNil(t, sessionCookie)

	// Now access protected endpoint with session
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL()+"/api/v1/status", nil)
	require.NoError(t, err)
	req.AddCookie(sessionCookie)

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
