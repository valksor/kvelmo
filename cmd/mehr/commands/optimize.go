package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/display"
)

var (
	optimizeTask  string
	optimizeAgent string
)

var optimizeCmd = &cobra.Command{
	Use:   "optimize --task <queue>/<task-id>",
	Short: "AI optimizes a task based on its notes",
	Long: `Use AI to improve a task description based on accumulated notes.

The agent will:
  - Review the current task and accumulated notes
  - Enhance the title for clarity
  - Expand the description with context from notes
  - Suggest relevant labels
  - Explain improvements made

USAGE:
  mehr optimize --task=<queue>/<task-id>

EXAMPLES:
  mehr optimize --task=quick-tasks/task-1
  mehr optimize --task=quick-tasks/task-1 --agent claude-opus
  mehr note --task=quick-tasks/task-1 "Add requirement"
  mehr optimize --task=quick-tasks/task-1  # Uses the new note

See also:
  mehr quick                 - Create a quick task
  mehr note --task=<ref>     - Add notes to a task
  mehr export --task=<ref>   - Export optimized task to file`,
	Args: cobra.NoArgs,
	RunE: runOptimize,
}

func init() {
	rootCmd.AddCommand(optimizeCmd)

	optimizeCmd.Flags().StringVar(&optimizeTask, "task", "", "Queue task ID (format: <queue-id>/<task-id>)")
	optimizeCmd.Flags().StringVar(&optimizeAgent, "agent", "", "Agent to use for optimization")
	//nolint:errcheck // Flag name is constant, error won't occur
	optimizeCmd.MarkFlagRequired("task")
}

func runOptimize(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Validate task reference
	if optimizeTask == "" {
		return errors.New("--task flag is required (format: <queue-id>/<task-id>)")
	}

	queueID, taskID, err := conductor.ParseQueueTaskRef(optimizeTask)
	if err != nil {
		return err
	}

	// Build conductor options
	opts := BuildConductorOptions(CommandOptions{
		Verbose: verbose,
	})

	if optimizeAgent != "" {
		opts = append(opts, conductor.WithAgent(optimizeAgent))
	}

	// Initialize conductor
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	// Show current task state
	if err := showTaskBeforeOptimization(cond, queueID, taskID); err != nil {
		return err
	}

	// Run optimization
	fmt.Println("\n✨ Optimizing task with AI...")
	fmt.Println()

	result, err := cond.OptimizeQueueTask(ctx, queueID, taskID)
	if err != nil {
		return fmt.Errorf("optimize task: %w", err)
	}

	// Display results
	displayOptimizeResult(result)

	// Show next steps
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  %s\n", display.Cyan(fmt.Sprintf("mehr export --task=%s/%s --output task.md", queueID, taskID)))
	fmt.Printf("  %s\n", display.Cyan(fmt.Sprintf("mehr submit --task=%s/%s --provider <provider>", queueID, taskID)))
	fmt.Printf("  %s\n", display.Cyan(fmt.Sprintf("mehr start queue:%s/%s", queueID, taskID)))

	return nil
}

// showTaskBeforeOptimization displays the current task state.
func showTaskBeforeOptimization(cond *conductor.Conductor, queueID, taskID string) error {
	// Load queue
	queue, err := storage.LoadTaskQueue(cond.GetWorkspace(), queueID)
	if err != nil {
		return fmt.Errorf("load queue: %w", err)
	}

	task := queue.GetTask(taskID)
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Load notes
	notes, _ := cond.GetWorkspace().LoadQueueNotes(queueID, taskID)

	fmt.Printf("Task: %s\n", display.Bold(task.ID))
	fmt.Printf("  Title: %s\n", task.Title)
	if task.Description != "" {
		fmt.Printf("  Description: %s\n", truncateText(task.Description, 80))
	}
	if len(task.Labels) > 0 {
		fmt.Printf("  Labels: %s\n", strings.Join(task.Labels, ", "))
	}
	if len(notes) > 0 {
		fmt.Printf("  Notes: %d\n", len(notes))
	}

	return nil
}

// displayOptimizeResult shows the result of task optimization.
func displayOptimizeResult(result *conductor.OptimizedTask) {
	fmt.Println("✨ Task optimized:")

	// Show title change
	if result.OriginalTitle != result.OptimizedTitle {
		fmt.Printf("  Title: %s → %s\n", display.Muted(result.OriginalTitle), display.Bold(result.OptimizedTitle))
	} else {
		fmt.Printf("  Title: %s (unchanged)\n", display.Bold(result.OptimizedTitle))
	}

	// Show description change
	if result.OriginalDesc != result.OptimizedDesc {
		fmt.Println("  Description: enhanced")
		if len(result.OptimizedDesc) > 0 && len(result.OptimizedDesc) < 200 {
			fmt.Printf("    %s\n", display.Muted(truncateText(result.OptimizedDesc, 100)))
		}
	} else {
		fmt.Println("  Description: unchanged")
	}

	// Show added labels
	if len(result.AddedLabels) > 0 {
		fmt.Printf("  Added labels: %s\n", strings.Join(result.AddedLabels, ", "))
	}

	// Show improvement notes
	if len(result.ImprovementNotes) > 0 {
		fmt.Println("\n  Improvements:")
		for _, note := range result.ImprovementNotes {
			fmt.Printf("    • %s\n", note)
		}
	}
}

// truncateText truncates text to maxLen with ellipsis.
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	return text[:maxLen-3] + "..."
}
