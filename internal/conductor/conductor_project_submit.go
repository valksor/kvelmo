package conductor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
)

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
