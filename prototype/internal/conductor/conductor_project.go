package conductor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/valksor/go-mehrhof/internal/export"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// ProjectPlanOptions configures the project planning operation.
type ProjectPlanOptions struct {
	Title              string // Project/queue title
	CustomInstructions string // Additional instructions for AI
	UseSchema          bool   // Use schema-driven extraction for parsing (fallback to regex if fails)
}

// ProjectPlanResult holds the result of creating a project plan.
type ProjectPlanResult struct {
	Queue         *storage.TaskQueue
	Tasks         []*storage.QueuedTask
	Questions     []string
	Blockers      []string
	RawOutputPath string // Path to saved raw AI response (plan.md)
}

// ResearchManifest holds metadata about a research source directory.
// Used for research: source type to provide file manifest without concatenation.
type ResearchManifest struct {
	BasePath    string              // Absolute path to research root
	FileCount   int                 // Total files found
	Structure   []DirEntry          // Directory tree structure
	EntryPoints []string            // Detected entry point files (absolute paths)
	ByCategory  map[string][]string // Files grouped by category
}

// DirEntry represents a file or directory in the research manifest.
type DirEntry struct {
	Path     string // Relative path from base
	Name     string // File/directory name
	Type     string // "file" or "dir"
	Size     int64  // File size in bytes (0 for dirs)
	Category string // "docs", "code", "config", "other"
}

// CreateProjectPlan creates a task breakdown from a source.
// The source can be:
//   - research:/path - Directory for AI to research (agent explores selectively)
//   - dir:/path - Directory of files to analyze (reads all content)
//   - file:/path - Single file to analyze
//   - github:123, jira:PROJ-123, etc. - Provider reference
func (c *Conductor) CreateProjectPlan(ctx context.Context, source string, opts ProjectPlanOptions) (*ProjectPlanResult, error) {
	c.publishProgress("Creating project plan...", 0)

	var prompt string

	// Handle research source type - uses manifest instead of concatenation
	if strings.HasPrefix(source, "research:") {
		dirPath := source[9:] // Strip "research:" prefix

		manifest, err := c.readResearchSource(dirPath)
		if err != nil {
			return nil, fmt.Errorf("read research source: %w", err)
		}

		prompt = buildResearchPlanningPrompt(opts.Title, manifest, opts.CustomInstructions)
		c.publishProgress("Research manifest prepared...", 20)
	} else {
		// Existing flow for dir:, file:, and providers
		sourceContent, err := c.readProjectSource(ctx, source)
		if err != nil {
			return nil, fmt.Errorf("read source: %w", err)
		}

		prompt = buildProjectPlanningPrompt(opts.Title, sourceContent, opts.CustomInstructions)
		c.publishProgress("Analyzing source content...", 20)
	}

	// Generate queue ID from title or source
	queueID := generateQueueID(opts.Title, source)

	// Get agent for project planning (use planning step)
	ag, err := c.GetAgentForStep(ctx, "planning")
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}

	if c.opts.Verbose {
		slog.Debug("executing project plan prompt",
			"prompt_length", len(prompt),
			"agent", ag.Name(),
		)
	}

	// Execute the planning prompt
	resp, err := ag.Run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("execute planning: %w", err)
	}
	// Extract response content from Summary or Messages
	response := resp.Summary
	if response == "" && len(resp.Messages) > 0 {
		response = strings.Join(resp.Messages, "\n")
	}

	if c.opts.Verbose {
		preview := response
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		slog.Debug("received agent response",
			"response_length", len(response),
			"response_preview", preview,
		)
	}

	// Save raw AI response for debugging (before parsing)
	if err := c.workspace.SavePlanOutput(queueID, response); err != nil {
		slog.Warn("failed to save raw plan output", "error", err)
	}

	c.publishProgress("Parsing task breakdown...", 60)

	// Parse the AI response into tasks
	var parsed *export.ParsedPlan
	if opts.UseSchema {
		// Use schema-driven extraction with agent fallback to regex
		parsed = export.ParseProjectPlanWithSchema(ctx, response, ag)
	} else {
		// Use regex-based parsing only
		parsed = export.ParseProjectPlanWithSchema(ctx, response, nil)
	}

	// Create the task queue
	queue := storage.NewTaskQueue(queueID, opts.Title, source)
	for _, task := range parsed.Tasks {
		queue.AddTask(task)
	}
	queue.Questions = parsed.Questions
	queue.Blockers = parsed.Blockers

	// Compute relationships
	queue.ComputeBlocksRelations()
	queue.ComputeSubtaskRelations()
	queue.ComputeTaskStatuses()

	c.publishProgress("Saving queue...", 80)

	// Save the queue
	if err := c.workspace.SaveTaskQueue(queue); err != nil {
		return nil, fmt.Errorf("save queue: %w", err)
	}

	c.publishProgress("Project plan created", 100)

	return &ProjectPlanResult{
		Queue:         queue,
		Tasks:         queue.Tasks,
		Questions:     queue.Questions,
		Blockers:      queue.Blockers,
		RawOutputPath: c.workspace.PlanOutputPath(queue.ID),
	}, nil
}

