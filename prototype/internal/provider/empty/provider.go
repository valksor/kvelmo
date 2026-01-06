package empty

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
)

const ProviderName = "empty"

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
