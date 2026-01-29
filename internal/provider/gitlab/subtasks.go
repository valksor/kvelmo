package gitlab

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
// It retrieves the parent issue for a GitLab subtask (task list item).
//
// GitLab subtasks are task list items within an issue description.
// The subtask ID format is: {project_path}#123-task-{n}.
func (p *Provider) FetchParent(ctx context.Context, workUnitID string) (*provider.WorkUnit, error) {
	// Check if this is a subtask (has "-task-" in the ID)
	if !strings.Contains(workUnitID, "-task-") {
		// Regular issue, not a subtask
		return nil, ErrNotASubtask
	}

	// Parse the subtask ID to extract parent issue
	// Format: {project_path}#123-task-{n}
	taskSplit := strings.Split(workUnitID, "-task-")
	if len(taskSplit) < 2 {
		return nil, fmt.Errorf("%w: invalid subtask ID format: %s", ErrInvalidReference, workUnitID)
	}

	// Extract the issue reference before "-task-"
	issueRef := taskSplit[0]

	// Parse the issue reference to get issue IID and project
	ref, err := ParseReference(issueRef)
	if err != nil {
		return nil, fmt.Errorf("parse issue reference: %w", err)
	}

	// Determine project
	projectPath := ref.ProjectPath
	if projectPath == "" {
		projectPath = p.projectPath
		if projectPath == "" {
			projectPath = p.config.ProjectPath
		}
	}

	// Set up client with correct project
	if ref.ProjectID > 0 {
		p.client.SetProjectID(ref.ProjectID)
	} else if projectPath != "" {
		p.client.SetProjectPath(projectPath)
	} else {
		return nil, ErrProjectNotConfigured
	}

	// Fetch the parent issue
	issue, err := p.client.GetIssue(ctx, ref.IssueIID)
	if err != nil {
		return nil, fmt.Errorf("get parent issue: %w", err)
	}

	// Determine display project path
	displayProject := projectPath
	if displayProject == "" && ref.ProjectID > 0 {
		displayProject = strconv.FormatInt(ref.ProjectID, 10)
	} else if displayProject == "" && issue.ProjectID != 0 {
		displayProject = strconv.FormatInt(issue.ProjectID, 10)
	}

	var createdAt, updatedAt time.Time
	if issue.CreatedAt != nil {
		createdAt = *issue.CreatedAt
	}
	if issue.UpdatedAt != nil {
		updatedAt = *issue.UpdatedAt
	}

	// Build parent WorkUnit
	parentID := fmt.Sprintf("%s#%d", displayProject, ref.IssueIID)

	return &provider.WorkUnit{
		ID:          parentID,
		ExternalID:  strconv.FormatInt(issue.ID, 10),
		ExternalKey: strconv.FormatInt(issue.IID, 10),
		Provider:    ProviderName,
		Title:       issue.Title,
		Description: issue.Description,
		Status:      mapGitLabState(issue.State),
		Priority:    provider.PriorityNormal,
		Labels:      issue.Labels,
		Assignees:   mapAssignees(issue.Assignees),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: parentID,
			SyncedAt:  time.Now(),
		},
		Metadata: map[string]any{
			"issue_iid":    ref.IssueIID,
			"project_path": displayProject,
			"project_id":   issue.ProjectID,
			"state":        issue.State,
		},
	}, nil
}

// FetchSubtasks implements the provider.SubtaskFetcher interface.
// It parses task list items from the GitLab issue description.
func (p *Provider) FetchSubtasks(ctx context.Context, workUnitID string) ([]*provider.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// Determine project
	projectPath := ref.ProjectPath
	if projectPath == "" {
		projectPath = p.projectPath
		if projectPath == "" {
			projectPath = p.config.ProjectPath
		}
	}

	// Set up client with correct project
	if ref.ProjectID > 0 {
		p.client.SetProjectID(ref.ProjectID)
	} else if projectPath != "" {
		p.client.SetProjectPath(projectPath)
	} else {
		return nil, ErrProjectNotConfigured
	}

	// Fetch the issue
	issue, err := p.client.GetIssue(ctx, ref.IssueIID)
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}

	// Parse task list from issue description
	tasks := ParseTaskList(issue.Description)
	if len(tasks) == 0 {
		return nil, nil
	}

	// Determine display project path
	displayProject := projectPath
	if displayProject == "" && ref.ProjectID > 0 {
		displayProject = strconv.FormatInt(ref.ProjectID, 10)
	} else if displayProject == "" && issue.ProjectID != 0 {
		displayProject = strconv.FormatInt(issue.ProjectID, 10)
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
		taskID := fmt.Sprintf("%s#%d-task-%d", displayProject, ref.IssueIID, i+1)

		var createdAt, updatedAt time.Time
		if issue.CreatedAt != nil {
			createdAt = *issue.CreatedAt
		}
		if issue.UpdatedAt != nil {
			updatedAt = *issue.UpdatedAt
		}

		wu := &provider.WorkUnit{
			ID:          taskID,
			ExternalID:  taskID,
			ExternalKey: fmt.Sprintf("%d-task-%d", ref.IssueIID, i+1),
			Provider:    ProviderName,
			Title:       task.Text,
			Status:      status,
			Priority:    provider.PriorityNormal,
			TaskType:    "subtask",
			Slug:        slug.Slugify(task.Text, 50),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
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
				"issue_iid":    ref.IssueIID,
				"project_path": displayProject,
				"project_id":   issue.ProjectID,
			},
		}

		result = append(result, wu)
	}

	return result, nil
}
