package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	tkdisplay "github.com/valksor/go-toolkit/display"
)

// maxReviewFileNumber is the maximum number to try when generating review filenames.
const maxReviewFileNumber = 100

var (
	reviewTool           string
	reviewOutput         string
	reviewAgentReviewing string
	reviewOptimize       bool
	// Standalone mode flags.
	reviewStandalone  bool
	reviewBranch      string
	reviewRange       string
	reviewContextSize int
	reviewFix         bool
	reviewCheckpoint  bool
)

var reviewCmd = &cobra.Command{
	Use:   "review [files...]",
	Short: "Run code review on current changes",
	Long: `Run an automated code review on the current task's changes or on standalone code.

By default, this uses the AI agent to review code changes for bugs, improvements,
and best practices. Alternatively, you can specify an external tool like CodeRabbit.

STANDALONE MODE (--standalone):
  Review code without an active task. Useful for reviewing:
  - Uncommitted changes (default)
  - Current branch vs main/master (--branch)
  - Specific commit ranges (--range)
  - Specific files (positional args)

FIX MODE (--fix):
  Use with --standalone to review AND fix issues. The agent will:
  - Review the code for bugs, security issues, and correctness problems
  - Automatically fix the issues it finds
  - Create a checkpoint before changes (use --checkpoint=false to skip)

Review Status:
  APPROVED - Review passed with no issues
  NEEDS_CHANGES - Review found issues that need attention
  ERROR - Review tool failed to run

Examples:
  mehr review                           # Review active task changes
  mehr review --tool coderabbit         # Use external tool instead
  mehr review --output review.txt       # Save to specific file

  # Standalone mode (no active task needed)
  mehr review --standalone              # Review uncommitted changes
  mehr review --standalone --branch     # Review current branch vs main
  mehr review --standalone --branch develop  # Review vs develop branch
  mehr review --standalone --range HEAD~3..HEAD  # Review commit range
  mehr review --standalone src/foo.go src/bar.go  # Review specific files

  # Fix mode (review AND apply fixes)
  mehr review --standalone --fix        # Review and fix uncommitted changes
  mehr review --standalone --fix --branch  # Review and fix branch changes
  mehr review --standalone --fix --checkpoint=false  # Fix without checkpoint`,
	RunE: runReview,
}

func init() {
	rootCmd.AddCommand(reviewCmd)

	reviewCmd.Flags().StringVar(&reviewTool, "tool", "", "Review tool to use (empty skips external tool review)")
	reviewCmd.Flags().StringVarP(&reviewOutput, "output", "o", "", "Output file name (default: review-N.txt)")
	reviewCmd.Flags().StringVar(&reviewAgentReviewing, "agent-review", "", "Agent for review step (when using agent-based review)")
	reviewCmd.Flags().BoolVar(&reviewOptimize, "optimize", false, "Optimize prompt before sending to agent")

	// Standalone mode flags
	reviewCmd.Flags().BoolVar(&reviewStandalone, "standalone", false, "Review without an active task")
	reviewCmd.Flags().StringVar(&reviewBranch, "branch", "", "Compare current branch against base branch (use with --standalone)")
	reviewCmd.Flags().StringVar(&reviewRange, "range", "", "Compare specific commit range, e.g. HEAD~3..HEAD (use with --standalone)")
	reviewCmd.Flags().IntVar(&reviewContextSize, "context", 3, "Lines of context in diff (use with --standalone)")
	reviewCmd.Flags().BoolVar(&reviewFix, "fix", false, "Apply suggested fixes (use with --standalone)")
	reviewCmd.Flags().BoolVar(&reviewCheckpoint, "checkpoint", true, "Create checkpoint before applying fixes (use with --fix)")
}

