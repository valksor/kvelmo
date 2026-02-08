// Package helper_test provides shared testing utilities for go-mehrhof tests.
package helper_test

import (
	"context"
	"sync"
	"time"

	"github.com/valksor/go-mehrhof/internal/browser"
)

// ──────────────────────────────────────────────────────────────────────────────
// MockBrowserController implements browser.Controller for testing
// ──────────────────────────────────────────────────────────────────────────────

// MockBrowserController is a configurable mock implementation of browser.Controller.
type MockBrowserController struct {
	mu sync.Mutex

	// Connection state
	ConnectErr     error
	DisconnectErr  error
	IsConnectedVal bool
	PortVal        int

	// Tab management
	ListTabsResult  []browser.Tab
	ListTabsErr     error
	OpenTabResult   *browser.Tab
	OpenTabErr      error
	CloseTabErr     error
	SwitchTabResult *browser.Tab
	SwitchTabErr    error
	NavigateErr     error
	ReloadErr       error

	// Page interaction
	ScreenshotData         []byte
	ScreenshotErr          error
	QuerySelectorResult    *browser.DOMElement
	QuerySelectorErr       error
	QuerySelectorAllResult []browser.DOMElement
	QuerySelectorAllErr    error
	ClickErr               error
	TypeErr                error
	EvalResult             any
	EvalErr                error

	// Monitoring
	ConsoleLogs        []browser.ConsoleMessage
	ConsoleLogsErr     error
	NetworkRequests    []browser.NetworkRequest
	NetworkRequestsErr error
	NetworkMonitorOpts browser.NetworkMonitorOptions
	WebSocketFrames    []browser.WebSocketFrame
	WebSocketFramesErr error

	// Source inspection
	PageSource       string
	PageSourceErr    error
	ScriptSources    []browser.ScriptSource
	ScriptSourcesErr error

	// CSS inspection
	ComputedStyles    []browser.ComputedStyle
	ComputedStylesErr error
	MatchedStyles     *browser.MatchedStyles
	MatchedStylesErr  error

	// Coverage
	CoverageSummary    *browser.CoverageSummary
	JSCoverageEntries  []browser.JSCoverageEntry
	CSSCoverageEntries []browser.CSSCoverageEntry
	CoverageErr        error

	// Authentication
	AuthRequirement *browser.AuthRequirement
	DetectAuthErr   error
	WaitForLoginErr error

	// Cookies
	Cookies       []browser.Cookie
	CookiesErr    error
	SetCookiesErr error

	// Call tracking
	CallLog []string
}

// NewMockBrowserController creates a new MockBrowserController with sensible defaults.
func NewMockBrowserController() *MockBrowserController {
	return &MockBrowserController{
		IsConnectedVal: true,
		PortVal:        9222,
		ListTabsResult: []browser.Tab{
			{ID: "tab-1", Title: "Test Page", URL: "https://example.com"},
		},
	}
}

// trackCall records a method call for verification.
func (m *MockBrowserController) trackCall(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallLog = append(m.CallLog, method)
}

// GetCalls returns all recorded method calls.
func (m *MockBrowserController) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.CallLog))
	copy(result, m.CallLog)

	return result
}

// ClearCalls clears recorded method calls.
func (m *MockBrowserController) ClearCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallLog = nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Connection management
// ──────────────────────────────────────────────────────────────────────────────

// Connect simulates connecting to the browser.
func (m *MockBrowserController) Connect(_ context.Context) error {
	m.trackCall("Connect")
	if m.ConnectErr != nil {
		return m.ConnectErr
	}
	m.mu.Lock()
	m.IsConnectedVal = true
	m.mu.Unlock()

	return nil
}

// Disconnect simulates disconnecting from the browser.
func (m *MockBrowserController) Disconnect() error {
	m.trackCall("Disconnect")
	if m.DisconnectErr != nil {
		return m.DisconnectErr
	}
	m.mu.Lock()
	m.IsConnectedVal = false
	m.mu.Unlock()

	return nil
}

// IsConnected returns the mock connection state.
func (m *MockBrowserController) IsConnected() bool {
	m.trackCall("IsConnected")
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.IsConnectedVal
}

