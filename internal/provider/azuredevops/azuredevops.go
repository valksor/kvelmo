package azuredevops

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/naming"
	"github.com/valksor/go-mehrhof/internal/provider"
)

// ProviderName is the canonical name for this provider.
const ProviderName = "azuredevops"

// Provider implements the Azure DevOps work item provider.
type Provider struct {
	client *Client
	config *Config
}

// Config holds Azure DevOps provider configuration.
type Config struct {
	Token         string
	Organization  string
	Project       string
	AreaPath      string // Default area path for filtering
	IterationPath string // Default iteration path for filtering
	RepoName      string // Default repository for PR creation
	TargetBranch  string // Default target branch for PRs
	BranchPattern string
	CommitPrefix  string
}

// Info returns provider metadata.
func Info() provider.ProviderInfo {
	return provider.ProviderInfo{
		Name:        ProviderName,
		Description: "Load work items from Azure DevOps",
		Schemes:     []string{"azdo", "azure"},
		Capabilities: provider.CapabilitySet{
			provider.CapRead:               true,
			provider.CapList:               true,
			provider.CapFetchComments:      true,
			provider.CapComment:            true,
			provider.CapUpdateStatus:       true,
			provider.CapManageLabels:       true,
			provider.CapDownloadAttachment: true,
			provider.CapSnapshot:           true,
			provider.CapCreatePR:           true,
			provider.CapLinkBranch:         true,
			provider.CapCreateWorkUnit:     true,
			provider.CapFetchSubtasks:      true,
			provider.CapFetchPR:            true,
			provider.CapPRComment:          true,
			provider.CapFetchPRComments:    true,
			provider.CapUpdatePRComment:    true,
			provider.CapCreateDependency:   true,
			provider.CapFetchDependencies:  true,
		},
	}
}

// New creates a new Azure DevOps provider instance.
func New(_ context.Context, cfg provider.Config) (any, error) {
	config := &Config{
		Token:         cfg.GetString("token"),
		Organization:  cfg.GetString("organization"),
		Project:       cfg.GetString("project"),
		AreaPath:      cfg.GetString("area_path"),
		IterationPath: cfg.GetString("iteration_path"),
		RepoName:      cfg.GetString("repo_name"),
		TargetBranch:  cfg.GetString("target_branch"),
		BranchPattern: cfg.GetString("branch_pattern"),
		CommitPrefix:  cfg.GetString("commit_prefix"),
	}

	if config.Organization == "" {
		return nil, ErrOrgRequired
	}
	if config.Project == "" {
		return nil, ErrProjectRequired
	}

	// Resolve token
	token, err := ResolveToken(config.Token)
	if err != nil {
		return nil, err
	}

	client := NewClient(config.Organization, config.Project, token)

	return &Provider{
		client: client,
		config: config,
	}, nil
}

// Match checks if the input looks like an Azure DevOps reference.
func (p *Provider) Match(input string) bool {
	// Check for azdo: or azure: prefix
	if strings.HasPrefix(input, "azdo:") || strings.HasPrefix(input, "azure:") {
		return true
	}

	// Check for Azure DevOps URL patterns
	if strings.Contains(input, "dev.azure.com") || strings.Contains(input, "visualstudio.com") {
		return true
	}

	// Check for bare work item ID or org/project#ID pattern
	_, err := ParseReference(input)

	return err == nil
}

// Parse parses an Azure DevOps reference and returns a canonical ID.
func (p *Provider) Parse(input string) (string, error) {
	ref, err := ParseReference(input)
	if err != nil {
		return "", err
	}

	return strconv.Itoa(ref.WorkItemID), nil
}

// Fetch retrieves a work item by its ID.
func (p *Provider) Fetch(ctx context.Context, id string) (*provider.WorkUnit, error) {
	ref, err := ParseReference(id)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// Override org/project if specified in reference
	if ref.Organization != "" && ref.Project != "" {
		p.client.SetOrganization(ref.Organization)
		p.client.SetProject(ref.Project)
	}

	workItem, err := p.client.GetWorkItem(ctx, ref.WorkItemID)
	if err != nil {
		return nil, fmt.Errorf("fetch work item %d: %w", ref.WorkItemID, err)
	}

	return p.workItemToWorkUnit(workItem), nil
}

