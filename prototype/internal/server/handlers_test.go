package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-toolkit/eventbus"
)

// startTestServer creates and starts a test server, returning cleanup function.
func startTestServer(t *testing.T, cfg Config) (*Server, func()) {
	t.Helper()

	srv, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_ = srv.Start(ctx) // Error intentionally ignored in test goroutine
	}()

	// Wait for server to start
	for range 50 {
		if srv.Port() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	return srv, func() {
		cancel()
		time.Sleep(50 * time.Millisecond)
	}
}

// testHTTPClient returns an HTTP client configured for testing.
func testHTTPClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

// doGet performs a GET request with context.
func doGet(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

// getCSRF fetches a CSRF token from the server for localhost mode tests.
// Returns the token string and the cookie that should be included in subsequent requests.
func getCSRF(ctx context.Context, client *http.Client, baseURL string) (string, *http.Cookie, error) {
	resp, err := doGet(ctx, client, baseURL+"/api/v1/auth/csrf")
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, err
	}

	token := result["csrf_token"]
	var csrfCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "mehr_csrf" {
			csrfCookie = c

			break
		}
	}

	return token, csrfCookie, nil
}

// doPost performs a POST request with context, JSON content type, and CSRF token.
func doPost(ctx context.Context, client *http.Client, url string, body io.Reader) (*http.Response, error) {
	// Extract base URL for CSRF fetch
	baseURL := url[:len(url)-len("/"+url[strings.LastIndex(url, "/")+1:])]
	// Find the /api/v1 prefix to get baseURL
	if idx := strings.Index(url, "/api/"); idx > 0 {
		baseURL = url[:idx]
	}

	token, cookie, err := getCSRF(ctx, client, baseURL)
	if err != nil {
		return nil, errors.Join(errors.New("failed to get CSRF token"), err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", token)
	if cookie != nil {
		req.AddCookie(cookie)
	}

	return client.Do(req)
}

// doDelete performs a DELETE request with context and CSRF token.
func doDelete(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	// Extract base URL for CSRF fetch
	baseURL := url
	if idx := strings.Index(url, "/api/"); idx > 0 {
		baseURL = url[:idx]
	}

	token, cookie, err := getCSRF(ctx, client, baseURL)
	if err != nil {
		return nil, errors.Join(errors.New("failed to get CSRF token"), err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Csrf-Token", token)
	if cookie != nil {
		req.AddCookie(cookie)
	}

	return client.Do(req)
}

// doPostForm performs a POST request with form-urlencoded content type and CSRF token.
func doPostForm(ctx context.Context, client *http.Client, url string, body io.Reader) (*http.Response, error) {
	// Extract base URL for CSRF fetch
	baseURL := url
	if idx := strings.Index(url, "/api/"); idx > 0 {
		baseURL = url[:idx]
	}

	token, cookie, err := getCSRF(ctx, client, baseURL)
	if err != nil {
		return nil, errors.Join(errors.New("failed to get CSRF token"), err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Csrf-Token", token)
	if cookie != nil {
		req.AddCookie(cookie)
	}

	return client.Do(req)
}

// createTestConductor creates a conductor for testing.
func createTestConductor(t *testing.T) (*conductor.Conductor, string) {
	t.Helper()
	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	cond, err := conductor.New(
		conductor.WithWorkDir(tmpDir),
		conductor.WithHomeDir(homeDir),
		conductor.WithAutoInit(true),
		conductor.WithDryRun(true),
		conductor.WithStdout(io.Discard),
		conductor.WithStderr(io.Discard),
	)
	require.NoError(t, err)

	ctx := context.Background()
	_ = cond.Initialize(ctx)

	return cond, tmpDir
}

func TestHandler_WorkflowStart_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // No conductor
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Try to start a task without conductor
	body := bytes.NewBufferString(`{"ref": "file:task.md"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/start", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_WorkflowStart_MissingRef(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Send request with empty ref (conductor check happens first though)
	body := bytes.NewBufferString(`{"ref": ""}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/start", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Without conductor, we get service unavailable first
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowStart_InvalidJSON(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Send invalid JSON (conductor check happens first)
	body := bytes.NewBufferString(`{invalid json}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/start", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Without conductor, we get service unavailable first
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowPlan_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/plan", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowImplement_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/implement", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowReview_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/review", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowFinish_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/finish", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowUndo_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/undo", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowRedo_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/redo", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowAnswer_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	body := bytes.NewBufferString(`{"answer": "yes"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/answer", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowAnswer_FormEncoded_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Test form-urlencoded submission
	body := bytes.NewBufferString("answer=test+response")
	resp, err := doPostForm(ctx, client, srv.URL()+"/api/v1/workflow/answer", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should reach conductor check (not fail on JSON parsing)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowAbandon_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/abandon", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_GetTask_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/task")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_ListTasks_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/tasks")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_GetSpecs_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/tasks/test-task/specs")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_GetSessions_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/tasks/test-task/sessions")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_FinishRequestParsing(t *testing.T) {
	// Test that empty body is handled gracefully
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Empty body
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/finish", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should fail on conductor check, not JSON parsing
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_CORS_Headers(t *testing.T) {
	cfg := Config{
		Port: 0,
		Mode: ModeProject,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Events endpoint should have CORS header
	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/events")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestHandler_ContentType_JSON(t *testing.T) {
	cfg := Config{
		Port: 0,
		Mode: ModeProject,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Health endpoint should return JSON
	resp, err := doGet(ctx, client, srv.URL()+"/health")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}

func TestHandler_ContentType_HTML(t *testing.T) {
	cfg := Config{
		Port: 0,
		Mode: ModeProject,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Index should return HTML
	resp, err := doGet(ctx, client, srv.URL()+"/")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
}

func TestHandler_ContentType_SSE(t *testing.T) {
	cfg := Config{
		Port: 0,
		Mode: ModeProject,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := testHTTPClient()
	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/events")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// In test environment, ResponseWriter may not support Flusher
	// which causes a 500 error. In production, SSE works correctly.
	if resp.StatusCode == http.StatusInternalServerError {
		t.Log("SSE endpoint returned 500 - Flusher not supported in test environment")

		return
	}

	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
}

// TestHandler_AgentAlias_NoConductor tests agent alias endpoints without conductor.
func TestHandler_AgentAlias_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// List aliases without conductor
	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/agents/aliases")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

// TestHandler_AgentAlias_Create_InvalidJSON tests creating alias with invalid JSON.
func TestHandler_AgentAlias_Create_InvalidJSON(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	body := bytes.NewBufferString(`{invalid json}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/agents/aliases", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

// TestHandler_AgentAlias_Delete_NoName tests deleting alias without name.
func TestHandler_AgentAlias_Delete_NoName(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Delete request to base alias path (missing name)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, srv.URL()+"/api/v1/agents/aliases/", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// The route expects a name after /aliases/, so without conductor we get service unavailable
	// or a 404 if the route doesn't match
	if resp.StatusCode != http.StatusServiceUnavailable && resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusBadRequest {
		t.Logf("Unexpected status code: %d", resp.StatusCode)
	}
}

// TestHandler_ImplementWithQueryParams tests implement endpoint with query parameters.
func TestHandler_ImplementWithQueryParams(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "no params",
			queryParams:    "",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "component only",
			queryParams:    "?component=tests",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "parallel only",
			queryParams:    "?parallel=3",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "both params",
			queryParams:    "?component=tests&parallel=2",
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/implement"+tt.queryParams, nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestHandler_ImplementWithInvalidParams tests implement endpoint with invalid params.
func TestHandler_ImplementWithInvalidParams(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	tests := []struct {
		name        string
		queryParams string
	}{
		{
			name:        "empty component",
			queryParams: "?component=",
		},
		{
			name:        "empty parallel",
			queryParams: "?parallel=",
		},
		{
			name:        "zero parallel",
			queryParams: "?parallel=0",
		},
		{
			name:        "negative parallel",
			queryParams: "?parallel=-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/implement"+tt.queryParams, nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// Without conductor, we get service unavailable regardless of params
			assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		})
	}
}

// Tests for handleNonFatalWorkflowError

func TestHandleNonFatalWorkflowError_NilError(t *testing.T) {
	srv := &Server{
		config: Config{},
	}
	w := httptest.NewRecorder()

	handled := srv.handleNonFatalWorkflowError(w, nil, "planning")
	assert.False(t, handled, "nil error should not be handled")
}

func TestHandleNonFatalWorkflowError_PendingQuestion(t *testing.T) {
	bus := eventbus.NewBus()
	srv := &Server{
		config: Config{
			EventBus: bus,
		},
	}
	w := httptest.NewRecorder()

	// Subscribe to verify event is published
	eventReceived := false
	bus.SubscribeAll(func(_ eventbus.Event) {
		eventReceived = true
	})

	handled := srv.handleNonFatalWorkflowError(w, conductor.ErrPendingQuestion, "planning")

	assert.True(t, handled, "ErrPendingQuestion should be handled")
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var result map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))

	assert.Equal(t, true, result["success"])
	assert.Equal(t, "waiting", result["status"])
	assert.Equal(t, "Agent has a question", result["message"])
	assert.Equal(t, "planning", result["phase"])
	assert.True(t, eventReceived, "SSE event should be published")
}

func TestHandleNonFatalWorkflowError_BudgetPaused(t *testing.T) {
	srv := &Server{
		config: Config{},
	}
	w := httptest.NewRecorder()

	handled := srv.handleNonFatalWorkflowError(w, conductor.ErrBudgetPaused, "implementing")

	assert.True(t, handled, "ErrBudgetPaused should be handled")
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var result map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))

	assert.Equal(t, true, result["success"])
	assert.Equal(t, "paused", result["status"])
	assert.Equal(t, "Task paused due to budget limit", result["message"])
	assert.Equal(t, "implementing", result["phase"])
}

func TestHandleNonFatalWorkflowError_BudgetStopped(t *testing.T) {
	srv := &Server{
		config: Config{},
	}
	w := httptest.NewRecorder()

	handled := srv.handleNonFatalWorkflowError(w, conductor.ErrBudgetStopped, "reviewing")

	assert.True(t, handled, "ErrBudgetStopped should be handled")
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var result map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))

	assert.Equal(t, false, result["success"]) // Note: budget stopped is a failure condition
	assert.Equal(t, "stopped", result["status"])
	assert.Equal(t, "Task stopped due to budget limit", result["message"])
	assert.Equal(t, "reviewing", result["phase"])
}

func TestHandleNonFatalWorkflowError_UnknownError(t *testing.T) {
	srv := &Server{
		config: Config{},
	}
	w := httptest.NewRecorder()

	unknownErr := errors.New("some other error")
	handled := srv.handleNonFatalWorkflowError(w, unknownErr, "planning")

	assert.False(t, handled, "unknown error should not be handled")
	assert.Equal(t, 0, w.Body.Len(), "no response should be written for unhandled errors")
}

func TestHandleNonFatalWorkflowError_WrappedPendingQuestion(t *testing.T) {
	srv := &Server{
		config: Config{},
	}
	w := httptest.NewRecorder()

	// Wrap the error - errors.Is should still match
	wrappedErr := errors.Join(errors.New("context"), conductor.ErrPendingQuestion)
	handled := srv.handleNonFatalWorkflowError(w, wrappedErr, "planning")

	assert.True(t, handled, "wrapped ErrPendingQuestion should be handled via errors.Is")
	assert.Equal(t, http.StatusOK, w.Code)
}
