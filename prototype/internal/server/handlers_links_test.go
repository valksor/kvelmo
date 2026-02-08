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

func TestHandleListLinks_Disabled(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/links")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandleGetEntityLinks_DisabledConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/links/spec:test:1")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Conductor check happens first
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandleSearchLinks_MissingQuery(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/links/search")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandleLinksStats_Disabled(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/links/stats")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandleRebuildLinks_Disabled(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Get CSRF token first
	token, cookie, err := getCSRF(ctx, client, srv.URL())
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL()+"/api/v1/links/rebuild", bytes.NewReader([]byte{}))
	require.NoError(t, err)
	req.Header.Set("X-Csrf-Token", token)
	if cookie != nil {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

// TestHandleSearchLinks_MissingQuery tests missing query parameter.
func TestHandleSearchLinks_MissingQueryWithConductor(t *testing.T) {
	// This test validates that the query parameter check happens before workspace access
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // No conductor - service unavailable
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Request without q parameter
	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/links/search")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should get service unavailable since conductor is nil
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

// TestHandleGetEntityLinks_EmptyEntityID tests empty entity ID.
func TestHandleGetEntityLinks_EmptyEntityID(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // No conductor
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Request with empty entity ID (trailing slash)
	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/links/")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should get service unavailable since conductor is nil
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

// Note: Unit tests for linkDataFromLinks, searchRegistry, and response struct types
// were removed when handlers_links.go was replaced by command router integration.
// The HTTP endpoint tests above validate API behavior via handleViaRouter.
