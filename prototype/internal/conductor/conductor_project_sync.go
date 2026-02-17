package conductor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/workunit"
)

// SyncProject pulls an entire project/epic from a provider into a local queue.
// Reference format: "provider:reference" (e.g., "wrike:https://...", "jira:PROJ-123").
func (c *Conductor) SyncProject(ctx context.Context, reference string, opts SyncProjectOptions) (*SyncProjectResult, error) {
	c.publishProgress("Syncing project structure...", 0)

	// Parse provider:reference format
	parts := strings.SplitN(reference, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid reference format: %s (expected provider:reference)", reference)
	}

	providerName := parts[0]
	projectRef := parts[1]

	// Get provider from registry
	_, factory, ok := c.providers.Get(providerName)
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}

	workspaceCfg, _ := c.workspace.LoadConfig()
	providerCfg := buildProviderConfig(ctx, workspaceCfg, providerName)
	providerInst, err := factory(ctx, providerCfg)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}

	c.publishProgress(fmt.Sprintf("Fetching from %s...", providerName), 20)

	// Check for project fetch capability
	projectFetcher, hasProject := providerInst.(workunit.ProjectFetcher)
	_, hasSubtasks := providerInst.(workunit.SubtaskFetcher)

	if !hasProject && !hasSubtasks {
		return nil, fmt.Errorf("provider %s does not support project fetching (no ProjectFetcher or SubtaskFetcher)", providerName)
	}

	var projectStruct *workunit.ProjectStructure

	// Try ProjectFetcher first (preferred for bulk operations)
	if hasProject {
		c.publishProgress("Fetching project structure...", 30)
		projectStruct, err = projectFetcher.FetchProject(ctx, projectRef)
		if err != nil {
			return nil, fmt.Errorf("fetch project: %w", err)
		}
	} else {
		// Fallback: Fetch single work unit and recursively fetch subtasks
		c.publishProgress("Fetching epic tasks (fallback mode)...", 30)
		projectStruct, err = c.fetchProjectRecursive(ctx, providerInst, projectRef, opts.MaxDepth)
		if err != nil {
			return nil, fmt.Errorf("fetch project recursively: %w", err)
		}
	}

	c.publishProgress("Processing tasks...", 50)

	// Apply status filtering
	var tasks []*workunit.ProjectTask
	if len(opts.IncludeStatus) > 0 {
		tasks = c.applyStatusFilter(projectStruct.Tasks, opts.IncludeStatus)
	} else {
		tasks = applySmartStatusFilter(projectStruct.Tasks)
	}

	// Generate queue ID with timestamp
	queueID := generateQueueID(projectStruct.Title, reference)

	// Create the task queue
	queue := storage.NewTaskQueue(queueID, projectStruct.Title, reference)

	// Track existing tasks by external ID to handle dependencies
	taskIDMap := make(map[string]string) // external ID -> local task ID

	for _, pt := range tasks {
		queuedTask := c.projectTaskToQueued(pt, queue, taskIDMap)
		queue.AddTask(queuedTask)
		if pt.ExternalID != "" {
			taskIDMap[pt.ExternalID] = queuedTask.ID
		}
	}

	// Compute dependency relationships
	queue.ComputeBlocksRelations()
	queue.ComputeTaskStatuses()

	c.publishProgress("Saving queue...", 80)

	// Save the queue
	if err := c.workspace.SaveTaskQueue(queue); err != nil {
		return nil, fmt.Errorf("save queue: %w", err)
	}

	c.publishProgress("Project sync complete", 100)

	return &SyncProjectResult{
		Queue:     queue,
		TasksSync: len(tasks),
		TasksNew:  len(queue.Tasks),
		Source:    providerName,
		URL:       projectStruct.URL,
	}, nil
}

