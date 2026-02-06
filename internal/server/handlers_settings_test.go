package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func newFormRequest(t *testing.T, form url.Values) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatalf("ParseForm failed: %v", err)
	}

	return req
}

func TestSettingsHelpers(t *testing.T) {
	if lines := parseTextareaLines(" a \n\nb\n  \n c "); len(lines) != 3 || lines[0] != "a" || lines[2] != "c" {
		t.Fatalf("parseTextareaLines = %#v", lines)
	}

	req := newFormRequest(t, url.Values{
		"x": []string{"1"},
	})
	if !hasAnyFormValue(req, "x", "y") {
		t.Fatalf("expected hasAnyFormValue true")
	}
	if hasAnyFormValue(req, "y", "z") {
		t.Fatalf("expected hasAnyFormValue false")
	}
}

func TestUpdateConfigFromFormSensitiveAndNonSensitive(t *testing.T) {
	form := url.Values{
		"project.code_dir":                           []string{"../code"},
		"git.auto_commit":                            []string{"true"},
		"git.sign_commits":                           []string{"true"},
		"agent.default":                              []string{"claude"},
		"agent.timeout":                              []string{"90"},
		"agent.max_retries":                          []string{"5"},
		"agent.optimize_prompts":                     []string{"true"},
		"agent.optimize_planning":                    []string{"true"},
		"workflow.auto_init":                         []string{"true"},
		"workflow.session_retention_days":            []string{"30"},
		"stack.auto_rebase":                          []string{"always"},
		"storage.save_in_project":                    []string{"true"},
		"storage.project_dir":                        []string{".mehrhof/work"},
		"specification.filename_pattern":             []string{"spec-{number}.md"},
		"review.filename_pattern":                    []string{"review-{number}.md"},
		"browser.enabled":                            []string{"true"},
		"browser.headless":                           []string{"true"},
		"browser.port":                               []string{"9222"},
		"browser.timeout":                            []string{"20"},
		"update.enabled":                             []string{"true"},
		"update.check_interval":                      []string{"48"},
		"budget.per_task.max_cost":                   []string{"12.5"},
		"budget.per_task.max_tokens":                 []string{"1000"},
		"budget.per_task.on_limit":                   []string{"stop"},
		"budget.per_task.warning_at":                 []string{"0.8"},
		"budget.monthly.max_cost":                    []string{"50"},
		"budget.monthly.warning_at":                  []string{"0.7"},
		"providers.default":                          []string{"github"},
		"github.token":                               []string{"gh-secret"},
		"github.owner":                               []string{"o"},
		"github.repo":                                []string{"r"},
		"gitlab.token":                               []string{"gl-secret"},
		"jira.token":                                 []string{"jira-secret"},
		"jira.base_url":                              []string{"https://jira.local"},
		"linear.token":                               []string{"lin-secret"},
		"notion.token":                               []string{"notion-secret"},
		"bitbucket.app_password":                     []string{"bb-secret"},
		"automation.enabled":                         []string{"true"},
		"automation.providers.github.enabled":        []string{"true"},
		"automation.providers.github.webhook_secret": []string{"gh-webhook"},
		"automation.providers.gitlab.enabled":        []string{"true"},
		"automation.providers.gitlab.webhook_secret": []string{"gl-webhook"},
		"automation.access_control.mode":             []string{"allowlist"},
		"automation.access_control.allowlist":        []string{"alice\nbob"},
		"automation.queue.max_concurrent":            []string{"3"},
	}

	req := newFormRequest(t, form)
	cfg := storage.NewDefaultWorkspaceConfig()
	updateConfigFromForm(cfg, req, true)

	if cfg.Project.CodeDir != "../code" || !cfg.Git.AutoCommit || cfg.Agent.Default != "claude" {
		t.Fatalf("core form mapping failed: %#v", cfg)
	}
	if cfg.Agent.Timeout != 90 || cfg.Agent.MaxRetries != 5 || !cfg.Agent.OptimizePrompts {
		t.Fatalf("agent mapping failed: %#v", cfg.Agent)
	}
	if cfg.Browser == nil || !cfg.Browser.Enabled || cfg.Browser.Port != 9222 {
		t.Fatalf("browser mapping failed: %#v", cfg.Browser)
	}
	if cfg.Budget.PerTask.MaxCost != 12.5 || cfg.Budget.PerTask.MaxTokens != 1000 {
		t.Fatalf("budget mapping failed: %#v", cfg.Budget.PerTask)
	}
	if cfg.GitHub == nil || cfg.GitHub.Token != "gh-secret" {
		t.Fatalf("sensitive provider fields should be set when allowed")
	}
	if cfg.Automation == nil || !cfg.Automation.Enabled {
		t.Fatalf("automation mapping failed: %#v", cfg.Automation)
	}
	if cfg.Automation.Providers["github"].WebhookSecret != "gh-webhook" {
		t.Fatalf("automation webhook secret not mapped")
	}
	if len(cfg.Automation.AccessControl.Allowlist) != 2 {
		t.Fatalf("allowlist parse failed: %#v", cfg.Automation.AccessControl.Allowlist)
	}

	// Non-sensitive update should not write token fields.
	cfg2 := storage.NewDefaultWorkspaceConfig()
	req2 := newFormRequest(t, form)
	updateConfigFromForm(cfg2, req2, false)
	if cfg2.GitHub != nil && cfg2.GitHub.Token != "" {
		t.Fatalf("github token should not be set in non-sensitive mode")
	}
	if cfg2.Automation.Providers["github"].WebhookSecret != "" {
		t.Fatalf("webhook secret should not be set in non-sensitive mode")
	}
	if cfg2.GitHub == nil || cfg2.GitHub.Owner != "o" {
		t.Fatalf("non-sensitive provider fields should still be set")
	}
}

