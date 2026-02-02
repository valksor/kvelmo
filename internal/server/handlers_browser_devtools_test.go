package server

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_BrowserNetwork_NoConductor(t *testing.T) {
	cfg := Config{Port: 0, Mode: ModeProject, Conductor: nil}
	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	body := bytes.NewBufferString(`{"duration": 1}`)
	resp, err := doPost(context.Background(), testHTTPClient(), srv.URL()+"/api/v1/browser/network", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	respBody, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(respBody), "conductor not initialized")
}

func TestHandler_BrowserConsole_NoConductor(t *testing.T) {
	cfg := Config{Port: 0, Mode: ModeProject, Conductor: nil}
	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	body := bytes.NewBufferString(`{"duration": 1}`)
	resp, err := doPost(context.Background(), testHTTPClient(), srv.URL()+"/api/v1/browser/console", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_BrowserWebSocket_NoConductor(t *testing.T) {
	cfg := Config{Port: 0, Mode: ModeProject, Conductor: nil}
	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	body := bytes.NewBufferString(`{"duration": 1}`)
	resp, err := doPost(context.Background(), testHTTPClient(), srv.URL()+"/api/v1/browser/websocket", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_BrowserSource_NoConductor(t *testing.T) {
	cfg := Config{Port: 0, Mode: ModeProject, Conductor: nil}
	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	body := bytes.NewBufferString(`{}`)
	resp, err := doPost(context.Background(), testHTTPClient(), srv.URL()+"/api/v1/browser/source", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_BrowserScripts_NoConductor(t *testing.T) {
	cfg := Config{Port: 0, Mode: ModeProject, Conductor: nil}
	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	body := bytes.NewBufferString(`{}`)
	resp, err := doPost(context.Background(), testHTTPClient(), srv.URL()+"/api/v1/browser/scripts", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_BrowserStyles_NoConductor(t *testing.T) {
	cfg := Config{Port: 0, Mode: ModeProject, Conductor: nil}
	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	body := bytes.NewBufferString(`{"selector": "h1", "computed": true}`)
	resp, err := doPost(context.Background(), testHTTPClient(), srv.URL()+"/api/v1/browser/styles", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_BrowserCoverage_NoConductor(t *testing.T) {
	cfg := Config{Port: 0, Mode: ModeProject, Conductor: nil}
	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	body := bytes.NewBufferString(`{"duration": 1, "track_js": true, "track_css": true}`)
	resp, err := doPost(context.Background(), testHTTPClient(), srv.URL()+"/api/v1/browser/coverage", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}