// fetchProjectRecursive builds a project structure by recursively fetching subtasks.
// Used as fallback when provider doesn't implement ProjectFetcher.
func (c *Conductor) fetchProjectRecursive(ctx context.Context, p any, workUnitID string, maxDepth int) (*workunit.ProjectStructure, error) {
	reader, ok := p.(workunit.Reader)
	if !ok {
		return nil, errors.New("provider does not implement Reader")
	}
	subtaskFetcher, ok := p.(workunit.SubtaskFetcher)
	if !ok {
		return nil, errors.New("provider does not implement SubtaskFetcher")
	}

	// Fetch parent work unit
	parent, err := reader.Fetch(ctx, workUnitID)
	if err != nil {
		return nil, fmt.Errorf("fetch parent: %w", err)
	}

	structure := &workunit.ProjectStructure{
		ID:          parent.ID,
		Title:       parent.Title,
		Description: parent.Description,
		Source:      parent.Provider,
		URL:         extractWorkUnitURL(parent),
		Tasks:       make([]*workunit.ProjectTask, 0),
		Metadata:    make(map[string]any),
	}

	// Recursively fetch all subtasks
	visited := make(map[string]bool)
	var fetchRecursive func(*workunit.WorkUnit, string, int) error

	fetchRecursive = func(wu *workunit.WorkUnit, parentID string, depth int) error {
		if maxDepth > 0 && depth >= maxDepth {
			return nil
		}
		if visited[wu.ID] {
			return nil
		}
		visited[wu.ID] = true

		// Add as a task
		pt := &workunit.ProjectTask{
			WorkUnit: wu,
			ParentID: parentID,
			Depth:    depth,
			Position: len(structure.Tasks),
		}
		structure.Tasks = append(structure.Tasks, pt)

		// Fetch and recurse into subtasks
		subtasks, err := subtaskFetcher.FetchSubtasks(ctx, wu.ID)
		if err != nil {
			// Non-fatal: log but continue with subtasks as empty
			subtasks = nil
		}

		for _, st := range subtasks {
			if err := fetchRecursive(st, wu.ID, depth+1); err != nil {
				return err
			}
		}

		return nil
	}

	if err := fetchRecursive(parent, "", 0); err != nil {
		return nil, err
	}

	return structure, nil
}

// applyStatusFilter filters tasks by status.
func (c *Conductor) applyStatusFilter(tasks []*workunit.ProjectTask, statuses []string) []*workunit.ProjectTask {
	statusMap := make(map[string]bool)
	for _, s := range statuses {
		statusMap[strings.ToLower(strings.TrimSpace(s))] = true
	}

	var filtered []*workunit.ProjectTask
	for _, task := range tasks {
		if statusMap[string(task.Status)] {
			filtered = append(filtered, task)
		}
	}

	return filtered
}

// applySmartStatusFilter applies smart default filtering:
// - Open tasks
// - In-progress tasks
// - Completed tasks from last 30 days.
func applySmartStatusFilter(tasks []*workunit.ProjectTask) []*workunit.ProjectTask {
	cutoff := time.Now().AddDate(0, 0, -30) // 30 days ago
	var filtered []*workunit.ProjectTask

	for _, task := range tasks {
		if task.Status == workunit.StatusOpen ||
			task.Status == workunit.StatusInProgress ||
			(task.Status == workunit.StatusDone && task.UpdatedAt.After(cutoff)) {
			filtered = append(filtered, task)
		}
	}

	return filtered
}

// projectTaskToQueued converts a provider ProjectTask to a storage QueuedTask.
func (c *Conductor) projectTaskToQueued(pt *workunit.ProjectTask, queue *storage.TaskQueue, taskIDMap map[string]string) *storage.QueuedTask {
	taskID := queue.NextTaskID()

	// Map provider status to queue status
	var status storage.TaskStatus
	switch pt.Status {
	case workunit.StatusOpen:
		status = storage.TaskStatusReady
	case workunit.StatusInProgress:
		status = storage.TaskStatusReady
	case workunit.StatusReview:
		status = storage.TaskStatusReady
	case workunit.StatusDone, workunit.StatusClosed:
		status = storage.TaskStatusSubmitted
	default:
		status = storage.TaskStatusPending
	}

	// Map priority
	priority := 3 // default
	switch pt.Priority {
	case workunit.PriorityCritical:
		priority = 1
	case workunit.PriorityHigh:
		priority = 2
	case workunit.PriorityNormal:
		priority = 3
	case workunit.PriorityLow:
		priority = 4
	}

	// Build dependencies from parent relationship
	var dependsOn []string
	if pt.ParentID != "" {
		// Find the task ID for this parent
		for localID, extID := range taskIDMap {
			if extID == pt.ParentID {
				dependsOn = append(dependsOn, localID)

				break
			}
		}
	}

	// Build labels list
	var labels []string
	labels = append(labels, pt.Labels...)

	// Preserve source path from provider reference
	var sourcePath string
	if pt.WorkUnit != nil && pt.Source.Reference != "" {
		sourcePath = pt.Source.Reference
	}

	// Preserve metadata from provider (custom frontmatter, readme paths, etc.)
	var metadata map[string]any
	if pt.WorkUnit != nil && len(pt.Metadata) > 0 {
		metadata = make(map[string]any, len(pt.Metadata))
		for k, v := range pt.Metadata {
			metadata[k] = v
		}
	}

	return &storage.QueuedTask{
		ID:          taskID,
		Title:       pt.Title,
		Description: pt.Description,
		Status:      status,
		Priority:    priority,
		DependsOn:   dependsOn,
		Labels:      labels,
		ExternalID:  pt.ExternalID,
		ExternalURL: extractWorkUnitURL(pt.WorkUnit),
		SourcePath:  sourcePath,
		Metadata:    metadata,
	}
}
