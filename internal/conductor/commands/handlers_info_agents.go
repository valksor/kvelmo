package commands

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "agents",
			Description:  "List available agents",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleAgents,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "providers",
			Description:  "List available providers",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleProviders,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "agent-alias",
			Description:  "Manage agent aliases (list, add, delete)",
			Category:     "info",
			Subcommands:  []string{"list", "add", "delete"},
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleAgentAlias,
	})
}

// agentData represents an agent's info in the response.
type agentData struct {
	Name         string             `json:"name"`
	Type         string             `json:"type"`
	Extends      string             `json:"extends,omitempty"`
	Description  string             `json:"description,omitempty"`
	Version      string             `json:"version,omitempty"`
	Available    bool               `json:"available"`
	Capabilities *agentCapabilities `json:"capabilities,omitempty"`
	Models       []agentModelData   `json:"models,omitempty"`
}

// agentCapabilities represents agent capabilities.
type agentCapabilities struct {
	Streaming      bool     `json:"streaming"`
	ToolUse        bool     `json:"tool_use"`
	FileOperations bool     `json:"file_operations"`
	CodeExecution  bool     `json:"code_execution"`
	MultiTurn      bool     `json:"multi_turn"`
	SystemPrompt   bool     `json:"system_prompt"`
	AllowedTools   []string `json:"allowed_tools,omitempty"`
}

// agentModelData represents model info.
type agentModelData struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Default    bool    `json:"default"`
	MaxTokens  int     `json:"max_tokens,omitempty"`
	InputCost  float64 `json:"input_cost_usd,omitempty"`
	OutputCost float64 `json:"output_cost_usd,omitempty"`
}