// GetPort returns the configured port.
func (m *MockBrowserController) GetPort() int {
	m.trackCall("GetPort")

	return m.PortVal
}

// ──────────────────────────────────────────────────────────────────────────────
// Tab management
// ──────────────────────────────────────────────────────────────────────────────

// ListTabs returns the configured tabs.
func (m *MockBrowserController) ListTabs(_ context.Context) ([]browser.Tab, error) {
	m.trackCall("ListTabs")
	if m.ListTabsErr != nil {
		return nil, m.ListTabsErr
	}

	return m.ListTabsResult, nil
}

// OpenTab simulates opening a new tab.
func (m *MockBrowserController) OpenTab(_ context.Context, url string) (*browser.Tab, error) {
	m.trackCall("OpenTab")
	if m.OpenTabErr != nil {
		return nil, m.OpenTabErr
	}
	if m.OpenTabResult != nil {
		return m.OpenTabResult, nil
	}

	return &browser.Tab{ID: "new-tab", Title: "New Tab", URL: url}, nil
}

// CloseTab simulates closing a tab.
func (m *MockBrowserController) CloseTab(_ context.Context, _ string) error {
	m.trackCall("CloseTab")

	return m.CloseTabErr
}

// SwitchTab simulates switching to a tab.
func (m *MockBrowserController) SwitchTab(_ context.Context, tabID string) (*browser.Tab, error) {
	m.trackCall("SwitchTab")
	if m.SwitchTabErr != nil {
		return nil, m.SwitchTabErr
	}
	if m.SwitchTabResult != nil {
		return m.SwitchTabResult, nil
	}

	return &browser.Tab{ID: tabID, Title: "Switched Tab", URL: "https://example.com"}, nil
}

// Navigate simulates navigating to a URL.
func (m *MockBrowserController) Navigate(_ context.Context, _, _ string) error {
	m.trackCall("Navigate")

	return m.NavigateErr
}

// Reload simulates reloading the page.
func (m *MockBrowserController) Reload(_ context.Context, _ string, _ bool) error {
	m.trackCall("Reload")

	return m.ReloadErr
}

// ──────────────────────────────────────────────────────────────────────────────
// Page interaction
// ──────────────────────────────────────────────────────────────────────────────

// Screenshot returns mock screenshot data.
func (m *MockBrowserController) Screenshot(_ context.Context, _ string, _ browser.ScreenshotOptions) ([]byte, error) {
	m.trackCall("Screenshot")
	if m.ScreenshotErr != nil {
		return nil, m.ScreenshotErr
	}
	if m.ScreenshotData != nil {
		return m.ScreenshotData, nil
	}
	// Return a minimal valid PNG header
	return []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, nil
}

// QuerySelector returns the configured element.
func (m *MockBrowserController) QuerySelector(_ context.Context, _, _ string) (*browser.DOMElement, error) {
	m.trackCall("QuerySelector")
	if m.QuerySelectorErr != nil {
		return nil, m.QuerySelectorErr
	}

	return m.QuerySelectorResult, nil
}

// QuerySelectorAll returns the configured elements.
func (m *MockBrowserController) QuerySelectorAll(_ context.Context, _, _ string) ([]browser.DOMElement, error) {
	m.trackCall("QuerySelectorAll")
	if m.QuerySelectorAllErr != nil {
		return nil, m.QuerySelectorAllErr
	}

	return m.QuerySelectorAllResult, nil
}

// Click simulates clicking an element.
func (m *MockBrowserController) Click(_ context.Context, _, _ string) error {
	m.trackCall("Click")

	return m.ClickErr
}

// Type simulates typing into an element.
func (m *MockBrowserController) Type(_ context.Context, _, _, _ string, _ bool) error {
	m.trackCall("Type")

	return m.TypeErr
}

