package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
)

var (
	initInteractive bool
	// initASC is deliberately undocumented - sets up ASC-compatible config.
	// See ASC-COMPATIBILITY.md for details.
	initASC bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the task workspace",
	Long: `Initialize the task workspace by creating the .mehrhof directory
and updating .gitignore.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", false, "Interactive setup for API key, provider, and agent")

	// Deliberately undocumented flag for ASC compatibility.
	// Hidden from help output but still functional.
	// See ASC-COMPATIBILITY.md for details on what this configures.
	initCmd.Flags().BoolVar(&initASC, "asc", false, "")
	_ = initCmd.Flags().MarkHidden("asc")

	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	// Try to find git root, fall back to the current directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}

	git, err := vcs.New(ctx, workDir)
	root := workDir
	if err == nil {
		root = git.Root()
	}

	ws, err := storage.OpenWorkspace(ctx, root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	if err := ws.EnsureInitialized(); err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}

	// Update .gitignore with standard entries
	if err := ws.UpdateGitignore(); err != nil {
		return fmt.Errorf("update .gitignore: %w", err)
	}

	// Create a config file with defaults if it doesn't exist
	if !ws.HasConfig() {
		cfg := storage.NewDefaultWorkspaceConfig()

		// Apply ASC-compatible settings if requested (deliberately undocumented)
		if initASC {
			applyASCConfig(cfg)
		}

		if err := ws.SaveConfig(cfg); err != nil {
			return fmt.Errorf("create config file: %w", err)
		}
		_, _ = fmt.Fprintf(out, "Created config file: %s\n", ws.ConfigPath())
		if initASC {
			_, _ = fmt.Fprintln(out, "Applied ASC-compatible configuration.")
		}
	} else {
		_, _ = fmt.Fprintf(out, "Config file already exists: %s\n", ws.ConfigPath())
		// Still apply ASC config to an existing file if requested
		if initASC {
			cfg, err := ws.LoadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			applyASCConfig(cfg)
			if err := ws.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			_, _ = fmt.Fprintln(out, "Applied ASC-compatible configuration.")
		}
	}

	// Create an .env template if it doesn't exist
	envPath := filepath.Join(ws.TaskRoot(), ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if err := createEnvTemplate(envPath); err != nil {
			_, _ = fmt.Fprintf(errOut, "warning: failed to create .env template: %v\n", err)
		} else {
			_, _ = fmt.Fprintf(out, "Created .env template: %s\n", envPath)
		}
	}

	// Run interactive setup if requested
	if initInteractive {
		if err := interactiveSetup(cmd, ws); err != nil {
			_, _ = fmt.Fprintf(errOut, "warning: interactive setup failed: %v\n", err)
		}
	}

	_, _ = fmt.Fprintf(out, "Workspace initialized in %s\n", root)

	// Show a welcome message and next steps
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Welcome to Mehrhof!")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Quick start:")
	_, _ = fmt.Fprintln(out, "  1. Start your first task:")
	_, _ = fmt.Fprintln(out, "     mehr start file:task.md")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "  2. Create specifications:")
	_, _ = fmt.Fprintln(out, "     mehr plan")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "  3. Implement the specifications:")
	_, _ = fmt.Fprintln(out, "     mehr implement")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Prerequisite: The default 'claude' agent uses the Claude CLI.")
	_, _ = fmt.Fprintln(out, "Install from: https://claude.ai/claude-code")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Run `mehr help` for configuration options.")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Note: Workspace data is stored in your home directory:")
	_, _ = fmt.Fprintf(out, "     %s\n", ws.TaskRoot())
	_, _ = fmt.Fprintln(out)

	return nil
}

func createEnvTemplate(path string) error {
	template := `# Mehrhof environment variables
# This file is gitignored - store secrets here safely.
# System environment variables take priority over values defined here.

# For custom agents defined in config.yaml (not needed for default claude agent)
# ANTHROPIC_API_KEY=sk-ant-...
# GLM_API_KEY=your-key-here

# Provider tokens (GitHub, GitLab, Jira, etc.)
# GITHUB_TOKEN=ghp_...
`

	return os.WriteFile(path, []byte(template), 0o600) // 0600 for secrets
}

// interactiveSetup guides the user through initial configuration.
func interactiveSetup(cmd *cobra.Command, ws *storage.Workspace) error {
	out := cmd.OutOrStdout()
	in := bufio.NewReader(cmd.InOrStdin())

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Interactive Setup")
	_, _ = fmt.Fprintln(out, "-----------------")

	// Step 1: Default Provider
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Available providers: file, dir, github, jira, linear, notion, wrike, youtrack")
	_, _ = fmt.Fprintf(out, "Enter default provider [file]: ")
	provider, _ := in.ReadString('\n')
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "file"
	}

	// Step 2: Default Agent
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Available built-in agents: claude")
	_, _ = fmt.Fprintf(out, "Enter default agent [claude]: ")
	agent, _ := in.ReadString('\n')
	agent = strings.TrimSpace(agent)

	if agent == "" {
		agent = "claude"
	}

	if agent == "glm" {
		_, _ = fmt.Fprintln(out, "Note: 'glm' is not a built-in agent. You'll need to configure it as an alias in config.yaml.")
	}

	// Update config.yaml with user's choices
	cfg, err := ws.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Agent and Providers are value types in WorkspaceConfig, not pointers
	cfg.Providers.Default = provider
	cfg.Agent.Default = agent

	if err := ws.SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Configuration saved!")

	return nil
}

// applyASCConfig configures the workspace to match ASC patterns.
// This is deliberately undocumented - see ASC-COMPATIBILITY.md for details.
//
// ASC patterns:
//   - Branch: <type>/<ticket-id>/<slug> (e.g., feat/WRIKE-123/add-auth)
//   - Commit prefix: <type>(<ticket-id>): (e.g., feat(1564280896): message)
//   - Specs: tickets/<task-id>/SPEC-N.md
//   - Reviews: tickets/<task-id>/CODERABBIT-N.txt
//   - Timezone: Europe/Riga
func applyASCConfig(cfg *storage.WorkspaceConfig) {
	cfg.Git.BranchPattern = "{type}/{key}/{slug}"
	cfg.Git.CommitPrefix = "{type}({key}):"
	cfg.Storage.SaveInProject = true
	cfg.Storage.ProjectDir = "tickets"
	cfg.Specification.FilenamePattern = "SPEC-{n}.md"
	cfg.Review.FilenamePattern = "CODERABBIT-{n}.txt"

	if cfg.Display == nil {
		cfg.Display = &storage.DisplaySettings{}
	}
	cfg.Display.Timezone = "Europe/Riga"
}
