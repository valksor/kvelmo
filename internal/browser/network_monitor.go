package browser

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// NetworkMonitor monitors network requests for a browser tab.
type NetworkMonitor struct {
	requests map[string]*NetworkRequest
	mu       sync.RWMutex
	cancel   context.CancelFunc
	done     chan struct{}
	wg       sync.WaitGroup // Track goroutine
}

// NewNetworkMonitor creates a new network monitor.
func NewNetworkMonitor() *NetworkMonitor {
	return &NetworkMonitor{
		requests: make(map[string]*NetworkRequest),
	}
}

// Start begins monitoring network requests for a page.
func (m *NetworkMonitor) Start(ctx context.Context, page *rod.Page) error {
	// Enable network events
	_ = proto.NetworkEnable{}.Call(page)

	// Create cancelable context for this monitor
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.done = make(chan struct{})

	// Start event listener in background
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.listenForEvents(ctx, page)
	}()

	return nil
}

// Stop stops the network monitor and cleans up resources.
// Returns error if timeout occurs waiting for goroutine to exit.
func (m *NetworkMonitor) Stop() error {
	if m.cancel != nil {
		m.cancel()
	}

	// Wait for goroutine to exit with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		// Log warning about goroutine leak
		slog.Warn("network monitor goroutine leak detected",
			"timeout", "5s")
		// Return error but at least we logged it
		return errors.New("network monitor stop timeout after 5s")
	}
}

// listenForEvents listens for network events.
func (m *NetworkMonitor) listenForEvents(ctx context.Context, page *rod.Page) {
	defer close(m.done)

	// Use EachEvent to listen for network events
	page.Context(ctx).EachEvent(
		func(e *proto.NetworkRequestWillBeSent) {
			// Convert headers from gson.JSON to string
			headers := make(map[string]string)
			for k, v := range e.Request.Headers {
				headers[k] = v.String()
			}

			m.AddRequest(&NetworkRequest{
				ID:           string(e.RequestID),
				URL:          e.Request.URL,
				Method:       e.Request.Method,
				Headers:      headers,
				ResourceType: string(e.Type),
				Timestamp:    time.Unix(0, int64(e.Timestamp.Duration())),
				RequestBody:  "", // Request body would need separate handling
			})
		},
		func(e *proto.NetworkResponseReceived) {
			// Update the existing request with response info
			m.mu.Lock()
			if req, exists := m.requests[string(e.RequestID)]; exists {
				req.Status = e.Response.Status
				req.StatusText = e.Response.StatusText
				req.MimeType = e.Response.MIMEType
				// No need to put back - pointer already in map, modifications are in-place
			}
			m.mu.Unlock()
		},
	)()

	// Wait for context cancellation
	<-ctx.Done()
}

// GetRequests returns all captured network requests.
func (m *NetworkMonitor) GetRequests() []NetworkRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]NetworkRequest, 0, len(m.requests))
	for _, req := range m.requests {
		result = append(result, *req)
	}

	return result
}

// AddRequest adds a network request to the monitor.
func (m *NetworkMonitor) AddRequest(req *NetworkRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests[req.ID] = req
}

// GetRequestsByType returns requests filtered by type (XHR, Fetch, etc).
func (m *NetworkMonitor) GetRequestsByType(resourceType string) []NetworkRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []NetworkRequest{}
	for _, req := range m.requests {
		if req.ResourceType == resourceType {
			result = append(result, *req)
		}
	}

	return result
}

// GetRequestsByURLPattern returns requests matching a URL pattern.
func (m *NetworkMonitor) GetRequestsByURLPattern(pattern string) []NetworkRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []NetworkRequest{}
	for _, req := range m.requests {
		// Simple contains check (could be enhanced with regex)
		if contains(req.URL, pattern) {
			result = append(result, *req)
		}
	}

	return result
}

// GetFailedRequests returns requests that failed (4xx, 5xx status codes).
func (m *NetworkMonitor) GetFailedRequests() []NetworkRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []NetworkRequest{}
	for _, req := range m.requests {
		if req.Status >= 400 {
			result = append(result, *req)
		}
	}

	return result
}

// GetRequestsForURL returns all requests for a specific URL.
func (m *NetworkMonitor) GetRequestsForURL(url string) []NetworkRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []NetworkRequest{}
	for _, req := range m.requests {
		if req.URL == url {
			result = append(result, *req)
		}
	}

	return result
}