// Eval evaluates JavaScript and returns the configured result.
func (m *MockBrowserController) Eval(_ context.Context, _, _ string) (any, error) {
	m.trackCall("Eval")
	if m.EvalErr != nil {
		return nil, m.EvalErr
	}

	return m.EvalResult, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Monitoring
// ──────────────────────────────────────────────────────────────────────────────

// GetConsoleLogs returns configured console messages.
func (m *MockBrowserController) GetConsoleLogs(_ context.Context, _ string, _ time.Duration) ([]browser.ConsoleMessage, error) {
	m.trackCall("GetConsoleLogs")
	if m.ConsoleLogsErr != nil {
		return nil, m.ConsoleLogsErr
	}

	return m.ConsoleLogs, nil
}

// GetNetworkRequests returns configured network requests.
func (m *MockBrowserController) GetNetworkRequests(_ context.Context, _ string, _ time.Duration) ([]browser.NetworkRequest, error) {
	m.trackCall("GetNetworkRequests")
	if m.NetworkRequestsErr != nil {
		return nil, m.NetworkRequestsErr
	}

	return m.NetworkRequests, nil
}

// SetNetworkMonitorOptions stores the network monitoring options.
func (m *MockBrowserController) SetNetworkMonitorOptions(opts browser.NetworkMonitorOptions) {
	m.trackCall("SetNetworkMonitorOptions")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.NetworkMonitorOpts = opts
}

// GetWebSocketFrames returns configured WebSocket frames.
func (m *MockBrowserController) GetWebSocketFrames(_ context.Context, _ string, _ time.Duration) ([]browser.WebSocketFrame, error) {
	m.trackCall("GetWebSocketFrames")
	if m.WebSocketFramesErr != nil {
		return nil, m.WebSocketFramesErr
	}

	return m.WebSocketFrames, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Source inspection
// ──────────────────────────────────────────────────────────────────────────────

// GetPageSource returns the configured page source.
func (m *MockBrowserController) GetPageSource(_ context.Context, _ string) (string, error) {
	m.trackCall("GetPageSource")
	if m.PageSourceErr != nil {
		return "", m.PageSourceErr
	}
	if m.PageSource == "" {
		return "<html><body>Test Page</body></html>", nil
	}

	return m.PageSource, nil
}

// GetScriptSources returns configured script sources.
func (m *MockBrowserController) GetScriptSources(_ context.Context, _ string) ([]browser.ScriptSource, error) {
	m.trackCall("GetScriptSources")
	if m.ScriptSourcesErr != nil {
		return nil, m.ScriptSourcesErr
	}

	return m.ScriptSources, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// CSS inspection
// ──────────────────────────────────────────────────────────────────────────────

// GetComputedStyles returns configured computed styles.
func (m *MockBrowserController) GetComputedStyles(_ context.Context, _, _ string) ([]browser.ComputedStyle, error) {
	m.trackCall("GetComputedStyles")
	if m.ComputedStylesErr != nil {
		return nil, m.ComputedStylesErr
	}

	return m.ComputedStyles, nil
}

// GetMatchedStyles returns configured matched styles.
func (m *MockBrowserController) GetMatchedStyles(_ context.Context, _, _ string) (*browser.MatchedStyles, error) {
	m.trackCall("GetMatchedStyles")
	if m.MatchedStylesErr != nil {
		return nil, m.MatchedStylesErr
	}

	return m.MatchedStyles, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Coverage
// ──────────────────────────────────────────────────────────────────────────────

// GetCoverage returns configured coverage data.
func (m *MockBrowserController) GetCoverage(_ context.Context, _ string, _ time.Duration, _, _ bool) (*browser.CoverageSummary, []browser.JSCoverageEntry, []browser.CSSCoverageEntry, error) {
	m.trackCall("GetCoverage")
	if m.CoverageErr != nil {
		return nil, nil, nil, m.CoverageErr
	}

	return m.CoverageSummary, m.JSCoverageEntries, m.CSSCoverageEntries, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Authentication
// ──────────────────────────────────────────────────────────────────────────────

// DetectAuth returns configured auth requirement.
func (m *MockBrowserController) DetectAuth(_ context.Context, _ string) (*browser.AuthRequirement, error) {
	m.trackCall("DetectAuth")
	if m.DetectAuthErr != nil {
		return nil, m.DetectAuthErr
	}

	return m.AuthRequirement, nil
}

// WaitForLogin simulates waiting for login.
func (m *MockBrowserController) WaitForLogin(_ context.Context, _ string, _ *browser.AuthRequirement) error {
	m.trackCall("WaitForLogin")

	return m.WaitForLoginErr
}

// ──────────────────────────────────────────────────────────────────────────────
// Cookie management
// ──────────────────────────────────────────────────────────────────────────────

// GetCookies returns configured cookies.
func (m *MockBrowserController) GetCookies(_ context.Context) ([]browser.Cookie, error) {
	m.trackCall("GetCookies")
	if m.CookiesErr != nil {
		return nil, m.CookiesErr
	}

	return m.Cookies, nil
}

// SetCookies simulates setting cookies.
func (m *MockBrowserController) SetCookies(_ context.Context, cookies []browser.Cookie) error {
	m.trackCall("SetCookies")
	if m.SetCookiesErr != nil {
		return m.SetCookiesErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Cookies = cookies

	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Builder methods for test configuration
// ──────────────────────────────────────────────────────────────────────────────

// WithConnectError configures the connect error.
func (m *MockBrowserController) WithConnectError(err error) *MockBrowserController {
	m.ConnectErr = err

	return m
}

// WithTabs configures the tab list.
func (m *MockBrowserController) WithTabs(tabs []browser.Tab) *MockBrowserController {
	m.ListTabsResult = tabs

	return m
}

// WithConsoleLogs configures console messages.
func (m *MockBrowserController) WithConsoleLogs(logs []browser.ConsoleMessage) *MockBrowserController {
	m.ConsoleLogs = logs

	return m
}

// WithNetworkRequests configures network requests.
func (m *MockBrowserController) WithNetworkRequests(requests []browser.NetworkRequest) *MockBrowserController {
	m.NetworkRequests = requests

	return m
}

// WithWebSocketFrames configures WebSocket frames.
func (m *MockBrowserController) WithWebSocketFrames(frames []browser.WebSocketFrame) *MockBrowserController {
	m.WebSocketFrames = frames

	return m
}

// WithPageSource configures the page source.
func (m *MockBrowserController) WithPageSource(source string) *MockBrowserController {
	m.PageSource = source

	return m
}

// WithScriptSources configures script sources.
func (m *MockBrowserController) WithScriptSources(sources []browser.ScriptSource) *MockBrowserController {
	m.ScriptSources = sources

	return m
}

// WithComputedStyles configures computed styles.
func (m *MockBrowserController) WithComputedStyles(styles []browser.ComputedStyle) *MockBrowserController {
	m.ComputedStyles = styles

	return m
}

// WithMatchedStyles configures matched styles.
func (m *MockBrowserController) WithMatchedStyles(styles *browser.MatchedStyles) *MockBrowserController {
	m.MatchedStyles = styles

	return m
}

// WithCoverage configures coverage data.
func (m *MockBrowserController) WithCoverage(summary *browser.CoverageSummary, js []browser.JSCoverageEntry, css []browser.CSSCoverageEntry) *MockBrowserController {
	m.CoverageSummary = summary
	m.JSCoverageEntries = js
	m.CSSCoverageEntries = css

	return m
}

// WithCookies configures cookies.
func (m *MockBrowserController) WithCookies(cookies []browser.Cookie) *MockBrowserController {
	m.Cookies = cookies

	return m
}

// WithQuerySelectorResult configures query selector result.
func (m *MockBrowserController) WithQuerySelectorResult(elem *browser.DOMElement) *MockBrowserController {
	m.QuerySelectorResult = elem

	return m
}

// WithQuerySelectorAllResult configures query selector all result.
func (m *MockBrowserController) WithQuerySelectorAllResult(elems []browser.DOMElement) *MockBrowserController {
	m.QuerySelectorAllResult = elems

	return m
}

// WithScreenshotData configures screenshot data.
func (m *MockBrowserController) WithScreenshotData(data []byte) *MockBrowserController {
	m.ScreenshotData = data

	return m
}

// WithEvalResult configures eval result.
func (m *MockBrowserController) WithEvalResult(result any) *MockBrowserController {
	m.EvalResult = result

	return m
}

// Verify MockBrowserController implements browser.Controller at compile time.
var _ browser.Controller = (*MockBrowserController)(nil)
