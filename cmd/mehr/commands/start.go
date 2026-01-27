package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/browser"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/template"
)

var (
	startAgent         string
	startNoBranch      bool
	startWorktree      bool
	startStash         bool   // Stash uncommitted changes before creating branch
	startKey           string // External key override (e.g., "FEATURE-123")
	startTitle         string // Title override for the task
	startSlug          string // Slug override for branch naming
	startCommitPrefix  string // Commit prefix template override
	startBranchPattern string // Branch pattern template override
	startTemplate      string // Template to apply

	// Per-step agent overrides.
	startAgentPlanning     string
	startAgentImplementing string
	startAgentReviewing    string
)

var startCmd = &cobra.Command{
	Use:   "start <reference>",
	Short: "Start a new task from a file, directory, or provider",
	Long: `Start a new task from a file, directory, or external provider.

This command reads the source, creates a git branch, and sets up the task
as active. It does NOT run planning - use 'mehr plan' for that.

USAGE:
  mehr start <reference>

PROVIDERS:
  empty:A-1                 Empty task (add description with 'mehr note')
  empty:"Implement auth"    Empty task with description as title
  file:task.md              Markdown file (default, can omit 'file:')
  dir:./tasks/              Directory of markdown files
  github:123                GitHub issue (requires configuration)
  notion:abc123 / nt:       Notion page by ID or URL
  jira:PROJ-123             Jira issue (requires configuration)
  linear:ABC-123            Linear issue (requires configuration)
  wrike:abc123              Wrike task (requires configuration)
  youtrack:PROJ-123         YouTrack issue (requires configuration)

AGENT SELECTION (highest to lowest priority):
  1. CLI flag: --agent or --agent-plan/--agent-implement/--agent-review
  2. Task frontmatter: agent: or agent_steps: in markdown
  3. Workspace config: agent.default or agent.steps in .mehrhof/config.yaml
  4. Auto-detect: uses default claude agent

GIT OPTIONS:
  --no-branch               Do not create a git branch
  --worktree                Create isolated git worktree (allows parallel tasks)
  --stash                   Stash uncommitted changes before creating branch

PER-STEP AGENT FLAGS:
  --agent-plan              Agent for planning step (overrides default)
  --agent-implement         Agent for implementation step (overrides default)
  --agent-review            Agent for review step (overrides default)

TEMPLATES:
  --template bug-fix        Apply bug-fix template
  --template feature        Apply feature template
  --template refactor       Apply refactor template
  mehr templates            List all available templates

EXAMPLES:
  mehr start empty:FEATURE-1      # Create empty task, then use 'mehr note'
  mehr start empty:"Implement auth"  # Create with descriptive title
  mehr start file:task.md         # Start from a markdown file
  mehr start dir:./tasks/         # Start from a directory
  mehr start --no-branch task.md  # Start without creating a branch
  mehr start --worktree task.md   # Start with a separate worktree
  mehr start --template bug-fix file:task.md  # Apply bug-fix template

See also:
  mehr plan                 - Create implementation specifications
  mehr status               - Show active task status
  mehr templates            - List available task templates`,
	Args: cobra.ExactArgs(1),
	RunE: runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringVarP(&startAgent, "agent", "A", "", "Agent to use (default: auto-detect)")
	startCmd.Flags().BoolVar(&startNoBranch, "no-branch", false, "Do not create a git branch")
	startCmd.Flags().BoolVarP(&startWorktree, "worktree", "w", false, "Create a separate git worktree for this task")
	startCmd.Flags().BoolVar(&startStash, "stash", false, "Stash uncommitted changes before creating branch")

	// Naming override flags
	startCmd.Flags().StringVarP(&startKey, "key", "k", "", "External key for branch/commit naming (e.g., FEATURE-123)")
	startCmd.Flags().StringVar(&startTitle, "title", "", "Task title override")
	startCmd.Flags().StringVar(&startSlug, "slug", "", "Branch slug override (e.g., custom-slug)")
	startCmd.Flags().StringVar(&startCommitPrefix, "commit-prefix", "", "Commit prefix template (e.g., [{key}])")
	startCmd.Flags().StringVar(&startBranchPattern, "branch-pattern", "", "Branch pattern template (e.g., {type}/{key}--{slug})")
	startCmd.Flags().StringVar(&startTemplate, "template", "", "Template to apply (bug-fix, feature, refactor, docs, test, chore)")

	// Per-step agent overrides
	startCmd.Flags().StringVar(&startAgentPlanning, "agent-plan", "", "Agent for planning step")
	startCmd.Flags().StringVar(&startAgentImplementing, "agent-implement", "", "Agent for implementation step")
	startCmd.Flags().StringVar(&startAgentReviewing, "agent-review", "", "Agent for review step")
}

