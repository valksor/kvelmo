package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	tkdisplay "github.com/valksor/go-toolkit/display"
)

var resetYes bool

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset workflow state to idle without losing work",
	Long: `Reset the workflow state to idle when the agent hangs or crashes.

This preserves all your work (specifications, notes, code changes) but
allows you to retry the current step.

Use this when:
- Agent hangs and you had to kill it
- State is stuck in planning/implementing/reviewing
- You want to retry a step without abandoning

Examples:
  mehr reset              # Reset with confirmation
  mehr reset --yes        # Skip confirmation`,
	RunE: runReset,
}

func init() {
	rootCmd.AddCommand(resetCmd)
	resetCmd.Flags().BoolVarP(&resetYes, "yes", "y", false, "Skip confirmation prompt")
}

func runReset(cmd *cobra.Command, _ []string) error {
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

	// Check if already idle
	if status.State == "idle" {
		fmt.Println(tkdisplay.InfoMsg("State is already idle, nothing to reset"))

		return nil
	}

	// Build confirmation prompt
	promptLines := fmt.Sprintf("About to reset workflow state from '%s' to 'idle'", status.State)
	if status.Title != "" {
		promptLines += "\n  Task: " + status.Title
	}
	promptLines += "\n\n  This preserves all specifications, notes, and code changes."

	// Confirmation prompt (unless --yes)
	confirmed, err := confirmAction(promptLines, resetYes)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Operation cancelled")

		return nil
	}

	// Perform reset
	if err := cond.ResetState(ctx); err != nil {
		return fmt.Errorf("reset state: %w", err)
	}

	fmt.Println(tkdisplay.SuccessMsg("State reset to idle"))
	fmt.Println()
	fmt.Println(tkdisplay.Muted("You can now retry the step:"))
	fmt.Printf("  %s - Create specifications\n", tkdisplay.Cyan("mehr plan"))
	fmt.Printf("  %s - Implement specifications\n", tkdisplay.Cyan("mehr implement"))
	fmt.Printf("  %s - Review changes\n", tkdisplay.Cyan("mehr review"))

	return nil
}
