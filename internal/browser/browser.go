package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// validateURL checks that a URL is safe to open.
// Only http and https schemes are allowed to prevent security issues
// like javascript: or file: URLs being passed by agents.
func validateURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("empty URL")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", u.Scheme)
	}

	return nil
}

// controller implements the Controller interface using Rod.
type controller struct {
	browser       *rod.Browser
	sessionMgr    *SessionManager
	config        Config
	mu            sync.RWMutex
	tabs          map[string]*rod.Page
	networkMon    map[string]*NetworkMonitor
	consoleMon    map[string]*ConsoleMonitor
	cleanupCancel context.CancelFunc // Function to stop cleanup goroutine
}

// NewController creates a new browser controller.
func NewController(config Config) Controller {
	return &controller{
		config:     config,
		tabs:       make(map[string]*rod.Page),
		networkMon: make(map[string]*NetworkMonitor),
		consoleMon: make(map[string]*ConsoleMonitor),
	}
}

// Connect establishes a connection to the browser.
func (c *controller) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.browser != nil {
		return nil // Already connected
	}

	// If port is explicitly set (not 0), connect to existing Chrome
	if c.config.Port != 0 {
		return c.connectToExisting(ctx)
	}

	// Otherwise, use session manager to launch isolated browser
	c.sessionMgr = NewSessionManager(".", c.config)
	session, err := c.sessionMgr.ConnectOrCreate(ctx)
	if err != nil {
		return errConnect(err)
	}

	// Fetch the WebSocket debugger URL from Chrome's /json/version endpoint
	// Rod's ControlURL expects a WebSocket URL, not an HTTP URL
	wsURL, err := c.getWebSocketURL(ctx, session.Host, session.Port)
	if err != nil {
		return errConnect(fmt.Errorf("get websocket url: %w", err))
	}

	c.browser = rod.New().ControlURL(wsURL)

	// Try to connect with retries (Chrome may still be initializing)
	var lastErr error
	for i := range 5 {
		err := c.browser.Connect()
		if err == nil {
			slog.Info("connected to browser", "url", wsURL)

			break
		}
		lastErr = err

		// Check cancellation BEFORE sleeping
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Wait before retry (Chrome needs time to initialize CDP)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(i+1) * 200 * time.Millisecond):
		}
	}

	if lastErr != nil {
		return errConnect(fmt.Errorf("connect to browser after retries: %w", lastErr))
	}

	// Start background goroutine to clean up stale monitors (only once after successful connection)
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	c.cleanupCancel = cleanupCancel
	go c.monitorPageLifecycle(cleanupCtx) //nolint:contextcheck

	return nil
}

// connectToExisting connects to an existing Chrome instance.
func (c *controller) connectToExisting(ctx context.Context) error {
	// Fetch the WebSocket debugger URL from Chrome's /json/version endpoint
	wsURL, err := c.getWebSocketURL(ctx, c.config.Host, c.config.Port)
	if err != nil {
		return errConnect(fmt.Errorf("get websocket url: %w", err))
	}

	c.browser = rod.New().ControlURL(wsURL)

	if err := c.browser.Connect(); err != nil {
		return errConnect(fmt.Errorf("connect to existing browser at %s: %w", wsURL, err))
	}

	slog.Info("connected to existing browser", "url", wsURL)

	// Start background goroutine to clean up stale monitors
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	c.cleanupCancel = cleanupCancel
	go c.monitorPageLifecycle(cleanupCtx) //nolint:contextcheck

	return nil
}

