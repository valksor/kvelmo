package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
)

var redoYes bool

var redoCmd = &cobra.Command{
	Use:   "redo",
	Short: "Restore the next checkpoint",
	Long: `Restore the current task to the next checkpoint.

HOW REDO WORKS:
  This redoes changes that were previously undone.

  • Moves HEAD forward to the next checkpoint after an undo
  • Only available if you previously used 'mehr undo'
  • Restores code changes that were undone

UNDO/REDO HISTORY:
  Undo and redo work like a linear history stack:
  1. Each 'mehr undo' moves backward in checkpoint history
  2. Each 'mehr redo' moves forward (restores undone changes)
  3. Creating new checkpoints clears the redo history

RELATED COMMANDS:
  mehr undo     - Revert to previous checkpoint
  mehr status   - Show current task state and checkpoint info

Examples:
  mehr redo                    # Redo changes (with confirmation)
  mehr redo --yes              # Redo without confirmation`,
	RunE: runRedo,
}

func init() {
	rootCmd.AddCommand(redoCmd)
	redoCmd.Flags().BoolVarP(&redoYes, "yes", "y", false, "Skip confirmation prompt")
}

func runRedo(cmd *cobra.Command, args []string) error {
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

	// Get status for confirmation
	status, err := cond.Status(ctx)
	if err != nil {
		return fmt.Errorf("get status: %w", err)
	}

	// Build confirmation prompt
	promptLines := "About to redo changes and restore to next checkpoint"
	if status.Title != "" {
		promptLines += "\n  Task: " + status.Title
	}
	promptLines += "\n  State: " + status.State

	// Confirmation prompt (unless --yes)
	confirmed, err := confirmAction(promptLines, redoYes)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Cancelled")

		return nil
	}

	// Perform redo
	if err := cond.Redo(ctx); err != nil {
		return fmt.Errorf("redo: %w", err)
	}

	fmt.Println("Restored to next checkpoint")

	return nil
}
