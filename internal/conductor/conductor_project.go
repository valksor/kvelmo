package conductor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/export"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// ProjectPlanOptions configures the project planning operation.
type ProjectPlanOptions struct {
	Title              string // Project/queue title
	CustomInstructions string // Additional instructions for AI
}

// ProjectPlanResult holds the result of creating a project plan.
type ProjectPlanResult struct {
	Queue     *storage.TaskQueue
	Tasks     []*storage.QueuedTask
	Questions []string
	Blockers  []string
}

// CreateProjectPlan creates a task breakdown from a source.
// The source can be a directory path (dir:/path), file path (file:/path),
// or provider reference (github:123, jira:PROJ-123, etc.).
func (c *Conductor) CreateProjectPlan(ctx context.Context, source string, opts ProjectPlanOptions) (*ProjectPlanResult, error) {
	c.publishProgress("Creating project plan...", 0)

	// Read source content
	sourceContent, err := c.readProjectSource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}

	// Generate queue ID from title or source
	queueID := generateQueueID(opts.Title, source)

	// Build the planning prompt
	prompt := buildProjectPlanningPrompt(opts.Title, sourceContent, opts.CustomInstructions)

	c.publishProgress("Analyzing source content...", 20)

	// Get agent for project planning (use planning step)
	ag, err := c.GetAgentForStep(ctx, "planning")
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
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

	c.publishProgress("Parsing task breakdown...", 60)

	// Parse the AI response into tasks
	parsed := export.ParseProjectPlan(response)

	// Create the task queue
	queue := storage.NewTaskQueue(queueID, opts.Title, source)
	for _, task := range parsed.Tasks {
		queue.AddTask(task)
	}
	queue.Questions = parsed.Questions
	queue.Blockers = parsed.Blockers

	// Compute dependency relationships
	queue.ComputeBlocksRelations()
	queue.ComputeTaskStatuses()

	c.publishProgress("Saving queue...", 80)

	// Save the queue
	if err := c.workspace.SaveTaskQueue(queue); err != nil {
		return nil, fmt.Errorf("save queue: %w", err)
	}

	c.publishProgress("Project plan created", 100)

	return &ProjectPlanResult{
		Queue:     queue,
		Tasks:     queue.Tasks,
		Questions: queue.Questions,
		Blockers:  queue.Blockers,
	}, nil
}

// SubmitOptions configures task submission to a provider.
type SubmitOptions struct {
	Provider   string   // Provider name (e.g., "wrike", "github", "jira")
	CreateEpic bool     // Create an epic/project to group tasks
	Labels     []string // Labels to apply to all tasks
	DryRun     bool     // Preview only, don't actually submit
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

// SubmitProjectTasks submits tasks from a queue to an external provider.
func (c *Conductor) SubmitProjectTasks(ctx context.Context, queueID string, opts SubmitOptions) (*SubmitResult, error) {
	// Load the queue
	queue, err := storage.LoadTaskQueue(c.workspace, queueID)
	if err != nil {
		return nil, fmt.Errorf("load queue: %w", err)
	}

	if opts.DryRun {
		return c.dryRunSubmit(queue, opts)
	}

	// Verify provider exists
	_, _, ok := c.providers.Get(opts.Provider)
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", opts.Provider)
	}

	result := &SubmitResult{
		Tasks:  make([]*SubmittedTask, 0, len(queue.Tasks)),
		DryRun: false,
	}

	c.publishProgress("Submitting tasks to "+opts.Provider+"...", 0)

	// Track mapping from local task IDs to external IDs for dependency resolution
	localToExternal := make(map[string]string)

	// First pass: submit all tasks (without dependencies)
	for i, task := range queue.Tasks {
		if task.Status == storage.TaskStatusSubmitted {
			// Already submitted - add to mapping
			if task.ExternalID != "" {
				localToExternal[task.ID] = task.ExternalID
			}

			continue
		}

		progress := int((float64(i+1) / float64(len(queue.Tasks))) * 50) // First pass is 50%
		c.publishProgress(fmt.Sprintf("Creating task %d/%d...", i+1, len(queue.Tasks)), progress)

		// Create work unit in provider (without dependencies for now)
		workUnit, err := c.submitTaskToProvider(ctx, task, opts, nil)
		if err != nil {
			return result, fmt.Errorf("submit task %s: %w", task.ID, err)
		}

		// Track mapping
		localToExternal[task.ID] = workUnit.ID

		// Update local task with external references
		if err := queue.UpdateTask(task.ID, func(t *storage.QueuedTask) {
			t.ExternalID = workUnit.ID
			t.ExternalURL = workUnit.URL
			t.Status = storage.TaskStatusSubmitted
		}); err != nil {
			c.logError(fmt.Errorf("update task %s: %w", task.ID, err))
		}

		result.Tasks = append(result.Tasks, &SubmittedTask{
			LocalID:     task.ID,
			ExternalID:  workUnit.ID,
			ExternalURL: workUnit.URL,
			Title:       task.Title,
		})
	}

	// Second pass: create dependencies using the external IDs
	c.publishProgress("Creating dependencies...", 55)
	for i, task := range queue.Tasks {
		if len(task.DependsOn) == 0 {
			continue
		}

		progress := 55 + int((float64(i+1)/float64(len(queue.Tasks)))*40)
		c.publishProgress(fmt.Sprintf("Creating dependencies for task %d/%d...", i+1, len(queue.Tasks)), progress)

		// Convert local dependency IDs to external IDs
		externalDeps := make([]string, 0, len(task.DependsOn))
		for _, localDep := range task.DependsOn {
			if extID, ok := localToExternal[localDep]; ok {
				externalDeps = append(externalDeps, extID)
			}
		}

		if len(externalDeps) > 0 {
			// Create dependencies in the provider
			if err := c.createProviderDependencies(ctx, opts.Provider, localToExternal[task.ID], externalDeps); err != nil {
				c.logError(fmt.Errorf("create dependencies for %s: %w", task.ID, err))
			}
		}
	}

	// Update queue status
	queue.Status = storage.QueueStatusSubmitted

	// Save the updated queue
	if err := queue.Save(); err != nil {
		c.logError(fmt.Errorf("save queue: %w", err))
	}

	c.publishProgress("Tasks submitted", 100)

	return result, nil
}

