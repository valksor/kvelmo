package youtrack

import (
	"context"
	"errors"
	"fmt"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// CreateWorkUnit creates a new YouTrack issue.
func (p *Provider) CreateWorkUnit(ctx context.Context, opts provider.CreateWorkUnitOptions) (*provider.WorkUnit, error) {
	// Map priority and type to custom fields
	customFields := p.buildCustomFields(opts)

	// Get project ID from custom fields, config, or return error
	var projectID string
	switch {
	case opts.CustomFields != nil && opts.CustomFields["project_id"] != nil:
		if pid, ok := opts.CustomFields["project_id"].(string); ok && pid != "" {
			projectID = pid
		}
	case p.config.DefaultProject != "":
		projectID = p.config.DefaultProject
	}
	if projectID == "" {
		return nil, errors.New("project_id required: set default_project in config or pass project_id in custom fields")
	}

	issue, err := p.client.CreateIssue(ctx, projectID, opts.Title, opts.Description, customFields)
	if err != nil {
		return nil, fmt.Errorf("create issue: %w", err)
	}

	return p.issueToWorkUnit(issue, nil, nil), nil
}

// buildCustomFields builds custom fields array from CreateWorkUnitOptions.
func (p *Provider) buildCustomFields(opts provider.CreateWorkUnitOptions) []map[string]interface{} {
	var fields []map[string]interface{}

	// Priority
	fields = append(fields, map[string]interface{}{
		"name":  "Priority",
		"$type": "SingleEnumIssueCustomField",
		"value": map[string]string{"name": priorityToYouTrack(opts.Priority)},
	})

	// Type - can be inferred or set to default
	taskType := "Task"
	if opts.CustomFields != nil {
		if t, ok := opts.CustomFields["type"].(string); ok {
			taskType = t
		}
	}
	fields = append(fields, map[string]interface{}{
		"name":  "Type",
		"$type": "SingleEnumIssueCustomField",
		"value": map[string]string{"name": taskType},
	})

	return fields
}

// priorityToYouTrack maps provider Priority to YouTrack priority name.
func priorityToYouTrack(p provider.Priority) string {
	switch p {
	case provider.PriorityCritical:
		return "Critical"
	case provider.PriorityHigh:
		return "High"
	case provider.PriorityNormal:
		return "Normal"
	case provider.PriorityLow:
		return "Low"
	}

	return "Normal"
}
