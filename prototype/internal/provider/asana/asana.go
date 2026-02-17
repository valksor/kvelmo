package asana

import (
	"context"
	"fmt"
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
const ProviderName = "asana"

// Provider implements the Asana task provider.
type Provider struct {
	client *Client
	config *Config
}

// Config holds Asana provider configuration.
type Config struct {
	Token          string
	WorkspaceGID   string
	DefaultProject string // Default project GID for list operations
	BranchPattern  string
	CommitPrefix   string
}

// Info returns provider metadata.
func Info() provider.ProviderInfo {
	return provider.ProviderInfo{
		Name:        ProviderName,
		Description: "Load tasks from Asana projects",
		Schemes:     []string{"asana", "as"},
		Capabilities: capability.CapabilitySet{
			capability.CapRead:               true,
			capability.CapList:               true,
			capability.CapFetchComments:      true,
			capability.CapComment:            true,
			capability.CapUpdateStatus:       true,
			capability.CapManageLabels:       true,
			capability.CapCreateWorkUnit:     true,
			capability.CapDownloadAttachment: true,
			capability.CapSnapshot:           true,
			capability.CapFetchSubtasks:      true,
			capability.CapFetchParent:        true,
			capability.CapCreateDependency:   true,
			capability.CapFetchDependencies:  true,
		},
	}
}

// New creates a new Asana provider instance.
func New(_ context.Context, cfg providerconfig.Config) (any, error) {
	config := &Config{
		Token:          cfg.GetString("token"),
		WorkspaceGID:   cfg.GetString("workspace_gid"),
		DefaultProject: cfg.GetString("default_project"),
		BranchPattern:  cfg.GetString("branch_pattern"),
		CommitPrefix:   cfg.GetString("commit_prefix"),
	}

	// Resolve token
	token, err := ResolveToken(config.Token)
	if err != nil {
		return nil, err
	}

	client := NewClient(token, config.WorkspaceGID)

	return &Provider{
		client: client,
		config: config,
	}, nil
}

// Match checks if the input looks like an Asana reference.
func (p *Provider) Match(input string) bool {
	// Check for asana: or as: prefix
	if strings.HasPrefix(input, "asana:") || strings.HasPrefix(input, "as:") {
		return true
	}

	// Check for Asana URL pattern
	if strings.Contains(input, "app.asana.com") {
		return true
	}

	// Check for bare GID pattern (long numeric string)
	_, err := ParseReference(input)

	return err == nil
}

// Parse parses an Asana reference and returns a canonical ID.
func (p *Provider) Parse(input string) (string, error) {
	ref, err := ParseReference(input)
	if err != nil {
		return "", err
	}

	return ref.TaskGID, nil
}

// Fetch retrieves a task by its GID.
func (p *Provider) Fetch(ctx context.Context, id string) (*workunit.WorkUnit, error) {
	task, err := p.client.GetTask(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetch task %s: %w", id, err)
	}

	wu := p.taskToWorkUnit(task)

	// Fetch attachments (optional - don't fail if this errors)
	attachments, err := p.client.GetTaskAttachments(ctx, id)
	if err == nil && len(attachments) > 0 {
		wu.Attachments = mapAttachments(attachments)
	}

	return wu, nil
}

// Snapshot creates a snapshot of the task's current state.
func (p *Provider) Snapshot(ctx context.Context, id string) (*snapshot.Snapshot, error) {
	task, err := p.client.GetTask(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("snapshot task %s: %w", id, err)
	}

	// Build markdown content
	content := buildSnapshotContent(task)

	return &snapshot.Snapshot{
		Type:    ProviderName,
		Ref:     "asana:" + id,
		Content: content,
	}, nil
}

// List retrieves tasks from a project.
func (p *Provider) List(ctx context.Context, opts workunit.ListOptions) ([]*workunit.WorkUnit, error) {
	projectGID := p.config.DefaultProject
	if projectGID == "" {
		return nil, ErrProjectRequired
	}

	// Filter by completion status
	var completedSince *time.Time
	if opts.Status == workunit.StatusOpen || opts.Status == "" {
		// Show only incomplete tasks
		t := time.Now().Add(-365 * 24 * time.Hour) // Last year's completed tasks only
		completedSince = &t
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	tasks, err := p.client.ListProjectTasks(ctx, projectGID, completedSince, limit)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	var units []*workunit.WorkUnit
	for _, task := range tasks {
		// Filter completed tasks if looking for open
		if opts.Status == workunit.StatusOpen && task.Completed {
			continue
		}

		// Filter by labels if specified
		if len(opts.Labels) > 0 && !hasAnyTag(task.Tags, opts.Labels) {
			continue
		}

		units = append(units, p.taskToWorkUnit(&task))
	}

	return units, nil
}

// FetchComments retrieves comments (stories) for a task.
func (p *Provider) FetchComments(ctx context.Context, id string) ([]workunit.Comment, error) {
	stories, err := p.client.GetTaskStories(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetch comments for %s: %w", id, err)
	}

	var comments []workunit.Comment
	for _, story := range stories {
		// Only include comment-type stories
		if story.ResourceSubtype != "comment_added" {
			continue
		}

		author := workunit.Person{}
		if story.CreatedBy != nil {
			author = workunit.Person{
				Name:  story.CreatedBy.Name,
				Email: story.CreatedBy.Email,
			}
		}

		comments = append(comments, workunit.Comment{
			ID:        story.GID,
			Author:    author,
			Body:      story.Text,
			CreatedAt: story.CreatedAt,
		})
	}

	return comments, nil
}

// AddComment adds a comment to a task.
func (p *Provider) AddComment(ctx context.Context, id string, body string) (*workunit.Comment, error) {
	story, err := p.client.AddTaskComment(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("add comment to %s: %w", id, err)
	}

	author := workunit.Person{}
	if story.CreatedBy != nil {
		author = workunit.Person{
			Name:  story.CreatedBy.Name,
			Email: story.CreatedBy.Email,
		}
	}

	return &workunit.Comment{
		ID:        story.GID,
		Author:    author,
		Body:      story.Text,
		CreatedAt: story.CreatedAt,
	}, nil
}

// UpdateStatus updates the task status (completes the task or moves to section).
func (p *Provider) UpdateStatus(ctx context.Context, id string, status workunit.Status) error {
	switch status {
	case workunit.StatusClosed, workunit.StatusDone:
		_, err := p.client.CompleteTask(ctx, id)
		if err != nil {
			return fmt.Errorf("complete task %s: %w", id, err)
		}
	case workunit.StatusOpen, workunit.StatusInProgress, workunit.StatusReview:
		// For these statuses, we could potentially move to sections
		// but this requires project context
		return nil
	}

	return nil
}

// --- Helper functions ---

func (p *Provider) taskToWorkUnit(task *Task) *workunit.WorkUnit {
	unit := &workunit.WorkUnit{
		ID:          task.GID,
		ExternalID:  task.GID,
		ExternalKey: task.GID,
		Provider:    ProviderName,
		Title:       task.Name,
		Description: task.Notes,
		Slug:        slug.Slugify(task.Name, 50),
		Status:      mapAsanaStatus(task),
		Priority:    workunit.PriorityNormal, // Asana doesn't have built-in priority
		TaskType:    mapTaskType(task),
		Labels:      extractTagNames(task.Tags),
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.ModifiedAt,
		Source: workunit.SourceInfo{
			Type:      ProviderName,
			Reference: "asana:" + task.GID,
			SyncedAt:  time.Now(),
		},
		Metadata: map[string]any{
			"permalink_url": task.PermalinkURL,
			"projects":      extractProjectNames(task.Projects),
		},
	}

	// Set assignees
	if task.Assignee != nil {
		unit.Assignees = []workunit.Person{
			{
				Name:  task.Assignee.Name,
				Email: task.Assignee.Email,
			},
		}
	}

	// Set due date in metadata
	if task.DueOn != "" {
		unit.Metadata["due_on"] = task.DueOn
	}

	return unit
}

func mapAsanaStatus(task *Task) workunit.Status {
	if task.Completed {
		return workunit.StatusClosed
	}

	// Check approval status for approval tasks
	if task.ResourceSubtype == "approval" {
		switch task.ApprovalStatus {
		case "approved":
			return workunit.StatusClosed
		case "rejected":
			return workunit.StatusOpen
		case "pending":
			return workunit.StatusInProgress
		}
	}

	// Check section for status (common Asana pattern)
	for _, membership := range task.Memberships {
		if membership.Section != nil {
			sectionName := strings.ToLower(membership.Section.Name)
			switch {
			case strings.Contains(sectionName, "done") || strings.Contains(sectionName, "complete"):
				return workunit.StatusClosed
			case strings.Contains(sectionName, "progress") || strings.Contains(sectionName, "doing"):
				return workunit.StatusInProgress
			case strings.Contains(sectionName, "review"):
				return workunit.StatusReview
			case strings.Contains(sectionName, "blocked") || strings.Contains(sectionName, "hold"):
				return workunit.StatusOpen // Blocked tasks are still open
			}
		}
	}

	return workunit.StatusOpen
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

	// Check subtask type
	if task.ResourceSubtype == "milestone" {
		return "milestone"
	}
	if task.ResourceSubtype == "approval" {
		return "approval"
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

func extractProjectNames(projects []Project) []string {
	var names []string
	for _, proj := range projects {
		names = append(names, proj.Name)
	}

	return names
}

func mapAttachments(attachments []Attachment) []workunit.Attachment {
	result := make([]workunit.Attachment, 0, len(attachments))
	for _, a := range attachments {
		// Use download_url if available, otherwise permanent_url
		url := a.DownloadURL
		if url == "" {
			url = a.PermanentURL
		}

		result = append(result, workunit.Attachment{
			ID:        url, // Use URL as ID for DownloadAttachment compatibility
			Name:      a.Name,
			URL:       url,
			Size:      a.Size,
			CreatedAt: a.CreatedAt,
		})
	}

	return result
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

func buildSnapshotContent(task *Task) string {
	var sb strings.Builder

	// Title
	sb.WriteString("# ")
	sb.WriteString(task.Name)
	sb.WriteString("\n\n")

	// Metadata
	sb.WriteString("**GID:** ")
	sb.WriteString(task.GID)
	sb.WriteString("\n")

	if task.Assignee != nil {
		sb.WriteString("**Assignee:** ")
		sb.WriteString(task.Assignee.Name)
		sb.WriteString("\n")
	}

	if task.DueOn != "" {
		sb.WriteString("**Due:** ")
		sb.WriteString(task.DueOn)
		sb.WriteString("\n")
	}

	if len(task.Tags) > 0 {
		sb.WriteString("**Tags:** ")
		sb.WriteString(strings.Join(extractTagNames(task.Tags), ", "))
		sb.WriteString("\n")
	}

	if len(task.Projects) > 0 {
		sb.WriteString("**Projects:** ")
		sb.WriteString(strings.Join(extractProjectNames(task.Projects), ", "))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Description
	if task.Notes != "" {
		sb.WriteString("## Description\n\n")
		sb.WriteString(task.Notes)
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetBranchSuggestion returns a suggested branch name for the task.
func (p *Provider) GetBranchSuggestion(task *workunit.WorkUnit) string {
	if p.config.BranchPattern == "" {
		// Default pattern
		return "task/" + task.ID
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
