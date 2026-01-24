package jira

import (
	"context"
	"errors"
	"fmt"
)

// CreateDependency creates a "blocks" link between two issues.
// In Jira, the predecessor blocks the successor.
func (p *Provider) CreateDependency(ctx context.Context, predecessorID, successorID string) error {
	if p.client == nil {
		return errors.New("client not initialized")
	}

	// Jira uses issue links with "Blocks" link type
	// The predecessor blocks the successor
	linkType := "Blocks" // Standard Jira link type

	err := p.client.CreateIssueLink(ctx, predecessorID, successorID, linkType)
	if err != nil {
		return fmt.Errorf("create issue link: %w", err)
	}

	return nil
}

// GetDependencies returns the issue keys that the given issue depends on (is blocked by).
func (p *Provider) GetDependencies(ctx context.Context, workUnitID string) ([]string, error) {
	if p.client == nil {
		return nil, errors.New("client not initialized")
	}

	links, err := p.client.GetIssueLinks(ctx, workUnitID)
	if err != nil {
		return nil, fmt.Errorf("get issue links: %w", err)
	}

	// Extract inward links where this issue "is blocked by" another
	var deps []string
	for _, link := range links {
		if link.Type.Name == "Blocks" && link.InwardIssue != nil {
			deps = append(deps, link.InwardIssue.Key)
		}
	}

	return deps, nil
}
