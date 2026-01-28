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

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/events"
)

func TestHandler_StandaloneReview_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // No conductor
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Try standalone review without conductor
	body := bytes.NewBufferString(`{"mode": "uncommitted"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/review/standalone", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_StandaloneReview_InvalidJSON(t *testing.T) {
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
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/review/standalone", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Conductor check happens before JSON parsing
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_StandaloneReview_EmptyBody(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Empty body should default to uncommitted mode
	body := bytes.NewBufferString(`{}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/review/standalone", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Fails because no conductor
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_StandaloneReview_AllModes(t *testing.T) {
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
		name string
		body string
	}{
		{
			name: "uncommitted mode",
			body: `{"mode": "uncommitted"}`,
		},
		{
			name: "branch mode",
			body: `{"mode": "branch", "base_branch": "main"}`,
		},
		{
			name: "range mode",
			body: `{"mode": "range", "range": "HEAD~3..HEAD"}`,
		},
		{
			name: "files mode",
			body: `{"mode": "files", "files": ["src/foo.go", "src/bar.go"]}`,
		},
		{
			name: "with agent",
			body: `{"mode": "uncommitted", "agent": "claude"}`,
		},
		{
			name: "with context lines",
			body: `{"mode": "uncommitted", "context": 5}`,
		},
		{
			name: "fix mode with checkpoint",
			body: `{"mode": "uncommitted", "apply_fixes": true, "create_checkpoint": true}`,
		},
		{
			name: "fix mode without checkpoint",
			body: `{"mode": "uncommitted", "apply_fixes": true, "create_checkpoint": false}`,
		},
		{
			name: "review only mode explicit",
			body: `{"mode": "uncommitted", "apply_fixes": false}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.NewBufferString(tt.body)
			resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/review/standalone", body)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// All fail because no conductor, but request should be parseable
			assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		})
	}
}

func TestHandler_StandaloneSimplify_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // No conductor
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Try standalone simplify without conductor
	body := bytes.NewBufferString(`{"mode": "uncommitted"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/simplify/standalone", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_StandaloneSimplify_InvalidJSON(t *testing.T) {
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
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/simplify/standalone", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Conductor check happens before JSON parsing
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_StandaloneSimplify_EmptyBody(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Empty body should default to uncommitted mode
	body := bytes.NewBufferString(`{}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/simplify/standalone", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Fails because no conductor
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_StandaloneSimplify_AllModes(t *testing.T) {
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
		name string
		body string
	}{
		{
			name: "uncommitted mode",
			body: `{"mode": "uncommitted"}`,
		},
		{
			name: "branch mode",
			body: `{"mode": "branch", "base_branch": "main"}`,
		},
		{
			name: "range mode",
			body: `{"mode": "range", "range": "HEAD~3..HEAD"}`,
		},
		{
			name: "files mode",
			body: `{"mode": "files", "files": ["src/foo.go", "src/bar.go"]}`,
		},
		{
			name: "with agent",
			body: `{"mode": "uncommitted", "agent": "claude"}`,
		},
		{
			name: "with checkpoint",
			body: `{"mode": "uncommitted", "create_checkpoint": true}`,
		},
		{
			name: "with context lines",
			body: `{"mode": "uncommitted", "context": 5}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.NewBufferString(tt.body)
			resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/simplify/standalone", body)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// All fail because no conductor, but request should be parseable
			assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		})
	}
}

func TestMapDiffMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected conductor.StandaloneDiffMode
	}{
		{
			name:     "uncommitted",
			mode:     "uncommitted",
			expected: conductor.DiffModeUncommitted,
		},
		{
			name:     "branch",
			mode:     "branch",
			expected: conductor.DiffModeBranch,
		},
		{
			name:     "range",
			mode:     "range",
			expected: conductor.DiffModeRange,
		},
		{
			name:     "files",
			mode:     "files",
			expected: conductor.DiffModeFiles,
		},
		{
			name:     "unknown defaults to uncommitted",
			mode:     "unknown",
			expected: conductor.DiffModeUncommitted,
		},
		{
			name:     "empty defaults to uncommitted",
			mode:     "",
			expected: conductor.DiffModeUncommitted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapDiffMode(tt.mode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStreamEvent(t *testing.T) {
	// This tests the streamEvent method with different event data shapes
	s := &Server{}

	tests := []struct {
		name   string
		data   map[string]any
		expect string
	}{
		{
			name: "content event",
			data: map[string]any{
				"event": map[string]any{
					"type": "content",
					"text": "Hello world",
				},
			},
			expect: `{"event":"content","text":"Hello world"}`,
		},
		{
			name: "progress event",
			data: map[string]any{
				"event": map[string]any{
					"type":    "progress",
					"message": "Processing...",
				},
			},
			expect: `{"event":"progress","message":"Processing..."}`,
		},
		{
			name: "unknown event type",
			data: map[string]any{
				"event": map[string]any{
					"type": "unknown",
				},
			},
			expect: "", // Should not output anything
		},
		{
			name: "missing event field",
			data: map[string]any{
				"other": "data",
			},
			expect: "", // Should not output anything
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &responseWriterRecorder{header: make(http.Header)}
			e := events.Event{
				Type: "test",
				Data: tt.data,
			}
			s.streamEvent(w, e)

			output := w.output.String()
			if tt.expect == "" {
				assert.Empty(t, output)
			} else {
				assert.Contains(t, output, tt.expect)
			}
		})
	}
}
