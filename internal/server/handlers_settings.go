package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/valksor/go-mehrhof/internal/coordination"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// handleSettingsPage renders the settings page.
func (s *Server) handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	if s.templates == nil {
		s.writeError(w, http.StatusInternalServerError, "templates not loaded")

		return
	}

	var cfg *storage.WorkspaceConfig
	var loadErr string
	var projects []storage.ProjectMetadata
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
			cfg, _, loadErr = loadProjectConfig(r.Context(), selectedProject)
			if cfg == nil {
				cfg = storage.NewDefaultWorkspaceConfig()
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
				var err error
				cfg, err = ws.LoadConfig()
				if err != nil {
					loadErr = "failed to load config: " + err.Error()
					cfg = storage.NewDefaultWorkspaceConfig()
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

	data := SettingsData{
		Mode:             s.modeString(),
		AuthEnabled:      s.config.AuthStore != nil,
		CanSwitchProject: s.canSwitchProject(),
		ShowSensitive:    isLocalRequest(r), // Only show tokens when accessed locally
		Config:           cfg,
		Agents:           agents,
		Success:          r.URL.Query().Get("success"),
		Error:            errorMsg,
		Projects:         projects,
		SelectedProject:  selectedProject,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.RenderSettings(w, data); err != nil {
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
		} else {
			s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")
		}

		return
	}

	// Load existing config to merge with
	cfg, err = ws.LoadConfig()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to load config: "+err.Error())

		return
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
