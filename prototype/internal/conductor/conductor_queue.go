package conductor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// QuickTaskOptions configures quick task creation.
type QuickTaskOptions struct {
	Description string
	Title       string
	Priority    int
	Labels      []string
	QueueID     string // Defaults to "quick-tasks"
}

// QuickTaskResult holds the result of creating a quick task.
type QuickTaskResult struct {
	QueueID   string
	TaskID    string
	Title     string
	CreatedAt time.Time
}

// OptimizedTask holds the result of optimizing a task.
type OptimizedTask struct {
	OriginalTitle    string
	OriginalDesc     string
	OptimizedTitle   string
	OptimizedDesc    string
	AddedLabels      []string
	ImprovementNotes []string
}

// CreateQuickTask creates a quick task in a queue.
func (c *Conductor) CreateQuickTask(ctx context.Context, opts QuickTaskOptions) (*QuickTaskResult, error) {
	queueID := opts.QueueID
	if queueID == "" {
		queueID = "quick-tasks"
	}

	// Load or create queue
	queue, err := loadOrCreateQueue(c.workspace, queueID)
	if err != nil {
		return nil, fmt.Errorf("load queue: %w", err)
	}

	// Extract title from description if not provided
	title := opts.Title
	if title == "" {
		title = extractTitle(opts.Description)
	}

	// Create task
	task := &storage.QueuedTask{
		ID:          queue.NextTaskID(),
		Title:       title,
		Description: opts.Description,
		Status:      storage.TaskStatusReady,
		Priority:    opts.Priority,
		Labels:      opts.Labels,
	}

	// Add default label if none provided
	if len(task.Labels) == 0 {
		task.Labels = []string{"quick-task"}
	}

	// Add to queue and save
	queue.AddTask(task)
	if err := queue.Save(); err != nil {
		return nil, fmt.Errorf("save queue: %w", err)
	}

	return &QuickTaskResult{
		QueueID:   queueID,
		TaskID:    task.ID,
		Title:     task.Title,
		CreatedAt: time.Now(),
	}, nil
}

// OptimizeQueueTask uses AI to optimize a task based on its notes.
func (c *Conductor) OptimizeQueueTask(ctx context.Context, queueID, taskID string) (*OptimizedTask, error) {
	// Load queue and task
	queue, err := storage.LoadTaskQueue(c.workspace, queueID)
	if err != nil {
		return nil, fmt.Errorf("load queue: %w", err)
	}

	task := queue.GetTask(taskID)
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// Load notes for this task
	notes, _ := c.workspace.LoadQueueNotes(queueID, taskID)

	// Build optimization prompt
	prompt := buildOptimizePrompt(task, notes)

	// Get agent for optimization
	ag, err := c.GetAgentForStep(ctx, workflow.StepOptimizing)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}

	// Run optimization
	resp, err := ag.Run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("run optimization: %w", err)
	}

	// Parse optimized result
	optimized, err := parseOptimizedTask(resp.Summary)
	if err != nil {
		return nil, fmt.Errorf("parse optimized task: %w", err)
	}

	// Store original for comparison
	result := &OptimizedTask{
		OriginalTitle: task.Title,
		OriginalDesc:  task.Description,
	}

	// Update task in queue
	if err := queue.UpdateTask(taskID, func(t *storage.QueuedTask) {
		if optimized.Title != "" {
			t.Title = optimized.Title
			result.OptimizedTitle = optimized.Title
		} else {
			result.OptimizedTitle = t.Title
		}

		if optimized.Description != "" {
			t.Description = optimized.Description
			result.OptimizedDesc = optimized.Description
		} else {
			result.OptimizedDesc = t.Description
		}

		if len(optimized.Labels) > 0 {
			// Merge labels (avoid duplicates)
			labelMap := make(map[string]bool)
			for _, l := range t.Labels {
				labelMap[l] = true
			}
			for _, l := range optimized.Labels {
				if !labelMap[l] {
					t.Labels = append(t.Labels, l)
					result.AddedLabels = append(result.AddedLabels, l)
				}
			}
		}

		result.ImprovementNotes = optimized.Notes
	}); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	// Save queue
	if err := queue.Save(); err != nil {
		return nil, fmt.Errorf("save queue: %w", err)
	}

	return result, nil
}

