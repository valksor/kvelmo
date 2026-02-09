package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/coordination"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/validation"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/display"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Commands for managing and validating mehrhof configuration.`,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration files",
	Long: `Validate workspace configuration (.mehrhof/config.yaml).

Performs the following checks:
  - YAML syntax validity
  - Required fields and valid enum values
  - Agent alias circular dependencies
  - Undefined agent references
  - Git pattern template validity
  - Plugin configuration

Examples:
  mehr config validate                    # Validate workspace config
  mehr config validate --strict           # Treat warnings as errors
  mehr config validate --format json      # JSON output for CI`,
	RunE: runConfigValidate,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new .mehrhof/config.yaml file",
	Long: `Create a new workspace configuration file with sensible defaults.

This command creates .mehrhof/config.yaml in the current directory with
default settings for git, agent, workflow, and plugins.

If a config file already exists, you'll be prompted to overwrite it
(unless --force is used).

The generated config includes helpful comments showing all available
options and examples for customization.

Examples:
  mehr config init              # Create config with interactive prompt
  mehr config init --force      # Overwrite existing config without prompt`,
	RunE: runConfigInit,
}

var configReinitCmd = &cobra.Command{
	Use:   "reinit",
	Short: "Re-initialize config while preserving key settings",
	Long: `Re-initialize workspace configuration with the latest schema.

This command updates your config to the current version while preserving
important settings:
  - Agent defaults and aliases
  - Git patterns (branch, commit prefix)
  - Provider configurations (GitHub, Jira, etc.)
  - Project settings (code_dir)
  - Environment variables

Use this when 'mehr config validate' reports an outdated config version.

Examples:
  mehr config reinit        # Re-init with confirmation
  mehr config reinit --yes  # Skip confirmation prompt`,
	RunE: runConfigReinit,
}

var configExplainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Explain configuration resolution",
	Long: `Show how agent configuration is resolved for a workflow step.

Displays the 7-level priority resolution path showing which configuration
source wins and why. This helps debug agent selection issues.

Priority order:
  1. CLI step-specific flag (--agent-plan, --agent-implement, --agent-review)
  2. CLI global flag (--agent)
  3. Task frontmatter step-specific (agent_steps.planning.agent)
  4. Task frontmatter default (agent)
  5. Workspace config step-specific (agent.steps.planning.name)
  6. Workspace config default (agent.default)
  7. Auto-detect (first available agent)

Examples:
  mehr config explain --agent planning      # Show agent for planning step
  mehr config explain --agent implementing   # Show agent for implementing step
  mehr config explain --agent reviewing      # Show agent for reviewing step`,
	RunE: runConfigExplain,
}

var (
	validateStrict     bool
	validateFormat     string
	configInitForce    bool
	configInitProject  string // Project type hint (go, node, python, php)
	configReinitYes    bool
	configExplainAgent string
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configReinitCmd)
	configCmd.AddCommand(configExplainCmd)

	configValidateCmd.Flags().BoolVar(&validateStrict, "strict", false,
		"Treat warnings as errors (exit code 1 if warnings present)")
	configValidateCmd.Flags().StringVar(&validateFormat, "format", "text",
		"Output format: text, json")

	configInitCmd.Flags().BoolVarP(&configInitForce, "force", "f", false,
		"Overwrite existing config file without prompting")
	configInitCmd.Flags().StringVar(&configInitProject, "project", "",
		"Project type for intelligent defaults (go, node, python, php)")

	configReinitCmd.Flags().BoolVarP(&configReinitYes, "yes", "y", false,
		"Skip confirmation prompt")

	configExplainCmd.Flags().StringVar(&configExplainAgent, "agent", "",
		"Workflow step to explain (planning, implementing, reviewing)")
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get a working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Get built-in agent names by initializing conductor
	// We use WithAutoInit(false) to avoid requiring a task context
	builtInAgents := getBuiltInAgents(ctx)

	// Create validator
	validator := validation.New(wd, validation.Options{
		Strict: validateStrict,
	})
	validator.SetBuiltInAgents(builtInAgents)

	// Print header
	if validateFormat == "text" {
		fmt.Println("Validating workspace configuration...")
		fmt.Println()
	}

	// Run validation
	result, err := validator.Validate(ctx)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Output results
	fmt.Print(result.Format(validateFormat))

	// Return error for exit code handling
	if !result.Valid {
		return errors.New("validation failed")
	}
	if validateStrict && result.Warnings > 0 {
		return fmt.Errorf("validation failed: %d warning(s) in strict mode", result.Warnings)
	}

	return nil
}

