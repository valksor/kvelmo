package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	gh "github.com/google/go-github/v67/github"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/cache"
	"github.com/valksor/go-toolkit/capability"
	"github.com/valksor/go-toolkit/providerconfig"
	"github.com/valksor/go-toolkit/slug"
	"github.com/valksor/go-toolkit/snapshot"
	"github.com/valksor/go-toolkit/workunit"
)

// ProviderName is the registered name for this provider.
const ProviderName = "github"

// Provider handles GitHub issue tasks.
type Provider struct {
	client *Client
	owner  string
	repo   string
	config *Config
	cache  *cache.Cache
}

// Config holds GitHub provider configuration.
type Config struct {
	Token         string
	Owner         string
	Repo          string
	BranchPattern string // Default: "issue/{key}-{slug}"
	CommitPrefix  string // Default: "[#{key}]"
	TargetBranch  string // Default: detected from repo
	DraftPR       bool
	Comments      *CommentsConfig
}

// CommentsConfig controls automated commenting.
type CommentsConfig struct {
	Enabled         bool
	OnBranchCreated bool
	OnPlanDone      bool
	OnImplementDone bool
	OnPRCreated     bool
}

// Info returns provider metadata.
func Info() provider.ProviderInfo {
	return provider.ProviderInfo{
		Name:        ProviderName,
		Description: "GitHub Issues task source",
		Schemes:     []string{"github", "gh"},
		Priority:    20, // Higher than file/directory
		Capabilities: capability.CapabilitySet{
			capability.CapRead:               true,
			capability.CapList:               true,
			capability.CapFetchComments:      true,
			capability.CapComment:            true,
			capability.CapUpdateStatus:       true,
			capability.CapManageLabels:       true,
			capability.CapCreateWorkUnit:     true,
			capability.CapCreatePR:           true,
			capability.CapDownloadAttachment: true,
			capability.CapSnapshot:           true,
			capability.CapFetchSubtasks:      true,
			capability.CapFetchParent:        true,
			capability.CapFetchPR:            true,
			capability.CapPRComment:          true,
			capability.CapFetchPRComments:    true,
			capability.CapUpdatePRComment:    true,
			capability.CapCreateDependency:   true,
			capability.CapFetchDependencies:  true,
		},
	}
}

// New creates a GitHub provider.
func New(ctx context.Context, cfg providerconfig.Config) (any, error) {
	// Validate configuration
	if err := provider.ValidateConfig("github", cfg, func(v *provider.Validator) {
		// Token is required (may be resolved from config or env)
		// Skip validation here as ResolveToken handles it with better error messages
		// Owner and repo are optional - can be specified in reference
	}); err != nil {
		return nil, err
	}

	// Extract config values
	token := cfg.GetString("token")
	owner := cfg.GetString("owner")
	repo := cfg.GetString("repo")
	branchPattern := cfg.GetString("branch_pattern")
	commitPrefix := cfg.GetString("commit_prefix")
	targetBranch := cfg.GetString("target_branch")
	draftPR := cfg.GetBool("draft_pr")

	// Resolve token
	resolvedToken, err := ResolveToken(token)
	if err != nil {
		return nil, err
	}

	// Set defaults
	if branchPattern == "" {
		branchPattern = "issue/{key}-{slug}"
	}
	if commitPrefix == "" {
		commitPrefix = "[#{key}]"
	}

	// Parse comments config
	commentsEnabled := cfg.GetBool("comments.enabled")
	comments := &CommentsConfig{
		Enabled:         commentsEnabled,
		OnBranchCreated: cfg.GetBool("comments.on_branch_created"),
		OnPlanDone:      cfg.GetBool("comments.on_plan_done"),
		OnImplementDone: cfg.GetBool("comments.on_implement_done"),
		OnPRCreated:     cfg.GetBool("comments.on_pr_created"),
	}

	config := &Config{
		Token:         resolvedToken,
		Owner:         owner,
		Repo:          repo,
		BranchPattern: branchPattern,
		CommitPrefix:  commitPrefix,
		TargetBranch:  targetBranch,
		DraftPR:       draftPR,
		Comments:      comments,
	}

	// Create cache (enabled by default, can be disabled via config or SetCache(nil))
	providerCache := cache.New()
	if cfg.GetBool("cache.disabled") {
		providerCache.Disable()
	}

	return &Provider{
		client: NewClientWithCache(ctx, resolvedToken, owner, repo, providerCache),
		owner:  owner,
		repo:   repo,
		config: config,
		cache:  providerCache,
	}, nil
}

