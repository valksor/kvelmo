package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	tkdisplay "github.com/valksor/go-toolkit/display"
)

var (
	simplifyNoCheckpoint bool
	simplifyAgent        string
)

var simplifyCmd = &cobra.Command{
	Use:   "simplify",
	Short: "Simplify content based on current workflow state",
	Long: `Simplify and refine content based on the current workflow state.

This command automatically determines what to simplify based on where you are
in the workflow, making it easier to get clearer, more maintainable output.

STATE-BASED BEHAVIOR:
  - Pre-plan (no specs): Simplifies task input/description
  - After planning: Simplifies specification files (specification-*.md)
  - After implementing: Simplifies code changes (even if review exists)

SAFETY:
  - Creates a git checkpoint before modifying files
  - Use --no-checkpoint to skip (not recommended)`,
	Example: `  # Auto-detect based on current state
  mehr simplify

  # Show simplification process
  mehr simplify --verbose

  # Use specific agent
  mehr simplify --agent opus

  # Skip checkpoint creation (not recommended)
  mehr simplify --no-checkpoint`,
	RunE: runSimplify,
}

func init() {
	rootCmd.AddCommand(simplifyCmd)

	simplifyCmd.Flags().BoolVar(&simplifyNoCheckpoint, "no-checkpoint", false,
		"Skip creating a checkpoint before simplifying")
	simplifyCmd.Flags().StringVar(&simplifyAgent, "agent", "",
		"Agent to use for simplification")
}

func runSimplify(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	opts := BuildConductorOptions(CommandOptions{
		Verbose: verbose,
	})

	if simplifyAgent != "" {
		opts = append(opts, conductor.WithStepAgent("simplifying", simplifyAgent))
	}

	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	if !RequireActiveTask(cond) {
		return nil
	}

	if verbose {
		SetupVerboseEventHandlers(cond)
	}

	var simplifyErr error
	if verbose {
		fmt.Println(tkdisplay.InfoMsg("Simplifying..."))
		simplifyErr = cond.Simplify(ctx, "", !simplifyNoCheckpoint)
	} else {
		spinner := display.NewSpinner("Simplifying...")
		spinner.Start()
		simplifyErr = cond.Simplify(ctx, "", !simplifyNoCheckpoint)
		if simplifyErr != nil {
			spinner.StopWithError("Simplification failed")
		} else {
			spinner.StopWithSuccess("Simplification complete")
		}
	}

	if simplifyErr != nil {
		return fmt.Errorf("simplify: %w", simplifyErr)
	}

	if verbose {
		fmt.Println()
		fmt.Println(tkdisplay.SuccessMsg("Simplification complete!"))
	}

	// Determine what was simplified based on state
	specs, _ := cond.GetWorkspace().ListSpecifications(cond.GetActiveTask().ID)
	hasSpecs := len(specs) > 0

	if !hasSpecs {
		fmt.Println("  Task input simplified")
	} else {
		fmt.Printf("  Specifications simplified: %s\n", tkdisplay.Bold(strconv.Itoa(len(specs))))
	}

	PrintNextSteps(
		"mehr status - View task status",
		"mehr cost - View token usage",
	)

	return nil
}
