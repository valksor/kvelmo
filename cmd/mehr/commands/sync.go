// Package commands provide CLI commands for mehr.
package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/display"
)

var syncCmd = &cobra.Command{
	Use:   "sync <task-id>",
	Short: "Sync task from provider and generate delta specification if changed",
	Long: `Sync task data from the provider (e.g., Wrike, GitHub) and generate
a delta specification if changes are detected.

This is useful when a task has been modified in the external system and you
want to update your local work accordingly.

The command will:
1. Fetch the latest version of the task from the provider
2. Compare it with the stored local version
3. Generate a delta specification if changes are detected
4. Save the new specification to the specifications directory

Example:
  mehr sync TASK-123`,
	Args: cobra.ExactArgs(1),
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	ctx := cmd.Context()

	// Initialize conductor
	c, err := initializeConductor(ctx)
	if err != nil {
		return err
	}

	result, err := c.SyncTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("sync task: %w", err)
	}

	if len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			fmt.Printf("%s Warning: %s\n", display.Warning("→"), warning)
		}
	}

	if !result.HasChanges {
		fmt.Println(display.Success("✓") + " No changes detected in the task.")

		return nil
	}

	fmt.Printf("\n%s Changes detected:\n", display.Warning("▲"))
	fmt.Printf("  %s\n\n", display.Muted(result.ChangesSummary))

	fmt.Printf("\n%s Generated delta specification: %s\n", display.Success("✓"), display.Bold(result.SpecGenerated))
	if result.PreviousSnapshotPath != "" {
		fmt.Printf("%s Previous Wrike snapshot: %s\n", display.Info("→"), result.PreviousSnapshotPath)
	}
	if result.DiffPath != "" {
		fmt.Printf("%s Wrike diff summary: %s\n", display.Info("→"), result.DiffPath)
	}
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Review the delta specification\n")
	fmt.Printf("  2. Run %s to create an implementation plan\n", display.Bold("mehr plan"))
	fmt.Printf("  3. Run %s to apply the changes\n", display.Bold("mehr implement"))

	return nil
}

// extractContent extracts displayable text from a work unit for legacy CLI tests.
func extractContent(wu *provider.WorkUnit) string {
	var content strings.Builder

	if wu.Title != "" {
		content.WriteString("# ")
		content.WriteString(wu.Title)
		content.WriteString("\n\n")
	}
	if wu.Description != "" {
		content.WriteString(wu.Description)
		content.WriteString("\n")
	}
	if len(wu.Comments) > 0 {
		content.WriteString("\n## Comments\n\n")
		for _, comment := range wu.Comments {
			author := provider.ResolveAuthor(comment)
			if author == "" {
				author = comment.Author.ID
			}
			content.WriteString("### ")
			content.WriteString(author)
			content.WriteString("\n\n")
			content.WriteString(comment.Body)
			content.WriteString("\n\n")
		}
	}

	return content.String()
}
