package clickup

import (
	"context"
	"errors"
	"fmt"

	"github.com/valksor/go-toolkit/workunit"
)

// ErrNotASubtask is returned when a work unit is not a subtask.
var ErrNotASubtask = errors.New("not a subtask")

// FetchParent implements the workunit.ParentFetcher interface.
// It retrieves the parent task for a ClickUp subtask.
func (p *Provider) FetchParent(ctx context.Context, workUnitID string) (*workunit.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	taskID := ref.TaskID
	if taskID == "" {
		taskID = ref.CustomID
	}

	// Get the task to check if it has a parent
	task, err := p.client.GetTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Check if this task has a parent (Parent is a string ID)
	if task.Parent == "" {
		// Not a subtask
		return nil, ErrNotASubtask
	}

	// Fetch the parent task
	parentTask, err := p.client.GetTask(ctx, task.Parent)
	if err != nil {
		return nil, fmt.Errorf("get parent task: %w", err)
	}

	// Convert to WorkUnit
	wu := p.taskToWorkUnit(parentTask)

	return wu, nil
}

// FetchSubtasks implements the workunit.SubtaskFetcher interface.
// It retrieves subtasks for a given ClickUp task.
func (p *Provider) FetchSubtasks(ctx context.Context, workUnitID string) ([]*workunit.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	taskID := ref.TaskID
	if taskID == "" {
		taskID = ref.CustomID
	}

	// Get subtasks via API
	subtasks, err := p.client.GetSubtasks(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get subtasks: %w", err)
	}

	if len(subtasks) == 0 {
		return nil, nil
	}

	// Convert to WorkUnits
	result := make([]*workunit.WorkUnit, 0, len(subtasks))
	for _, st := range subtasks {
		wu := p.taskToWorkUnit(&st)
		wu.Metadata["parent_id"] = workUnitID
		wu.Metadata["is_subtask"] = true
		wu.TaskType = "subtask"
		result = append(result, wu)
	}

	return result, nil
}
