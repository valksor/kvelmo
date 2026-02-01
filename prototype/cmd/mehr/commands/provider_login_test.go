package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// openTestWorkspace creates a test workspace with a temporary home directory.
func openTestWorkspace(tb testing.TB, repoRoot string) *storage.Workspace {
	tb.Helper()

	homeDir := tb.TempDir()
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir

	ws, err := storage.OpenWorkspace(context.Background(), repoRoot, cfg)
	if err != nil {
		tb.Fatalf("OpenWorkspace: %v", err)
	}

	return ws
}

// TestProviderRegistry ensures all expected providers are registered.
func TestProviderRegistry(t *testing.T) {
	expectedProviders := []string{
		"github", "gitlab", "notion", "jira", "linear", "wrike", "youtrack",
		"bitbucket", "asana", "clickup", "trello", "azuredevops",
	}

	for _, provider := range expectedProviders {
		cfg := getProviderLoginConfig(provider)
		if cfg == nil {
			t.Errorf("Provider %q not found in registry", provider)

			continue
		}

		// Validate required fields
		if cfg.Name == "" {
			t.Errorf("Provider %q: Name is empty", provider)
		}
		if cfg.EnvVar == "" {
			t.Errorf("Provider %q: EnvVar is empty", provider)
		}
		if cfg.ConfigField == "" {
			t.Errorf("Provider %q: ConfigField is empty", provider)
		}
		if cfg.HelpURL == "" {
			t.Errorf("Provider %q: HelpURL is empty", provider)
		}
		if cfg.HelpSteps == "" {
			t.Errorf("Provider %q: HelpSteps is empty", provider)
		}
		if cfg.Scopes == "" {
			t.Errorf("Provider %q: Scopes is empty", provider)
		}
	}
}

// TestNormalizeProviderName tests provider name normalization.
func TestNormalizeProviderName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"github", "github"},
		{"GitHub", "github"},
		{"GITHUB", "github"},
		{"gh", "github"},
		{"gitlab", "gitlab"},
		{"gl", "gitlab"},
		{"notion", "notion"},
		{"nt", "notion"},
		{"jira", "jira"},
		{"linear", "linear"},
		{"wrike", "wrike"},
		{"youtrack", "youtrack"},
		{"yt", "youtrack"},
		{"bitbucket", "bitbucket"},
		{"bb", "bitbucket"},
		{"asana", "asana"},
		{"clickup", "clickup"},
		{"cu", "clickup"},
		{"trello", "trello"},
		{"azuredevops", "azuredevops"},
		{"ado", "azuredevops"},
		{"azure", "azuredevops"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeProviderName(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeProviderName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestMaskToken tests token masking.
func TestMaskToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "long token",
			token:    "ghp_1234567890abcdefghij",
			expected: "ghp_...ghij",
		},
		{
			name:     "short token",
			token:    "abc123",
			expected: "*******",
		},
		{
			name:     "exactly 8 chars",
			token:    "12345678",
			expected: "*******",
		},
		{
			name:     "empty token",
			token:    "",
			expected: "*******",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskToken(tt.token)
			if got != tt.expected {
				t.Errorf("maskToken(%q) = %q, want %q", tt.token, got, tt.expected)
			}
		})
	}
}

// TestWriteTokenToEnv tests writing tokens to .env file.
func TestWriteTokenToEnv(t *testing.T) {
	t.Run("new file", func(t *testing.T) {
		tmpDir := t.TempDir()
		envPath := filepath.Join(tmpDir, ".env")

		err := writeTokenToEnv(envPath, "GITHUB_TOKEN", "ghp_test123")
		if err != nil {
			t.Fatalf("writeTokenToEnv failed: %v", err)
		}

		// Verify the file was created
		data, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("read .env: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, "GITHUB_TOKEN=ghp_test123") {
			t.Errorf("Expected token not found in .env: %s", content)
		}

		// Verify file permissions
		info, err := os.Stat(envPath)
		if err != nil {
			t.Fatalf("stat .env: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("File permissions = %o, want %o", info.Mode().Perm(), 0o600)
		}
	})

	t.Run("append to existing", func(t *testing.T) {
		tmpDir := t.TempDir()
		envPath := filepath.Join(tmpDir, ".env")

		// Create an initial file
		err := os.WriteFile(envPath, []byte("EXISTING_VAR=value\n"), 0o600)
		if err != nil {
			t.Fatalf("create initial .env: %v", err)
		}

		err = writeTokenToEnv(envPath, "GITHUB_TOKEN", "ghp_test123")
		if err != nil {
			t.Fatalf("writeTokenToEnv failed: %v", err)
		}

		data, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("read .env: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, "EXISTING_VAR=value") {
			t.Errorf("Existing var lost: %s", content)
		}
		if !strings.Contains(content, "GITHUB_TOKEN=ghp_test123") {
			t.Errorf("New token not found: %s", content)
		}
	})

	t.Run("replace existing token", func(t *testing.T) {
		tmpDir := t.TempDir()
		envPath := filepath.Join(tmpDir, ".env")

		// Create an initial file with token
		err := os.WriteFile(envPath, []byte("GITHUB_TOKEN=old_token\n"), 0o600)
		if err != nil {
			t.Fatalf("create initial .env: %v", err)
		}

		err = writeTokenToEnv(envPath, "GITHUB_TOKEN", "new_token")
		if err != nil {
			t.Fatalf("writeTokenToEnv failed: %v", err)
		}

		data, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("read .env: %v", err)
		}

		content := string(data)
		if strings.Contains(content, "old_token") {
			t.Errorf("Old token still present: %s", content)
		}
		if !strings.Contains(content, "GITHUB_TOKEN=new_token") {
			t.Errorf("New token not found: %s", content)
		}
	})
}

