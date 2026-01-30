package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-toolkit/display"
)

var (
	simplifyNoCheckpoint bool
	simplifyAgent        string
	// Standalone mode flags.
	simplifyStandalone  bool
	simplifyBranch      string
	simplifyRange       string
	simplifyContextSize int
)

var simplifyCmd = &cobra.Command{
	Use:   "simplify [files...]",
	Short: "Simplify content based on current workflow state",
	Long: `Simplify and refine content based on the current workflow state.

This command automatically determines what to simplify based on where you are
in the workflow, making it easier to get clearer, more maintainable output.

STATE-BASED BEHAVIOR:
  - Pre-plan (no specs): Simplifies task input/description
  - After planning: Simplifies specification files (specification-*.md)
  - After implementing: Simplifies code changes (even if review exists)

STANDALONE MODE (--standalone):
  Simplify code without an active task. Useful for simplifying:
  - Uncommitted changes (default)
  - Current branch vs main/master (--branch)
  - Specific commit ranges (--range)
  - Specific files (positional args)

SAFETY:
  - Creates a git checkpoint before modifying files
  - Use --no-checkpoint to skip (not recommended)`,
	Example: `  # Auto-detect based on current state
  mehr simplify

  # Show simplification process
  mehr simplify --verbose

  # Use specific agent
  mehr simplify --agent-simplify opus

  # Skip checkpoint creation (not recommended)
  mehr simplify --no-checkpoint

  # Standalone mode (no active task needed)
  mehr simplify --standalone              # Simplify uncommitted changes
  mehr simplify --standalone --branch     # Simplify current branch vs main
  mehr simplify --standalone --branch develop  # Simplify vs develop branch
  mehr simplify --standalone --range HEAD~3..HEAD  # Simplify commit range
  mehr simplify --standalone src/foo.go src/bar.go  # Simplify specific files`,
	RunE: runSimplify,
}

func init() {
	rootCmd.AddCommand(simplifyCmd)

	simplifyCmd.Flags().BoolVar(&simplifyNoCheckpoint, "no-checkpoint", false,
		"Skip creating a checkpoint before simplifying")
	simplifyCmd.Flags().StringVar(&simplifyAgent, "agent-simplify", "",
		"Agent to use for simplification")

	// Standalone mode flags
	simplifyCmd.Flags().BoolVar(&simplifyStandalone, "standalone", false, "Simplify without an active task")
	simplifyCmd.Flags().StringVar(&simplifyBranch, "branch", "", "Compare current branch against base branch (use with --standalone)")
	simplifyCmd.Flags().StringVar(&simplifyRange, "range", "", "Compare specific commit range, e.g. HEAD~3..HEAD (use with --standalone)")
	simplifyCmd.Flags().IntVar(&simplifyContextSize, "context", 3, "Lines of context in diff (use with --standalone)")
}

func runSimplify(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Handle standalone mode
	if simplifyStandalone {
		return runStandaloneSimplify(cmd, args)
	}

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
		fmt.Println()
		fmt.Println(display.InfoMsg("Tip: Use --standalone to simplify without an active task:"))
		fmt.Println("  mehr simplify --standalone              # Simplify uncommitted changes")
		fmt.Println("  mehr simplify --standalone --branch     # Simplify current branch vs main")

		return nil
	}

	if verbose {
		SetupVerboseEventHandlers(cond)
	}

	var simplifyErr error
	if verbose {
		fmt.Println(display.InfoMsg("Simplifying..."))
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
		fmt.Println(display.SuccessMsg("Simplification complete!"))
	}

	// Determine what was simplified based on state
	specs, _ := cond.GetWorkspace().ListSpecifications(cond.GetActiveTask().ID)
	hasSpecs := len(specs) > 0

	if !hasSpecs {
		fmt.Println("  Task input simplified")
	} else {
		fmt.Printf("  Specifications simplified: %s\n", display.Bold(strconv.Itoa(len(specs))))
	}

	PrintNextSteps(
		"mehr status - View task status",
		"mehr cost - View token usage",
	)

	return nil
}

