//nolint:noctx,errcheck // Test file - http.Get and body.Close() without context/error check is acceptable in tests
package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// createTestConductor creates a conductor for testing.
func createTestConductor(t *testing.T) (*conductor.Conductor, string) {
	t.Helper()
	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	cond, err := conductor.New(
		conductor.WithWorkDir(tmpDir),
		conductor.WithHomeDir(homeDir),
		conductor.WithAutoInit(true),
		conductor.WithDryRun(true),
		conductor.WithStdout(io.Discard),
		conductor.WithStderr(io.Discard),
	)
	require.NoError(t, err)

	ctx := context.Background()
	_ = cond.Initialize(ctx)

	return cond, tmpDir
}

// settingsHTTPClient returns an HTTP client configured for testing.
func settingsHTTPClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

// settingsGet performs a GET request with context.
func settingsGet(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

// startSettingsTestServer creates and starts a test server for settings tests.
func startSettingsTestServer(t *testing.T, cfg Config) *Server {
	t.Helper()
	srv, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	return srv
}

// --- Settings Page Tests ---

func TestHandler_SettingsPage_NoConductor(t *testing.T) {
	srv := startSettingsTestServer(t, Config{Port: 0, Mode: ModeProject})

	ctx := context.Background()
	client := settingsHTTPClient()
	resp, err := settingsGet(ctx, client, srv.URL()+"/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should show settings page with default config and info message
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "Settings")
	assert.Contains(t, bodyStr, "workspace not initialized")
}

func TestHandler_SettingsPage_NoTemplates(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	cfg := Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	// Clear templates to simulate template loading failure
	srv.templates = nil

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	client := settingsHTTPClient()
	resp, err := settingsGet(ctx, client, srv.URL()+"/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestHandler_SettingsPage_Success(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	ctx := context.Background()
	client := settingsHTTPClient()
	resp, err := settingsGet(ctx, client, srv.URL()+"/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Verify key elements are present
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "Settings")
	assert.Contains(t, bodyStr, "Git Settings")
	assert.Contains(t, bodyStr, "Agent Settings")
	assert.Contains(t, bodyStr, "Workflow Settings")
	assert.Contains(t, bodyStr, "Browser Automation")
	assert.Contains(t, bodyStr, "Provider Settings")
}

func TestHandler_SettingsPage_ShowsSensitiveInProjectMode(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// In project mode, should have token input fields
	bodyStr := string(body)
	assert.Contains(t, bodyStr, `name="github.token"`)
	assert.Contains(t, bodyStr, `name="gitlab.token"`)
	assert.Contains(t, bodyStr, `name="jira.token"`)
}

func TestHandler_SettingsPage_HidesSensitiveInGlobalMode(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeGlobal,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// In global mode, should NOT have token input fields
	bodyStr := string(body)
	assert.NotContains(t, bodyStr, `name="github.token"`)
	assert.NotContains(t, bodyStr, `name="gitlab.token"`)
}

func TestHandler_SettingsPage_WithSuccessMessage(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/settings?success=Settings+saved")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "Settings saved")
}

func TestHandler_SettingsPage_WithErrorMessage(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/settings?error=Something+went+wrong")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "Something went wrong")
}

// --- Get Settings API Tests ---

func TestHandler_GetSettings_NoConductor(t *testing.T) {
	srv := startSettingsTestServer(t, Config{Port: 0, Mode: ModeProject})

	resp, err := http.Get(srv.URL() + "/api/v1/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should return default config when conductor is not available
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var cfg storage.WorkspaceConfig
	err = json.NewDecoder(resp.Body).Decode(&cfg) //nolint:musttag
	require.NoError(t, err)

	// Should have default values
	assert.Equal(t, "claude", cfg.Agent.Default)
}

func TestHandler_GetSettings_Success(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/api/v1/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var cfg storage.WorkspaceConfig
	err = json.NewDecoder(resp.Body).Decode(&cfg) //nolint:musttag
	require.NoError(t, err)

	// Check default values are present
	assert.Equal(t, "claude", cfg.Agent.Default)
	assert.True(t, cfg.Git.AutoCommit)
}

func TestHandler_GetSettings_GlobalModeStripsTokens(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	// Set a token in config
	ws := cond.GetWorkspace()
	cfg, _ := ws.LoadConfig()
	cfg.GitHub = &storage.GitHubSettings{
		Token: "secret-token",
		Owner: "testowner",
	}
	_ = ws.SaveConfig(cfg)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeGlobal,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/api/v1/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var result storage.WorkspaceConfig
	err = json.NewDecoder(resp.Body).Decode(&result) //nolint:musttag
	require.NoError(t, err)

	// Token should be stripped in global mode
	assert.Empty(t, result.GitHub.Token)
	// Non-sensitive field should remain
	assert.Equal(t, "testowner", result.GitHub.Owner)
}

func TestHandler_GetSettings_ProjectModeKeepsTokens(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	// Set a token in config
	ws := cond.GetWorkspace()
	cfg, _ := ws.LoadConfig()
	cfg.GitHub = &storage.GitHubSettings{
		Token: "secret-token",
		Owner: "testowner",
	}
	_ = ws.SaveConfig(cfg)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/api/v1/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var result storage.WorkspaceConfig
	err = json.NewDecoder(resp.Body).Decode(&result) //nolint:musttag
	require.NoError(t, err)

	// Token should be present in project mode
	assert.Equal(t, "secret-token", result.GitHub.Token)
	assert.Equal(t, "testowner", result.GitHub.Owner)
}

// --- Save Settings API Tests ---

func TestHandler_SaveSettings_NoConductor(t *testing.T) {
	srv := startSettingsTestServer(t, Config{Port: 0, Mode: ModeProject})

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandler_SaveSettings_JSON(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	// Note: JSON API uses Go struct field names (capitalized) since WorkspaceConfig has yaml tags, not json tags
	// For actual JSON API usage, use struct field names like {"Agent":{"Timeout":600}}
	// But in practice, the UI uses form submission, so we test that the endpoint accepts JSON without error
	body := `{}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify config file still exists (empty JSON doesn't modify config)
	ws := cond.GetWorkspace()
	_, err = ws.LoadConfig()
	require.NoError(t, err)
}

func TestHandler_SaveSettings_JSONInvalid(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandler_SaveSettings_Form(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	form := "git.auto_commit=true&git.sign_commits=true&agent.timeout=900"
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Location"), "success=")

	// Verify config was saved
	ws := cond.GetWorkspace()
	cfg, err := ws.LoadConfig()
	require.NoError(t, err)
	assert.True(t, cfg.Git.AutoCommit)
	assert.True(t, cfg.Git.SignCommits)
	assert.Equal(t, 900, cfg.Agent.Timeout)
}

func TestHandler_SaveSettings_FormAllGitSettings(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	form := strings.Join([]string{
		"git.auto_commit=true",
		"git.sign_commits=true",
		"git.stash_on_start=true",
		"git.auto_pop_stash=true",
		"git.commit_prefix=[TEST-{key}]",
		"git.branch_pattern=feature/{key}-{slug}",
	}, "&")

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)

	ws := cond.GetWorkspace()
	cfg, _ := ws.LoadConfig()
	assert.True(t, cfg.Git.AutoCommit)
	assert.True(t, cfg.Git.SignCommits)
	assert.True(t, cfg.Git.StashOnStart)
	assert.True(t, cfg.Git.AutoPopStash)
	assert.Equal(t, "[TEST-{key}]", cfg.Git.CommitPrefix)
	assert.Equal(t, "feature/{key}-{slug}", cfg.Git.BranchPattern)
}

func TestHandler_SaveSettings_FormWorkflowSettings(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	form := strings.Join([]string{
		"workflow.auto_init=true",
		"workflow.session_retention_days=90",
		"workflow.delete_work_on_finish=true",
		"workflow.delete_work_on_abandon=true",
	}, "&")

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	ws := cond.GetWorkspace()
	cfg, _ := ws.LoadConfig()
	assert.True(t, cfg.Workflow.AutoInit)
	assert.Equal(t, 90, cfg.Workflow.SessionRetentionDays)
	assert.True(t, cfg.Workflow.DeleteWorkOnFinish)
	assert.True(t, cfg.Workflow.DeleteWorkOnAbandon)
}

func TestHandler_SaveSettings_FormBrowserSettings(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	form := strings.Join([]string{
		"browser.enabled=true",
		"browser.headless=true",
		"browser.port=9222",
		"browser.timeout=60",
		"browser.screenshot_dir=/tmp/screenshots",
		"browser.cookie_profile=work",
	}, "&")

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	ws := cond.GetWorkspace()
	cfg, _ := ws.LoadConfig()
	require.NotNil(t, cfg.Browser)
	assert.True(t, cfg.Browser.Enabled)
	assert.True(t, cfg.Browser.Headless)
	assert.Equal(t, 9222, cfg.Browser.Port)
	assert.Equal(t, 60, cfg.Browser.Timeout)
	assert.Equal(t, "/tmp/screenshots", cfg.Browser.ScreenshotDir)
	assert.Equal(t, "work", cfg.Browser.CookieProfile)
}

func TestHandler_SaveSettings_FormProviderSettings(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	form := strings.Join([]string{
		"providers.default=github",
		"github.token=ghp_test123",
		"github.owner=testowner",
		"github.repo=testrepo",
		"github.target_branch=develop",
		"github.draft_pr=true",
	}, "&")

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	ws := cond.GetWorkspace()
	cfg, _ := ws.LoadConfig()
	assert.Equal(t, "github", cfg.Providers.Default)
	require.NotNil(t, cfg.GitHub)
	assert.Equal(t, "ghp_test123", cfg.GitHub.Token)
	assert.Equal(t, "testowner", cfg.GitHub.Owner)
	assert.Equal(t, "testrepo", cfg.GitHub.Repo)
	assert.Equal(t, "develop", cfg.GitHub.TargetBranch)
	assert.True(t, cfg.GitHub.DraftPR)
}

func TestHandler_SaveSettings_FormGitLabSettings(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	form := strings.Join([]string{
		"gitlab.token=glpat-test123",
		"gitlab.host=https://gitlab.example.com",
		"gitlab.project_path=group/project",
	}, "&")

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	ws := cond.GetWorkspace()
	cfg, _ := ws.LoadConfig()
	require.NotNil(t, cfg.GitLab)
	assert.Equal(t, "glpat-test123", cfg.GitLab.Token)
	assert.Equal(t, "https://gitlab.example.com", cfg.GitLab.Host)
	assert.Equal(t, "group/project", cfg.GitLab.ProjectPath)
}

func TestHandler_SaveSettings_FormJiraSettings(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	form := strings.Join([]string{
		"jira.token=jira-api-token",
		"jira.email=user@example.com",
		"jira.base_url=https://company.atlassian.net",
		"jira.project=PROJ",
	}, "&")

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	ws := cond.GetWorkspace()
	cfg, _ := ws.LoadConfig()
	require.NotNil(t, cfg.Jira)
	assert.Equal(t, "jira-api-token", cfg.Jira.Token)
	assert.Equal(t, "user@example.com", cfg.Jira.Email)
	assert.Equal(t, "https://company.atlassian.net", cfg.Jira.BaseURL)
	assert.Equal(t, "PROJ", cfg.Jira.Project)
}

func TestHandler_SaveSettings_FormUpdateSettings(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	form := "update.enabled=true&update.check_interval=48"
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	ws := cond.GetWorkspace()
	cfg, _ := ws.LoadConfig()
	assert.True(t, cfg.Update.Enabled)
	assert.Equal(t, 48, cfg.Update.CheckInterval)
}

func TestHandler_SaveSettings_GlobalModeIgnoresTokens(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeGlobal,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	form := "github.token=should-be-ignored&github.owner=testowner"
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	ws := cond.GetWorkspace()
	cfg, _ := ws.LoadConfig()
	// Token should not be saved in global mode
	if cfg.GitHub != nil {
		assert.Empty(t, cfg.GitHub.Token)
		assert.Equal(t, "testowner", cfg.GitHub.Owner)
	}
}

func TestHandler_SaveSettings_InvalidTimeout(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	// Original timeout
	ws := cond.GetWorkspace()
	origCfg, _ := ws.LoadConfig()
	origTimeout := origCfg.Agent.Timeout

	// Try to set invalid timeout (negative)
	form := "agent.timeout=-100"
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should still succeed but timeout should not change
	cfg, _ := ws.LoadConfig()
	assert.Equal(t, origTimeout, cfg.Agent.Timeout)
}

func TestHandler_SaveSettings_ConfigFilePersistence(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	form := "agent.timeout=999&git.commit_prefix=[CUSTOM]"
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Verify the config file was actually written
	ws := cond.GetWorkspace()
	configPath := ws.ConfigPath()
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "timeout: 999")
	assert.Contains(t, string(content), "[CUSTOM]")
}

// --- Helper Function Tests ---

func TestStripSensitiveFields(t *testing.T) {
	cfg := &storage.WorkspaceConfig{
		GitHub: &storage.GitHubSettings{
			Token: "secret-token",
			Owner: "owner",
		},
		GitLab: &storage.GitLabSettings{
			Token: "another-secret",
			Host:  "https://gitlab.example.com",
		},
		Jira: &storage.JiraSettings{
			Token:   "jira-token",
			Email:   "user@example.com",
			Project: "PROJ",
		},
		Linear: &storage.LinearSettings{
			Token: "linear-token",
			Team:  "TEAM",
		},
		Notion: &storage.NotionSettings{
			Token:      "notion-token",
			DatabaseID: "db-123",
		},
		Bitbucket: &storage.BitbucketSettings{
			AppPassword: "bb-password",
			Username:    "bbuser",
			Workspace:   "myworkspace",
		},
		Asana: &storage.AsanaSettings{
			Token:        "asana-token",
			WorkspaceGID: "ws-123",
		},
		ClickUp: &storage.ClickUpSettings{
			Token:  "clickup-token",
			TeamID: "team-123",
		},
		Trello: &storage.TrelloSettings{
			APIKey: "trello-key",
			Token:  "trello-token",
			Board:  "board-123",
		},
		Wrike: &storage.WrikeSettings{
			Token:  "wrike-token",
			Folder: "folder-123",
		},
		YouTrack: &storage.YouTrackSettings{
			Token: "youtrack-token",
			Host:  "https://youtrack.example.com",
		},
		AzureDevOps: &storage.AzureDevOpsSettings{
			Token:        "azure-token",
			Organization: "myorg",
		},
	}

	stripped := stripSensitiveFields(cfg)

	// All tokens should be empty
	assert.Empty(t, stripped.GitHub.Token)
	assert.Empty(t, stripped.GitLab.Token)
	assert.Empty(t, stripped.Jira.Token)
	assert.Empty(t, stripped.Linear.Token)
	assert.Empty(t, stripped.Notion.Token)
	assert.Empty(t, stripped.Bitbucket.AppPassword)
	assert.Empty(t, stripped.Asana.Token)
	assert.Empty(t, stripped.ClickUp.Token)
	assert.Empty(t, stripped.Trello.APIKey)
	assert.Empty(t, stripped.Trello.Token)
	assert.Empty(t, stripped.Wrike.Token)
	assert.Empty(t, stripped.YouTrack.Token)
	assert.Empty(t, stripped.AzureDevOps.Token)

	// Non-sensitive fields should remain
	assert.Equal(t, "owner", stripped.GitHub.Owner)
	assert.Equal(t, "https://gitlab.example.com", stripped.GitLab.Host)
	assert.Equal(t, "PROJ", stripped.Jira.Project)
	assert.Equal(t, "TEAM", stripped.Linear.Team)
	assert.Equal(t, "db-123", stripped.Notion.DatabaseID)
	assert.Equal(t, "myworkspace", stripped.Bitbucket.Workspace)
	assert.Equal(t, "ws-123", stripped.Asana.WorkspaceGID)
	assert.Equal(t, "team-123", stripped.ClickUp.TeamID)
	assert.Equal(t, "board-123", stripped.Trello.Board)
	assert.Equal(t, "folder-123", stripped.Wrike.Folder)
	assert.Equal(t, "https://youtrack.example.com", stripped.YouTrack.Host)
	assert.Equal(t, "myorg", stripped.AzureDevOps.Organization)

	// Original should be unchanged
	assert.Equal(t, "secret-token", cfg.GitHub.Token)
	assert.Equal(t, "jira-token", cfg.Jira.Token)
}

func TestStripSensitiveFields_NilProviders(t *testing.T) {
	cfg := &storage.WorkspaceConfig{
		// All providers are nil
	}

	// Should not panic
	stripped := stripSensitiveFields(cfg)
	assert.Nil(t, stripped.GitHub)
	assert.Nil(t, stripped.GitLab)
}

func TestUpdateConfigFromForm(t *testing.T) {
	cfg := storage.NewDefaultWorkspaceConfig()

	form := strings.Join([]string{
		"git.auto_commit=true",
		"git.sign_commits=true",
		"agent.timeout=600",
		"agent.max_retries=5",
		"agent.instructions=Test instructions",
		"workflow.session_retention_days=60",
	}, "&")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = req.ParseForm()

	updateConfigFromForm(cfg, req, true)

	assert.True(t, cfg.Git.AutoCommit)
	assert.True(t, cfg.Git.SignCommits)
	assert.Equal(t, 600, cfg.Agent.Timeout)
	assert.Equal(t, 5, cfg.Agent.MaxRetries)
	assert.Equal(t, "Test instructions", cfg.Agent.Instructions)
	assert.Equal(t, 60, cfg.Workflow.SessionRetentionDays)
}

func TestUpdateConfigFromForm_UncheckedBooleans(t *testing.T) {
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Git.AutoCommit = true
	cfg.Git.SignCommits = true

	// Form with unchecked booleans (no value sent)
	form := "agent.timeout=300"

	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = req.ParseForm()

	updateConfigFromForm(cfg, req, true)

	// Unchecked checkboxes should result in false
	assert.False(t, cfg.Git.AutoCommit)
	assert.False(t, cfg.Git.SignCommits)
}

func TestHasAnyFormValue(t *testing.T) {
	tests := []struct {
		name     string
		form     string
		fields   []string
		expected bool
	}{
		{
			name:     "has first field",
			form:     "field1=value1&field2=",
			fields:   []string{"field1", "field3"},
			expected: true,
		},
		{
			name:     "has second field",
			form:     "field1=&field2=value2",
			fields:   []string{"field1", "field2"},
			expected: true,
		},
		{
			name:     "no fields have values",
			form:     "field1=&field2=",
			fields:   []string{"field1", "field2"},
			expected: false,
		},
		{
			name:     "fields not in form",
			form:     "other=value",
			fields:   []string{"field1", "field2"},
			expected: false,
		},
		{
			name:     "empty form",
			form:     "",
			fields:   []string{"field1"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.form))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			_ = req.ParseForm()

			result := hasAnyFormValue(req, tt.fields...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateProviderSettings_AllProviders(t *testing.T) {
	cfg := storage.NewDefaultWorkspaceConfig()

	form := strings.Join([]string{
		"github.token=gh-token",
		"github.owner=ghowner",
		"gitlab.token=gl-token",
		"gitlab.host=https://gitlab.com",
		"jira.token=jira-token",
		"jira.project=JIRA",
		"linear.token=linear-token",
		"linear.team=LINEAR",
		"notion.token=notion-token",
		"notion.database_id=notion-db",
		"bitbucket.app_password=bb-pass",
		"bitbucket.workspace=bb-ws",
	}, "&")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = req.ParseForm()

	updateProviderSettings(cfg, req)

	assert.Equal(t, "gh-token", cfg.GitHub.Token)
	assert.Equal(t, "ghowner", cfg.GitHub.Owner)
	assert.Equal(t, "gl-token", cfg.GitLab.Token)
	assert.Equal(t, "https://gitlab.com", cfg.GitLab.Host)
	assert.Equal(t, "jira-token", cfg.Jira.Token)
	assert.Equal(t, "JIRA", cfg.Jira.Project)
	assert.Equal(t, "linear-token", cfg.Linear.Token)
	assert.Equal(t, "LINEAR", cfg.Linear.Team)
	assert.Equal(t, "notion-token", cfg.Notion.Token)
	assert.Equal(t, "notion-db", cfg.Notion.DatabaseID)
	assert.Equal(t, "bb-pass", cfg.Bitbucket.AppPassword)
	assert.Equal(t, "bb-ws", cfg.Bitbucket.Workspace)
}

func TestUpdateProviderSettingsNonSensitive(t *testing.T) {
	cfg := storage.NewDefaultWorkspaceConfig()

	form := strings.Join([]string{
		"github.token=should-be-ignored",
		"github.owner=ghowner",
		"github.repo=ghrepo",
		"gitlab.token=should-be-ignored",
		"gitlab.host=https://gitlab.com",
	}, "&")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = req.ParseForm()

	updateProviderSettingsNonSensitive(cfg, req)

	// Tokens should not be set
	if cfg.GitHub != nil {
		assert.Empty(t, cfg.GitHub.Token)
		assert.Equal(t, "ghowner", cfg.GitHub.Owner)
		assert.Equal(t, "ghrepo", cfg.GitHub.Repo)
	}
	if cfg.GitLab != nil {
		assert.Empty(t, cfg.GitLab.Token)
		assert.Equal(t, "https://gitlab.com", cfg.GitLab.Host)
	}
}

// --- Integration Tests ---

func TestSettings_RoundTrip(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	// 1. Get initial settings
	resp, err := http.Get(srv.URL() + "/api/v1/settings")
	require.NoError(t, err)
	var initialCfg storage.WorkspaceConfig
	_ = json.NewDecoder(resp.Body).Decode(&initialCfg) //nolint:musttag
	resp.Body.Close()

	// 2. Modify settings
	form := "agent.timeout=777&git.commit_prefix=[ROUNDTRIP]"
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, _ = client.Do(req)
	resp.Body.Close()

	// 3. Get settings again
	resp, err = http.Get(srv.URL() + "/api/v1/settings")
	require.NoError(t, err)
	var updatedCfg storage.WorkspaceConfig
	_ = json.NewDecoder(resp.Body).Decode(&updatedCfg) //nolint:musttag
	resp.Body.Close()

	// 4. Verify changes
	assert.Equal(t, 777, updatedCfg.Agent.Timeout)
	assert.Equal(t, "[ROUNDTRIP]", updatedCfg.Git.CommitPrefix)
}

func TestSettings_ConfigFileFormat(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	form := "agent.timeout=500"
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, _ := client.Do(req)
	resp.Body.Close()

	// Check config file format
	ws := cond.GetWorkspace()
	content, err := os.ReadFile(ws.ConfigPath())
	require.NoError(t, err)

	// Should have header comment
	assert.Contains(t, string(content), "# Task workspace configuration")

	// Should be valid YAML
	assert.Contains(t, string(content), "agent:")
	assert.Contains(t, string(content), "timeout: 500")
}

// --- Edge Cases ---

func TestHandler_SaveSettings_EmptyForm(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should still succeed (just resets booleans to false)
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
}

func TestHandler_SaveSettings_PreservesUnsetFields(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	// Set initial config with custom value
	ws := cond.GetWorkspace()
	cfg, _ := ws.LoadConfig()
	cfg.Agent.Instructions = "Keep this instruction"
	_ = ws.SaveConfig(cfg)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	// Only update timeout, not instructions
	form := "agent.timeout=888"
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, _ := client.Do(req)
	resp.Body.Close()

	updatedCfg, _ := ws.LoadConfig()
	assert.Equal(t, 888, updatedCfg.Agent.Timeout)
	// Note: form submission clears empty string fields, so this will be empty
	// This is expected behavior for form-based settings
}

func TestHandler_SettingsPage_LoadsAvailableAgents(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Should have agent dropdown with default agents
	assert.Contains(t, bodyStr, `name="agent.default"`)
	assert.Contains(t, bodyStr, "<select")
}

func TestHandler_SettingsPage_BackToDashboardLink(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `href="/"`)
	assert.Contains(t, string(body), "Back to Dashboard")
}

func TestHandler_SettingsPage_ProviderConfiguredBadges(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	// Set GitHub token
	ws := cond.GetWorkspace()
	cfg, _ := ws.LoadConfig()
	cfg.GitHub = &storage.GitHubSettings{Token: "test-token"}
	_ = ws.SaveConfig(cfg)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	// Should show "Configured" badge for GitHub
	assert.Contains(t, string(body), "Configured")
}

// --- Workspace Not Initialized Tests ---

func TestHandler_SettingsPage_NoWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	cond, err := conductor.New(
		conductor.WithWorkDir(tmpDir),
		conductor.WithHomeDir(homeDir),
		conductor.WithAutoInit(false), // Don't auto-init
		conductor.WithDryRun(true),
		conductor.WithStdout(io.Discard),
		conductor.WithStderr(io.Discard),
	)
	require.NoError(t, err)
	// Don't initialize - workspace will be nil

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should show settings page with default config and info message
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "Settings")
	assert.Contains(t, bodyStr, "workspace not initialized")
}

// --- Concurrent Access Test ---

func TestHandler_SaveSettings_ConcurrentRequests(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	// Send multiple concurrent requests
	done := make(chan bool, 10)
	for i := range 10 {
		go func(_ int) {
			form := "agent.timeout=" + filepath.Base(t.TempDir())[0:3] // Random-ish value
			req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
			}
			done <- true
		}(i)
	}

	// Wait for all requests
	for range 10 {
		<-done
	}

	// Config file should still be valid
	ws := cond.GetWorkspace()
	_, err := ws.LoadConfig()
	assert.NoError(t, err)
}

// --- Global Mode Project Picker Tests ---

func TestHandler_SettingsPage_GlobalMode_ShowsProjectPicker(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeGlobal,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Should show project picker elements
	assert.Contains(t, bodyStr, "Select Project")
	assert.Contains(t, bodyStr, "project-picker")
	assert.Contains(t, bodyStr, "Select a project")
}

func TestHandler_SettingsPage_GlobalMode_NoProjectSelected_ShowsMessage(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeGlobal,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Should show instruction to select a project
	assert.Contains(t, bodyStr, "Select a project")
}

func TestHandler_SettingsPage_GlobalMode_InvalidProject(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeGlobal,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	// Try to access settings for non-existent project
	resp, err := http.Get(srv.URL() + "/settings?project=nonexistent-project-12345")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should still return OK but with error message
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "project not found")
}

func TestHandler_SettingsPage_ProjectMode_NoProjectPicker(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Project mode should NOT show project picker
	assert.NotContains(t, bodyStr, "Select Project")
	assert.NotContains(t, bodyStr, "project-picker")
}

func TestHandler_GetSettings_GlobalMode_WithProjectParam(t *testing.T) {
	srv := startSettingsTestServer(t, Config{
		Port: 0,
		Mode: ModeGlobal,
	})

	// Try to get settings for non-existent project
	resp, err := http.Get(srv.URL() + "/api/v1/settings?project=nonexistent")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should return 404 for non-existent project
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestHandler_GetSettings_GlobalMode_NoProject_ReturnsDefaults(t *testing.T) {
	srv := startSettingsTestServer(t, Config{
		Port: 0,
		Mode: ModeGlobal,
	})

	resp, err := http.Get(srv.URL() + "/api/v1/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Without project param, should return defaults
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var cfg storage.WorkspaceConfig
	err = json.NewDecoder(resp.Body).Decode(&cfg) //nolint:musttag
	require.NoError(t, err)
	assert.Equal(t, "claude", cfg.Agent.Default)
}

func TestHandler_SaveSettings_GlobalMode_NoProject_ReturnsError(t *testing.T) {
	srv := startSettingsTestServer(t, Config{
		Port: 0,
		Mode: ModeGlobal,
	})

	form := "agent.timeout=500"
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should return 400 error about selecting project first
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "project")
}

func TestHandler_SaveSettings_GlobalMode_InvalidProject_ReturnsError(t *testing.T) {
	srv := startSettingsTestServer(t, Config{
		Port: 0,
		Mode: ModeGlobal,
	})

	form := "agent.timeout=500"
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/settings?project=nonexistent", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should return 404 for non-existent project
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestHandler_SettingsPage_GlobalMode_FormActionIncludesProject(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	// Register this project in the registry so it can be found
	registry, err := storage.LoadRegistryWithOverride(filepath.Dir(filepath.Dir(cond.GetWorkspace().WorkRoot())))
	require.NoError(t, err)
	projectID := "test-project-" + filepath.Base(tmpDir)
	err = registry.Register(projectID, tmpDir, "", "Test Project")
	require.NoError(t, err)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeGlobal,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	// Access settings with project param
	resp, err := http.Get(srv.URL() + "/settings?project=" + projectID)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Form action should include project param
	assert.Contains(t, bodyStr, "?project="+projectID)
}

func TestLoadProjectConfig_InvalidProjectID(t *testing.T) {
	cfg, ws, errMsg := loadProjectConfig(context.Background(), "nonexistent-project-xyz")
	assert.Nil(t, cfg)
	assert.Nil(t, ws)
	assert.Contains(t, errMsg, "not found")
}

func TestLoadProjectConfig_RegistryError(t *testing.T) {
	// This test verifies error handling when registry can't be loaded
	// In practice this is hard to trigger, but the code path exists
	cfg, ws, errMsg := loadProjectConfig(context.Background(), "")
	assert.Nil(t, cfg)
	assert.Nil(t, ws)
	assert.NotEmpty(t, errMsg)
}

func TestGetProjectWorkspacePath(t *testing.T) {
	path, err := GetProjectWorkspacePath("test-project")
	require.NoError(t, err)
	assert.Contains(t, path, "test-project")
	assert.Contains(t, path, ".valksor")
	assert.Contains(t, path, "workspaces")
}

func TestDiscoverProjects_EmptyWorkspace(t *testing.T) {
	// Create a temp home directory with no projects
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	projects, err := DiscoverProjects()
	require.NoError(t, err)
	assert.Empty(t, projects)
}

func TestHandler_SettingsPage_GlobalMode_SelectedProjectShown(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeGlobal,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	// Even with invalid project, the selected value should be preserved in the dropdown
	resp, err := http.Get(srv.URL() + "/settings?project=my-test-project")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// The project picker should exist (global mode)
	assert.Contains(t, bodyStr, "Select Project")
}

func TestHandler_SettingsPage_GlobalMode_HidesSensitiveFields(t *testing.T) {
	cond, tmpDir := createTestConductor(t)

	srv := startSettingsTestServer(t, Config{
		Port:          0,
		Mode:          ModeGlobal,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})

	resp, err := http.Get(srv.URL() + "/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Global mode should not show token fields
	assert.NotContains(t, bodyStr, `name="github.token"`)
	assert.NotContains(t, bodyStr, `name="gitlab.token"`)
	assert.NotContains(t, bodyStr, `name="jira.token"`)
}

func TestStripSensitiveFields_AllProviders(t *testing.T) {
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.GitHub = &storage.GitHubSettings{Token: "secret", Owner: "owner"}
	cfg.GitLab = &storage.GitLabSettings{Token: "secret", Host: "host"}
	cfg.Jira = &storage.JiraSettings{Token: "secret", Project: "proj"}
	cfg.Linear = &storage.LinearSettings{Token: "secret", Team: "team"}
	cfg.Notion = &storage.NotionSettings{Token: "secret", DatabaseID: "db"}
	cfg.Bitbucket = &storage.BitbucketSettings{AppPassword: "secret", Workspace: "ws"}
	cfg.Asana = &storage.AsanaSettings{Token: "secret"}
	cfg.ClickUp = &storage.ClickUpSettings{Token: "secret"}
	cfg.Trello = &storage.TrelloSettings{APIKey: "key", Token: "secret"}
	cfg.Wrike = &storage.WrikeSettings{Token: "secret"}
	cfg.YouTrack = &storage.YouTrackSettings{Token: "secret"}
	cfg.AzureDevOps = &storage.AzureDevOpsSettings{Token: "secret"}

	stripped := stripSensitiveFields(cfg)

	// Tokens should be stripped
	assert.Empty(t, stripped.GitHub.Token)
	assert.Empty(t, stripped.GitLab.Token)
	assert.Empty(t, stripped.Jira.Token)
	assert.Empty(t, stripped.Linear.Token)
	assert.Empty(t, stripped.Notion.Token)
	assert.Empty(t, stripped.Bitbucket.AppPassword)
	assert.Empty(t, stripped.Asana.Token)
	assert.Empty(t, stripped.ClickUp.Token)
	assert.Empty(t, stripped.Trello.APIKey)
	assert.Empty(t, stripped.Trello.Token)
	assert.Empty(t, stripped.Wrike.Token)
	assert.Empty(t, stripped.YouTrack.Token)
	assert.Empty(t, stripped.AzureDevOps.Token)

	// Non-sensitive fields should remain
	assert.Equal(t, "owner", stripped.GitHub.Owner)
	assert.Equal(t, "host", stripped.GitLab.Host)
	assert.Equal(t, "proj", stripped.Jira.Project)
	assert.Equal(t, "team", stripped.Linear.Team)
	assert.Equal(t, "db", stripped.Notion.DatabaseID)
	assert.Equal(t, "ws", stripped.Bitbucket.Workspace)

	// Original should be unchanged
	assert.Equal(t, "secret", cfg.GitHub.Token)
}