// Snapshot creates a snapshot of the work item's current state.
func (p *Provider) Snapshot(ctx context.Context, id string) (*provider.Snapshot, error) {
	ref, err := ParseReference(id)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// Override org/project if specified in reference
	if ref.Organization != "" && ref.Project != "" {
		p.client.SetOrganization(ref.Organization)
		p.client.SetProject(ref.Project)
	}

	workItem, err := p.client.GetWorkItem(ctx, ref.WorkItemID)
	if err != nil {
		return nil, fmt.Errorf("snapshot work item %d: %w", ref.WorkItemID, err)
	}

	// Build markdown content
	content := buildSnapshotContent(workItem)

	return &provider.Snapshot{
		Type:    ProviderName,
		Ref:     fmt.Sprintf("azdo:%d", workItem.ID),
		Content: content,
	}, nil
}

// List retrieves work items based on filter criteria.
func (p *Provider) List(ctx context.Context, opts provider.ListOptions) ([]*provider.WorkUnit, error) {
	// Build WIQL query
	wiql := buildWIQLQuery(p.config, opts)

	ids, err := p.client.QueryWorkItems(ctx, wiql)
	if err != nil {
		return nil, fmt.Errorf("query work items: %w", err)
	}

	if len(ids) == 0 {
		return nil, nil
	}

	// Apply limit
	if opts.Limit > 0 && len(ids) > opts.Limit {
		ids = ids[:opts.Limit]
	}

	workItems, err := p.client.GetWorkItems(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get work items: %w", err)
	}

	var units []*provider.WorkUnit
	for _, wi := range workItems {
		units = append(units, p.workItemToWorkUnit(&wi))
	}

	return units, nil
}

// FetchComments retrieves comments for a work item.
func (p *Provider) FetchComments(ctx context.Context, id string) ([]provider.Comment, error) {
	ref, err := ParseReference(id)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	comments, err := p.client.GetWorkItemComments(ctx, ref.WorkItemID)
	if err != nil {
		return nil, fmt.Errorf("fetch comments for %d: %w", ref.WorkItemID, err)
	}

	var result []provider.Comment
	for _, comment := range comments {
		author := provider.Person{}
		if comment.CreatedBy != nil {
			author = provider.Person{
				ID:    comment.CreatedBy.ID,
				Name:  comment.CreatedBy.DisplayName,
				Email: comment.CreatedBy.UniqueName,
			}
		}

		result = append(result, provider.Comment{
			ID:        strconv.Itoa(comment.ID),
			Author:    author,
			Body:      comment.Text,
			CreatedAt: parseAzureTime(comment.CreatedDate),
		})
	}

	return result, nil
}

// AddComment adds a comment to a work item.
func (p *Provider) AddComment(ctx context.Context, id string, body string) (*provider.Comment, error) {
	ref, err := ParseReference(id)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	comment, err := p.client.AddWorkItemComment(ctx, ref.WorkItemID, body)
	if err != nil {
		return nil, fmt.Errorf("add comment to %d: %w", ref.WorkItemID, err)
	}

	author := provider.Person{}
	if comment.CreatedBy != nil {
		author = provider.Person{
			ID:    comment.CreatedBy.ID,
			Name:  comment.CreatedBy.DisplayName,
			Email: comment.CreatedBy.UniqueName,
		}
	}

	return &provider.Comment{
		ID:        strconv.Itoa(comment.ID),
		Author:    author,
		Body:      comment.Text,
		CreatedAt: parseAzureTime(comment.CreatedDate),
	}, nil
}

// UpdateStatus updates the work item state.
func (p *Provider) UpdateStatus(ctx context.Context, id string, status provider.Status) error {
	ref, err := ParseReference(id)
	if err != nil {
		return fmt.Errorf("parse reference: %w", err)
	}

	// Map provider status to Azure DevOps state
	azState := mapToAzureState(status)
	if azState == "" {
		return nil // No mapping for this status
	}

	_, err = p.client.UpdateWorkItemState(ctx, ref.WorkItemID, azState)
	if err != nil {
		return fmt.Errorf("update work item state %d: %w", ref.WorkItemID, err)
	}

	return nil
}

