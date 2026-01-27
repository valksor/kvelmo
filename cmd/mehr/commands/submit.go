package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-toolkit/display"
)

var (
	submitTask     string
	submitProvider string
	submitLabels   []string
	submitDryRun   bool
)

var submitCmd = &cobra.Command{
	Use:   "submit --task <queue>/<task-id> --provider <name>",
	Short: "Submit a queue task to an external provider",
	Long: `Submit a single queue task to an external provider.

The task will be created in the external provider (GitHub, Jira, Wrike, etc.)
and the queue task will be updated with the external ID and URL.

USAGE:
  mehr submit --task=<queue>/<task-id> --provider=<name>

EXAMPLES:
  mehr submit --task=quick-tasks/task-1 --provider github
  mehr submit --task=quick-tasks/task-1 --provider wrike --labels urgent
  mehr submit --task=quick-tasks/task-1 --provider jira --dry-run

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
	//nolint:errcheck // Flag names are constants, error won't occur
	submitCmd.MarkFlagRequired("task")
	//nolint:errcheck // Flag names are constants, error won't occur
	submitCmd.MarkFlagRequired("provider")
}

func runSubmit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Validate flags
	if submitTask == "" {
		return errors.New("--task flag is required (format: <queue-id>/<task-id>)")
	}
	if submitProvider == "" {
		return errors.New("--provider flag is required")
	}

	// Parse queue task reference
	queueID, taskID, err := conductor.ParseQueueTaskRef(submitTask)
	if err != nil {
		return err
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

	// Show what we're doing
	fmt.Println()
	if submitDryRun {
		fmt.Printf("📤 Dry-run: Previewing submission to %s\n", display.Bold(submitProvider))
	} else {
		fmt.Printf("📤 Submitting to %s\n", display.Bold(submitProvider))
	}
	fmt.Printf("  Task: %s/%s\n", queueID, taskID)
	if len(submitLabels) > 0 {
		fmt.Printf("  Labels: %s\n", strings.Join(submitLabels, ", "))
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
