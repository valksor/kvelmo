package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/events"
	tkdisplay "github.com/valksor/go-toolkit/display"
	"github.com/valksor/go-toolkit/eventbus"
)

// implementOptions holds options for the implement command.
type implementOptions struct {
	force bool
}

// implementReviewOptions holds options for the implement review command.
type implementReviewOptions struct {
	reviewNumber int
	force        bool
}

var (
	implementDryRun            bool
	implementAgentImplementing string
	implementOptimize          bool
	implementOnly              string
	implementParallel          string
	implementForce             bool // Force reset state before implementing
	implementLibrary           bool // Auto-include library documentation

	// Hierarchical context flags (override workspace config).
	implementWithParent      bool // Include parent task context
	implementWithoutParent   bool // Exclude parent task context
	implementWithSiblings    bool // Include sibling subtask context
	implementWithoutSiblings bool // Exclude sibling subtask context
	implementMaxSiblings     int  // Maximum sibling tasks to include
)

var implementCmd = &cobra.Command{
	Use:   "implement [review <number>]",
	Short: "Implement the specifications for the active task",
	Long: `Run the implementation phase to generate code based on specifications.

The agent will read all specification files in the work directory along with any
notes, then implement them by creating or modifying files.

Requires at least one specification file to exist (run 'mehr plan' first).

Examples:
  mehr implement                # Implement the specifications
  mehr implement --dry-run      # Preview without making changes
  mehr implement --verbose      # Show agent output
  mehr implement review 1       # Implement fixes from review 1`,
	RunE: runImplement,
}

var implementReviewCmd = &cobra.Command{
	Use:   "review <number>",
	Short: "Implement fixes from a specific code review",
	Long: `Implement code fixes based on review feedback.

Unlike regular implementation which follows specifications, this command
focuses specifically on addressing issues identified in a code review.

The agent will read the review content and implement fixes for each issue
mentioned. A checkpoint is created before changes are applied.

Examples:
  mehr implement review 1       # Implement fixes from review 1
  mehr implement review 2 --dry-run  # Preview fixes without applying`,
	Args: cobra.ExactArgs(1),
	RunE: runImplementReview,
}

func init() {
	rootCmd.AddCommand(implementCmd)
	implementCmd.AddCommand(implementReviewCmd)

	implementCmd.Flags().BoolVarP(&implementDryRun, "dry-run", "n", false, "Don't apply file changes (preview only)")
	implementCmd.Flags().StringVar(&implementAgentImplementing, "agent-implement", "", "Agent for implementation step")
	implementCmd.Flags().BoolVar(&implementOptimize, "optimize", false, "Optimize prompt before sending to agent")
	implementCmd.Flags().StringVar(&implementOnly, "only", "", "Only implement this component (e.g., backend, frontend, tests)")
	implementCmd.Flags().StringVar(&implementParallel, "parallel", "", "Run N agents in parallel, or comma-separated agent list")
	implementCmd.Flags().BoolVar(&implementForce, "force", false, "Reset workflow state and retry (use after hung agent)")
	implementCmd.Flags().BoolVar(&implementLibrary, "library", false, "Auto-include relevant library documentation based on working directory")

	// Hierarchical context flags (override workspace config)
	implementCmd.Flags().BoolVar(&implementWithParent, "with-parent", false, "Include parent task context (overrides config)")
	implementCmd.Flags().BoolVar(&implementWithoutParent, "without-parent", false, "Exclude parent task context (overrides config)")
	implementCmd.Flags().BoolVar(&implementWithSiblings, "with-siblings", false, "Include sibling subtask context (overrides config)")
	implementCmd.Flags().BoolVar(&implementWithoutSiblings, "without-siblings", false, "Exclude sibling subtask context (overrides config)")
	implementCmd.Flags().IntVar(&implementMaxSiblings, "max-siblings", 0, "Maximum sibling tasks to include (overrides config, 0 = use config)")
}

