package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-toolkit/display"
)

var (
	exportTask   string
	exportOutput string
)

var exportCmd = &cobra.Command{
	Use:   "export --task <queue>/<task-id> --output <file>",
	Short: "Export a queue task to a markdown file",
	Long: `Export a queue task to a markdown file for use with the standard workflow.

The exported file includes:
  - YAML frontmatter with title, labels, priority
  - Task description
  - All accumulated notes

The exported file can be used with:
  mehr start file:<exported-file>

USAGE:
  mehr export --task=<queue>/<task-id> --output=<file>

EXAMPLES:
  mehr export --task=quick-tasks/task-1 --output task.md
  mehr export --task=quick-tasks/task-1 --output tasks/feature.md
  mehr start file:task.md  # Use the exported file

See also:
  mehr quick              - Create a quick task
  mehr optimize --task=<ref>  - Optimize before exporting
  mehr start file:<file>  - Start from exported file`,
	Args: cobra.NoArgs,
	RunE: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVar(&exportTask, "task", "", "Queue task ID (format: <queue-id>/<task-id>)")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path")
	//nolint:errcheck // Flag names are constants, error won't occur
	exportCmd.MarkFlagRequired("task")
	//nolint:errcheck // Flag names are constants, error won't occur
	exportCmd.MarkFlagRequired("output")
}

func runExport(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Validate flags
	if exportTask == "" {
		return errors.New("--task flag is required (format: <queue-id>/<task-id>)")
	}
	if exportOutput == "" {
		return errors.New("--output flag is required")
	}

	// Parse queue task reference
	queueID, taskID, err := conductor.ParseQueueTaskRef(exportTask)
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

	// Export task
	markdown, err := cond.ExportQueueTask(queueID, taskID)
	if err != nil {
		return fmt.Errorf("export task: %w", err)
	}

	// Write to a file
	if err := os.WriteFile(exportOutput, []byte(markdown), 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Println()
	fmt.Printf("✓ Exported to: %s\n", display.Bold(exportOutput))
	fmt.Printf("  Use with: %s\n", display.Cyan("mehr start file:"+exportOutput))

	return nil
}
