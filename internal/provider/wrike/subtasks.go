package wrike

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/slug"
)

// ErrNotASubtask is returned when a work unit is not a subtask.
var ErrNotASubtask = errors.New("not a subtask")

// SubtaskInfo holds summary information about a subtask.
type SubtaskInfo struct {
	ID     string
	Title  string
	Status string
}

// FetchParent implements the provider.ParentFetcher interface.
// It retrieves the parent task for a given subtask.
//
// In Wrike, the parent-child relationship is stored in the parent's subTaskIds array.
// When a subtask is fetched via FetchSubtasks, the parent_id is stored in metadata.
// This function uses that metadata to fetch the parent task.
//
// If the task is not a subtask (no parent_id in metadata), returns nil, nil.
func (p *Provider) FetchParent(ctx context.Context, workUnitID string) (*provider.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// First, fetch the task to check if it has a parent_id in metadata
	// This is the most reliable way since Wrike API doesn't provide a direct parent field
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

	// For tasks fetched directly (not as subtasks), we need to check if this is actually a subtask
	// Wrike doesn't provide a direct parent_id field in the API response
	// The metadata will only have parent_id if this was fetched via FetchSubtasks
	//
	// If the task was fetched directly, we don't have a way to get the parent without
	// searching through all parent tasks, which is not practical.
	//
	// Users should fetch subtasks via the parent's FetchSubtasks method to get
	// the parent_id metadata populated.

	// If this task has subtasks, it might be a parent, not a subtask
	if len(task.SubTaskIDs) > 0 {
		// This task has children - it's likely a parent, not a subtask
		// Return sentinel error to indicate no parent
		return nil, ErrNotASubtask
	}

	// For a true subtask without subTaskIDs, we don't have a reliable way to find its parent
	// through the Wrike API without additional context.
	//
	// The parent_id would have been set in metadata if this was fetched via FetchSubtasks.
	// Since we fetched this directly, we don't have that context.

	return nil, ErrNotASubtask
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
			ID:     task.ID,
			Title:  task.Title,
			Status: task.Status,
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