// runStandaloneSimplify runs code simplification without requiring an active task.
func runStandaloneSimplify(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Build conductor options
	opts := BuildConductorOptions(CommandOptions{
		Verbose: verbose,
	})

	if simplifyAgent != "" {
		opts = append(opts, conductor.WithStepAgent("simplifying", simplifyAgent))
	}

	// Initialize conductor
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	// Set up progress callback using helper
	if verbose {
		SetupVerboseEventHandlers(cond)
	}

	// Determine diff mode based on flags
	diffOpts := conductor.StandaloneDiffOptions{
		Context: simplifyContextSize,
	}

	switch {
	case simplifyRange != "":
		diffOpts.Mode = conductor.DiffModeRange
		diffOpts.Range = simplifyRange
	case simplifyBranch != "" || cmd.Flags().Changed("branch"):
		diffOpts.Mode = conductor.DiffModeBranch
		diffOpts.BaseBranch = simplifyBranch // May be empty, will auto-detect
	case len(args) > 0:
		diffOpts.Mode = conductor.DiffModeFiles
		diffOpts.Files = args
	default:
		diffOpts.Mode = conductor.DiffModeUncommitted
	}

	// Build simplify options
	simplifyOpts := conductor.StandaloneSimplifyOptions{
		StandaloneDiffOptions: diffOpts,
		Agent:                 simplifyAgent,
		CreateCheckpoint:      !simplifyNoCheckpoint,
	}

	// Show what we're simplifying
	printSimplifyModeInfo(diffOpts)

	// Run standalone simplify with spinner in non-verbose mode
	var result *conductor.StandaloneSimplifyResult
	var simplifyErr error
	if verbose {
		fmt.Println(display.InfoMsg("Simplifying..."))
		result, simplifyErr = cond.SimplifyStandalone(ctx, simplifyOpts)
	} else {
		spinner := display.NewSpinner("Simplifying code...")
		spinner.Start()
		result, simplifyErr = cond.SimplifyStandalone(ctx, simplifyOpts)
		if simplifyErr != nil {
			spinner.StopWithError("Simplification failed")
		} else {
			spinner.StopWithSuccess("Simplification complete")
		}
	}

	if simplifyErr != nil {
		return fmt.Errorf("simplify: %w", simplifyErr)
	}

	// Print results
	fmt.Println()
	printStandaloneSimplifyResult(result)

	return nil
}

// printSimplifyModeInfo prints information about what is being simplified.
func printSimplifyModeInfo(opts conductor.StandaloneDiffOptions) {
	switch opts.Mode {
	case conductor.DiffModeUncommitted:
		fmt.Println(display.InfoMsg("Simplifying uncommitted changes (staged + unstaged)..."))
	case conductor.DiffModeBranch:
		if opts.BaseBranch != "" {
			fmt.Printf("%s Simplifying current branch vs %s...\n", display.InfoMsg(""), opts.BaseBranch)
		} else {
			fmt.Println(display.InfoMsg("Simplifying current branch vs default branch..."))
		}
	case conductor.DiffModeRange:
		fmt.Printf("%s Simplifying commit range: %s...\n", display.InfoMsg(""), opts.Range)
	case conductor.DiffModeFiles:
		fmt.Printf("%s Simplifying files: %s...\n", display.InfoMsg(""), strings.Join(opts.Files, ", "))
	}
}

// printStandaloneSimplifyResult prints the simplification results to stdout.
func printStandaloneSimplifyResult(result *conductor.StandaloneSimplifyResult) {
	fmt.Println(display.SuccessMsg("✓ Simplification complete"))

	// Print summary
	if result.Summary != "" {
		fmt.Println()
		fmt.Println(display.Bold("Summary:"))
		fmt.Println(result.Summary)
	}

	// Print changes
	if len(result.Changes) > 0 {
		fmt.Println()
		fmt.Println(display.Bold("Suggested Changes:"))
		for _, change := range result.Changes {
			fmt.Printf("  [%s] %s\n", strings.ToUpper(string(change.Operation)), change.Path)
		}
	}

	// Print usage info
	if result.Usage != nil {
		fmt.Println()
		fmt.Printf("Tokens: %d input, %d output", result.Usage.InputTokens, result.Usage.OutputTokens)
		if result.Usage.CostUSD > 0 {
			fmt.Printf(" ($%.4f)", result.Usage.CostUSD)
		}
		fmt.Println()
	}
}