func runImplement(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Build conductor options
	condOpts := []conductor.Option{
		conductor.WithVerbose(verbose),
		conductor.WithDryRun(implementDryRun),
	}

	// Per-step agent override
	if implementAgentImplementing != "" {
		condOpts = append(condOpts, conductor.WithStepAgent("implementing", implementAgentImplementing))
	}

	// Prompt optimization
	if implementOptimize {
		condOpts = append(condOpts, conductor.WithOptimizePrompts(true))
	}

	// Component filtering
	if implementOnly != "" {
		condOpts = append(condOpts, conductor.WithOnlyComponent(implementOnly))
	}

	// Parallel execution
	if implementParallel != "" {
		condOpts = append(condOpts, conductor.WithParallel(implementParallel))
	}

	// Library auto-include
	if implementLibrary {
		condOpts = append(condOpts, conductor.WithLibraryAutoInclude(true))
	}

	// Use deduplicating stdout in verbose mode to suppress duplicate lines
	if verbose {
		condOpts = append(condOpts, conductor.WithStdout(getDeduplicatingStdout()))
	}

	// Initialize conductor with standard providers and agents
	cond, err := initializeConductor(ctx, condOpts...)
	if err != nil {
		return err
	}

	// Set up event handlers for verbose mode
	if verbose {
		w := cond.GetStdout()
		cond.GetEventBus().SubscribeAll(func(e eventbus.Event) {
			switch e.Type {
			case events.TypeProgress:
				if msg, ok := e.Data["message"].(string); ok {
					_, err := fmt.Fprintf(w, "  %s\n", msg)
					if err != nil {
						slog.Debug("write progress", "error", err)
					}
				}
			case events.TypeFileChanged:
				if path, ok := e.Data["path"].(string); ok {
					op, _ := e.Data["operation"].(string)
					_, err := fmt.Fprintf(w, "  [%s] %s\n", op, path)
					if err != nil {
						slog.Debug("write file change", "error", err)
					}
				}
			case events.TypeCheckpoint:
				if num, ok := e.Data["checkpoint"].(int); ok {
					_, err := fmt.Fprintf(w, "  Checkpoint #%d created\n", num)
					if err != nil {
						slog.Debug("write checkpoint", "error", err)
					}
				}
			case events.TypeStateChanged, events.TypeError, events.TypeAgentMessage, events.TypeBlueprintReady, events.TypeBranchCreated, events.TypePlanCompleted, events.TypeImplementDone, events.TypePRCreated, events.TypeBrowserAction, events.TypeBrowserTabOpened, events.TypeBrowserScreenshot:
				// Ignore other event types
			}
		})
	}

	// Run implementation logic with spinner in non-verbose mode
	implOpts := implementOptions{force: implementForce}
	var implErr error
	spinnerMsg := "Implementing code..."
	if implementDryRun {
		spinnerMsg = "Implementing code (dry-run)..."
	}

	if verbose {
		if implementDryRun {
			fmt.Println(tkdisplay.InfoMsg("Implementing (dry-run)..."))
		} else {
			fmt.Println(tkdisplay.InfoMsg("Implementing..."))
		}
		implErr = runImplementLogic(ctx, cond, implOpts, cond.GetStdout())
	} else {
		spinner := tkdisplay.NewSpinner(spinnerMsg)
		spinner.Start()
		implErr = runImplementLogic(ctx, cond, implOpts, nil)
		if implErr != nil && !errors.Is(implErr, conductor.ErrBudgetPaused) && !errors.Is(implErr, conductor.ErrBudgetStopped) {
			spinner.StopWithError("Implementation failed")
		} else if errors.Is(implErr, conductor.ErrBudgetPaused) {
			spinner.StopWithWarning("Implementation paused due to budget limit")
		} else if errors.Is(implErr, conductor.ErrBudgetStopped) {
			spinner.StopWithError("Implementation stopped due to budget limit")
		} else {
			if implementDryRun {
				spinner.StopWithSuccess("Implementation preview complete")
			} else {
				spinner.StopWithSuccess("Implementation complete")
			}
		}
	}

	// Handle budget errors with appropriate UI
	if errors.Is(implErr, conductor.ErrBudgetPaused) {
		fmt.Println(tkdisplay.WarningMsg("Task paused due to budget limit."))
		fmt.Println(tkdisplay.Muted("Review budgets and resume when ready:"))
		fmt.Printf("  %s\n", tkdisplay.Cyan("mehr budget status"))
		fmt.Printf("  %s\n", tkdisplay.Cyan("mehr budget resume --confirm"))

		return nil
	}
	if errors.Is(implErr, conductor.ErrBudgetStopped) {
		fmt.Println(tkdisplay.ErrorMsg("Task stopped due to budget limit."))
		fmt.Println(tkdisplay.Muted("Update budgets or start a new task:"))
		fmt.Printf("  %s\n", tkdisplay.Cyan("mehr budget set"))
		fmt.Printf("  %s\n", tkdisplay.Cyan("mehr start <ref>"))

		return nil
	}
	if implErr != nil {
		return implErr
	}

	// Get status
	status, err := cond.Status(ctx)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Println()
		if implementDryRun {
			fmt.Println(tkdisplay.SuccessMsg("Implementation preview finished"))
		} else {
			fmt.Println(tkdisplay.SuccessMsg("Implementation complete!"))
		}
	}
	fmt.Printf("  Checkpoints: %s\n", tkdisplay.Bold(strconv.Itoa(status.Checkpoints)))
	if implementDryRun {
		fmt.Println()
		fmt.Println(tkdisplay.Muted("  (Dry-run mode - no files were modified)"))
	}
	fmt.Println()
	fmt.Println(tkdisplay.Muted("Next steps:"))
	fmt.Printf("  %s - View task status\n", tkdisplay.Cyan("mehr status"))
	fmt.Printf("  %s - Run code review\n", tkdisplay.Cyan("mehr review"))
	fmt.Printf("  %s - Revert last changes\n", tkdisplay.Cyan("mehr undo"))
	fmt.Printf("  %s - Complete the task\n", tkdisplay.Cyan("mehr finish"))

	return nil
}

