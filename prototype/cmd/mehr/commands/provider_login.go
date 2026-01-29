package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// providerLoginConfig holds configuration for a provider's login command.
type providerLoginConfig struct {
	Name        string // Display name (e.g., "GitHub")
	EnvVar      string // Environment variable name (e.g., "GITHUB_TOKEN")
	ConfigField string // Field path in WorkspaceConfig (e.g., "GitHub.Token")
	HelpURL     string // URL for getting a token
	TokenPrefix string // Optional token prefix for validation (e.g., "ghp_")
	HelpSteps   string // Navigation steps to get token
	Scopes      string // Required permissions/scopes
}

// providerLoginConfigs maps provider names to their login configuration.
var providerLoginConfigs = map[string]providerLoginConfig{
	"github": {
		Name:        "GitHub",
		EnvVar:      "GITHUB_TOKEN",
		ConfigField: "GitHub.Token",
		HelpURL:     "https://github.com/settings/tokens",
		TokenPrefix: "ghp_",
		HelpSteps:   "Settings → Developer settings → Personal access tokens → Tokens (classic)",
		Scopes:      "repo, read:user (or Fine-grained with repository access)",
	},
	"gitlab": {
		Name:        "GitLab",
		EnvVar:      "GITLAB_TOKEN",
		ConfigField: "GitLab.Token",
		HelpURL:     "https://gitlab.com/-/user_settings/personal_access_tokens",
		TokenPrefix: "glpat-",
		HelpSteps:   "Preferences → Access Tokens → Add new token",
		Scopes:      "api, read_user, read_repository",
	},
	"notion": {
		Name:        "Notion",
		EnvVar:      "NOTION_TOKEN",
		ConfigField: "Notion.Token",
		HelpURL:     "https://www.notion.so/my-integrations",
		TokenPrefix: "secret_",
		HelpSteps:   "Settings → My connections → Develop or manage integrations",
		Scopes:      "Database access to your workspace",
	},
	"jira": {
		Name:        "Jira",
		EnvVar:      "JIRA_TOKEN",
		ConfigField: "Jira.Token",
		HelpURL:     "https://id.atlassian.com/manage-profile/security/api-tokens",
		TokenPrefix: "",
		HelpSteps:   "Profile → Security → Create and manage API tokens",
		Scopes:      "Full account access (Jira Cloud)",
	},
	"linear": {
		Name:        "Linear",
		EnvVar:      "LINEAR_API_KEY",
		ConfigField: "Linear.Token",
		HelpURL:     "https://linear.app/settings/api",
		TokenPrefix: "lin_api_",
		HelpSteps:   "Settings → API → Personal API keys → Create key",
		Scopes:      "Workspace access",
	},
	"wrike": {
		Name:        "Wrike",
		EnvVar:      "WRIKE_TOKEN",
		ConfigField: "Wrike.Token",
		HelpURL:     "https://www.wrike.com/frontend/apps/index.html#/api",
		TokenPrefix: "",
		HelpSteps:   "Profile → Apps & Integrations → API",
		Scopes:      "Full access (permanent token)",
	},
	"youtrack": {
		Name:        "YouTrack",
		EnvVar:      "YOUTRACK_TOKEN",
		ConfigField: "YouTrack.Token",
		HelpURL:     "https://www.jetbrains.com/help/youtrack/manage-user-token.html",
		TokenPrefix: "",
		HelpSteps:   "Profile → Account Security → Tokens → New token",
		Scopes:      "Hub service scope for your project",
	},
	"bitbucket": {
		Name:        "Bitbucket",
		EnvVar:      "BITBUCKET_APP_PASSWORD",
		ConfigField: "Bitbucket.AppPassword",
		HelpURL:     "https://bitbucket.org/account/settings/app-passwords",
		TokenPrefix: "",
		HelpSteps:   "Settings → App passwords → Create app password",
		Scopes:      "Repositories: read, write; Pull requests: read, write",
	},
	"asana": {
		Name:        "Asana",
		EnvVar:      "ASANA_TOKEN",
		ConfigField: "Asana.Token",
		HelpURL:     "https://app.asana.com/0/developer-console",
		TokenPrefix: "",
		HelpSteps:   "Profile → Apps → Developer Console → Personal access token",
		Scopes:      "Full access to your Asana account",
	},
	"clickup": {
		Name:        "ClickUp",
		EnvVar:      "CLICKUP_TOKEN",
		ConfigField: "ClickUp.Token",
		HelpURL:     "https://app.clickup.com/settings/apps",
		TokenPrefix: "",
		HelpSteps:   "Settings → Apps → Generate API Token",
		Scopes:      "Full access to your workspace",
	},
	"trello": {
		Name:        "Trello",
		EnvVar:      "TRELLO_TOKEN",
		ConfigField: "Trello.Token",
		HelpURL:     "https://trello.com/power-ups/admin",
		TokenPrefix: "",
		HelpSteps:   "Power-Ups Admin → Developer API Keys → Generate Token",
		Scopes:      "Read/write boards (also set TRELLO_KEY for API key)",
	},
	"azuredevops": {
		Name:        "Azure DevOps",
		EnvVar:      "AZURE_DEVOPS_PAT",
		ConfigField: "AzureDevOps.Token",
		HelpURL:     "https://dev.azure.com/_usersSettings/tokens",
		TokenPrefix: "",
		HelpSteps:   "User Settings → Personal access tokens → New Token",
		Scopes:      "Work Items: read/write; Code: read",
	},
}

