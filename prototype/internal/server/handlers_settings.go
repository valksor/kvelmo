package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/coordination"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// handleSettingsPage renders the settings page.
func (s *Server) handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil {
		s.writeError(w, http.StatusInternalServerError, "renderer not loaded")

		return
	}

	var cfg *storage.WorkspaceConfig
	var loadErr string
	var projects []storage.ProjectMetadata
	var workspaceRoot string // Track workspace root for language detection
	selectedProject := r.URL.Query().Get("project")

	// Global mode: need to handle project selection
	if s.config.Mode == ModeGlobal {
		// Get list of registered projects (from projects.yaml, same as dashboard)
		if registry, err := storage.LoadRegistry(); err == nil {
			projects = registry.List()
		} else {
			loadErr = "failed to load project registry: " + err.Error()
		}

		if selectedProject != "" {
			// Load selected project's config
			var ws *storage.Workspace
			cfg, ws, loadErr = loadProjectConfig(r.Context(), selectedProject)
			if cfg == nil {
				cfg = storage.NewDefaultWorkspaceConfig()
			}
			if ws != nil {
				workspaceRoot = ws.CodeRoot()
			}
		} else if len(projects) > 0 {
			// No project selected, show picker with message
			loadErr = "Select a project to view its settings"
			cfg = storage.NewDefaultWorkspaceConfig()
		} else if loadErr == "" {
			loadErr = "No projects found"
			cfg = storage.NewDefaultWorkspaceConfig()
		} else {
			cfg = storage.NewDefaultWorkspaceConfig()
		}
	} else {
		// Project mode: use conductor's workspace
		if s.config.Conductor != nil {
			ws := s.config.Conductor.GetWorkspace()
			if ws != nil {
				workspaceRoot = ws.CodeRoot()
				var err error
				cfg, err = ws.LoadConfig()
				if err != nil {
					loadErr = "failed to load config: " + err.Error()
					cfg = storage.NewDefaultWorkspaceConfig()
				} else if !ws.HasConfig() {
					// Workspace opened but no config file exists yet
					loadErr = "workspace not initialized - showing default settings"
				}
			} else {
				loadErr = "workspace not initialized - showing default settings"
				cfg = storage.NewDefaultWorkspaceConfig()
			}
		} else {
			loadErr = "workspace not initialized - showing default settings"
			cfg = storage.NewDefaultWorkspaceConfig()
		}
	}

	// Get available agents
	var agents []string
	if s.config.Conductor != nil {
		registry := s.config.Conductor.GetAgentRegistry()
		if registry != nil {
			agents = registry.List()
		}
	}
	if len(agents) == 0 {
		agents = []string{"claude", "gemini", "ollama"}
	}

	// Combine any load error with query param error
	errorMsg := r.URL.Query().Get("error")
	if loadErr != "" && errorMsg == "" {
		errorMsg = loadErr
	}

	pageData := views.ComputePageData(
		s.modeString(),
		s.config.Mode == ModeGlobal,
		s.config.AuthStore != nil,
		s.canSwitchProject(),
		s.isViewer(r),
		s.getCurrentUser(r),
	)
	pageData.Success = r.URL.Query().Get("success")
	pageData.Error = errorMsg

	// Compute project info for security scanner detection
	var projectInfo *views.ProjectInfoData
	if workspaceRoot != "" {
		projectInfo = views.ComputeProjectInfo(workspaceRoot)
	}

	data := views.SettingsData{
		PageData:        pageData,
		ShowSensitive:   isLocalRequest(r), // Only show tokens when accessed locally
		Config:          cfg,
		Agents:          agents,
		Projects:        views.ComputeSettingsProjects(projects),
		SelectedProject: selectedProject,
		SandboxStatus:   s.getSandboxStatus(),
		ProjectInfo:     projectInfo,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderSettings(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// loadProjectConfig loads config for a specific project by ID.
// Returns the config and workspace, or an error message.
func loadProjectConfig(ctx context.Context, projectID string) (*storage.WorkspaceConfig, *storage.Workspace, string) {
	// Load project registry to get the project's repo path
	registry, err := storage.LoadRegistry()
	if err != nil {
		return nil, nil, "failed to load project registry: " + err.Error()
	}

	project, ok := registry.Projects[projectID]
	if !ok {
		return nil, nil, "project not found in registry"
	}

	// Open workspace using the project's repo path
	ws, err := storage.OpenWorkspace(ctx, project.Path, nil)
	if err != nil {
		return nil, nil, "failed to open workspace: " + err.Error()
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return nil, nil, "failed to load config: " + err.Error()
	}

	return cfg, ws, ""
}

// handleGetSettings returns the current config as JSON.
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	var cfg *storage.WorkspaceConfig
	selectedProject := r.URL.Query().Get("project")

	// Global mode with project selection
	if s.config.Mode == ModeGlobal && selectedProject != "" {
		var loadErr string
		cfg, _, loadErr = loadProjectConfig(r.Context(), selectedProject)
		if cfg == nil {
			s.writeError(w, http.StatusNotFound, "project not found: "+loadErr)

			return
		}
	} else if s.config.Conductor != nil {
		ws := s.config.Conductor.GetWorkspace()
		if ws != nil {
			var err error
			cfg, err = ws.LoadConfig()
			if err != nil {
				// Fall back to defaults on error
				cfg = storage.NewDefaultWorkspaceConfig()
			}
		} else {
			cfg = storage.NewDefaultWorkspaceConfig()
		}
	} else {
		cfg = storage.NewDefaultWorkspaceConfig()
	}

	// In global mode, strip sensitive fields
	if s.config.Mode == ModeGlobal {
		cfg = stripSensitiveFields(cfg)
	}

	s.writeJSON(w, http.StatusOK, cfg)
}

// handleSaveSettings saves the settings from form submission.
func (s *Server) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify settings")

		return
	}

	selectedProject := r.URL.Query().Get("project")
	contentType := r.Header.Get("Content-Type")

	var ws *storage.Workspace
	var cfg *storage.WorkspaceConfig
	var err error

	// Global mode with project selection
	if s.config.Mode == ModeGlobal && selectedProject != "" {
		var loadErr string
		_, ws, loadErr = loadProjectConfig(r.Context(), selectedProject)
		if ws == nil {
			s.writeError(w, http.StatusNotFound, "project not found: "+loadErr)

			return
		}
	} else if s.config.Conductor != nil {
		ws = s.config.Conductor.GetWorkspace()
	}

	if ws == nil {
		if s.config.Mode == ModeGlobal {
			s.writeError(w, http.StatusBadRequest, "select a project first")

			return
		}
		// Project mode: open workspace directly using WorkspaceRoot
		// This allows saving settings even when workspace isn't initialized yet
		if s.config.WorkspaceRoot == "" {
			s.writeError(w, http.StatusServiceUnavailable, "workspace root not configured")

			return
		}
		var err error
		ws, err = storage.OpenWorkspace(r.Context(), s.config.WorkspaceRoot, nil)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to open workspace: "+err.Error())

			return
		}
	}

	// Load existing config to merge with, or use defaults if not initialized
	cfg, err = ws.LoadConfig()
	if err != nil {
		// If config doesn't exist, use defaults (expected for uninitialized workspace)
		cfg = storage.NewDefaultWorkspaceConfig()
	}

	if contentType == "application/json" {
		// JSON submission - decode directly
		//nolint:musttag // WorkspaceConfig uses yaml tags which json pkg uses as field names
		if err := json.NewDecoder(r.Body).Decode(cfg); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())

			return
		}
	} else {
		// Form submission - parse form data
		if err := r.ParseForm(); err != nil {
			s.writeError(w, http.StatusBadRequest, "failed to parse form: "+err.Error())

			return
		}

		// Update config from form values (no sensitive fields in global mode)
		updateConfigFromForm(cfg, r, s.config.Mode == ModeProject)
	}

	// Save config
	if err := ws.SaveConfig(cfg); err != nil {
		// For HTMX form submission, redirect with error
		if contentType != "application/json" {
			redirectURL := "/settings?error=" + err.Error()
			if selectedProject != "" {
				redirectURL += "&project=" + selectedProject
			}
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)

			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())

		return
	}

	// Reinitialize conductor to pick up the newly created/updated workspace
	if s.config.Conductor != nil {
		if err := s.config.Conductor.Initialize(r.Context()); err != nil {
			// Log but don't fail - settings were saved, conductor refresh is secondary
			slog.Warn("failed to reinitialize conductor after saving settings", "error", err)
		}
	}

	// For HTMX form submission, redirect with success
	if contentType != "application/json" {
		redirectURL := "/settings?success=Settings+saved+successfully"
		if selectedProject != "" {
			redirectURL += "&project=" + selectedProject
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "Settings saved successfully",
	})
}

