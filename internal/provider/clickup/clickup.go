package clickup

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/capability"
	"github.com/valksor/go-toolkit/providerconfig"
	"github.com/valksor/go-toolkit/slug"
	"github.com/valksor/go-toolkit/snapshot"
	"github.com/valksor/go-toolkit/workunit"
)

// ProviderName is the canonical name for this provider.
const ProviderName = "clickup"

// Provider implements the ClickUp task provider.
type Provider struct {
	client *Client
	config *Config
}

// Config holds ClickUp provider configuration.
type Config struct {
	Token         string
	TeamID        string // Team/Workspace ID
	DefaultList   string // Default list ID for list operations
	BranchPattern string
	CommitPrefix  string
}

// Info returns provider metadata.
func Info() provider.ProviderInfo {
	return provider.ProviderInfo{
		Name:        ProviderName,
		Description: "Load tasks from ClickUp",
		Schemes:     []string{"clickup", "cu"},
		Capabilities: capability.CapabilitySet{
			capability.CapRead:               true,
			capability.CapList:               true,
			capability.CapFetchComments:      true,
			capability.CapComment:            true,
			capability.CapUpdateStatus:       true,
			capability.CapManageLabels:       true,
			capability.CapDownloadAttachment: true,
			capability.CapSnapshot:           true,
			capability.CapCreateWorkUnit:     true,
			capability.CapFetchSubtasks:      true,
			capability.CapFetchParent:        true,
			capability.CapCreateDependency:   true,
			capability.CapFetchDependencies:  true,
		},
	}
}

// New creates a new ClickUp provider instance.
func New(_ context.Context, cfg providerconfig.Config) (any, error) {
	config := &Config{
		Token:         cfg.GetString("token"),
		TeamID:        cfg.GetString("team_id"),
		DefaultList:   cfg.GetString("default_list"),
		BranchPattern: cfg.GetString("branch_pattern"),
		CommitPrefix:  cfg.GetString("commit_prefix"),
	}

	// Resolve token
	token, err := ResolveToken(config.Token)
	if err != nil {
		return nil, err
	}

	client := NewClient(token)

	return &Provider{
		client: client,
		config: config,
	}, nil
}

// Match checks if the input looks like a ClickUp reference.
func (p *Provider) Match(input string) bool {
	// Check for clickup: or cu: prefix
	if strings.HasPrefix(input, "clickup:") || strings.HasPrefix(input, "cu:") {
		return true
	}

	// Check for ClickUp URL patterns
	if strings.Contains(input, "app.clickup.com") || strings.Contains(input, "sharing.clickup.com") {
		return true
	}

	// Check for bare task ID or custom ID pattern
	_, err := ParseReference(input)

	return err == nil
}

// Parse parses a ClickUp reference and returns a canonical ID.
func (p *Provider) Parse(input string) (string, error) {
	ref, err := ParseReference(input)
	if err != nil {
		return "", err
	}
	if ref.CustomID != "" {
		return ref.CustomID, nil
	}

	return ref.TaskID, nil
}

// Fetch retrieves a task by its ID.
func (p *Provider) Fetch(ctx context.Context, id string) (*workunit.WorkUnit, error) {
	// Determine if this is a custom ID or task ID
	ref, err := ParseReference(id)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	var task *Task
	if ref.CustomID != "" && p.config.TeamID != "" {
		task, err = p.client.GetTaskByCustomID(ctx, p.config.TeamID, ref.CustomID)
	} else {
		taskID := ref.TaskID
		if taskID == "" {
			taskID = ref.CustomID
		}
		task, err = p.client.GetTask(ctx, taskID)
	}

	if err != nil {
		return nil, fmt.Errorf("fetch task %s: %w", id, err)
	}

	return p.taskToWorkUnit(task), nil
}

// Snapshot creates a snapshot of the task's current state.
func (p *Provider) Snapshot(ctx context.Context, id string) (*snapshot.Snapshot, error) {
	ref, err := ParseReference(id)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	var task *Task
	if ref.CustomID != "" && p.config.TeamID != "" {
		task, err = p.client.GetTaskByCustomID(ctx, p.config.TeamID, ref.CustomID)
	} else {
		taskID := ref.TaskID
		if taskID == "" {
			taskID = ref.CustomID
		}
		task, err = p.client.GetTask(ctx, taskID)
	}

	if err != nil {
		return nil, fmt.Errorf("snapshot task %s: %w", id, err)
	}

	// Build markdown content
	content := buildSnapshotContent(task)

	return &snapshot.Snapshot{
		Type:    ProviderName,
		Ref:     "clickup:" + task.ID,
		Content: content,
	}, nil
}

