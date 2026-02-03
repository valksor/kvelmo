package wrike

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/slug"
)

const (
	// ProviderName is the name of the Wrike provider.
	ProviderName = "wrike"
)

// Provider implements the Wrike task provider.
type Provider struct {
	client       *Client
	customFields map[string]string // Cache: custom field ID -> title
}

// New creates a new Wrike provider instance.
func New(ctx context.Context, cfg provider.Config) (any, error) {
	token := cfg.GetString("token")
	host := cfg.GetString("host")
	spaceID := cfg.GetString("space_id")
	folderID := cfg.GetString("folder_id")
	projectID := cfg.GetString("project_id")

	// Try to resolve token from env if not provided
	if token == "" {
		resolvedToken, err := ResolveToken("")
		if err != nil {
			return nil, err
		}
		token = resolvedToken
	}

	// Create a temporary client for ID resolution
	tempClient := NewClientWithConfig(ClientConfig{
		Token: token,
		Host:  host,
	})

	// Resolve numeric project ID to API ID (primary target for task creation)
	if projectID != "" && isNumericID(projectID) {
		resolved, err := tempClient.GetFolderByPermalink(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("resolve project ID %s: %w", projectID, err)
		}
		entityType := "folder"
		if resolved.Project != nil {
			entityType = "project"
		}
		slog.Info("Resolved Wrike project",
			"numeric_id", projectID,
			"api_id", resolved.ID,
			"title", resolved.Title,
			"type", entityType,
		)
		projectID = resolved.ID
	}

	// Resolve numeric folder ID to API ID
	if folderID != "" && isNumericID(folderID) {
		resolved, err := tempClient.GetFolderByPermalink(ctx, folderID)
		if err != nil {
			return nil, fmt.Errorf("resolve folder ID %s: %w", folderID, err)
		}
		entityType := "folder"
		if resolved.Project != nil {
			entityType = "project"
		}
		slog.Info("Resolved Wrike folder",
			"numeric_id", folderID,
			"api_id", resolved.ID,
			"title", resolved.Title,
			"type", entityType,
		)
		folderID = resolved.ID
	}

	// Determine target for task creation: project > folder
	targetID := projectID
	if targetID == "" {
		targetID = folderID
	}

	return &Provider{
		client: NewClientWithConfig(ClientConfig{
			Token:    token,
			Host:     host,
			FolderID: targetID,
			SpaceID:  spaceID,
		}),
	}, nil
}

// isNumericID returns true if the ID contains only digits (URL-style numeric ID).
func isNumericID(id string) bool {
	return numericIDPattern.MatchString(id)
}

// getCustomFieldName returns the human-readable name for a custom field ID.
// Falls back to the ID if the name cannot be resolved.
func (p *Provider) getCustomFieldName(ctx context.Context, fieldID string) string {
	// Check cache first
	if p.customFields != nil {
		if name, ok := p.customFields[fieldID]; ok {
			return name
		}
	}

	// Fetch definitions if not cached
	if p.customFields == nil {
		p.customFields = make(map[string]string)
		defs, err := p.client.GetCustomFields(ctx)
		if err == nil {
			for _, def := range defs {
				p.customFields[def.ID] = def.Title
			}
		}
	}

	// Look up from cache
	if name, ok := p.customFields[fieldID]; ok {
		return name
	}

	// Fall back to ID if name not found
	return fieldID
}

// Match checks if the input matches a Wrike reference.
func (p *Provider) Match(input string) bool {
	input = strings.TrimSpace(input)

	return strings.HasPrefix(input, "wrike:") ||
		strings.HasPrefix(input, "wk:") ||
		permalinkPattern.MatchString(input) ||
		apiIDPattern.MatchString(input) ||
		numericIDPattern.MatchString(input)
}

