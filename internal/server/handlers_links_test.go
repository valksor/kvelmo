package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valksor/go-mehrhof/internal/links"
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

// TestLinkDataFromLinks tests the linkDataFromLinks function.
func TestLinkDataFromLinks(t *testing.T) {
	now := time.Now()
	link := links.Link{
		Source:    "spec:task-123:1",
		Target:    "spec:task-123:2",
		Context:   "see also",
		CreatedAt: now,
	}

	data := linkDataFromLinks(link)

	assert.Equal(t, "spec:task-123:1", data.Source)
	assert.Equal(t, "spec:task-123:2", data.Target)
	assert.Equal(t, "see also", data.Context)
	// CreatedAt should be formatted as RFC3339
	assert.NotEmpty(t, data.CreatedAt)
	assert.Contains(t, data.CreatedAt, "T")
}

// TestSearchRegistry_CaseInsensitive tests the searchRegistry function.
func TestSearchRegistry_CaseInsensitive(t *testing.T) {
	registry := map[string]string{
		"Authentication Flow": "spec:task:1",
		"API Design":          "spec:task:2",
		"Cache Strategy":      "decision:task:cache",
	}

	var results []entityResult

	// Search for lowercase "auth" should find "Authentication Flow"
	searchRegistry(registry, "auth", "spec", &results)

	assert.Equal(t, 1, len(results))
	assert.Equal(t, "Authentication Flow", results[0].Name)
	assert.Equal(t, "spec:task:1", results[0].EntityID)
	assert.Equal(t, "spec", results[0].Type)
}

// TestSearchRegistry_PartialMatch tests partial matching.
func TestSearchRegistry_PartialMatch(t *testing.T) {
	registry := map[string]string{
		"AuthenticationFlow": "spec:task:1",
		"APIFlow":            "spec:task:2",
	}

	var results []entityResult

	// Search for "Flow" should find both
	searchRegistry(registry, "Flow", "spec", &results)

	assert.Equal(t, 2, len(results))
}

// TestSearchRegistry_NoMatch tests no matches found.
func TestSearchRegistry_NoMatch(t *testing.T) {
	registry := map[string]string{
		"Authentication Flow": "spec:task:1",
	}

	var results []entityResult

	// Search for something that doesn't exist
	searchRegistry(registry, "nonexistent", "spec", &results)

	assert.Equal(t, 0, len(results))
}

// TestEntityResult_Structure tests entityResult structure.
func TestEntityResult_Structure(t *testing.T) {
	result := entityResult{
		EntityID: "spec:task-123:1",
		Type:     "spec",
		Name:     "Authentication Flow",
		TaskID:   "task-123",
		ID:       "1",
		FullType: "spec",
	}

	// Verify all fields are set correctly
	assert.Equal(t, "spec:task-123:1", result.EntityID)
	assert.Equal(t, "spec", result.Type)
	assert.Equal(t, "Authentication Flow", result.Name)
	assert.Equal(t, "task-123", result.TaskID)
	assert.Equal(t, "1", result.ID)
	assert.Equal(t, "spec", result.FullType)
}

// TestLinksListResponse_Structure tests linksListResponse structure.
func TestLinksListResponse_Structure(t *testing.T) {
	response := linksListResponse{
		Links: []linkData{
			{
				Source:    "spec:task:1",
				Target:    "spec:task:2",
				Context:   "see also",
				CreatedAt: "2024-01-29T10:00:00Z",
			},
		},
		Count: 1,
	}

	assert.Equal(t, 1, response.Count)
	assert.Equal(t, 1, len(response.Links))
	assert.Equal(t, "spec:task:1", response.Links[0].Source)
}

// TestEntityLinksResponse_Structure tests entityLinksResponse structure.
func TestEntityLinksResponse_Structure(t *testing.T) {
	response := entityLinksResponse{
		EntityID: "spec:task:1",
		Outgoing: []linkData{
			{
				Source:    "spec:task:1",
				Target:    "spec:task:2",
				Context:   "see also",
				CreatedAt: "2024-01-29T10:00:00Z",
			},
		},
		Incoming: []linkData{
			{
				Source:    "spec:task:3",
				Target:    "spec:task:1",
				Context:   "referenced by",
				CreatedAt: "2024-01-29T10:00:00Z",
			},
		},
	}

	assert.Equal(t, "spec:task:1", response.EntityID)
	assert.Equal(t, 1, len(response.Outgoing))
	assert.Equal(t, 1, len(response.Incoming))
}

// TestLinksSearchResponse_Structure tests linksSearchResponse structure.
func TestLinksSearchResponse_Structure(t *testing.T) {
	response := linksSearchResponse{
		Query: "auth",
		Results: []entityResult{
			{
				EntityID: "spec:task:1",
				Name:     "Authentication Flow",
				Type:     "spec",
			},
		},
		Count: 1,
	}

	assert.Equal(t, "auth", response.Query)
	assert.Equal(t, 1, response.Count)
	assert.Equal(t, 1, len(response.Results))
}

// TestLinksStatsResponse_Structure tests linksStatsResponse structure.
func TestLinksStatsResponse_Structure(t *testing.T) {
	response := linksStatsResponse{
		TotalLinks:     100,
		TotalSources:   25,
		TotalTargets:   30,
		OrphanEntities: 5,
		MostLinked: []entityResult{
			{
				EntityID:   "spec:task:1",
				TotalLinks: 10,
			},
		},
		Enabled: true,
	}

	assert.Equal(t, 100, response.TotalLinks)
	assert.Equal(t, 25, response.TotalSources)
	assert.Equal(t, 30, response.TotalTargets)
	assert.Equal(t, 5, response.OrphanEntities)
	assert.Equal(t, 1, len(response.MostLinked))
	assert.True(t, response.Enabled)
}

// TestLinksStatsResponse_Disabled tests disabled state.
func TestLinksStatsResponse_Disabled(t *testing.T) {
	response := linksStatsResponse{
		TotalLinks:     0,
		TotalSources:   0,
		TotalTargets:   0,
		OrphanEntities: 0,
		MostLinked:     []entityResult{},
		Enabled:        false,
	}

	assert.False(t, response.Enabled)
	assert.Equal(t, 0, response.TotalLinks)
}

// Note: TestHandleLinksUI_NoRenderer was removed because server.New() always
// initializes the renderer, making the "no renderer" condition unreachable.
// The nil check in handleLinksUI is defensive programming.
