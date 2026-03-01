package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/settings"
	"golang.org/x/term"
)

// ProviderLoginConfig holds configuration for a provider's login command.
type ProviderLoginConfig struct {
	Name        string // Display name (e.g., "GitHub")
	EnvVar      string // Environment variable name (e.g., "GITHUB_TOKEN")
	HelpURL     string // URL for getting a token
	HelpSteps   string // Navigation steps to get a token
	Scopes      string // Required permissions/scopes
	TokenPrefix string // Optional: expected token prefix (informational only)
}

// providerLoginConfigs maps provider names to their login configuration.
var providerLoginConfigs = map[string]ProviderLoginConfig{
	"github": {
		Name:        "GitHub",
		EnvVar:      "GITHUB_TOKEN",
		HelpURL:     "https://github.com/settings/tokens",
		HelpSteps:   "Settings -> Developer settings -> Personal access tokens -> Tokens (classic)",
		Scopes:      "repo, read:user (or Fine-grained with repository access)",
		TokenPrefix: "ghp_",
	},
	"gitlab": {
		Name:        "GitLab",
		EnvVar:      "GITLAB_TOKEN",
		HelpURL:     "https://gitlab.com/-/user_settings/personal_access_tokens",
		HelpSteps:   "Preferences -> Access Tokens -> Add new token",
		Scopes:      "api, read_user, read_repository",
		TokenPrefix: "glpat-",
	},
	"linear": {
		Name:        "Linear",
		EnvVar:      "LINEAR_TOKEN",
		HelpURL:     "https://linear.app/settings/api",
		HelpSteps:   "Settings -> API -> Personal API keys -> Create key",
		Scopes:      "Workspace access",
		TokenPrefix: "lin_api_",
	},
	"wrike": {
		Name:        "Wrike",
		EnvVar:      "WRIKE_TOKEN",
		HelpURL:     "https://www.wrike.com/frontend/apps/index.html#/api",
		HelpSteps:   "Apps & Integrations -> API -> Permanent access tokens",
		Scopes:      "Default (read/write access)",
		TokenPrefix: "",
	},
}

// tokenSource represents where a token value was found.
type tokenSource struct {
	Source string // Description of where the token was found
	Value  string // The masked token value
}

// detectExistingToken checks if a token is already configured.
func detectExistingToken(envVar string, scope settings.Scope, projectRoot string) *tokenSource {
	// Check system environment variable
	if val := os.Getenv(envVar); val != "" {
		return &tokenSource{
			Source: envVar + " environment variable",
			Value:  settings.MaskToken(val),
		}
	}

	// Check the appropriate .env file based on scope
	var envPath string
	if scope == settings.ScopeProject && projectRoot != "" {
		envPath = settings.ProjectEnvPath(projectRoot)
	} else {
		var err error
		envPath, err = settings.GlobalEnvPath()
		if err != nil {
			return nil
		}
	}

	// Read and check .env file
	if token := readEnvVar(envPath, envVar); token != "" {
		return &tokenSource{
			Source: envPath,
			Value:  settings.MaskToken(token),
		}
	}

	return nil
}

// readEnvVar reads a single environment variable from a .env file.
func readEnvVar(path, key string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		if strings.TrimSpace(line[:idx]) == key {
			value := strings.TrimSpace(line[idx+1:])
			// Remove surrounding quotes
			if len(value) >= 2 {
				if (value[0] == '"' && value[len(value)-1] == '"') ||
					(value[0] == '\'' && value[len(value)-1] == '\'') {
					value = value[1 : len(value)-1]
				}
			}

			return value
		}
	}

	// Check for scanner errors (I/O failures)
	if err := scanner.Err(); err != nil {
		return ""
	}

	return ""
}

// readToken reads a token from stdin, using secure input when available.
func readToken(prompt string) (string, error) {
	fmt.Print(prompt)

	// Check if stdin is a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Non-interactive: read from stdin
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}

		return strings.TrimSpace(line), nil
	}

	// Interactive: use secure password input
	tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Move to next line after password entry
	if err != nil {
		return "", fmt.Errorf("read token: %w", err)
	}

	return strings.TrimSpace(string(tokenBytes)), nil
}

