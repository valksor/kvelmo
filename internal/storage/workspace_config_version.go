package storage

// CurrentConfigVersion is the latest config schema version.
// Increment when making breaking changes that require re-initialization.
const CurrentConfigVersion = 1

// ConfigVersionStatus represents the result of a version check.
type ConfigVersionStatus struct {
	Current    int  // Version in config (0 if missing)
	Required   int  // CurrentConfigVersion constant
	IsOutdated bool // true if Current < Required or Current == 0
}

// CheckConfigVersion checks if a config is outdated.
func CheckConfigVersion(cfg *WorkspaceConfig) ConfigVersionStatus {
	return ConfigVersionStatus{
		Current:    cfg.Version,
		Required:   CurrentConfigVersion,
		IsOutdated: cfg.Version == 0 || cfg.Version < CurrentConfigVersion,
	}
}

// ReinitConfig creates a new config while preserving key user values.
// This is used when upgrading from an outdated config version.
func ReinitConfig(old *WorkspaceConfig) *WorkspaceConfig {
	cfg := NewDefaultWorkspaceConfig()

	// Preserve user-configured agent settings
	if old.Agent.Default != "" {
		cfg.Agent.Default = old.Agent.Default
	}
	if old.Agent.Timeout > 0 {
		cfg.Agent.Timeout = old.Agent.Timeout
	}
	if old.Agent.MaxRetries > 0 {
		cfg.Agent.MaxRetries = old.Agent.MaxRetries
	}
	if old.Agent.Steps != nil {
		cfg.Agent.Steps = old.Agent.Steps
	}

	// Preserve git patterns
	if old.Git.CommitPrefix != "" {
		cfg.Git.CommitPrefix = old.Git.CommitPrefix
	}
	if old.Git.BranchPattern != "" {
		cfg.Git.BranchPattern = old.Git.BranchPattern
	}
	cfg.Git.AutoCommit = old.Git.AutoCommit
	cfg.Git.SignCommits = old.Git.SignCommits
	cfg.Git.StashOnStart = old.Git.StashOnStart
	cfg.Git.AutoPopStash = old.Git.AutoPopStash

	// Preserve project settings
	if old.Project.CodeDir != "" {
		cfg.Project.CodeDir = old.Project.CodeDir
	}

	// Preserve provider default
	if old.Providers.Default != "" {
		cfg.Providers.Default = old.Providers.Default
	}

	// Preserve all provider configurations (tokens, etc.)
	cfg.GitHub = old.GitHub
	cfg.GitLab = old.GitLab
	cfg.Jira = old.Jira
	cfg.Linear = old.Linear
	cfg.Notion = old.Notion
	cfg.Wrike = old.Wrike
	cfg.YouTrack = old.YouTrack
	cfg.Bitbucket = old.Bitbucket
	cfg.Asana = old.Asana
	cfg.ClickUp = old.ClickUp
	cfg.AzureDevOps = old.AzureDevOps
	cfg.Trello = old.Trello

	// Preserve agent aliases
	if len(old.Agents) > 0 {
		cfg.Agents = old.Agents
	}

	// Preserve environment variables
	if len(old.Env) > 0 {
		cfg.Env = old.Env
	}

	// Preserve plugin configuration
	if len(old.Plugins.Enabled) > 0 {
		cfg.Plugins.Enabled = old.Plugins.Enabled
	}
	if len(old.Plugins.Config) > 0 {
		cfg.Plugins.Config = old.Plugins.Config
	}

	return cfg
}