// ExportQueueTask exports a queue task to markdown format.
func (c *Conductor) ExportQueueTask(queueID, taskID string) (string, error) {
	// Load queue and task
	queue, err := storage.LoadTaskQueue(c.workspace, queueID)
	if err != nil {
		return "", fmt.Errorf("load queue: %w", err)
	}

	task := queue.GetTask(taskID)
	if task == nil {
		return "", fmt.Errorf("task not found: %s", taskID)
	}

	// Load notes
	notes, _ := c.workspace.LoadQueueNotes(queueID, taskID)

	// Build markdown
	return buildTaskMarkdown(task, notes), nil
}

// SubmitQueueTask submits a single queue task to a provider.
func (c *Conductor) SubmitQueueTask(ctx context.Context, queueID, taskID string, opts SubmitOptions) (*SubmitResult, error) {
	// Ensure TaskIDs is set for single task
	if opts.TaskIDs == nil {
		opts.TaskIDs = []string{taskID}
	}

	// Use existing SubmitProjectTasks
	return c.SubmitProjectTasks(ctx, queueID, opts)
}

// loadOrCreateQueue loads an existing queue or creates a new one.
func loadOrCreateQueue(ws *storage.Workspace, queueID string) (*storage.TaskQueue, error) {
	// Try to load existing queue
	if ws.QueueExists(queueID) {
		return storage.LoadTaskQueue(ws, queueID)
	}

	// Create new queue
	queue := storage.NewTaskQueue(queueID, queueTitleFromID(queueID), "quick")

	// Ensure queue directory exists
	queuePath := ws.QueuePath(queueID)
	queueDir := filepath.Dir(queuePath)
	if err := os.MkdirAll(queueDir, 0o755); err != nil {
		return nil, fmt.Errorf("create queue directory: %w", err)
	}

	// Save new queue
	if err := ws.SaveTaskQueue(queue); err != nil {
		return nil, fmt.Errorf("save queue: %w", err)
	}

	return queue, nil
}

// extractTitle creates a title from a description.
// Takes the first ~50 characters, cutting at word boundary.
func extractTitle(description string) string {
	// Remove leading/trailing whitespace
	description = strings.TrimSpace(description)
	if description == "" {
		return "Untitled Task"
	}

	// If short enough, use as-is
	if len(description) <= 50 {
		return description
	}

	// Truncate to ~50 chars at word boundary
	maxLen := 50
	truncated := description[:maxLen]

	// Find last space and cut there
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// buildOptimizePrompt creates the AI prompt for task optimization.
func buildOptimizePrompt(task *storage.QueuedTask, notes []storage.QueueNote) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`You are optimizing a task based on accumulated notes and context.

Current timestamp: %s

## Current Task

Title: %s
Description: %s
Priority: %d
Labels: %s

`, currentTime, task.Title, task.Description, task.Priority, strings.Join(task.Labels, ", ")))

	if len(notes) > 0 {
		sb.WriteString("## Notes\n\n")
		for _, note := range notes {
			sb.WriteString(fmt.Sprintf("**[%s]** %s\n", note.Timestamp, note.Content))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`## Instructions

Review the current task and notes. Improve the task by:

1. **Clarify the title** - Make it more specific and actionable
2. **Enhance the description** - Incorporate context from notes
3. **Add relevant labels** - Suggest labels for categorization (bug, feature, docs, etc.)
4. **Identify requirements** - Add any missing requirements from notes
5. **Note improvements** - Briefly explain what changed and why

## Output Format

Respond with a YAML-like format between --- markers:

---
title: Improved Title Here
labels: label1, label2, label3
description: |
  Enhanced description that incorporates notes.
  Can be multiple lines.
notes: |
  - Added requirement X from note
  - Clarified Y based on context
---

Do not include any text outside the --- markers. Only output the YAML block.`)

	return sb.String()
}

