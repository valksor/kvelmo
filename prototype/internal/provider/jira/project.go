package jira

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/slug"
)

// FetchProject implements provider.ProjectFetcher for Jira.
// Fetches an epic with all its child issues/stories recursively.
//
// Reference format:
//   - Epic Key: "PROJ-123" (issue key)
//   - With scheme: "jira:PROJ-123" or "j:PROJ-123"
//   - Full URL: "https://domain.atlassian.net/browse/PROJ-123"
//
// Returns the epic issue plus all stories/issues in the epic, including subtasks.
// Note: The epic itself is included as depth=0, with stories at depth=1.
func (p *Provider) FetchProject(ctx context.Context, reference string) (*provider.ProjectStructure, error) {
	ref, err := ParseReference(reference)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// Update base URL if detected from reference
	if ref.BaseURL != "" && p.baseURL == "" {
		p.baseURL = ref.BaseURL
		p.client.SetBaseURL(ref.BaseURL)
	}

	// Fetch the epic issue first
	epic, err := p.client.GetIssue(ctx, ref.IssueKey)
	if err != nil {
		return nil, fmt.Errorf("fetch epic: %w", err)
	}

	// Check if this is actually an epic-type issue
	if !isEpicIssue(epic) {
		return nil, fmt.Errorf("issue %s is not an epic (type: %s)", ref.IssueKey, getIssueTypeName(epic))
	}

	// Fetch all issues in this epic using JQL
	// syntax for finding issues in an epic varies by Jira instance
	// Try multiple approaches
	issues := p.fetchEpicIssues(ctx, ref.IssueKey)

	// Convert epic and its issues to ProjectTasks
	tasks := make([]*provider.ProjectTask, 0, 1+len(issues))

	// Add the epic itself at depth 0
	epicWorkUnit := convertIssueToWorkUnit(epic, "", 0)
	tasks = append(tasks, &provider.ProjectTask{
		WorkUnit: epicWorkUnit,
		Depth:    0,
		Position: 0,
	})

	// Build a map of issue keys to their IDs for parent reference
	issueKeyToID := make(map[string]string)
	issueKeyToID[epic.Key] = epic.ID

	// Process all issues in the epic
	for _, issue := range issues {
		// Skip the epic itself (already added)
		if issue.Key == ref.IssueKey {
			continue
		}

		issueKeyToID[issue.Key] = issue.ID

		// Determine parent ID
		var parentID string
		if issue.Fields.Parent != nil {
			// This is a subtask, parent is the parent issue
			parentID = issue.Fields.Parent.ID
		} else {
			// This is a story/task in the epic, parent is the epic
			parentID = epic.ID
		}

		wu := convertIssueToWorkUnit(issue, epic.ID, 1)
		pt := &provider.ProjectTask{
			WorkUnit: wu,
			ParentID: parentID,
			Depth:    1,
			Position: len(tasks),
		}
		tasks = append(tasks, pt)

		// Fetch subtasks for this issue if it has any
		if len(issue.Fields.Subtasks) > 0 {
			for _, subtask := range issue.Fields.Subtasks {
				// Fetch full subtask details
				fullSubtask, err := p.client.GetIssue(ctx, subtask.Key)
				if err != nil {
					continue // Skip if we can't fetch
				}

				subtaskWU := convertIssueToWorkUnit(fullSubtask, issue.ID, 2)
				subtaskPT := &provider.ProjectTask{
					WorkUnit: subtaskWU,
					ParentID: issue.ID,
					Depth:    2,
					Position: len(tasks),
				}
				tasks = append(tasks, subtaskPT)
			}
		}
	}

	// Build the project URL
	url := ref.URL
	if url == "" && p.baseURL != "" {
		url = fmt.Sprintf("%s/browse/%s", strings.TrimSuffix(p.baseURL, "/"), epic.Key)
	} else if url == "" {
		url = "https://Atlassian.net/browse/" + epic.Key
	}

	return &provider.ProjectStructure{
		ID:          epic.ID,
		Title:       "Epic: " + epic.Fields.Summary,
		Description: epic.Fields.Description,
		Source:      ProviderName,
		URL:         url,
		Tasks:       tasks,
		Metadata: map[string]any{
			"epic_key":     epic.Key,
			"epic_name":    epic.Fields.Summary,
			"issue_count":  len(issues),
			"project_key":  getProjectKey(epic),
			"project_name": getProjectName(epic),
		},
	}, nil
}