// List retrieves tasks from a list.
func (p *Provider) List(ctx context.Context, opts workunit.ListOptions) ([]*workunit.WorkUnit, error) {
	listID := p.config.DefaultList
	if listID == "" {
		return nil, ErrListRequired
	}

	// Include closed tasks based on status filter
	includeClosed := opts.Status == "" || opts.Status == workunit.StatusClosed || opts.Status == workunit.StatusDone

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	tasks, err := p.client.ListTasks(ctx, listID, false, limit)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	var units []*workunit.WorkUnit
	for _, task := range tasks {
		// Filter by status
		if opts.Status != "" {
			taskStatus := mapClickUpStatus(&task)
			if taskStatus != opts.Status && !statusMatches(taskStatus, opts.Status) {
				continue
			}
		}

		// Filter closed tasks if only looking for open
		if opts.Status == workunit.StatusOpen && !includeClosed {
			if task.Status != nil && task.Status.Type == "closed" {
				continue
			}
		}

		// Filter by labels if specified
		if len(opts.Labels) > 0 && !hasAnyTag(task.Tags, opts.Labels) {
			continue
		}

		units = append(units, p.taskToWorkUnit(&task))
	}

	return units, nil
}

// FetchComments retrieves comments for a task.
func (p *Provider) FetchComments(ctx context.Context, id string) ([]workunit.Comment, error) {
	ref, err := ParseReference(id)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	taskID := ref.TaskID
	if taskID == "" {
		taskID = ref.CustomID
	}

	comments, err := p.client.GetTaskComments(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("fetch comments for %s: %w", id, err)
	}

	var result []workunit.Comment
	for _, comment := range comments {
		// Parse date
		createdAt := parseTimestamp(comment.Date)

		author := workunit.Person{
			ID:    strconv.Itoa(comment.User.ID),
			Name:  comment.User.Username,
			Email: comment.User.Email,
		}

		result = append(result, workunit.Comment{
			ID:        comment.ID,
			Author:    author,
			Body:      comment.CommentText,
			CreatedAt: createdAt,
		})
	}

	return result, nil
}

// AddComment adds a comment to a task.
func (p *Provider) AddComment(ctx context.Context, id string, body string) (*workunit.Comment, error) {
	ref, err := ParseReference(id)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	taskID := ref.TaskID
	if taskID == "" {
		taskID = ref.CustomID
	}

	comment, err := p.client.AddTaskComment(ctx, taskID, body)
	if err != nil {
		return nil, fmt.Errorf("add comment to %s: %w", id, err)
	}

	return &workunit.Comment{
		ID: comment.ID,
		Author: workunit.Person{
			ID:    strconv.Itoa(comment.User.ID),
			Name:  comment.User.Username,
			Email: comment.User.Email,
		},
		Body:      comment.CommentText,
		CreatedAt: parseTimestamp(comment.Date),
	}, nil
}

// UpdateStatus updates the task status.
func (p *Provider) UpdateStatus(ctx context.Context, id string, status workunit.Status) error {
	ref, err := ParseReference(id)
	if err != nil {
		return fmt.Errorf("parse reference: %w", err)
	}

	taskID := ref.TaskID
	if taskID == "" {
		taskID = ref.CustomID
	}

	// Map provider status to ClickUp status
	clickUpStatus := mapToClickUpStatus(status)
	if clickUpStatus == "" {
		return nil // No mapping for this status
	}

	_, err = p.client.UpdateTaskStatus(ctx, taskID, clickUpStatus)
	if err != nil {
		return fmt.Errorf("update task status %s: %w", id, err)
	}

	return nil
}

// --- Helper functions ---

