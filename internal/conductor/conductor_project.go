package conductor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// SubmitProjectTasks submits tasks from a queue to an external provider.
func (c *Conductor) SubmitProjectTasks(ctx context.Context, queueID string, opts SubmitOptions) (*SubmitResult, error) {
	// Load the queue
	queue, err := storage.LoadTaskQueue(c.workspace, queueID)
	if err != nil {
		return nil, fmt.Errorf("load queue: %w", err)
	}

	tasksToSubmit, err := selectSubmitTasks(queue, opts.TaskIDs)
	if err != nil {
		return nil, err
	}
	if len(tasksToSubmit) == 0 {
		return nil, errors.New("no tasks selected for submission")
	}

	if err := validateSubmitSelection(queue, tasksToSubmit, opts); err != nil {
		return nil, err
	}

	if opts.DryRun {
		return c.dryRunSubmit(tasksToSubmit), nil
	}

	// Verify provider exists
	_, factory, ok := c.providers.Get(opts.Provider)
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", opts.Provider)
	}

	workspaceCfg, _ := c.workspace.LoadConfig()
	providerCfg := buildProviderConfig(workspaceCfg, opts.Provider)
	providerInst, err := factory(ctx, providerCfg)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}

	creator, _ := providerInst.(provider.WorkUnitCreator)
	commenter, _ := providerInst.(provider.Commenter)

	commentText := strings.TrimSpace(opts.Comment)
	if commentText != "" && commenter == nil {
		return nil, fmt.Errorf("provider %s does not support comments", opts.Provider)
	}

	var newTasks []*storage.QueuedTask
	for _, task := range tasksToSubmit {
		if task.Status == storage.TaskStatusSubmitted || task.ExternalID != "" {
			continue
		}
		newTasks = append(newTasks, task)
	}
	if len(newTasks) > 0 && creator == nil {
		return nil, fmt.Errorf("provider %s does not support task creation", opts.Provider)
	}

	// Sort tasks topologically so parents are submitted before subtasks
	sortedTasks, err := topologicalSortWithParents(newTasks)
	if err != nil {
		return nil, fmt.Errorf("sort tasks: %w", err)
	}

	result := &SubmitResult{
		Tasks:  make([]*SubmittedTask, 0, len(tasksToSubmit)),
		DryRun: false,
	}

	c.publishProgress("Submitting tasks to "+opts.Provider+"...", 0)

	// Track mapping from local task IDs to external IDs for dependency and parent resolution
	localToExternal := make(map[string]string)
	for _, task := range queue.Tasks {
		if task.Status == storage.TaskStatusSubmitted && task.ExternalID != "" {
			localToExternal[task.ID] = task.ExternalID
		}
	}

	// First pass: submit all tasks with parent IDs (topologically sorted)
	for i, task := range sortedTasks {
		progress := int((float64(i+1) / float64(len(sortedTasks))) * 50) // First pass is 50%
		c.publishProgress(fmt.Sprintf("Creating task %d/%d...", i+1, len(sortedTasks)), progress)

		// Resolve parent external ID if this is a subtask
		var parentExternalID string
		if task.ParentID != "" {
			if extID, ok := localToExternal[task.ParentID]; ok {
				parentExternalID = extID
			}
		}

		// Create work unit in provider with parent ID
		workUnit, err := c.submitTaskToProvider(ctx, creator, task, opts, parentExternalID)
		if err != nil {
			return result, fmt.Errorf("submit task %s: %w", task.ID, err)
		}

		// Track mapping
		localToExternal[task.ID] = workUnit.ExternalID

		// Update local task with external references
		if err := queue.UpdateTask(task.ID, func(t *storage.QueuedTask) {
			t.ExternalID = workUnit.ExternalID
			t.ExternalURL = workUnit.URL
			t.Status = storage.TaskStatusSubmitted
		}); err != nil {
			c.logError(fmt.Errorf("update task %s: %w", task.ID, err))
		}

		result.Tasks = append(result.Tasks, &SubmittedTask{
			LocalID:     task.ID,
			ExternalID:  workUnit.ExternalID,
			ExternalURL: workUnit.URL,
			Title:       task.Title,
		})
	}

	// Second pass: create dependencies using the external IDs
	c.publishProgress("Creating dependencies...", 55)
	for i, task := range sortedTasks {
		if len(task.DependsOn) == 0 {
			continue
		}

		progress := 55 + int((float64(i+1)/float64(len(sortedTasks)))*40)
		c.publishProgress(fmt.Sprintf("Creating dependencies for task %d/%d...", i+1, len(sortedTasks)), progress)

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

	if commentText != "" {
		for _, task := range tasksToSubmit {
			if task.ExternalID == "" {
				return nil, fmt.Errorf("task %s has no external ID for commenting", task.ID)
			}
			if _, err := commenter.AddComment(ctx, task.ExternalID, commentText); err != nil {
				return nil, fmt.Errorf("comment task %s: %w", task.ID, err)
			}
			found := false
			for _, submitted := range result.Tasks {
				if submitted.LocalID == task.ID {
					found = true

					break
				}
			}
			if !found {
				result.Tasks = append(result.Tasks, &SubmittedTask{
					LocalID:     task.ID,
					ExternalID:  task.ExternalID,
					ExternalURL: task.ExternalURL,
					Title:       task.Title,
				})
			}
		}
	}

	// Add mentions as comments to all submitted tasks (if provider supports it and mention is specified)
	if opts.Mention != "" && commenter != nil {
		for _, submitted := range result.Tasks {
			if _, err := commenter.AddComment(ctx, submitted.ExternalID, opts.Mention); err != nil {
				c.logError(fmt.Errorf("add mention to task %s: %w", submitted.LocalID, err))
			}
		}
	}

	// Update queue status
	allSubmitted := true
	for _, task := range queue.Tasks {
		if task.Status != storage.TaskStatusSubmitted {
			allSubmitted = false

			break
		}
	}
	if allSubmitted {
		queue.Status = storage.QueueStatusSubmitted
	}

	// Save the updated queue
	if err := queue.Save(); err != nil {
		c.logError(fmt.Errorf("save queue: %w", err))
	}

	c.publishProgress("Tasks submitted", 100)

	return result, nil
}