// getProviderLoginConfig returns the login config for a provider, or nil if not found.
func getProviderLoginConfig(name string) *providerLoginConfig {
	// Normalize aliases
	normalized := normalizeProviderName(name)
	if cfg, ok := providerLoginConfigs[normalized]; ok {
		return &cfg
	}

	return nil
}

// normalizeProviderName converts provider aliases to canonical names.
func normalizeProviderName(name string) string {
	switch strings.ToLower(name) {
	case "gh", "git":
		return "github"
	case "gl":
		return "gitlab"
	case "nt":
		return "notion"
	case "yt":
		return "youtrack"
	case "bb":
		return "bitbucket"
	case "ado", "azure":
		return "azuredevops"
	case "cu":
		return "clickup"
	default:
		return strings.ToLower(name)
	}
}

// tokenSource represents where a token value was found.
type tokenSource struct {
	Source string // Description of where token was found
	Value  string // The token value (possibly masked)
}

// detectExistingToken checks if a token is already configured.
// Returns (source, value) or ("", "") if not found.
func detectExistingToken(cfg providerLoginConfig, ws *storage.Workspace) *tokenSource {
	// 1. Check system environment variable
	if val := os.Getenv(cfg.EnvVar); val != "" {
		return &tokenSource{Source: cfg.EnvVar + " environment variable", Value: maskToken(val)}
	}

	// 2. Check .env file
	envVars, err := ws.LoadEnv()
	if err == nil {
		if val, ok := envVars[cfg.EnvVar]; ok && val != "" {
			return &tokenSource{Source: ".mehrhof/.env file", Value: maskToken(val)}
		}
	}

	// 3. Check config.yaml
	workspaceCfg, err := ws.LoadConfig()
	if err == nil {
		val := getConfigToken(workspaceCfg, cfg.ConfigField)
		if val != "" {
			return &tokenSource{Source: "config.yaml", Value: maskToken(val)}
		}
	}

	return nil
}

// getConfigToken retrieves a token from WorkspaceConfig using field path (e.g., "GitHub.Token").
func getConfigToken(cfg *storage.WorkspaceConfig, fieldPath string) string {
	parts := strings.Split(fieldPath, ".")
	if len(parts) != 2 {
		return ""
	}

	switch parts[0] {
	case "GitHub":
		if cfg.GitHub != nil && parts[1] == "Token" {
			return cfg.GitHub.Token
		}
	case "GitLab":
		if cfg.GitLab != nil && parts[1] == "Token" {
			return cfg.GitLab.Token
		}
	case "Notion":
		if cfg.Notion != nil && parts[1] == "Token" {
			return cfg.Notion.Token
		}
	case "Jira":
		if cfg.Jira != nil && parts[1] == "Token" {
			return cfg.Jira.Token
		}
	case "Linear":
		if cfg.Linear != nil && parts[1] == "Token" {
			return cfg.Linear.Token
		}
	case "Wrike":
		if cfg.Wrike != nil && parts[1] == "Token" {
			return cfg.Wrike.Token
		}
	case "YouTrack":
		if cfg.YouTrack != nil && parts[1] == "Token" {
			return cfg.YouTrack.Token
		}
	case "Bitbucket":
		if cfg.Bitbucket != nil && parts[1] == "AppPassword" {
			return cfg.Bitbucket.AppPassword
		}
	case "Asana":
		if cfg.Asana != nil && parts[1] == "Token" {
			return cfg.Asana.Token
		}
	case "ClickUp":
		if cfg.ClickUp != nil && parts[1] == "Token" {
			return cfg.ClickUp.Token
		}
	case "Trello":
		if cfg.Trello != nil && parts[1] == "Token" {
			return cfg.Trello.Token
		}
	case "AzureDevOps":
		if cfg.AzureDevOps != nil && parts[1] == "Token" {
			return cfg.AzureDevOps.Token
		}
	}

	return ""
}

