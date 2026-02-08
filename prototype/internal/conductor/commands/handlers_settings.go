package commands

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/coordination"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/schema"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "settings-get",
			Description:  "Get workspace settings",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleSettingsGet,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "settings-save",
			Description:  "Save workspace settings",
			Category:     "control",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleSettingsSave,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "config-explain",
			Description:  "Explain agent configuration resolution",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleConfigExplain,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "provider-health",
			Description:  "Check provider health status",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleProviderHealth,
	})
}

func handleSettingsGet(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	project := GetString(inv.Options, "project")
	mode := GetString(inv.Options, "mode")

	var cfg *storage.WorkspaceConfig

	if mode == "global" && project != "" {
		// Load project-specific config
		registry, err := storage.LoadRegistry()
		if err != nil {
			return nil, fmt.Errorf("failed to load project registry: %w", err)
		}

		proj, ok := registry.Projects[project]
		if !ok {
			return nil, errors.New("project not found in registry")
		}

		ws, err := storage.OpenWorkspace(ctx, proj.Path, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to open workspace: %w", err)
		}

		cfg, err = ws.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	} else if cond != nil {
		if ws := cond.GetWorkspace(); ws != nil {
			var err error
			cfg, err = ws.LoadConfig()
			if err != nil {
				cfg = storage.NewDefaultWorkspaceConfig()
			}
		} else {
			cfg = storage.NewDefaultWorkspaceConfig()
		}
	} else {
		cfg = storage.NewDefaultWorkspaceConfig()
	}

	// Strip sensitive fields in global mode
	if mode == "global" {
		cfg = stripSensitiveConfigFields(cfg)
	}

	// Always return schema format
	sch := schema.Generate(reflect.TypeOf(storage.WorkspaceConfig{}))

	return NewResult("Settings loaded").WithData(map[string]any{
		"schema": sch,
		"values": cfg,
	}), nil
}

func handleSettingsSave(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	project := GetString(inv.Options, "project")
	mode := GetString(inv.Options, "mode")
	workspaceRoot := GetString(inv.Options, "workspace_root")

	var ws *storage.Workspace

	if mode == "global" && project != "" {
		registry, err := storage.LoadRegistry()
		if err != nil {
			return nil, fmt.Errorf("failed to load project registry: %w", err)
		}

		proj, ok := registry.Projects[project]
		if !ok {
			return nil, errors.New("project not found in registry")
		}

		ws, err = storage.OpenWorkspace(ctx, proj.Path, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to open workspace: %w", err)
		}
	} else if condWs := cond.GetWorkspace(); condWs != nil {
		ws = condWs
	} else if workspaceRoot != "" {
		var err error
		ws, err = storage.OpenWorkspace(ctx, workspaceRoot, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to open workspace: %w", err)
		}
	}

	if ws == nil {
		if mode == "global" {
			return nil, errors.New("select a project first")
		}

		return nil, errors.New("workspace not available")
	}

	// Load existing config to merge with
	cfg, err := ws.LoadConfig()
	if err != nil {
		cfg = storage.NewDefaultWorkspaceConfig()
	}

	// Apply config data from invocation
	if configData, ok := inv.Options["config"]; ok {
		if configMap, ok := configData.(*storage.WorkspaceConfig); ok {
			cfg = configMap
		}
	}

	if err := ws.SaveConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	// Reinitialize conductor to pick up changes (non-fatal if it fails)
	_ = cond.Initialize(ctx)

	return NewResult("Settings saved").WithData(map[string]string{
		"status":  "ok",
		"message": "Settings saved successfully",
	}), nil
}

func handleConfigExplain(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	step := GetString(inv.Options, "step")
	if step == "" {
		return nil, errors.New("missing step parameter (planning, implementing, reviewing)")
	}

	var workflowStep workflow.Step
	switch step {
	case "planning":
		workflowStep = workflow.StepPlanning
	case "implementing", "implementation":
		workflowStep = workflow.StepImplementing
	case "reviewing", "review":
		workflowStep = workflow.StepReviewing
	default:
		return nil, errors.New("invalid step: " + step + " (must be planning, implementing, or reviewing)")
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	agents := cond.GetAgentRegistry()
	if agents == nil {
		return nil, errors.New("agent registry not available")
	}

	resolver := coordination.NewResolver(agents, ws)
	req := coordination.ResolveRequest{
		WorkspaceCfg: cfg,
		Step:         workflowStep,
	}

	explanation, err := resolver.ExplainAgentResolution(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to explain resolution: %w", err)
	}

	type resolutionStepData struct {
		Priority int    `json:"priority"`
		Source   string `json:"source"`
		Agent    string `json:"agent"`
		Skipped  bool   `json:"skipped"`
	}

	steps := make([]resolutionStepData, len(explanation.AllSteps))
	for i, s := range explanation.AllSteps {
		steps[i] = resolutionStepData{
			Priority: s.Priority,
			Source:   s.Source,
			Agent:    s.Agent,
			Skipped:  s.Skipped,
		}
	}

	return NewResult("Config explanation loaded").WithData(map[string]any{
		"step":      explanation.Step,
		"effective": explanation.Effective,
		"source":    explanation.Source,
		"steps":     steps,
	}), nil
}

func handleProviderHealth(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	health := provider.NewProviderHealth()

	if cfg.GitHub != nil && cfg.GitHub.Token != "" {
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

	return NewResult("Provider health loaded").WithData(health), nil
}

// stripSensitiveConfigFields returns a config copy with tokens and secrets cleared.
func stripSensitiveConfigFields(cfg *storage.WorkspaceConfig) *storage.WorkspaceConfig {
	result := *cfg

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
