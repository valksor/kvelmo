package empty

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
)

const ProviderName = "empty"

// ParseMetadataFromNotes parses metadata directives from note content.
// Supported formats (case-insensitive, must be at start of note):
//
//	@status: in_progress
//	@priority: high
//	@labels: bug,urgent
//
// Label parsing supports:
//   - Comma-separated values
//   - Trailing commas are ignored
//   - Basic quote stripping (removes surrounding quotes from labels)
//
// Limitations:
//   - Does not support quoted strings with commas inside them
//   - For complex label parsing, use frontmatter instead
//   - Validation of status/priority values is the caller's responsibility
//
// Returns parsed status, priority, and labels. Unset fields are returned as empty/nil.
// If multiple notes have the same metadata key, the last one wins.
func ParseMetadataFromNotes(notes []string) (string, string, []string) {
	var status, priority string
	var labels []string

	for _, note := range notes {
		note = strings.TrimSpace(note)
		if note == "" {
			continue
		}

		// Only match if the note STARTS with the metadata tag (case-insensitive)
		// This prevents false positives like "I checked the @status: flag"
		lowerNote := strings.ToLower(note)

		if strings.HasPrefix(lowerNote, "@status:") {
			// Trim the @status: prefix from original (to preserve case of value)
			status = strings.TrimSpace(note[len("@status:"):])
		}
		if strings.HasPrefix(lowerNote, "@priority:") {
			priority = strings.TrimSpace(note[len("@priority:"):])
		}
		if strings.HasPrefix(lowerNote, "@labels:") {
			labelsStr := strings.TrimSpace(note[len("@labels:"):])
			labels = parseLabels(labelsStr)
		}
	}

	return status, priority, labels
}

// parseLabels parses a comma-separated label string.
// Supports trailing commas and basic quote stripping.
func parseLabels(s string) []string {
	if s == "" {
		return nil
	}

	labelParts := strings.Split(s, ",")
	labels := make([]string, 0, len(labelParts))
	for _, part := range labelParts {
		label := strings.TrimSpace(part)
		// Strip surrounding quotes if present
		label = strings.Trim(label, `"`)
		if label != "" {
			labels = append(labels, label)
		}
	}

	// Return nil instead of empty slice for consistency
	if len(labels) == 0 {
		return nil
	}

	return labels
}

type Provider struct{}

// Info returns provider metadata.
func Info() provider.ProviderInfo {
	return provider.ProviderInfo{
		Name:        ProviderName,
		Description: "Empty task source for creating tasks from scratch",
		Schemes:     []string{"empty"},
		Priority:    5,
		Capabilities: provider.CapabilitySet{
			provider.CapRead: true,
		},
	}
}

// New creates an empty provider.
func New(ctx context.Context, cfg provider.Config) (any, error) {
	return &Provider{}, nil
}

// Match checks if input has the empty: scheme prefix.
func (p *Provider) Match(input string) bool {
	return strings.HasPrefix(input, "empty:")
}

// Parse extracts the task identifier from input.
// Input: "empty:A-1" → ID: "A-1".
// Input: "empty:Implement auth" → ID: "Implement auth".
func (p *Provider) Parse(input string) (string, error) {
	identifier := strings.TrimPrefix(input, "empty:")
	if identifier == "" {
		return "", errors.New("empty task identifier after 'empty:' prefix")
	}

	return identifier, nil
}

// Fetch creates a minimal WorkUnit with empty description.
func (p *Provider) Fetch(ctx context.Context, id string) (*provider.WorkUnit, error) {
	now := time.Now()

	// ID is the task identifier (e.g., "A-1" or "Implement auth")
	// Title is set to the identifier
	// Description is intentionally empty - user will add via 'mehr note'
	wu := &provider.WorkUnit{
		ID:          id,
		ExternalID:  id,
		Provider:    ProviderName,
		Title:       id,
		Description: "", // Empty - user adds via 'mehr note'
		Status:      provider.StatusOpen,
		Priority:    provider.PriorityNormal,
		Labels:      []string{},
		Metadata:    make(map[string]any),
		CreatedAt:   now,
		UpdatedAt:   now,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: "empty:" + id,
			SyncedAt:  now,
		},
		ExternalKey: id,
		TaskType:    "task",
		Slug:        "",
	}

	return wu, nil
}

// Register adds empty provider to registry.
func Register(r *provider.Registry) {
	_ = r.Register(Info(), New)
}
