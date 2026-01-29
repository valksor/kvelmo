package wrike

import (
	"context"
	"fmt"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/slug"
)

// FetchProject implements provider.ProjectFetcher for Wrike.
// Fetches a folder/project with all descendant tasks recursively.
//
// Reference format:
//   - Permalink URL: "https://www.wrike.com/open.htm?id=123456"
//   - Folder ID: "IEAAJXXXXXXXX" (API ID)
//   - Numeric ID: "123456" (numeric ID from URL)
//
// Returns all tasks in the specified folder/project, including nested subtasks.
// Note: Does NOT traverse into child folders - only tasks within the specified folder.
func (p *Provider) FetchProject(ctx context.Context, reference string) (*provider.ProjectStructure, error) {
	ref, err := ParseReference(reference)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	var folderID string
	var folder *Folder

	switch {
	case ref.Permalink != "":
		// Resolve permalink to folder
		folder, err = p.client.GetFolderByPermalink(ctx, ref.Permalink)
		if err != nil {
			return nil, fmt.Errorf("resolve permalink: %w", err)
		}
		folderID = folder.ID
	case ref.TaskID != "":
		// Direct folder ID (API ID like "IEAAJXXXXXXXX")
		folderID = ref.TaskID
		// Try to fetch folder info for metadata
		// For API IDs, we don't have a direct GetFolder method, so we'll use minimal metadata
		folder = &Folder{
			ID:    folderID,
			Title: "Folder " + folderID,
			Scope: "WsFolder",
		}
	default:
		return nil, fmt.Errorf("%w: cannot determine folder ID from: %s", ErrInvalidReference, reference)
	}

	// Fetch all tasks in this folder
	tasks, err := p.fetchTasksRecursive(ctx, folderID, 0, make(map[string]bool))
	if err != nil {
		return nil, fmt.Errorf("fetch folder tasks: %w", err)
	}

	// Build permalink from folder ID for the URL
	permalink := folder.Permalink
	if permalink == "" {
		// Try to extract numeric ID and build permalink
		numericID := ExtractNumericID(folderID)
		if numericID != "" {
			permalink = BuildPermalinkURL(numericID)
		} else {
			permalink = "https://www.wrike.com/open.htm?id=" + folderID
		}
	}

	return &provider.ProjectStructure{
		ID:          folder.ID,
		Title:       folder.Title,
		Description: fmt.Sprintf("Wrike folder/project with %d tasks", len(tasks)),
		Source:      ProviderName,
		URL:         permalink,
		Tasks:       tasks,
		Metadata: map[string]any{
			"folder_id":    folderID,
			"project_type": "folder",
			"scope":        folder.Scope,
		},
	}, nil
}

