package wrike

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/slug"
)

// ErrNotASubtask is returned when a work unit is not a subtask.
var ErrNotASubtask = errors.New("not a subtask")

// SubtaskInfo holds summary information about a subtask.
type SubtaskInfo struct {
	ID          string
	Title       string
	Status      string
	Description string
}

// ParentTaskInfo holds summary information about a parent task.
// Intentionally excludes SubTaskIDs to avoid showing sibling subtasks.
type ParentTaskInfo struct {
	ID          string
	Title       string
	Status      string
	Description string
	Permalink   string
}

// FetchParent implements the provider.ParentFetcher interface.
// It retrieves the parent task for a given subtask.
//
// Uses SuperTaskIDs from the Wrike API to identify and fetch the parent task.
// If the task is not a subtask (no SuperTaskIDs), returns ErrNotASubtask.
func (p *Provider) FetchParent(ctx context.Context, workUnitID string) (*provider.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	var task *Task

	switch {
	case ref.Permalink != "":
		task, err = p.client.GetTaskByPermalinkParam(ctx, ref.Permalink)
	case apiIDPattern.MatchString(ref.TaskID):
		task, err = p.client.GetTask(ctx, ref.TaskID)
	case numericIDPattern.MatchString(ref.TaskID):
		permalink := BuildPermalinkURL(ref.TaskID)
		task, err = p.client.GetTaskByPermalinkParam(ctx, permalink)
	default:
		return nil, fmt.Errorf("%w: unrecognized task ID format: %s", ErrInvalidReference, ref.TaskID)
	}

	if err != nil {
		return nil, fmt.Errorf("fetch task: %w", err)
	}

	// Check if task has SuperTaskIDs (is a subtask)
	if len(task.SuperTaskIDs) == 0 {
		return nil, ErrNotASubtask
	}

	parentID := task.SuperTaskIDs[0]

	// Guard against circular reference (malformed API data)
	if parentID == "" || parentID == task.ID {
		return nil, ErrNotASubtask
	}

	// Fetch first parent (typically there's only one)
	parentTask, err := p.client.GetTask(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("fetch parent task: %w", err)
	}

	return p.taskToWorkUnit(parentTask), nil
}

// FetchSubtasks implements the provider.SubtaskFetcher interface.
// It retrieves all subtasks for a given work unit as full WorkUnit objects.
func (p *Provider) FetchSubtasks(ctx context.Context, workUnitID string) ([]*provider.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	var task *Task

	switch {
	case ref.Permalink != "":
		task, err = p.client.GetTaskByPermalinkParam(ctx, ref.Permalink)
	case apiIDPattern.MatchString(ref.TaskID):
		task, err = p.client.GetTask(ctx, ref.TaskID)
	case numericIDPattern.MatchString(ref.TaskID):
		permalink := BuildPermalinkURL(ref.TaskID)
		task, err = p.client.GetTaskByPermalinkParam(ctx, permalink)
	default:
		return nil, fmt.Errorf("%w: unrecognized task ID format: %s", ErrInvalidReference, ref.TaskID)
	}

	if err != nil {
		return nil, fmt.Errorf("fetch parent task: %w", err)
	}

	if len(task.SubTaskIDs) == 0 {
		return nil, nil
	}

	// Fetch subtasks as full Task objects
	tasks, err := p.client.GetTasks(ctx, task.SubTaskIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch subtasks: %w", err)
	}

	// Convert to WorkUnits
	workUnits := make([]*provider.WorkUnit, 0, len(tasks))
	for _, st := range tasks {
		wu := p.subtaskToWorkUnit(&st, workUnitID)
		workUnits = append(workUnits, wu)
	}

	return workUnits, nil
}