// Parse extracts the task ID from a Wrike reference.
func (p *Provider) Parse(input string) (string, error) {
	input = strings.TrimSpace(input)

	// Strip scheme prefix if present
	schemeStripped := strings.TrimPrefix(input, "wrike:")
	schemeStripped = strings.TrimPrefix(schemeStripped, "wk:")

	// Check for permalink - extract numeric ID (use schemeStripped to handle wrike:https://... format)
	if matches := permalinkPattern.FindStringSubmatch(schemeStripped); matches != nil {
		return matches[1], nil
	}

	// Use scheme-stripped version for remaining checks
	taskID := schemeStripped

	// Validate that it's a valid ID format
	if !apiIDPattern.MatchString(taskID) && !numericIDPattern.MatchString(taskID) {
		return "", fmt.Errorf("%w: invalid Wrike reference: %s", ErrInvalidReference, input)
	}

	return taskID, nil
}

// Fetch retrieves a task from Wrike and converts it to a WorkUnit.
func (p *Provider) Fetch(ctx context.Context, id string) (*provider.WorkUnit, error) {
	// Parse the reference to determine ID type
	ref, err := ParseReference(id)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	var task *Task

	// Strategy: Determine how to fetch based on ID type
	switch {
	case ref.Permalink != "":
		// Full permalink URL provided - use permalink query parameter
		task, err = p.client.GetTaskByPermalinkParam(ctx, ref.Permalink)
	case apiIDPattern.MatchString(ref.TaskID):
		// API ID format (IEAAJXXXX) - use direct task endpoint
		task, err = p.client.GetTask(ctx, ref.TaskID)
	case numericIDPattern.MatchString(ref.TaskID):
		// Numeric ID format - construct permalink and use permalink query parameter
		permalink := BuildPermalinkURL(ref.TaskID)
		task, err = p.client.GetTaskByPermalinkParam(ctx, permalink)
	default:
		return nil, fmt.Errorf("%w: unrecognized task ID format: %s", ErrInvalidReference, ref.TaskID)
	}

	if err != nil {
		return nil, fmt.Errorf("fetch task: %w", err)
	}

	// Fetch comments
	comments, err := p.client.GetComments(ctx, task.ID)
	if err != nil {
		// Log error but continue - comments are optional
		comments = nil
	}

	// Fetch attachments
	attachments, err := p.client.GetAttachments(ctx, task.ID)
	if err != nil {
		// Log error but continue - attachments are optional
		attachments = nil
	}

	// Fetch subtasks
	var subtaskInfos []SubtaskInfo
	if len(task.SubTaskIDs) > 0 {
		var err error
		subtaskInfos, err = p.fetchSubtasks(ctx, task.SubTaskIDs, 0)
		_ = err // Subtasks are optional, ignore fetch errors
	}

	// Fetch parent task content if this is a subtask
	var parentInfo *ParentTaskInfo
	if len(task.SuperTaskIDs) > 0 {
		var err error
		parentInfo, err = p.fetchParentTask(ctx, task.ID, task.SuperTaskIDs)
		_ = err // Parent is optional, ignore fetch errors
	}

	// Extract numeric ID from permalink for ExternalKey
	numericID := ExtractNumericID(task.Permalink)
	if numericID == "" {
		numericID = task.ID
	}

	// Build WorkUnit
	wu := &provider.WorkUnit{
		ID:          numericID,
		ExternalID:  task.ID,
		Provider:    ProviderName,
		Title:       task.Title,
		Description: task.Description,
		Status:      mapStatus(task.Status),
		Priority:    mapPriority(task.Priority),
		Labels:      []string{},
		Assignees:   []provider.Person{},
		Comments:    mapComments(comments),
		Attachments: mapAttachments(attachments),
		Subtasks:    task.SubTaskIDs,
		Metadata:    buildMetadata(task, subtaskInfos),
		CreatedAt:   task.CreatedDate,
		UpdatedAt:   task.UpdatedDate,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: task.Permalink,
			SyncedAt:  time.Now(),
		},
		ExternalKey: numericID,
		TaskType:    "task",
		Slug:        slug.Slugify(task.Title, 50),
	}

	// Add parent task content to metadata if available
	if parentInfo != nil {
		wu.Metadata["parent_task"] = map[string]any{
			"title":       parentInfo.Title,
			"description": parentInfo.Description,
			"status":      parentInfo.Status,
			"permalink":   parentInfo.Permalink,
		}
	}

	return wu, nil
}

