package notion

import (
	"context"
	"fmt"

	"github.com/valksor/go-toolkit/workunit"
)

// CreateWorkUnit creates a new Notion page.
func (p *Provider) CreateWorkUnit(ctx context.Context, opts workunit.CreateWorkUnitOptions) (*workunit.WorkUnit, error) {
	databaseID := p.databaseID
	if databaseID == "" {
		return nil, fmt.Errorf("%w: specify notion.database_id in config", ErrDatabaseRequired)
	}

	// Determine status from CustomFields or default to open
	status := workunit.StatusOpen
	if opts.CustomFields != nil {
		if s, ok := opts.CustomFields["status"].(workunit.Status); ok {
			status = s
		}
	}

	// Build properties
	properties := map[string]Property{
		"Name":           MakeTitleProperty(opts.Title),
		p.statusProperty: MakeStatusProperty(mapProviderStatusToNotion(status)),
	}

	// Add description if provided
	if opts.Description != "" {
		properties[p.descriptionProperty] = MakeRichTextProperty(opts.Description)
	}

	// Add labels if provided
	if len(opts.Labels) > 0 {
		properties[p.labelsProperty] = MakeMultiSelectProperty(opts.Labels)
	}

	// Add assignee if provided (use first from list)
	if len(opts.Assignees) > 0 {
		properties["Assignee"] = Property{
			Type: "people",
			People: &PeopleProp{
				Type: "people",
				People: []User{
					{
						ID: opts.Assignees[0],
					},
				},
			},
		}
	}

	createInput := &CreatePageInput{
		Parent: Parent{
			Type:       "database_id",
			DatabaseID: databaseID,
		},
		Properties: properties,
	}

	page, err := p.client.CreatePage(ctx, createInput)
	if err != nil {
		return nil, err
	}

	return &workunit.WorkUnit{
		ID:          page.ID,
		ExternalID:  page.ID,
		Provider:    ProviderName,
		Title:       opts.Title,
		Description: opts.Description,
		Status:      status,
		Priority:    opts.Priority,
		Labels:      opts.Labels,
		CreatedAt:   page.CreatedTime,
		UpdatedAt:   page.LastEditedTime,
		Source: workunit.SourceInfo{
			Type:      ProviderName,
			Reference: page.URL,
			SyncedAt:  page.CreatedTime,
		},
		ExternalKey: page.ID[:8],
		TaskType:    "page",
	}, nil
}