// maskToken returns a masked version of a token for display.
func maskToken(token string) string {
	if len(token) <= 8 {
		return "*******"
	}

	return token[:4] + "..." + token[len(token)-4:]
}

// confirmOverride asks the user if they want to replace an existing token.
func confirmOverride(cmd *cobra.Command, source, maskedValue string) (bool, error) {
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Token already configured via %s: %s\n", source, maskedValue)
	_, _ = fmt.Fprintf(out, "Override? [y/N]: ")

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))

	return response == "y" || response == "yes", nil
}

// printTokenHelp displays formatted guidance for obtaining a token.
func printTokenHelp(cmd *cobra.Command, cfg providerLoginConfig) {
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintf(out, "%s Token Setup\n", cfg.Name)
	_, _ = fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintf(out, "📍 Get token: %s\n", cfg.HelpURL)
	if cfg.HelpSteps != "" {
		_, _ = fmt.Fprintf(out, "📋 Steps:     %s\n", cfg.HelpSteps)
	}
	if cfg.Scopes != "" {
		_, _ = fmt.Fprintf(out, "🔑 Required:  %s\n", cfg.Scopes)
	}
	if cfg.TokenPrefix != "" {
		_, _ = fmt.Fprintf(out, "💡 Format:    Token starts with '%s'\n", cfg.TokenPrefix)
	}
	_, _ = fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Token will be saved to .mehrhof/.env and referenced in config.yaml")
}

// promptForToken interactively prompts the user for a token.
func promptForToken(cmd *cobra.Command, cfg providerLoginConfig) (string, error) {
	printTokenHelp(cmd, cfg)

	var token string
	prompt := &survey.Password{
		Message: fmt.Sprintf("Enter your %s API token (leave empty to cancel):", cfg.Name),
	}

	if err := survey.AskOne(prompt, &token); err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("cancelled")
	}

	// Optional: validate token prefix
	if cfg.TokenPrefix != "" && !strings.HasPrefix(token, cfg.TokenPrefix) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Warning: Token doesn't start with expected prefix '%s'\n", cfg.TokenPrefix)
	}

	return token, nil
}

// writeTokenToEnv writes a token to the .env file, creating or updating it.
func writeTokenToEnv(envPath, key, value string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(envPath), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Read existing content
	var existingLines []string
	if data, err := os.ReadFile(envPath); err == nil {
		existingLines = strings.Split(string(data), "\n")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read .env: %w", err)
	}

	// Find and replace or append
	found := false
	var result strings.Builder

	for _, line := range existingLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			result.WriteString(key + "=" + value + "\n")
			found = true
		} else if trimmed != "" {
			result.WriteString(line + "\n")
		}
	}

	if !found {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(key + "=" + value + "\n")
	}

	// Atomic write: write to temp file then rename
	content := result.String()
	tmpPath := envPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}

	if err := os.Rename(tmpPath, envPath); err != nil {
		// Clean up temp file on error
		_ = os.Remove(tmpPath)

		return fmt.Errorf("rename .env: %w", err)
	}

	return nil
}