// providerData represents a provider's info.
type providerData struct {
	Scheme      string   `json:"scheme"`
	Shorthand   string   `json:"shorthand,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	EnvVars     []string `json:"env_vars,omitempty"`
}

func handleAgents(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	registry := cond.GetAgentRegistry()
	if registry == nil {
		return NewResult("No agents available").WithData(map[string]any{
			"agents": []agentData{},
			"count":  0,
		}), nil
	}

	agentNames := registry.List()
	slices.Sort(agentNames)

	var agents []agentData
	for _, name := range agentNames {
		a, err := registry.Get(name)
		if err != nil {
			continue
		}

		info := agentData{
			Name:      name,
			Available: a.Available() == nil,
		}

		if alias, ok := a.(*agent.AliasAgent); ok {
			info.Type = "alias"
			info.Description = alias.Description()
			if base := alias.BaseAgent(); base != nil {
				info.Extends = base.Name()
			}
		} else {
			info.Type = "built-in"
			if mp, ok := a.(agent.MetadataProvider); ok {
				meta := mp.Metadata()
				info.Description = meta.Description
				info.Version = meta.Version

				if meta.Capabilities.Streaming || meta.Capabilities.ToolUse ||
					meta.Capabilities.FileOperations || meta.Capabilities.CodeExecution ||
					meta.Capabilities.MultiTurn || meta.Capabilities.SystemPrompt ||
					len(meta.Capabilities.AllowedTools) > 0 {
					info.Capabilities = &agentCapabilities{
						Streaming:      meta.Capabilities.Streaming,
						ToolUse:        meta.Capabilities.ToolUse,
						FileOperations: meta.Capabilities.FileOperations,
						CodeExecution:  meta.Capabilities.CodeExecution,
						MultiTurn:      meta.Capabilities.MultiTurn,
						SystemPrompt:   meta.Capabilities.SystemPrompt,
					}
					if len(meta.Capabilities.AllowedTools) > 0 {
						info.Capabilities.AllowedTools = meta.Capabilities.AllowedTools
					}
				}

				for _, m := range meta.Models {
					info.Models = append(info.Models, agentModelData{
						ID:         m.ID,
						Name:       m.Name,
						Default:    m.Default,
						MaxTokens:  m.MaxTokens,
						InputCost:  m.InputCost,
						OutputCost: m.OutputCost,
					})
				}
			}
		}

		agents = append(agents, info)
	}

	return NewResult("Agents loaded").WithData(map[string]any{
		"agents": agents,
		"count":  len(agents),
	}), nil
}

func handleProviders(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
	providers := []providerData{
		{Scheme: "file", Shorthand: "f", Name: "File", Description: "Single markdown file"},
		{Scheme: "dir", Shorthand: "d", Name: "Directory", Description: "Directory with README.md"},
		{Scheme: "github", Shorthand: "gh", Name: "GitHub", Description: "GitHub issues and pull requests", EnvVars: []string{"GITHUB_TOKEN"}},
		{Scheme: "gitlab", Name: "GitLab", Description: "GitLab issues and merge requests", EnvVars: []string{"GITLAB_TOKEN"}},
		{Scheme: "jira", Name: "Jira", Description: "Atlassian Jira tickets", EnvVars: []string{"JIRA_TOKEN"}},
		{Scheme: "linear", Name: "Linear", Description: "Linear issues", EnvVars: []string{"LINEAR_API_KEY"}},
		{Scheme: "notion", Name: "Notion", Description: "Notion pages and databases", EnvVars: []string{"NOTION_TOKEN"}},
		{Scheme: "wrike", Name: "Wrike", Description: "Wrike tasks", EnvVars: []string{"WRIKE_TOKEN"}},
		{Scheme: "youtrack", Shorthand: "yt", Name: "YouTrack", Description: "JetBrains YouTrack issues", EnvVars: []string{"YOUTRACK_TOKEN"}},
		{Scheme: "bitbucket", Shorthand: "bb", Name: "Bitbucket", Description: "Bitbucket issues and pull requests", EnvVars: []string{"BITBUCKET_TOKEN"}},
		{Scheme: "azure", Name: "Azure DevOps", Description: "Azure DevOps work items", EnvVars: []string{"AZURE_DEVOPS_TOKEN"}},
		{Scheme: "clickup", Name: "ClickUp", Description: "ClickUp tasks", EnvVars: []string{"CLICKUP_TOKEN"}},
		{Scheme: "asana", Name: "Asana", Description: "Asana tasks", EnvVars: []string{"ASANA_TOKEN"}},
		{Scheme: "monday", Name: "Monday", Description: "Monday.com items", EnvVars: []string{"MONDAY_TOKEN"}},
		{Scheme: "trello", Name: "Trello", Description: "Trello cards", EnvVars: []string{"TRELLO_KEY", "TRELLO_TOKEN"}},
	}

	return NewResult("Providers loaded").WithData(map[string]any{
		"providers": providers,
		"count":     len(providers),
	}), nil
}

func handleAgentAlias(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	subcommand := GetString(inv.Options, "subcommand")
	if subcommand == "" && len(inv.Args) > 0 {
		subcommand = inv.Args[0]
	}
	if subcommand == "" {
		subcommand = "list"
	}

	switch subcommand {
	case "list":
		return handleAgentAliasList(cond)
	case "add":
		return handleAgentAliasAdd(cond, inv)
	case "delete":
		return handleAgentAliasDelete(cond, inv)
	default:
		return nil, errors.New("unknown subcommand: " + subcommand + " (use list, add, or delete)")
	}
}

func handleAgentAliasList(cond *conductor.Conductor) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return nil, errors.New("failed to load config: " + err.Error())
	}

	return NewResult("Agent aliases loaded").WithData(map[string]any{
		"agents": cfg.Agents,
	}), nil
}

func handleAgentAliasAdd(cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	name := strings.TrimSpace(GetString(inv.Options, "name"))
	if name == "" {
		return nil, errors.New("name is required")
	}

	extends := strings.TrimSpace(GetString(inv.Options, "extends"))
	if extends == "" {
		return nil, errors.New("extends is required")
	}

	description := GetString(inv.Options, "description")

	var components []string
	if raw, ok := inv.Options["components"]; ok {
		if s, ok := raw.([]string); ok {
			components = s
		}
		if arr, ok := raw.([]any); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					components = append(components, s)
				}
			}
		}
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return nil, errors.New("failed to load config: " + err.Error())
	}

	if cfg.Agents == nil {
		cfg.Agents = make(map[string]storage.AgentAliasConfig)
	}

	cfg.Agents[name] = storage.AgentAliasConfig{
		Extends:     extends,
		Description: description,
		Components:  components,
	}

	if err := ws.SaveConfig(cfg); err != nil {
		return nil, errors.New("failed to save config: " + err.Error())
	}

	return NewResult("Agent alias created").WithData(map[string]string{
		"status":  "created",
		"message": "Agent alias created",
	}), nil
}

func handleAgentAliasDelete(cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	name := strings.TrimSpace(GetString(inv.Options, "name"))
	if name == "" {
		return nil, errors.New("alias name is required")
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return nil, errors.New("failed to load config: " + err.Error())
	}

	if cfg.Agents == nil {
		return nil, errors.New("alias not found")
	}

	if _, exists := cfg.Agents[name]; !exists {
		return nil, errors.New("alias not found: " + name)
	}

	delete(cfg.Agents, name)

	if err := ws.SaveConfig(cfg); err != nil {
		return nil, errors.New("failed to save config: " + err.Error())
	}

	return NewResult("Agent alias deleted").WithData(map[string]string{
		"status":  "deleted",
		"message": "Agent alias deleted",
	}), nil
}