// CreatePullRequest creates a pull request.
// Work items can be linked automatically via AB#123 syntax in title/body.
func (p *Provider) CreatePullRequest(ctx context.Context, opts provider.PullRequestOptions) (*provider.PullRequest, error) {
	repoName := p.config.RepoName
	if repoName == "" {
		// Try to find default repository
		repos, err := p.client.GetRepositories(ctx)
		if err != nil {
			return nil, fmt.Errorf("get repositories: %w", err)
		}
		if len(repos) == 0 {
			return nil, errors.New("no repositories found in project")
		}
		repoName = repos[0].Name
	}

	targetBranch := opts.TargetBranch
	if targetBranch == "" {
		targetBranch = p.config.TargetBranch
	}
	if targetBranch == "" {
		// Query the repository for its default branch instead of assuming "main"
		repo, err := p.client.GetRepository(ctx, repoName)
		if err != nil {
			return nil, fmt.Errorf("get repository for default branch: %w", err)
		}
		// Azure DevOps returns default branch as "refs/heads/main"
		targetBranch = strings.TrimPrefix(repo.DefaultBranch, "refs/heads/")
		if targetBranch == "" {
			return nil, errors.New("repository has no default branch; set target_branch in config")
		}
	}

	// Extract work item IDs from title/body for auto-linking (AB#123 or #123 format)
	workItemIDs := ExtractWorkItemIDs(opts.Title + " " + opts.Body)

	pr, err := p.client.CreatePullRequest(ctx, repoName, opts.SourceBranch, targetBranch, opts.Title, opts.Body, workItemIDs)
	if err != nil {
		return nil, fmt.Errorf("create pull request: %w", err)
	}

	prURL := fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s/pullrequest/%d",
		p.config.Organization, p.config.Project, repoName, pr.PullRequestID)

	return &provider.PullRequest{
		ID:    strconv.Itoa(pr.PullRequestID),
		URL:   prURL,
		Title: pr.Title,
		State: pr.Status,
	}, nil
}

// LinkBranch links a branch to the work item.
func (p *Provider) LinkBranch(ctx context.Context, id string, branch string) error {
	ref, err := ParseReference(id)
	if err != nil {
		return fmt.Errorf("parse reference: %w", err)
	}

	// Add branch link relation
	updates := []PatchOperation{
		{
			Op:   "add",
			Path: "/relations/-",
			Value: map[string]any{
				"rel": "ArtifactLink",
				"url": fmt.Sprintf("vstfs:///Git/Ref/%s/%s/GB%s", p.config.Organization, p.config.Project, branch),
				"attributes": map[string]any{
					"name": "Branch",
				},
			},
		},
	}

	_, err = p.client.UpdateWorkItem(ctx, ref.WorkItemID, updates)
	if err != nil {
		return fmt.Errorf("link branch to %d: %w", ref.WorkItemID, err)
	}

	return nil
}

// --- Helper functions ---

func (p *Provider) workItemToWorkUnit(wi *WorkItem) *provider.WorkUnit {
	unit := &provider.WorkUnit{
		ID:          strconv.Itoa(wi.ID),
		ExternalID:  strconv.Itoa(wi.ID),
		ExternalKey: strconv.Itoa(wi.ID),
		Provider:    ProviderName,
		Title:       wi.Fields.Title,
		Description: wi.Fields.Description,
		Status:      mapAzureState(wi.Fields.State),
		Priority:    mapAzurePriority(wi.Fields.Priority),
		TaskType:    mapWorkItemType(wi.Fields.WorkItemType),
		Labels:      parseTags(wi.Fields.Tags),
		CreatedAt:   parseAzureTime(wi.Fields.CreatedDate),
		UpdatedAt:   parseAzureTime(wi.Fields.ChangedDate),
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: fmt.Sprintf("azdo:%d", wi.ID),
			SyncedAt:  time.Now(),
		},
		Metadata: map[string]any{
			"url":            wi.URL,
			"area_path":      wi.Fields.AreaPath,
			"iteration_path": wi.Fields.IterationPath,
			"work_item_type": wi.Fields.WorkItemType,
			"state":          wi.Fields.State,
			"reason":         wi.Fields.Reason,
		},
	}

	// Set assignee
	if wi.Fields.AssignedTo != nil {
		unit.Assignees = []provider.Person{
			{
				ID:    wi.Fields.AssignedTo.ID,
				Name:  wi.Fields.AssignedTo.DisplayName,
				Email: wi.Fields.AssignedTo.UniqueName,
			},
		}
	}

	// Add repro steps for bugs
	if wi.Fields.ReproSteps != "" {
		unit.Metadata["repro_steps"] = wi.Fields.ReproSteps
	}

	// Add acceptance criteria
	if wi.Fields.AcceptanceCriteria != "" {
		unit.Metadata["acceptance_criteria"] = wi.Fields.AcceptanceCriteria
	}

	// Extract attachments from relations
	if len(wi.Relations) > 0 {
		unit.Attachments = extractAttachments(wi.Relations)
	}

	return unit
}

