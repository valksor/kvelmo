package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-toolkit/display"
)

var (
	submitTask         string
	submitProvider     string
	submitLabels       []string
	submitDryRun       bool
	submitSource       string
	submitNotes        []string
	submitTitle        string
	submitInstructions string
	submitQueue        string
	submitOptimize     bool
)

var submitCmd = &cobra.Command{
	Use:   "submit --provider <name> [--task <queue>/<task-id> | --source <path>]",
	Short: "Submit a task to an external provider",
	Long: `Submit a single queue task to an external provider, or create one from a source and submit.

The task will be created in the external provider (GitHub, Jira, Wrike, etc.)
and the queue task will be updated with the external ID and URL.

USAGE:
  mehr submit --provider=<name> --task=<queue>/<task-id>
  mehr submit --provider=<name> --source=<path-or-ref>

EXAMPLES:
  mehr submit --task=quick-tasks/task-1 --provider github
  mehr submit --task=quick-tasks/task-1 --provider wrike --labels urgent
  mehr submit --task=quick-tasks/task-1 --provider jira --dry-run
  mehr submit --provider github --source ./specs/overview.md --note "Prefer tasks scoped to backend"
  mehr submit --provider jira --source ./docs/ --optimize --dry-run

Supported providers: github, gitlab, jira, linear, asana, notion, trello, wrike,
youtrack, bitbucket, clickup, azuredevops

See also:
  mehr quick              - Create a quick task
  mehr optimize --task=<ref>  - Optimize before submitting
  mehr export --task=<ref>   - Export to file instead`,
	Args: cobra.NoArgs,
	RunE: runSubmit,
}

func init() {
	rootCmd.AddCommand(submitCmd)

	submitCmd.Flags().StringVar(&submitTask, "task", "", "Queue task ID (format: <queue-id>/<task-id>)")
	submitCmd.Flags().StringVar(&submitProvider, "provider", "", "Provider name (github, wrike, jira, etc.)")
	submitCmd.Flags().StringSliceVar(&submitLabels, "labels", []string{}, "Additional labels to apply")
	submitCmd.Flags().BoolVar(&submitDryRun, "dry-run", false, "Preview without submitting")
	submitCmd.Flags().StringVar(&submitSource, "source", "", "Create task from a file/dir/provider ref and submit")
	submitCmd.Flags().StringSliceVar(&submitNotes, "note", []string{}, "Notes to guide task creation (repeatable)")
	submitCmd.Flags().StringVar(&submitTitle, "title", "", "Title override when creating from source")
	submitCmd.Flags().StringVar(&submitInstructions, "instructions", "", "Custom instructions for task creation")
	submitCmd.Flags().StringVar(&submitQueue, "queue", "", "Queue ID to store the created task (default: quick-tasks)")
	submitCmd.Flags().BoolVar(&submitOptimize, "optimize", false, "Optimize the generated task before submitting")
	//nolint:errcheck // Flag names are constants, error won't occur
	submitCmd.MarkFlagRequired("provider")
}

func runSubmit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Validate flags
	if submitProvider == "" {
		return errors.New("--provider flag is required")
	}
	if submitTask != "" && submitSource != "" {
		return errors.New("--task and --source cannot be used together")
	}
	if submitSource == "" && submitTask == "" {
		return errors.New("either --task or --source is required")
	}

	// Build conductor options
	opts := BuildConductorOptions(CommandOptions{
		Verbose: verbose,
	})

	// Initialize conductor
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	if submitSource != "" {
		return runSubmitFromSource(ctx, cond)
	}

	// Parse queue task reference
	queueID, taskID, err := conductor.ParseQueueTaskRef(submitTask)
	if err != nil {
		return err
	}

	// Submit task
	result, err := cond.SubmitQueueTask(ctx, queueID, taskID, conductor.SubmitOptions{
		Provider: submitProvider,
		Labels:   submitLabels,
		TaskIDs:  []string{taskID},
		DryRun:   submitDryRun,
	})
	if err != nil {
		return fmt.Errorf("submit task: %w", err)
	}

	// Display results
	displaySubmitResults(result, submitDryRun)

	return nil
}

func runSubmitFromSource(ctx context.Context, cond *conductor.Conductor) error {
	fmt.Println()
	if submitDryRun {
		fmt.Printf("📤 Dry-run: Previewing submission to %s\n", display.Bold(submitProvider))
	} else {
		fmt.Printf("📤 Submitting to %s\n", display.Bold(submitProvider))
	}
	fmt.Printf("  Source: %s\n", submitSource)
	if submitTitle != "" {
		fmt.Printf("  Title: %s\n", submitTitle)
	}
	if len(submitNotes) > 0 {
		fmt.Printf("  Notes: %d\n", len(submitNotes))
	}
	if len(submitLabels) > 0 {
		fmt.Printf("  Labels: %s\n", strings.Join(submitLabels, ", "))
	}

	sourceResult, err := cond.CreateQueueTaskFromSource(ctx, submitSource, conductor.SourceTaskOptions{
		QueueID:      submitQueue,
		Title:        submitTitle,
		Instructions: submitInstructions,
		Notes:        submitNotes,
		Provider:     submitProvider,
		Labels:       submitLabels,
	})
	if err != nil {
		return fmt.Errorf("create task from source: %w", err)
	}

	queueID := sourceResult.QueueID
	taskID := sourceResult.TaskID

	fmt.Printf("  Task: %s/%s\n", queueID, taskID)

	if submitOptimize {
		fmt.Println("\n✨ Optimizing task with AI...")
		fmt.Println()
		optResult, err := cond.OptimizeQueueTask(ctx, queueID, taskID)
		if err != nil {
			return fmt.Errorf("optimize task: %w", err)
		}
		displayOptimizeResult(optResult)
	}

	result, err := cond.SubmitQueueTask(ctx, queueID, taskID, conductor.SubmitOptions{
		Provider: submitProvider,
		Labels:   submitLabels,
		TaskIDs:  []string{taskID},
		DryRun:   submitDryRun,
	})
	if err != nil {
		return fmt.Errorf("submit task: %w", err)
	}

	displaySubmitResults(result, submitDryRun)

	return nil
}

// displaySubmitResults displays the submission results.
func displaySubmitResults(result *conductor.SubmitResult, dryRun bool) {
	if len(result.Tasks) == 0 {
		fmt.Println("\n  No tasks submitted")

		return
	}

	task := result.Tasks[0]
	fmt.Println()

	if dryRun {
		fmt.Println("  Dry-run preview:")
		fmt.Printf("    Task ID: %s\n", task.LocalID)
		fmt.Printf("    Title: %s\n", task.Title)
		fmt.Println("\n  Remove --dry-run to actually submit.")
	} else {
		fmt.Println("  ✓ Submitted:")
		fmt.Printf("    Local ID: %s\n", display.Bold(task.LocalID))
		fmt.Printf("    External ID: %s\n", display.Bold(task.ExternalID))
		if task.ExternalURL != "" {
			fmt.Printf("    URL: %s\n", display.Cyan(task.ExternalURL))
		}
	}

	// Show epic if created
	if result.Epic != nil {
		fmt.Printf("    Epic: %s\n", display.Bold(result.Epic.ExternalID))
		if result.Epic.ExternalURL != "" {
			fmt.Printf("    Epic URL: %s\n", display.Cyan(result.Epic.ExternalURL))
		}
	}
}