func runImplementReview(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Parse review number (validation done by runImplementReviewLogic)
	reviewNumber, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid review number %q: must be an integer", args[0])
	}

	// Build conductor options
	condOpts := []conductor.Option{
		conductor.WithVerbose(verbose),
		conductor.WithDryRun(implementDryRun),
	}

	// Per-step agent override
	if implementAgentImplementing != "" {
		condOpts = append(condOpts, conductor.WithStepAgent("implementing", implementAgentImplementing))
	}

	// Prompt optimization
	if implementOptimize {
		condOpts = append(condOpts, conductor.WithOptimizePrompts(true))
	}

	// Use deduplicating stdout in verbose mode
	if verbose {
		condOpts = append(condOpts, conductor.WithStdout(getDeduplicatingStdout()))
	}

	// Initialize conductor
	cond, err := initializeConductor(ctx, condOpts...)
	if err != nil {
		return err
	}

	// Run review implementation logic with spinner in non-verbose mode
	implOpts := implementReviewOptions{reviewNumber: reviewNumber, force: implementForce}
	var implErr error
	spinnerMsg := fmt.Sprintf("Implementing fixes from review %d...", reviewNumber)
	if implementDryRun {
		spinnerMsg = fmt.Sprintf("Implementing fixes from review %d (dry-run)...", reviewNumber)
	}

	if verbose {
		if implementDryRun {
			fmt.Println(tkdisplay.InfoMsg("%s", fmt.Sprintf("Implementing fixes from review %d (dry-run)...", reviewNumber)))
		} else {
			fmt.Println(tkdisplay.InfoMsg("%s", fmt.Sprintf("Implementing fixes from review %d...", reviewNumber)))
		}
		implErr = runImplementReviewLogic(ctx, cond, implOpts, cond.GetStdout())
	} else {
		spinner := tkdisplay.NewSpinner(spinnerMsg)
		spinner.Start()
		implErr = runImplementReviewLogic(ctx, cond, implOpts, nil)
		if implErr != nil {
			spinner.StopWithError("Review fix implementation failed")
		} else {
			if implementDryRun {
				spinner.StopWithSuccess("Review fix preview complete")
			} else {
				spinner.StopWithSuccess("Review fixes applied")
			}
		}
	}

	if implErr != nil {
		return implErr
	}

	// Get status
	status, err := cond.Status(ctx)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Println()
		if implementDryRun {
			fmt.Println(tkdisplay.SuccessMsg("%s", fmt.Sprintf("Review %d fix preview finished", reviewNumber)))
		} else {
			fmt.Println(tkdisplay.SuccessMsg("%s", fmt.Sprintf("Review %d fixes applied!", reviewNumber)))
		}
	}
	fmt.Printf("  Checkpoints: %s\n", tkdisplay.Bold(strconv.Itoa(status.Checkpoints)))
	if implementDryRun {
		fmt.Println()
		fmt.Println(tkdisplay.Muted("  (Dry-run mode - no files were modified)"))
	}
	fmt.Println()
	fmt.Println(tkdisplay.Muted("Next steps:"))
	fmt.Printf("  %s - Run another review to verify fixes\n", tkdisplay.Cyan("mehr review"))
	fmt.Printf("  %s - View task status\n", tkdisplay.Cyan("mehr status"))
	fmt.Printf("  %s - Revert changes if needed\n", tkdisplay.Cyan("mehr undo"))
	fmt.Printf("  %s - Complete the task\n", tkdisplay.Cyan("mehr finish"))

	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Testable logic functions