func selectSubmitTasks(queue *storage.TaskQueue, taskIDs []string) ([]*storage.QueuedTask, error) {
	if len(taskIDs) == 0 {
		return queue.Tasks, nil
	}

	targets := make(map[string]bool, len(taskIDs))
	for _, id := range taskIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		targets[id] = true
	}
	if len(targets) == 0 {
		return nil, errors.New("no valid task IDs provided")
	}

	var selected []*storage.QueuedTask
	for _, task := range queue.Tasks {
		if targets[task.ID] {
			selected = append(selected, task)
			delete(targets, task.ID)
		}
	}

	if len(targets) > 0 {
		missing := make([]string, 0, len(targets))
		for id := range targets {
			missing = append(missing, id)
		}

		return nil, fmt.Errorf("tasks not found in queue: %s", strings.Join(missing, ", "))
	}

	return selected, nil
}

func validateSubmitSelection(queue *storage.TaskQueue, selected []*storage.QueuedTask, opts SubmitOptions) error {
	if len(opts.TaskIDs) == 0 {
		return nil
	}

	var alreadySubmitted []string
	submitted := make(map[string]bool)
	for _, task := range queue.Tasks {
		if task.Status == storage.TaskStatusSubmitted || task.ExternalID != "" {
			submitted[task.ID] = true
		}
	}

	selectedSet := make(map[string]bool, len(selected))
	for _, task := range selected {
		selectedSet[task.ID] = true
		if submitted[task.ID] {
			alreadySubmitted = append(alreadySubmitted, task.ID)
		}
	}
	if len(alreadySubmitted) > 0 {
		if strings.TrimSpace(opts.Comment) == "" {
			return fmt.Errorf("task(s) already submitted: %s", strings.Join(alreadySubmitted, ", "))
		}
	}

	var missingDeps []string
	for _, task := range selected {
		for _, dep := range task.DependsOn {
			if selectedSet[dep] || submitted[dep] {
				continue
			}
			missingDeps = append(missingDeps, fmt.Sprintf("%s -> %s", task.ID, dep))
		}
	}
	if len(missingDeps) > 0 {
		return fmt.Errorf("missing dependencies (submit them first or include with --task): %s", strings.Join(missingDeps, ", "))
	}

	// Validate parent references - parents must be in selection or already submitted
	var missingParents []string
	for _, task := range selected {
		if task.ParentID == "" {
			continue
		}
		if selectedSet[task.ParentID] || submitted[task.ParentID] {
			continue
		}
		missingParents = append(missingParents, fmt.Sprintf("%s -> parent:%s", task.ID, task.ParentID))
	}
	if len(missingParents) > 0 {
		return fmt.Errorf("missing parents (submit them first or include with --task): %s", strings.Join(missingParents, ", "))
	}

	return nil
}