// subtaskToWorkUnit converts a subtask Task to a WorkUnit with parent reference.
func (p *Provider) subtaskToWorkUnit(task *Task, parentID string) *provider.WorkUnit {
	numericID := ExtractNumericID(task.Permalink)
	if numericID == "" {
		numericID = task.ID
	}

	return &provider.WorkUnit{
		ID:          numericID,
		ExternalID:  task.ID,
		Provider:    ProviderName,
		Title:       task.Title,
		Description: task.Description,
		Status:      mapStatus(task.Status),
		Priority:    mapPriority(task.Priority),
		Labels:      []string{},
		Assignees:   []provider.Person{},
		Subtasks:    task.SubTaskIDs,
		Metadata: map[string]any{
			"permalink":      task.Permalink,
			"api_id":         task.ID,
			"wrike_status":   task.Status,
			"wrike_priority": task.Priority,
			"parent_id":      parentID,
			"is_subtask":     true,
			"subtask_count":  len(task.SubTaskIDs),
		},
		CreatedAt: task.CreatedDate,
		UpdatedAt: task.UpdatedDate,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: task.Permalink,
			SyncedAt:  time.Now(),
		},
		ExternalKey: numericID,
		TaskType:    "subtask",
		Slug:        slug.Slugify(task.Title, 50),
	}
}

// fetchParentTask fetches parent task info for a subtask (internal helper for Snapshot).
// Returns nil, nil if no parent exists or on circular reference.
// Intentionally excludes sibling subtasks per user requirement.
func (p *Provider) fetchParentTask(ctx context.Context, taskID string, superTaskIDs []string) (*ParentTaskInfo, error) {
	if len(superTaskIDs) == 0 {
		return nil, nil //nolint:nilnil // No parent exists (not an error)
	}

	parentID := superTaskIDs[0]

	// Guard against circular reference (malformed API data)
	if parentID == "" || parentID == taskID {
		return nil, nil //nolint:nilnil // Circular reference treated as no parent (not an error)
	}

	// Log if multiple parents exist (unusual but possible)
	if len(superTaskIDs) > 1 {
		slog.Debug("task has multiple parents, using first", "task_id", taskID, "parent_count", len(superTaskIDs))
	}

	// Fetch first parent only - Wrike typically has single parent
	// NOTE: Intentionally excludes sibling subtasks per user requirement
	parentTask, err := p.client.GetTask(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("fetch parent task %s: %w", parentID, err)
	}

	return &ParentTaskInfo{
		ID:          parentTask.ID,
		Title:       parentTask.Title,
		Status:      parentTask.Status,
		Description: parentTask.Description,
		Permalink:   parentTask.Permalink,
	}, nil
}

// fetchSubtasks recursively fetches all subtasks for a task (internal helper)
// Returns a list of subtask info.
func (p *Provider) fetchSubtasks(ctx context.Context, subtaskIDs []string, depth int) ([]SubtaskInfo, error) {
	const maxDepth = 5 // Prevent infinite recursion
	if depth > maxDepth {
		return nil, nil
	}

	if len(subtaskIDs) == 0 {
		return nil, nil
	}

	// Fetch all subtasks in one batch
	tasks, err := p.client.GetTasks(ctx, subtaskIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch subtasks: %w", err)
	}

	var infos []SubtaskInfo
	var allSubtaskIDs []string

	for _, task := range tasks {
		info := SubtaskInfo{
			ID:          task.ID,
			Title:       task.Title,
			Status:      task.Status,
			Description: task.Description,
		}
		infos = append(infos, info)

		// Collect nested subtask IDs for recursive fetching
		if len(task.SubTaskIDs) > 0 {
			allSubtaskIDs = append(allSubtaskIDs, task.SubTaskIDs...)
		}
	}

	// Recursively fetch nested subtasks
	if len(allSubtaskIDs) > 0 {
		nestedInfos, err := p.fetchSubtasks(ctx, allSubtaskIDs, depth+1)
		if err != nil {
			return infos, err // Return what we have so far
		}
		infos = append(infos, nestedInfos...)
	}

	return infos, nil
}
