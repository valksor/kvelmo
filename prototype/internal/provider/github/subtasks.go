package github

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/slug"
)

// ErrNotASubtask is returned when a work unit is not a subtask.
var ErrNotASubtask = errors.New("not a subtask")

// FetchParent implements the provider.ParentFetcher interface.
// It retrieves the parent issue for a GitHub subtask (task list item).
//
// GitHub subtasks are task list items within an issue body.
// The subtask ID format is: {owner}/{repo}#{issue_number}-task-{n}.
func (p *Provider) FetchParent(ctx context.Context, workUnitID string) (*provider.WorkUnit, error) {
	// Check if this is a subtask (has "-task-" in the ID)
	if !strings.Contains(workUnitID, "-task-") {
		// Regular issue, not a subtask
		return nil, ErrNotASubtask
	}

	// Parse the subtask ID to extract parent issue number
	// Format: {owner}/{repo}#{issue_number}-task-{n}
	//       or #{issue_number}-task-{n}
	var issueNumber int
	var owner, repo string

	// Try explicit owner/repo format first
	if matches := explicitRefPattern.FindStringSubmatch(workUnitID); matches != nil {
		// Extract the issue number before "-task-"
		beforeTask := strings.Split(matches[3], "-task-")[0]
		num, err := strconv.Atoi(beforeTask)
		if err != nil {
			return nil, fmt.Errorf("parse issue number: %w", err)
		}
		issueNumber = num
		owner = matches[1]
		repo = matches[2]
	} else {
		// Try to find the issue number in simple format
		// Look for pattern like "#123-task-1" or "123-task-1"
		taskSplit := strings.Split(workUnitID, "-task-")
		if len(taskSplit) < 2 {
			return nil, fmt.Errorf("%w: invalid subtask ID format: %s", ErrInvalidReference, workUnitID)
		}

		// Extract issue number from the first part
		firstPart := strings.TrimSpace(taskSplit[0])
		firstPart = strings.TrimPrefix(firstPart, "#")

		num, err := strconv.Atoi(firstPart)
		if err != nil {
			return nil, fmt.Errorf("parse issue number: %w", err)
		}
		issueNumber = num

		// Use configured owner/repo
		owner = p.owner
		repo = p.repo
	}

	if owner == "" || repo == "" {
		return nil, ErrRepoNotConfigured
	}

	// Update client with correct owner/repo
	p.client.SetOwnerRepo(owner, repo)

	// Fetch the parent issue
	issue, err := p.client.GetIssue(ctx, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("get parent issue: %w", err)
	}

	// Build parent WorkUnit
	parentID := fmt.Sprintf("%s/%s#%d", owner, repo, issueNumber)

	return &provider.WorkUnit{
		ID:          parentID,
		ExternalID:  strconv.FormatInt(issue.GetID(), 10),
		ExternalKey: strconv.FormatInt(issue.GetID(), 10),
		Provider:    ProviderName,
		Title:       issue.GetTitle(),
		Description: issue.GetBody(),
		Status:      mapGitHubState(issue.GetState()),
		Priority:    inferPriorityFromLabels(issue.Labels),
		Labels:      extractLabelNames(issue.Labels),
		Assignees:   mapAssignees(issue.Assignees),
		CreatedAt:   issue.GetCreatedAt().Time,
		UpdatedAt:   issue.GetUpdatedAt().Time,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: parentID,
			SyncedAt:  time.Now(),
		},
		Metadata: map[string]any{
			"issue_number": issueNumber,
			"owner":        owner,
			"repo":         repo,
			"state":        issue.GetState(),
		},
	}, nil
}

// FetchSubtasks implements the provider.SubtaskFetcher interface.
// It parses task list items from the GitHub issue body.
func (p *Provider) FetchSubtasks(ctx context.Context, workUnitID string) ([]*provider.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// Determine owner/repo
	owner := ref.Owner
	repo := ref.Repo
	if owner == "" {
		owner = p.owner
	}
	if repo == "" {
		repo = p.repo
	}
	if owner == "" || repo == "" {
		return nil, ErrRepoNotConfigured
	}

	// Update client with correct owner/repo
	p.client.SetOwnerRepo(owner, repo)

	// Fetch the issue
	issue, err := p.client.GetIssue(ctx, ref.IssueNumber)
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}

	// Parse task list from issue body
	tasks := ParseTaskList(issue.GetBody())
	if len(tasks) == 0 {
		return nil, nil
	}

	// Convert to WorkUnits
	result := make([]*provider.WorkUnit, 0, len(tasks))
	for i, task := range tasks {
		// Determine status from completion state
		status := provider.StatusOpen
		if task.Completed {
			status = provider.StatusDone
		}

		// Create a unique ID for this task item
		taskID := fmt.Sprintf("%s/%s#%d-task-%d", owner, repo, ref.IssueNumber, i+1)

		wu := &provider.WorkUnit{
			ID:          taskID,
			ExternalID:  taskID,
			ExternalKey: fmt.Sprintf("%d-task-%d", ref.IssueNumber, i+1),
			Provider:    ProviderName,
			Title:       task.Text,
			Status:      status,
			Priority:    provider.PriorityNormal,
			TaskType:    "subtask",
			Slug:        slug.Slugify(task.Text, 50),
			CreatedAt:   issue.GetCreatedAt().Time,
			UpdatedAt:   issue.GetUpdatedAt().Time,
			Source: provider.SourceInfo{
				Type:      ProviderName,
				Reference: taskID,
				SyncedAt:  time.Now(),
			},
			Metadata: map[string]any{
				"parent_id":    workUnitID,
				"is_subtask":   true,
				"line_number":  task.Line,
				"completed":    task.Completed,
				"issue_number": ref.IssueNumber,
				"owner":        owner,
				"repo":         repo,
			},
		}

		result = append(result, wu)
	}

	return result, nil
}
