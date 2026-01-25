package gitlab

import (
	"context"
	"fmt"
	"strings"
	"time"

	gl "gitlab.com/gitlab-org/api/client-go"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// HealthCheck performs a health check on the GitLab provider.
func (p *Provider) HealthCheck() (*provider.HealthInfo, error) {
	info := &provider.HealthInfo{
		LastSync: time.Now(),
	}

	// Check if configured
	if p.config == nil || p.config.Token == "" {
		info.Status = provider.HealthStatusNotConfigured
		info.Message = "Set GITLAB_TOKEN in .mehrhof/.env or config"

		return info, nil
	}

	// Try to make an authenticated API call
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get current user to verify authentication
	user, _, err := p.client.gl.Users.CurrentUser(gl.WithContext(ctx))
	if err != nil {
		info.Status = provider.HealthStatusError
		info.Error = fmt.Sprintf("API error: %v", err)
		info.Message = "Failed to connect to GitLab API"

		// Check for specific error types
		if strings.Contains(err.Error(), "401") {
			info.Error = "Authentication failed - check your token"
		} else if strings.Contains(err.Error(), "403") {
			info.Error = "Insufficient permissions"
		}

		return info, nil
	}

	// All checks passed
	info.Status = provider.HealthStatusConnected
	info.Message = "Connected as " + user.Username

	// Add project path info if configured
	if p.config.ProjectPath != "" {
		info.Message += " • " + p.config.ProjectPath
	}

	// GitLab doesn't provide rate limit info in the same way as GitHub
	// We could add it if needed from response headers

	return info, nil
}
