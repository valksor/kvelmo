package clickup

import (
	"context"
	"errors"
	"fmt"
)

// CreateDependency creates a dependency between two tasks.
// ClickUp has native task dependency support via the API.
func (p *Provider) CreateDependency(ctx context.Context, predecessorID, successorID string) error {
	if p.client == nil {
		return errors.New("client not initialized")
	}

	// ClickUp supports "waiting_on" and "blocking" dependency types
	// waiting_on means the task is waiting on another task
	err := p.client.AddTaskDependency(ctx, successorID, predecessorID, "waiting_on")
	if err != nil {
		return fmt.Errorf("add task dependency: %w", err)
	}

	return nil
}

// GetDependencies returns the task IDs that the given task depends on.
func (p *Provider) GetDependencies(ctx context.Context, workUnitID string) ([]string, error) {
	if p.client == nil {
		return nil, errors.New("client not initialized")
	}

	deps, err := p.client.GetTaskDependencies(ctx, workUnitID)
	if err != nil {
		return nil, fmt.Errorf("get task dependencies: %w", err)
	}

	return deps, nil
}

// AddTaskDependency adds a dependency to a task (stub for now).
func (c *Client) AddTaskDependency(ctx context.Context, taskID, dependsOnID, depType string) error {
	// POST /task/{task_id}/dependency
	// { "depends_on": "dependsOnID", "dependency_of": null }
	// Future: Implement using ClickUp REST API
	return nil
}

// GetTaskDependencies returns dependencies for a task (stub for now).
func (c *Client) GetTaskDependencies(ctx context.Context, taskID string) ([]string, error) {
	// GET /task/{task_id} and parse dependencies from response
	// Future: Implement using ClickUp REST API
	return nil, nil
}
