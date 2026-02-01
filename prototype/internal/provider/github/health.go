package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// HealthCheck performs a health check on the GitHub provider.
func (p *Provider) HealthCheck() (*provider.HealthInfo, error) {
	info := &provider.HealthInfo{
		LastSync: time.Now(),
	}

	// Check if configured
	if p.config == nil || p.config.Token == "" {
		info.Status = provider.HealthStatusNotConfigured
		info.Message = "Set GITHUB_TOKEN in .mehrhof/.env or config"

		return info, nil
	}

	// Try to make an authenticated API call
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get rate limit info using the RateLimit service
	rateLimits, _, err := p.client.gh.RateLimit.Get(ctx)
	if err != nil {
		info.Status = provider.HealthStatusError
		info.Error = fmt.Sprintf("API error: %v", err)
		info.Message = "Failed to connect to GitHub API"

		// Check for specific error types
		if strings.Contains(err.Error(), "401") {
			info.Error = "Authentication failed - check your token"
		} else if strings.Contains(err.Error(), "403") && strings.Contains(err.Error(), "rate limit") {
			info.Error = "Rate limit exceeded - wait before retrying"
		}

		return info, nil
	}

	// Check core rate limit
	core := rateLimits.Core
	if core != nil {
		info.RateLimit = &provider.RateLimitInfo{
			Used:    core.Remaining,
			Limit:   core.Limit,
			ResetAt: core.Reset.Time,
		}

		// Calculate human-readable reset time
		resetIn := time.Until(core.Reset.Time)
		if resetIn > 0 {
			info.RateLimit.ResetIn = formatDuration(resetIn)
		} else {
			info.RateLimit.ResetIn = "now"
		}
	}

	// Verify authentication by checking if we can access user info
	_, _, err = p.client.gh.Users.Get(ctx, "")
	if err != nil {
		info.Status = provider.HealthStatusError
		info.Error = fmt.Sprintf("Authentication failed: %v", err)
		info.Message = "Token may be invalid or expired"

		return info, nil
	}

	// All checks passed
	info.Status = provider.HealthStatusConnected

	// Build status message
	if info.RateLimit != nil {
		percentage := float64(info.RateLimit.Used) / float64(info.RateLimit.Limit) * 100
		info.Message = fmt.Sprintf("Rate: %d/%d (%.1f%%)",
			info.RateLimit.Used,
			info.RateLimit.Limit,
			percentage)
	} else {
		info.Message = "Connected"
	}

	// Add owner/repo info if configured
	if p.owner != "" && p.repo != "" {
		info.Message += fmt.Sprintf(" • %s/%s", p.owner, p.repo)
	}

	return info, nil
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		if mins > 0 {
			return fmt.Sprintf("%dh %dm", hours, mins)
		}

		return fmt.Sprintf("%dh", hours)
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}

	return fmt.Sprintf("%dd", days)
}