// updateConfigFromForm updates config from form values.
func updateConfigFromForm(cfg *storage.WorkspaceConfig, r *http.Request, allowSensitive bool) {
	// Project settings
	cfg.Project.CodeDir = r.FormValue("project.code_dir")

	// Git settings
	cfg.Git.AutoCommit = r.FormValue("git.auto_commit") == "true"
	cfg.Git.SignCommits = r.FormValue("git.sign_commits") == "true"
	cfg.Git.StashOnStart = r.FormValue("git.stash_on_start") == "true"
	cfg.Git.AutoPopStash = r.FormValue("git.auto_pop_stash") == "true"
	if v := r.FormValue("git.commit_prefix"); v != "" {
		cfg.Git.CommitPrefix = v
	}
	if v := r.FormValue("git.branch_pattern"); v != "" {
		cfg.Git.BranchPattern = v
	}

	// Agent settings
	if v := r.FormValue("agent.default"); v != "" {
		cfg.Agent.Default = v
	}
	if v := r.FormValue("agent.timeout"); v != "" {
		if timeout, err := strconv.Atoi(v); err == nil && timeout > 0 {
			cfg.Agent.Timeout = timeout
		}
	}
	if v := r.FormValue("agent.max_retries"); v != "" {
		if retries, err := strconv.Atoi(v); err == nil && retries >= 0 {
			cfg.Agent.MaxRetries = retries
		}
	}
	cfg.Agent.Instructions = r.FormValue("agent.instructions")

	// Agent settings - prompt optimization
	cfg.Agent.OptimizePrompts = r.FormValue("agent.optimize_prompts") == "true"

	// Per-step optimization settings
	if cfg.Agent.Steps == nil {
		cfg.Agent.Steps = make(map[string]storage.StepAgentConfig)
	}

	// Update or create per-step config for optimization
	for _, step := range []string{"planning", "implementing", "reviewing"} {
		formKey := "agent.optimize_" + step
		if r.FormValue(formKey) == "true" {
			stepCfg := cfg.Agent.Steps[step]
			stepCfg.OptimizePrompts = true
			cfg.Agent.Steps[step] = stepCfg
		} else if r.FormValue(formKey) == "" {
			// Checkbox was unchecked - explicitly set to false
			stepCfg := cfg.Agent.Steps[step]
			stepCfg.OptimizePrompts = false
			cfg.Agent.Steps[step] = stepCfg
		}
	}

	// Workflow settings
	cfg.Workflow.AutoInit = r.FormValue("workflow.auto_init") == "true"
	cfg.Workflow.DeleteWorkOnFinish = r.FormValue("workflow.delete_work_on_finish") == "true"
	cfg.Workflow.DeleteWorkOnAbandon = r.FormValue("workflow.delete_work_on_abandon") == "true"
	if v := r.FormValue("workflow.session_retention_days"); v != "" {
		if days, err := strconv.Atoi(v); err == nil && days > 0 {
			cfg.Workflow.SessionRetentionDays = days
		}
	}

	// Specification storage settings
	cfg.Specification.SaveInProject = r.FormValue("specification.save_in_project") == "true"
	if v := r.FormValue("specification.project_dir"); v != "" {
		cfg.Specification.ProjectDir = v
	}
	if v := r.FormValue("specification.filename_pattern"); v != "" {
		cfg.Specification.FilenamePattern = v
	}

	// Review storage settings
	cfg.Review.SaveInProject = r.FormValue("review.save_in_project") == "true"
	if v := r.FormValue("review.filename_pattern"); v != "" {
		cfg.Review.FilenamePattern = v
	}

	// Browser settings
	browserEnabled := r.FormValue("browser.enabled") == "true"
	if browserEnabled || cfg.Browser != nil {
		if cfg.Browser == nil {
			cfg.Browser = &storage.BrowserSettings{}
		}
		cfg.Browser.Enabled = browserEnabled
		cfg.Browser.Headless = r.FormValue("browser.headless") == "true"
		if v := r.FormValue("browser.port"); v != "" {
			if port, err := strconv.Atoi(v); err == nil && port >= 0 {
				cfg.Browser.Port = port
			}
		}
		if v := r.FormValue("browser.timeout"); v != "" {
			if timeout, err := strconv.Atoi(v); err == nil && timeout > 0 {
				cfg.Browser.Timeout = timeout
			}
		}
		if v := r.FormValue("browser.screenshot_dir"); v != "" {
			cfg.Browser.ScreenshotDir = v
		}
		if v := r.FormValue("browser.cookie_profile"); v != "" {
			cfg.Browser.CookieProfile = v
		}
	}

	// Update settings
	cfg.Update.Enabled = r.FormValue("update.enabled") == "true"
	if v := r.FormValue("update.check_interval"); v != "" {
		if interval, err := strconv.Atoi(v); err == nil && interval > 0 {
			cfg.Update.CheckInterval = interval
		}
	}

	// Budget settings - per task
	if v := r.FormValue("budget.per_task.max_cost"); v != "" {
		if cost, err := strconv.ParseFloat(v, 64); err == nil && cost >= 0 {
			cfg.Budget.PerTask.MaxCost = cost
		}
	}
	if v := r.FormValue("budget.per_task.max_tokens"); v != "" {
		if tokens, err := strconv.Atoi(v); err == nil && tokens >= 0 {
			cfg.Budget.PerTask.MaxTokens = tokens
		}
	}
	if v := r.FormValue("budget.per_task.on_limit"); v != "" {
		cfg.Budget.PerTask.OnLimit = v
	}
	if v := r.FormValue("budget.per_task.warning_at"); v != "" {
		if pct, err := strconv.ParseFloat(v, 64); err == nil && pct >= 0 && pct <= 1 {
			cfg.Budget.PerTask.WarningAt = pct
		}
	}

	// Budget settings - monthly
	if v := r.FormValue("budget.monthly.max_cost"); v != "" {
		if cost, err := strconv.ParseFloat(v, 64); err == nil && cost >= 0 {
			cfg.Budget.Monthly.MaxCost = cost
		}
	}
	if v := r.FormValue("budget.monthly.warning_at"); v != "" {
		if pct, err := strconv.ParseFloat(v, 64); err == nil && pct >= 0 && pct <= 1 {
			cfg.Budget.Monthly.WarningAt = pct
		}
	}

	// Provider default
	if v := r.FormValue("providers.default"); v != "" {
		cfg.Providers.Default = v
	}

	// Provider-specific settings (only if sensitive fields allowed)
	if allowSensitive {
		updateProviderSettings(cfg, r)
	} else {
		// Still allow non-sensitive provider settings
		updateProviderSettingsNonSensitive(cfg, r)
	}

	// Automation settings
	updateAutomationSettings(cfg, r, allowSensitive)
}

