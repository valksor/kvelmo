package linear

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

// FetchParent implements the provider.ParentFetcher interface.
// It retrieves the parent issue for a Linear child issue.
//
// In Linear, child issues have a Parent field in their data.
func (p *Provider) FetchParent(ctx context.Context, workUnitID string) (*provider.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// Get the issue to check if it has a parent
	issue, err := p.client.GetIssue(ctx, ref.IssueID)
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}

	// Check if this issue has a parent (is a child)
	if issue.Parent == nil || issue.Parent.ID == "" {
		// Not a child issue
		return nil, ErrNotASubtask
	}

	// Fetch the parent issue
	parentIssue, err := p.client.GetIssue(ctx, issue.Parent.ID)
	if err != nil {
		return nil, fmt.Errorf("get parent issue: %w", err)
	}

	// Build parent WorkUnit
	wu := &provider.WorkUnit{
		ID:          parentIssue.ID,
		ExternalID:  parentIssue.Identifier,
		ExternalKey: parentIssue.Identifier,
		Provider:    ProviderName,
		Title:       parentIssue.Title,
		Description: parentIssue.Description,
		Status:      mapLinearStatus(parentIssue.State),
		Priority:    mapLinearPriority(parentIssue.Priority),
		Labels:      extractLabelNames(parentIssue.Labels),
		Assignees:   mapAssignees(parentIssue.Assignee),
		CreatedAt:   parentIssue.CreatedAt,
		UpdatedAt:   parentIssue.UpdatedAt,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: parentIssue.Identifier,
			SyncedAt:  time.Now(),
		},
		Metadata: map[string]any{
			"url": parentIssue.URL,
		},
	}

	if parentIssue.State != nil {
		wu.Metadata["state_id"] = parentIssue.State.ID
		wu.Metadata["state_name"] = parentIssue.State.Name
	}

	if parentIssue.Team != nil {
		wu.Metadata["team_key"] = parentIssue.Team.Key
	}

	return wu, nil
}

// FetchSubtasks implements the provider.SubtaskFetcher interface.
// It retrieves child issues for a given Linear issue.
func (p *Provider) FetchSubtasks(ctx context.Context, workUnitID string) ([]*provider.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// Get child issues via GraphQL
	children, err := p.client.GetChildIssues(ctx, ref.IssueID)
	if err != nil {
		return nil, fmt.Errorf("get child issues: %w", err)
	}

	if len(children) == 0 {
		return nil, nil
	}

	// Convert to WorkUnits
	result := make([]*provider.WorkUnit, 0, len(children))
	for _, issue := range children {
		wu := &provider.WorkUnit{
			ID:          issue.ID,
			ExternalID:  issue.Identifier,
			ExternalKey: issue.Identifier,
			Provider:    ProviderName,
			Title:       issue.Title,
			Description: issue.Description,
			Status:      mapLinearStatus(issue.State),
			Priority:    mapLinearPriority(issue.Priority),
			Labels:      extractLabelNames(issue.Labels),
			Assignees:   mapAssignees(issue.Assignee),
			CreatedAt:   issue.CreatedAt,
			UpdatedAt:   issue.UpdatedAt,
			TaskType:    "subtask",
			Slug:        slug.Slugify(issue.Title, 50),
			Source: provider.SourceInfo{
				Type:      ProviderName,
				Reference: issue.Identifier,
				SyncedAt:  time.Now(),
			},
			Metadata: map[string]any{
				"parent_id":  workUnitID,
				"is_subtask": true,
				"url":        issue.URL,
			},
		}

		if issue.State != nil {
			wu.Metadata["state_id"] = issue.State.ID
			wu.Metadata["state_name"] = issue.State.Name
		}

		if issue.Team != nil {
			wu.Metadata["team_key"] = issue.Team.Key
		}

		result = append(result, wu)
	}

	return result, nil
}
