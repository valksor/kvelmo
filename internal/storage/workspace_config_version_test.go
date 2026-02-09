package storage

import (
	"testing"
)

func TestCheckConfigVersion(t *testing.T) {
	tests := []struct {
		name       string
		version    int
		wantStatus ConfigVersionStatus
	}{
		{
			name:    "missing version field (zero value)",
			version: 0,
			wantStatus: ConfigVersionStatus{
				Current:    0,
				Required:   CurrentConfigVersion,
				IsOutdated: true,
			},
		},
		{
			name:    "old version",
			version: CurrentConfigVersion - 1,
			wantStatus: ConfigVersionStatus{
				Current:    CurrentConfigVersion - 1,
				Required:   CurrentConfigVersion,
				IsOutdated: true,
			},
		},
		{
			name:    "current version",
			version: CurrentConfigVersion,
			wantStatus: ConfigVersionStatus{
				Current:    CurrentConfigVersion,
				Required:   CurrentConfigVersion,
				IsOutdated: false,
			},
		},
		{
			name:    "future version (not outdated)",
			version: CurrentConfigVersion + 1,
			wantStatus: ConfigVersionStatus{
				Current:    CurrentConfigVersion + 1,
				Required:   CurrentConfigVersion,
				IsOutdated: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &WorkspaceConfig{Version: tt.version}
			got := CheckConfigVersion(cfg)

			if got.Current != tt.wantStatus.Current {
				t.Errorf("Current = %d, want %d", got.Current, tt.wantStatus.Current)
			}
			if got.Required != tt.wantStatus.Required {
				t.Errorf("Required = %d, want %d", got.Required, tt.wantStatus.Required)
			}
			if got.IsOutdated != tt.wantStatus.IsOutdated {
				t.Errorf("IsOutdated = %v, want %v", got.IsOutdated, tt.wantStatus.IsOutdated)
			}
		})
	}
}

func TestReinitConfig(t *testing.T) {
	t.Run("preserves agent settings", func(t *testing.T) {
		old := &WorkspaceConfig{
			Version: 0,
			Agent: AgentSettings{
				Default:    "custom-agent",
				Timeout:    600,
				MaxRetries: 5,
			},
		}

		got := ReinitConfig(old)

		if got.Version != CurrentConfigVersion {
			t.Errorf("Version = %d, want %d", got.Version, CurrentConfigVersion)
		}
		if got.Agent.Default != "custom-agent" {
			t.Errorf("Agent.Default = %q, want %q", got.Agent.Default, "custom-agent")
		}
		if got.Agent.Timeout != 600 {
			t.Errorf("Agent.Timeout = %d, want %d", got.Agent.Timeout, 600)
		}
		if got.Agent.MaxRetries != 5 {
			t.Errorf("Agent.MaxRetries = %d, want %d", got.Agent.MaxRetries, 5)
		}
	})

	t.Run("preserves git patterns", func(t *testing.T) {
		old := &WorkspaceConfig{
			Version: 0,
			Git: GitSettings{
				CommitPrefix:  "feat({key}):",
				BranchPattern: "feature/{key}",
				AutoCommit:    false,
				SignCommits:   true,
			},
		}

		got := ReinitConfig(old)

		if got.Git.CommitPrefix != "feat({key}):" {
			t.Errorf("Git.CommitPrefix = %q, want %q", got.Git.CommitPrefix, "feat({key}):")
		}
		if got.Git.BranchPattern != "feature/{key}" {
			t.Errorf("Git.BranchPattern = %q, want %q", got.Git.BranchPattern, "feature/{key}")
		}
		if got.Git.AutoCommit != false {
			t.Errorf("Git.AutoCommit = %v, want %v", got.Git.AutoCommit, false)
		}
		if got.Git.SignCommits != true {
			t.Errorf("Git.SignCommits = %v, want %v", got.Git.SignCommits, true)
		}
	})

	t.Run("preserves project settings", func(t *testing.T) {
		old := &WorkspaceConfig{
			Version: 0,
			Project: ProjectSettings{
				CodeDir: "../my-code",
			},
		}

		got := ReinitConfig(old)

		if got.Project.CodeDir != "../my-code" {
			t.Errorf("Project.CodeDir = %q, want %q", got.Project.CodeDir, "../my-code")
		}
	})

	t.Run("preserves provider configs", func(t *testing.T) {
		old := &WorkspaceConfig{
			Version: 0,
			GitHub: &GitHubSettings{
				Token: "ghp_secret",
				Owner: "my-org",
			},
			Jira: &JiraSettings{
				Token:   "jira-token",
				BaseURL: "https://my.atlassian.net",
			},
		}

		got := ReinitConfig(old)

		if got.GitHub == nil {
			t.Fatal("GitHub config not preserved")
		}
		if got.GitHub.Token != "ghp_secret" {
			t.Errorf("GitHub.Token = %q, want %q", got.GitHub.Token, "ghp_secret")
		}
		if got.GitHub.Owner != "my-org" {
			t.Errorf("GitHub.Owner = %q, want %q", got.GitHub.Owner, "my-org")
		}

		if got.Jira == nil {
			t.Fatal("Jira config not preserved")
		}
		if got.Jira.Token != "jira-token" {
			t.Errorf("Jira.Token = %q, want %q", got.Jira.Token, "jira-token")
		}
	})

	t.Run("preserves agent aliases", func(t *testing.T) {
		old := &WorkspaceConfig{
			Version: 0,
			Agents: map[string]AgentAliasConfig{
				"my-alias": {
					Extends:     "claude",
					Description: "My custom agent",
				},
			},
		}

		got := ReinitConfig(old)

		if len(got.Agents) != 1 {
			t.Fatalf("Agents length = %d, want 1", len(got.Agents))
		}
		if got.Agents["my-alias"].Extends != "claude" {
			t.Errorf("Agents[my-alias].Extends = %q, want %q", got.Agents["my-alias"].Extends, "claude")
		}
	})

	t.Run("preserves environment variables", func(t *testing.T) {
		old := &WorkspaceConfig{
			Version: 0,
			Env: map[string]string{
				"CUSTOM_VAR": "value",
			},
		}

		got := ReinitConfig(old)

		if got.Env["CUSTOM_VAR"] != "value" {
			t.Errorf("Env[CUSTOM_VAR] = %q, want %q", got.Env["CUSTOM_VAR"], "value")
		}
	})

	t.Run("uses defaults for empty old config", func(t *testing.T) {
		old := &WorkspaceConfig{
			Version: 0,
		}

		got := ReinitConfig(old)

		// Should have default values
		if got.Version != CurrentConfigVersion {
			t.Errorf("Version = %d, want %d", got.Version, CurrentConfigVersion)
		}
		// Default agent should be set from NewDefaultWorkspaceConfig
		defaults := NewDefaultWorkspaceConfig()
		if got.Agent.Default != defaults.Agent.Default {
			t.Errorf("Agent.Default = %q, want default %q", got.Agent.Default, defaults.Agent.Default)
		}
	})
}
