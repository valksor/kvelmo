package commands

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	tkdisplay "github.com/valksor/go-toolkit/display"
)

var (
	finishYes           bool
	finishMerge         bool
	finishDelete        bool
	finishPush          bool
	finishSquash        bool
	finishTargetBranch  string
	finishSkipQuality   bool
	finishQualityTarget string
	finishDeleteWork    bool
	// PR-related flags.
	finishDraftPR bool
	finishPRTitle string
	finishPRBody  string
)

var finishCmd = &cobra.Command{
	Use:   "finish",
	Short: "Complete the task (creates PR by default for supported providers)",
	Long: `Complete the current task by creating a pull request or merging locally.

ENDING A TASK:
  mehr finish    Complete task and create PR (for GitHub/GitLab) or merge locally
  mehr abandon   Discard all work and delete task branch (when task is no longer needed)

PROVIDER BEHAVIOR:
  github:     Creates a pull request automatically
  gitlab:     Creates a merge request automatically
  file:, dir: Prompts for action (merge locally / mark done / cancel)
  jira:       Prompts for action (merge locally / mark done / cancel)
  others:     Prompts for action (merge locally / mark done / cancel)

By default, this:
- Runs 'make quality' if available (code formatting, linting, etc.)
- For PR providers: creates a pull/merge request
- For other providers: prompts what to do
- Keeps the task branch (use --delete to remove it)
- Keeps the work directory (use --delete-work to remove it)
- Does NOT push after local merge (use --push to enable)

When using --merge, this performs a local merge instead of creating a PR:
- Performs a regular merge (--no-ff) to preserve history (use --squash for squash merge)
- Does NOT delete the task branch by default
- Does NOT push to remote by default

If quality checks modify files (e.g., auto-formatting), you'll be prompted
to confirm before proceeding.

FLAG COMBINATIONS:
  PR mode (default):
    --draft, --pr-title, --pr-body are allowed
    --merge is NOT allowed with these flags

  Merge mode (--merge):
    --delete, --push, --squash, --target are allowed
    --draft, --pr-title, --pr-body are NOT allowed

Examples:
  mehr finish                      # Create PR (github/gitlab) or prompt for action
  mehr finish --yes                # Skip confirmation prompt
  mehr finish --merge              # Force local merge instead of PR
  mehr finish --merge --delete     # Merge and delete task branch
  mehr finish --merge --push       # Merge and push to remote
  mehr finish --merge --squash     # Squash merge instead of regular merge
  mehr finish --target develop     # Merge to specific branch
  mehr finish --no-quality         # Skip quality checks
  mehr finish --quality-target lint # Use custom make target
  mehr finish --draft              # Create PR as draft
  mehr finish --pr-title "Fix bug" # Custom PR title
  mehr finish --delete-work        # Delete work directory after finishing`,
	RunE: runFinish,
}

func init() {
	rootCmd.AddCommand(finishCmd)

	finishCmd.Flags().BoolVarP(&finishYes, "yes", "y", false, "Skip confirmation prompt")
	finishCmd.Flags().BoolVar(&finishMerge, "merge", false, "Force local merge instead of creating PR")
	finishCmd.Flags().BoolVar(&finishDelete, "delete", false, "Delete branch after merge")
	finishCmd.Flags().BoolVar(&finishPush, "push", false, "Push to remote after local merge")
	finishCmd.Flags().BoolVar(&finishSquash, "squash", false, "Use squash merge instead of regular merge")
	finishCmd.Flags().StringVarP(&finishTargetBranch, "target", "t", "", "Target branch to merge into")
	finishCmd.Flags().BoolVar(&finishSkipQuality, "no-quality", false, "Skip quality checks (make quality)")
	finishCmd.Flags().StringVar(&finishQualityTarget, "quality-target", "quality", "Make target for quality checks")
	finishCmd.Flags().BoolVar(&finishDeleteWork, "delete-work", false, "Delete work directory after finishing")

	// PR-related flags
	finishCmd.Flags().BoolVar(&finishDraftPR, "draft", false, "Create PR as draft")
	finishCmd.Flags().StringVar(&finishPRTitle, "pr-title", "", "Custom PR title")
	finishCmd.Flags().StringVar(&finishPRBody, "pr-body", "", "Custom PR body")

	// PR flags are mutually exclusive with merge mode
	finishCmd.MarkFlagsMutuallyExclusive("merge", "draft")
	finishCmd.MarkFlagsMutuallyExclusive("merge", "pr-title")
	finishCmd.MarkFlagsMutuallyExclusive("merge", "pr-body")
}

