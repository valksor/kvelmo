package provider

import (
	"time"
)

// HealthStatus represents the health status of a provider.
type HealthStatus string

const (
	HealthStatusConnected     HealthStatus = "connected"
	HealthStatusNotConfigured HealthStatus = "not_configured"
	HealthStatusError         HealthStatus = "error"
)

// HealthInfo contains health check information for a provider.
type HealthInfo struct {
	Status    HealthStatus   `json:"status"`
	Message   string         `json:"message,omitempty"`
	RateLimit *RateLimitInfo `json:"rate_limit,omitempty"`
	LastSync  time.Time      `json:"last_sync,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// RateLimitInfo contains rate limit information.
type RateLimitInfo struct {
	Used    int       `json:"used"`
	Limit   int       `json:"limit"`
	ResetAt time.Time `json:"reset_at"`
	ResetIn string    `json:"reset_in"` // Human-readable duration
}

// HealthChecker is the interface for provider health checks.
type HealthChecker interface {
	// HealthCheck performs a health check on the provider.
	// Returns health information or an error if the check failed.
	HealthCheck() (*HealthInfo, error)
}

// ProviderHealth contains health information for all providers.
type ProviderHealth struct {
	Providers map[string]*HealthInfo `json:"providers"`
	CheckedAt time.Time              `json:"checked_at"`
}

// NewProviderHealth creates a new provider health container.
func NewProviderHealth() *ProviderHealth {
	return &ProviderHealth{
		Providers: make(map[string]*HealthInfo),
		CheckedAt: time.Now(),
	}
}

// Add adds health information for a provider.
func (ph *ProviderHealth) Add(name string, info *HealthInfo) {
	ph.Providers[name] = info
}

// Get returns health information for a specific provider.
func (ph *ProviderHealth) Get(name string) (*HealthInfo, bool) {
	info, ok := ph.Providers[name]

	return info, ok
}