// updateProviderSettings updates provider settings including tokens.
func updateProviderSettings(cfg *storage.WorkspaceConfig, r *http.Request) {
	// GitHub
	if hasAnyFormValue(r, "github.token", "github.owner", "github.repo", "github.target_branch") {
		if cfg.GitHub == nil {
			cfg.GitHub = &storage.GitHubSettings{}
		}
		if v := r.FormValue("github.token"); v != "" {
			cfg.GitHub.Token = v
		}
		if v := r.FormValue("github.owner"); v != "" {
			cfg.GitHub.Owner = v
		}
		if v := r.FormValue("github.repo"); v != "" {
			cfg.GitHub.Repo = v
		}
		if v := r.FormValue("github.target_branch"); v != "" {
			cfg.GitHub.TargetBranch = v
		}
		cfg.GitHub.DraftPR = r.FormValue("github.draft_pr") == "true"
	}

	// GitLab
	if hasAnyFormValue(r, "gitlab.token", "gitlab.host", "gitlab.project_path") {
		if cfg.GitLab == nil {
			cfg.GitLab = &storage.GitLabSettings{}
		}
		if v := r.FormValue("gitlab.token"); v != "" {
			cfg.GitLab.Token = v
		}
		if v := r.FormValue("gitlab.host"); v != "" {
			cfg.GitLab.Host = v
		}
		if v := r.FormValue("gitlab.project_path"); v != "" {
			cfg.GitLab.ProjectPath = v
		}
	}

	// Jira
	if hasAnyFormValue(r, "jira.token", "jira.email", "jira.base_url", "jira.project") {
		if cfg.Jira == nil {
			cfg.Jira = &storage.JiraSettings{}
		}
		if v := r.FormValue("jira.token"); v != "" {
			cfg.Jira.Token = v
		}
		if v := r.FormValue("jira.email"); v != "" {
			cfg.Jira.Email = v
		}
		if v := r.FormValue("jira.base_url"); v != "" {
			cfg.Jira.BaseURL = v
		}
		if v := r.FormValue("jira.project"); v != "" {
			cfg.Jira.Project = v
		}
	}

	// Linear
	if hasAnyFormValue(r, "linear.token", "linear.team") {
		if cfg.Linear == nil {
			cfg.Linear = &storage.LinearSettings{}
		}
		if v := r.FormValue("linear.token"); v != "" {
			cfg.Linear.Token = v
		}
		if v := r.FormValue("linear.team"); v != "" {
			cfg.Linear.Team = v
		}
	}

	// Notion
	if hasAnyFormValue(r, "notion.token", "notion.database_id", "notion.status_property") {
		if cfg.Notion == nil {
			cfg.Notion = &storage.NotionSettings{}
		}
		if v := r.FormValue("notion.token"); v != "" {
			cfg.Notion.Token = v
		}
		if v := r.FormValue("notion.database_id"); v != "" {
			cfg.Notion.DatabaseID = v
		}
		if v := r.FormValue("notion.status_property"); v != "" {
			cfg.Notion.StatusProperty = v
		}
	}

	// Bitbucket
	if hasAnyFormValue(r, "bitbucket.username", "bitbucket.app_password", "bitbucket.workspace", "bitbucket.repo") {
		if cfg.Bitbucket == nil {
			cfg.Bitbucket = &storage.BitbucketSettings{}
		}
		if v := r.FormValue("bitbucket.username"); v != "" {
			cfg.Bitbucket.Username = v
		}
		if v := r.FormValue("bitbucket.app_password"); v != "" {
			cfg.Bitbucket.AppPassword = v
		}
		if v := r.FormValue("bitbucket.workspace"); v != "" {
			cfg.Bitbucket.Workspace = v
		}
		if v := r.FormValue("bitbucket.repo"); v != "" {
			cfg.Bitbucket.RepoSlug = v
		}
	}
}