func runReview(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Handle standalone mode
	if reviewStandalone {
		return runStandaloneReview(cmd, args)
	}

	// Build conductor options
	opts := []conductor.Option{
		conductor.WithVerbose(verbose),
	}
	if reviewOptimize {
		opts = append(opts, conductor.WithOptimizePrompts(true))
	}

	// Initialize conductor with standard providers and agents
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	// Check for active task
	activeTask := cond.GetActiveTask()
	if activeTask == nil {
		fmt.Print(display.NoActiveTaskError())
		fmt.Println()
		fmt.Println(tkdisplay.InfoMsg("Tip: Use --standalone to review without an active task:"))
		fmt.Println("  mehr review --standalone              # Review uncommitted changes")
		fmt.Println("  mehr review --standalone --branch     # Review current branch vs main")

		return errors.New("no active task")
	}

	// Get workspace
	ws := cond.GetWorkspace()
	if ws == nil {
		return errors.New("workspace not available")
	}

	// Check if review tool is specified
	if reviewTool == "" {
		// Use agent-based code review when no external tool is specified
		fmt.Println("Running agent-based code review...")

		return cond.RunReview(ctx)
	}

	// Check if review tool is available
	toolPath, err := exec.LookPath(reviewTool)
	if err != nil {
		return fmt.Errorf("review tool '%s' not found in PATH\n\nInstall with: npm install -g %s\n\nOr use --tool to specify a different tool:\n  mehr review --tool <tool-name>", reviewTool, reviewTool)
	}

	fmt.Printf("Running %s review...\n", reviewTool)

	// Run the review tool
	reviewCmd := exec.CommandContext(ctx, toolPath, "review")
	reviewCmd.Dir = ws.Root()

	output, err := reviewCmd.CombinedOutput()
	outputStr := string(output)

	// Determine review status
	var status string
	if err != nil {
		status = "ERROR"
		fmt.Printf("Review failed: %v\n", err)
	} else if containsIssues(outputStr) {
		status = "ISSUES"
		fmt.Println("Review completed with issues")
	} else {
		status = "COMPLETE"
		fmt.Println("Review completed successfully")
	}

	// Get next review number
	reviewNum, err := ws.NextReviewNumber(activeTask.ID)
	if err != nil {
		return fmt.Errorf("get next review number: %w", err)
	}

	// Build review content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# Code Review - %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("Tool: %s\n", reviewTool))
	content.WriteString(fmt.Sprintf("Status: %s\n", status))
	content.WriteString(fmt.Sprintf("Task: %s\n", activeTask.ID))
	if activeTask.Branch != "" {
		content.WriteString(fmt.Sprintf("Branch: %s\n", activeTask.Branch))
	}
	content.WriteString("\n---\n\n")
	content.WriteString(outputStr)

	// Save review output using workspace storage
	if err := ws.SaveReview(activeTask.ID, reviewNum, content.String()); err != nil {
		return fmt.Errorf("save review output: %w", err)
	}

	// Get the path for display (load config for pattern)
	cfg, _ := ws.LoadConfig()
	outputPath := ws.ReviewPath(activeTask.ID, reviewNum, cfg)
	fmt.Printf("\nReview saved to: %s\n", outputPath)
	fmt.Printf("Status: %s\n", status)

	if status == "ISSUES" {
		fmt.Println("\nPlease review the issues and address them before finishing the task.")
	}

	return nil
}

// runStandaloneReview runs a code review without requiring an active task.
func runStandaloneReview(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Build conductor options
	opts := []conductor.Option{
		conductor.WithVerbose(verbose),
	}
	if reviewOptimize {
		opts = append(opts, conductor.WithOptimizePrompts(true))
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
		Context: reviewContextSize,
	}

	switch {
	case reviewRange != "":
		diffOpts.Mode = conductor.DiffModeRange
		diffOpts.Range = reviewRange
	case reviewBranch != "" || cmd.Flags().Changed("branch"):
		diffOpts.Mode = conductor.DiffModeBranch
		diffOpts.BaseBranch = reviewBranch // May be empty, will auto-detect
	case len(args) > 0:
		diffOpts.Mode = conductor.DiffModeFiles
		diffOpts.Files = args
	default:
		diffOpts.Mode = conductor.DiffModeUncommitted
	}

	// Build review options
	reviewOpts := conductor.StandaloneReviewOptions{
		StandaloneDiffOptions: diffOpts,
		Agent:                 reviewAgentReviewing,
		ApplyFixes:            reviewFix,
		CreateCheckpoint:      reviewCheckpoint,
	}

	// Show what we're reviewing
	printReviewModeInfo(diffOpts, reviewFix)

	// Run standalone review with spinner in non-verbose mode
	var result *conductor.StandaloneReviewResult
	var reviewErr error
	if verbose {
		fmt.Println(tkdisplay.InfoMsg("Reviewing..."))
		result, reviewErr = cond.ReviewStandalone(ctx, reviewOpts)
	} else {
		spinner := display.NewSpinner("Reviewing code...")
		spinner.Start()
		result, reviewErr = cond.ReviewStandalone(ctx, reviewOpts)
		if reviewErr != nil {
			spinner.StopWithError("Review failed")
		} else {
			spinner.StopWithSuccess("Review complete")
		}
	}

	if reviewErr != nil {
		return fmt.Errorf("review: %w", reviewErr)
	}

	// Print results
	fmt.Println()
	printStandaloneReviewResult(result)

	// Save to file if output is specified
	if reviewOutput != "" {
		if err := saveStandaloneReviewToFile(result, reviewOutput); err != nil {
			return fmt.Errorf("save review: %w", err)
		}
		fmt.Printf("\nReview saved to: %s\n", reviewOutput)
	}

	return nil
}

