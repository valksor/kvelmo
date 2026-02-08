package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valksor/go-mehrhof/internal/sandbox"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestHandler_SandboxStatus_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/sandbox/status")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result sandbox.Status
	require.NoError(t, json.Unmarshal(respBody, &result))

	// When no conductor, should return defaults
	assert.False(t, result.Enabled)
	assert.NotNil(t, result.Platform)
}

func TestHandler_SandboxStatus_WithConductor(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	cfg := Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/sandbox/status")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result sandbox.Status
	require.NoError(t, json.Unmarshal(respBody, &result))

	// Default is disabled
	assert.False(t, result.Enabled)
	assert.NotNil(t, result.Platform)
	assert.False(t, result.Active, "not active when no task is running")
}

func TestHandler_SandboxStatus_Enabled(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	// Enable sandbox in config
	ws := cond.GetWorkspace()
	require.NotNil(t, ws)
	cfg, err := ws.LoadConfig()
	require.NoError(t, err)
	cfg.Sandbox = &storage.SandboxSettings{
		Enabled: true,
		Network: true,
	}
	require.NoError(t, ws.SaveConfig(cfg))

	serverCfg := Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	}

	srv, cleanup := startTestServer(t, serverCfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/sandbox/status")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result sandbox.Status
	require.NoError(t, json.Unmarshal(respBody, &result))

	assert.True(t, result.Enabled)
	assert.True(t, result.Network)
}

func TestHandler_SandboxEnable_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/sandbox/enable", strings.NewReader("{}"))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// handleViaRouter returns 503 when conductor is nil
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_SandboxEnable_WithConductor(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	cfg := Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Enable sandbox
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/sandbox/enable", strings.NewReader("{}"))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result sandbox.Status
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.True(t, result.Enabled)

	// Verify by checking status
	resp2, err := doGet(ctx, client, srv.URL()+"/api/v1/sandbox/status")
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()

	respBody2, _ := io.ReadAll(resp2.Body)
	var statusResult sandbox.Status
	require.NoError(t, json.Unmarshal(respBody2, &statusResult))
	assert.True(t, statusResult.Enabled)
}

func TestHandler_SandboxDisable_WithConductor(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	// First enable sandbox
	ws := cond.GetWorkspace()
	require.NotNil(t, ws)
	cfg, err := ws.LoadConfig()
	require.NoError(t, err)
	cfg.Sandbox = &storage.SandboxSettings{
		Enabled: true,
		Network: true,
	}
	require.NoError(t, ws.SaveConfig(cfg))

	serverCfg := Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	}

	srv, cleanup := startTestServer(t, serverCfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Verify enabled first
	resp1, err := doGet(ctx, client, srv.URL()+"/api/v1/sandbox/status")
	require.NoError(t, err)
	defer func() { _ = resp1.Body.Close() }()

	respBody1, _ := io.ReadAll(resp1.Body)
	var status1 sandbox.Status
	require.NoError(t, json.Unmarshal(respBody1, &status1))
	assert.True(t, status1.Enabled, "should start enabled")

	// Disable sandbox
	resp2, err := doPost(ctx, client, srv.URL()+"/api/v1/sandbox/disable", strings.NewReader("{}"))
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	respBody2, _ := io.ReadAll(resp2.Body)
	var result sandbox.Status
	require.NoError(t, json.Unmarshal(respBody2, &result))
	assert.False(t, result.Enabled)

	// Verify by checking status
	resp3, err := doGet(ctx, client, srv.URL()+"/api/v1/sandbox/status")
	require.NoError(t, err)
	defer func() { _ = resp3.Body.Close() }()

	respBody3, _ := io.ReadAll(resp3.Body)
	var statusResult sandbox.Status
	require.NoError(t, json.Unmarshal(respBody3, &statusResult))
	assert.False(t, statusResult.Enabled)
}

func TestHandler_SandboxStatus_ResponseStructure(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	cfg := Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/sandbox/status")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	require.NoError(t, json.Unmarshal(respBody, &result))

	// Verify all expected fields exist
	expectedFields := []string{"enabled", "platform", "active", "supported", "network"}
	for _, field := range expectedFields {
		_, hasField := result[field]
		assert.True(t, hasField, "response should have %s field", field)
	}
}

func TestHandler_SandboxEnable_SSEEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping SSE test in short mode")
	}

	cond, tmpDir := createTestConductor(t)

	cfg := Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Note: Full SSE testing requires a more complex setup with event bus subscription
	// This test verifies the endpoint responds correctly
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/sandbox/enable", bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// The event would be published via SSE to connected clients
	// Verifying actual SSE event delivery requires a separate integration test
}
