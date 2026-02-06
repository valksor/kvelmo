package server

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valksor/go-mehrhof/internal/helper_test"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestHandleBudgetMonthlyStatus_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/budget/monthly/status")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should return OK with default config (disabled by default)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	// Without conductor, budget defaults to disabled
	assert.Contains(t, bodyStr, `"enabled":false`)
}

func TestHandleBudgetMonthlyStatus_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a conductor with budget config
	c := helper_test.NewTestConductor(t,
		helper_test.TestConductorOptions(tmpDir)...,
	)

	ws := c.GetWorkspace()
	require.NotNil(t, ws)

	// Set up budget config
	wsConfig := storage.NewDefaultWorkspaceConfig()
	wsConfig.Budget.Monthly.MaxCost = 100.0
	wsConfig.Budget.Monthly.WarningAt = 0.8
	wsConfig.Budget.Enabled = true
	err := ws.SaveConfig(wsConfig)
	require.NoError(t, err)

	// Create budget state
	state := &storage.MonthlyBudgetState{
		Month:       time.Now().Format("2006-01"),
		Spent:       45.50,
		WarningSent: false,
	}
	err = ws.SaveMonthlyBudgetState(state)
	require.NoError(t, err)

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: c,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/budget/monthly/status")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	// JSON response should contain budget information
	assert.Contains(t, bodyStr, `"spent":45.5`)     // spent amount
	assert.Contains(t, bodyStr, `"max_cost":100`)   // max cost
	assert.Contains(t, bodyStr, `"remaining":54.5`) // remaining
}

func TestHandleBudgetMonthlyStatus_ExceededBudget(t *testing.T) {
	c := helper_test.NewTestConductor(t)

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: c,
	}

	// Set up workspace with exceeded budget
	ws := c.GetWorkspace()
	require.NotNil(t, ws)

	wsConfig := storage.NewDefaultWorkspaceConfig()
	wsConfig.Budget.Monthly.MaxCost = 50.0
	wsConfig.Budget.Monthly.WarningAt = 0.8
	wsConfig.Budget.Enabled = true
	err := ws.SaveConfig(wsConfig)
	require.NoError(t, err)

	// Set state with exceeded budget
	state := &storage.MonthlyBudgetState{
		Month:       time.Now().Format("2006-01"),
		Spent:       75.0, // Over budget
		WarningSent: true,
	}
	err = ws.SaveMonthlyBudgetState(state)
	require.NoError(t, err)

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/budget/monthly/status")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	// JSON response should indicate exceeded budget
	assert.Contains(t, bodyStr, `"spent":75`)       // spent amount
	assert.Contains(t, bodyStr, `"limit_hit":true`) // over budget
	assert.Contains(t, bodyStr, `"warned":true`)    // warning was sent
}

func TestHandleBudgetMonthlyStatus_NoBudgetSet(t *testing.T) {
	c := helper_test.NewTestConductor(t)

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: c,
	}

	ws := c.GetWorkspace()
	require.NotNil(t, ws)

	// Config with budget disabled
	wsConfig := storage.NewDefaultWorkspaceConfig()
	wsConfig.Budget.Monthly.MaxCost = 0 // Explicitly set to 0
	wsConfig.Budget.Enabled = false     // Disabled
	err := ws.SaveConfig(wsConfig)
	require.NoError(t, err)

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/budget/monthly/status")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	// JSON response shows budget is disabled
	assert.Contains(t, bodyStr, `"enabled":false`)
}

func TestHandleBudgetMonthlyReset_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/budget/monthly/reset", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should return service unavailable
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "conductor not initialized")
}

func TestHandleBudgetMonthlyReset_NoWorkspace(t *testing.T) {
	// Create conductor without workspace
	c := helper_test.NewTestConductor(t)

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: c,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/budget/monthly/reset", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Workspace should exist, so this should succeed
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandleBudgetMonthlyReset_Success(t *testing.T) {
	c := helper_test.NewTestConductor(t)

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: c,
	}

	ws := c.GetWorkspace()
	require.NotNil(t, ws)

	// Set up budget config
	wsConfig := storage.NewDefaultWorkspaceConfig()
	wsConfig.Budget.Monthly.MaxCost = 100.0
	err := ws.SaveConfig(wsConfig)
	require.NoError(t, err)

	// Set initial state with some spending
	state := &storage.MonthlyBudgetState{
		Month:       time.Now().Format("2006-01"),
		Spent:       75.0,
		WarningSent: true,
	}
	err = ws.SaveMonthlyBudgetState(state)
	require.NoError(t, err)

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/budget/monthly/reset", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify the budget was reset
	newState, err := ws.LoadMonthlyBudgetState()
	require.NoError(t, err)
	assert.Equal(t, 0.0, newState.Spent)
	assert.False(t, newState.WarningSent)
}

func TestHandleBudgetMonthlyStatus_NilState(t *testing.T) {
	c := helper_test.NewTestConductor(t)

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: c,
	}

	ws := c.GetWorkspace()
	require.NotNil(t, ws)

	// Set up budget config but no state
	wsConfig := storage.NewDefaultWorkspaceConfig()
	wsConfig.Budget.Monthly.MaxCost = 100.0
	wsConfig.Budget.Enabled = true
	err := ws.SaveConfig(wsConfig)
	require.NoError(t, err)

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/budget/monthly/status")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	// JSON response should show 0 spent since no state file exists
	assert.Contains(t, bodyStr, `"spent":0`)
	assert.Contains(t, bodyStr, `"max_cost":100`)
}

func TestHandleBudgetMonthlyReset_JSONResponse(t *testing.T) {
	c := helper_test.NewTestConductor(t)

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: c,
	}

	ws := c.GetWorkspace()
	require.NotNil(t, ws)

	// Set up budget config
	wsConfig := storage.NewDefaultWorkspaceConfig()
	wsConfig.Budget.Monthly.MaxCost = 100.0
	err := ws.SaveConfig(wsConfig)
	require.NoError(t, err)

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Make POST request to reset
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/budget/monthly/reset", strings.NewReader(""))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Response should be JSON
	ct := resp.Header.Get("Content-Type")
	assert.Contains(t, ct, "application/json")
}