// printReviewModeInfo prints information about what is being reviewed.
func printReviewModeInfo(opts conductor.StandaloneDiffOptions, applyFixes bool) {
	action := "Reviewing"
	if applyFixes {
		action = "Reviewing and fixing"
	}

	switch opts.Mode {
	case conductor.DiffModeUncommitted:
		fmt.Printf("%s %s uncommitted changes (staged + unstaged)...\n", tkdisplay.InfoMsg(""), action)
	case conductor.DiffModeBranch:
		if opts.BaseBranch != "" {
			fmt.Printf("%s %s current branch vs %s...\n", tkdisplay.InfoMsg(""), action, opts.BaseBranch)
		} else {
			fmt.Printf("%s %s current branch vs default branch...\n", tkdisplay.InfoMsg(""), action)
		}
	case conductor.DiffModeRange:
		fmt.Printf("%s %s commit range: %s...\n", tkdisplay.InfoMsg(""), action, opts.Range)
	case conductor.DiffModeFiles:
		fmt.Printf("%s %s files: %s...\n", tkdisplay.InfoMsg(""), action, strings.Join(opts.Files, ", "))
	}
}

// printStandaloneReviewResult prints the review results to stdout.
func printStandaloneReviewResult(result *conductor.StandaloneReviewResult) {
	// Print verdict with appropriate styling
	switch result.Verdict {
	case "APPROVED":
		fmt.Println(tkdisplay.SuccessMsg("✓ Review: APPROVED"))
	case "NEEDS_CHANGES":
		fmt.Println(tkdisplay.WarningMsg("⚠ Review: NEEDS_CHANGES"))
	default:
		fmt.Println(tkdisplay.InfoMsg("%s", "● Review: "+result.Verdict))
	}

	// Print summary
	if result.Summary != "" {
		fmt.Println()
		fmt.Println(tkdisplay.Bold("Summary:"))
		fmt.Println(result.Summary)
	}

	// Print issues
	if len(result.Issues) > 0 {
		fmt.Println()
		fmt.Println(tkdisplay.Bold("Issues Found:"))
		for _, issue := range result.Issues {
			severity := strings.ToUpper(issue.Severity)
			location := issue.File
			if issue.Line > 0 {
				location = fmt.Sprintf("%s:%d", issue.File, issue.Line)
			}
			fmt.Printf("  [%s] %s: %s\n", severity, location, issue.Message)
		}
	}

	// Print applied changes
	if len(result.Changes) > 0 {
		fmt.Println()
		fmt.Println(tkdisplay.Bold("Changes Applied:"))
		for _, change := range result.Changes {
			op := string(change.Operation)
			if op == "" {
				op = "update"
			}
			fmt.Printf("  [%s] %s\n", strings.ToUpper(op), change.Path)
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

// saveStandaloneReviewToFile saves the review result to a file.
func saveStandaloneReviewToFile(result *conductor.StandaloneReviewResult, outputPath string) error {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# Standalone Code Review - %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("Verdict: %s\n\n", result.Verdict))

	if result.Summary != "" {
		content.WriteString("## Summary\n\n")
		content.WriteString(result.Summary)
		content.WriteString("\n\n")
	}

	if len(result.Issues) > 0 {
		content.WriteString("## Issues\n\n")
		for _, issue := range result.Issues {
			location := issue.File
			if issue.Line > 0 {
				location = fmt.Sprintf("%s:%d", issue.File, issue.Line)
			}
			content.WriteString(fmt.Sprintf("- [%s] %s: %s\n", strings.ToUpper(issue.Severity), location, issue.Message))
		}
		content.WriteString("\n")
	}

	content.WriteString("## Diff Reviewed\n\n")
	content.WriteString("```diff\n")
	content.WriteString(result.Diff)
	content.WriteString("\n```\n")

	return os.WriteFile(outputPath, []byte(content.String()), 0o644)
}

// containsIssues checks if the review output indicates issues.
func containsIssues(output string) bool {
	lowerOutput := strings.ToLower(output)
	issueIndicators := []string{
		"error",
		"warning",
		"issue",
		"problem",
		"fix",
		"should",
		"must",
		"consider",
		"recommend",
	}

	for _, indicator := range issueIndicators {
		if strings.Contains(lowerOutput, indicator) {
			return true
		}
	}

	return false
}

// getNextReviewFilename returns the next available review-N.txt filename.
func getNextReviewFilename(workDir string) string {
	for i := 1; i <= maxReviewFileNumber; i++ {
		filename := fmt.Sprintf("review-%d.txt", i)
		path := filepath.Join(workDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return filename
		}
	}

	return fmt.Sprintf("review-%d.txt", time.Now().Unix())
}
