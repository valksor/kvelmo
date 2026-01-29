package asana

import (
	"context"
	"errors"
	"fmt"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// ErrNotASubtask is returned when a work unit is not a subtask.
var ErrNotASubtask = errors.New("not a subtask")

// FetchParent implements the provider.ParentFetcher interface.
// It retrieves the parent task for an Asana subtask.
//
// In Asana, the parent relationship is available via the Parent field on the task.
func (p *Provider) FetchParent(ctx context.Context, workUnitID string) (*provider.WorkUnit, error) {
	// Get the task first to check if it has a parent
	task, err := p.client.GetTask(ctx, workUnitID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Check if this task has a parent
	if task.Parent == nil || task.Parent.GID == "" {
		// Not a subtask
		return nil, ErrNotASubtask
	}

	// Fetch the parent task
	parentTask, err := p.client.GetTask(ctx, task.Parent.GID)
	if err != nil {
		return nil, fmt.Errorf("get parent task: %w", err)
	}

	// Convert to WorkUnit
	wu := p.taskToWorkUnit(parentTask)

	return wu, nil
}

// FetchSubtasks implements the provider.SubtaskFetcher interface.
// It retrieves subtasks for a given Asana task.
func (p *Provider) FetchSubtasks(ctx context.Context, workUnitID string) ([]*provider.WorkUnit, error) {
	// Get subtasks via API
	subtasks, err := p.client.GetSubtasks(ctx, workUnitID)
	if err != nil {
		return nil, fmt.Errorf("get subtasks: %w", err)
	}

	if len(subtasks) == 0 {
		return nil, nil
	}

	// Convert to WorkUnits
	result := make([]*provider.WorkUnit, 0, len(subtasks))
	for _, st := range subtasks {
		wu := p.taskToWorkUnit(&st)
		wu.Metadata["parent_id"] = workUnitID
		wu.Metadata["is_subtask"] = true
		wu.TaskType = "subtask"
		result = append(result, wu)
	}

	return result, nil
}
