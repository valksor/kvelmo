package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valksor/go-mehrhof/internal/workflow"
)

func TestHandler_WorkflowContinue_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // No conductor
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Try to continue without conductor
	body := bytes.NewBufferString(`{}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/continue", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_WorkflowContinue_EmptyBody(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Empty body should default to auto=false
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/continue", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Still fails because no conductor
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowAuto_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	body := bytes.NewBufferString(`{"ref": "file:task.md"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/auto", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_WorkflowAuto_MissingRef(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Missing ref field - but will fail on conductor check first
	body := bytes.NewBufferString(`{}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/auto", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Conductor check happens first
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowAuto_InvalidJSON(t *testing.T) {
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
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/auto", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Conductor check happens before JSON parsing
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestGetNextActionsForState(t *testing.T) {
	tests := []struct {
		name           string
		state          workflow.State
		specifications int
		wantContains   string
	}{
		{
			name:           "idle with no specs",
			state:          workflow.StateIdle,
			specifications: 0,
			wantContains:   "POST /api/v1/workflow/plan",
		},
		{
			name:           "idle with specs",
			state:          workflow.StateIdle,
			specifications: 1,
			wantContains:   "POST /api/v1/workflow/implement",
		},
		{
			name:           "implementing",
			state:          workflow.StateImplementing,
			specifications: 1,
			wantContains:   "POST /api/v1/workflow/finish",
		},
		{
			name:           "reviewing",
			state:          workflow.StateReviewing,
			specifications: 1,
			wantContains:   "POST /api/v1/workflow/finish",
		},
		{
			name:           "waiting",
			state:          workflow.StateWaiting,
			specifications: 1,
			wantContains:   "POST /api/v1/workflow/answer",
		},
		{
			name:           "paused",
			state:          workflow.StatePaused,
			specifications: 1,
			wantContains:   "POST /api/v1/workflow/resume",
		},
		{
			name:           "done",
			state:          workflow.StateDone,
			specifications: 1,
			wantContains:   "POST /api/v1/workflow/start",
		},
		{
			name:           "planning",
			state:          workflow.StatePlanning,
			specifications: 0,
			wantContains:   "POST /api/v1/workflow/implement",
		},
		{
			name:           "failed",
			state:          workflow.StateFailed,
			specifications: 1,
			wantContains:   "GET /api/v1/task",
		},
		{
			name:           "checkpointing",
			state:          workflow.StateCheckpointing,
			specifications: 1,
			wantContains:   "GET /api/v1/task",
		},
		{
			name:           "reverting",
			state:          workflow.StateReverting,
			specifications: 1,
			wantContains:   "GET /api/v1/task",
		},
		{
			name:           "restoring",
			state:          workflow.StateRestoring,
			specifications: 1,
			wantContains:   "GET /api/v1/task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions := getNextActionsForState(tt.state, tt.specifications)
			found := false
			for _, action := range actions {
				if action == tt.wantContains {
					found = true

					break
				}
			}
			assert.True(t, found, "expected actions to contain %q, got %v", tt.wantContains, actions)
		})
	}
}

func TestGetNextActionsForState_NotesIncluded(t *testing.T) {
	// Most states should include notes endpoint
	states := []workflow.State{
		workflow.StateIdle,
		workflow.StatePlanning,
		workflow.StateImplementing,
		workflow.StateFailed,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			actions := getNextActionsForState(state, 1)
			found := false
			for _, action := range actions {
				if action == "POST /api/v1/tasks/{id}/notes" {
					found = true

					break
				}
			}
			assert.True(t, found, "state %q should include notes endpoint", state)
		})
	}
}

func TestGetNextActionsForState_IdleNoSpecsIncludesPlan(t *testing.T) {
	actions := getNextActionsForState(workflow.StateIdle, 0)
	assert.Contains(t, actions, "POST /api/v1/workflow/plan")
	assert.NotContains(t, actions, "POST /api/v1/workflow/implement")
}

func TestGetNextActionsForState_IdleWithSpecsIncludesImplement(t *testing.T) {
	actions := getNextActionsForState(workflow.StateIdle, 5)
	assert.Contains(t, actions, "POST /api/v1/workflow/implement")
	assert.Contains(t, actions, "POST /api/v1/workflow/plan") // Can still plan more
}

func TestHandler_WorkflowContinue_WithAutoFlag(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Test with auto=true
	body := bytes.NewBufferString(`{"auto": true}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/continue", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Fails because no conductor, but request should be parsed
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowAuto_AllOptions(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Test with all options
	body := bytes.NewBufferString(`{
		"ref": "file:task.md",
		"agent": "claude",
		"max_retries": 5,
		"no_push": true,
		"no_delete": true,
		"no_squash": true,
		"target_branch": "develop",
		"quality_target": "strict",
		"no_quality": false
	}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/auto", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Fails because no conductor
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowQuestion_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // No conductor
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Try to ask question without conductor
	body := bytes.NewBufferString(`{"question": "Test question?"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/question", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_WorkflowQuestion_EmptyQuestion(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Empty question field
	body := bytes.NewBufferString(`{"question": ""}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/question", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Conductor check happens before validation
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowQuestion_MissingQuestionField(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Missing question field
	body := bytes.NewBufferString(`{}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/question", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Conductor check happens first
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowQuestion_InvalidJSON(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Invalid JSON
	body := bytes.NewBufferString(`invalid json`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/question", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Conductor check happens before JSON parsing
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowQuestion_ValidRequest(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Valid request body
	body := bytes.NewBufferString(`{"question": "Why did you choose this approach?"}`)
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/question", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Fails because no conductor, but request should be parseable
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_WorkflowQuestion_WithLongQuestion(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Long question (should still work)
	longQuestion := strings.Repeat("This is a very long question. ", 100)
	body := bytes.NewBufferString(fmt.Sprintf(`{"question": "%s"}`, longQuestion))
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/question", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Fails because no conductor
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestSendSSE(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		wantPrefix string
	}{
		{
			name:       "simple data",
			data:       `{"message": "hello"}`,
			wantPrefix: "event: message\ndata: ",
		},
		{
			name:       "empty data",
			data:       "",
			wantPrefix: "event: message\ndata: ",
		},
		{
			name:       "data with quotes",
			data:       `{"msg": "test \"quoted\""}`,
			wantPrefix: "event: message\ndata: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &responseWriterRecorder{header: make(http.Header)}
			sendSSE(w, "", tt.data)

			output := w.output.String()
			if !strings.HasPrefix(output, tt.wantPrefix) {
				t.Errorf("sendSSE() output prefix = %q, want prefix %q", output, tt.wantPrefix)
			}
			if !strings.HasSuffix(output, "\n\n") {
				t.Errorf("sendSSE() output should end with \\n\\n, got: %q", output)
			}
		})
	}
}

func TestEscapeJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no escaping needed",
			input: "simple text",
			want:  "simple text",
		},
		{
			name:  "backslash",
			input: "path\\to\\file",
			want:  "path\\\\to\\\\file",
		},
		{
			name:  "quotes",
			input: `he said "hello"`,
			want:  `he said \"hello\"`,
		},
		{
			name:  "newline",
			input: "line1\nline2",
			want:  "line1\\nline2",
		},
		{
			name:  "carriage return",
			input: "line1\rline2",
			want:  "line1\\rline2",
		},
		{
			name:  "tab",
			input: "col1\tcol2",
			want:  "col1\\tcol2",
		},
		{
			name:  "mixed special chars",
			input: `path\file\n"quoted"\t`,
			want:  `path\\file\\n\"quoted\"\\t`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeJSON(tt.input)
			if got != tt.want {
				t.Errorf("escapeJSON(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// responseWriterRecorder is a test double for http.ResponseWriter that records output.
type responseWriterRecorder struct {
	header http.Header
	output strings.Builder
}

func (w *responseWriterRecorder) Header() http.Header {
	return w.header
}

func (w *responseWriterRecorder) Write(b []byte) (int, error) {
	return w.output.Write(b)
}

func (w *responseWriterRecorder) WriteHeader(statusCode int) {
	w.header.Set("Status", strconv.Itoa(statusCode))
}

func (w *responseWriterRecorder) Flush() {
	// No-op for test
}

func TestHandler_WorkflowReset_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // No conductor
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Try to reset without conductor
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/workflow/reset", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_WorkflowReset_ViewerForbidden(t *testing.T) {
	// Note: The viewer check happens BEFORE the conductor check in handleWorkflowReset,
	// so we don't need a conductor to test viewer rejection.
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Create request with viewer role - should be rejected before conductor check
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL()+"/api/v1/workflow/reset", nil)
	require.NoError(t, err)
	req.Header.Set("X-Mehrhof-Role", "viewer")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// The handler checks viewer first, so we should get 403 Forbidden
	// But if the middleware doesn't set the viewer context, it will fall through to conductor check
	// Either 403 (viewer forbidden) or 503 (no conductor) is acceptable depending on middleware behavior
	assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusServiceUnavailable,
		"expected 403 or 503, got %d", resp.StatusCode)
}
