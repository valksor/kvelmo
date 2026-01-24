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

func TestHandler_SecurityScan_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	body := bytes.NewBufferString(`{}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/scan", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_SecurityScan_InvalidFailLevel(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Invalid fail level - but conductor check fails first
	body := bytes.NewBufferString(`{"fail_level": "invalid"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/scan", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_SecurityScan_InvalidScanner(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Invalid scanner - but conductor check fails first
	body := bytes.NewBufferString(`{"scanners": ["invalid"]}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/scan", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_SecurityScan_WithAllOptions(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	body := bytes.NewBufferString(`{
		"dir": "/tmp",
		"scanners": ["gosec", "gitleaks"],
		"fail_level": "high",
		"format": "sarif"
	}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/scan", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_SecurityScan_ValidFailLevels(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	levels := []string{"critical", "high", "medium", "low", "any"}
	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			body := bytes.NewBufferString(`{"fail_level": "` + level + `"}`)
			resp, err := doPost(ctx, client, srv.URL()+"/api/v1/scan", body)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// All fail on conductor check, but the level is valid
			assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		})
	}
}

func TestHandler_SecurityScan_ValidScanners(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	scanners := []string{"gosec", "gitleaks", "govulncheck"}
	for _, scanner := range scanners {
		t.Run(scanner, func(t *testing.T) {
			body := bytes.NewBufferString(`{"scanners": ["` + scanner + `"]}`)
			resp, err := doPost(ctx, client, srv.URL()+"/api/v1/scan", body)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// All fail on conductor check, but the scanner is valid
			assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		})
	}
}

func TestHandler_SecurityScan_EmptyBody(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Empty body should use defaults
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/scan", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}
