package display

import (
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/valksor/go-toolkit/display"
)

// QuickTaskSuccess formats a success message for quick task creation.
func QuickTaskSuccess(taskID, title, queueID string) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("✓ Created task: %s\n", display.Success(taskID)))
	sb.WriteString(fmt.Sprintf("  Title: %s\n", display.Bold(title)))
	sb.WriteString(fmt.Sprintf("  Queue: %s\n", queueID))

	return sb.String()
}

// QuickTaskList formats a list of quick tasks for display.
func QuickTaskList(tasks []QuickTaskItem) string {
	if len(tasks) == 0 {
		return "No tasks found."
	}

	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	//nolint:errcheck // Writing to string builder won't fail
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "ID", "Title", "Labels", "Notes")
	//nolint:errcheck // Writing to string builder won't fail
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "──", "─────", "─────", "─────")

	for _, task := range tasks {
		labels := strings.Join(task.Labels, ",")
		if labels == "" {
			labels = "-"
		}
		noteCount := strconv.Itoa(task.NoteCount)
		if task.NoteCount == 0 {
			noteCount = "-"
		}

		title := task.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		//nolint:errcheck // Writing to string builder won't fail
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", task.ID, title, labels, noteCount)
	}

	//nolint:errcheck // Flushing to string builder won't fail
	w.Flush()

	return sb.String()
}

// OptimizedTaskChange formats the changes made during optimization.
func OptimizedTaskChange(result OptimizedTaskResult) string {
	var sb strings.Builder

	sb.WriteString("✨ Task optimized:\n")

	// Title change
	if result.OriginalTitle != result.OptimizedTitle {
		sb.WriteString(fmt.Sprintf("  Title: %s → %s\n",
			display.Muted(result.OriginalTitle),
			display.Bold(result.OptimizedTitle)))
	} else {
		sb.WriteString(fmt.Sprintf("  Title: %s (unchanged)\n", display.Bold(result.OptimizedTitle)))
	}

	// Description change
	if result.DescriptionChanged {
		sb.WriteString("  Description: enhanced\n")
	}

	// Added labels
	if len(result.AddedLabels) > 0 {
		sb.WriteString(fmt.Sprintf("  Added labels: %s\n", strings.Join(result.AddedLabels, ", ")))
	}

	// Improvements
	if len(result.Improvements) > 0 {
		sb.WriteString("\n  Improvements:\n")
		for _, note := range result.Improvements {
			sb.WriteString(fmt.Sprintf("    • %s\n", note))
		}
	}

	return sb.String()
}

// QuickTaskMenu formats the interactive menu shown after creating a task.
func QuickTaskMenu() string {
	return `
What next?
  [d]iscuss - Enter discussion mode (add notes)
  [o]ptimize - AI optimizes task based on notes
  [s]ubmit - Submit to provider
  [tart]   - Start working on it
  [x]exit   - Done for now
`
}

// QuickTaskItem represents a quick task for list display.
type QuickTaskItem struct {
	ID        string
	Title     string
	Labels    []string
	NoteCount int
}

// OptimizedTaskResult represents the result of task optimization for display.
type OptimizedTaskResult struct {
	OriginalTitle      string
	OptimizedTitle     string
	DescriptionChanged bool
	AddedLabels        []string
	Improvements       []string
}

// FormatSubmitSuccess formats the success message for task submission.
func FormatSubmitSuccess(provider, externalID, externalURL string) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("✓ Submitted:\n")
	sb.WriteString(fmt.Sprintf("  Provider: %s\n", provider))
	sb.WriteString(fmt.Sprintf("  External ID: %s\n", display.Bold(externalID)))
	if externalURL != "" {
		sb.WriteString(fmt.Sprintf("  URL: %s\n", display.Cyan(externalURL)))
	}

	return sb.String()
}

// FormatExportSuccess formats the success message for task export.
func FormatExportSuccess(filename string) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("✓ Exported to: %s\n", display.Bold(filename)))
	sb.WriteString(fmt.Sprintf("  Use with: mehr start file:%s\n", display.Cyan(filename)))

	return sb.String()
}