// updateProviderSettingsNonSensitive updates only non-sensitive provider settings.
func updateProviderSettingsNonSensitive(cfg *storage.WorkspaceConfig, r *http.Request) {
	// GitHub (non-sensitive only)
	if hasAnyFormValue(r, "github.owner", "github.repo", "github.target_branch") {
		if cfg.GitHub == nil {
			cfg.GitHub = &storage.GitHubSettings{}
		}
		if v := r.FormValue("github.owner"); v != "" {
			cfg.GitHub.Owner = v
		}
		if v := r.FormValue("github.repo"); v != "" {
			cfg.GitHub.Repo = v
		}
		if v := r.FormValue("github.target_branch"); v != "" {
			cfg.GitHub.TargetBranch = v
		}
		cfg.GitHub.DraftPR = r.FormValue("github.draft_pr") == "true"
	}

	// GitLab (non-sensitive only)
	if hasAnyFormValue(r, "gitlab.host", "gitlab.project_path") {
		if cfg.GitLab == nil {
			cfg.GitLab = &storage.GitLabSettings{}
		}
		if v := r.FormValue("gitlab.host"); v != "" {
			cfg.GitLab.Host = v
		}
		if v := r.FormValue("gitlab.project_path"); v != "" {
			cfg.GitLab.ProjectPath = v
		}
	}

	// Jira (non-sensitive only)
	if hasAnyFormValue(r, "jira.base_url", "jira.project") {
		if cfg.Jira == nil {
			cfg.Jira = &storage.JiraSettings{}
		}
		if v := r.FormValue("jira.base_url"); v != "" {
			cfg.Jira.BaseURL = v
		}
		if v := r.FormValue("jira.project"); v != "" {
			cfg.Jira.Project = v
		}
	}

	// Linear (non-sensitive only)
	if v := r.FormValue("linear.team"); v != "" {
		if cfg.Linear == nil {
			cfg.Linear = &storage.LinearSettings{}
		}
		cfg.Linear.Team = v
	}

	// Notion (non-sensitive only)
	if hasAnyFormValue(r, "notion.database_id", "notion.status_property") {
		if cfg.Notion == nil {
			cfg.Notion = &storage.NotionSettings{}
		}
		if v := r.FormValue("notion.database_id"); v != "" {
			cfg.Notion.DatabaseID = v
		}
		if v := r.FormValue("notion.status_property"); v != "" {
			cfg.Notion.StatusProperty = v
		}
	}

	// Bitbucket (non-sensitive only)
	if hasAnyFormValue(r, "bitbucket.workspace", "bitbucket.repo") {
		if cfg.Bitbucket == nil {
			cfg.Bitbucket = &storage.BitbucketSettings{}
		}
		if v := r.FormValue("bitbucket.workspace"); v != "" {
			cfg.Bitbucket.Workspace = v
		}
		if v := r.FormValue("bitbucket.repo"); v != "" {
			cfg.Bitbucket.RepoSlug = v
		}
	}
}

