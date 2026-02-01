// Package commands provide CLI commands for mehr.
package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/workflow"
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

	// Get workspace from a conductor
	ws := c.GetWorkspace()

	// Load task work
	work, err := ws.LoadWork(taskID)
	if err != nil {
		return fmt.Errorf("load task work: %w", err)
	}

	taskDir := ws.WorkPath(taskID)

	// Validate task directory
	if taskDir == "" {
		return errors.New("task directory is empty")
	}

	// Load source content to reconstruct the current work unit
	var sourcePath string
	if len(work.Source.Files) > 0 {
		sourcePath = work.Source.Files[0]
		// Validate path is not absolute
		if filepath.IsAbs(sourcePath) {
			return fmt.Errorf("source file path is absolute, expected relative: %s", sourcePath)
		}
		sourcePath = filepath.Join(taskDir, sourcePath)
	} else {
		// Fallback for older tasks
		sourcePath = filepath.Join(taskDir, "source", work.Source.Type+".txt")
	}

	sourceContent, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}

	// Reconstruct work unit from work metadata
	workUnit := &provider.WorkUnit{
		ID:          work.Metadata.ID,
		ExternalID:  work.Source.Ref,
		Provider:    work.Source.Type,
		Title:       work.Metadata.Title,
		Description: string(sourceContent),
	}

	fmt.Printf(display.Info("→")+" Fetching latest task data from %s...\n", workUnit.Provider)

	// Fetch updated version from the provider
	updated, err := fetchUpdatedFromProvider(ctx, c, workUnit)
	if err != nil {
		return fmt.Errorf("fetch updated task: %w", err)
	}

	fmt.Println(display.Info("→") + " Detecting changes...")

	// Detect changes
	changes := provider.DetectChanges(workUnit, updated)

	// If no changes, report and exit
	if !changes.HasChanges {
		fmt.Println(display.Success("✓") + " No changes detected in the task.")

		return nil
	}

	// Report changes
	fmt.Printf("\n%s Changes detected:\n", display.Warning("▲"))
	fmt.Printf("  %s\n\n", display.Muted(changes.Summary()))

	// Create delta specification generator
	gen := workflow.NewGenerator(taskDir)

	// Back up original source file
	if err := gen.BackupSourceFile(sourcePath); err != nil {
		fmt.Printf("%s Warning: could not backup source file: %v\n", display.Warning("→"), err)
	}

	// Write a diff file
	if err := gen.WriteDiffFile(changes); err != nil {
		fmt.Printf("%s Warning: could not write diff file: %v\n", display.Warning("→"), err)
	}

	// Extract old and new content for comparison
	oldContent := extractContent(workUnit)
	newContent := extractContent(updated)

	fmt.Println(display.Info("→") + " Generating delta specification...")

	// Add timeout for AI agent call (10 minutes max)
	genCtx, genCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer genCancel()

	// Generate delta specification
	specificationPath, err := gen.GenerateDeltaSpecification(genCtx, changes, oldContent, newContent)
	if err != nil {
		return fmt.Errorf("generate delta specification: %w", err)
	}

	fmt.Printf("\n%s Generated delta specification: %s\n", display.Success("✓"), display.Bold(specificationPath))
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Review the delta specification\n")
	fmt.Printf("  2. Run %s to create an implementation plan\n", display.Bold("mehr plan"))
	fmt.Printf("  3. Run %s to apply the changes\n", display.Bold("mehr implement"))

	return nil
}

// fetchUpdatedFromProvider fetches the updated version of the task from the provider.
func fetchUpdatedFromProvider(ctx context.Context, c *conductor.Conductor, old *provider.WorkUnit) (*provider.WorkUnit, error) {
	// Resolve provider and get instance
	// Use empty config - providers will use environment variables for authentication
	registry := c.GetProviderRegistry()
	providerInstance, id, err := registry.Resolve(ctx, old.ExternalID, provider.NewConfig(), provider.ResolveOptions{})
	if err != nil {
		return nil, fmt.Errorf("resolve provider: %w", err)
	}

	// Check if the provider supports reading
	reader, ok := providerInstance.(provider.Reader)
	if !ok {
		return nil, fmt.Errorf("provider %s does not support reading", old.Provider)
	}

	// Fetch the updated task
	updated, err := reader.Fetch(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetch from provider: %w", err)
	}

	return updated, nil
}

// extractContent extracts the main content from a work unit for comparison.
func extractContent(wu *provider.WorkUnit) string {
	var content string

	if wu.Title != "" {
		content += fmt.Sprintf("# %s\n\n", wu.Title)
	}

	if wu.Description != "" {
		content += wu.Description
		content += "\n"
	}

	if len(wu.Comments) > 0 {
		content += "\n## Comments\n\n"
		var commentsBuilder strings.Builder
		for _, comment := range wu.Comments {
			author := provider.ResolveAuthor(comment)
			if author == "" {
				author = comment.Author.ID
			}
			commentsBuilder.WriteString("### " + author + "\n\n" + comment.Body + "\n\n")
		}
		content += commentsBuilder.String()
	}

	return content
}