// fetchTasksRecursive recursively fetches all tasks in a folder tree.
// Processes tasks at the given depth and tracks visited tasks to avoid duplicates.
func (p *Provider) fetchTasksRecursive(ctx context.Context, folderID string, depth int, visited map[string]bool) ([]*provider.ProjectTask, error) {
	const maxDepth = 10 // Prevent infinite recursion

	if depth > maxDepth {
		return nil, nil
	}

	// Get all tasks in this folder
	wrikeTasks, err := p.client.GetTasksInFolder(ctx, folderID)
	if err != nil {
		return nil, fmt.Errorf("get tasks in folder: %w", err)
	}

	var results []*provider.ProjectTask

	for _, task := range wrikeTasks {
		// Skip already visited tasks
		if visited[task.ID] {
			continue
		}
		visited[task.ID] = true

		// Convert Wrike Task to WorkUnit
		workUnit := &provider.WorkUnit{
			ID:          task.ID,
			ExternalID:  task.ID,
			Provider:    ProviderName,
			Title:       task.Title,
			Description: task.Description,
			Status:      mapStatus(task.Status),
			Priority:    mapPriority(task.Priority),
			Labels:      task.Tags,
			Assignees:   nil, // Wrike assignee info would need separate fetch
			Subtasks:    task.SubTaskIDs,
			Metadata: map[string]any{
				"permalink": task.Permalink,
				"api_id":    task.ID,
				"folder_id": folderID,
				"depth":     depth,
			},
			CreatedAt: task.CreatedDate,
			UpdatedAt: task.UpdatedDate,
			Source: provider.SourceInfo{
				Type:      ProviderName,
				Reference: task.Permalink,
				SyncedAt:  time.Now(),
			},
			ExternalKey: task.ID,
			TaskType:    "task",
			Slug:        slug.Slugify(task.Title, 50),
		}

		// Create ProjectTask
		pt := &provider.ProjectTask{
			WorkUnit: workUnit,
			Depth:    depth,
			Position: len(results),
		}
		results = append(results, pt)

		// Recursively fetch subtasks
		if len(task.SubTaskIDs) > 0 {
			subtasks, err := p.client.GetTasks(ctx, task.SubTaskIDs)
			if err == nil {
				// Process each subtask recursively
				for _, subtask := range subtasks {
					subPT := &provider.ProjectTask{
						WorkUnit: convertTaskToWorkUnit(subtask, folderID, depth+1),
						ParentID: task.ID,
						Depth:    depth + 1,
						Position: len(results),
					}
					results = append(results, subPT)
					visited[subtask.ID] = true

					// Recurse into sub-subtasks
					nested, err := p.fetchTasksRecursiveFromTasks(ctx, subtask.SubTaskIDs, depth+2, visited)
					if err == nil {
						results = append(results, nested...)
					}
				}
			}
		}
	}

	return results, nil
}

// fetchTasksRecursiveFromTasks fetches tasks starting from a list of task IDs.
// Helper function for recursing into subtask hierarchies.
func (p *Provider) fetchTasksRecursiveFromTasks(ctx context.Context, taskIDs []string, depth int, visited map[string]bool) ([]*provider.ProjectTask, error) {
	if depth > 10 { // Max recursion depth
		return nil, nil
	}

	if len(taskIDs) == 0 {
		return nil, nil
	}

	tasks, err := p.client.GetTasks(ctx, taskIDs)
	if err != nil {
		return nil, fmt.Errorf("get tasks: %w", err)
	}

	var results []*provider.ProjectTask

	for _, task := range tasks {
		if visited[task.ID] {
			continue
		}
		visited[task.ID] = true

		// Convert to WorkUnit - note we need to figure out the folder ID context
		// For subtasks, we'll use empty folder ID since we're getting them by ID
		workUnit := convertTaskToWorkUnit(task, "", depth+1)

		pt := &provider.ProjectTask{
			WorkUnit: workUnit,
			Depth:    depth + 1,
			Position: len(results),
		}
		results = append(results, pt)

		// Recurse into subtasks
		if len(task.SubTaskIDs) > 0 {
			nested, err := p.fetchTasksRecursiveFromTasks(ctx, task.SubTaskIDs, depth+2, visited)
			if err == nil {
				results = append(results, nested...)
			}
		}
	}

	return results, nil
}

// convertTaskToWorkUnit converts a Wrike Task to a provider.WorkUnit.
func convertTaskToWorkUnit(task Task, folderID string, depth int) *provider.WorkUnit {
	return &provider.WorkUnit{
		ID:          task.ID,
		ExternalID:  task.ID,
		Provider:    ProviderName,
		Title:       task.Title,
		Description: task.Description,
		Status:      mapStatus(task.Status),
		Priority:    mapPriority(task.Priority),
		Labels:      task.Tags,
		Assignees:   nil,
		Subtasks:    task.SubTaskIDs,
		Metadata: map[string]any{
			"permalink": task.Permalink,
			"api_id":    task.ID,
			"folder_id": folderID,
			"depth":     depth,
		},
		CreatedAt: task.CreatedDate,
		UpdatedAt: task.UpdatedDate,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: task.Permalink,
			SyncedAt:  time.Now(),
		},
		ExternalKey: task.ID,
		TaskType:    "task",
		Slug:        slug.Slugify(task.Title, 50),
	}
}