func runStart(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	reference := args[0]

	// Apply template if specified (only works for file: provider)
	if startTemplate != "" {
		if !strings.HasPrefix(reference, "file:") {
			return errors.New("--template only works with file: provider\n\nExample: mehr start --template bug-fix file:task.md")
		}

		filePath := strings.TrimPrefix(reference, "file:")
		tpl, err := template.LoadBuiltIn(startTemplate)
		if err != nil {
			return fmt.Errorf("load template: %w", err)
		}

		// Read existing content
		var content string
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		content = string(data)

		// Apply template and write back
		newContent := tpl.ApplyToContent(content)
		if err := os.WriteFile(filePath, []byte(newContent), 0o644); err != nil {
			return fmt.Errorf("write file with template: %w", err)
		}

		fmt.Printf("Applied template '%s' to %s\n", startTemplate, filePath)
	}

	// Determine branch behavior
	// Branch creation is default, --no-branch disables it
	// --worktree forces branch creation (even with --no-branch)
	createBranch := !startNoBranch || startWorktree

	// Build conductor options
	opts := []conductor.Option{
		conductor.WithVerbose(verbose),
		conductor.WithCreateBranch(createBranch),
		conductor.WithUseWorktree(startWorktree),
		conductor.WithStashChanges(startStash),
		conductor.WithAutoInit(true),
	}

	if startAgent != "" {
		opts = append(opts, conductor.WithAgent(startAgent))
	}

	// Per-step agent options
	if startAgentPlanning != "" {
		opts = append(opts, conductor.WithStepAgent("planning", startAgentPlanning))
	}
	if startAgentImplementing != "" {
		opts = append(opts, conductor.WithStepAgent("implementing", startAgentImplementing))
	}
	if startAgentReviewing != "" {
		opts = append(opts, conductor.WithStepAgent("reviewing", startAgentReviewing))
	}

	// Pass default provider from workspace config
	if wd, err := os.Getwd(); err == nil {
		if ws, err := storage.OpenWorkspace(context.Background(), wd, nil); err == nil {
			if wsCfg, err := ws.LoadConfig(); err == nil {
				// Apply default provider from config if not set via flag
				if wsCfg.Providers.Default != "" {
					opts = append(opts, conductor.WithDefaultProvider(wsCfg.Providers.Default))
				}
				// Apply stash-on-start from config if not explicitly set via flag
				if !startStash && wsCfg.Git.StashOnStart {
					opts = append(opts, conductor.WithStashChanges(true))
					// Also apply auto-pop-stash setting from config
					opts = append(opts, conductor.WithAutoPopStash(wsCfg.Git.AutoPopStash))
				}
				// If stash was explicitly set via flag, apply auto-pop-stash from config
				if startStash {
					opts = append(opts, conductor.WithAutoPopStash(wsCfg.Git.AutoPopStash))
				}
				// Apply browser configuration if enabled
				if wsCfg.Browser != nil && wsCfg.Browser.Enabled {
					browserCfg := browser.Config{
						Host:             wsCfg.Browser.Host,
						Port:             wsCfg.Browser.Port,
						Headless:         wsCfg.Browser.Headless,
						IgnoreCertErrors: wsCfg.Browser.IgnoreCertErrors,
						Timeout:          time.Duration(wsCfg.Browser.Timeout) * time.Second,
						ScreenshotDir:    wsCfg.Browser.ScreenshotDir,
					}
					// Set defaults if not specified
					if browserCfg.Host == "" {
						browserCfg.Host = browser.DefaultConfig().Host
					}
					if browserCfg.ScreenshotDir == "" {
						browserCfg.ScreenshotDir = browser.DefaultConfig().ScreenshotDir
					}
					if browserCfg.Timeout == 0 {
						browserCfg.Timeout = browser.DefaultConfig().Timeout
					}
					// Normalize IgnoreCertErrors from defaults (use CLI flag for strict mode)
					if !browserCfg.IgnoreCertErrors {
						browserCfg.IgnoreCertErrors = browser.DefaultConfig().IgnoreCertErrors
					}
					opts = append(opts, conductor.WithBrowserConfig(browserCfg))
				}
				// Apply sandbox configuration if enabled
				if wsCfg.Sandbox != nil && wsCfg.Sandbox.Enabled {
					opts = append(opts, conductor.WithSandbox(true))
				}
			}
		}
	}

	// CLI flag overrides config for sandbox
	if sandbox {
		opts = append(opts, conductor.WithSandbox(true))
	}

	// Naming override options
	if startKey != "" {
		opts = append(opts, conductor.WithExternalKey(startKey))
	}
	if startTitle != "" {
		opts = append(opts, conductor.WithTitleOverride(startTitle))
	}
	if startSlug != "" {
		opts = append(opts, conductor.WithSlugOverride(startSlug))
	}
	if startCommitPrefix != "" {
		opts = append(opts, conductor.WithCommitPrefixTemplate(startCommitPrefix))
	}
	if startBranchPattern != "" {
		opts = append(opts, conductor.WithBranchPatternTemplate(startBranchPattern))
	}

	// Initialize conductor with standard providers and agents
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	// Check for existing task
	if cond.GetActiveTask() != nil {
		return fmt.Errorf("task already active: %s\n\nOptions:\n  mehr status   - View task details\n  mehr finish   - Complete the task\n  mehr abandon  - Cancel and start fresh", cond.GetActiveTask().ID)
	}

	// Check for existing work directories from finished tasks
	workDirs, err := cond.ListExistingWorkDirs()
	if err == nil && len(workDirs) > 0 {
		fmt.Printf("Found %d previous task(s) with existing work directories:\n", len(workDirs))
		for _, taskID := range workDirs {
			// Try to load work to get title
			if work, err := cond.GetWorkspace().LoadWork(taskID); err == nil {
				fmt.Printf("  - %s: %s\n", taskID, work.Metadata.Title)
			} else {
				fmt.Printf("  - %s\n", taskID)
			}
		}

		fmt.Println("\nOptions:")
		fmt.Println("  [d]elete and archive - Archive old work, start fresh (recommended)")
		fmt.Println("  [c]ontinue with existing - Reuse directory, reset to idle state")

		// Read user choice
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\nYour choice [D/c]: ")
		choice, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read choice: %w", err)
		}
		choice = strings.ToLower(strings.TrimSpace(choice))

		switch choice {
		case "", "d", "delete":
			// Archive all existing work directories
			for _, taskID := range workDirs {
				if err := cond.ArchiveWorkDir(taskID); err != nil {
					fmt.Printf("Warning: failed to archive %s: %v\n", taskID, err)
				}
			}
			fmt.Println("\nArchived existing work directories")
		case "c", "continue":
			// Continue with first existing work directory
			existingTaskID := workDirs[0]
			fmt.Printf("\nReusing existing work directory: %s\n", existingTaskID)

			if err := cond.ContinueWithExisting(ctx, reference, existingTaskID); err != nil {
				return fmt.Errorf("continue with existing: %w", err)
			}

			// Get status and display
			status, err := cond.Status()
			if err != nil {
				return err
			}

			info := display.TaskInfo{
				TaskID:      status.TaskID,
				Title:       status.Title,
				ExternalKey: status.ExternalKey,
				State:       status.State,
				Source:      status.Ref,
				Branch:      status.Branch,
				Worktree:    status.WorktreePath,
			}
			displayOpts := display.DefaultTaskInfoOptions()
			displayOpts.ShowStarted = false
			displayOpts.Compact = true
			fmt.Print(display.FormatTaskInfo("Task resumed", info, displayOpts))

			// Show next steps
			steps := []display.NextStep{
				{Command: "mehr plan", Description: "Create implementation specifications"},
				{Command: "mehr note", Description: "Add notes to the task"},
			}
			if status.WorktreePath != "" {
				steps = append([]display.NextStep{
					{Command: "cd " + status.WorktreePath, Description: "Switch to the worktree"},
				}, steps...)
			}
			fmt.Print(display.FormatNextSteps(steps))

			return nil
		default:
			return fmt.Errorf("invalid choice: %s (please run 'mehr start' again)", choice)
		}
	}

	// Start (register) task
	if err := cond.Start(ctx, reference); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	// Get status
	status, err := cond.Status()
	if err != nil {
		return err
	}

	// Display task info
	info := display.TaskInfo{
		TaskID:      status.TaskID,
		Title:       status.Title,
		ExternalKey: status.ExternalKey,
		State:       status.State,
		Source:      status.Ref,
		Branch:      status.Branch,
		Worktree:    status.WorktreePath,
	}
	displayOpts := display.DefaultTaskInfoOptions()
	displayOpts.ShowStarted = false // Not relevant for just-started task
	displayOpts.Compact = true      // Don't need state description on start
	fmt.Print(display.FormatTaskInfo("Task started", info, displayOpts))

	// Show next steps
	steps := []display.NextStep{
		{Command: "mehr plan", Description: "Create implementation specifications"},
		{Command: "mehr note", Description: "Add notes to the task"},
	}
	if status.WorktreePath != "" {
		// Prepend worktree cd command
		steps = append([]display.NextStep{
			{Command: "cd " + status.WorktreePath, Description: "Switch to the worktree"},
		}, steps...)
	}
	fmt.Print(display.FormatNextSteps(steps))

	return nil
}