func runFinish(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Initialize conductor with standard providers and agents
	cond, err := initializeConductor(ctx, conductor.WithVerbose(verbose))
	if err != nil {
		return err
	}

	// Check for an active task
	activeTask := cond.GetActiveTask()
	if activeTask == nil {
		fmt.Print(display.NoActiveTaskError())

		return errors.New("no active task")
	}

	// Get status for display
	status, err := cond.Status(ctx)
	if err != nil {
		return fmt.Errorf("get status: %w", err)
	}

	// Build confirmation prompt
	promptLines := "About to finish task: " + status.TaskID
	if status.Title != "" {
		promptLines += "\n  Title: " + status.Title
	}
	if status.Branch != "" {
		promptLines += "\n  Branch: " + status.Branch
	}
	promptLines += "\n  State: " + status.State
	promptLines += fmt.Sprintf("\n  Specifications: %d", status.Specifications)

	if finishMerge {
		promptLines += "\n\nThis will perform a local merge"
		if finishDelete && status.Branch != "" {
			promptLines += " and delete the task branch"
		}
		if finishPush {
			promptLines += " and push to remote"
		}
		promptLines += "."
	} else {
		promptLines += "\n\nThis will create a pull request (if provider supports it)"
		if finishDraftPR {
			promptLines += " as draft"
		}
		promptLines += "."
	}

	// Confirmation prompt (unless --yes)
	confirmed, err := confirmAction(promptLines, finishYes)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Operation cancelled")

		return nil
	}

	// Run quality checks (unless skipped)
	if !finishSkipQuality {
		qualityOpts := conductor.QualityOptions{
			Target: finishQualityTarget,
		}

		result, err := cond.RunQuality(ctx, qualityOpts)
		if err != nil {
			return fmt.Errorf("quality check: %w", err)
		}

		if result.Ran {
			if result.UserAborted {
				fmt.Println("Finish cancelled by user")

				return nil
			}

			if result.Passed {
				fmt.Println(tkdisplay.SuccessMsg("Quality checks passed"))
			}
		}
	}

	// Generate commit message preview for squash merges
	var commitMessage string
	if finishSquash {
		if msg, err := cond.GenerateCommitMessagePreview(ctx); err == nil && msg != "" {
			commitMessage = msg
			fmt.Println("\nGenerated commit message:")
			fmt.Print(tkdisplay.InfoMsg("%s", msg))
			fmt.Println()
		}
	}

	// Build finish options
	// Use tri-state for DeleteWork: nil=defer to config, true=delete, false=keep
	var deleteWork *bool
	if cmd.Flags().Changed("delete-work") {
		deleteWork = conductor.BoolPtr(finishDeleteWork)
	}

	opts := conductor.FinishOptions{
		SquashMerge:  finishSquash,
		DeleteBranch: finishDelete,
		TargetBranch: finishTargetBranch,
		PushAfter:    finishPush,
		DeleteWork:   deleteWork,
		// PR options
		ForceMerge:    finishMerge,
		DraftPR:       finishDraftPR,
		PRTitle:       finishPRTitle,
		PRBody:        finishPRBody,
		CommitMessage: commitMessage,
	}

	// Perform finish
	if err := cond.Finish(ctx, opts); errors.Is(err, conductor.ErrPendingQuestion) {
		// Provider doesn't support PRs — prompt the user for an action
		action := promptFinishAction()
		opts.FinishAction = action
		if err := cond.Finish(ctx, opts); err != nil {
			return fmt.Errorf("finish: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("finish: %w", err)
	}

	// Success message depends on what happened
	if finishMerge || opts.FinishAction == "merge" {
		fmt.Println(tkdisplay.SuccessMsg("Task completed and merged"))
	} else {
		fmt.Println(tkdisplay.SuccessMsg("Task completed"))
	}

	// Suggest next steps
	PrintNextSteps(
		"mehr list              - View other tasks",
		"mehr start <ref>       - Start a new task",
		"mehr guide             - Get context-aware help",
	)

	return nil
}

// finishOptions holds the options for the finish command logic.
// This struct enables testing without terminal I/O.
type finishOptions struct {
	skipQuality   bool
	qualityTarget string
	squash        bool
	merge         bool
	delete        bool
	push          bool
	targetBranch  string
	deleteWork    *bool // tri-state: nil=defer to config
	draftPR       bool
	prTitle       string
	prBody        string
	finishAction  string // pre-set action for non-PR providers (bypasses prompt)
}

// runFinishLogic contains the core finish logic, extracted for testing.
// It assumes confirmation has already been done by the caller.
func runFinishLogic(ctx context.Context, cond ConductorAPI, opts finishOptions, stdout io.Writer) error {
	// Check for an active task
	activeTask := cond.GetActiveTask()
	if activeTask == nil {
		return errors.New("no active task")
	}

	// Run quality checks (unless skipped)
	if !opts.skipQuality {
		qualityOpts := conductor.QualityOptions{
			Target: opts.qualityTarget,
		}

		result, err := cond.RunQuality(ctx, qualityOpts)
		if err != nil {
			return fmt.Errorf("quality check: %w", err)
		}

		if result != nil && result.Ran {
			if result.UserAborted {
				return nil // User cancelled, not an error
			}

			if result.Passed && stdout != nil {
				_, _ = fmt.Fprintln(stdout, tkdisplay.SuccessMsg("Quality checks passed"))
			}
		}
	}

	// Generate commit message preview for squash merges
	var commitMessage string
	if opts.squash {
		if msg, err := cond.GenerateCommitMessagePreview(ctx); err == nil && msg != "" {
			commitMessage = msg
			if stdout != nil {
				_, _ = fmt.Fprintln(stdout, "\nGenerated commit message:")
				_, _ = fmt.Fprint(stdout, tkdisplay.InfoMsg("%s", msg))
				_, _ = fmt.Fprintln(stdout)
			}
		}
	}

	// Build finish options
	finishOpts := conductor.FinishOptions{
		SquashMerge:   opts.squash,
		DeleteBranch:  opts.delete,
		TargetBranch:  opts.targetBranch,
		PushAfter:     opts.push,
		DeleteWork:    opts.deleteWork,
		ForceMerge:    opts.merge,
		DraftPR:       opts.draftPR,
		PRTitle:       opts.prTitle,
		PRBody:        opts.prBody,
		CommitMessage: commitMessage,
		FinishAction:  opts.finishAction,
	}

	// Perform finish
	if err := cond.Finish(ctx, finishOpts); err != nil {
		return fmt.Errorf("finish: %w", err)
	}

	return nil
}

// promptFinishAction prompts the user to choose an action when the provider
// doesn't support pull requests. This keeps terminal I/O in the CLI layer.
func promptFinishAction() string {
	fmt.Println("\nThe provider for this task does not support pull requests.")
	fmt.Println("What would you like to do?")
	fmt.Println("  1. Merge changes to target branch locally")
	fmt.Println("  2. Mark task as done (no merge)")
	fmt.Println("  3. Cancel")

	for {
		var choice string
		fmt.Print("\nEnter choice (1-3): ")
		if _, err := fmt.Scanln(&choice); err != nil {
			fmt.Println("\nCancelled")

			return "cancel"
		}

		switch choice {
		case "1", "merge":
			return "merge"
		case "2", "done":
			return "done"
		case "3", "cancel", "q":
			return "cancel"
		default:
			fmt.Println("Invalid choice. Please enter 1, 2, or 3.")
		}
	}
}