func (p *Provider) taskToWorkUnit(task *Task) *workunit.WorkUnit {
	unit := &workunit.WorkUnit{
		ID:          task.ID,
		ExternalID:  task.ID,
		ExternalKey: task.ID,
		Provider:    ProviderName,
		Title:       task.Name,
		Description: task.Description,
		Slug:        slug.Slugify(task.Name, 50),
		Status:      mapClickUpStatus(task),
		Priority:    mapClickUpPriority(task.Priority),
		TaskType:    mapTaskType(task),
		Labels:      extractTagNames(task.Tags),
		CreatedAt:   parseTimestamp(task.DateCreated),
		UpdatedAt:   parseTimestamp(task.DateUpdated),
		Source: workunit.SourceInfo{
			Type:      ProviderName,
			Reference: "clickup:" + task.ID,
			SyncedAt:  time.Now(),
		},
		Metadata: map[string]any{
			"url":       task.URL,
			"team_id":   task.TeamID,
			"list_id":   getListID(task),
			"folder_id": getFolderID(task),
			"space_id":  getSpaceID(task),
		},
	}

	// Use custom ID if available
	if task.CustomID != "" {
		unit.ExternalKey = task.CustomID
	}

	// Set assignees
	for _, assignee := range task.Assignees {
		unit.Assignees = append(unit.Assignees, workunit.Person{
			ID:    strconv.Itoa(assignee.ID),
			Name:  assignee.Username,
			Email: assignee.Email,
		})
	}

	// Set due date in metadata
	if task.DueDate != nil {
		unit.Metadata["due_date"] = time.UnixMilli(*task.DueDate).Format(time.RFC3339)
	}

	// Set time estimate in metadata
	if task.TimeEstimate != nil {
		unit.Metadata["time_estimate_ms"] = *task.TimeEstimate
	}

	// Set points in metadata
	if task.Points != nil {
		unit.Metadata["points"] = *task.Points
	}

	// Set attachments
	if len(task.Attachments) > 0 {
		unit.Attachments = make([]workunit.Attachment, len(task.Attachments))
		for i, att := range task.Attachments {
			unit.Attachments[i] = workunit.Attachment{
				ID:   att.URL, // Use URL as ID for consistency with DownloadAttachment
				Name: att.Title,
				URL:  att.URL,
			}
		}
	}

	return unit
}

func mapClickUpStatus(task *Task) workunit.Status {
	if task.Status == nil {
		return workunit.StatusOpen
	}

	// Check status type
	switch task.Status.Type {
	case "closed", "done":
		return workunit.StatusClosed
	case "open":
		return workunit.StatusOpen
	}

	// Check status name
	statusLower := strings.ToLower(task.Status.Status)
	switch {
	case strings.Contains(statusLower, "done") || strings.Contains(statusLower, "complete") || strings.Contains(statusLower, "closed"):
		return workunit.StatusClosed
	case strings.Contains(statusLower, "progress") || strings.Contains(statusLower, "doing") || strings.Contains(statusLower, "started"):
		return workunit.StatusInProgress
	case strings.Contains(statusLower, "review") || strings.Contains(statusLower, "qa"):
		return workunit.StatusReview
	case strings.Contains(statusLower, "todo") || strings.Contains(statusLower, "open") || strings.Contains(statusLower, "backlog"):
		return workunit.StatusOpen
	}

	return workunit.StatusOpen
}

func mapClickUpPriority(priority *Priority) workunit.Priority {
	if priority == nil {
		return workunit.PriorityNormal
	}

	switch strings.ToLower(priority.Priority) {
	case "urgent", "1":
		return workunit.PriorityCritical
	case "high", "2":
		return workunit.PriorityHigh
	case "normal", "3":
		return workunit.PriorityNormal
	case "low", "4":
		return workunit.PriorityLow
	}

	return workunit.PriorityNormal
}

func mapToClickUpStatus(status workunit.Status) string {
	switch status {
	case workunit.StatusClosed, workunit.StatusDone:
		return "complete"
	case workunit.StatusInProgress:
		return "in progress"
	case workunit.StatusOpen:
		return "to do"
	case workunit.StatusReview:
		return "in review"
	default:
		return ""
	}
}

func statusMatches(taskStatus, filterStatus workunit.Status) bool {
	// StatusDone and StatusClosed are equivalent
	if filterStatus == workunit.StatusDone && taskStatus == workunit.StatusClosed {
		return true
	}
	if filterStatus == workunit.StatusClosed && taskStatus == workunit.StatusDone {
		return true
	}

	return false
}

