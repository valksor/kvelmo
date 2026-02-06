package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/storage"
	tkdisplay "github.com/valksor/go-toolkit/display"
)

var (
	specificationViewNumber int
	specificationViewAll    bool
	specificationViewOutput string

	specificationDiffNumber  int
	specificationDiffFile    string
	specificationDiffContext int
	specificationDiffOutput  string
)

var specificationViewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View a specification's content",
	Long: `Display the full content of a specification with metadata.

Shows the complete specification content with markdown formatting,
along with metadata like status, component, and timestamps.

Examples:
  mehr specification view 1              # View specification-1
  mehr specification view 1 -o spec.md   # Save to file
  mehr specification view --all          # View all specifications`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSpecificationView,
}

var specificationCmd = &cobra.Command{
	Use:   "specification",
	Short: "Manage specifications",
	Long: `View and manage task specifications.

Specifications are detailed implementation plans created by the AI
during the planning phase. Each specification contains what needs to
be built and how to implement it.`,
}

var specificationDiffCmd = &cobra.Command{
	Use:   "diff <number>",
	Short: "View unified diff for an implemented file in a specification",
	Long: `Display the actual git diff for a file listed in specification implemented_files.

This is read-only output intended for inspection. It does not apply changes
or trigger review workflows.

Examples:
  mehr specification diff 1 --file web/src/components/task/SpecificationsList.tsx
  mehr specification diff 1 --file internal/server/handlers.go --context 5
  mehr specification diff -n 1 -f internal/conductor/handlers.go -o spec.diff`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSpecificationDiff,
}

func init() {
	rootCmd.AddCommand(specificationCmd)
	specificationCmd.AddCommand(specificationViewCmd)
	specificationCmd.AddCommand(specificationDiffCmd)

	specificationViewCmd.Flags().IntVarP(&specificationViewNumber, "number", "n", 0, "Specification number")
	specificationViewCmd.Flags().BoolVarP(&specificationViewAll, "all", "a", false, "View all specifications")
	specificationViewCmd.Flags().StringVarP(&specificationViewOutput, "output", "o", "", "Save to file instead of printing")

	specificationDiffCmd.Flags().IntVarP(&specificationDiffNumber, "number", "n", 0, "Specification number")
	specificationDiffCmd.Flags().StringVarP(&specificationDiffFile, "file", "f", "", "Implemented file path")
	specificationDiffCmd.Flags().IntVarP(&specificationDiffContext, "context", "c", 3, "Diff context lines")
	specificationDiffCmd.Flags().StringVarP(&specificationDiffOutput, "output", "o", "", "Save diff to file instead of printing")
}

func runSpecificationView(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Build conductor
	cond, err := initializeConductor(ctx)
	if err != nil {
		return err
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return errors.New("no workspace available")
	}

	// Get active task
	activeTask := cond.GetActiveTask()
	if activeTask == nil {
		return errors.New("no active task. Start a task first with 'mehr start'")
	}
	taskID := activeTask.ID

	// Parse specification number from argument or flag
	number := specificationViewNumber
	if number == 0 && len(args) > 0 {
		num, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid specification number: %w", err)
		}
		number = num
	}

	// Get all specifications to check what exists
	specifications, err := ws.ListSpecificationsWithStatus(taskID)
	if err != nil {
		return fmt.Errorf("list specifications: %w", err)
	}

	if len(specifications) == 0 {
		fmt.Printf("No specifications yet. Run 'mehr plan' to create them.\n")

		return nil
	}

	// View all specifications if --all flag is set
	if specificationViewAll {
		return viewAllSpecifications(ctx, ws, taskID, specifications, specificationViewOutput)
	}

	// Validate specification number
	if number == 0 {
		return errors.New("specification number required. Use: mehr specification view <number>")
	}

	// Find the specification
	var target *storage.Specification
	for _, spec := range specifications {
		if spec.Number == number {
			target = spec

			break
		}
	}

	if target == nil {
		// Show available specifications
		fmt.Printf("Specification %d not found. Available specifications:\n", number)
		for _, spec := range specifications {
			statusIcon := display.GetSpecificationStatusIcon(spec.Status)
			fmt.Printf("  %s specification-%d: %s [%s]\n",
				statusIcon, spec.Number, spec.Title, display.FormatSpecificationStatus(spec.Status))
		}

		return fmt.Errorf("specification %d not found", number)
	}

	// Load and display specification
	return displaySpecification(ctx, ws, taskID, target, specificationViewOutput)
}