// Snapshot captures the task content from Wrike.
func (p *Provider) Snapshot(ctx context.Context, id string) (*provider.Snapshot, error) {
	slog.Debug("wrike snapshot called", "id", id)

	ref, err := ParseReference(id)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	slog.Debug("wrike snapshot parsed reference", "taskID", ref.TaskID, "permalink", ref.Permalink)

	var task *Task

	switch {
	case ref.Permalink != "":
		slog.Debug("wrike snapshot using permalink param")
		task, err = p.client.GetTaskByPermalinkParam(ctx, ref.Permalink)
	case apiIDPattern.MatchString(ref.TaskID):
		slog.Debug("wrike snapshot using API ID")
		task, err = p.client.GetTask(ctx, ref.TaskID)
	case numericIDPattern.MatchString(ref.TaskID):
		permalink := BuildPermalinkURL(ref.TaskID)
		slog.Debug("wrike snapshot using numeric ID -> permalink", "permalink", permalink)
		task, err = p.client.GetTaskByPermalinkParam(ctx, permalink)
	default:
		return nil, fmt.Errorf("%w: unrecognized task ID format: %s", ErrInvalidReference, ref.TaskID)
	}

	if err != nil {
		slog.Error("wrike snapshot fetch failed", "error", err)

		return nil, fmt.Errorf("fetch task for snapshot: %w", err)
	}

	slog.Debug("wrike snapshot got task", "title", task.Title, "subTaskCount", len(task.SubTaskIDs))

	comments, _ := p.client.GetComments(ctx, task.ID)

	var content strings.Builder
	content.WriteString(fmt.Sprintf("# %s\n\n", task.Title))
	content.WriteString(fmt.Sprintf("**Status:** %s\n", task.Status))
	content.WriteString(fmt.Sprintf("**Priority:** %s\n", task.Priority))
	content.WriteString(fmt.Sprintf("**Permalink:** %s\n", task.Permalink))

	// Add custom fields if present (often contain useful metadata like estimates)
	if len(task.CustomFields) > 0 {
		for _, cf := range task.CustomFields {
			fieldName := p.getCustomFieldName(ctx, cf.ID)
			content.WriteString(fmt.Sprintf("**%s:** %s\n", fieldName, cf.Value))
		}
	}
	content.WriteString("\n")

	// Fetch and include parent task (if this is a subtask)
	if len(task.SuperTaskIDs) > 0 {
		parentInfo, fetchErr := p.fetchParentTask(ctx, task.ID, task.SuperTaskIDs)
		if fetchErr != nil {
			// Log error but continue - parent context is optional
			slog.Warn("failed to fetch parent task", "error", fetchErr)
			content.WriteString("## Parent Task\n\n")
			content.WriteString("*(Parent task information unavailable)*\n\n")
		} else if parentInfo != nil {
			content.WriteString("## Parent Task\n\n")
			content.WriteString(fmt.Sprintf("**[%s](%s)**\n\n", parentInfo.Title, parentInfo.Permalink))
			content.WriteString(fmt.Sprintf("**Status:** %s\n\n", parentInfo.Status))
			if parentInfo.Description != "" {
				content.WriteString(parentInfo.Description)
				content.WriteString("\n\n")
			}
		}
	}

	if task.Description != "" {
		content.WriteString("## Description\n\n")
		content.WriteString(task.Description)
		content.WriteString("\n\n")
	}

	// Add dependencies if present
	if len(task.DependencyIDs) > 0 {
		content.WriteString("## Dependencies\n\n")
		content.WriteString("This task depends on:\n")
		for _, depID := range task.DependencyIDs {
			content.WriteString(fmt.Sprintf("- %s\n", depID))
		}
		content.WriteString("\n")
	}

	if len(comments) > 0 {
		content.WriteString("## Comments\n\n")
		for _, c := range comments {
			content.WriteString(fmt.Sprintf("### %s - %s\n\n", c.AuthorName, c.CreatedDate.Format(time.RFC3339)))
			content.WriteString(c.Text)
			content.WriteString("\n\n")
		}
	}

	// Fetch and include subtasks
	if len(task.SubTaskIDs) > 0 {
		subtaskInfos, fetchErr := p.fetchSubtasks(ctx, task.SubTaskIDs, 0)
		if fetchErr == nil && len(subtaskInfos) > 0 {
			content.WriteString("## Subtasks\n\n")
			for i, st := range subtaskInfos {
				content.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, st.Title))
				content.WriteString(fmt.Sprintf("**Status:** %s\n\n", st.Status))
				if st.Description != "" {
					content.WriteString(st.Description)
					content.WriteString("\n\n")
				}
			}
		}
	}

	return &provider.Snapshot{
		Type: ProviderName,
		Ref:  id,
		Files: []provider.SnapshotFile{
			{
				Path:    "task.md",
				Content: content.String(),
			},
		},
	}, nil
}