// topologicalSortWithParents sorts tasks so that parents come before subtasks and
// dependencies come before dependents. Returns error if a cycle is detected.
func topologicalSortWithParents(tasks []*storage.QueuedTask) ([]*storage.QueuedTask, error) {
	// Build adjacency list: task -> tasks that depend on it (must come after)
	// An edge from A to B means A must be submitted before B
	taskMap := make(map[string]*storage.QueuedTask)
	inDegree := make(map[string]int)
	edges := make(map[string][]string)

	for _, task := range tasks {
		taskMap[task.ID] = task
		inDegree[task.ID] = 0
		edges[task.ID] = nil
	}

	// Add edges for ParentID (parent must come before subtask)
	for _, task := range tasks {
		if task.ParentID != "" {
			if _, exists := taskMap[task.ParentID]; exists {
				edges[task.ParentID] = append(edges[task.ParentID], task.ID)
				inDegree[task.ID]++
			}
		}
	}

	// Add edges for DependsOn (dependency must come before dependent)
	for _, task := range tasks {
		for _, depID := range task.DependsOn {
			if _, exists := taskMap[depID]; exists {
				edges[depID] = append(edges[depID], task.ID)
				inDegree[task.ID]++
			}
		}
	}

	// Kahn's algorithm for topological sort
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	// Sort initial queue by priority for consistent ordering (lower priority = first)
	sort.Slice(queue, func(i, j int) bool {
		return taskMap[queue[i]].Priority < taskMap[queue[j]].Priority
	})

	var result []*storage.QueuedTask
	for len(queue) > 0 {
		// Pop first task
		taskID := queue[0]
		queue = queue[1:]
		result = append(result, taskMap[taskID])

		// Reduce in-degree for all dependent tasks
		for _, nextID := range edges[taskID] {
			inDegree[nextID]--
			if inDegree[nextID] == 0 {
				queue = append(queue, nextID)
			}
		}

		// Re-sort queue by priority for consistent ordering
		sort.Slice(queue, func(i, j int) bool {
			return taskMap[queue[i]].Priority < taskMap[queue[j]].Priority
		})
	}

	// Check for cycles
	if len(result) != len(tasks) {
		return nil, errors.New("circular dependency detected in task graph")
	}

	return result, nil
}

func mergeLabels(taskLabels, submitLabels []string) []string {
	seen := make(map[string]bool)
	var labels []string

	for _, label := range append(taskLabels, submitLabels...) {
		label = strings.TrimSpace(label)
		if label == "" || seen[label] {
			continue
		}
		seen[label] = true
		labels = append(labels, label)
	}

	return labels
}

func mapQueuedPriority(priority int) provider.Priority {
	switch {
	case priority <= 1:
		return provider.PriorityHigh
	case priority == 2:
		return provider.PriorityNormal
	case priority >= 3:
		return provider.PriorityLow
	default:
		return provider.PriorityNormal
	}
}

func extractWorkUnitURL(workUnit *provider.WorkUnit) string {
	if workUnit == nil {
		return ""
	}
	if workUnit.Metadata == nil {
		return ""
	}
	if v, ok := workUnit.Metadata["html_url"].(string); ok && v != "" {
		return v
	}
	if v, ok := workUnit.Metadata["web_url"].(string); ok && v != "" {
		return v
	}
	if v, ok := workUnit.Metadata["permalink_url"].(string); ok && v != "" {
		return v
	}
	if v, ok := workUnit.Metadata["url"].(string); ok && v != "" {
		return v
	}

	return ""
}

