package jira

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
// It retrieves the parent issue for a Jira subtask.
//
// In Jira, subtasks are first-class issues with a Parent field in their Fields.
func (p *Provider) FetchParent(ctx context.Context, workUnitID string) (*provider.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// Update base URL if detected from reference
	if ref.BaseURL != "" && p.baseURL == "" {
		p.baseURL = ref.BaseURL
		p.client.SetBaseURL(ref.BaseURL)
	}

	// First, fetch the issue to check if it has a parent
	issue, err := p.client.GetIssue(ctx, ref.IssueKey)
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}

	// Check if this issue has a parent (is a subtask)
	if issue.Fields.Parent == nil || issue.Fields.Parent.Key == "" {
		// Not a subtask
		return nil, ErrNotASubtask
	}

	// Fetch the parent issue
	parentIssue, err := p.client.GetIssue(ctx, issue.Fields.Parent.Key)
	if err != nil {
		return nil, fmt.Errorf("get parent issue: %w", err)
	}

	// Build parent WorkUnit
	return &provider.WorkUnit{
		ID:          parentIssue.ID,
		ExternalID:  parentIssue.Key,
		ExternalKey: parentIssue.Key,
		Provider:    ProviderName,
		Title:       parentIssue.Fields.Summary,
		Description: parentIssue.Fields.Description,
		Status:      mapJiraStatus(parentIssue.Fields.Status.Name),
		Priority:    mapJiraPriority(parentIssue.Fields.Priority),
		Labels:      parentIssue.Fields.Labels,
		Assignees:   mapAssignees(parentIssue.Fields.Assignee),
		CreatedAt:   parentIssue.Fields.Created,
		UpdatedAt:   parentIssue.Fields.Updated,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: parentIssue.Key,
			SyncedAt:  time.Now(),
		},
		Metadata: map[string]any{
			"key": parentIssue.Key,
		},
	}, nil
}

// FetchSubtasks implements the provider.SubtaskFetcher interface.
// It retrieves subtasks for a given Jira issue.
func (p *Provider) FetchSubtasks(ctx context.Context, workUnitID string) ([]*provider.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// Update base URL if detected from reference
	if ref.BaseURL != "" && p.baseURL == "" {
		p.baseURL = ref.BaseURL
		p.client.SetBaseURL(ref.BaseURL)
	}

	// Get subtasks from the issue
	subtasks, err := p.client.GetSubtasks(ctx, ref.IssueKey)
	if err != nil {
		return nil, fmt.Errorf("get subtasks: %w", err)
	}

	if len(subtasks) == 0 {
		return nil, nil
	}

	// Convert to WorkUnits
	result := make([]*provider.WorkUnit, 0, len(subtasks))
	for _, issue := range subtasks {
		wu := &provider.WorkUnit{
			ID:          issue.ID,
			ExternalID:  issue.Key,
			ExternalKey: issue.Key,
			Provider:    ProviderName,
			Title:       issue.Fields.Summary,
			Description: issue.Fields.Description,
			Status:      mapJiraStatus(issue.Fields.Status.Name),
			Priority:    mapJiraPriority(issue.Fields.Priority),
			Labels:      issue.Fields.Labels,
			Assignees:   mapAssignees(issue.Fields.Assignee),
			CreatedAt:   issue.Fields.Created,
			UpdatedAt:   issue.Fields.Updated,
			TaskType:    "subtask",
			Slug:        slug.Slugify(issue.Fields.Summary, 50),
			Source: provider.SourceInfo{
				Type:      ProviderName,
				Reference: issue.Key,
				SyncedAt:  time.Now(),
			},
			Metadata: map[string]any{
				"parent_id":  workUnitID,
				"is_subtask": true,
				"key":        issue.Key,
			},
		}

		if issue.Fields.Status != nil {
			wu.Metadata["status_name"] = issue.Fields.Status.Name
		}

		if issue.Fields.Project != nil {
			wu.Metadata["project_key"] = issue.Fields.Project.Key
		}

		if issue.Fields.Issuetype != nil {
			wu.Metadata["issue_type"] = issue.Fields.Issuetype.Name
		}

		result = append(result, wu)
	}

	return result, nil
}