func viewAllSpecifications(_ context.Context, ws *storage.Workspace, taskID string, specifications []*storage.Specification, outputPath string) error {
	var outputs []string

	for _, spec := range specifications {
		var content string
		var err error

		if outputPath != "" {
			// For multiple specs, append number to filename
			if len(specifications) > 1 {
				baseName := strings.TrimSuffix(outputPath, ".md")
				content, err = ws.LoadSpecification(taskID, spec.Number)
				if err != nil {
					return fmt.Errorf("load specification %d: %w", spec.Number, err)
				}
				outputPath := fmt.Sprintf("%s-%d.md", baseName, spec.Number)
				if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
					return fmt.Errorf("write specification %d: %w", spec.Number, err)
				}
				fmt.Printf("Specification %d saved to: %s\n", spec.Number, outputPath)
			}
		} else {
			// Print to stdout with separator
			if len(outputs) > 0 {
				outputs = append(outputs, "\n"+strings.Repeat("─", 80)+"\n")
			}
			specContent, err := ws.LoadSpecification(taskID, spec.Number)
			if err != nil {
				return fmt.Errorf("load specification %d: %w", spec.Number, err)
			}
			outputs = append(outputs, formatSpecificationHeader(spec))
			outputs = append(outputs, specContent)
		}
	}

	if outputPath == "" {
		fmt.Print(strings.Join(outputs, ""))
	}

	return nil
}

func displaySpecification(_ context.Context, ws *storage.Workspace, taskID string, spec *storage.Specification, outputPath string) error {
	// Load specification content
	content, err := ws.LoadSpecification(taskID, spec.Number)
	if err != nil {
		return fmt.Errorf("load specification: %w", err)
	}

	// Format output
	output := formatSpecificationHeader(spec) + content

	// Write to file or stdout
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(output), 0o644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
		fmt.Printf("Specification %d saved to: %s\n", spec.Number, outputPath)
	} else {
		fmt.Print(output)
	}

	return nil
}

func formatSpecificationHeader(spec *storage.Specification) string {
	var sb strings.Builder
	statusIcon := display.GetSpecificationStatusIcon(spec.Status)

	// Header with number and title
	sb.WriteString(tkdisplay.Bold(fmt.Sprintf("─ Specification %d", spec.Number)))
	if spec.Title != "" {
		sb.WriteString(": " + spec.Title)
	}
	sb.WriteString("\n\n")

	// Metadata
	sb.WriteString(tkdisplay.Muted("Status:     "))
	sb.WriteString(fmt.Sprintf("%s %s\n", statusIcon, display.FormatSpecificationStatus(spec.Status)))

	if spec.Component != "" {
		sb.WriteString(tkdisplay.Muted("Component:  "))
		sb.WriteString(spec.Component + "\n")
	}

	if !spec.CreatedAt.IsZero() {
		sb.WriteString(tkdisplay.Muted("Created:    "))
		sb.WriteString(spec.CreatedAt.Format("2006-01-02 15:04") + "\n")
	}

	if !spec.CompletedAt.IsZero() {
		sb.WriteString(tkdisplay.Muted("Completed:  "))
		sb.WriteString(spec.CompletedAt.Format("2006-01-02 15:04") + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", 80))
	sb.WriteString("\n\n")

	return sb.String()
}

func runSpecificationDiff(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cond, err := initializeConductor(ctx)
	if err != nil {
		return err
	}

	activeTask := cond.GetActiveTask()
	if activeTask == nil {
		return errors.New("no active task. Start a task first with 'mehr start'")
	}

	number := specificationDiffNumber
	if number == 0 && len(args) > 0 {
		num, parseErr := strconv.Atoi(args[0])
		if parseErr != nil {
			return fmt.Errorf("invalid specification number: %w", parseErr)
		}
		number = num
	}

	if number <= 0 {
		return errors.New("specification number required. Use: mehr specification diff <number>")
	}

	if specificationDiffFile == "" {
		return errors.New("file path required. Use: --file <path>")
	}

	diff, err := cond.GetSpecificationFileDiff(ctx, activeTask.ID, number, specificationDiffFile, specificationDiffContext)
	if err != nil {
		return err
	}

	if diff == "" {
		fmt.Printf("No diff found for %s in specification-%d.\n", specificationDiffFile, number)

		return nil
	}

	if specificationDiffOutput != "" {
		if writeErr := os.WriteFile(specificationDiffOutput, []byte(diff), 0o644); writeErr != nil {
			return fmt.Errorf("write diff file: %w", writeErr)
		}
		fmt.Printf("Diff saved to: %s\n", specificationDiffOutput)

		return nil
	}

	fmt.Print(diff)

	return nil
}
