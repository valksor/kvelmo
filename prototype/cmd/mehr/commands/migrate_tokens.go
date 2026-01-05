package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// migrateTokensCmd migrates existing provider tokens to use ${VAR} syntax.
var migrateTokensCmd = &cobra.Command{
	Use:   "migrate-tokens",
	Short: "Migrate provider tokens to use ${VAR} syntax in config.yaml",
	Long: `Migrate provider tokens from plaintext to ${VAR} syntax.

This command converts provider tokens in config.yaml to use environment variable
references (${VAR} syntax). The actual token values are moved to .mehrhof/.env.

This migration is optional - plaintext tokens in config.yaml will continue to work.
The benefit of ${VAR} syntax is that config.yaml becomes the single source of truth
for all configuration, making it easier to see what tokens are configured.

Example:
  Before: config.yaml has github: { token: "ghp_abc123..." }
  After:  config.yaml has github: { token: "${GITHUB_TOKEN}" }
          .mehrhof/.env has GITHUB_TOKEN=ghp_abc123...`,
	RunE: runMigrateTokens,
}

func init() {
	rootCmd.AddCommand(migrateTokensCmd)
}

func runMigrateTokens(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get working directory
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	ws, err := storage.OpenWorkspace(ctx, root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	// Load existing config
	cfg, err := ws.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	modified := false
	envVarsToAdd := make(map[string]string)

	// Migrate GitHub token
	if cfg.GitHub != nil && cfg.GitHub.Token != "" && !strings.Contains(cfg.GitHub.Token, "${") {
		envVarsToAdd["GITHUB_TOKEN"] = cfg.GitHub.Token
		cfg.GitHub.Token = "${GITHUB_TOKEN}"
		modified = true
		fmt.Println("Migrated GitHub token to ${GITHUB_TOKEN}")
	}

	// Migrate GitLab token
	if cfg.GitLab != nil && cfg.GitLab.Token != "" && !strings.Contains(cfg.GitLab.Token, "${") {
		envVarsToAdd["GITLAB_TOKEN"] = cfg.GitLab.Token
		cfg.GitLab.Token = "${GITLAB_TOKEN}"
		modified = true
		fmt.Println("Migrated GitLab token to ${GITLAB_TOKEN}")
	}

	// Migrate Notion token
	if cfg.Notion != nil && cfg.Notion.Token != "" && !strings.Contains(cfg.Notion.Token, "${") {
		envVarsToAdd["NOTION_TOKEN"] = cfg.Notion.Token
		cfg.Notion.Token = "${NOTION_TOKEN}"
		modified = true
		fmt.Println("Migrated Notion token to ${NOTION_TOKEN}")
	}

	// Migrate Jira token
	if cfg.Jira != nil && cfg.Jira.Token != "" && !strings.Contains(cfg.Jira.Token, "${") {
		envVarsToAdd["JIRA_TOKEN"] = cfg.Jira.Token
		cfg.Jira.Token = "${JIRA_TOKEN}"
		modified = true
		fmt.Println("Migrated Jira token to ${JIRA_TOKEN}")
	}

	// Migrate Linear token
	if cfg.Linear != nil && cfg.Linear.Token != "" && !strings.Contains(cfg.Linear.Token, "${") {
		envVarsToAdd["LINEAR_API_KEY"] = cfg.Linear.Token
		cfg.Linear.Token = "${LINEAR_API_KEY}"
		modified = true
		fmt.Println("Migrated Linear token to ${LINEAR_API_KEY}")
	}

	// Migrate Wrike token
	if cfg.Wrike != nil && cfg.Wrike.Token != "" && !strings.Contains(cfg.Wrike.Token, "${") {
		envVarsToAdd["WRIKE_TOKEN"] = cfg.Wrike.Token
		cfg.Wrike.Token = "${WRIKE_TOKEN}"
		modified = true
		fmt.Println("Migrated Wrike token to ${WRIKE_TOKEN}")
	}

	// Migrate YouTrack token
	if cfg.YouTrack != nil && cfg.YouTrack.Token != "" && !strings.Contains(cfg.YouTrack.Token, "${") {
		envVarsToAdd["YOUTRACK_TOKEN"] = cfg.YouTrack.Token
		cfg.YouTrack.Token = "${YOUTRACK_TOKEN}"
		modified = true
		fmt.Println("Migrated YouTrack token to ${YOUTRACK_TOKEN}")
	}

	if !modified {
		fmt.Println("\nNo migration needed. Tokens already use ${VAR} syntax or are not configured.")

		return nil
	}

	// Write tokens to .env file
	envPath := ws.EnvPath()
	for envVar, tokenValue := range envVarsToAdd {
		if err := writeTokenToEnv(envPath, envVar, tokenValue); err != nil {
			return fmt.Errorf("write token to .env: %w", err)
		}
	}

	// Save updated config
	if err := ws.SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("\nMigration complete!\n")
	fmt.Printf("- Tokens moved to: %s\n", envPath)
	fmt.Printf("- Token references updated in config.yaml\n")
	fmt.Printf("\nNote: If you had these tokens set as environment variables in your shell,\n")
	fmt.Printf("you may want to unset them to avoid confusion:\n")
	for envVar := range envVarsToAdd {
		fmt.Printf("  unset %s\n", envVar)
	}

	return nil
}
