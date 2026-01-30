package commands

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/display"
)

var deleteTask string

var deleteCmd = &cobra.Command{
	Use:   "delete --task <queue>/<task-id>",
	Short: "Delete a queue task",
	Long: `Delete a task from a queue.

This permanently removes the task and its associated notes from the queue.

USAGE:
  mehr delete --task=<queue>/<task-id>

EXAMPLES:
  mehr delete --task=quick-tasks/task-1
  mehr delete --task=project-queue/task-5

NOTES:
  - This action is permanent and cannot be undone
  - The task's notes file is also deleted
  - Use 'mehr list' to see available tasks

See also:
  mehr quick                 - Create a quick task
  mehr list                  - List all tasks
  mehr optimize --task=<ref> - Optimize a task`,
	Args: cobra.NoArgs,
	RunE: runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().StringVar(&deleteTask, "task", "", "Queue task ID (format: <queue-id>/<task-id>)")
	//nolint:errcheck // Flag name is constant, error won't occur
	deleteCmd.MarkFlagRequired("task")
}

func runDelete(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Validate task reference
	if deleteTask == "" {
		return errors.New("--task flag is required (format: <queue-id>/<task-id>)")
	}

	queueID, taskID, err := conductor.ParseQueueTaskRef(deleteTask)
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

	ws := cond.GetWorkspace()
	if ws == nil {
		return errors.New("workspace not initialized")
	}

	// Load queue
	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		return fmt.Errorf("queue not found: %s", queueID)
	}

	// Get task info before deletion (for display)
	task := queue.GetTask(taskID)
	if task == nil {
		return fmt.Errorf("task not found: %s/%s", queueID, taskID)
	}
	taskTitle := task.Title

	// Remove task
	if !queue.RemoveTask(taskID) {
		return fmt.Errorf("task not found: %s/%s", queueID, taskID)
	}

	// Save queue
	if err := queue.Save(); err != nil {
		return fmt.Errorf("failed to save queue: %w", err)
	}

	// Delete notes file
	notesPath := ws.QueueNotePath(queueID, taskID)
	_ = ws.DeleteFile(notesPath)

	slog.Info("queue task deleted", "queue_id", queueID, "task_id", taskID)

	// Display success
	fmt.Printf("✓ Deleted task %s\n", display.Bold(taskID))
	fmt.Printf("  Queue: %s\n", display.Cyan(queueID))
	if taskTitle != "" {
		fmt.Printf("  Title: %s\n", display.Muted(taskTitle))
	}

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  %s\n", display.Cyan("mehr list"))
	fmt.Printf("  %s\n", display.Cyan("mehr quick <description>"))

	return nil
}