// Match checks if input has the github: or gh: scheme prefix.
func (p *Provider) Match(input string) bool {
	return strings.HasPrefix(input, "github:") || strings.HasPrefix(input, "gh:")
}

// Parse extracts the issue reference from input.
func (p *Provider) Parse(input string) (string, error) {
	ref, err := ParseReference(input)
	if err != nil {
		return "", err
	}

	// If explicit owner/repo provided, use it
	if ref.IsExplicit {
		return fmt.Sprintf("%s/%s#%d", ref.Owner, ref.Repo, ref.IssueNumber), nil
	}

	// Otherwise, check if we have owner/repo configured
	owner := p.owner
	repo := p.repo

	if owner == "" || repo == "" {
		return "", fmt.Errorf("%w: use github:owner/repo#N format or configure github.owner and github.repo", ErrRepoNotConfigured)
	}

	return fmt.Sprintf("%s/%s#%d", owner, repo, ref.IssueNumber), nil
}

// Fetch reads a GitHub issue and creates a WorkUnit.
func (p *Provider) Fetch(ctx context.Context, id string) (*workunit.WorkUnit, error) {
	ref, err := ParseReference(id)
	if err != nil {
		return nil, err
	}

	// Determine owner/repo
	owner := ref.Owner
	repo := ref.Repo
	if owner == "" {
		owner = p.owner
	}
	if repo == "" {
		repo = p.repo
	}
	if owner == "" || repo == "" {
		return nil, ErrRepoNotConfigured
	}

	// Update client with correct owner/repo
	p.client.SetOwnerRepo(owner, repo)

	// Fetch issue
	issue, err := p.client.GetIssue(ctx, ref.IssueNumber)
	if err != nil {
		return nil, err
	}

	// Map to WorkUnit
	wu := &workunit.WorkUnit{
		ID:          strconv.Itoa(issue.GetNumber()),
		ExternalID:  fmt.Sprintf("%s/%s#%d", owner, repo, issue.GetNumber()),
		Provider:    ProviderName,
		Title:       issue.GetTitle(),
		Description: issue.GetBody(),
		Status:      mapGitHubState(issue.GetState()),
		Priority:    inferPriorityFromLabels(issue.Labels),
		Labels:      extractLabelNames(issue.Labels),
		Assignees:   mapAssignees(issue.Assignees),
		CreatedAt:   issue.GetCreatedAt().Time,
		UpdatedAt:   issue.GetUpdatedAt().Time,
		Source: workunit.SourceInfo{
			Type:      ProviderName,
			Reference: id,
			SyncedAt:  time.Now(),
		},

		// Naming fields for branch/commit customization
		ExternalKey: strconv.Itoa(issue.GetNumber()),
		TaskType:    inferTypeFromLabels(issue.Labels),
		Slug:        slug.Slugify(issue.GetTitle(), 50),

		Metadata: map[string]any{
			"html_url":       issue.GetHTMLURL(),
			"owner":          owner,
			"repo":           repo,
			"issue_number":   issue.GetNumber(),
			"labels_raw":     issue.Labels,
			"branch_pattern": p.config.BranchPattern,
			"commit_prefix":  p.config.CommitPrefix,
		},
	}

	// Add milestone if present
	if issue.Milestone != nil && issue.Milestone.GetTitle() != "" {
		wu.Metadata["milestone"] = issue.Milestone.GetTitle()
	}

	// Fetch comments if available
	comments, err := p.client.GetIssueComments(ctx, ref.IssueNumber)
	if err == nil && len(comments) > 0 {
		wu.Comments = mapComments(comments)
	}

	// Extract linked issues
	linkedNums := ExtractLinkedIssues(issue.GetBody())
	if len(linkedNums) > 0 {
		wu.Metadata["linked_issues"] = linkedNums
	}

	// Extract image URLs
	imageURLs := ExtractImageURLs(issue.GetBody())
	if len(imageURLs) > 0 {
		wu.Attachments = make([]workunit.Attachment, len(imageURLs))
		for i, url := range imageURLs {
			wu.Attachments[i] = workunit.Attachment{
				ID:   AttachmentIDFromURL(url), // Stable ID based on URL hash
				Name: fmt.Sprintf("image-%d", i+1),
				URL:  url,
			}
		}
	}

	return wu, nil
}