// fetchEpicIssues fetches all issues belonging to an epic using JQL.
func (p *Provider) fetchEpicIssues(ctx context.Context, epicKey string) []*Issue {
	// Try multiple JQL approaches for finding epic issues
	// The exact syntax depends on Jira version and configuration

	jqlQueries := []string{
		// Jira Cloud with Portfolio/Advanced Roadmaps
		"\"Epic Link\" = " + epicKey,
		// Alternative field name
		fmt.Sprintf(`issueFunction in issuesInEpic("%s")`, epicKey),
		// Another common pattern
		"epic = " + epicKey,
		// Project-specific query
		fmt.Sprintf(`"Epic Link" = %s ORDER BY rank`, epicKey),
	}

	var allIssues []*Issue
	seenKeys := make(map[string]bool)

	for _, jql := range jqlQueries {
		issues, _, err := p.client.ListIssues(ctx, jql, 0, 1000)
		if err != nil {
			continue // Try next query
		}

		// Add new issues we haven't seen yet
		for _, issue := range issues {
			if !seenKeys[issue.Key] {
				allIssues = append(allIssues, issue)
				seenKeys[issue.Key] = true
			}
		}

		// If we found issues, stop trying other queries
		if len(allIssues) > 1 { // More than just the epic itself
			break
		}
	}

	// If no issues found with epic-specific queries, return empty list
	if len(allIssues) == 0 {
		return []*Issue{}
	}

	return allIssues
}

// isEpicIssue checks if an issue is an epic type.
func isEpicIssue(issue *Issue) bool {
	if issue.Fields.Issuetype == nil {
		return false
	}

	issueType := strings.ToLower(issue.Fields.Issuetype.Name)

	return strings.Contains(issueType, "epic")
}

// getIssueTypeName returns the issue type name.
func getIssueTypeName(issue *Issue) string {
	if issue.Fields.Issuetype == nil {
		return "Unknown"
	}

	return issue.Fields.Issuetype.Name
}

// getProjectKey extracts the project key from an issue.
func getProjectKey(issue *Issue) string {
	if issue.Fields.Project == nil {
		return ""
	}

	return issue.Fields.Project.Key
}

// getProjectName extracts the project name from an issue.
func getProjectName(issue *Issue) string {
	if issue.Fields.Project == nil {
		return ""
	}

	return issue.Fields.Project.Name
}

// convertIssueToWorkUnit converts a Jira Issue to a provider.WorkUnit.
func convertIssueToWorkUnit(issue *Issue, epicID string, depth int) *provider.WorkUnit {
	// Build metadata
	metadata := buildMetadata(issue)
	metadata["depth"] = depth
	if epicID != "" {
		metadata["epic_id"] = epicID
	}

	return &provider.WorkUnit{
		ID:          issue.ID,
		ExternalID:  issue.Key,
		Provider:    ProviderName,
		Title:       issue.Fields.Summary,
		Description: issue.Fields.Description,
		Status:      mapJiraStatus(issue.Fields.Status.Name),
		Priority:    mapJiraPriority(issue.Fields.Priority),
		Labels:      issue.Fields.Labels,
		Assignees:   mapAssignees(issue.Fields.Assignee),
		CreatedAt:   issue.Fields.Created,
		UpdatedAt:   issue.Fields.Updated,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: issue.Key,
			SyncedAt:  time.Now(),
		},
		ExternalKey: issue.Key,
		TaskType:    inferTaskTypeFromLabels(issue.Fields.Labels),
		Slug:        slug.Slugify(issue.Fields.Summary, 50),
		Metadata:    metadata,
		Subtasks:    extractSubtaskKeys(issue),
	}
}

// extractSubtaskKeys extracts subtask keys from an issue.
func extractSubtaskKeys(issue *Issue) []string {
	if len(issue.Fields.Subtasks) == 0 {
		return nil
	}

	keys := make([]string, len(issue.Fields.Subtasks))
	for i, st := range issue.Fields.Subtasks {
		keys[i] = st.Key
	}

	return keys
}
