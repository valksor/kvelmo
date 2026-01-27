package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/storage"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks in workspace",
	Long: `List all tasks in the workspace with their worktree paths and states.

This is useful for seeing all parallel tasks across multiple terminals.
Tasks with worktrees can be worked on independently in separate terminals.

Examples:
  mehr list                   # List all tasks
  mehr list --worktrees       # Show only tasks with worktrees
  mehr list --json            # Output as JSON
  mehr list --search "api"    # Search tasks by title
  mehr list --filter state:done  # Filter by state
  mehr list --sort cost        # Sort by cost (highest first)
  mehr list --format json     # JSON output`,
	RunE: runList,
}

var (
	listWorktreesOnly bool
	listSearch        string
	listFilter        string
	listSort          string
	listFormat        string
	listJSON          bool
	listLabelFilter   string
	listLabelAny      []string
	listNoLabel       bool
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listWorktreesOnly, "worktrees", "w", false, "Show only tasks with worktrees")
	listCmd.Flags().StringVar(&listSearch, "search", "", "Search tasks by title or description")
	listCmd.Flags().StringVar(&listFilter, "filter", "", "Filter tasks (format: key:value, e.g., state:done)")
	listCmd.Flags().StringVar(&listSort, "sort", "", "Sort tasks (date, cost, duration)")
	listCmd.Flags().StringVar(&listFormat, "format", "table", "Output format (table, json, csv)")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON (deprecated, use --format json)")
	listCmd.Flags().StringVar(&listLabelFilter, "label", "", "Filter by label (e.g., --label=priority:high)")
	listCmd.Flags().StringSliceVar(&listLabelAny, "label-any", nil, "Filter by any label (OR logic)")
	listCmd.Flags().BoolVar(&listNoLabel, "no-label", false, "Show only tasks without labels")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Handle legacy --json flag (map to --format json)
	if listJSON {
		listFormat = "json"
	}

	// Resolve workspace root and git context
	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}

	root := res.Root // Capture for later use

	ws, err := storage.OpenWorkspace(ctx, root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	// Get all tasks
	taskIDs, err := ws.ListWorks()
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}

	if len(taskIDs) == 0 {
		if listFormat == "json" {
			return outputJSON([]jsonListTask{})
		}
		fmt.Println("No tasks found in workspace.")
		fmt.Println("\nUse 'mehr start <reference>' to create a new task.")

		return nil
	}

	// Check which task is active (in main repo)
	var activeID string
	if ws.HasActiveTask() {
		active, _ := ws.LoadActiveTask()
		if active != nil {
			activeID = active.ID
		}
	}

	// Get current worktree path if we're in one
	var currentWorktreePath string
	if res.IsWorktree {
		currentWorktreePath = res.Git.Root()
	}

	// Load all tasks into a slice for filtering/sorting
	type taskInfo struct {
		ID           string
		State        string
		Title        string
		WorktreePath string
		IsActive     bool
		IsCurrent    bool
		Cost         int
		Duration     string
		Labels       string
	}

	var tasks []taskInfo
	for _, taskID := range taskIDs {
		work, err := ws.LoadWork(taskID)
		if err != nil {
			continue
		}

		// Filter by worktrees if requested
		if listWorktreesOnly && work.Git.WorktreePath == "" {
			continue
		}

		// Apply search filter
		if listSearch != "" {
			searchLower := strings.ToLower(listSearch)
			title := strings.ToLower(work.Metadata.Title)
			if !strings.Contains(title, searchLower) {
				continue
			}
		}

		// Get state
		state := "idle"
		isActive := taskID == activeID
		if isActive {
			active, _ := ws.LoadActiveTask()
			if active != nil {
				state = display.FormatStateString(active.State)
			}
		}

		// Apply state filter
		if listFilter != "" {
			parts := strings.SplitN(listFilter, ":", 2)
			if len(parts) == 2 && parts[0] == "state" {
				filterState := strings.ToLower(parts[1])
				stateLower := strings.ToLower(state)
				if !strings.Contains(stateLower, filterState) {
					continue
				}
			}
		}

		// Apply label filter (AND - must have all specified labels)
		if listLabelFilter != "" {
			found := false
			for _, label := range work.Metadata.Labels {
				if label == listLabelFilter {
					found = true

					break
				}
			}
			if !found {
				continue
			}
		}

		// Apply label-any filter (OR - must have at least one)
		if len(listLabelAny) > 0 {
			found := false
			for _, requiredLabel := range listLabelAny {
				for _, label := range work.Metadata.Labels {
					if label == requiredLabel {
						found = true

						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter for tasks without labels
		if listNoLabel && len(work.Metadata.Labels) > 0 {
			continue
		}

		// Format worktree path (relative if possible)
		worktreePath := ""
		if work.Git.WorktreePath != "" {
			worktreePath = work.Git.WorktreePath
			// Try to make it relative
			if rel, err := filepath.Rel(root, worktreePath); err == nil && len(rel) < len(worktreePath) {
				worktreePath = rel
			}
		}

		// Calculate cost (from sessions)
		cost := 0
		sessions, _ := ws.ListSessions(taskID)
		for _, s := range sessions {
			if s.Usage != nil {
				cost += s.Usage.InputTokens + s.Usage.OutputTokens
			}
		}

		// Calculate duration (rough estimate from created time)
		duration := "-"
		if !work.Metadata.CreatedAt.IsZero() {
			elapsed := time.Since(work.Metadata.CreatedAt)
			if elapsed < time.Minute {
				duration = fmt.Sprintf("%ds", int(elapsed.Seconds()))
			} else if elapsed < time.Hour {
				duration = fmt.Sprintf("%dm", int(elapsed.Minutes()))
			} else {
				duration = fmt.Sprintf("%dh", int(elapsed.Hours()))
			}
		}

		tasks = append(tasks, taskInfo{
			ID:           taskID,
			State:        state,
			Title:        work.Metadata.Title,
			WorktreePath: worktreePath,
			IsActive:     isActive,
			IsCurrent:    currentWorktreePath != "" && work.Git.WorktreePath == currentWorktreePath,
			Cost:         cost,
			Duration:     duration,
			Labels:       formatLabels(work.Metadata.Labels),
		})
	}

	// Apply sorting
	if listSort != "" {
		switch listSort {
		case "date":
			// Already sorted by date (task ID contains timestamp)
		case "cost":
			// Sort by cost (highest first)
			for i := range len(tasks) - 1 {
				for j := i + 1; j < len(tasks); j++ {
					if tasks[i].Cost < tasks[j].Cost {
						tasks[i], tasks[j] = tasks[j], tasks[i]
					}
				}
			}
		case "duration":
			// Sort by duration (would need to parse duration string)
			// For now, skip this
		}
	}

	// JSON output
	if listFormat == "json" {
		var jsonTasks []jsonListTask
		for _, task := range tasks {
			// Load work to get labels
			var labels []string
			if work, err := ws.LoadWork(task.ID); err == nil {
				labels = work.Metadata.Labels
			}

			jsonTasks = append(jsonTasks, jsonListTask{
				TaskID:       task.ID,
				State:        task.State,
				Title:        task.Title,
				WorktreePath: task.WorktreePath,
				IsActive:     task.IsActive,
				IsCurrent:    task.IsCurrent,
				Labels:       labels,
			})
		}

		return outputJSON(jsonTasks)
	}

	// CSV output
	if listFormat == "csv" {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ',', 0)
		if _, err := fmt.Fprintln(w, "Task ID,State,Title,Worktree,Active,Cost"); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
		for _, task := range tasks {
			title := task.Title
			if title == "" {
				title = "(untitled)"
			}
			activeMark := ""
			if task.IsActive {
				activeMark = "*"
			}
			if _, err := fmt.Fprintf(w, "%s,%s,%s,%s,%s,%d\n",
				task.ID, task.State, title, task.WorktreePath, activeMark, task.Cost); err != nil {
				return fmt.Errorf("write csv row: %w", err)
			}
		}
		if err := w.Flush(); err != nil {
			return fmt.Errorf("flush csv: %w", err)
		}

		return nil
	}

	// Regular text output (default)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "TASK ID\tSTATE\tTITLE\tWORKTREE\tACTIVE\tCOST\tLABELS"); err != nil {
		return fmt.Errorf("print header: %w", err)
	}

	for _, task := range tasks {
		// Format title
		title := task.Title
		if title == "" {
			title = "(untitled)"
		}
		if len(title) > 35 {
			title = title[:32] + "..."
		}

		// Format worktree path
		worktreePath := task.WorktreePath
		if worktreePath == "" {
			worktreePath = "-"
		}

		// Active marker
		activeMarker := ""
		if task.IsActive {
			activeMarker = "*"
		}
		if task.IsCurrent {
			activeMarker = "→"
		}

		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			task.ID, task.State, title, worktreePath, activeMarker, task.Cost, task.Labels); err != nil {
			return fmt.Errorf("print row: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush list table: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println()
		fmt.Println("No tasks found matching criteria.")
	} else {
		fmt.Println()
		fmt.Println("Legend: * = active task, → = current worktree")
		fmt.Println("Use --search, --filter, --sort for more options")
	}

	return nil
}

// JSON output structures for list command.
type jsonListTask struct {
	TaskID       string   `json:"task_id"`
	State        string   `json:"state"`
	Title        string   `json:"title"`
	WorktreePath string   `json:"worktree_path,omitempty"`
	IsActive     bool     `json:"is_active"`
	IsCurrent    bool     `json:"is_current_worktree"`
	Labels       []string `json:"labels,omitempty"`
}

// formatLabels formats a slice of labels as a comma-separated string.
func formatLabels(labels []string) string {
	if len(labels) == 0 {
		return "-"
	}

	return strings.Join(labels, ", ")
}
