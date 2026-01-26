package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valksor/go-mehrhof/internal/workflow"
)

func TestHandler_Guide_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/guide")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_ListAgents_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/agents")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_ListProviders(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // Providers endpoint doesn't require conductor
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/providers")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result providersListResponse
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Greater(t, result.Count, 0)
	assert.Greater(t, len(result.Providers), 0)

	// Verify some known providers
	var foundFile, foundGithub bool
	for _, p := range result.Providers {
		if p.Scheme == "file" {
			foundFile = true
		}
		if p.Scheme == "github" {
			foundGithub = true
		}
	}
	assert.True(t, foundFile, "expected file provider")
	assert.True(t, foundGithub, "expected github provider")
}

func TestGetGuideActions(t *testing.T) {
	tests := []struct {
		name           string
		state          workflow.State
		specifications int
		wantEndpoint   string
	}{
		{
			name:           "idle with no specs",
			state:          workflow.StateIdle,
			specifications: 0,
			wantEndpoint:   "POST /api/v1/workflow/plan",
		},
		{
			name:           "idle with specs",
			state:          workflow.StateIdle,
			specifications: 1,
			wantEndpoint:   "POST /api/v1/workflow/implement",
		},
		{
			name:           "implementing",
			state:          workflow.StateImplementing,
			specifications: 1,
			wantEndpoint:   "POST /api/v1/workflow/finish",
		},
		{
			name:           "done",
			state:          workflow.StateDone,
			specifications: 1,
			wantEndpoint:   "POST /api/v1/workflow/start",
		},
		{
			name:           "waiting",
			state:          workflow.StateWaiting,
			specifications: 1,
			wantEndpoint:   "POST /api/v1/workflow/answer",
		},
		{
			name:           "planning",
			state:          workflow.StatePlanning,
			specifications: 0,
			wantEndpoint:   "GET /api/v1/task",
		},
		{
			name:           "reviewing",
			state:          workflow.StateReviewing,
			specifications: 1,
			wantEndpoint:   "POST /api/v1/workflow/finish",
		},
		{
			name:           "failed",
			state:          workflow.StateFailed,
			specifications: 1,
			wantEndpoint:   "POST /api/v1/workflow/implement",
		},
		{
			name:           "checkpointing",
			state:          workflow.StateCheckpointing,
			specifications: 1,
			wantEndpoint:   "GET /api/v1/task",
		},
		{
			name:           "reverting",
			state:          workflow.StateReverting,
			specifications: 1,
			wantEndpoint:   "GET /api/v1/task",
		},
		{
			name:           "restoring",
			state:          workflow.StateRestoring,
			specifications: 1,
			wantEndpoint:   "GET /api/v1/task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions := getGuideActions(tt.state, tt.specifications)
			found := false
			for _, action := range actions {
				if action.Endpoint == tt.wantEndpoint {
					found = true

					break
				}
			}
			assert.True(t, found, "expected actions to contain endpoint %q, got %v", tt.wantEndpoint, actions)
		})
	}
}

func TestGetGuideActions_HasCommandAndDescription(t *testing.T) {
	states := []workflow.State{
		workflow.StateIdle,
		workflow.StatePlanning,
		workflow.StateImplementing,
		workflow.StateReviewing,
		workflow.StateDone,
		workflow.StateWaiting,
		workflow.StateFailed,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			actions := getGuideActions(state, 1)
			require.Greater(t, len(actions), 0)
			for _, action := range actions {
				assert.NotEmpty(t, action.Command, "action should have command")
				assert.NotEmpty(t, action.Description, "action should have description")
				assert.NotEmpty(t, action.Endpoint, "action should have endpoint")
			}
		})
	}
}

func TestHandler_ListProviders_AllExpectedProviders(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/providers")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result providersListResponse
	require.NoError(t, json.Unmarshal(respBody, &result))

	// Check for expected providers
	expectedProviders := []string{"file", "dir", "github", "gitlab", "jira", "linear", "notion"}
	for _, expected := range expectedProviders {
		found := false
		for _, p := range result.Providers {
			if p.Scheme == expected {
				found = true

				break
			}
		}
		assert.True(t, found, "expected provider %q", expected)
	}
}

