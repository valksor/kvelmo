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
	wsMon         map[string]*WebSocketMonitor // TabID -> WebSocket Monitor
	cleanupCancel context.CancelFunc           // Function to stop cleanup goroutine
	cookieStorage *CookieStorage               // Cookie storage manager
	cookieProfile string                       // Current cookie profile
	networkOpts   NetworkMonitorOptions        // Options for new network monitors
}

// NewController creates a new browser controller.
func NewController(config Config) Controller {
	return &controller{
		config:     config,
		tabs:       make(map[string]*rod.Page),
		networkMon: make(map[string]*NetworkMonitor),
		consoleMon: make(map[string]*ConsoleMonitor),
		wsMon:      make(map[string]*WebSocketMonitor),
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

	// Initialize cookie storage if enabled
	if c.config.CookieAutoLoad || c.config.CookieAutoSave {
		// Determine cookie profile
		c.cookieProfile = c.config.CookieProfile
		if c.cookieProfile == "" {
			c.cookieProfile = "default"
		}

		// Initialize cookie storage
		cookieDir := c.config.CookieDir
		c.cookieStorage = NewCookieStorage(cookieDir)

		// Auto-load cookies after successful connection
		if c.config.CookieAutoLoad {
			if err := c.loadCookies(); err != nil {
				slog.Warn("failed to load cookies", "profile", c.cookieProfile, "error", err)
			}
		}
	}

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

	// Auto-save cookies before closing browser
	if c.config.CookieAutoSave && c.cookieStorage != nil {
		if err := c.saveCookies(); err != nil {
			slog.Warn("failed to save cookies", "profile", c.cookieProfile, "error", err)
		}
	}

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
// This includes EOF which occurs when the WebSocket connection is dropped.
func isConnectionClosedError(err error) bool {
	if err == nil {
		return false
	}
	// Check for io.EOF specifically
	if errors.Is(err, io.EOF) {
		return true
	}
	errStr := err.Error()
	// Common error messages for closed connections
	return strings.Contains(errStr, "use of closed network connection") ||
		strings.Contains(errStr, "connection reset by peer") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "EOF")
}

// isTargetNotFoundError checks if an error is a CDP "target not found" error.
// This happens during concurrent tab operations when a target is closed
// between getting the target list and accessing the target.
func isTargetNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()

	return strings.Contains(errStr, "No target with given id found")
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
// Implements retry logic to handle race conditions with newly created pages
// that may not immediately appear in Chrome's target list.
func (c *controller) ListTabs(ctx context.Context) ([]Tab, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.browser == nil {
		return nil, errNotFound("not connected")
	}

	// Try up to 3 times to handle race condition with newly created pages.
	// After OpenTab creates a page, there's a timing window where Chrome's CDP
	// target list hasn't yet registered the new target, causing Pages() to
	// return an incomplete list.
	const maxRetries = 3

	for i := range maxRetries {
		pages, err := c.browser.Pages()
		if err != nil {
			// Handle transient errors that can occur during browser operations:
			// - Target not found: tab was closed between list request and response
			// - Connection closed/EOF: WebSocket connection dropped temporarily
			if isTargetNotFoundError(err) || isConnectionClosedError(err) {
				// On transient error, retry with backoff if retries remaining
				if i < maxRetries-1 {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(time.Duration(i+1) * 100 * time.Millisecond):
						continue
					}
				}
				// Last retry, return empty list instead of hard failure
				return []Tab{}, nil
			}

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

		// If we got results or this is the last retry, return immediately.
		// This avoids unnecessary delays when the page list is populated correctly.
		if len(result) > 0 || i == maxRetries-1 {
			return result, nil
		}

		// Short backoff before retry (100ms, 200ms) to give Chrome time to register the target.
		// Check context cancellation to respect caller's timeout.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(i+1) * 100 * time.Millisecond):
		}
	}

	return []Tab{}, nil
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

	// Wait for page to load using caller's context for timeout/cancellation
	if err := page.Context(ctx).WaitLoad(); err != nil {
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

	if err := page.Context(ctx).Navigate(url); err != nil {
		return errNavigate(err)
	}

	// Wait for navigation to complete using caller's context
	if err := page.Context(ctx).WaitLoad(); err != nil {
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

	if err := page.Context(ctx).Reload(); err != nil {
		return errNavigate(err)
	}

	return nil
}

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
