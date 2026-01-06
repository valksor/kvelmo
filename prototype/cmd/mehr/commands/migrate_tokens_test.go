//go:build !testbinary
// +build !testbinary

package commands

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// TestRunMigrateTokens_NoTokensToMigrate tests migration when config has no tokens.
func TestRunMigrateTokens_NoTokensToMigrate(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Create workspace with default (empty) config
	ws := openTestWorkspace(t, tmpDir)

	cfg := storage.NewDefaultWorkspaceConfig()
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Run migration - should print "no migration needed"
	cmd := &cobra.Command{}
	err := runMigrateTokens(cmd, []string{})
	if err != nil {
		t.Fatalf("runMigrateTokens: %v", err)
	}

	// Verify .env file was not created
	envPath := ws.EnvPath()
	if _, err := os.Stat(envPath); !os.IsNotExist(err) {
		t.Error(".env file should not exist when no tokens to migrate")
	}
}

// TestRunMigrateTokens_GitHubToken tests migrating a GitHub token.
func TestRunMigrateTokens_GitHubToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ws := openTestWorkspace(t, tmpDir)

	// Initialize workspace to create .mehrhof directory
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.GitHub = &storage.GitHubSettings{Token: "ghp_test_token_12345"}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	cmd := &cobra.Command{}
	err := runMigrateTokens(cmd, []string{})
	if err != nil {
		t.Fatalf("runMigrateTokens: %v", err)
	}

	// Set the env var so LoadConfig will expand it correctly
	t.Setenv("GITHUB_TOKEN", "ghp_test_token_12345")

	// Verify config was updated with ${VAR} syntax
	updatedCfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if updatedCfg.GitHub == nil {
		t.Fatal("GitHub config should not be nil")
	}
	// LoadConfig expands ${GITHUB_TOKEN} to the actual env var value
	if updatedCfg.GitHub.Token != "ghp_test_token_12345" {
		t.Errorf("GitHub.Token = %q, want ghp_test_token_12345 (expanded from ${GITHUB_TOKEN})", updatedCfg.GitHub.Token)
	}

	// Verify .env file was created with the token
	envPath := ws.EnvPath()
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}

	envContent := string(content)
	if !strings.Contains(envContent, "GITHUB_TOKEN=ghp_test_token_12345") {
		t.Errorf(".env should contain GITHUB_TOKEN, got: %s", envContent)
	}
}

// TestRunMigrateTokens_MultipleTokens tests migrating multiple provider tokens.
func TestRunMigrateTokens_MultipleTokens(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ws := openTestWorkspace(t, tmpDir)

	// Initialize workspace to create .mehrhof directory
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.GitHub = &storage.GitHubSettings{Token: "ghp_test"}
	cfg.GitLab = &storage.GitLabSettings{Token: "glpat_test"}
	cfg.Notion = &storage.NotionSettings{Token: "secret_test"}
	cfg.Jira = &storage.JiraSettings{Token: "jira_test"}
	cfg.Linear = &storage.LinearSettings{Token: "lin_api_test"}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	cmd := &cobra.Command{}
	err := runMigrateTokens(cmd, []string{})
	if err != nil {
		t.Fatalf("runMigrateTokens: %v", err)
	}

	// Set env vars so LoadConfig will expand them correctly
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("GITLAB_TOKEN", "glpat_test")
	t.Setenv("NOTION_TOKEN", "secret_test")
	t.Setenv("JIRA_TOKEN", "jira_test")
	t.Setenv("LINEAR_API_KEY", "lin_api_test")

	// Verify all tokens were migrated
	updatedCfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	tests := []struct {
		name     string
		getToken func() string
		expected string
	}{
		{"GitHub", func() string {
			if updatedCfg.GitHub == nil {
				return ""
			}

			return updatedCfg.GitHub.Token
		}, "ghp_test"},
		{"GitLab", func() string {
			if updatedCfg.GitLab == nil {
				return ""
			}

			return updatedCfg.GitLab.Token
		}, "glpat_test"},
		{"Notion", func() string {
			if updatedCfg.Notion == nil {
				return ""
			}

			return updatedCfg.Notion.Token
		}, "secret_test"},
		{"Jira", func() string {
			if updatedCfg.Jira == nil {
				return ""
			}

			return updatedCfg.Jira.Token
		}, "jira_test"},
		{"Linear", func() string {
			if updatedCfg.Linear == nil {
				return ""
			}

			return updatedCfg.Linear.Token
		}, "lin_api_test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.getToken()
			if token != tt.expected {
				t.Errorf("%s.Token = %q, want %q", tt.name, token, tt.expected)
			}
		})
	}

	// Verify .env file contains all tokens
	envPath := ws.EnvPath()
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}

	envContent := string(content)
	expectedTokens := []string{
		"GITHUB_TOKEN=ghp_test",
		"GITLAB_TOKEN=glpat_test",
		"NOTION_TOKEN=secret_test",
		"JIRA_TOKEN=jira_test",
		"LINEAR_API_KEY=lin_api_test",
	}

	for _, expected := range expectedTokens {
		if !strings.Contains(envContent, expected) {
			t.Errorf(".env should contain %s, got: %s", expected, envContent)
		}
	}
}

