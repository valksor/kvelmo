package gitlab

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	gl "gitlab.com/gitlab-org/api/client-go"
)

// CreateDependency creates a dependency by updating the issue body to include a reference.
// GitLab doesn't have native task dependencies, so we use a "Depends on: #N" format in the issue description.
func (p *Provider) CreateDependency(ctx context.Context, predecessorID, successorID string) error {
	projectPath := p.projectPath
	if projectPath == "" {
		return ErrProjectNotConfigured
	}

	// Get the successor issue to update its description
	successorNum, err := strconv.Atoi(successorID)
	if err != nil {
		return fmt.Errorf("invalid successor ID: %w", err)
	}

	issue, _, err := p.client.gl.Issues.GetIssue(projectPath, int64(successorNum))
	if err != nil {
		return fmt.Errorf("get issue: %w", err)
	}

	// Build the dependency reference
	depRef := "#" + predecessorID

	// Check if the description already contains a dependencies section
	description := issue.Description
	dependsOnPattern := regexp.MustCompile(`(?m)^(?:\*\*)?Depends on:(?:\*\*)?\s*(.*)$`)

	if match := dependsOnPattern.FindStringSubmatch(description); match != nil {
		// Dependencies section exists - check if this dependency is already listed
		existingDeps := match[1]
		if strings.Contains(existingDeps, depRef) {
			return nil // Already exists
		}
		// Add to existing dependencies
		newDeps := strings.TrimSpace(existingDeps) + ", " + depRef
		description = dependsOnPattern.ReplaceAllString(description, "**Depends on:** "+newDeps)
	} else {
		// Add new dependencies section at the beginning
		if description != "" {
			description = fmt.Sprintf("**Depends on:** %s\n\n%s", depRef, description)
		} else {
			description = "**Depends on:** " + depRef
		}
	}

	// Update the issue
	updateOpts := &gl.UpdateIssueOptions{
		Description: gl.Ptr(description),
	}

	_, _, err = p.client.gl.Issues.UpdateIssue(projectPath, int64(successorNum), updateOpts)
	if err != nil {
		return fmt.Errorf("update issue: %w", err)
	}

	return nil
}

// GetDependencies returns the issue numbers that the given issue depends on.
// It parses the "Depends on:" line in the issue description.
func (p *Provider) GetDependencies(ctx context.Context, workUnitID string) ([]string, error) {
	projectPath := p.projectPath
	if projectPath == "" {
		return nil, ErrProjectNotConfigured
	}

	issueNum, err := strconv.Atoi(workUnitID)
	if err != nil {
		return nil, fmt.Errorf("invalid work unit ID: %w", err)
	}

	issue, _, err := p.client.gl.Issues.GetIssue(projectPath, int64(issueNum))
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}

	return parseDependencies(issue.Description), nil
}

// parseDependencies extracts issue numbers from a "Depends on:" line.
// Supports formats like: "Depends on: #1, #2, #3" or "**Depends on:** #1 #2".
func parseDependencies(description string) []string {
	if description == "" {
		return nil
	}

	dependsOnPattern := regexp.MustCompile(`(?m)^(?:\*\*)?Depends on:(?:\*\*)?\s*(.*)$`)
	match := dependsOnPattern.FindStringSubmatch(description)
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
