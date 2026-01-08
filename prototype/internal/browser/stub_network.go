//go:build no_browser
// +build no_browser

package browser

import (
	"context"
	"sync"
)

// NetworkMonitor stub when browser is disabled.
type NetworkMonitor struct {
	requests map[string]*NetworkRequest
	mu       sync.RWMutex
}

// NewNetworkMonitor returns a stub network monitor.
func NewNetworkMonitor() *NetworkMonitor {
	return &NetworkMonitor{
		requests: make(map[string]*NetworkRequest),
	}
}

// Start returns an error - browser is disabled.
func (m *NetworkMonitor) Start(ctx context.Context, page interface{}) error {
	return ErrDisabled
}

// Stop is a no-op.
func (m *NetworkMonitor) Stop() error {
	return nil
}

// GetRequests returns empty list - no network activity when disabled.
func (m *NetworkMonitor) GetRequests() []NetworkRequest {
	return []NetworkRequest{}
}

// GetRequestsByType returns empty list.
func (m *NetworkMonitor) GetRequestsByType(resourceType string) []NetworkRequest {
	return []NetworkRequest{}
}

// GetRequestsByURLPattern returns empty list.
func (m *NetworkMonitor) GetRequestsByURLPattern(pattern string) []NetworkRequest {
	return []NetworkRequest{}
}

// GetFailedRequests returns empty list.
func (m *NetworkMonitor) GetFailedRequests() []NetworkRequest {
	return []NetworkRequest{}
}

// GetRequestsForURL returns empty list.
func (m *NetworkMonitor) GetRequestsForURL(url string) []NetworkRequest {
	return []NetworkRequest{}
}