// TestGetConfigToken tests extracting tokens from WorkspaceConfig.
func TestGetConfigToken(t *testing.T) {
	tests := []struct {
		name      string
		fieldPath string
		setup     func(*storage.WorkspaceConfig)
		want      string
	}{
		{
			name:      "github token",
			fieldPath: "GitHub.Token",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.GitHub = &storage.GitHubSettings{Token: "ghp_test"}
			},
			want: "ghp_test",
		},
		{
			name:      "gitlab token",
			fieldPath: "GitLab.Token",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.GitLab = &storage.GitLabSettings{Token: "glpat_test"}
			},
			want: "glpat_test",
		},
		{
			name:      "notion token",
			fieldPath: "Notion.Token",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.Notion = &storage.NotionSettings{Token: "secret_test"}
			},
			want: "secret_test",
		},
		{
			name:      "jira token",
			fieldPath: "Jira.Token",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.Jira = &storage.JiraSettings{Token: "jira_test"}
			},
			want: "jira_test",
		},
		{
			name:      "linear token",
			fieldPath: "Linear.Token",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.Linear = &storage.LinearSettings{Token: "lin_api_test"}
			},
			want: "lin_api_test",
		},
		{
			name:      "wrike token",
			fieldPath: "Wrike.Token",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.Wrike = &storage.WrikeSettings{Token: "wrike_test"}
			},
			want: "wrike_test",
		},
		{
			name:      "youtrack token",
			fieldPath: "YouTrack.Token",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.YouTrack = &storage.YouTrackSettings{Token: "yt_test"}
			},
			want: "yt_test",
		},
		{
			name:      "bitbucket app password",
			fieldPath: "Bitbucket.AppPassword",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.Bitbucket = &storage.BitbucketSettings{AppPassword: "bb_test"}
			},
			want: "bb_test",
		},
		{
			name:      "asana token",
			fieldPath: "Asana.Token",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.Asana = &storage.AsanaSettings{Token: "asana_test"}
			},
			want: "asana_test",
		},
		{
			name:      "clickup token",
			fieldPath: "ClickUp.Token",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.ClickUp = &storage.ClickUpSettings{Token: "cu_test"}
			},
			want: "cu_test",
		},
		{
			name:      "trello token",
			fieldPath: "Trello.Token",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.Trello = &storage.TrelloSettings{Token: "trello_test"}
			},
			want: "trello_test",
		},
		{
			name:      "azuredevops token",
			fieldPath: "AzureDevOps.Token",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.AzureDevOps = &storage.AzureDevOpsSettings{Token: "ado_test"}
			},
			want: "ado_test",
		},
		{
			name:      "nil provider config",
			fieldPath: "GitHub.Token",
			setup:     func(cfg *storage.WorkspaceConfig) {},
			want:      "",
		},
		{
			name:      "invalid field path",
			fieldPath: "Invalid.Field",
			setup:     func(cfg *storage.WorkspaceConfig) {},
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := storage.NewDefaultWorkspaceConfig()
			tt.setup(cfg)

			got := getConfigToken(cfg, tt.fieldPath)
			if got != tt.want {
				t.Errorf("getConfigToken(%q) = %q, want %q", tt.fieldPath, got, tt.want)
			}
		})
	}
}