// hasAnyFormValue checks if any of the given form fields have a non-empty value.
func hasAnyFormValue(r *http.Request, fields ...string) bool {
	for _, field := range fields {
		if r.FormValue(field) != "" {
			return true
		}
	}

	return false
}

// updateAutomationSettings updates automation settings from form values.
func updateAutomationSettings(cfg *storage.WorkspaceConfig, r *http.Request, allowSensitive bool) {
	// Ensure automation is initialized
	if cfg.Automation == nil {
		cfg.Automation = &storage.AutomationSettings{
			Providers: make(map[string]storage.ProviderAutoConfig),
		}
	}
	if cfg.Automation.Providers == nil {
		cfg.Automation.Providers = make(map[string]storage.ProviderAutoConfig)
	}

	// Master enable
	cfg.Automation.Enabled = r.FormValue("automation.enabled") == "true"

	// GitHub provider
	github := cfg.Automation.Providers["github"]
	github.Enabled = r.FormValue("automation.providers.github.enabled") == "true"
	if allowSensitive {
		if v := r.FormValue("automation.providers.github.webhook_secret"); v != "" {
			github.WebhookSecret = v
		}
	}
	if v := r.FormValue("automation.providers.github.command_prefix"); v != "" {
		github.CommandPrefix = v
	}
	github.UseWorktrees = r.FormValue("automation.providers.github.use_worktrees") == "true"
	github.DryRun = r.FormValue("automation.providers.github.dry_run") == "true"
	github.TriggerOn.IssueOpened = r.FormValue("automation.providers.github.trigger_on.issue_opened") == "true"
	github.TriggerOn.PROpened = r.FormValue("automation.providers.github.trigger_on.pr_opened") == "true"
	github.TriggerOn.PRUpdated = r.FormValue("automation.providers.github.trigger_on.pr_updated") == "true"
	github.TriggerOn.CommentCommands = r.FormValue("automation.providers.github.trigger_on.comment_commands") == "true"
	cfg.Automation.Providers["github"] = github

	// GitLab provider
	gitlab := cfg.Automation.Providers["gitlab"]
	gitlab.Enabled = r.FormValue("automation.providers.gitlab.enabled") == "true"
	if allowSensitive {
		if v := r.FormValue("automation.providers.gitlab.webhook_secret"); v != "" {
			gitlab.WebhookSecret = v
		}
	}
	if v := r.FormValue("automation.providers.gitlab.command_prefix"); v != "" {
		gitlab.CommandPrefix = v
	}
	gitlab.UseWorktrees = r.FormValue("automation.providers.gitlab.use_worktrees") == "true"
	gitlab.DryRun = r.FormValue("automation.providers.gitlab.dry_run") == "true"
	gitlab.TriggerOn.IssueOpened = r.FormValue("automation.providers.gitlab.trigger_on.issue_opened") == "true"
	gitlab.TriggerOn.MROpened = r.FormValue("automation.providers.gitlab.trigger_on.mr_opened") == "true"
	gitlab.TriggerOn.MRUpdated = r.FormValue("automation.providers.gitlab.trigger_on.mr_updated") == "true"
	gitlab.TriggerOn.CommentCommands = r.FormValue("automation.providers.gitlab.trigger_on.comment_commands") == "true"
	cfg.Automation.Providers["gitlab"] = gitlab

	// Access control
	if v := r.FormValue("automation.access_control.mode"); v != "" {
		cfg.Automation.AccessControl.Mode = v
	}
	cfg.Automation.AccessControl.AllowBots = r.FormValue("automation.access_control.allow_bots") == "true"
	cfg.Automation.AccessControl.RequireOrg = r.FormValue("automation.access_control.require_org") == "true"

	// Allowlist/blocklist from textarea (newline-separated)
	if v := r.FormValue("automation.access_control.allowlist"); v != "" {
		cfg.Automation.AccessControl.Allowlist = parseTextareaLines(v)
	} else {
		cfg.Automation.AccessControl.Allowlist = nil
	}
	if v := r.FormValue("automation.access_control.blocklist"); v != "" {
		cfg.Automation.AccessControl.Blocklist = parseTextareaLines(v)
	} else {
		cfg.Automation.AccessControl.Blocklist = nil
	}

	// Queue settings
	if v := r.FormValue("automation.queue.max_concurrent"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Automation.Queue.MaxConcurrent = n
		}
	}
	if v := r.FormValue("automation.queue.job_timeout"); v != "" {
		cfg.Automation.Queue.JobTimeout = v
	}
	if v := r.FormValue("automation.queue.retry_attempts"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.Automation.Queue.RetryAttempts = n
		}
	}
	if v := r.FormValue("automation.queue.retry_delay"); v != "" {
		cfg.Automation.Queue.RetryDelay = v
	}

	// Labels
	if v := r.FormValue("automation.labels.mehr_generated"); v != "" {
		cfg.Automation.Labels.MehrhofGenerated = v
	}
	if v := r.FormValue("automation.labels.in_progress"); v != "" {
		cfg.Automation.Labels.InProgress = v
	}
	if v := r.FormValue("automation.labels.failed"); v != "" {
		cfg.Automation.Labels.Failed = v
	}
	if v := r.FormValue("automation.labels.skip_review"); v != "" {
		cfg.Automation.Labels.SkipReview = v
	}
}