// SubmitOptions configures task submission to a provider.
type SubmitOptions struct {
	Provider   string   // Provider name (e.g., "wrike", "github", "jira")
	CreateEpic bool     // Create an epic/project to group tasks
	Labels     []string // Labels to apply to all tasks
	DryRun     bool     // Preview only, don't actually submit
	TaskIDs    []string // Optional: submit only these task IDs
	Comment    string   // Optional: add comment when tasks already submitted
	Mention    string   // Optional: mention/notification to add to all submitted tasks
}

// SubmitResult holds the result of submitting tasks to a provider.
type SubmitResult struct {
	Epic   *SubmittedItem   // The created epic (if CreateEpic was true)
	Tasks  []*SubmittedTask // Individual submitted tasks
	DryRun bool             // Whether this was a dry run
}

// SubmittedItem represents a submitted epic or project.
type SubmittedItem struct {
	ExternalID  string
	ExternalURL string
	Title       string
}

// SubmittedTask represents a submitted task with its external reference.
type SubmittedTask struct {
	LocalID     string // Local task ID (task-1, task-2, etc.)
	ExternalID  string // Provider's task ID
	ExternalURL string // Provider's task URL
	Title       string
}

// AutoReorderResult holds the result of AI-based task reordering.
type AutoReorderResult struct {
	OldOrder  []string // Previous task order (task IDs)
	NewOrder  []string // New optimized task order (task IDs)
	Reasoning string   // AI's explanation of the reordering
}

// AutoReorderTasks uses AI to suggest an optimal task order based on dependencies and priorities.
func (c *Conductor) AutoReorderTasks(ctx context.Context, queueID string) (*AutoReorderResult, error) {
	// Load the queue
	queue, err := storage.LoadTaskQueue(c.workspace, queueID)
	if err != nil {
		return nil, fmt.Errorf("load queue: %w", err)
	}

	if len(queue.Tasks) == 0 {
		return nil, errors.New("no tasks to reorder")
	}

	// Capture old order
	oldOrder := make([]string, len(queue.Tasks))
	for i, task := range queue.Tasks {
		oldOrder[i] = task.ID
	}

	// Build the prompt with task information
	prompt := buildReorderingPrompt(queue)

	// Get the planning agent
	ag, err := c.GetAgentForStep(ctx, "planning")
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}

	// Call the AI
	resp, err := ag.Run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI reordering failed: %w", err)
	}

	// Parse the response to get the new order
	newOrder, reasoning, err := export.ParseTaskOrder(resp.Summary)
	if err != nil {
		return nil, fmt.Errorf("parse AI response: %w", err)
	}

	// Validate the order contains all tasks
	if len(newOrder) != len(queue.Tasks) {
		return nil, fmt.Errorf("AI returned %d tasks, expected %d", len(newOrder), len(queue.Tasks))
	}

	// Apply the reordering
	for targetIdx, taskID := range newOrder {
		if err := queue.ReorderTask(taskID, targetIdx); err != nil {
			return nil, fmt.Errorf("reorder task %s: %w", taskID, err)
		}
	}

	// Recompute relationships and statuses
	queue.ComputeBlocksRelations()
	queue.ComputeTaskStatuses()

	// Save the queue
	if err := queue.Save(); err != nil {
		return nil, fmt.Errorf("save queue: %w", err)
	}

	return &AutoReorderResult{
		OldOrder:  oldOrder,
		NewOrder:  newOrder,
		Reasoning: reasoning,
	}, nil
}

