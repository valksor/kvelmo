package bitbucket

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/slug"
)

// taskListPattern matches GitHub/GitLab-style task list items
// Matches: - [ ] Task item or - [x] Completed task.
var taskListPattern = regexp.MustCompile(`(?m)^[\s]*[-*]\s*\[([ xX])\]\s*(.+)$`)

// ErrNotASubtask is returned when a work unit is not a subtask.
var ErrNotASubtask = errors.New("not a subtask")

// FetchParent implements the provider.ParentFetcher interface.
// It retrieves the parent issue for a Bitbucket task list item.
func (p *Provider) FetchParent(ctx context.Context, workUnitID string) (*provider.WorkUnit, error) {
	// Check if this is a subtask (has ":task-" in the ID)
	if !strings.Contains(workUnitID, ":task-") {
		// Regular issue, not a subtask
		return nil, ErrNotASubtask
	}

	// Parse the subtask ID to extract parent issue
	// Format: {parentID}:task-{n}
	taskSplit := strings.Split(workUnitID, ":task-")
	if len(taskSplit) < 2 {
		return nil, fmt.Errorf("%w: invalid subtask ID format: %s", ErrInvalidReference, workUnitID)
	}

	parentID := taskSplit[0]

	// Parse the parent reference
	ref, err := ParseReference(parentID)
	if err != nil {
		return nil, fmt.Errorf("parse parent reference: %w", err)
	}

	workspace := ref.Workspace
	repoSlug := ref.RepoSlug
	if workspace == "" {
		workspace = p.config.Workspace
	}
	if repoSlug == "" {
		repoSlug = p.config.RepoSlug
	}

	if workspace == "" || repoSlug == "" {
		return nil, ErrRepoNotConfigured
	}

	p.client.SetWorkspaceRepo(workspace, repoSlug)

	// Fetch the parent issue
	issue, err := p.client.GetIssue(ctx, ref.IssueID)
	if err != nil {
		return nil, fmt.Errorf("get parent issue: %w", err)
	}

	// Build parent WorkUnit
	displayID := fmt.Sprintf("%s/%s#%d", workspace, repoSlug, ref.IssueID)

	// Get description
	description := ""
	if issue.Content != nil {
		description = issue.Content.Raw
	}

	return &provider.WorkUnit{
		ID:          displayID,
		ExternalID:  displayID,
		ExternalKey: strconv.Itoa(ref.IssueID),
		Provider:    ProviderName,
		Title:       issue.Title,
		Description: description,
		Status:      mapBitbucketState(issue.State),
		Priority:    provider.PriorityNormal,
		Labels:      []string{}, // Bitbucket uses components, not labels
		CreatedAt:   issue.CreatedOn,
		UpdatedAt:   issue.UpdatedOn,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: displayID,
			SyncedAt:  time.Now(),
		},
		Metadata: map[string]any{
			"workspace": workspace,
			"repo_slug": repoSlug,
			"state":     issue.State,
		},
	}, nil
}

// FetchSubtasks implements the provider.SubtaskFetcher interface.
// For Bitbucket, this parses task list items (- [ ] item) from the issue body.
// Note: Bitbucket doesn't have native subtasks, so we parse markdown task lists.
func (p *Provider) FetchSubtasks(ctx context.Context, workUnitID string) ([]*provider.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, err
	}

	workspace := ref.Workspace
	repoSlug := ref.RepoSlug
	if workspace == "" {
		workspace = p.config.Workspace
	}
	if repoSlug == "" {
		repoSlug = p.config.RepoSlug
	}

	if workspace == "" || repoSlug == "" {
		return nil, ErrRepoNotConfigured
	}

	p.client.SetWorkspaceRepo(workspace, repoSlug)

	// Fetch the issue
	issue, err := p.client.GetIssue(ctx, ref.IssueID)
	if err != nil {
		return nil, fmt.Errorf("fetch issue: %w", err)
	}

	// Parse task list from body
	if issue.Content == nil || issue.Content.Raw == "" {
		return nil, nil
	}

	return parseTaskListToWorkUnits(issue.Content.Raw, workUnitID), nil
}

// parseTaskListToWorkUnits extracts task list items from markdown and converts them to WorkUnits.
func parseTaskListToWorkUnits(body, parentID string) []*provider.WorkUnit {
	matches := taskListPattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	workUnits := make([]*provider.WorkUnit, 0, len(matches))
	for i, match := range matches {
		if len(match) < 3 {
			continue
		}

		completed := strings.ToLower(match[1]) == "x"
		title := strings.TrimSpace(match[2])

		status := provider.StatusOpen
		if completed {
			status = provider.StatusDone
		}

		// Generate a synthetic ID for the subtask
		subtaskID := fmt.Sprintf("%s:task-%d", parentID, i)

		wu := &provider.WorkUnit{
			ID:          subtaskID,
			ExternalID:  subtaskID,
			Provider:    ProviderName,
			Title:       title,
			Description: "",
			Status:      status,
			Priority:    provider.PriorityNormal,
			Labels:      []string{},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Source: provider.SourceInfo{
				Type:      ProviderName,
				Reference: parentID,
				SyncedAt:  time.Now(),
			},
			ExternalKey: fmt.Sprintf("task-%d", i),
			TaskType:    "subtask",
			Slug:        slug.Slugify(title, 50),
			Metadata: map[string]any{
				"parent_id":   parentID,
				"is_subtask":  true,
				"task_index":  i,
				"completed":   completed,
				"parsed_from": "task_list",
			},
		}

		workUnits = append(workUnits, wu)
	}

	return workUnits
}
