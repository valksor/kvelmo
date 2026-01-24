package trello

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// CreateDependency creates a dependency by updating the card description.
// Trello doesn't have native blocking dependencies via API, so we use description.
func (p *Provider) CreateDependency(ctx context.Context, predecessorID, successorID string) error {
	if p.client == nil {
		return errors.New("client not initialized")
	}

	// Get the successor card
	card, err := p.client.GetCard(ctx, successorID)
	if err != nil {
		return fmt.Errorf("get card: %w", err)
	}

	// Build the dependency reference
	depRef := predecessorID

	// Check if the description already contains a dependencies section
	description := card.Desc
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

	// Update the card
	err = p.client.UpdateCardDescription(ctx, successorID, description)
	if err != nil {
		return fmt.Errorf("update card: %w", err)
	}

	return nil
}

// GetDependencies returns the card IDs that the given card depends on.
func (p *Provider) GetDependencies(ctx context.Context, workUnitID string) ([]string, error) {
	if p.client == nil {
		return nil, errors.New("client not initialized")
	}

	card, err := p.client.GetCard(ctx, workUnitID)
	if err != nil {
		return nil, fmt.Errorf("get card: %w", err)
	}

	return parseDependenciesFromDescription(card.Desc), nil
}

// UpdateCardDescription updates a card's description.
func (c *Client) UpdateCardDescription(ctx context.Context, cardID, description string) error {
	endpoint := "/cards/" + cardID
	params := map[string]string{
		"desc": description,
	}

	return c.putForm(ctx, endpoint, params, nil)
}

// putForm sends a PUT request with form data.
func (c *Client) putForm(ctx context.Context, endpoint string, params map[string]string, result any) error {
	urlParams := make(map[string][]string)
	for k, v := range params {
		urlParams[k] = []string{v}
	}

	return c.put(ctx, endpoint, urlParams, result)
}

// parseDependenciesFromDescription extracts card IDs from a "Depends on:" line.
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
