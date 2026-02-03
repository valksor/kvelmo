package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	tkdisplay "github.com/valksor/go-toolkit/display"
)

var undoYes bool

var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Revert to the previous checkpoint",
	Long: `Revert the current task to its previous checkpoint.

CHECKPOINT SYSTEM:
  Mehrhof automatically creates git checkpoints during workflow operations:
  - After each plan/implement/review step completes
  - Before standalone simplify/review operations modify files
  - When explicitly requested via commands

  Each checkpoint is a git commit tagged with the task ID. Undo/redo
  navigates between these checkpoints without losing work.

HOW UNDO WORKS:
  • Moves HEAD to the previous checkpoint for the current task
  • Preserves redo history - you can redo to restore changes
  • Only affects the current task's checkpoints
  • Safe operation - no data is permanently lost

HISTORY DEPTH:
  All checkpoints created during a task are preserved. You can undo
  multiple times to go back through the history. Use 'mehr status'
  to see the current checkpoint number.

RELATED COMMANDS:
  mehr redo     - Restore undone changes (move forward in history)
  mehr reset    - Reset workflow state to idle (keeps code changes)
  mehr status   - Show current task state and checkpoint info

Examples:
  mehr undo                    # Undo last changes (with confirmation)
  mehr undo --yes              # Undo without confirmation`,
	RunE: runUndo,
}

func init() {
	rootCmd.AddCommand(undoCmd)
	undoCmd.Flags().BoolVarP(&undoYes, "yes", "y", false, "Skip confirmation prompt")
}

func runUndo(cmd *cobra.Command, args []string) error {
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
	promptLines := "About to undo last changes and revert to previous checkpoint"
	if status.Title != "" {
		promptLines += "\n  Task: " + status.Title
	}
	promptLines += "\n  State: " + status.State

	// Confirmation prompt (unless --yes)
	confirmed, err := confirmAction(promptLines, undoYes)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Operation cancelled")

		return nil
	}

	// Perform undo
	if err := cond.Undo(ctx); err != nil {
		return fmt.Errorf("undo: %w", err)
	}

	fmt.Println(tkdisplay.SuccessMsg("Reverted to previous checkpoint"))

	return nil
}
