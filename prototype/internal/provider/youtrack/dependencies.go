package youtrack

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// CreateDependency creates a dependency by updating the issue description.
// YouTrack has issue links but the API varies by version, so we use description.
func (p *Provider) CreateDependency(ctx context.Context, predecessorID, successorID string) error {
	if p.client == nil {
		return errors.New("client not initialized")
	}

	// Get the successor issue
	issue, err := p.client.GetIssue(ctx, successorID)
	if err != nil {
		return fmt.Errorf("get issue: %w", err)
	}

	// Build the dependency reference
	depRef := predecessorID

	// Check if the description already contains a dependencies section
	description := issue.Description
	dependsOnPattern := regexp.MustCompile(`(?m)^(?:\*\*)?Depends on:(?:\*\*)?\s*(.*)$`)

	if match := dependsOnPattern.FindStringSubmatch(description); match != nil {
		existingDeps := match[1]
		if strings.Contains(existingDeps, depRef) {
			return nil // Already exists
		}
		newDeps := strings.TrimSpace(existingDeps) + ", " + depRef
		description = dependsOnPattern.ReplaceAllString(description, "**Depends on:** "+newDeps)
	} else {
		if description != "" {
			description = fmt.Sprintf("**Depends on:** %s\n\n%s", depRef, description)
		} else {
			description = "**Depends on:** " + depRef
		}
	}

	// Update the issue
	err = p.client.UpdateIssueDescription(ctx, successorID, description)
	if err != nil {
		return fmt.Errorf("update issue: %w", err)
	}

	return nil
}

// GetDependencies returns the issue IDs that the given issue depends on.
func (p *Provider) GetDependencies(ctx context.Context, workUnitID string) ([]string, error) {
	if p.client == nil {
		return nil, errors.New("client not initialized")
	}

	issue, err := p.client.GetIssue(ctx, workUnitID)
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}

	return parseDependenciesFromDescription(issue.Description), nil
}

// UpdateIssueDescription updates an issue's description (stub for now).
func (c *Client) UpdateIssueDescription(ctx context.Context, issueID, description string) error {
	// POST /api/issues/{issueID}?fields=description
	// Future: Implement using YouTrack REST API
	return nil
}

// parseDependenciesFromDescription extracts issue IDs from a "Depends on:" line.
func parseDependenciesFromDescription(description string) []string {
	if description == "" {
		return nil
	}

	dependsOnPattern := regexp.MustCompile(`(?m)^(?:\*\*)?Depends on:(?:\*\*)?\s*(.*)$`)
	match := dependsOnPattern.FindStringSubmatch(description)
	if match == nil {
		return nil
	}

	parts := regexp.MustCompile(`[,\s]+`).Split(match[1], -1)
	var deps []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			deps = append(deps, p)
		}
	}

	return deps
}