// parseTextareaLines splits textarea content by newlines and trims whitespace.
func parseTextareaLines(s string) []string {
	lines := strings.Split(s, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}

	return result
}

// stripSensitiveFields returns a copy of config with sensitive fields cleared.
func stripSensitiveFields(cfg *storage.WorkspaceConfig) *storage.WorkspaceConfig {
	// Create a shallow copy
	result := *cfg

	// Strip provider tokens
	if result.GitHub != nil {
		gh := *result.GitHub
		gh.Token = ""
		result.GitHub = &gh
	}
	if result.GitLab != nil {
		gl := *result.GitLab
		gl.Token = ""
		result.GitLab = &gl
	}
	if result.Jira != nil {
		j := *result.Jira
		j.Token = ""
		result.Jira = &j
	}
	if result.Linear != nil {
		l := *result.Linear
		l.Token = ""
		result.Linear = &l
	}
	if result.Notion != nil {
		n := *result.Notion
		n.Token = ""
		result.Notion = &n
	}
	if result.Bitbucket != nil {
		bb := *result.Bitbucket
		bb.AppPassword = ""
		result.Bitbucket = &bb
	}
	if result.Asana != nil {
		a := *result.Asana
		a.Token = ""
		result.Asana = &a
	}
	if result.ClickUp != nil {
		c := *result.ClickUp
		c.Token = ""
		result.ClickUp = &c
	}
	if result.Trello != nil {
		t := *result.Trello
		t.APIKey = ""
		t.Token = ""
		result.Trello = &t
	}
	if result.Wrike != nil {
		w := *result.Wrike
		w.Token = ""
		result.Wrike = &w
	}
	if result.YouTrack != nil {
		y := *result.YouTrack
		y.Token = ""
		result.YouTrack = &y
	}
	if result.AzureDevOps != nil {
		a := *result.AzureDevOps
		a.Token = ""
		result.AzureDevOps = &a
	}

	return &result
}