// submitTaskToProvider creates a work unit in the provider for the given task.
// For now, returns a stub implementation - actual provider integration will use
// the provider.WorkUnitCreator interface.
func (c *Conductor) submitTaskToProvider(_ context.Context, task *storage.QueuedTask, opts SubmitOptions, _ []string) (*submittedWorkUnit, error) {
	// Validate required fields
	if task == nil || task.ID == "" {
		return nil, errors.New("task is required")
	}
	if opts.Provider == "" {
		return nil, errors.New("provider is required")
	}

	// Build description with labels and any metadata
	description := task.Description
	if len(task.Labels) > 0 {
		description += "\n\n**Labels:** " + strings.Join(task.Labels, ", ")
	}

	// For now, return a stub implementation
	// Full provider integration would:
	// 1. Get provider factory and create instance
	// 2. Check for WorkUnitCreator interface
	// 3. Create work unit with proper options
	// 4. Handle dependencies via provider-specific APIs (e.g., Wrike's CreateDependency)
	_ = description

	return &submittedWorkUnit{
		ID:    "ext-" + task.ID,
		URL:   fmt.Sprintf("https://%s.example.com/%s", opts.Provider, task.ID),
		Title: task.Title,
	}, nil
}

// createProviderDependencies creates dependency relationships in the provider.
// Provider-specific implementations:
// - Wrike: Uses CreateDependency API (predecessor -> successor)
// - GitHub: Dependencies via task list in epic body
// - Jira: Issue links (blocks/is-blocked-by).
func (c *Conductor) createProviderDependencies(ctx context.Context, providerName, taskID string, predecessorIDs []string) error {
	// This is a stub - actual implementation would:
	// 1. Get provider instance
	// 2. Check for dependency support interface
	// 3. Call provider-specific dependency creation API
	return nil
}

type submittedWorkUnit struct {
	ID    string
	URL   string
	Title string
}

// dryRunSubmit simulates submission and returns what would be created.
func (c *Conductor) dryRunSubmit(queue *storage.TaskQueue, _ SubmitOptions) (*SubmitResult, error) {
	result := &SubmitResult{
		Tasks:  make([]*SubmittedTask, 0, len(queue.Tasks)),
		DryRun: true,
	}

	for _, task := range queue.Tasks {
		if task.Status == storage.TaskStatusSubmitted {
			continue
		}

		result.Tasks = append(result.Tasks, &SubmittedTask{
			LocalID:     task.ID,
			ExternalID:  "[dry-run]",
			ExternalURL: "[dry-run]",
			Title:       task.Title,
		})
	}

	return result, nil
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
			// Update task in queue as completed
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

// readProjectSource reads content from various source types.
func (c *Conductor) readProjectSource(ctx context.Context, source string) (string, error) {
	// Parse source type
	if strings.HasPrefix(source, "dir:") {
		return c.readDirectorySource(source[4:])
	}
	if strings.HasPrefix(source, "file:") {
		return c.readFileSource(source[5:])
	}
	// Provider reference (github:123, jira:PROJ-123, etc.)
	return c.readProviderSource(ctx, source)
}

// readDirectorySource reads all relevant files from a directory.
func (c *Conductor) readDirectorySource(dirPath string) (string, error) {
	var content strings.Builder

	// Walk the directory and collect relevant files
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Read text files (markdown, txt, yaml, json, etc.)
		ext := strings.ToLower(filepath.Ext(path))
		textExts := map[string]bool{
			".md": true, ".txt": true, ".yaml": true, ".yml": true,
			".json": true, ".xml": true, ".html": true, ".css": true,
			".js": true, ".ts": true, ".go": true, ".py": true,
			".java": true, ".rs": true, ".rb": true, ".sh": true,
		}

		if !textExts[ext] {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil //nolint:nilerr // Skip unreadable files intentionally
		}

		relPath, _ := filepath.Rel(dirPath, path)
		content.WriteString(fmt.Sprintf("\n--- %s ---\n", relPath))
		content.Write(data)
		content.WriteString("\n")

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk directory: %w", err)
	}

	return content.String(), nil
}

