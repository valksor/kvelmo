package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/storage"
)

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Manage task labels",
	Long:  `Add, remove, list, and manage labels for tasks.`,
}

var labelAddCmd = &cobra.Command{
	Use:   "add <task-id> <label>...",
	Short: "Add labels to a task",
	Long:  `Add one or more labels to a task. Labels are added to existing labels.`,
	Args:  cobra.MinimumNArgs(2),
	RunE:  runLabelAdd,
}

var labelRemoveCmd = &cobra.Command{
	Use:   "remove <task-id> <label>...",
	Short: "Remove labels from a task",
	Long:  `Remove one or more labels from a task.`,
	Args:  cobra.MinimumNArgs(2),
	RunE:  runLabelRemove,
}

var labelSetCmd = &cobra.Command{
	Use:   "set <task-id> <label>...",
	Short: "Set task labels (replace all)",
	Long:  `Replace all labels on a task with the provided labels. Use with no labels to clear all.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runLabelSet,
}

var labelListCmd = &cobra.Command{
	Use:   "list <task-id>",
	Short: "List labels for a task",
	Long:  `Show all labels assigned to a task.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runLabelList,
}

func init() {
	rootCmd.AddCommand(labelCmd)
	labelCmd.AddCommand(labelAddCmd, labelRemoveCmd, labelSetCmd, labelListCmd)
}

func runLabelAdd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}

	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	taskID := args[0]
	labels := args[1:]

	if !ws.WorkExists(taskID) {
		return fmt.Errorf("task not found: %s", taskID)
	}

	for _, label := range labels {
		if err := ws.AddLabel(taskID, label); err != nil {
			return fmt.Errorf("add label %q: %w", label, err)
		}
	}

	fmt.Printf("Added %d label(s) to task %s\n", len(labels), taskID)

	return nil
}

func runLabelRemove(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}

	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	taskID := args[0]
	labels := args[1:]

	if !ws.WorkExists(taskID) {
		return fmt.Errorf("task not found: %s", taskID)
	}

	for _, label := range labels {
		if err := ws.RemoveLabel(taskID, label); err != nil {
			return fmt.Errorf("remove label %q: %w", label, err)
		}
	}

	fmt.Printf("Removed %d label(s) from task %s\n", len(labels), taskID)

	return nil
}

func runLabelSet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}

	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	taskID := args[0]
	labels := args[1:]

	if !ws.WorkExists(taskID) {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if err := ws.SetLabels(taskID, labels); err != nil {
		return fmt.Errorf("set labels: %w", err)
	}

	if len(labels) == 0 {
		fmt.Printf("Cleared all labels from task %s\n", taskID)
	} else {
		fmt.Printf("Set %d label(s) on task %s\n", len(labels), taskID)
	}

	return nil
}

func runLabelList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}

	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	taskID := args[0]

	labels, err := ws.GetLabels(taskID)
	if err != nil {
		return fmt.Errorf("get labels: %w", err)
	}

	work, _ := ws.LoadWork(taskID)
	title := taskID
	if work != nil && work.Metadata.Title != "" {
		title = fmt.Sprintf("%s (%s)", work.Metadata.Title, taskID)
	}

	fmt.Printf("Labels for %s:\n", title)
	if len(labels) == 0 {
		fmt.Println("  (no labels)")

		return nil
	}

	for _, label := range labels {
		fmt.Printf("  - %s\n", label)
	}

	return nil
}