// parseOptimizedTask parses the AI response into an optimized task.
//
//nolint:unparam // Error return reserved for future validation needs
func parseOptimizedTask(response string) (*struct {
	Title       string
	Description string
	Labels      []string
	Notes       []string
}, error,
) {
	result := struct {
		Title       string
		Description string
		Labels      []string
		Notes       []string
	}{}

	// Find YAML content between --- markers
	content := response
	if strings.Contains(content, "---") {
		parts := strings.Split(content, "---")
		if len(parts) >= 2 {
			content = strings.TrimSpace(parts[1])
		}
	}

	// Parse YAML-like content
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "title:") {
			result.Title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
		} else if strings.HasPrefix(line, "description:") {
			// Multi-line description - collect subsequent lines
			desc := strings.TrimPrefix(line, "description:")
			if strings.HasPrefix(desc, "|") {
				desc = ""
			} else {
				desc = strings.TrimSpace(desc)
			}
			result.Description = desc
		} else if strings.HasPrefix(line, "labels:") {
			labelsStr := strings.TrimSpace(strings.TrimPrefix(line, "labels:"))
			if labelsStr != "" {
				// Parse comma-separated labels
				for _, l := range strings.Split(labelsStr, ",") {
					l = strings.TrimSpace(l)
					if l != "" {
						result.Labels = append(result.Labels, l)
					}
				}
			}
		}
	}

	// Handle multi-line description
	if strings.Contains(response, "description: |") {
		descStart := strings.Index(response, "description: |")
		if descStart > 0 {
			afterDesc := response[descStart+14:] // Skip "description: |"
			// Find next field or end
			nextField := strings.Index(afterDesc, "\n---")
			if nextField > 0 {
				afterDesc = afterDesc[:nextField]
			} else if nextField = strings.Index(afterDesc, "\nlabels:"); nextField > 0 {
				afterDesc = afterDesc[:nextField]
			} else if nextField = strings.Index(afterDesc, "\nnotes:"); nextField > 0 {
				afterDesc = afterDesc[:nextField]
			}
			result.Description = strings.TrimSpace(afterDesc)
		}
	}

	return &result, nil
}

// buildTaskMarkdown creates a markdown file from a task and its notes.
func buildTaskMarkdown(task *storage.QueuedTask, notes []storage.QueueNote) string {
	var sb strings.Builder

	// Frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %s\n", task.Title))
	if len(task.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("labels: %s\n", strings.Join(task.Labels, ", ")))
	}
	if task.Priority > 0 {
		priority := "normal"
		if task.Priority <= 1 {
			priority = "high"
		} else if task.Priority >= 3 {
			priority = "low"
		}
		sb.WriteString(fmt.Sprintf("priority: %s\n", priority))
	}
	sb.WriteString("---\n\n")

	// Title and description
	sb.WriteString(fmt.Sprintf("# %s\n\n", task.Title))
	if task.Description != "" {
		sb.WriteString(task.Description)
		sb.WriteString("\n\n")
	}

	// Notes section
	if len(notes) > 0 {
		sb.WriteString("## Notes\n\n")
		for _, note := range notes {
			sb.WriteString(fmt.Sprintf("**[%s]** %s\n\n", note.Timestamp, note.Content))
		}
	}

	return sb.String()
}

// queueTitleFromID creates a readable title from a queue ID.
func queueTitleFromID(queueID string) string {
	switch queueID {
	case "quick-tasks":
		return "Quick Tasks"
	default:
		// Convert kebab-case to Title Case
		parts := strings.Split(queueID, "-")
		for i, p := range parts {
			if p != "" {
				// Capitalize first rune of each part
				parts[i] = string(unicode.ToUpper(rune(p[0]))) + p[1:]
			}
		}

		return strings.Join(parts, " ")
	}
}

// ParseQueueTaskRef parses a queue task reference like "quick-tasks/task-1".
func ParseQueueTaskRef(ref string) (string, string, error) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid queue task reference: %s (expected format: <queue-id>/<task-id>)", ref)
	}

	queueID := strings.TrimSpace(parts[0])
	taskID := strings.TrimSpace(parts[1])

	if queueID == "" || taskID == "" {
		return "", "", fmt.Errorf("invalid queue task reference: %s (queue-id and task-id cannot be empty)", ref)
	}

	return queueID, taskID, nil
}
