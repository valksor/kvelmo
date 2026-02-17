package azuredevops

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/valksor/go-toolkit/workunit"
)

// ErrNotASubtask is returned when a work unit is not a subtask.
var ErrNotASubtask = errors.New("not a subtask")

// FetchParent implements the workunit.ParentFetcher interface.
// It retrieves the parent work item for an Azure DevOps child work item.
func (p *Provider) FetchParent(ctx context.Context, workUnitID string) (*workunit.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// Override org/project if specified in reference
	if ref.Organization != "" && ref.Project != "" {
		p.client.SetOrganization(ref.Organization)
		p.client.SetProject(ref.Project)
	}

	// Get the work item to check if it has a parent
	workItem, err := p.client.GetWorkItem(ctx, ref.WorkItemID)
	if err != nil {
		return nil, fmt.Errorf("get work item: %w", err)
	}

	// Check if this work item has a parent (is a child)
	// Look for reverse hierarchy link
	var parentID int
	for _, rel := range workItem.Relations {
		if rel.Rel == "System.LinkTypes.Hierarchy-Reverse" && rel.URL != "" {
			// Extract parent ID from URL
			// format: https://dev.azure.com/{org}/{project}/_workitems/{id}
			parts := strings.Split(rel.URL, "/_workitems/")
			if len(parts) > 1 {
				idStr := strings.Split(parts[1], "/")[0]
				parentID, err = strconv.Atoi(idStr)
				if err != nil {
					continue
				}

				break
			}
		}
	}

	if parentID == 0 {
		// Not a child work item
		return nil, ErrNotASubtask
	}

	// Fetch the parent work item
	parentWorkItem, err := p.client.GetWorkItem(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("get parent work item: %w", err)
	}

	// Convert to WorkUnit
	wu := p.workItemToWorkUnit(parentWorkItem)

	return wu, nil
}

// FetchSubtasks implements the workunit.SubtaskFetcher interface.
// It retrieves child work items for a given work item.
func (p *Provider) FetchSubtasks(ctx context.Context, workUnitID string) ([]*workunit.WorkUnit, error) {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// Override org/project if specified in reference
	if ref.Organization != "" && ref.Project != "" {
		p.client.SetOrganization(ref.Organization)
		p.client.SetProject(ref.Project)
	}

	// Query for child work items using WIQL
	wiql := fmt.Sprintf(`
		SELECT [System.Id], [System.Title], [System.State]
		FROM WorkItemLinks
		WHERE (
			[Source].[System.Id] = %d
			AND [System.Links.LinkType] = 'System.LinkTypes.Hierarchy-Forward'
		)
		MODE (MustContain)
	`, ref.WorkItemID)

	// Execute tree query
	childIDs, err := p.client.QueryWorkItemLinks(ctx, wiql)
	if err != nil {
		// If tree query fails, try to get from relations
		return p.fetchSubtasksFromRelations(ctx, ref.WorkItemID)
	}

	if len(childIDs) == 0 {
		return nil, nil
	}

	// Fetch full work items
	workItems, err := p.client.GetWorkItems(ctx, childIDs)
	if err != nil {
		return nil, fmt.Errorf("get child work items: %w", err)
	}

	// Convert to WorkUnits
	result := make([]*workunit.WorkUnit, 0, len(workItems))
	for _, wi := range workItems {
		wu := p.workItemToWorkUnit(&wi)
		// Add parent reference
		wu.Metadata["parent_id"] = workUnitID
		wu.Metadata["is_subtask"] = true
		wu.TaskType = "subtask"
		result = append(result, wu)
	}

	return result, nil
}

// fetchSubtasksFromRelations fetches child work items from the relations array.
func (p *Provider) fetchSubtasksFromRelations(ctx context.Context, workItemID int) ([]*workunit.WorkUnit, error) {
	// Get work item with relations
	workItem, err := p.client.GetWorkItem(ctx, workItemID)
	if err != nil {
		return nil, fmt.Errorf("get work item: %w", err)
	}

	// Extract child IDs from relations
	var childIDs []int
	for _, rel := range workItem.Relations {
		if rel.Rel == "System.LinkTypes.Hierarchy-Forward" {
			// Extract ID from URL
			// format: https://dev.azure.com/{org}/{project}/_apis/wit/workItems/{id}
			parts := strings.Split(rel.URL, "/")
			if len(parts) > 0 {
				idStr := parts[len(parts)-1]
				if id, err := strconv.Atoi(idStr); err == nil {
					childIDs = append(childIDs, id)
				}
			}
		}
	}

	if len(childIDs) == 0 {
		return nil, nil
	}

	// Fetch full work items
	workItems, err := p.client.GetWorkItems(ctx, childIDs)
	if err != nil {
		return nil, fmt.Errorf("get child work items: %w", err)
	}

	// Convert to WorkUnits
	parentID := strconv.Itoa(workItemID)
	result := make([]*workunit.WorkUnit, 0, len(workItems))
	for _, wi := range workItems {
		wu := p.workItemToWorkUnit(&wi)
		wu.Metadata["parent_id"] = parentID
		wu.Metadata["is_subtask"] = true
		wu.TaskType = "subtask"
		result = append(result, wu)
	}

	return result, nil
}
