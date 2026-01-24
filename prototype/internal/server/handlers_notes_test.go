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

func TestHandler_AddNote_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // No conductor
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	body := bytes.NewBufferString(`{"content": "test note"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/tasks/abc123/notes", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_AddNote_MissingContent(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Empty content - but fails on conductor check first
	body := bytes.NewBufferString(`{"content": ""}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/tasks/abc123/notes", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Conductor check happens first
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_AddNote_InvalidJSON(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	body := bytes.NewBufferString(`invalid json`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/tasks/abc123/notes", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Conductor check happens before JSON parsing
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_GetNotes_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/tasks/abc123/notes")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}