// getWebSocketURL fetches the WebSocket debugger URL from Chrome's /json/version endpoint.
func (c *controller) getWebSocketURL(ctx context.Context, host string, port int) (string, error) {
	url := fmt.Sprintf("http://%s:%d/json/version", host, port)

	// Try to fetch with retries
	var lastErr error
	for i := range 10 {
		req, err := newRequestWithContext(ctx, "GET", url)
		if err != nil {
			return "", err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err

			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(i+1) * 200 * time.Millisecond):
			}

			continue
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close() // Close immediately, not defer (prevents leak in retry loop)
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)

			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(i+1) * 200 * time.Millisecond):
			}

			continue
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close() // Close immediately after reading
		if err != nil {
			return "", fmt.Errorf("read response body: %w", err)
		}

		// Parse JSON to get webSocketDebuggerUrl
		var result struct {
			WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return "", fmt.Errorf("parse json: %w", err)
		}

		if result.WebSocketDebuggerURL == "" {
			return "", errors.New("webSocketDebuggerUrl not found in response")
		}

		return result.WebSocketDebuggerURL, nil
	}

	return "", fmt.Errorf("failed to fetch websocket URL after retries: %w", lastErr)
}

// newRequestWithContext creates an HTTP request with context.
func newRequestWithContext(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// Disconnect closes the browser connection.
func (c *controller) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.browser == nil {
		return nil
	}

	// Stop cleanup goroutine
	if c.cleanupCancel != nil {
		c.cleanupCancel()
		c.cleanupCancel = nil
	}

	// Collect all errors for proper reporting
	var errs []error

	// Stop all monitors
	for tabID, mon := range c.consoleMon {
		if err := mon.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop console monitor %s: %w", tabID, err))
		}
	}
	for tabID, mon := range c.networkMon {
		if err := mon.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop network monitor %s: %w", tabID, err))
		}
	}
	c.consoleMon = make(map[string]*ConsoleMonitor)
	c.networkMon = make(map[string]*NetworkMonitor)

	// Close browser connection first (closes all pages implicitly)
	if err := c.browser.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close browser: %w", err))
	}

	// Clear tabs map after closing browser
	c.tabs = make(map[string]*rod.Page)
	c.browser = nil

	// Cleanup session if we launched it
	if c.sessionMgr != nil {
		if err := c.sessionMgr.Cleanup(); err != nil {
			errs = append(errs, fmt.Errorf("cleanup session: %w", err))
		}
	}

	// Only return error if it's not a connection closed error
	// (connection may already be closed by browser or timeout)
	if len(errs) > 0 {
		for _, err := range errs {
			if !isConnectionClosedError(err) {
				return errDisconnect(errors.Join(errs...))
			}
		}
	}

	return nil
}

// isConnectionClosedError checks if an error is a "connection closed" type error.
func isConnectionClosedError(err error) bool {
	if err == nil {
		return false
	}
	// Import strings package at top of file if not already present
	errStr := err.Error()
	// Common error messages for closed connections
	return strings.Contains(errStr, "use of closed network connection") ||
		strings.Contains(errStr, "connection reset by peer") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection refused")
}

// IsConnected returns true if connected to a browser.
func (c *controller) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.browser != nil
}

// GetPort returns the actual port being used.
// For random port allocation (port=0), this returns the actual allocated port.
// For fixed ports, returns the configured port.
func (c *controller) GetPort() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// If we have a session manager with an active session, use that port
	if c.sessionMgr != nil {
		if session := c.sessionMgr.GetSession(); session != nil {
			return session.Port
		}
	}

	// Fall back to configured port
	return c.config.Port
}

// ListTabs returns all open tabs.
func (c *controller) ListTabs(ctx context.Context) ([]Tab, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.browser == nil {
		return nil, errNotFound("not connected")
	}

	pages, err := c.browser.Pages()
	if err != nil {
		return nil, errListTabs(err)
	}

	var result []Tab
	for _, page := range pages {
		info, err := page.Info()
		if err != nil {
			// Page was closed concurrently, skip it
			continue
		}
		result = append(result, Tab{
			ID:    string(info.TargetID),
			Title: info.Title,
			URL:   info.URL,
		})
	}

	return result, nil
}