func TestHandler_ListProviders_ProviderHasAllFields(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/providers")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	var result providersListResponse
	require.NoError(t, json.Unmarshal(respBody, &result))

	for _, p := range result.Providers {
		assert.NotEmpty(t, p.Scheme, "provider should have scheme")
		assert.NotEmpty(t, p.Name, "provider should have name")
		assert.NotEmpty(t, p.Description, "provider should have description")
	}
}

func TestHandler_ListProviders_CountMatchesLength(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/providers")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	var result providersListResponse
	require.NoError(t, json.Unmarshal(respBody, &result))

	assert.Equal(t, len(result.Providers), result.Count)
}

func TestHandler_ListAgents_ResponseStructure(t *testing.T) {
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

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/agents")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)

	// Verify response contains JSON structure
	assert.Contains(t, string(respBody), `"agents"`, "response should contain agents field")
	assert.Contains(t, string(respBody), `"count"`, "response should contain count field")

	var result map[string]any
	require.NoError(t, json.Unmarshal(respBody, &result))

	// Verify basic structure
	_, hasAgents := result["agents"]
	_, hasCount := result["count"]
	assert.True(t, hasAgents, "response should have agents field")
	assert.True(t, hasCount, "response should have count field")
}

func TestHandler_ListAgents_WithCapabilities(t *testing.T) {
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

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/agents")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)

	// Parse as map to handle potential nil slices
	var result map[string]any
	require.NoError(t, json.Unmarshal(respBody, &result))

	// Verify response structure is correct
	agentsField, hasAgents := result["agents"]
	count, hasCount := result["count"]

	assert.True(t, hasAgents, "response should have agents field")
	assert.True(t, hasCount, "response should have count field")

	// Extract agents slice for count validation
	var agentsSlice []any
	if agents, ok := agentsField.([]any); ok {
		agentsSlice = agents

		// Verify structure of each agent
		for _, agentAny := range agentsSlice {
			if agent, ok := agentAny.(map[string]any); ok {
				assert.NotEmpty(t, agent["name"], "agent should have name")
				assert.NotEmpty(t, agent["type"], "agent should have type")
			}
		}
	}

	// Verify count is correct
	if countFloat, ok := count.(float64); ok {
		assert.Equal(t, float64(len(agentsSlice)), countFloat, "count should match agents length")
	}
}

func TestHandler_ListAgents_WithModels(t *testing.T) {
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

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/agents")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result agentsListResponse
	require.NoError(t, json.Unmarshal(respBody, &result))

	// Verify model structure is correct (if any agents have models)
	for _, agent := range result.Agents {
		for _, model := range agent.Models {
			assert.NotEmpty(t, model.ID, "model should have ID")
			assert.NotEmpty(t, model.Name, "model should have Name")
		}
	}
}

func TestHandler_WorkflowDiagram_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/workflow/diagram")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(respBody, &result))
	assert.Contains(t, result["error"], "conductor not initialized")
}

func TestHandler_WorkflowDiagram_ReturnsSVG(t *testing.T) {
	// Create conductor for this test
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

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/workflow/diagram")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify content type is SVG
	contentType := resp.Header.Get("Content-Type")
	assert.Equal(t, "image/svg+xml", contentType)

	// Verify body contains SVG content
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "<svg", "response should contain SVG")
	assert.Contains(t, bodyStr, "state-box", "SVG should contain state boxes")
	assert.Contains(t, bodyStr, "</svg>", "response should close SVG tag")
}

func TestHandler_License(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // License endpoint doesn't require conductor
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Test project license endpoint
	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/license")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "BSD 3-Clause")
	assert.Contains(t, string(body), "SIA Valksor")
}

func TestHandler_LicenseInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil, // License info endpoint doesn't require conductor
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/license/info")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result licensesListResponse
	require.NoError(t, json.Unmarshal(body, &result))

	assert.Greater(t, result.Count, 0, "expected at least one license")
	assert.NotEmpty(t, result.Licenses)

	// Check structure
	for _, lic := range result.Licenses {
		assert.NotEmpty(t, lic.Path, "license path should not be empty")
		assert.NotEmpty(t, lic.License, "license should not be empty")
	}
}