// TestLoadEnv tests loading .env file contents.
func TestLoadEnv(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		ws := openTestWorkspace(t, tmpDir)

		env, err := ws.LoadEnv()
		if err != nil {
			t.Fatalf("LoadEnv failed: %v", err)
		}
		if len(env) != 0 {
			t.Errorf("LoadEnv returned %d vars, want 0", len(env))
		}
	})

	t.Run("parse env file", func(t *testing.T) {
		tmpDir := t.TempDir()
		ws := openTestWorkspace(t, tmpDir)

		// Ensure a directory exists
		if err := ws.EnsureInitialized(); err != nil {
			t.Fatalf("EnsureInitialized: %v", err)
		}

		envPath := ws.EnvPath()
		content := `# Comment
VAR1=value1
VAR2=value2

  VAR3  =  value3
`
		if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
			t.Fatalf("write .env: %v", err)
		}

		env, err := ws.LoadEnv()
		if err != nil {
			t.Fatalf("LoadEnv failed: %v", err)
		}

		if env["VAR1"] != "value1" {
			t.Errorf("VAR1 = %q, want %q", env["VAR1"], "value1")
		}
		if env["VAR2"] != "value2" {
			t.Errorf("VAR2 = %q, want %q", env["VAR2"], "value2")
		}
		if env["VAR3"] != "value3" {
			t.Errorf("VAR3 = %q, want %q", env["VAR3"], "value3")
		}
	})
}