// Snapshot captures the issue content for storage.
func (p *Provider) Snapshot(ctx context.Context, id string) (*snapshot.Snapshot, error) {
	ref, err := ParseReference(id)
	if err != nil {
		return nil, err
	}

	owner := ref.Owner
	repo := ref.Repo
	if owner == "" {
		owner = p.owner
	}
	if repo == "" {
		repo = p.repo
	}
	if owner == "" || repo == "" {
		return nil, ErrRepoNotConfigured
	}

	p.client.SetOwnerRepo(owner, repo)

	// Fetch main issue
	issue, err := p.client.GetIssue(ctx, ref.IssueNumber)
	if err != nil {
		return nil, err
	}

	snap := &snapshot.Snapshot{
		Type: ProviderName,
		Ref:  id,
		Files: []snapshot.SnapshotFile{
			{
				Path:    "issue.md",
				Content: formatIssueMarkdown(issue),
			},
		},
	}

	// Fetch and include comments
	comments, err := p.client.GetIssueComments(ctx, ref.IssueNumber)
	if err == nil && len(comments) > 0 {
		snap.Files = append(snap.Files, snapshot.SnapshotFile{
			Path:    "comments.md",
			Content: formatCommentsMarkdown(comments),
		})
	}

	// Fetch linked issues
	linkedNums := ExtractLinkedIssues(issue.GetBody())
	for _, num := range linkedNums {
		if num == ref.IssueNumber {
			continue // Skip self-reference
		}
		linked, err := p.client.GetIssue(ctx, num)
		if err != nil {
			continue // Skip issues we can't fetch
		}
		snap.Files = append(snap.Files, snapshot.SnapshotFile{
			Path:    fmt.Sprintf("linked/issue-%d.md", num),
			Content: formatIssueMarkdown(linked),
		})
	}

	return snap, nil
}

// GetConfig returns the provider configuration.
func (p *Provider) GetConfig() *Config {
	return p.config
}

// GetClient returns the GitHub API client.
func (p *Provider) GetClient() *Client {
	return p.client
}

// SetCache sets or replaces the cache for this provider and its client.
func (p *Provider) SetCache(c *cache.Cache) {
	p.cache = c
	p.client.SetCache(c)
}

// GetCache returns the cache for this provider.
func (p *Provider) GetCache() *cache.Cache {
	return p.cache
}

// --- Helper functions ---

func mapGitHubState(state string) workunit.Status {
	switch state {
	case "open":
		return workunit.StatusOpen
	case "closed":
		return workunit.StatusClosed
	default:
		return workunit.StatusOpen
	}
}

// labelTypeMap maps GitHub labels to task types.
var labelTypeMap = map[string]string{
	"bug":           "fix",
	"bugfix":        "fix",
	"fix":           "fix",
	"feature":       "feature",
	"enhancement":   "feature",
	"docs":          "docs",
	"documentation": "docs",
	"refactor":      "refactor",
	"chore":         "chore",
	"test":          "test",
	"ci":            "ci",
}

