package asana

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/valksor/go-toolkit/slug"
	"github.com/valksor/go-toolkit/workunit"
)

// CreateWorkUnit creates a new task in Asana.
func (p *Provider) CreateWorkUnit(ctx context.Context, opts workunit.CreateWorkUnitOptions) (*workunit.WorkUnit, error) {
	// Determine target project
	projectGID := opts.ParentID
	if projectGID == "" {
		projectGID = p.config.DefaultProject
	}

	// Validate project is configured
	if projectGID == "" {
		return nil, errors.New("no project specified: set ParentID in options or configure default_project in Asana provider settings")
	}

	// Create the task
	task, err := p.client.CreateTask(ctx, opts.Title, opts.Description, projectGID)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// Build WorkUnit response
	return &workunit.WorkUnit{
		ID:          task.GID,
		ExternalID:  task.GID,
		Provider:    ProviderName,
		Title:       task.Name,
		Description: task.Notes,
		Status:      mapAsanaStatus(task),
		Priority:    workunit.PriorityNormal, // Asana doesn't have built-in priority
		Labels:      extractTagNames(task.Tags),
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.ModifiedAt,
		Source: workunit.SourceInfo{
			Type:      ProviderName,
			Reference: "asana:" + task.GID,
			SyncedAt:  time.Now(),
		},
		Metadata: map[string]any{
			"permalink_url": task.PermalinkURL,
			"projects":      extractProjectNames(task.Projects),
		},
		ExternalKey: task.GID,
		TaskType:    "task",
		Slug:        slug.Slugify(task.Name, 50),
	}, nil
}
