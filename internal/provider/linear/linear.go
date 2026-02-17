package linear

import (
	"context"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/capability"
	"github.com/valksor/go-toolkit/providerconfig"
	"github.com/valksor/go-toolkit/slug"
	"github.com/valksor/go-toolkit/workunit"
)

// ProviderName is the registered name for this provider.
const ProviderName = "linear"

// Provider handles Linear issue tasks.
type Provider struct {
	client *Client
	team   string // Default team key
}

// Config holds Linear provider configuration.
type Config struct {
	Token string
	Team  string // Default team key for operations
}

// Info returns provider metadata.
func Info() provider.ProviderInfo {
	return provider.ProviderInfo{
		Name:        ProviderName,
		Description: "Linear issue source",
		Schemes:     []string{"linear", "ln"},
		Priority:    20, // Same as GitHub and Wrike
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

// New creates a Linear provider.
func New(_ context.Context, cfg providerconfig.Config) (any, error) {
	token := cfg.GetString("token")
	team := cfg.GetString("team")

	// Try to resolve token from env if not provided
	if token == "" {
		resolvedToken, err := ResolveToken("")
		if err != nil {
			return nil, err
		}
		token = resolvedToken
	}

	return &Provider{
		client: NewClient(token),
		team:   team,
	}, nil
}

// Match checks if input has the linear: or ln: scheme prefix.
func (p *Provider) Match(input string) bool {
	return strings.HasPrefix(input, "linear:") || strings.HasPrefix(input, "ln:")
}

// Parse extracts the issue reference from input.
func (p *Provider) Parse(input string) (string, error) {
	ref, err := ParseReference(input)
	if err != nil {
		return "", err
	}

	return ref.IssueID, nil
}

// Fetch reads a Linear issue and creates a WorkUnit.
func (p *Provider) Fetch(ctx context.Context, id string) (*workunit.WorkUnit, error) {
	ref, err := ParseReference(id)
	if err != nil {
		return nil, err
	}

	// Fetch issue from Linear
	issue, err := p.client.GetIssue(ctx, ref.IssueID)
	if err != nil {
		return nil, err
	}

	// Map to WorkUnit
	wu := &workunit.WorkUnit{
		ID:          issue.ID,
		ExternalID:  issue.Identifier,
		Provider:    ProviderName,
		Title:       issue.Title,
		Description: issue.Description,
		Status:      mapLinearStatus(issue.State),
		Priority:    mapLinearPriority(issue.Priority),
		Labels:      extractLabelNames(issue.Labels),
		Assignees:   mapAssignees(issue.Assignee),
		CreatedAt:   issue.CreatedAt,
		UpdatedAt:   issue.UpdatedAt,
		Source: workunit.SourceInfo{
			Type:      ProviderName,
			Reference: issue.Identifier,
			SyncedAt:  time.Now(),
		},
		// Naming fields for branch/commit customization
		ExternalKey: issue.Identifier,
		TaskType:    "issue",
		Slug:        slug.Slugify(issue.Title, 50),
		Metadata:    buildMetadata(issue),
	}

	// Fetch comments if available
	comments, err := p.client.GetComments(ctx, issue.ID)
	if err == nil && len(comments) > 0 {
		wu.Comments = mapComments(comments)
	}

	// Map attachments if available
	if issue.Attachments != nil && len(issue.Attachments.Nodes) > 0 {
		wu.Attachments = mapAttachments(issue.Attachments.Nodes)
	}

	return wu, nil
}

// GetConfig returns the provider configuration.
func (p *Provider) GetConfig() *Config {
	return &Config{
		Team: p.team,
	}
}

// GetClient returns the Linear API client.
func (p *Provider) GetClient() *Client {
	return p.client
}

// GetDefaultTeam returns the default team key.
func (p *Provider) GetDefaultTeam() string {
	return p.team
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper functions
// ──────────────────────────────────────────────────────────────────────────────

// mapLinearStatus converts Linear state to provider status.
func mapLinearStatus(state *State) workunit.Status {
	if state == nil {
		return workunit.StatusOpen
	}

	switch strings.ToLower(state.Name) {
	case "backlog", "todo", "unstarted":
		return workunit.StatusOpen
	case "in progress", "started", "in review":
		return workunit.StatusInProgress
	case "done", "completed", "closed":
		return workunit.StatusDone
	case "canceled", "cancelled":
		return workunit.StatusClosed
	default:
		// Check state type as fallback
		switch strings.ToLower(state.Type) {
		case "backlog", "unstarted":
			return workunit.StatusOpen
		case "started", "in_progress":
			return workunit.StatusInProgress
		case "completed", "canceled":
			return workunit.StatusDone
		default:
			return workunit.StatusOpen
		}
	}
}

// mapLinearPriority converts Linear priority to provider priority
// Linear priority: 0 = No priority, 1 = Urgent, 2 = High, 3 = Medium, 4 = Low.
func mapLinearPriority(priority int) workunit.Priority {
	switch priority {
	case 1: // Urgent
		return workunit.PriorityCritical
	case 2: // High
		return workunit.PriorityHigh
	case 4: // Low
		return workunit.PriorityLow
	case 0, 3: // No priority, Medium
		return workunit.PriorityNormal
	default:
		return workunit.PriorityNormal
	}
}

// mapProviderPriorityToLinear converts provider priority to Linear priority.
func mapProviderPriorityToLinear(priority workunit.Priority) *int {
	var p int
	switch priority {
	case workunit.PriorityCritical:
		p = 1 // Urgent
	case workunit.PriorityHigh:
		p = 2 // High
	case workunit.PriorityNormal:
		p = 3 // Medium
	case workunit.PriorityLow:
		p = 4 // Low
	}

	return &p
}

// mapProviderStatusToLinearStateName converts provider status to Linear state name.
func mapProviderStatusToLinearStateName(status workunit.Status) string {
	switch status {
	case workunit.StatusOpen:
		return "Todo"
	case workunit.StatusInProgress:
		return "In Progress"
	case workunit.StatusReview:
		return "In Review"
	case workunit.StatusDone:
		return "Done"
	case workunit.StatusClosed:
		return "Canceled"
	default:
		return "Todo"
	}
}

// extractLabelNames extracts label names from Linear labels.
func extractLabelNames(labels *LabelConnection) []string {
	if labels == nil || len(labels.Nodes) == 0 {
		return []string{}
	}
	names := make([]string, len(labels.Nodes))
	for i, label := range labels.Nodes {
		names[i] = label.Name
	}

	return names
}

func mapAttachments(attachments []*Attachment) []workunit.Attachment {
	result := make([]workunit.Attachment, 0, len(attachments))
	for _, a := range attachments {
		result = append(result, workunit.Attachment{
			ID:        a.URL, // Use URL as ID for DownloadAttachment compatibility
			Name:      a.Title,
			URL:       a.URL,
			CreatedAt: a.CreatedAt,
		})
	}

	return result
}

// mapAssignees converts Linear assignee to provider Person.
func mapAssignees(assignee *User) []workunit.Person {
	if assignee == nil {
		return []workunit.Person{}
	}

	return []workunit.Person{
		{
			ID:    assignee.ID,
			Name:  assignee.Name,
			Email: assignee.Email,
		},
	}
}

// mapComments converts Linear comments to provider comments.
func mapComments(comments []*Comment) []workunit.Comment {
	if comments == nil {
		return nil
	}

	result := make([]workunit.Comment, 0, len(comments))
	for _, c := range comments {
		var author workunit.Person
		if c.User != nil {
			author = workunit.Person{
				ID:   c.User.ID,
				Name: c.User.Name,
			}
		}
		result = append(result, workunit.Comment{
			ID:        c.ID,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			Author:    author,
		})
	}

	return result
}

// buildMetadata creates metadata map from issue.
func buildMetadata(issue *Issue) map[string]any {
	metadata := make(map[string]any)

	metadata["url"] = issue.URL
	metadata["state_id"] = issue.State.ID
	metadata["state_name"] = issue.State.Name
	metadata["state_type"] = issue.State.Type
	metadata["identifier"] = issue.Identifier

	if issue.Team != nil {
		metadata["team_key"] = issue.Team.Key
		metadata["team_name"] = issue.Team.Name
	}

	return metadata
}