// getBuiltInAgents returns the list of built-in agent names.
// It attempts to initialize the conductor to get the full registry,
// falling back to hardcoded defaults if initialization fails.
func getBuiltInAgents(ctx context.Context) []string {
	// Try to get from the conductor registry
	cond, err := initializeConductor(ctx, conductor.WithAutoInit(false))
	if err == nil {
		return cond.GetAgentRegistry().List()
	}

	// Fall back to known built-in agents
	return []string{"claude"}
}

// runConfigInit creates a new workspace configuration file.
func runConfigInit(cmd *cobra.Command, args []string) error {
	// Get a working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Open workspace to find a config path
	ws, err := storage.OpenWorkspace(context.Background(), wd, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	configPath := ws.ConfigPath()

	// Check if config already exists
	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
	}

	if configExists {
		fmt.Printf(display.Warning("WARNING: Config file already exists: %s\n"), configPath)
		fmt.Println()
		fmt.Println("The 'mehr config init' command is for creating NEW configurations.")
		fmt.Println("Your existing config contains custom settings that would be lost.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Printf("  %s - View your current config\n", display.Cyan("cat .mehrhof/config.yaml"))
		fmt.Printf("  %s - Validate your current config\n", display.Cyan("mehr config validate"))
		fmt.Printf("  %s - Edit your current config\n", display.Cyan("vim .mehrhof/config.yaml"))
		fmt.Println()

		if !configInitForce {
			return nil // Safe default: do nothing
		}

		// With --force, require extra explicit confirmation
		fmt.Println(display.ErrorMsg("--force flag: This will DELETE your custom configuration!"))
		confirmed, err := confirmAction("Type 'yes' to confirm deletion and overwrite", false)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Cancelled - your config is safe")

			return nil
		}
		fmt.Println("Proceeding with overwrite...")
	} else {
		fmt.Printf("Creating new configuration: %s\n", configPath)
	}

	// Detect the project type if not specified
	projectType := configInitProject
	if projectType == "" {
		projectType = detectProjectType(wd)
		if projectType != "" {
			fmt.Printf("Detected project type: %s\n", display.Cyan(projectType))
		}
	}

	// Create default config
	cfg := storage.NewDefaultWorkspaceConfig()

	// Apply project-specific customizations
	applyProjectCustomizations(cfg, projectType)

	// Ensure .mehrhof directory exists
	if err := os.MkdirAll(ws.TaskRoot(), 0o755); err != nil {
		return fmt.Errorf("create .mehrhof directory: %w", err)
	}

	// Save config
	if err := ws.SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println(display.SuccessMsg("Configuration created successfully"))
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  %s - Edit the configuration\n", display.Cyan("vim .mehrhof/config.yaml"))
	fmt.Printf("  %s - Validate the configuration\n", display.Cyan("mehr config validate"))
	fmt.Printf("  %s - Start your first task\n", display.Cyan("mehr start task.md"))

	return nil
}

// runConfigReinit re-initializes the workspace config with preserved values.
func runConfigReinit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Open workspace
	ws, err := storage.OpenWorkspace(ctx, wd, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	// Check if config exists
	configPath := ws.ConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("No config file found.")
		fmt.Printf("Run %s to create a new configuration.\n", display.Cyan("mehr config init"))

		return nil
	}

	// Load current config
	oldCfg, err := ws.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Check version status
	status := storage.CheckConfigVersion(oldCfg)
	if !status.IsOutdated {
		fmt.Printf("%s Config is already up to date (version %d)\n",
			display.Success("✓"), status.Current)

		return nil
	}

	// Show version info
	fmt.Printf("Config version: %d (current: %d)\n", status.Current, status.Required)
	fmt.Println()
	fmt.Println("The following settings will be preserved:")
	fmt.Println("  - Agent defaults and aliases")
	fmt.Println("  - Git patterns (branch, commit prefix)")
	fmt.Println("  - Provider configurations")
	fmt.Println("  - Project settings (code_dir)")
	fmt.Println("  - Environment variables")
	fmt.Println()

	// Prompt for confirmation
	if !configReinitYes {
		confirmed, err := confirmAction("Re-initialize config?", false)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Cancelled")

			return nil
		}
	}

	// Re-initialize with preserved values
	newCfg := storage.ReinitConfig(oldCfg)
	if err := ws.SaveConfig(newCfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("%s Config re-initialized to version %d\n",
		display.Success("✓"), newCfg.Version)
	fmt.Println()
	fmt.Println("Run", display.Cyan("mehr config validate"), "to verify the new configuration.")

	return nil
}

