package youtrack

import (
	"context"
	"errors"
	"fmt"

	"github.com/valksor/go-toolkit/workunit"
)

// ErrNotASubtask is returned when a work unit is not a subtask.
var ErrNotASubtask = errors.New("not a subtask")

// FetchParent implements the workunit.ParentFetcher interface.
// It retrieves the parent issue for a YouTrack subtask.
func (p *Provider) FetchParent(ctx context.Context, workUnitID string) (*workunit.WorkUnit, error) {
	// Get the issue to check if it has a parent
	issue, err := p.client.GetIssue(ctx, workUnitID)
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}

	// Check if this issue has a parent (is a subtask)
	// YouTrack stores parent info in the issue's parent field
	if issue.Parent == nil || issue.Parent.ID == "" {
		// Not a subtask
		return nil, ErrNotASubtask
	}

	// Fetch the parent issue
	parentIssue, err := p.client.GetIssue(ctx, issue.Parent.IDReadable)
	if err != nil {
		return nil, fmt.Errorf("get parent issue: %w", err)
	}

	// Fetch comments and attachments for the parent
	comments, _ := p.client.GetComments(ctx, issue.Parent.IDReadable)
	attachments, _ := p.client.GetAttachments(ctx, issue.Parent.IDReadable)

	// Convert to WorkUnit
	wu := p.issueToWorkUnit(parentIssue, comments, attachments)

	return wu, nil
}

// FetchSubtasks implements the workunit.SubtaskFetcher interface.
// It retrieves subtasks for a given YouTrack issue.
func (p *Provider) FetchSubtasks(ctx context.Context, workUnitID string) ([]*workunit.WorkUnit, error) {
	// First, get the parent issue to get subtask links
	issue, err := p.client.GetIssue(ctx, workUnitID)
	if err != nil {
		return nil, fmt.Errorf("get parent issue: %w", err)
	}

	if len(issue.Subtasks) == 0 {
		return nil, nil
	}

	// Fetch each subtask as a full issue
	result := make([]*workunit.WorkUnit, 0, len(issue.Subtasks))
	for _, subtaskLink := range issue.Subtasks {
		// Fetch the subtask issue by its readable ID
		subtaskIssue, err := p.client.GetIssue(ctx, subtaskLink.IDReadable)
		if err != nil {
			// Continue with other subtasks if one fails
			continue
		}

		// Fetch comments and attachments for the subtask
		comments, _ := p.client.GetComments(ctx, subtaskLink.IDReadable)
		attachments, _ := p.client.GetAttachments(ctx, subtaskLink.IDReadable)

		// Convert to WorkUnit using the existing method
		wu := p.issueToWorkUnit(subtaskIssue, comments, attachments)

		// Override task type to indicate it's a subtask
		wu.TaskType = "subtask"

		// Add parent reference to metadata
		wu.Metadata["parent_id"] = workUnitID
		wu.Metadata["is_subtask"] = true

		result = append(result, wu)
	}

	return result, nil
}