func TestStripSensitiveFields(t *testing.T) {
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.GitHub = &storage.GitHubSettings{Token: "gh", Owner: "o"}
	cfg.GitLab = &storage.GitLabSettings{Token: "gl", Host: "h"}
	cfg.Jira = &storage.JiraSettings{Token: "j", BaseURL: "u"}
	cfg.Linear = &storage.LinearSettings{Token: "l", Team: "T"}
	cfg.Notion = &storage.NotionSettings{Token: "n", DatabaseID: "db"}
	cfg.Bitbucket = &storage.BitbucketSettings{AppPassword: "bb", Workspace: "ws"}
	cfg.Asana = &storage.AsanaSettings{Token: "as"}
	cfg.ClickUp = &storage.ClickUpSettings{Token: "cu"}
	cfg.Trello = &storage.TrelloSettings{APIKey: "tk", Token: "tt"}
	cfg.Wrike = &storage.WrikeSettings{Token: "wr"}
	cfg.YouTrack = &storage.YouTrackSettings{Token: "yt"}
	cfg.AzureDevOps = &storage.AzureDevOpsSettings{Token: "az"}

	stripped := stripSensitiveFields(cfg)
	if stripped.GitHub.Token != "" || stripped.GitLab.Token != "" || stripped.Jira.Token != "" {
		t.Fatalf("expected sensitive tokens to be stripped")
	}
	if stripped.Bitbucket.AppPassword != "" || stripped.Trello.Token != "" || stripped.AzureDevOps.Token != "" {
		t.Fatalf("expected all sensitive fields stripped")
	}
	if cfg.GitHub.Token != "gh" {
		t.Fatalf("original config should remain unchanged")
	}
}

func TestHandleGetSettingsDefaultsGlobal(t *testing.T) {
	srv, err := New(Config{
		Mode: ModeGlobal,
	})
	if err != nil {
		t.Fatalf("New server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rr := httptest.NewRecorder()
	srv.handleGetSettings(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var cfg storage.WorkspaceConfig
	if err := json.NewDecoder(strings.NewReader(rr.Body.String())).Decode(&cfg); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// Global mode should return a config object and never expose tokens.
	if cfg.GitHub != nil && cfg.GitHub.Token != "" {
		t.Fatalf("global settings should not expose github token")
	}
}