// TestWriteTokenReferenceToConfig tests writing ${VAR} references to config.
func TestWriteTokenReferenceToConfig(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		envVar     string
		tokenValue string // Dummy value for env var expansion
		setup      func(*storage.WorkspaceConfig)
		verify     func(*testing.T, *storage.WorkspaceConfig)
		wantErr    bool
	}{
		{
			name:       "github - creates provider section",
			provider:   "github",
			envVar:     "GITHUB_TOKEN",
			tokenValue: "test_gh_token",
			setup:      func(cfg *storage.WorkspaceConfig) {},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.GitHub == nil {
					t.Error("GitHub config should not be nil")

					return
				}
				if cfg.GitHub.Token != "test_gh_token" {
					t.Errorf("GitHub.Token = %q, want test_gh_token", cfg.GitHub.Token)
				}
			},
		},
		{
			name:       "gitlab - creates provider section",
			provider:   "gitlab",
			envVar:     "GITLAB_TOKEN",
			tokenValue: "test_gl_token",
			setup:      func(cfg *storage.WorkspaceConfig) {},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.GitLab == nil {
					t.Error("GitLab config should not be nil")

					return
				}
				if cfg.GitLab.Token != "test_gl_token" {
					t.Errorf("GitLab.Token = %q, want test_gl_token", cfg.GitLab.Token)
				}
			},
		},
		{
			name:       "notion - creates provider section",
			provider:   "notion",
			envVar:     "NOTION_TOKEN",
			tokenValue: "test_notion_token",
			setup:      func(cfg *storage.WorkspaceConfig) {},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Notion == nil {
					t.Error("Notion config should not be nil")

					return
				}
				if cfg.Notion.Token != "test_notion_token" {
					t.Errorf("Notion.Token = %q, want test_notion_token", cfg.Notion.Token)
				}
			},
		},
		{
			name:       "jira - creates provider section",
			provider:   "jira",
			envVar:     "JIRA_TOKEN",
			tokenValue: "test_jira_token",
			setup:      func(cfg *storage.WorkspaceConfig) {},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Jira == nil {
					t.Error("Jira config should not be nil")

					return
				}
				if cfg.Jira.Token != "test_jira_token" {
					t.Errorf("Jira.Token = %q, want test_jira_token", cfg.Jira.Token)
				}
			},
		},
		{
			name:       "linear - creates provider section",
			provider:   "linear",
			envVar:     "LINEAR_API_KEY",
			tokenValue: "test_linear_key",
			setup:      func(cfg *storage.WorkspaceConfig) {},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Linear == nil {
					t.Error("Linear config should not be nil")

					return
				}
				if cfg.Linear.Token != "test_linear_key" {
					t.Errorf("Linear.Token = %q, want test_linear_key", cfg.Linear.Token)
				}
			},
		},
		{
			name:       "wrike - creates provider section",
			provider:   "wrike",
			envVar:     "WRIKE_TOKEN",
			tokenValue: "test_wrike_token",
			setup:      func(cfg *storage.WorkspaceConfig) {},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Wrike == nil {
					t.Error("Wrike config should not be nil")

					return
				}
				if cfg.Wrike.Token != "test_wrike_token" {
					t.Errorf("Wrike.Token = %q, want test_wrike_token", cfg.Wrike.Token)
				}
			},
		},
		{
			name:       "youtrack - creates provider section",
			provider:   "youtrack",
			envVar:     "YOUTRACK_TOKEN",
			tokenValue: "test_yt_token",
			setup:      func(cfg *storage.WorkspaceConfig) {},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.YouTrack == nil {
					t.Error("YouTrack config should not be nil")

					return
				}
				if cfg.YouTrack.Token != "test_yt_token" {
					t.Errorf("YouTrack.Token = %q, want test_yt_token", cfg.YouTrack.Token)
				}
			},
		},
		{
			name:       "bitbucket - creates provider section",
			provider:   "bitbucket",
			envVar:     "BITBUCKET_APP_PASSWORD",
			tokenValue: "test_bb_password",
			setup:      func(cfg *storage.WorkspaceConfig) {},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Bitbucket == nil {
					t.Error("Bitbucket config should not be nil")

					return
				}
				if cfg.Bitbucket.AppPassword != "test_bb_password" {
					t.Errorf("Bitbucket.AppPassword = %q, want test_bb_password", cfg.Bitbucket.AppPassword)
				}
			},
		},
		{
			name:       "asana - creates provider section",
			provider:   "asana",
			envVar:     "ASANA_TOKEN",
			tokenValue: "test_asana_token",
			setup:      func(cfg *storage.WorkspaceConfig) {},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Asana == nil {
					t.Error("Asana config should not be nil")

					return
				}
				if cfg.Asana.Token != "test_asana_token" {
					t.Errorf("Asana.Token = %q, want test_asana_token", cfg.Asana.Token)
				}
			},
		},
		{
			name:       "clickup - creates provider section",
			provider:   "clickup",
			envVar:     "CLICKUP_TOKEN",
			tokenValue: "test_cu_token",
			setup:      func(cfg *storage.WorkspaceConfig) {},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.ClickUp == nil {
					t.Error("ClickUp config should not be nil")

					return
				}
				if cfg.ClickUp.Token != "test_cu_token" {
					t.Errorf("ClickUp.Token = %q, want test_cu_token", cfg.ClickUp.Token)
				}
			},
		},
		{
			name:       "trello - creates provider section",
			provider:   "trello",
			envVar:     "TRELLO_TOKEN",
			tokenValue: "test_trello_token",
			setup:      func(cfg *storage.WorkspaceConfig) {},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Trello == nil {
					t.Error("Trello config should not be nil")

					return
				}
				if cfg.Trello.Token != "test_trello_token" {
					t.Errorf("Trello.Token = %q, want test_trello_token", cfg.Trello.Token)
				}
			},
		},
		{
			name:       "azuredevops - creates provider section",
			provider:   "azuredevops",
			envVar:     "AZURE_DEVOPS_PAT",
			tokenValue: "test_ado_token",
			setup:      func(cfg *storage.WorkspaceConfig) {},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.AzureDevOps == nil {
					t.Error("AzureDevOps config should not be nil")

					return
				}
				if cfg.AzureDevOps.Token != "test_ado_token" {
					t.Errorf("AzureDevOps.Token = %q, want test_ado_token", cfg.AzureDevOps.Token)
				}
			},
		},
		{
			name:       "github - replaces existing token",
			provider:   "github",
			envVar:     "GITHUB_TOKEN",
			tokenValue: "new_token_value",
			setup: func(cfg *storage.WorkspaceConfig) {
				cfg.GitHub = &storage.GitHubSettings{Token: "old_plaintext_token"}
			},
			verify: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.GitHub.Token != "new_token_value" {
					t.Errorf("GitHub.Token = %q, want new_token_value", cfg.GitHub.Token)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			ws := openTestWorkspace(t, tmpDir)

			// Load and modify config
			cfg, err := ws.LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig: %v", err)
			}

			tt.setup(cfg)

			// Save modified config
			if err := ws.SaveConfig(cfg); err != nil {
				t.Fatalf("SaveConfig: %v", err)
			}

			// Run writeTokenReferenceToConfig
			err = writeTokenReferenceToConfig(ws, tt.provider, tt.envVar)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeTokenReferenceToConfig() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			// Set env var so LoadConfig will expand it correctly
			t.Setenv(tt.envVar, tt.tokenValue)

			// Verify the change
			updatedCfg, err := ws.LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig after update: %v", err)
			}

			tt.verify(t, updatedCfg)
		})
	}
}