func inferTypeFromLabels(labels []*gh.Label) string {
	for _, label := range labels {
		name := strings.ToLower(label.GetName())
		if t, ok := labelTypeMap[name]; ok {
			return t
		}
	}

	return "issue"
}

// labelPriorityMap maps GitHub labels to priorities.
var labelPriorityMap = map[string]workunit.Priority{
	"critical":      workunit.PriorityCritical,
	"urgent":        workunit.PriorityCritical,
	"priority:high": workunit.PriorityHigh,
	"high-priority": workunit.PriorityHigh,
	"priority:low":  workunit.PriorityLow,
	"low-priority":  workunit.PriorityLow,
}

func inferPriorityFromLabels(labels []*gh.Label) workunit.Priority {
	for _, label := range labels {
		name := strings.ToLower(label.GetName())
		if p, ok := labelPriorityMap[name]; ok {
			return p
		}
	}

	return workunit.PriorityNormal
}

func extractLabelNames(labels []*gh.Label) []string {
	names := make([]string, len(labels))
	for i, label := range labels {
		names[i] = label.GetName()
	}

	return names
}

func mapAssignees(assignees []*gh.User) []workunit.Person {
	persons := make([]workunit.Person, len(assignees))
	for i, u := range assignees {
		persons[i] = workunit.Person{
			ID:    strconv.FormatInt(u.GetID(), 10),
			Name:  u.GetLogin(),
			Email: u.GetEmail(),
		}
	}

	return persons
}

func mapComments(comments []*gh.IssueComment) []workunit.Comment {
	result := make([]workunit.Comment, len(comments))
	for i, c := range comments {
		result[i] = workunit.Comment{
			ID:        strconv.FormatInt(c.GetID(), 10),
			Body:      c.GetBody(),
			CreatedAt: c.GetCreatedAt().Time,
			Author: workunit.Person{
				ID:   strconv.FormatInt(c.GetUser().GetID(), 10),
				Name: c.GetUser().GetLogin(),
			},
		}
	}

	return result
}

func formatIssueMarkdown(issue *gh.Issue) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# #%d: %s\n\n", issue.GetNumber(), issue.GetTitle()))

	// Metadata
	sb.WriteString("## Metadata\n\n")
	sb.WriteString(fmt.Sprintf("- **State:** %s\n", issue.GetState()))
	sb.WriteString(fmt.Sprintf("- **Created:** %s\n", issue.GetCreatedAt().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- **Updated:** %s\n", issue.GetUpdatedAt().Format(time.RFC3339)))

	if issue.GetUser() != nil {
		sb.WriteString(fmt.Sprintf("- **Author:** @%s\n", issue.GetUser().GetLogin()))
	}

	if len(issue.Labels) > 0 {
		labels := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			labels[i] = l.GetName()
		}
		sb.WriteString(fmt.Sprintf("- **Labels:** %s\n", strings.Join(labels, ", ")))
	}

	if len(issue.Assignees) > 0 {
		assignees := make([]string, len(issue.Assignees))
		for i, a := range issue.Assignees {
			assignees[i] = "@" + a.GetLogin()
		}
		sb.WriteString(fmt.Sprintf("- **Assignees:** %s\n", strings.Join(assignees, ", ")))
	}

	sb.WriteString(fmt.Sprintf("- **URL:** %s\n", issue.GetHTMLURL()))

	// Body
	sb.WriteString("\n## Description\n\n")
	sb.WriteString(issue.GetBody())
	sb.WriteString("\n")

	return sb.String()
}

func formatCommentsMarkdown(comments []*gh.IssueComment) string {
	var sb strings.Builder

	sb.WriteString("# Comments\n\n")

	for _, c := range comments {
		sb.WriteString(fmt.Sprintf("## Comment by @%s\n\n", c.GetUser().GetLogin()))
		sb.WriteString(fmt.Sprintf("*%s*\n\n", c.GetCreatedAt().Format(time.RFC3339)))
		sb.WriteString(c.GetBody())
		sb.WriteString("\n\n---\n\n")
	}

	return sb.String()
}
