package bitbucket

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// CreateDependency creates a dependency by updating the issue description.
// Bitbucket doesn't have native issue dependencies, so we use description.
func (p *Provider) CreateDependency(ctx context.Context, predecessorID, successorID string) error {
	if p.client == nil {
		return errors.New("client not initialized")
	}

	// Parse the successor reference
	ref, err := ParseReference(successorID)
	if err != nil {
		return fmt.Errorf("parse reference: %w", err)
	}

	// Set workspace/repo
	workspace := ref.Workspace
	repoSlug := ref.RepoSlug
	if workspace == "" {
		workspace = p.config.Workspace
	}
	if repoSlug == "" {
		repoSlug = p.config.RepoSlug
	}

	if workspace == "" || repoSlug == "" {
		return ErrRepoNotConfigured
	}

	p.client.SetWorkspaceRepo(workspace, repoSlug)

	// Get the successor issue
	issue, err := p.client.GetIssue(ctx, ref.IssueID)
	if err != nil {
		return fmt.Errorf("get issue: %w", err)
	}

	// Build the dependency reference
	depRef := predecessorID

	// Check if the description already contains a dependencies section
	description := ""
	if issue.Content != nil {
		description = issue.Content.Raw
	}
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
	err = p.client.UpdateIssueContent(ctx, ref.IssueID, description)
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

	// Parse the reference
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse reference: %w", err)
	}

	// Set workspace/repo
	workspace := ref.Workspace
	repoSlug := ref.RepoSlug
	if workspace == "" {
		workspace = p.config.Workspace
	}
	if repoSlug == "" {
		repoSlug = p.config.RepoSlug
	}

	if workspace == "" || repoSlug == "" {
		return nil, ErrRepoNotConfigured
	}

	p.client.SetWorkspaceRepo(workspace, repoSlug)

	issue, err := p.client.GetIssue(ctx, ref.IssueID)
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}

	description := ""
	if issue.Content != nil {
		description = issue.Content.Raw
	}

	return parseDependenciesFromDescription(description), nil
}

// UpdateIssueContent updates an issue's content/description.
func (c *Client) UpdateIssueContent(ctx context.Context, issueID int, content string) error {
	path := fmt.Sprintf("/repositories/%s/%s/issues/%d", c.workspace, c.repoSlug, issueID)

	reqBody := map[string]any{
		"content": map[string]string{
			"raw": content,
		},
	}

	_, err := c.doRequest(ctx, "PUT", path, reqBody)

	return err
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

	// Split by comma or space and extract IDs
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
