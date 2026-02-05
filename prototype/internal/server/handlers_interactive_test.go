package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_InteractivePage_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// GET /interactive should render even without conductor
	resp, err := doGet(ctx, client, srv.URL()+"/interactive")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should return OK (page renders with empty state)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandler_InteractiveChat_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// POST /api/v1/interactive/chat without conductor
	body := bytes.NewBufferString(`{"message": "test message"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/interactive/chat", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_InteractiveChat_EmptyMessage(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// POST with empty message - returns 503 because no conductor
	body := bytes.NewBufferString(`{"message": ""}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/interactive/chat", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_InteractiveChat_InvalidBody(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// POST with invalid JSON - returns 503 because conductor check comes before JSON parsing
	body := bytes.NewBufferString(`{invalid json}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/interactive/chat", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_InteractiveCommand_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// POST /api/v1/interactive/command without conductor
	body := bytes.NewBufferString(`{"command": "status"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/interactive/command", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_InteractiveCommand_EmptyBody(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// POST with empty body
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/interactive/command", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_InteractiveCommand_UnknownCommand(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// POST with unknown command
	body := bytes.NewBufferString(`{"command": "unknown-command"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/interactive/command", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_InteractiveState_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// GET /api/v1/interactive/state without conductor
	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/interactive/state")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should return OK with empty state
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	require.NoError(t, json.Unmarshal(respBody, &result))
	success, ok := result["success"].(bool)
	require.True(t, ok, "success should be a bool")
	assert.True(t, success)
	assert.Nil(t, result["state"])
}

func TestHandler_InteractiveAnswer_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// POST /api/v1/interactive/answer without conductor
	body := bytes.NewBufferString(`{"response": "yes"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/interactive/answer", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_InteractiveAnswer_EmptyResponse(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// POST with empty response
	body := bytes.NewBufferString(`{"response": ""}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/interactive/answer", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_InteractiveStop_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// POST /api/v1/interactive/stop without conductor
	body := bytes.NewBufferString(`{}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/interactive/stop", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Stop should succeed even without conductor (it's a no-op)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	require.NoError(t, json.Unmarshal(respBody, &result))
	success, ok := result["success"].(bool)
	require.True(t, ok, "success should be a bool")
	assert.True(t, success)
}

func TestHandler_InteractiveCommand_StartRequiresReference(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// POST start command without reference (would fail at handler level)
	body := bytes.NewBufferString(`{"command": "start", "args": []}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/interactive/command", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_InteractiveState_WrongMethod(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Try POST on GET endpoint - should return 405 method not allowed
	body := bytes.NewBufferString(`{}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/interactive/state", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestHandler_InteractiveCommands_ReturnsCommandList(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // Discovery doesn't require conductor
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// GET /api/v1/interactive/commands should return command metadata
	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/interactive/commands")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Commands []struct {
			Name        string   `json:"name"`
			Aliases     []string `json:"aliases"`
			Description string   `json:"description"`
			Category    string   `json:"category"`
		} `json:"commands"`
	}
	require.NoError(t, json.Unmarshal(respBody, &result))

	// Should have multiple commands
	assert.Greater(t, len(result.Commands), 10, "expected many commands")

	// Check for core commands
	commandNames := make(map[string]bool)
	for _, cmd := range result.Commands {
		commandNames[cmd.Name] = true
	}

	assert.True(t, commandNames["plan"], "expected 'plan' command")
	assert.True(t, commandNames["implement"], "expected 'implement' command")
	assert.True(t, commandNames["review"], "expected 'review' command")
	assert.True(t, commandNames["status"], "expected 'status' command")
	assert.True(t, commandNames["reset"], "expected 'reset' command (Web-specific)")
}

func TestHandler_InteractiveCommands_WrongMethod(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// POST on GET endpoint should return 405
	body := bytes.NewBufferString(`{}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/interactive/commands", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}
