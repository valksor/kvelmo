package browser

import (
	"context"
	"fmt"
	"time"
)

// getOrCreateConsoleMonitor returns an existing console monitor or creates a new one.
// Caller must NOT hold c.mu.
func (c *controller) getOrCreateConsoleMonitor(ctx context.Context, tabID string) (*ConsoleMonitor, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if monitor already exists
	if mon, exists := c.consoleMon[tabID]; exists {
		return mon, nil
	}

	// Get page while holding lock
	page, err := c.getPage(tabID)
	if err != nil {
		return nil, err
	}

	// Create and start new monitor while holding lock
	// This prevents race where multiple goroutines create monitors
	newMon := NewConsoleMonitorAll()
	if err := newMon.Start(ctx, page); err != nil {
		return nil, fmt.Errorf("start console monitor: %w", err)
	}

	c.consoleMon[tabID] = newMon

	return newMon, nil
}

// GetConsoleLogs captures console logs using the console monitor.
func (c *controller) GetConsoleLogs(ctx context.Context, tabID string, duration time.Duration) ([]ConsoleMessage, error) {
	mon, err := c.getOrCreateConsoleMonitor(ctx, tabID)
	if err != nil {
		return nil, err
	}

	// Wait for duration to collect logs
	if duration > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(duration):
		}
	}

	return mon.GetMessages(), nil
}

// getOrCreateNetworkMonitor returns an existing network monitor or creates a new one.
// Caller must NOT hold c.mu.
func (c *controller) getOrCreateNetworkMonitor(ctx context.Context, tabID string) (*NetworkMonitor, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if monitor already exists
	if mon, exists := c.networkMon[tabID]; exists {
		return mon, nil
	}

	// Get page while holding lock
	page, err := c.getPage(tabID)
	if err != nil {
		return nil, err
	}

	// Create and start new monitor while holding lock
	// This prevents race where multiple goroutines create monitors
	newMon := NewNetworkMonitorWithOptions(c.networkOpts)
	if err := newMon.Start(ctx, page); err != nil {
		return nil, fmt.Errorf("start network monitor: %w", err)
	}

	c.networkMon[tabID] = newMon

	return newMon, nil
}

// GetNetworkRequests captures network requests using the network monitor.
func (c *controller) GetNetworkRequests(ctx context.Context, tabID string, duration time.Duration) ([]NetworkRequest, error) {
	mon, err := c.getOrCreateNetworkMonitor(ctx, tabID)
	if err != nil {
		return nil, err
	}

	// Wait for duration to collect requests
	if duration > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(duration):
		}
	}

	return mon.GetRequests(), nil
}

// SetNetworkMonitorOptions configures options for new network monitors.
// Existing monitors are not affected — call before GetNetworkRequests.
func (c *controller) SetNetworkMonitorOptions(opts NetworkMonitorOptions) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.networkOpts = opts
}

// getOrCreateWebSocketMonitor returns an existing WebSocket monitor or creates a new one.
func (c *controller) getOrCreateWebSocketMonitor(ctx context.Context, tabID string) (*WebSocketMonitor, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if mon, exists := c.wsMon[tabID]; exists {
		return mon, nil
	}

	page, err := c.getPage(tabID)
	if err != nil {
		return nil, err
	}

	newMon := NewWebSocketMonitor()
	if err := newMon.Start(ctx, page); err != nil {
		return nil, fmt.Errorf("start websocket monitor: %w", err)
	}

	c.wsMon[tabID] = newMon

	return newMon, nil
}

// GetWebSocketFrames captures WebSocket frames using the WebSocket monitor.
func (c *controller) GetWebSocketFrames(ctx context.Context, tabID string, duration time.Duration) ([]WebSocketFrame, error) {
	mon, err := c.getOrCreateWebSocketMonitor(ctx, tabID)
	if err != nil {
		return nil, err
	}

	if duration > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(duration):
		}
	}

	return mon.GetFrames(), nil
}

// GetCoverage runs coverage tracking for the specified duration and returns results.
func (c *controller) GetCoverage(ctx context.Context, tabID string, duration time.Duration, trackJS, trackCSS bool) (*CoverageSummary, []JSCoverageEntry, []CSSCoverageEntry, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()
	if err != nil {
		return nil, nil, nil, errNotFound("tab " + tabID)
	}

	mon := NewCoverageMonitor(trackJS, trackCSS)
	if err := mon.Start(ctx, page); err != nil {
		return nil, nil, nil, fmt.Errorf("start coverage: %w", err)
	}

	// Wait for duration to let page execute
	if duration > 0 {
		select {
		case <-ctx.Done():
			return nil, nil, nil, ctx.Err()
		case <-time.After(duration):
		}
	}

	return mon.Collect(ctx)
}

// monitorPageLifecycle runs a background goroutine that periodically cleans up stale monitors.
func (c *controller) monitorPageLifecycle(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.cleanupStaleMonitors()
		}
	}
}

// cleanupStaleMonitors removes monitors for tabs that no longer exist.
func (c *controller) cleanupStaleMonitors() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.browser == nil {
		return
	}

	pages, err := c.browser.Pages()
	if err != nil {
		return
	}

	// Build set of active tab IDs
	activeTabs := make(map[string]bool)
	for _, page := range pages {
		info, err := page.Info()
		if err == nil && info != nil {
			activeTabs[string(info.TargetID)] = true
		}
	}

	// Remove monitors for closed tabs
	for tabID := range c.consoleMon {
		if !activeTabs[tabID] {
			if mon := c.consoleMon[tabID]; mon != nil {
				_ = mon.Stop()
			}
			delete(c.consoleMon, tabID)
		}
	}
	for tabID := range c.networkMon {
		if !activeTabs[tabID] {
			if mon := c.networkMon[tabID]; mon != nil {
				_ = mon.Stop()
			}
			delete(c.networkMon, tabID)
		}
	}
	for tabID := range c.wsMon {
		if !activeTabs[tabID] {
			if mon := c.wsMon[tabID]; mon != nil {
				_ = mon.Stop()
			}
			delete(c.wsMon, tabID)
		}
	}
}
