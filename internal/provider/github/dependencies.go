package github

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v67/github"
)

// CreateDependency creates a dependency by updating the issue body to include a task list reference.
// GitHub doesn't have native task dependencies, so we use a "Depends on: #N" format in the issue body.
func (p *Provider) CreateDependency(ctx context.Context, predecessorID, successorID string) error {
	owner := p.owner
	repo := p.repo

	if owner == "" || repo == "" {
		return ErrRepoNotConfigured
	}

	p.client.SetOwnerRepo(owner, repo)

	// Get the successor issue to update its body
	successorNum, err := strconv.Atoi(successorID)
	if err != nil {
		return fmt.Errorf("invalid successor ID: %w", err)
	}

	issue, _, err := p.client.gh.Issues.Get(ctx, owner, repo, successorNum)
	if err != nil {
		return fmt.Errorf("get issue: %w", err)
	}

	// Build the dependency reference
	depRef := "#" + predecessorID

	// Check if the body already contains a dependencies section
	body := issue.GetBody()
	dependsOnPattern := regexp.MustCompile(`(?m)^(?:\*\*)?Depends on:(?:\*\*)?\s*(.*)$`)

	if match := dependsOnPattern.FindStringSubmatch(body); match != nil {
		// Dependencies section exists - check if this dependency is already listed
		existingDeps := match[1]
		if strings.Contains(existingDeps, depRef) {
			return nil // Already exists
		}
		// Add to existing dependencies
		newDeps := strings.TrimSpace(existingDeps) + ", " + depRef
		body = dependsOnPattern.ReplaceAllString(body, "**Depends on:** "+newDeps)
	} else {
		// Add new dependencies section at the beginning
		if body != "" {
			body = fmt.Sprintf("**Depends on:** %s\n\n%s", depRef, body)
		} else {
			body = "**Depends on:** " + depRef
		}
	}

	// Update the issue
	updateReq := &IssueRequest{
		Body: &body,
	}

	_, _, err = p.client.gh.Issues.Edit(ctx, owner, repo, successorNum, updateReq)
	if err != nil {
		return fmt.Errorf("update issue: %w", err)
	}

	return nil
}

// GetDependencies returns the issue numbers that the given issue depends on.
// It parses the "Depends on:" line in the issue body.
func (p *Provider) GetDependencies(ctx context.Context, workUnitID string) ([]string, error) {
	owner := p.owner
	repo := p.repo

	if owner == "" || repo == "" {
		return nil, ErrRepoNotConfigured
	}

	p.client.SetOwnerRepo(owner, repo)

	issueNum, err := strconv.Atoi(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("invalid work unit ID: %w", err)
	}

	issue, _, err := p.client.gh.Issues.Get(ctx, owner, repo, issueNum)
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}

	return parseDependencies(issue.GetBody()), nil
}

// parseDependencies extracts issue numbers from a "Depends on:" line.
// Supports formats like: "Depends on: #1, #2, #3" or "**Depends on:** #1 #2".
func parseDependencies(body string) []string {
	if body == "" {
		return nil
	}

	dependsOnPattern := regexp.MustCompile(`(?m)^(?:\*\*)?Depends on:(?:\*\*)?\s*(.*)$`)
	match := dependsOnPattern.FindStringSubmatch(body)
	if match == nil {
		return nil
	}

	// Extract issue numbers from the dependencies line
	issuePattern := regexp.MustCompile(`#(\d+)`)
	matches := issuePattern.FindAllStringSubmatch(match[1], -1)

	deps := make([]string, 0, len(matches))
	for _, m := range matches {
		deps = append(deps, m[1])
	}

	return deps
}

// IssueRequest wraps GitHub issue request for editing.
type IssueRequest = github.IssueRequest