func mapTaskType(task *Task) string {
	// Check tags for type hints
	for _, tag := range task.Tags {
		tagLower := strings.ToLower(tag.Name)
		switch {
		case strings.Contains(tagLower, "bug") || strings.Contains(tagLower, "fix"):
			return "fix"
		case strings.Contains(tagLower, "feature") || strings.Contains(tagLower, "enhancement"):
			return "feature"
		case strings.Contains(tagLower, "chore") || strings.Contains(tagLower, "task"):
			return "task"
		case strings.Contains(tagLower, "doc"):
			return "docs"
		}
	}

	return "task"
}

func extractTagNames(tags []Tag) []string {
	var names []string
	for _, tag := range tags {
		names = append(names, tag.Name)
	}

	return names
}

func hasAnyTag(tags []Tag, tagNames []string) bool {
	for _, tag := range tags {
		tagLower := strings.ToLower(tag.Name)
		for _, name := range tagNames {
			if strings.ToLower(name) == tagLower {
				return true
			}
		}
	}

	return false
}

func parseTimestamp(ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}

	// Try millisecond timestamp
	if ms, err := strconv.ParseInt(ts, 10, 64); err == nil {
		return time.UnixMilli(ms)
	}

	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t
	}

	return time.Time{}
}

func getListID(task *Task) string {
	if task.List != nil {
		return task.List.ID
	}

	return ""
}

func getFolderID(task *Task) string {
	if task.Folder != nil {
		return task.Folder.ID
	}

	return ""
}

func getSpaceID(task *Task) string {
	if task.Space != nil {
		return task.Space.ID
	}

	return ""
}

func buildSnapshotContent(task *Task) string {
	var sb strings.Builder

	// Title
	sb.WriteString("# ")
	sb.WriteString(task.Name)
	sb.WriteString("\n\n")

	// Metadata
	sb.WriteString("**ID:** ")
	sb.WriteString(task.ID)
	sb.WriteString("\n")

	if task.CustomID != "" {
		sb.WriteString("**Custom ID:** ")
		sb.WriteString(task.CustomID)
		sb.WriteString("\n")
	}

	if task.Status != nil {
		sb.WriteString("**Status:** ")
		sb.WriteString(task.Status.Status)
		sb.WriteString("\n")
	}

	if task.Priority != nil {
		sb.WriteString("**Priority:** ")
		sb.WriteString(task.Priority.Priority)
		sb.WriteString("\n")
	}

	if len(task.Assignees) > 0 {
		sb.WriteString("**Assignees:** ")
		var names []string
		for _, a := range task.Assignees {
			names = append(names, a.Username)
		}
		sb.WriteString(strings.Join(names, ", "))
		sb.WriteString("\n")
	}

	if task.DueDate != nil {
		sb.WriteString("**Due:** ")
		sb.WriteString(time.UnixMilli(*task.DueDate).Format("2006-01-02"))
		sb.WriteString("\n")
	}

	if len(task.Tags) > 0 {
		sb.WriteString("**Tags:** ")
		sb.WriteString(strings.Join(extractTagNames(task.Tags), ", "))
		sb.WriteString("\n")
	}

	if task.URL != "" {
		sb.WriteString("**URL:** ")
		sb.WriteString(task.URL)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Description
	if task.Description != "" {
		sb.WriteString("## Description\n\n")
		sb.WriteString(task.Description)
		sb.WriteString("\n")
	}

	// Checklists
	if len(task.Checklists) > 0 {
		sb.WriteString("\n## Checklists\n\n")
		for _, cl := range task.Checklists {
			sb.WriteString("### ")
			sb.WriteString(cl.Name)
			sb.WriteString("\n\n")
			for _, item := range cl.Items {
				if item.Resolved {
					sb.WriteString("- [x] ")
				} else {
					sb.WriteString("- [ ] ")
				}
				sb.WriteString(item.Name)
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// GetBranchSuggestion returns a suggested branch name for the task.
func (p *Provider) GetBranchSuggestion(task *workunit.WorkUnit) string {
	if p.config.BranchPattern == "" {
		// Default pattern
		return "task/" + task.ExternalKey
	}

	// Simple template replacement
	result := p.config.BranchPattern
	result = strings.ReplaceAll(result, "{key}", task.ExternalKey)
	result = strings.ReplaceAll(result, "{id}", task.ID)

	// Slugify title
	titleSlug := slug.Slugify(task.Title, 50)
	result = strings.ReplaceAll(result, "{slug}", titleSlug)

	return result
}