// detectProjectType attempts to detect the project type from common files.
func detectProjectType(dir string) string {
	// Check for Go projects
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return "go"
	}

	// Check for Node.js projects (package.json)
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		return "node"
	}

	// Check for Python projects (pyproject.toml, requirements.txt, setup.py)
	for _, name := range []string{"pyproject.toml", "requirements.txt", "setup.py", "setup.cfg"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return "python"
		}
	}

	// Check for PHP projects (composer.json)
	if _, err := os.Stat(filepath.Join(dir, "composer.json")); err == nil {
		return "php"
	}

	return ""
}

// runConfigExplain shows the agent resolution path for a given step.
func runConfigExplain(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate step argument
	if configExplainAgent == "" {
		return errors.New("required flag --agent not specified (planning, implementing, reviewing)")
	}

	// Map step string to workflow.Step
	var step workflow.Step
	switch configExplainAgent {
	case "planning":
		step = workflow.StepPlanning
	case "implementing", "implementation":
		step = workflow.StepImplementing
	case "reviewing", "review":
		step = workflow.StepReviewing
	default:
		return fmt.Errorf("invalid step: %s (must be planning, implementing, or reviewing)", configExplainAgent)
	}

	// Initialize conductor to get registry and resolver
	cond, err := initializeConductor(ctx, conductor.WithAutoInit(false))
	if err != nil {
		return fmt.Errorf("initialize conductor: %w", err)
	}

	// Get workspace config
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	ws, err := storage.OpenWorkspace(ctx, wd, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		// Continue without config - will show as skipped
		cfg = nil
	}

	// Get agent registry
	agents := cond.GetAgentRegistry()

	// Create resolver
	resolver := coordination.NewResolver(agents, ws)

	// Build resolution request (no CLI flags, no task config)
	req := coordination.ResolveRequest{
		WorkspaceCfg: cfg,
		Step:         step,
	}

	// Get explanation
	explanation, err := resolver.ExplainAgentResolution(ctx, req)
	if err != nil {
		return fmt.Errorf("explain resolution: %w", err)
	}

	// Print output
	fmt.Printf("Effective agent for %s: %s\n\n", display.Cyan(explanation.Step), display.Bold(explanation.Effective))
	fmt.Printf("Source: %s\n\n", explanation.Source)

	fmt.Println("Resolution path:")
	for _, step := range explanation.AllSteps {
		if step.Skipped {
			fmt.Printf("  %d. %s %s\n", step.Priority, display.Muted(step.Source), display.Muted("(not set)"))
		} else {
			marker := " "
			if step.Agent == explanation.Effective {
				marker = display.Success("✓")
			}
			fmt.Printf("  %d. %s %s: %s %s\n", step.Priority, marker, display.Bold(step.Source), display.Cyan(step.Agent), display.Muted("(selected)"))
		}
	}

	fmt.Println()
	fmt.Println("To override:")
	fmt.Printf("  %s %s\n", display.Cyan("mehr plan"), display.Cyan("--agent-"+configExplainAgent+" <agent-name>"))

	return nil
}

// applyProjectCustomizations customizes the config based on the project type.
func applyProjectCustomizations(cfg *storage.WorkspaceConfig, projectType string) {
	switch projectType {
	case "go":
		// Go projects: use claude as default, enable go-specific agents
		cfg.Agent.Default = "claude"
		cfg.Git.CommitPrefix = "[{key}]"
		cfg.Git.BranchPattern = "{type}/{key}--{slug}"

	case "node":
		// Node.js projects: optimize for JS/TS workflows
		cfg.Agent.Default = "claude"
		cfg.Git.CommitPrefix = "feat({key}):"
		cfg.Git.BranchPattern = "{type}/{key}--{slug}"

	case "python":
		// Python projects: conventional commits style
		cfg.Agent.Default = "claude"
		cfg.Git.CommitPrefix = "[{key}]"
		cfg.Git.BranchPattern = "{type}/{key}--{slug}"

	case "php":
		// PHP projects: typical PHP conventions
		cfg.Agent.Default = "claude"
		cfg.Git.CommitPrefix = "[{key}]"
		cfg.Git.BranchPattern = "{type}/{key}--{slug}"
	}
}