func mapAzureState(state string) provider.Status {
	stateLower := strings.ToLower(state)
	switch {
	case strings.Contains(stateLower, "done") || strings.Contains(stateLower, "closed") || strings.Contains(stateLower, "resolved"):
		return provider.StatusClosed
	case strings.Contains(stateLower, "active") || strings.Contains(stateLower, "committed") || strings.Contains(stateLower, "in progress"):
		return provider.StatusInProgress
	case strings.Contains(stateLower, "review") || strings.Contains(stateLower, "pr"):
		return provider.StatusReview
	case strings.Contains(stateLower, "new") || strings.Contains(stateLower, "to do") || strings.Contains(stateLower, "proposed"):
		return provider.StatusOpen
	}

	return provider.StatusOpen
}

func mapToAzureState(status provider.Status) string {
	switch status {
	case provider.StatusClosed, provider.StatusDone:
		return "Done"
	case provider.StatusInProgress:
		return "Active"
	case provider.StatusOpen:
		return "New"
	case provider.StatusReview:
		return "Resolved" // Or could use custom state if available
	default:
		return ""
	}
}

func mapAzurePriority(priority int) provider.Priority {
	switch priority {
	case 1:
		return provider.PriorityCritical
	case 2:
		return provider.PriorityHigh
	case 3:
		return provider.PriorityNormal
	case 4:
		return provider.PriorityLow
	default:
		return provider.PriorityNormal
	}
}

func mapWorkItemType(wiType string) string {
	typeLower := strings.ToLower(wiType)
	switch {
	case strings.Contains(typeLower, "bug"):
		return "fix"
	case strings.Contains(typeLower, "feature") || strings.Contains(typeLower, "user story"):
		return "feature"
	case strings.Contains(typeLower, "task"):
		return "task"
	case strings.Contains(typeLower, "epic"):
		return "epic"
	case strings.Contains(typeLower, "issue"):
		return "issue"
	}

	return "task"
}

func parseTags(tags string) []string {
	if tags == "" {
		return nil
	}
	// Azure DevOps stores tags as semicolon-separated string
	parts := strings.Split(tags, ";")
	var result []string
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag != "" {
			result = append(result, tag)
		}
	}

	return result
}

// extractAttachments extracts attachments from work item relations.
// Azure DevOps stores attachments as relations with rel="AttachedFile".
func extractAttachments(relations []WorkItemRelation) []provider.Attachment {
	var attachments []provider.Attachment
	for _, rel := range relations {
		if rel.Rel == "AttachedFile" {
			// Extract filename from attributes if available
			name := ""
			if rel.Attributes != nil {
				if n, ok := rel.Attributes["name"].(string); ok {
					name = n
				}
			}
			// Use URL as the attachment ID for DownloadAttachment compatibility
			attachments = append(attachments, provider.Attachment{
				ID:   rel.URL,
				Name: name,
				URL:  rel.URL,
			})
		}
	}

	return attachments
}

func parseAzureTime(ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}

	// Azure DevOps uses ISO 8601 format
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02T15:04:05Z", ts); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02T15:04:05.999Z", ts); err == nil {
		return t
	}

	return time.Time{}
}

func buildWIQLQuery(config *Config, opts provider.ListOptions) string {
	var conditions []string

	// Filter by area path
	if config.AreaPath != "" {
		conditions = append(conditions, fmt.Sprintf("[System.AreaPath] UNDER '%s'", config.AreaPath))
	}

	// Filter by iteration path
	if config.IterationPath != "" {
		conditions = append(conditions, fmt.Sprintf("[System.IterationPath] UNDER '%s'", config.IterationPath))
	}

	// Filter by status
	switch opts.Status {
	case provider.StatusOpen:
		conditions = append(conditions, "[System.State] IN ('New', 'To Do', 'Proposed')")
	case provider.StatusInProgress:
		conditions = append(conditions, "[System.State] IN ('Active', 'In Progress', 'Committed')")
	case provider.StatusReview:
		conditions = append(conditions, "[System.State] IN ('Resolved', 'In Review')")
	case provider.StatusClosed, provider.StatusDone:
		conditions = append(conditions, "[System.State] IN ('Done', 'Closed', 'Removed')")
	}

	// Filter by labels (tags)
	for _, label := range opts.Labels {
		conditions = append(conditions, fmt.Sprintf("[System.Tags] CONTAINS '%s'", label))
	}

	query := "SELECT [System.Id] FROM WorkItems"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY [System.ChangedDate] DESC"

	return query
}