// TestRunMigrateTokens_AlreadyHasVarSyntax tests that tokens with ${VAR} syntax are skipped.
func TestRunMigrateTokens_AlreadyHasVarSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ws := openTestWorkspace(t, tmpDir)

	cfg := storage.NewDefaultWorkspaceConfig()
	// Already using ${VAR} syntax - no actual value ever provided
	cfg.GitHub = &storage.GitHubSettings{Token: "${GITHUB_TOKEN}"}
	// Plain token that should be migrated
	cfg.GitLab = &storage.GitLabSettings{Token: "glpat_test"}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	cmd := &cobra.Command{}
	err := runMigrateTokens(cmd, []string{})
	if err != nil {
		t.Fatalf("runMigrateTokens: %v", err)
	}

	// Set env vars for LoadConfig expansion
	// GitHub was never set with a value, so it won't be in .env
	// GitLab was migrated, so set its env var
	t.Setenv("GITLAB_TOKEN", "glpat_test")

	// Verify tokens were migrated correctly
	updatedCfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	// GitHub token was never actually set (just ${VAR} reference), so it expands to ""
	if updatedCfg.GitHub.Token != "" {
		t.Errorf("GitHub.Token should be empty (no value ever provided), got %q", updatedCfg.GitHub.Token)
	}

	// GitLab token was migrated and should expand correctly
	if updatedCfg.GitLab.Token != "glpat_test" {
		t.Errorf("GitLab.Token = %q, want glpat_test", updatedCfg.GitLab.Token)
	}

	// Verify .env only has GitLab token (not GitHub)
	envPath := ws.EnvPath()
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}

	envContent := string(content)
	if strings.Contains(envContent, "GITHUB_TOKEN=") {
		t.Error(".env should not contain GITHUB_TOKEN (already had ${VAR} syntax)")
	}
	if !strings.Contains(envContent, "GITLAB_TOKEN=glpat_test") {
		t.Error(".env should contain GITLAB_TOKEN")
	}
}

// TestRunMigrateTokens_TrelloCredentials tests migrating Trello API key and token.
func TestRunMigrateTokens_TrelloCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ws := openTestWorkspace(t, tmpDir)

	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Trello = &storage.TrelloSettings{
		APIKey: "test_api_key",
		Token:  "test_token",
	}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	cmd := &cobra.Command{}
	err := runMigrateTokens(cmd, []string{})
	if err != nil {
		t.Fatalf("runMigrateTokens: %v", err)
	}

	// Set env vars so LoadConfig will expand them correctly
	t.Setenv("TRELLO_API_KEY", "test_api_key")
	t.Setenv("TRELLO_TOKEN", "test_token")

	// Verify both credentials were migrated
	updatedCfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if updatedCfg.Trello.APIKey != "test_api_key" {
		t.Errorf("Trello.APIKey = %q, want test_api_key", updatedCfg.Trello.APIKey)
	}
	if updatedCfg.Trello.Token != "test_token" {
		t.Errorf("Trello.Token = %q, want test_token", updatedCfg.Trello.Token)
	}
}