// ──────────────────────────────────────────────────────────────────────────────

// runImplementLogic contains the core implementation logic, extracted for testing.
// Returns conductor.ErrBudgetPaused or conductor.ErrBudgetStopped for budget limits.
// Callers should check for these sentinel errors and handle them as non-failure cases.
func runImplementLogic(ctx context.Context, api ConductorAPI, opts implementOptions, stdout io.Writer) error {
	// Check for an active task
	if api.GetActiveTask() == nil {
		if stdout != nil {
			_, _ = fmt.Fprint(stdout, display.NoActiveTaskError())
		}

		return errors.New("no active task")
	}

	// Handle --force flag to reset stuck state
	if opts.force {
		if err := api.ResetState(ctx); err != nil {
			return fmt.Errorf("reset state: %w", err)
		}
		if stdout != nil {
			_, _ = fmt.Fprintln(stdout, tkdisplay.InfoMsg("State reset to idle"))
		}
	}

	// Enter implementation phase
	if err := api.Implement(ctx); err != nil {
		return fmt.Errorf("implement: %w", err)
	}

	// Run implementation
	implErr := api.RunImplementation(ctx)

	// Return budget errors as sentinel values for caller to handle UI
	if errors.Is(implErr, conductor.ErrBudgetPaused) {
		return conductor.ErrBudgetPaused
	}
	if errors.Is(implErr, conductor.ErrBudgetStopped) {
		return conductor.ErrBudgetStopped
	}
	if implErr != nil {
		return fmt.Errorf("run implementation: %w", implErr)
	}

	return nil
}

// runImplementReviewLogic contains the core review implementation logic, extracted for testing.
func runImplementReviewLogic(ctx context.Context, api ConductorAPI, opts implementReviewOptions, stdout io.Writer) error {
	// Validate review number
	if opts.reviewNumber <= 0 {
		return fmt.Errorf("review number must be positive, got %d", opts.reviewNumber)
	}

	// Check for an active task
	if api.GetActiveTask() == nil {
		if stdout != nil {
			_, _ = fmt.Fprint(stdout, display.NoActiveTaskError())
		}

		return errors.New("no active task")
	}

	// Handle --force flag to reset stuck state
	if opts.force {
		if err := api.ResetState(ctx); err != nil {
			return fmt.Errorf("reset state: %w", err)
		}
		if stdout != nil {
			_, _ = fmt.Fprintln(stdout, tkdisplay.InfoMsg("State reset to idle"))
		}
	}

	// Enter implementation state for review fixes
	if err := api.ImplementReview(ctx, opts.reviewNumber); err != nil {
		return fmt.Errorf("implement review: %w", err)
	}

	// Run review implementation
	implErr := api.RunReviewImplementation(ctx, opts.reviewNumber)
	if implErr != nil {
		return fmt.Errorf("run review implementation: %w", implErr)
	}

	return nil
}