// readFileSource reads content from a single file.
func (c *Conductor) readFileSource(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return string(data), nil
}

// readProviderSource fetches content from a provider reference.
func (c *Conductor) readProviderSource(ctx context.Context, reference string) (string, error) {
	// Parse provider:id format
	parts := strings.SplitN(reference, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid reference format: %s (expected provider:id)", reference)
	}

	providerName := parts[0]
	taskID := parts[1]

	// Get the provider factory from registry
	_, factory, ok := c.providers.Get(providerName)
	if !ok {
		return "", fmt.Errorf("provider not found: %s", providerName)
	}

	// Create provider instance
	providerCfg := provider.NewConfig()
	instance, err := factory(ctx, providerCfg)
	if err != nil {
		return "", fmt.Errorf("create provider: %w", err)
	}

	// Check if provider implements Reader interface
	reader, ok := instance.(provider.Reader)
	if !ok {
		return "", fmt.Errorf("provider %s does not support fetching work units", providerName)
	}

	// Fetch the work unit
	workUnit, err := reader.Fetch(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch from %s: %w", providerName, err)
	}

	// Format as planning input
	return formatWorkUnitAsSource(workUnit), nil
}

// formatWorkUnitAsSource converts a WorkUnit to a markdown-formatted planning source.
func formatWorkUnitAsSource(wu *provider.WorkUnit) string {
	var sb strings.Builder

	sb.WriteString("# " + wu.Title + "\n\n")

	if wu.Description != "" {
		sb.WriteString(wu.Description + "\n\n")
	}

	if len(wu.Labels) > 0 {
		sb.WriteString("**Labels:** " + strings.Join(wu.Labels, ", ") + "\n")
	}

	if wu.Priority != 0 {
		sb.WriteString("**Priority:** " + wu.Priority.String() + "\n")
	}

	if wu.Status != "" {
		sb.WriteString("**Status:** " + string(wu.Status) + "\n")
	}

	if len(wu.Assignees) > 0 {
		var assigneeNames []string
		for _, a := range wu.Assignees {
			if a.Name != "" {
				assigneeNames = append(assigneeNames, a.Name)
			} else if a.ID != "" {
				assigneeNames = append(assigneeNames, a.ID)
			}
		}
		if len(assigneeNames) > 0 {
			sb.WriteString("**Assignees:** " + strings.Join(assigneeNames, ", ") + "\n")
		}
	}

	return sb.String()
}

// generateQueueID creates a queue ID from title or source.
func generateQueueID(title, source string) string {
	// Use title if provided
	base := title
	if base == "" {
		// Extract from source
		if strings.HasPrefix(source, "dir:") {
			base = filepath.Base(source[4:])
		} else if strings.HasPrefix(source, "file:") {
			base = strings.TrimSuffix(filepath.Base(source[5:]), filepath.Ext(source[5:]))
		} else {
			base = strings.ReplaceAll(source, ":", "-")
		}
	}

	// Normalize: lowercase, replace spaces with dashes, remove special chars
	id := strings.ToLower(base)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}

		return -1
	}, id)

	// Add timestamp suffix for uniqueness
	timestamp := time.Now().Format("20060102-150405")

	return fmt.Sprintf("%s-%s", id, timestamp)
}

// buildProjectPlanningPrompt creates the prompt for project task breakdown.
func buildProjectPlanningPrompt(title, sourceContent, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	prompt := fmt.Sprintf(`You are an expert project manager and software architect. Your task is to analyze the provided content and create a structured task breakdown.

Current timestamp: %s

## Project
%s

## Source Content
%s

`, currentTime, title, sourceContent)

	if customInstructions != "" {
		prompt += fmt.Sprintf(`## Custom Instructions
%s

`, customInstructions)
	}

	prompt += `## Output Format

Create a structured task breakdown in the following format:

## Tasks

For each task, use this format:

### task-N: Task Title
- **Priority**: N (1 = highest)
- **Status**: ready OR blocked
- **Labels**: comma, separated, labels
- **Depends on**: task-X, task-Y (if blocked)
- **Description**: Detailed description of what needs to be done

## Questions
List any questions that need to be resolved before implementation:
1. Question one?
2. Question two?

## Blockers
List any blockers that prevent progress:
- Blocker description

## Guidelines

1. Break down the work into atomic, implementable tasks
2. Each task should be completable in 1-4 hours of work
3. Identify dependencies clearly - if task B needs task A, mark B as blocked
4. Tasks without dependencies should be marked as "ready"
5. Prioritize tasks: core functionality first, then enhancements
6. Include labels for categorization (e.g., backend, frontend, testing, docs)
7. If requirements are unclear, add specific questions
8. If there are external blockers, list them

Do not include any other text or explanation. Only output the structured task breakdown.
`

	return prompt
}
