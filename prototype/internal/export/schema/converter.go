// Package schema provides JSON Schema-based extraction for project plans.
package schema

import (
	"strings"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// ToStorageTasks converts a schema.ParsedPlan to storage tasks and returns the components.
// This avoids importing the export package which would create a circular dependency.
func ToStorageTasks(plan *ParsedPlan) ([]*storage.QueuedTask, []string, []string) {
	if plan == nil {
		return nil, nil, nil
	}

	return toStorageTasks(plan.Tasks), plan.Questions, plan.Blockers
}

// toStorageTasks converts schema tasks to storage tasks.
func toStorageTasks(schemaTasks []*Task) []*storage.QueuedTask {
	if schemaTasks == nil {
		return nil
	}

	tasks := make([]*storage.QueuedTask, 0, len(schemaTasks))
	for _, st := range schemaTasks {
		if st == nil {
			continue
		}

		tasks = append(tasks, &storage.QueuedTask{
			ID:          st.ID,
			Title:       st.Title,
			Description: st.Description,
			Status:      parseStatus(st.Status),
			Priority:    st.Priority,
			Labels:      st.Labels,
			DependsOn:   st.DependsOn,
			Assignee:    st.Assignee,
		})
	}

	return tasks
}

// parseStatus converts a string status to storage.TaskStatus.
// It handles case-insensitive input and defaults to pending.
func parseStatus(status string) storage.TaskStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ready":
		return storage.TaskStatusReady
	case "blocked":
		return storage.TaskStatusBlocked
	case "submitted":
		return storage.TaskStatusSubmitted
	case "pending", "":
		return storage.TaskStatusPending
	default:
		// Default to pending for unknown statuses
		return storage.TaskStatusPending
	}
}
