package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test-local response types matching command handler output shapes.
type testProvidersListResponse struct {
	Providers []testProviderInfo `json:"providers"`
	Count     int                `json:"count"`
}

type testProviderInfo struct {
	Scheme      string `json:"scheme"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type testAgentsListResponse struct {
	Agents []testAgentInfo `json:"agents"`
	Count  int             `json:"count"`
}

type testAgentInfo struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Models []testModelInfo `json:"models,omitempty"`
}

type testModelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type testLicensesListResponse struct {
	Licenses []testLicenseEntry `json:"licenses"`
	Count    int                `json:"count"`
}

type testLicenseEntry struct {
	Path    string `json:"path"`
	License string `json:"license"`
}

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
	var result testProvidersListResponse
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
	var result testProvidersListResponse
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
	var result testProvidersListResponse
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
	var result testProvidersListResponse
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
	var result testAgentsListResponse
	require.NoError(t, json.Unmarshal(respBody, &result))

	// Verify model structure is correct (if any agents have models)
	for _, agent := range result.Agents {
		for _, model := range agent.Models {
			assert.NotEmpty(t, model.ID, "model should have ID")
			assert.NotEmpty(t, model.Name, "model should have Name")
		}
	}
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
	var result testLicensesListResponse
	require.NoError(t, json.Unmarshal(body, &result))

	assert.Greater(t, result.Count, 0, "expected at least one license")
	assert.NotEmpty(t, result.Licenses)

	// Check structure
	for _, lic := range result.Licenses {
		assert.NotEmpty(t, lic.Path, "license path should not be empty")
		assert.NotEmpty(t, lic.License, "license should not be empty")
	}
}