// handleConfigExplain returns agent configuration explanation as JSON.
func (s *Server) handleConfigExplain(w http.ResponseWriter, r *http.Request) {
	step := r.URL.Query().Get("step")
	if step == "" {
		s.writeError(w, http.StatusBadRequest, "missing step parameter (planning, implementing, reviewing)")

		return
	}

	// Map step string to workflow.Step
	var workflowStep workflow.Step
	switch step {
	case "planning":
		workflowStep = workflow.StepPlanning
	case "implementing", "implementation":
		workflowStep = workflow.StepImplementing
	case "reviewing", "review":
		workflowStep = workflow.StepReviewing
	default:
		s.writeError(w, http.StatusBadRequest, "invalid step: "+step+" (must be planning, implementing, or reviewing)")

		return
	}

	// Get conductor and workspace
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Load config
	cfg, err := ws.LoadConfig()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to load config: "+err.Error())

		return
	}

	// Get agent registry
	agents := s.config.Conductor.GetAgentRegistry()
	if agents == nil {
		s.writeError(w, http.StatusServiceUnavailable, "agent registry not available")

		return
	}

	// Create resolver
	resolver := coordination.NewResolver(agents, ws)

	// Build resolution request (no CLI flags, no task config)
	req := coordination.ResolveRequest{
		WorkspaceCfg: cfg,
		Step:         workflowStep,
	}

	// Get explanation
	explanation, err := resolver.ExplainAgentResolution(r.Context(), req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to explain resolution: "+err.Error())

		return
	}

	// Convert to JSON-friendly format
	type jsonResolutionStep struct {
		Priority int    `json:"priority"`
		Source   string `json:"source"`
		Agent    string `json:"agent"`
		Skipped  bool   `json:"skipped"`
	}

	steps := make([]jsonResolutionStep, len(explanation.AllSteps))
	for i, step := range explanation.AllSteps {
		steps[i] = jsonResolutionStep{
			Priority: step.Priority,
			Source:   step.Source,
			Agent:    step.Agent,
			Skipped:  step.Skipped,
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"step":      explanation.Step,
		"effective": explanation.Effective,
		"source":    explanation.Source,
		"steps":     steps,
	})
}

