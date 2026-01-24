package asana

import (
	"context"
	"errors"
	"fmt"
)

// CreateDependency creates a dependency between two tasks.
// Asana has native task dependency support via the API.
func (p *Provider) CreateDependency(ctx context.Context, predecessorID, successorID string) error {
	if p.client == nil {
		return errors.New("client not initialized")
	}

	// Asana uses task dependencies - successor depends on predecessor
	err := p.client.AddTaskDependency(ctx, successorID, predecessorID)
	if err != nil {
		return fmt.Errorf("add task dependency: %w", err)
	}

	return nil
}

// GetDependencies returns the task GIDs that the given task depends on.
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
func (c *Client) AddTaskDependency(ctx context.Context, taskGID, dependsOnGID string) error {
	// POST /tasks/{task_gid}/addDependencies
	// { "data": { "dependencies": ["dependsOnGID"] } }
	// Future: Implement using Asana REST API
	return nil
}

// GetTaskDependencies returns dependencies for a task (stub for now).
func (c *Client) GetTaskDependencies(ctx context.Context, taskGID string) ([]string, error) {
	// GET /tasks/{task_gid}/dependencies
	// Future: Implement using Asana REST API
	return nil, nil
}