// writeTokenReferenceToConfig adds ${VAR} reference to config.yaml.
// Creates provider section if it doesn't exist.
func writeTokenReferenceToConfig(ws *storage.Workspace, providerName, envVar string) error {
	// Load existing config
	cfg, err := ws.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Set the token reference based on provider name
	switch providerName {
	case "github":
		if cfg.GitHub == nil {
			cfg.GitHub = &storage.GitHubSettings{}
		}
		cfg.GitHub.Token = "${" + envVar + "}"
	case "gitlab":
		if cfg.GitLab == nil {
			cfg.GitLab = &storage.GitLabSettings{}
		}
		cfg.GitLab.Token = "${" + envVar + "}"
	case "notion":
		if cfg.Notion == nil {
			cfg.Notion = &storage.NotionSettings{}
		}
		cfg.Notion.Token = "${" + envVar + "}"
	case "jira":
		if cfg.Jira == nil {
			cfg.Jira = &storage.JiraSettings{}
		}
		cfg.Jira.Token = "${" + envVar + "}"
	case "linear":
		if cfg.Linear == nil {
			cfg.Linear = &storage.LinearSettings{}
		}
		cfg.Linear.Token = "${" + envVar + "}"
	case "wrike":
		if cfg.Wrike == nil {
			cfg.Wrike = &storage.WrikeSettings{}
		}
		cfg.Wrike.Token = "${" + envVar + "}"
	case "youtrack":
		if cfg.YouTrack == nil {
			cfg.YouTrack = &storage.YouTrackSettings{}
		}
		cfg.YouTrack.Token = "${" + envVar + "}"
	case "bitbucket":
		if cfg.Bitbucket == nil {
			cfg.Bitbucket = &storage.BitbucketSettings{}
		}
		cfg.Bitbucket.AppPassword = "${" + envVar + "}"
	case "asana":
		if cfg.Asana == nil {
			cfg.Asana = &storage.AsanaSettings{}
		}
		cfg.Asana.Token = "${" + envVar + "}"
	case "clickup":
		if cfg.ClickUp == nil {
			cfg.ClickUp = &storage.ClickUpSettings{}
		}
		cfg.ClickUp.Token = "${" + envVar + "}"
	case "trello":
		if cfg.Trello == nil {
			cfg.Trello = &storage.TrelloSettings{}
		}
		cfg.Trello.Token = "${" + envVar + "}"
	case "azuredevops":
		if cfg.AzureDevOps == nil {
			cfg.AzureDevOps = &storage.AzureDevOpsSettings{}
		}
		cfg.AzureDevOps.Token = "${" + envVar + "}"
	}

	// Save config
	if err := ws.SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}

// runProviderLogin executes the login flow for a provider.
func runProviderLogin(providerName string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cfg := getProviderLoginConfig(providerName)
		if cfg == nil {
			return fmt.Errorf("unknown provider: %s", providerName)
		}

		// Get working directory
		root, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}

		ws, err := storage.OpenWorkspace(context.Background(), root, nil)
		if err != nil {
			return fmt.Errorf("open workspace: %w", err)
		}

		// Check for existing token
		existing := detectExistingToken(*cfg, ws)

		if existing != nil {
			override, err := confirmOverride(cmd, existing.Source, existing.Value)
			if err != nil {
				return err
			}
			if !override {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")

				return nil
			}
		}

		// Prompt for token
		token, err := promptForToken(cmd, *cfg)
		if err != nil && err.Error() == "cancelled" {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")

			return nil
		}
		if err != nil {
			return err
		}

		// Write to .env
		envPath := ws.EnvPath()
		if err := writeTokenToEnv(envPath, cfg.EnvVar, token); err != nil {
			return fmt.Errorf("write token to .env: %w", err)
		}

		// Write ${VAR} reference to config.yaml
		if err := writeTokenReferenceToConfig(ws, providerName, cfg.EnvVar); err != nil {
			return fmt.Errorf("write token reference to config: %w", err)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nToken saved to %s\n", envPath)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Token reference added to config.yaml\n")

		return nil
	}
}

// Create provider commands.
func createProviderCommands() []*cobra.Command {
	var commands []*cobra.Command

	for name := range providerLoginConfigs {
		providerName := name // Capture loop variable
		cmd := &cobra.Command{
			Use:    providerName,
			Short:  providerName + " provider commands",
			Hidden: true,
		}

		loginCmd := &cobra.Command{
			Use:    "login",
			Short:  "Authenticate with " + providerName,
			RunE:   runProviderLogin(providerName),
			Hidden: false,
		}

		cmd.AddCommand(loginCmd)
		commands = append(commands, cmd)
	}

	return commands
}

func init() {
	// Register all provider commands
	for _, cmd := range createProviderCommands() {
		rootCmd.AddCommand(cmd)
	}
}