// confirmOverride asks the user if they want to replace an existing token.
func confirmOverride(cmd *cobra.Command) bool {
	_, _ = fmt.Fprint(cmd.OutOrStdout(), "Override? [y/N]: ")

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))

	return response == "y" || response == "yes"
}

// printTokenHelp displays formatted guidance for getting a token.
func printTokenHelp(w io.Writer, cfg ProviderLoginConfig) {
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "%s Token Setup\n", cfg.Name)
	_, _ = fmt.Fprintln(w, "--------------------------------------------------")
	_, _ = fmt.Fprintf(w, "Get token: %s\n", cfg.HelpURL)
	if cfg.HelpSteps != "" {
		_, _ = fmt.Fprintf(w, "Steps:     %s\n", cfg.HelpSteps)
	}
	if cfg.Scopes != "" {
		_, _ = fmt.Fprintf(w, "Required:  %s\n", cfg.Scopes)
	}
	if cfg.TokenPrefix != "" {
		_, _ = fmt.Fprintf(w, "Format:    Token starts with '%s'\n", cfg.TokenPrefix)
	}
	_, _ = fmt.Fprintln(w, "--------------------------------------------------")
	_, _ = fmt.Fprintln(w)
}

// runProviderLogin executes the login flow for a provider.
func runProviderLogin(providerName string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		cfg, ok := providerLoginConfigs[providerName]
		if !ok {
			return fmt.Errorf("unknown provider: %s", providerName)
		}

		// Determine scope
		useProject, _ := cmd.Flags().GetBool("project")
		scope := settings.ScopeGlobal
		projectRoot := ""

		if useProject {
			scope = settings.ScopeProject
			var err error
			projectRoot, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
		}

		// Check for existing token
		existing := detectExistingToken(cfg.EnvVar, scope, projectRoot)
		if existing != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Token already configured via %s: %s\n", existing.Source, existing.Value)
			if !confirmOverride(cmd) {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")

				return nil
			}
		}

		// Print help
		printTokenHelp(cmd.OutOrStdout(), cfg)

		// Prompt for token
		token, err := readToken(fmt.Sprintf("Enter your %s API token: ", cfg.Name))
		if err != nil {
			return err
		}

		if token == "" {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")

			return nil
		}

		// Optional: warn about token prefix (informational only)
		if cfg.TokenPrefix != "" && !strings.HasPrefix(token, cfg.TokenPrefix) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Note: Token doesn't start with expected prefix '%s'\n", cfg.TokenPrefix)
		}

		// Save token
		if err := settings.SaveEnvVar(scope, projectRoot, cfg.EnvVar, token); err != nil {
			return fmt.Errorf("save token: %w", err)
		}

		// Print success
		var envPath string
		if scope == settings.ScopeProject {
			envPath = settings.ProjectEnvPath(projectRoot)
		} else {
			envPath, _ = settings.GlobalEnvPath()
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nToken saved to %s\n", envPath)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Use '%s start <task>' to begin working.\n", meta.Name)

		return nil
	}
}

// createProviderCommand creates a provider command with a login subcommand.
func createProviderCommand(providerName string) *cobra.Command {
	cfg := providerLoginConfigs[providerName]

	providerCmd := &cobra.Command{
		Use:   providerName,
		Short: cfg.Name + " provider commands",
	}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with " + cfg.Name,
		Long: fmt.Sprintf(`Authenticate with %s by providing an API token.

The token is securely stored in a .env file:
  - Global (default): ~/.valksor/%s/.env
  - Project (--project): .valksor/.env

Get your token at: %s`, cfg.Name, meta.Name, cfg.HelpURL),
		RunE: runProviderLogin(providerName),
	}

	loginCmd.Flags().Bool("project", false, "Save token to project .valksor/.env instead of global")

	providerCmd.AddCommand(loginCmd)

	return providerCmd
}

// Provider commands exported for registration in main.go.
var (
	GitHubCmd = createProviderCommand("github")
	GitLabCmd = createProviderCommand("gitlab")
	LinearCmd = createProviderCommand("linear")
	WrikeCmd  = createProviderCommand("wrike")
)