// buildReorderingPrompt creates the AI prompt for task reordering.
func buildReorderingPrompt(queue *storage.TaskQueue) string {
	var sb strings.Builder

	sb.WriteString("You are an expert project manager helping to optimize task execution order.\n\n")
	sb.WriteString("## Project: " + queue.Title + "\n\n")
	sb.WriteString("## Current Tasks\n\n")

	for i, task := range queue.Tasks {
		sb.WriteString(fmt.Sprintf("### %d. %s: %s\n", i+1, task.ID, task.Title))
		sb.WriteString(fmt.Sprintf("- **Priority**: %d\n", task.Priority))
		sb.WriteString(fmt.Sprintf("- **Status**: %s\n", task.Status))
		if len(task.DependsOn) > 0 {
			sb.WriteString(fmt.Sprintf("- **Depends on**: %s\n", strings.Join(task.DependsOn, ", ")))
		}
		if len(task.Blocks) > 0 {
			sb.WriteString(fmt.Sprintf("- **Blocks**: %s\n", strings.Join(task.Blocks, ", ")))
		}
		if task.Description != "" {
			sb.WriteString(fmt.Sprintf("- **Description**: %s\n", task.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`## Instructions

Analyze the tasks and suggest an optimal execution order. Consider:
1. **Dependencies**: Tasks that are depended upon should come first
2. **Priorities**: Higher priority tasks (lower numbers) should be preferred
3. **Blocking relationships**: Tasks that block many others should be done early
4. **Logical grouping**: Related tasks should be near each other when possible

## Output Format

Provide your response in this exact format:

## Recommended Order

1. task-X - Brief reason
2. task-Y - Brief reason
...

## Reasoning

A 2-3 sentence explanation of your reordering strategy.
`)

	return sb.String()
}

// StartNextTask begins implementing the next ready task from the queue.
func (c *Conductor) StartNextTask(ctx context.Context, queueID string) (*storage.QueuedTask, error) {
	// Load the queue
	queue, err := storage.LoadTaskQueue(c.workspace, queueID)
	if err != nil {
		return nil, fmt.Errorf("load queue: %w", err)
	}

	// Find the next ready task
	readyTasks := queue.GetReadyTasks()
	if len(readyTasks) == 0 {
		return nil, errors.New("no ready tasks in queue")
	}

	// Get the highest priority ready task
	var nextTask *storage.QueuedTask
	for _, task := range readyTasks {
		if nextTask == nil || task.Priority < nextTask.Priority {
			nextTask = task
		}
	}

	// Start the task using the existing workflow
	reference := nextTask.ExternalURL
	if reference == "" {
		// Use local task reference if not submitted to provider
		reference = fmt.Sprintf("queue:%s/%s", queueID, nextTask.ID)
	}

	if err := c.Start(ctx, reference); err != nil {
		return nil, fmt.Errorf("start task: %w", err)
	}

	return nextTask, nil
}

// ProjectAutoOptions configures the full project automation run.
type ProjectAutoOptions struct {
	ProjectPlanOptions
	SubmitOptions
	AutoOptions
}

// ProjectAutoResult holds the result of a full project auto run.
type ProjectAutoResult struct {
	Queue          *storage.TaskQueue
	TasksPlanned   int
	TasksSubmitted int
	TasksCompleted int
	Error          error
	FailedAt       string
}

// RunProjectAuto executes the full project automation cycle.
func (c *Conductor) RunProjectAuto(ctx context.Context, source string, opts ProjectAutoOptions) (*ProjectAutoResult, error) {
	result := &ProjectAutoResult{}

	// Step 1: Create project plan
	c.publishProgress("Creating project plan...", 5)
	planResult, err := c.CreateProjectPlan(ctx, source, opts.ProjectPlanOptions)
	if err != nil {
		result.Error = err
		result.FailedAt = "plan"

		return result, fmt.Errorf("create plan: %w", err)
	}
	result.Queue = planResult.Queue
	result.TasksPlanned = len(planResult.Tasks)
	c.publishProgress("Project plan created", 20)

	// Step 2: Submit tasks to provider (if provider specified)
	if opts.Provider != "" {
		c.publishProgress("Submitting tasks...", 25)
		submitResult, err := c.SubmitProjectTasks(ctx, planResult.Queue.ID, opts.SubmitOptions)
		if err != nil {
			result.Error = err
			result.FailedAt = "submit"

			return result, fmt.Errorf("submit tasks: %w", err)
		}
		result.TasksSubmitted = len(submitResult.Tasks)
		c.publishProgress("Tasks submitted", 40)
	}

	// Step 3: Implement each task in order
	tasksCompleted := 0
	totalTasks := len(planResult.Tasks)

	for i := range totalTasks {
		progress := 40 + int((float64(i)/float64(totalTasks))*60)
		c.publishProgress(fmt.Sprintf("Implementing task %d/%d...", i+1, totalTasks), progress)

		nextTask, err := c.StartNextTask(ctx, planResult.Queue.ID)
		if err != nil {
			// No more ready tasks - might be blocked or all done
			break
		}

		// Run the full auto workflow for this task
		autoResult, err := c.RunAuto(ctx, "", opts.AutoOptions)
		if err != nil {
			result.Error = err
			result.FailedAt = "implement-" + nextTask.ID

			return result, fmt.Errorf("implement task %s: %w", nextTask.ID, err)
		}

		if autoResult.FinishDone {
			tasksCompleted++
			// Update task status
			_ = planResult.Queue.UpdateTask(nextTask.ID, func(t *storage.QueuedTask) {
				t.Status = storage.TaskStatusSubmitted // Mark as done
			})
			// Recompute statuses for blocked tasks
			planResult.Queue.ComputeTaskStatuses()
			_ = planResult.Queue.Save()
		}
	}

	result.TasksCompleted = tasksCompleted
	c.publishProgress("Project automation complete", 100)

	return result, nil //nolint:nilerr // StartNextTask error means no more ready tasks, which is success
}

// SyncProjectOptions configures project sync from provider.
type SyncProjectOptions struct {
	IncludeStatus    []string // Filter by status (empty = smart default)
	MaxDepth         int      // Max depth for recursive fetch (0 = unlimited)
	PreserveExternal bool     // Keep external IDs/URLs from provider
}

// SyncProjectResult holds the result of syncing a project.
type SyncProjectResult struct {
	Queue     *storage.TaskQueue
	TasksSync int
	TasksNew  int
	Source    string
	URL       string
}