// submitTaskToProvider creates a work unit in the provider for the given task.
// parentExternalID is the provider's ID for the parent task (if this is a subtask).
func (c *Conductor) submitTaskToProvider(ctx context.Context, creator provider.WorkUnitCreator, task *storage.QueuedTask, opts SubmitOptions, parentExternalID string) (*submittedWorkUnit, error) {
	// Validate required fields
	if task == nil || task.ID == "" {
		return nil, errors.New("task is required")
	}
	if opts.Provider == "" {
		return nil, errors.New("provider is required")
	}
	if creator == nil {
		return nil, errors.New("provider does not support task creation")
	}

	labels := mergeLabels(task.Labels, opts.Labels)
	assignees := []string{}
	if task.Assignee != "" {
		assignees = append(assignees, task.Assignee)
	}

	createOpts := provider.CreateWorkUnitOptions{
		Title:       task.Title,
		Description: task.Description,
		Labels:      labels,
		Assignees:   assignees,
		Priority:    mapQueuedPriority(task.Priority),
		ParentID:    parentExternalID, // Pass parent ID to create subtask
	}

	workUnit, err := creator.CreateWorkUnit(ctx, createOpts)
	if err != nil {
		return nil, err
	}

	externalID := workUnit.ExternalID
	if externalID == "" {
		externalID = workUnit.ID
	}

	return &submittedWorkUnit{
		ID:         workUnit.ID,
		ExternalID: externalID,
		URL:        extractWorkUnitURL(workUnit),
		Title:      workUnit.Title,
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
	ID         string
	ExternalID string
	URL        string
	Title      string
}

// dryRunSubmit simulates submission and returns what would be created.
func (c *Conductor) dryRunSubmit(tasks []*storage.QueuedTask) *SubmitResult {
	result := &SubmitResult{
		Tasks:  make([]*SubmittedTask, 0, len(tasks)),
		DryRun: true,
	}

	for _, task := range tasks {
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

	return result
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

// readResearchSource scans a directory and builds a research manifest.
// This does NOT read file contents - it builds metadata for agent exploration.
// Used for research: source type to avoid token bloat from large documentation bases.
func (c *Conductor) readResearchSource(dirPath string) (*ResearchManifest, error) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", absPath)
	}

	manifest := &ResearchManifest{
		BasePath:    absPath,
		Structure:   make([]DirEntry, 0),
		EntryPoints: make([]string, 0),
		ByCategory:  make(map[string][]string),
	}

	// Entry point patterns to detect
	entryPointPatterns := []string{
		"tasks/README.md", "tasks/index.md",
		"README.md", "readme.md",
		"TODOS.md", "TODO.md", "ROADMAP.md",
	}

	// File extension categories
	docExts := map[string]bool{".md": true, ".txt": true, ".rst": true, ".adoc": true}
	codeExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
		".py": true, ".java": true, ".rs": true, ".rb": true, ".php": true, ".c": true, ".cpp": true,
	}
	configExts := map[string]bool{".yaml": true, ".yml": true, ".json": true, ".toml": true, ".xml": true}

	// Walk directory and collect metadata
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // Skip unreadable files
		}

		// Skip hidden files/directories
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		// Skip common exclusions
		if info.IsDir() {
			switch info.Name() {
			case "node_modules", "vendor", "target", "build", "dist", ".git", "venv", "__pycache__":
				return filepath.SkipDir
			}
		}

		relPath, _ := filepath.Rel(absPath, path)

		entry := DirEntry{
			Path: relPath,
			Name: info.Name(),
			Type: map[bool]string{true: "dir", false: "file"}[info.IsDir()],
			Size: info.Size(),
		}

		if info.IsDir() {
			manifest.Structure = append(manifest.Structure, entry)

			return nil
		}

		// Categorize file
		ext := strings.ToLower(filepath.Ext(path))
		switch {
		case docExts[ext]:
			entry.Category = "docs"
		case codeExts[ext]:
			entry.Category = "code"
		case configExts[ext]:
			entry.Category = "config"
		default:
			entry.Category = "other"
		}

		manifest.Structure = append(manifest.Structure, entry)
		manifest.FileCount++

		// Track by category (store absolute paths)
		manifest.ByCategory[entry.Category] = append(manifest.ByCategory[entry.Category], path)

		// Check for entry points
		if entry.Category == "docs" {
			for _, pattern := range entryPointPatterns {
				if strings.EqualFold(relPath, pattern) ||
					strings.Contains(strings.ToLower(relPath), strings.ToLower(pattern)) {
					manifest.EntryPoints = append(manifest.EntryPoints, path)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}

	// Sort entry points by path length (shorter first = more likely to be root-level)
	sort.Slice(manifest.EntryPoints, func(i, j int) bool {
		return len(manifest.EntryPoints[i]) < len(manifest.EntryPoints[j])
	})

	return manifest, nil
}

// buildResearchPlanningPrompt creates the prompt for research-based planning.
// The prompt provides a file manifest and instructs the agent to use Read/Grep tools
// for selective exploration, rather than concatenating all file contents.
func buildResearchPlanningPrompt(title string, manifest *ResearchManifest, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`You are an expert project manager and software architect. Your task is to research documentation and create a structured task breakdown.

Current timestamp: %s

## Project
%s

## Research Base Path
%s

## Documentation Structure
This directory contains %d files for you to research.
`, currentTime, title, manifest.BasePath, manifest.FileCount))

	// Entry points
	if len(manifest.EntryPoints) > 0 {
		sb.WriteString("## Detected Entry Points\n")
		sb.WriteString("The following files appear to be task/index files:\n\n")
		for i, ep := range manifest.EntryPoints {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, ep))
		}
		sb.WriteString("\nStart by reading these files to understand existing task structure.\n\n")
	}

	// Directory tree
	sb.WriteString("## Directory Structure\n\n")
	sb.WriteString("```\n")
	for _, entry := range manifest.Structure {
		indent := strings.Repeat("  ", strings.Count(entry.Path, string(filepath.Separator)))
		if entry.Type == "dir" {
			sb.WriteString(fmt.Sprintf("%s%s/\n", indent, entry.Name))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s (%s, %d bytes)\n", indent, entry.Name, entry.Category, entry.Size))
		}
	}
	sb.WriteString("```\n\n")

	// Custom instructions
	if customInstructions != "" {
		sb.WriteString(fmt.Sprintf(`## Custom Instructions
%s

`, customInstructions))
	}

	sb.WriteString(`## Research Instructions

IMPORTANT: You have access to Read, Glob, and Grep tools to explore these files.

1. **Start with entry points** - Read the detected entry point files first to understand any existing task structure
2. **Explore selectively** - Use Glob to find relevant files, Grep to search content, and Read to examine specific files
3. **Preserve existing structure** - If tasks/README.md or similar exists, incorporate those tasks rather than creating new ones
4. **Categorize intelligently** - Group related tasks based on the documentation structure you discover

## Output Format

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

Do not include any other text or explanation. Only output the structured task breakdown.
`)

	return sb.String()
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
- **Parent**: task-X (if this is a subtask)
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
3. Use **Parent** vs **Depends on** appropriately:
   - **Parent**: Hierarchical grouping (organizational structure). Use when a task is logically part of a larger task.
   - **Depends on**: Execution ordering. Use when a task cannot start until another task completes.
   - A task can have BOTH a parent AND dependencies (different concepts)
4. Identify dependencies clearly - if task B needs task A, mark B as blocked
5. Tasks without dependencies should be marked as "ready"
6. Prioritize tasks: core functionality first, then enhancements
7. Include labels for categorization (e.g., backend, frontend, testing, docs)
8. If requirements are unclear, add specific questions
9. If there are external blockers, list them

Do not include any other text or explanation. Only output the structured task breakdown.
`

	return prompt
}