// TestRunMigrateTokens_AppendToExistingEnv tests appending to existing .env file.
func TestRunMigrateTokens_AppendToExistingEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ws := openTestWorkspace(t, tmpDir)

	// Create existing .env file
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	envPath := ws.EnvPath()
	existingContent := "EXISTING_VAR=existing_value\n"
	if err := os.WriteFile(envPath, []byte(existingContent), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.GitHub = &storage.GitHubSettings{Token: "ghp_test"}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	cmd := &cobra.Command{}
	err := runMigrateTokens(cmd, []string{})
	if err != nil {
		t.Fatalf("runMigrateTokens: %v", err)
	}

	// Verify existing content is preserved
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}

	envContent := string(content)
	if !strings.Contains(envContent, "EXISTING_VAR=existing_value") {
		t.Error(".env should preserve existing variables")
	}
	if !strings.Contains(envContent, "GITHUB_TOKEN=ghp_test") {
		t.Error(".env should contain new GITHUB_TOKEN")
	}
}

// TestRunMigrateTokens_ReplacesExistingEnvVar tests replacing an existing env var.
func TestRunMigrateTokens_ReplacesExistingEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ws := openTestWorkspace(t, tmpDir)

	// Create .env file with existing GITHUB_TOKEN
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	envPath := ws.EnvPath()
	existingContent := "GITHUB_TOKEN=old_token_value\n"
	if err := os.WriteFile(envPath, []byte(existingContent), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.GitHub = &storage.GitHubSettings{Token: "ghp_new_token"}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	cmd := &cobra.Command{}
	err := runMigrateTokens(cmd, []string{})
	if err != nil {
		t.Fatalf("runMigrateTokens: %v", err)
	}

	// Verify token was replaced
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}

	envContent := string(content)
	if strings.Contains(envContent, "old_token_value") {
		t.Error(".env should not contain old token value")
	}
	if !strings.Contains(envContent, "GITHUB_TOKEN=ghp_new_token") {
		t.Error(".env should contain new token value")
	}
}

