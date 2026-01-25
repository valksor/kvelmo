package jira

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// HealthCheck performs a health check on the Jira provider.
func (p *Provider) HealthCheck() (*provider.HealthInfo, error) {
	info := &provider.HealthInfo{
		LastSync: time.Now(),
	}

	// Check if configured
	if p.client == nil {
		info.Status = provider.HealthStatusNotConfigured
		info.Message = "Set JIRA_TOKEN, JIRA_EMAIL, and JIRA_BASE_URL in .mehrhof/.env or config"

		return info, nil
	}

	// Try to make an authenticated API call
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get current user to verify authentication using the client's doRequest method
	user := struct {
		DisplayName string `json:"displayName"`
		AccountID   string `json:"accountId"`
	}{}

	err := p.client.doRequest(ctx, "GET", "/myself", nil, &user)
	if err != nil {
		info.Status = provider.HealthStatusError
		info.Error = fmt.Sprintf("API error: %v", err)
		info.Message = "Failed to connect to Jira API"

		// Check for specific error types
		if strings.Contains(err.Error(), "401") {
			info.Error = "Authentication failed - check your token and email"
		} else if strings.Contains(err.Error(), "403") {
			info.Error = "Insufficient permissions"
		}

		return info, nil
	}

	// All checks passed
	info.Status = provider.HealthStatusConnected
	info.Message = "Connected as " + user.DisplayName

	// Add project info if configured
	if p.defaultProject != "" {
		info.Message += " • Project: " + p.defaultProject
	}

	return info, nil
}