func buildSnapshotContent(wi *WorkItem) string {
	var sb strings.Builder

	// Title
	sb.WriteString("# ")
	sb.WriteString(wi.Fields.Title)
	sb.WriteString("\n\n")

	// Metadata
	sb.WriteString("**ID:** ")
	sb.WriteString(strconv.Itoa(wi.ID))
	sb.WriteString("\n")

	sb.WriteString("**Type:** ")
	sb.WriteString(wi.Fields.WorkItemType)
	sb.WriteString("\n")

	sb.WriteString("**State:** ")
	sb.WriteString(wi.Fields.State)
	sb.WriteString("\n")

	if wi.Fields.AssignedTo != nil {
		sb.WriteString("**Assigned To:** ")
		sb.WriteString(wi.Fields.AssignedTo.DisplayName)
		sb.WriteString("\n")
	}

	if wi.Fields.AreaPath != "" {
		sb.WriteString("**Area:** ")
		sb.WriteString(wi.Fields.AreaPath)
		sb.WriteString("\n")
	}

	if wi.Fields.IterationPath != "" {
		sb.WriteString("**Iteration:** ")
		sb.WriteString(wi.Fields.IterationPath)
		sb.WriteString("\n")
	}

	if wi.Fields.Priority > 0 {
		sb.WriteString("**Priority:** ")
		sb.WriteString(strconv.Itoa(wi.Fields.Priority))
		sb.WriteString("\n")
	}

	if wi.Fields.Tags != "" {
		sb.WriteString("**Tags:** ")
		sb.WriteString(wi.Fields.Tags)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Description
	if wi.Fields.Description != "" {
		sb.WriteString("## Description\n\n")
		sb.WriteString(wi.Fields.Description)
		sb.WriteString("\n")
	}

	// Repro steps for bugs
	if wi.Fields.ReproSteps != "" {
		sb.WriteString("\n## Repro Steps\n\n")
		sb.WriteString(wi.Fields.ReproSteps)
		sb.WriteString("\n")
	}

	// Acceptance criteria
	if wi.Fields.AcceptanceCriteria != "" {
		sb.WriteString("\n## Acceptance Criteria\n\n")
		sb.WriteString(wi.Fields.AcceptanceCriteria)
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetBranchSuggestion returns a suggested branch name for the work item.
func (p *Provider) GetBranchSuggestion(task *provider.WorkUnit) string {
	if p.config.BranchPattern == "" {
		// Default pattern
		return "ab" + task.ID
	}

	// Simple template replacement
	result := p.config.BranchPattern
	result = strings.ReplaceAll(result, "{key}", task.ExternalKey)
	result = strings.ReplaceAll(result, "{id}", task.ID)

	// Slugify title
	slug := naming.Slugify(task.Title, 50)
	result = strings.ReplaceAll(result, "{slug}", slug)

	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// PR Review Support (PRFetcher, PRCommenter, PRCommentFetcher, PRCommentUpdater)
// ─────────────────────────────────────────────────────────────────────────────

// FetchPullRequest retrieves pull request details.
func (p *Provider) FetchPullRequest(ctx context.Context, number int) (*provider.PullRequest, error) {
	pr, err := p.client.GetPullRequest(ctx, number)
	if err != nil {
		return nil, err
	}

	// Extract branch names from refs
	sourceBranch := strings.TrimPrefix(pr.SourceRefName, "refs/heads/")
	targetBranch := strings.TrimPrefix(pr.TargetRefName, "refs/heads/")

	// Parse creation date for timestamps
	createdAt := parseAzureTime(pr.CreationDate)
	// We'll use creation date for both if modified date is not available

	return &provider.PullRequest{
		ID:         strconv.Itoa(pr.PullRequestID),
		URL:        pr.URL,
		Title:      pr.Title,
		State:      pr.Status,
		Number:     pr.PullRequestID,
		Body:       pr.Description,
		HeadSHA:    "", // Not available in basic PR type
		HeadBranch: sourceBranch,
		BaseBranch: targetBranch,
		Author:     "", // Not available in basic PR type
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
		Labels:     nil, // Not available
		Assignees:  nil, // Would need to fetch reviewers separately
	}, nil
}

// FetchPullRequestDiff retrieves the diff for a pull request.
func (p *Provider) FetchPullRequestDiff(ctx context.Context, number int) (*provider.PullRequestDiff, error) {
	// Get PR details first for branch info
	pr, err := p.client.GetPullRequest(ctx, number)
	if err != nil {
		return nil, err
	}

	sourceBranch := strings.TrimPrefix(pr.SourceRefName, "refs/heads/")
	targetBranch := strings.TrimPrefix(pr.TargetRefName, "refs/heads/")

	// Get diff
	rawDiff, diffs, additions, deletions, err := p.client.GetPullRequestDiff(ctx, number)
	if err != nil {
		return nil, err
	}

	// Map files
	providerFiles := make([]provider.FileDiff, len(diffs))
	for i, d := range diffs {
		providerFiles[i] = provider.FileDiff{
			Path:      d.Path,
			Mode:      mapAzureChangeType(d.ChangeType),
			Patch:     d.Patch,
			Additions: 0, // Not separated in Azure DevOps diff
			Deletions: 0, // Not separated in Azure DevOps diff
		}
	}

	return &provider.PullRequestDiff{
		URL:        pr.URL + "/diff",
		BaseBranch: targetBranch,
		HeadBranch: sourceBranch,
		Files:      providerFiles,
		Patch:      rawDiff,
		Additions:  additions,
		Deletions:  deletions,
		Commits:    0, // Not available
	}, nil
}

// AddPullRequestComment posts a comment to a pull request.
func (p *Provider) AddPullRequestComment(ctx context.Context, number int, body string) (*provider.Comment, error) {
	thread, err := p.client.CreatePullRequestThread(ctx, number, body)
	if err != nil {
		return nil, err
	}

	// Convert thread to provider Comment
	if len(thread.Comments) > 0 {
		return mapPRThreadToProviderComment(thread, &thread.Comments[0]), nil
	}

	// Return empty comment if no comments returned
	return &provider.Comment{
		ID:   thread.ID,
		Body: body,
	}, nil
}

// FetchPullRequestComments retrieves all comments on a pull request.
func (p *Provider) FetchPullRequestComments(ctx context.Context, number int) ([]provider.Comment, error) {
	threads, err := p.client.GetPullRequestThreads(ctx, number)
	if err != nil {
		return nil, err
	}

	var result []provider.Comment
	for _, t := range threads {
		for i := range t.Comments {
			// Skip system comments
			if t.Comments[i].CommentType == "system" {
				continue
			}
			result = append(result, *mapPRThreadToProviderComment(&t, &t.Comments[i]))
		}
	}

	return result, nil
}

// UpdatePullRequestComment updates an existing comment on a pull request.
func (p *Provider) UpdatePullRequestComment(ctx context.Context, number int, commentID string, body string) (*provider.Comment, error) {
	// Azure DevOps uses threadID/commentID structure
	// For simplicity, we'll create a new comment thread
	// In a full implementation, you'd need to parse the thread ID from the comment ID
	thread, err := p.client.CreatePullRequestThread(ctx, number, body)
	if err != nil {
		return nil, err
	}

	if len(thread.Comments) > 0 {
		return mapPRThreadToProviderComment(thread, &thread.Comments[0]), nil
	}

	return &provider.Comment{
		ID:   thread.ID,
		Body: body,
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper functions for PR review support
// ─────────────────────────────────────────────────────────────────────────────

// mapPRThreadToProviderComment converts an Azure DevOps PR thread comment to a provider Comment.
func mapPRThreadToProviderComment(_ *PRThread, comment *PRThreadComment) *provider.Comment {
	author := ""
	if comment.Author != nil {
		author = comment.Author.DisplayName
	}

	return &provider.Comment{
		ID:        strconv.Itoa(comment.ID),
		Body:      comment.Content,
		Author:    provider.Person{ID: author, Name: author},
		CreatedAt: parseAzureTime(comment.PublishedDate),
		UpdatedAt: parseAzureTime(comment.PublishedDate),
	}
}

// mapAzureChangeType maps Azure DevOps change type to provider file mode.
func mapAzureChangeType(changeType string) string {
	switch changeType {
	case "add":
		return "added"
	case "delete", "sourceDeleted":
		return "deleted"
	case "edit":
		return "modified"
	case "rename":
		return "renamed"
	default:
		return "modified"
	}
}