// TestRunMigrateTokens_AllProviders tests that all providers are handled.
func TestRunMigrateTokens_AllProviders(t *testing.T) {
	providers := []struct {
		name      string
		token     string
		envVar    string
		setupCfg  func(*storage.WorkspaceConfig)
		verifyCfg func(*testing.T, *storage.WorkspaceConfig)
	}{
		{
			name:   "GitHub",
			token:  "ghp_test",
			envVar: "GITHUB_TOKEN",
			setupCfg: func(cfg *storage.WorkspaceConfig) {
				cfg.GitHub = &storage.GitHubSettings{Token: "ghp_test"}
			},
			verifyCfg: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.GitHub.Token != "ghp_test" {
					t.Errorf("GitHub.Token = %q, want ghp_test", cfg.GitHub.Token)
				}
			},
		},
		{
			name:   "GitLab",
			token:  "glpat_test",
			envVar: "GITLAB_TOKEN",
			setupCfg: func(cfg *storage.WorkspaceConfig) {
				cfg.GitLab = &storage.GitLabSettings{Token: "glpat_test"}
			},
			verifyCfg: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.GitLab.Token != "glpat_test" {
					t.Errorf("GitLab.Token = %q, want glpat_test", cfg.GitLab.Token)
				}
			},
		},
		{
			name:   "Notion",
			token:  "secret_test",
			envVar: "NOTION_TOKEN",
			setupCfg: func(cfg *storage.WorkspaceConfig) {
				cfg.Notion = &storage.NotionSettings{Token: "secret_test"}
			},
			verifyCfg: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Notion.Token != "secret_test" {
					t.Errorf("Notion.Token = %q, want secret_test", cfg.Notion.Token)
				}
			},
		},
		{
			name:   "Jira",
			token:  "jira_test",
			envVar: "JIRA_TOKEN",
			setupCfg: func(cfg *storage.WorkspaceConfig) {
				cfg.Jira = &storage.JiraSettings{Token: "jira_test"}
			},
			verifyCfg: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Jira.Token != "jira_test" {
					t.Errorf("Jira.Token = %q, want jira_test", cfg.Jira.Token)
				}
			},
		},
		{
			name:   "Linear",
			token:  "lin_api_test",
			envVar: "LINEAR_API_KEY",
			setupCfg: func(cfg *storage.WorkspaceConfig) {
				cfg.Linear = &storage.LinearSettings{Token: "lin_api_test"}
			},
			verifyCfg: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Linear.Token != "lin_api_test" {
					t.Errorf("Linear.Token = %q, want lin_api_test", cfg.Linear.Token)
				}
			},
		},
		{
			name:   "Wrike",
			token:  "wrike_test",
			envVar: "WRIKE_TOKEN",
			setupCfg: func(cfg *storage.WorkspaceConfig) {
				cfg.Wrike = &storage.WrikeSettings{Token: "wrike_test"}
			},
			verifyCfg: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Wrike.Token != "wrike_test" {
					t.Errorf("Wrike.Token = %q, want wrike_test", cfg.Wrike.Token)
				}
			},
		},
		{
			name:   "YouTrack",
			token:  "yt_test",
			envVar: "YOUTRACK_TOKEN",
			setupCfg: func(cfg *storage.WorkspaceConfig) {
				cfg.YouTrack = &storage.YouTrackSettings{Token: "yt_test"}
			},
			verifyCfg: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.YouTrack.Token != "yt_test" {
					t.Errorf("YouTrack.Token = %q, want yt_test", cfg.YouTrack.Token)
				}
			},
		},
		{
			name:   "Asana",
			token:  "asana_test",
			envVar: "ASANA_TOKEN",
			setupCfg: func(cfg *storage.WorkspaceConfig) {
				cfg.Asana = &storage.AsanaSettings{Token: "asana_test"}
			},
			verifyCfg: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Asana.Token != "asana_test" {
					t.Errorf("Asana.Token = %q, want asana_test", cfg.Asana.Token)
				}
			},
		},
		{
			name:   "ClickUp",
			token:  "cu_test",
			envVar: "CLICKUP_TOKEN",
			setupCfg: func(cfg *storage.WorkspaceConfig) {
				cfg.ClickUp = &storage.ClickUpSettings{Token: "cu_test"}
			},
			verifyCfg: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.ClickUp.Token != "cu_test" {
					t.Errorf("ClickUp.Token = %q, want cu_test", cfg.ClickUp.Token)
				}
			},
		},
		{
			name:   "Azure DevOps",
			token:  "azdo_test",
			envVar: "AZURE_DEVOPS_TOKEN",
			setupCfg: func(cfg *storage.WorkspaceConfig) {
				cfg.AzureDevOps = &storage.AzureDevOpsSettings{Token: "azdo_test"}
			},
			verifyCfg: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.AzureDevOps.Token != "azdo_test" {
					t.Errorf("AzureDevOps.Token = %q, want azdo_test", cfg.AzureDevOps.Token)
				}
			},
		},
		{
			name:   "Bitbucket",
			token:  "bb_test",
			envVar: "BITBUCKET_APP_PASSWORD",
			setupCfg: func(cfg *storage.WorkspaceConfig) {
				cfg.Bitbucket = &storage.BitbucketSettings{AppPassword: "bb_test"}
			},
			verifyCfg: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Bitbucket.AppPassword != "bb_test" {
					t.Errorf("Bitbucket.AppPassword = %q, want bb_test", cfg.Bitbucket.AppPassword)
				}
			},
		},
	}

	for _, tt := range providers {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			ws := openTestWorkspace(t, tmpDir)

			cfg := storage.NewDefaultWorkspaceConfig()
			tt.setupCfg(cfg)
			if err := ws.SaveConfig(cfg); err != nil {
				t.Fatalf("SaveConfig: %v", err)
			}

			cmd := &cobra.Command{}
			err := runMigrateTokens(cmd, []string{})
			if err != nil {
				t.Fatalf("runMigrateTokens: %v", err)
			}

			// Set env var so LoadConfig will expand it correctly
			t.Setenv(tt.envVar, tt.token)

			// Verify config was updated
			updatedCfg, err := ws.LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig: %v", err)
			}

			tt.verifyCfg(t, updatedCfg)

			// Verify .env file was created
			envPath := ws.EnvPath()
			content, err := os.ReadFile(envPath)
			if err != nil {
				t.Fatalf("read .env: %v", err)
			}

			envContent := string(content)
			expectedLine := tt.envVar + "=" + tt.token
			if !strings.Contains(envContent, expectedLine) {
				t.Errorf(".env should contain %s, got: %s", expectedLine, envContent)
			}
		})
	}
}

// TestMigrateTokensCommand_Structure tests the migrate tokens command structure.
func TestMigrateTokensCommand_Structure(t *testing.T) {
	if migrateTokensCmd.Use != "migrate-tokens" {
		t.Errorf("Use = %q, want 'migrate-tokens'", migrateTokensCmd.Use)
	}

	if migrateTokensCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if migrateTokensCmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if migrateTokensCmd.RunE == nil {
		t.Error("RunE should be set")
	}
}