// OpenTab opens a new tab with the given URL.
func (c *controller) OpenTab(ctx context.Context, url string) (*Tab, error) {
	// Validate URL before opening (security: prevent javascript:, file:, etc.)
	if err := validateURL(url); err != nil {
		return nil, errOpenTab(err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.browser == nil {
		return nil, errNotFound("not connected")
	}

	page, err := c.browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return nil, errOpenTab(err)
	}

	// Wait for page to load
	if err := page.Timeout(c.config.Timeout).WaitLoad(); err != nil {
		return nil, errOpenTab(fmt.Errorf("wait for load: %w", err))
	}

	info, err := page.Info()
	if err != nil {
		return nil, fmt.Errorf("get page info: %w", err)
	}

	tab := &Tab{
		ID:    string(info.TargetID),
		Title: info.Title,
		URL:   info.URL,
	}

	c.tabs[tab.ID] = page

	return tab, nil
}

// CloseTab closes a tab by ID.
func (c *controller) CloseTab(ctx context.Context, tabID string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Extract monitor references and remove from maps while holding lock
	var page *rod.Page
	var getPageErr error
	var consoleMon *ConsoleMonitor
	var networkMon *NetworkMonitor

	c.mu.Lock()
	page, getPageErr = c.getPage(tabID)
	if getPageErr == nil {
		// Extract monitor references
		consoleMon = c.consoleMon[tabID]
		networkMon = c.networkMon[tabID]

		// Remove from maps immediately (prevents other goroutines from accessing)
		delete(c.consoleMon, tabID)
		delete(c.networkMon, tabID)
		delete(c.tabs, tabID)
	}
	c.mu.Unlock()

	if getPageErr != nil {
		return getPageErr
	}

	// Collect all errors for proper reporting
	var errs []error

	// Close page (no lock held)
	if err := page.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close page: %w", err))
	}

	// Stop monitors WITHOUT holding lock (safe - already removed from maps)
	if consoleMon != nil {
		if err := consoleMon.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop console monitor: %w", err))
		}
	}
	if networkMon != nil {
		if err := networkMon.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop network monitor: %w", err))
		}
	}

	if len(errs) > 0 {
		return errCloseTab(errors.Join(errs...))
	}

	return nil
}

// SwitchTab switches to a tab by ID.
func (c *controller) SwitchTab(ctx context.Context, tabID string) (*Tab, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	page, err := c.getPage(tabID)
	if err != nil {
		return nil, err
	}

	// Activate the page by calling Activate on it
	if _, err := page.Activate(); err != nil {
		return nil, errSwitchTab(err)
	}

	info, err := page.Info()
	if err != nil {
		return nil, fmt.Errorf("get page info: %w", err)
	}

	return &Tab{
		ID:    tabID,
		Title: info.Title,
		URL:   info.URL,
	}, nil
}

// Navigate navigates a tab to a URL.
func (c *controller) Navigate(ctx context.Context, tabID, url string) error {
	// Validate URL before navigating (security: prevent javascript:, file:, etc.)
	if err := validateURL(url); err != nil {
		return errNavigate(err)
	}

	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return err
	}

	if err := page.Navigate(url); err != nil {
		return errNavigate(err)
	}

	// Wait for navigation to complete
	if err := page.Timeout(c.config.Timeout).WaitLoad(); err != nil {
		return errNavigate(fmt.Errorf("wait for load: %w", err))
	}

	return nil
}

// Reload reloads a tab.
func (c *controller) Reload(ctx context.Context, tabID string, hard bool) error {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return err
	}

	if err := page.Reload(); err != nil {
		return errNavigate(err)
	}

	return nil
}

// Screenshot captures a screenshot of a tab.
func (c *controller) Screenshot(ctx context.Context, tabID string, opts ScreenshotOptions) ([]byte, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	var data []byte
	quality := opts.Quality
	if opts.FullPage {
		data, err = page.Screenshot(true, &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormat(opts.Format),
			Quality: &quality,
		})
	} else {
		data, err = page.Screenshot(false, &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormat(opts.Format),
			Quality: &quality,
		})
	}

	if err != nil {
		return nil, errScreenshot(err)
	}

	return data, nil
}