// handleProviderHealth returns provider health information as JSON.
func (s *Server) handleProviderHealth(w http.ResponseWriter, r *http.Request) {
	// Get conductor and workspace
	if s.config.Conductor == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "conductor not initialized",
		})

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "workspace not initialized",
		})

		return
	}

	// Load config
	cfg, err := ws.LoadConfig()
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "failed to load config: " + err.Error(),
		})

		return
	}

	// Create provider health container
	health := provider.NewProviderHealth()

	// Check GitHub provider
	if cfg.GitHub != nil && cfg.GitHub.Token != "" {
		// For now, return a basic status
		// Future improvement: Implement full provider instance creation and health checks
		health.Add("github", &provider.HealthInfo{
			Status:  provider.HealthStatusConnected,
			Message: "Connected",
		})
	} else {
		health.Add("github", &provider.HealthInfo{
			Status:  provider.HealthStatusNotConfigured,
			Message: "Set GITHUB_TOKEN in .mehrhof/.env",
		})
	}

	// Check GitLab provider
	if cfg.GitLab != nil && cfg.GitLab.Token != "" {
		health.Add("gitlab", &provider.HealthInfo{
			Status:  provider.HealthStatusConnected,
			Message: "Connected",
		})
	} else {
		health.Add("gitlab", &provider.HealthInfo{
			Status:  provider.HealthStatusNotConfigured,
			Message: "Set GITLAB_TOKEN in .mehrhof/.env",
		})
	}

	// Check Jira provider
	if cfg.Jira != nil && cfg.Jira.Token != "" && cfg.Jira.BaseURL != "" {
		health.Add("jira", &provider.HealthInfo{
			Status:  provider.HealthStatusConnected,
			Message: "Connected",
		})
	} else {
		health.Add("jira", &provider.HealthInfo{
			Status:  provider.HealthStatusNotConfigured,
			Message: "Set JIRA_TOKEN and JIRA_BASE_URL in .mehrhof/.env",
		})
	}

	// Add other providers...
	health.Add("linear", &provider.HealthInfo{
		Status:  provider.HealthStatusNotConfigured,
		Message: "Set LINEAR_API_KEY in .mehrhof/.env",
	})

	health.Add("notion", &provider.HealthInfo{
		Status:  provider.HealthStatusNotConfigured,
		Message: "Set NOTION_TOKEN in .mehrhof/.env",
	})

	health.Add("bitbucket", &provider.HealthInfo{
		Status:  provider.HealthStatusNotConfigured,
		Message: "Set BITBUCKET_APP_PASSWORD in .mehrhof/.env",
	})

	s.writeJSON(w, http.StatusOK, health)
}