// ListTasks returns tasks from a folder or space.
// scopeType should be "folder" or "space".
func (p *Provider) ListTasks(ctx context.Context, scopeType, scopeID string) ([]*provider.WorkUnit, error) {
	var tasks []Task
	var err error

	switch scopeType {
	case "folder":
		tasks, err = p.client.GetTasksInFolder(ctx, scopeID)
	case "space":
		tasks, err = p.client.GetTasksInSpace(ctx, scopeID)
	default:
		return nil, fmt.Errorf("invalid scope type: %s (use 'folder' or 'space')", scopeType)
	}

	if err != nil {
		return nil, err
	}

	result := make([]*provider.WorkUnit, 0, len(tasks))
	for _, task := range tasks {
		wu := p.taskToWorkUnit(&task)
		result = append(result, wu)
	}

	return result, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper functions
// ──────────────────────────────────────────────────────────────────────────────

// mapStatus converts Wrike status to provider status.
func mapStatus(status string) provider.Status {
	switch strings.ToLower(status) {
	case "active", "new", "in progress", "inprogress", "draft":
		return provider.StatusOpen
	case "completed", "done":
		return provider.StatusDone
	case "cancelled", "canceled", "deferred", "closed":
		return provider.StatusClosed
	case "review":
		return provider.StatusReview
	default:
		return provider.StatusOpen
	}
}

// mapPriority converts Wrike priority to provider priority.
func mapPriority(priority string) provider.Priority {
	switch strings.ToLower(priority) {
	case "critical", "urgent":
		return provider.PriorityCritical
	case "high":
		return provider.PriorityHigh
	case "low":
		return provider.PriorityLow
	default:
		return provider.PriorityNormal
	}
}

// mapComments converts Wrike comments to provider comments.
func mapComments(comments []Comment) []provider.Comment {
	if comments == nil {
		return nil
	}

	result := make([]provider.Comment, 0, len(comments))
	for _, c := range comments {
		author := provider.Person{
			ID:   c.AuthorID,
			Name: c.AuthorName,
		}
		result = append(result, provider.Comment{
			ID:        c.ID,
			Author:    author,
			Body:      c.Text,
			CreatedAt: c.CreatedDate,
			UpdatedAt: c.UpdatedDate,
		})
	}

	return result
}

// buildMetadata creates metadata map from task and subtasks.
// Includes all optional fields returned by the Wrike API.
func buildMetadata(task *Task, subtasks []SubtaskInfo) map[string]any {
	metadata := make(map[string]any)

	// Core identifiers
	metadata["permalink"] = task.Permalink
	metadata["api_id"] = task.ID
	metadata["wrike_status"] = task.Status
	metadata["wrike_priority"] = task.Priority

	// Brief description (truncated version)
	if task.BriefDescription != "" {
		metadata["brief_description"] = task.BriefDescription
	}

	// Hierarchy relationships
	if len(task.ParentIDs) > 0 {
		metadata["parent_ids"] = task.ParentIDs
	}
	if len(task.SuperParentIDs) > 0 {
		metadata["super_parent_ids"] = task.SuperParentIDs
	}
	if len(task.SuperTaskIDs) > 0 {
		metadata["super_task_ids"] = task.SuperTaskIDs
		metadata["is_subtask"] = true
		metadata["parent_task_ids"] = task.SuperTaskIDs
	}

	// Dependencies
	if len(task.DependencyIDs) > 0 {
		metadata["dependency_ids"] = task.DependencyIDs
	}

	// People
	if len(task.ResponsibleIDs) > 0 {
		metadata["responsible_ids"] = task.ResponsibleIDs
	}
	if len(task.AuthorIDs) > 0 {
		metadata["author_ids"] = task.AuthorIDs
	}
	if len(task.SharedIDs) > 0 {
		metadata["shared_ids"] = task.SharedIDs
	}

	// Attachments
	if task.AttachmentCount > 0 {
		metadata["attachment_count"] = task.AttachmentCount
	}
	metadata["has_attachments"] = task.HasAttachments

	// Recurrence
	metadata["recurrent"] = task.Recurrent

	// Custom fields
	if len(task.CustomFields) > 0 {
		cfList := make([]map[string]string, 0, len(task.CustomFields))
		for _, cf := range task.CustomFields {
			cfList = append(cfList, map[string]string{
				"id":    cf.ID,
				"value": cf.Value,
			})
		}
		metadata["custom_fields"] = cfList
	}

	// Metadata key-value pairs
	if len(task.Metadata) > 0 {
		mdList := make([]map[string]string, 0, len(task.Metadata))
		for _, md := range task.Metadata {
			mdList = append(mdList, map[string]string{
				"key":   md.Key,
				"value": md.Value,
			})
		}
		metadata["wrike_metadata"] = mdList
	}

	// Subtasks (if provided)
	if len(subtasks) > 0 {
		subtaskList := make([]map[string]string, 0, len(subtasks))
		for _, st := range subtasks {
			subtaskList = append(subtaskList, map[string]string{
				"id":     st.ID,
				"title":  st.Title,
				"status": st.Status,
			})
		}
		metadata["subtasks"] = subtaskList
		metadata["subtask_count"] = len(subtasks)
	}

	return metadata
}

// taskToWorkUnit converts a Task to a WorkUnit without fetching nested data
// Used by ListTasks for efficiency when listing multiple tasks.
func (p *Provider) taskToWorkUnit(task *Task) *provider.WorkUnit {
	// Extract numeric ID from permalink for ExternalKey
	numericID := ExtractNumericID(task.Permalink)
	if numericID == "" {
		numericID = task.ID
	}

	return &provider.WorkUnit{
		ID:          numericID,
		ExternalID:  task.ID,
		Provider:    ProviderName,
		Title:       task.Title,
		Description: task.Description,
		Status:      mapStatus(task.Status),
		Priority:    mapPriority(task.Priority),
		Labels:      []string{},
		Assignees:   []provider.Person{},
		Comments:    nil,
		Attachments: nil,
		Subtasks:    task.SubTaskIDs,
		Metadata:    buildMetadata(task, nil),
		CreatedAt:   task.CreatedDate,
		UpdatedAt:   task.UpdatedDate,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: task.Permalink,
			SyncedAt:  time.Now(),
		},
		ExternalKey: numericID,
		TaskType:    "task",
		Slug:        slug.Slugify(task.Title, 50),
	}
}