// QuerySelector queries a single element.
func (c *controller) QuerySelector(ctx context.Context, tabID, selector string) (*DOMElement, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	elem, err := page.Element(selector)
	if err != nil {
		return nil, errQuerySelector(err)
	}

	return c.elementToDOM(elem, page)
}

// QuerySelectorAll queries all matching elements.
func (c *controller) QuerySelectorAll(ctx context.Context, tabID, selector string) ([]DOMElement, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	elems, err := page.Elements(selector)
	if err != nil {
		return nil, errQuerySelector(err)
	}

	result := make([]DOMElement, 0, len(elems))
	for _, elem := range elems {
		dom, err := c.elementToDOM(elem, page)
		if err != nil {
			continue
		}
		result = append(result, *dom)
	}

	return result, nil
}

// Click clicks an element.
func (c *controller) Click(ctx context.Context, tabID, selector string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return err
	}

	elem, err := page.Context(ctx).Element(selector)
	if err != nil {
		return errClick(err)
	}

	if err := elem.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return errClick(err)
	}

	return nil
}

// Type types text into an element.
func (c *controller) Type(ctx context.Context, tabID, selector, text string, clearField bool) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return err
	}

	elem, err := page.Context(ctx).Element(selector)
	if err != nil {
		return errType(err)
	}

	if clearField {
		if err := elem.SelectAllText(); err != nil {
			return errType(fmt.Errorf("select all: %w", err))
		}
		if err := elem.Input(""); err != nil {
			return errType(fmt.Errorf("clear: %w", err))
		}
	}

	if err := elem.Input(text); err != nil {
		return errType(err)
	}

	return nil
}

// Eval evaluates JavaScript.
func (c *controller) Eval(ctx context.Context, tabID, expression string) (any, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	result, err := page.Eval(expression)
	if err != nil {
		return nil, errEval(err)
	}

	return result.Value, nil
}

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
	newMon := NewNetworkMonitor()
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

// Helper methods

// getPage retrieves a page by ID.
func (c *controller) getPage(tabID string) (*rod.Page, error) {
	if c.browser == nil {
		return nil, errNotFound("not connected")
	}

	// First check our cache
	if page, ok := c.tabs[tabID]; ok {
		return page, nil
	}

	// Try to find the page in the browser
	pages, err := c.browser.Pages()
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		info, err := page.Info()
		if err != nil {
			continue // Skip pages with errors
		}
		if info == nil {
			continue
		}
		if string(info.TargetID) == tabID {
			c.tabs[tabID] = page

			return page, nil
		}
	}

	return nil, errNotFound("tab " + tabID)
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
}

// elementToDOM converts a Rod element to our DOMElement type.
func (c *controller) elementToDOM(elem *rod.Element, _ *rod.Page) (*DOMElement, error) {
	// Get element info
	text, _ := elem.Text()
	visible, _ := elem.Visible()
	html, _ := elem.HTML()

	// Get full node description
	node, err := elem.Describe(1, false)
	if err != nil {
		// Fallback to basic info if Describe fails
		return &DOMElement{
			TagName:     "element",
			TextContent: text,
			OuterHTML:   html,
			Visible:     visible,
			X:           0,
			Y:           0,
		}, err
	}

	// Get attributes
	attributes := make(map[string]string)
	if node.Attributes != nil {
		for i := 0; i < len(node.Attributes)-1; i += 2 {
			attributes[node.Attributes[i]] = node.Attributes[i+1]
		}
	}

	// Get child count
	childCount := 0
	if node.ChildNodeCount != nil {
		childCount = *node.ChildNodeCount
	}

	return &DOMElement{
		NodeID:      int64(node.NodeID),
		BackendID:   int64(node.BackendNodeID),
		TagName:     node.NodeName,
		Attributes:  attributes,
		TextContent: text,
		OuterHTML:   html,
		ChildCount:  childCount,
		Visible:     visible,
		X:           0, // BoxModel requires separate call
		Y:           0,
	}, nil
}
