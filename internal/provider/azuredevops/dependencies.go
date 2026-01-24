package azuredevops

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// CreateDependency creates a predecessor link between two work items.
// In Azure DevOps, we use the "Predecessor" link type.
func (p *Provider) CreateDependency(ctx context.Context, predecessorID, successorID string) error {
	if p.client == nil {
		return errors.New("client not initialized")
	}

	// Azure DevOps uses work item links
	// System.LinkTypes.Dependency-Forward means "Successor"
	// System.LinkTypes.Dependency-Reverse means "Predecessor"
	err := p.client.CreateWorkItemLink(ctx, successorID, predecessorID, "System.LinkTypes.Dependency-Reverse")
	if err != nil {
		return fmt.Errorf("create work item link: %w", err)
	}

	return nil
}

// GetDependencies returns the work item IDs that the given work item depends on.
func (p *Provider) GetDependencies(ctx context.Context, workUnitID string) ([]string, error) {
	if p.client == nil {
		return nil, errors.New("client not initialized")
	}

	links, err := p.client.GetWorkItemLinks(ctx, workUnitID)
	if err != nil {
		return nil, fmt.Errorf("get work item links: %w", err)
	}

	// Extract predecessor links
	var deps []string
	for _, link := range links {
		if link.Type == "System.LinkTypes.Dependency-Reverse" {
			deps = append(deps, link.TargetID)
		}
	}

	return deps, nil
}

// WorkItemLink represents a link between work items.
type WorkItemLink struct {
	Type     string
	TargetID string
}

// CreateWorkItemLink creates a link between work items (stub for now).
func (c *Client) CreateWorkItemLink(ctx context.Context, sourceID, targetID, linkType string) error {
	// Future: Implement using Azure DevOps REST API
	// PATCH https://dev.azure.com/{org}/{project}/_apis/wit/workitems/{id}
	// with JSON Patch operation to add relation
	return nil
}

// GetWorkItemLinks returns links for a work item (stub for now).
func (c *Client) GetWorkItemLinks(ctx context.Context, workItemID string) ([]WorkItemLink, error) {
	// Future: Implement using Azure DevOps REST API
	// GET https://dev.azure.com/{org}/{project}/_apis/wit/workitems/{id}?$expand=relations
	return nil, nil
}

// parseDependenciesFromDescription extracts work item IDs from description.
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
