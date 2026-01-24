package wrike

import (
	"context"
	"errors"
)

// CreateDependency creates a dependency where predecessorID must complete before successorID.
// Uses Wrike's native dependency API with FinishToStart relationship type.
func (p *Provider) CreateDependency(ctx context.Context, predecessorID, successorID string) error {
	if p.client == nil {
		return errors.New("wrike client not initialized")
	}

	return p.client.CreateDependency(ctx, predecessorID, successorID)
}

// GetDependencies returns the IDs of work units that the given work unit depends on (predecessors).
// Uses Wrike's native dependency API to fetch task dependencies.
func (p *Provider) GetDependencies(ctx context.Context, workUnitID string) ([]string, error) {
	if p.client == nil {
		return nil, errors.New("wrike client not initialized")
	}

	deps, err := p.client.GetTaskDependencies(ctx, workUnitID)
	if err != nil {
		return nil, err
	}

	// Extract predecessor IDs - these are the tasks that the given task depends on
	var predecessorIDs []string
	for _, dep := range deps {
		// If workUnitID is the successor, the predecessor is what it depends on
		if dep.SuccessorID == workUnitID && dep.PredecessorID != "" {
			predecessorIDs = append(predecessorIDs, dep.PredecessorID)
		}
	}

	return predecessorIDs, nil
}
