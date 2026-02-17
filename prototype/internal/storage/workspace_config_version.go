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

// ResetConfigPreserveEnv creates a fresh default config while preserving only
// environment-related fields (API keys, tokens). This is for the "force reset"
// use case where the user wants defaults but needs to keep secrets.
func ResetConfigPreserveEnv(old *WorkspaceConfig) *WorkspaceConfig {
	cfg := NewDefaultWorkspaceConfig()

	// Preserve top-level Env map
	if len(old.Env) > 0 {
		cfg.Env = old.Env
	}

	// Preserve agent alias envs (only the Env field, not other alias settings)
	for name, alias := range old.Agents {
		if len(alias.Env) > 0 {
			if cfg.Agents == nil {
				cfg.Agents = make(map[string]AgentAliasConfig)
			}
			newAlias := cfg.Agents[name]
			newAlias.Env = alias.Env
			cfg.Agents[name] = newAlias
		}
	}

	// Preserve provider tokens
	preserveProviderTokens(cfg, old)

	return cfg
}

// preserveProviderTokens copies provider authentication tokens from old to new config.
func preserveProviderTokens(newCfg, oldCfg *WorkspaceConfig) {
	if oldCfg.GitHub != nil && oldCfg.GitHub.Token != "" {
		if newCfg.GitHub == nil {
			newCfg.GitHub = &GitHubSettings{}
		}
		newCfg.GitHub.Token = oldCfg.GitHub.Token
	}

	if oldCfg.GitLab != nil && oldCfg.GitLab.Token != "" {
		if newCfg.GitLab == nil {
			newCfg.GitLab = &GitLabSettings{}
		}
		newCfg.GitLab.Token = oldCfg.GitLab.Token
	}

	if oldCfg.Notion != nil && oldCfg.Notion.Token != "" {
		if newCfg.Notion == nil {
			newCfg.Notion = &NotionSettings{}
		}
		newCfg.Notion.Token = oldCfg.Notion.Token
	}

	if oldCfg.Jira != nil && oldCfg.Jira.Token != "" {
		if newCfg.Jira == nil {
			newCfg.Jira = &JiraSettings{}
		}
		newCfg.Jira.Token = oldCfg.Jira.Token
	}

	if oldCfg.Linear != nil && oldCfg.Linear.Token != "" {
		if newCfg.Linear == nil {
			newCfg.Linear = &LinearSettings{}
		}
		newCfg.Linear.Token = oldCfg.Linear.Token
	}

	if oldCfg.Wrike != nil && oldCfg.Wrike.Token != "" {
		if newCfg.Wrike == nil {
			newCfg.Wrike = &WrikeSettings{}
		}
		newCfg.Wrike.Token = oldCfg.Wrike.Token
	}

	if oldCfg.YouTrack != nil && oldCfg.YouTrack.Token != "" {
		if newCfg.YouTrack == nil {
			newCfg.YouTrack = &YouTrackSettings{}
		}
		newCfg.YouTrack.Token = oldCfg.YouTrack.Token
	}

	if oldCfg.Bitbucket != nil && oldCfg.Bitbucket.AppPassword != "" {
		if newCfg.Bitbucket == nil {
			newCfg.Bitbucket = &BitbucketSettings{}
		}
		newCfg.Bitbucket.AppPassword = oldCfg.Bitbucket.AppPassword
	}

	if oldCfg.Asana != nil && oldCfg.Asana.Token != "" {
		if newCfg.Asana == nil {
			newCfg.Asana = &AsanaSettings{}
		}
		newCfg.Asana.Token = oldCfg.Asana.Token
	}

	if oldCfg.ClickUp != nil && oldCfg.ClickUp.Token != "" {
		if newCfg.ClickUp == nil {
			newCfg.ClickUp = &ClickUpSettings{}
		}
		newCfg.ClickUp.Token = oldCfg.ClickUp.Token
	}

	if oldCfg.AzureDevOps != nil && oldCfg.AzureDevOps.Token != "" {
		if newCfg.AzureDevOps == nil {
			newCfg.AzureDevOps = &AzureDevOpsSettings{}
		}
		newCfg.AzureDevOps.Token = oldCfg.AzureDevOps.Token
	}

	if oldCfg.Trello != nil {
		if oldCfg.Trello.APIKey != "" || oldCfg.Trello.Token != "" {
			if newCfg.Trello == nil {
				newCfg.Trello = &TrelloSettings{}
			}
			if oldCfg.Trello.APIKey != "" {
				newCfg.Trello.APIKey = oldCfg.Trello.APIKey
			}
			if oldCfg.Trello.Token != "" {
				newCfg.Trello.Token = oldCfg.Trello.Token
			}
		}
	}
}
