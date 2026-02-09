package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	tkdisplay "github.com/valksor/go-toolkit/display"
)

var (
	abandonYes        bool
	abandonKeepBranch bool
	abandonKeepWork   bool
)

var abandonCmd = &cobra.Command{
	Use:   "abandon",
	Short: "Abandon the current task without merging",
	Long: `Abandon the current task, its branch, and work directory without merging changes.

This is useful when you want to abandon a task completely, such as when:
- The approach didn't work out
- The task is no longer needed
- You want to start fresh

By default, this command:
- Deletes the task branch (if one was created)
- Removes the work directory (.task/work/<task-id>)
- Clears the active task reference

Examples:
  mehr abandon                 # Abandon with confirmation
  mehr abandon --yes           # Abandon without confirmation
  mehr abandon -y              # Same as --yes
  mehr abandon --keep-branch   # Abandon task but keep the git branch
  mehr abandon --keep-work     # Abandon branch but keep the work directory`,
	RunE: runAbandon,
}

func init() {
	rootCmd.AddCommand(abandonCmd)

	abandonCmd.Flags().BoolVarP(&abandonYes, "yes", "y", false, "Skip confirmation prompt")
	abandonCmd.Flags().BoolVar(&abandonKeepBranch, "keep-branch", false, "Keep the git branch")
	abandonCmd.Flags().BoolVar(&abandonKeepWork, "keep-work", false, "Keep the work directory")
}

// abandonOptions captures flag state for the abandon command.
// This enables testing without Cobra runtime dependencies.
type abandonOptions struct {
	skipConfirm     bool
	keepBranch      bool
	keepWorkChanged bool // true if --keep-work was explicitly set
	keepWork        bool // value of --keep-work
}

func runAbandon(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Initialize conductor with standard providers and agents
	cond, err := initializeConductor(ctx, conductor.WithVerbose(verbose))
	if err != nil {
		return err
	}

	opts := abandonOptions{
		skipConfirm:     abandonYes,
		keepBranch:      abandonKeepBranch,
		keepWorkChanged: cmd.Flags().Changed("keep-work"),
		keepWork:        abandonKeepWork,
	}

	return runAbandonLogic(ctx, cond, opts)
}

// runAbandonLogic contains the testable business logic for the abandon command.
// It accepts a ConductorAPI to enable mock-based testing.
func runAbandonLogic(ctx context.Context, cond ConductorAPI, opts abandonOptions) error {
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
	promptLines := "About to abandon task: " + status.TaskID
	if status.Title != "" {
		promptLines += "\n  Title: " + status.Title
	}
	if status.Branch != "" {
		promptLines += "\n  Branch: " + status.Branch
	}
	promptLines += "\n  State: " + status.State
	promptLines += fmt.Sprintf("\n  Specifications: %d", status.Specifications)

	if !opts.keepBranch && status.Branch != "" {
		promptLines += "\n\nWARNING: This will delete the git branch and all uncommitted changes!"
	}

	// Confirmation prompt (unless skipConfirm)
	confirmed, err := confirmAction(promptLines, opts.skipConfirm)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Operation cancelled")

		return nil
	}

	// Build delete options
	// Use tri-state for DeleteWork: nil=defer to config, true=delete, false=keep
	var deleteWork *bool
	if opts.keepWorkChanged {
		deleteWork = conductor.BoolPtr(!opts.keepWork) // --keep-work means don't delete
	}

	deleteOpts := conductor.DeleteOptions{
		Force:      opts.skipConfirm,
		KeepBranch: opts.keepBranch,
		DeleteWork: deleteWork,
	}

	// Perform delete
	if err := cond.Delete(ctx, deleteOpts); err != nil {
		return fmt.Errorf("abandon: %w", err)
	}

	fmt.Println(tkdisplay.SuccessMsg("Task abandoned successfully"))

	return nil
}
