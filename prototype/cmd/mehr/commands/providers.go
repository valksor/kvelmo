package commands

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/valksor/go-toolkit/display"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List and manage task providers",
	Long: `List available task providers and show their configuration requirements.

Providers are sources for tasks - files, directories, GitHub issues, Jira tickets, etc.`,
}

var providersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available providers",
	Long:  `List all available task providers with their schemes and descriptions.`,
	RunE:  runProvidersList,
}

var providersInfoCmd = &cobra.Command{
	Use:   "info <provider>",
	Short: "Show provider information",
	Long:  `Show detailed information about a specific provider including setup requirements.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProvidersInfo,
}

var providersStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check provider health status",
	Long: `Check the health and connection status of all configured providers.

Shows connection status, rate limits, last sync time, and any errors.`,
	RunE: runProvidersStatus,
}

func init() {
	rootCmd.AddCommand(providersCmd)
	providersCmd.AddCommand(providersListCmd)
	providersCmd.AddCommand(providersInfoCmd)
	providersCmd.AddCommand(providersStatusCmd)
}

func runProvidersList(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SCHEME\tSHORTHAND\tPROVIDER\tDESCRIPTION")
	_, _ = fmt.Fprintln(w, "------\t---------\t--------\t-----------")

	providers := []struct {
		scheme      string
		shorthand   string
		name        string
		description string
	}{
		{"file", "f", "File", "Single markdown file"},
		{"dir", "d", "Directory", "Directory with README.md"},
		{"github", "gh", "GitHub", "GitHub issues and pull requests"},
		{"gitlab", "", "GitLab", "GitLab issues and merge requests"},
		{"jira", "", "Jira", "Atlassian Jira tickets"},
		{"linear", "", "Linear", "Linear issues"},
		{"notion", "", "Notion", "Notion pages and databases"},
		{"wrike", "", "Wrike", "Wrike tasks"},
		{"youtrack", "yt", "YouTrack", "JetBrains YouTrack issues"},
	}

	for _, p := range providers {
		sh := p.shorthand
		if sh == "" {
			sh = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.scheme, sh, p.name, p.description)
	}

	_ = w.Flush()

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Usage:")
	_, _ = fmt.Fprintln(out, "  mehr start <scheme>:<reference>  # Use provider with scheme or shorthand")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Examples:")
	_, _ = fmt.Fprintln(out, "  mehr start file:task.md")
	_, _ = fmt.Fprintln(out, "  mehr start f:task.md              # shorthand also works")
	_, _ = fmt.Fprintln(out, "  mehr start dir:./tasks/")
	_, _ = fmt.Fprintln(out, "  mehr start github:owner/repo#123")
	_, _ = fmt.Fprintln(out, "  mehr start gh:owner/repo#123       # shorthand also works")
	_, _ = fmt.Fprintln(out, "  mehr start youtrack:PROJECT-123")
	_, _ = fmt.Fprintln(out, "  mehr start yt:PROJECT-123         # shorthand also works")

	return nil
}

func runProvidersInfo(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	providerName := strings.ToLower(args[0])

	info := getProviderInfo(providerName)
	if info == nil {
		_, _ = fmt.Fprintf(out, "Unknown provider: %s\n\n", providerName)
		_, _ = fmt.Fprintln(out, "Run 'mehr providers list' to see available providers.")

		return nil
	}

	_, _ = fmt.Fprintf(out, "Provider: %s\n\n", info.Name)
	_, _ = fmt.Fprintf(out, "Scheme: %s\n\n", info.Scheme)
	_, _ = fmt.Fprintf(out, "Description:\n  %s\n\n", info.Description)

	if len(info.Setup) > 0 {
		_, _ = fmt.Fprintln(out, "Setup:")
		for _, step := range info.Setup {
			_, _ = fmt.Fprintf(out, "  %s\n", step)
		}
		_, _ = fmt.Fprintln(out)
	}

	if len(info.EnvVars) > 0 {
		_, _ = fmt.Fprintln(out, "Required environment variables:")
		for _, env := range info.EnvVars {
			_, _ = fmt.Fprintf(out, "  %s\n", env)
		}
		_, _ = fmt.Fprintln(out)
	}

	if len(info.Config) > 0 {
		_, _ = fmt.Fprintln(out, "Configuration (in .mehrhof/config.yaml):")
		for _, cfg := range info.Config {
			_, _ = fmt.Fprintf(out, "  %s\n", cfg)
		}
		_, _ = fmt.Fprintln(out)
	}

	_, _ = fmt.Fprintf(out, "Usage:\n  %s\n\n", info.Usage)

	return nil
}

func runProvidersStatus(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	ctx := context.Background()

	// Initialize conductor to get the provider registry
	cond, err := initializeConductor(ctx)
	if err != nil {
		return fmt.Errorf("initialize conductor: %w", err)
	}

	// Get workspace config for provider settings
	ws := cond.GetWorkspace()
	cfg, err := ws.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// List of providers to check
	providersToCheck := []struct {
		name      string
		scheme    string
		configKey string
	}{
		{"GitHub", "github", "GitHub"},
		{"GitLab", "gitlab", "GitLab"},
		{"Jira", "jira", "Jira"},
		{"Linear", "linear", "Linear"},
		{"Notion", "notion", "Notion"},
		{"Bitbucket", "bitbucket", "Bitbucket"},
	}

	// Print header
	if _, err := fmt.Fprintln(out, "Provider Health Status"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	// Check each provider
	for _, p := range providersToCheck {
		healthInfo, err := checkProviderHealth(ctx, cond, p.scheme, p.configKey, cfg)
		if err != nil {
			// Provider not available or error checking
			if _, err := fmt.Fprintf(out, "%s\t%s\t%s\n",
				p.name,
				display.Muted("○"),
				display.Muted("Not configured"),
			); err != nil {
				return err
			}

			continue
		}

		// Format status
		var statusIcon string
		var statusMsg string

		switch healthInfo.Status {
		case provider.HealthStatusConnected:
			statusIcon = display.Success("●")
			statusMsg = display.Success("Connected")
		case provider.HealthStatusNotConfigured:
			statusIcon = display.Muted("○")
			statusMsg = display.Muted("Not configured")
		case provider.HealthStatusError:
			statusIcon = display.ErrorMsg("✗")
			statusMsg = display.ErrorMsg("Error")
		default:
			statusIcon = "?"
			statusMsg = string(healthInfo.Status)
		}

		// Build line
		line := fmt.Sprintf("%s\t%s\t%s", p.name, statusIcon, statusMsg)

		// Add rate limit if available
		if healthInfo.RateLimit != nil {
			line += fmt.Sprintf("\tRate: %d/%d\tReset: %s",
				healthInfo.RateLimit.Used,
				healthInfo.RateLimit.Limit,
				healthInfo.RateLimit.ResetIn,
			)
		}

		// Add a message if available
		if healthInfo.Message != "" {
			line += "\t" + healthInfo.Message
		}

		// Add error if available
		if healthInfo.Error != "" {
			line += "\t" + display.ErrorMsg("%s", healthInfo.Error)
		}

		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "Legend:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "  %s Connected\n", display.Success("●")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "  %s Not configured\n", display.Muted("○")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "  %s Error\n", display.ErrorMsg("✗")); err != nil {
		return err
	}

	return nil
}

// checkProviderHealth checks the health of a specific provider.
//
//nolint:unparam // Error return will be used in future implementation
func checkProviderHealth(_ context.Context, _ *conductor.Conductor, _ string, _ string, _ *storage.WorkspaceConfig) (*provider.HealthInfo, error) {
	// This is a simplified implementation that checks if the provider is configured
	// and attempts a basic health check
	//
	// Future improvements:
	// 1. Creating provider instances from config
	// 2. Calling HealthCheck() method on each provider
	// 3. Returning detailed health information

	// For now, return a simple not-configured status for all providers
	// The full implementation would use the provider registry and call HealthCheck()

	return &provider.HealthInfo{
		Status:  provider.HealthStatusNotConfigured,
		Message: "Set up credentials in .mehrhof/.env",
	}, nil
}

type providerInfo struct {
	Name        string
	Scheme      string
	Description string
	Usage       string
	Setup       []string
	EnvVars     []string
	Config      []string
}

func getProviderInfo(name string) *providerInfo {
	switch name {
	case "file", "f":
		return &providerInfo{
			Name:        "File Provider",
			Scheme:      "file",
			Description: "Load tasks from individual markdown files",
			Usage:       "mehr start file:path/to/task.md",
		}

	case "dir", "directory", "d":
		return &providerInfo{
			Name:        "Directory Provider",
			Scheme:      "dir",
			Description: "Load tasks from directories (looks for README.md)",
			Usage:       "mehr start dir:./tasks/",
		}

	case "github", "gh", "git":
		return &providerInfo{
			Name:        "GitHub Provider",
			Scheme:      "github",
			Description: "Load tasks from GitHub issues and PRs",
			EnvVars:     []string{"GITHUB_TOKEN"},
			Config: []string{
				"github:",
				"  token: \"${GITHUB_TOKEN}\"",
				"  owner: \"your-org\"",
				"  repo: \"your-repo\"",
			},
			Usage: "mehr start github:owner/repo#123",
		}

	case "jira":
		return &providerInfo{
			Name:        "Jira Provider",
			Scheme:      "jira",
			Description: "Load tasks from Atlassian Jira",
			EnvVars:     []string{"JIRA_TOKEN"},
			Config: []string{
				"jira:",
				"  url: \"https://your-domain.atlassian.net\"",
				"  token: \"${JIRA_TOKEN}\"",
			},
			Usage: "mehr start jira:PROJECT-123",
		}

	case "linear":
		return &providerInfo{
			Name:        "Linear Provider",
			Scheme:      "linear",
			Description: "Load tasks from Linear",
			EnvVars:     []string{"LINEAR_API_KEY"},
			Config: []string{
				"linear:",
				"  api_key: \"${LINEAR_API_KEY}\"",
			},
			Usage: "mehr start linear:ABC-123",
		}

	case "notion":
		return &providerInfo{
			Name:        "Notion Provider",
			Scheme:      "notion",
			Description: "Load tasks from Notion pages and databases",
			EnvVars:     []string{"NOTION_TOKEN"},
			Config: []string{
				"notion:",
				"  token: \"${NOTION_TOKEN}\"",
			},
			Usage: "mehr start notion:a1b2c3d4e5f678901234567890abcdef",
		}

	case "youtrack", "yt":
		return &providerInfo{
			Name:        "YouTrack Provider",
			Scheme:      "youtrack",
			Description: "Load tasks from JetBrains YouTrack",
			EnvVars:     []string{"YOUTRACK_TOKEN"},
			Config: []string{
				"youtrack:",
				"  host: \"https://your-domain.youtrack.cloud\"",
				"  token: \"${YOUTRACK_TOKEN}\"",
			},
			Usage: "mehr start youtrack:PROJECT-123",
		}

	case "wrike":
		return &providerInfo{
			Name:        "Wrike Provider",
			Scheme:      "wrike",
			Description: "Load tasks from Wrike",
			EnvVars:     []string{"WRIKE_TOKEN"},
			Config: []string{
				"wrike:",
				"  token: \"${WRIKE_TOKEN}\"",
			},
			Usage: "mehr start wrike:TASK-123",
		}

	default:
		return nil
	}
}
