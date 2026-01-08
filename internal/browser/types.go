package browser

import (
	"context"
	"strings"
	"time"
)

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Cookie represents a browser cookie.
type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Secure   bool   `json:"secure"`
	HTTPOnly bool   `json:"http_only"`
	SameSite string `json:"same_site,omitempty"` // "Strict", "Lax", "None"
	Expires  int64  `json:"expires,omitempty"`   // Unix timestamp
}

// CookieStorage manages cookie persistence.
type CookieStorage struct {
	cookieDir string // Directory containing cookie files
}

// Config holds browser configuration.
type Config struct {
	// Host is the CDP host to connect to
	Host string
	// Port is the CDP port (0 = random port for isolated browser, 9222 = existing Chrome)
	Port int
	// RemoteDebug indicates whether to connect to existing Chrome with remote debugging
	RemoteDebug bool
	// Headless indicates whether to launch browser in headless mode
	Headless bool
	// IgnoreCertErrors indicates whether to ignore SSL certificate errors (default: true for local dev)
	IgnoreCertErrors bool
	// Timeout is the default timeout for operations
	Timeout time.Duration
	// ScreenshotDir is the directory to save screenshots
	ScreenshotDir string
	// UserDataDir is the user data directory for isolated browser (empty = auto-generate)
	UserDataDir string
	// CookieProfile is the cookie profile name to use (default: "default")
	CookieProfile string
	// CookieAutoLoad enables automatic cookie loading on connect
	CookieAutoLoad bool
	// CookieAutoSave enables automatic cookie saving on disconnect
	CookieAutoSave bool
	// CookieDir is the directory for cookie storage (empty = use default ~/.mehrhof/)
	CookieDir string
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Host:             "localhost",
		Port:             0, // Random port by default
		RemoteDebug:      false,
		Headless:         false,
		IgnoreCertErrors: true, // Ignore cert errors by default for local dev
		Timeout:          30 * time.Second,
		ScreenshotDir:    ".mehrhof/screenshots",
	}
}

// Tab represents a browser tab/page.
type Tab struct {
	ID    string
	Title string
	URL   string
}

// ScreenshotOptions for capture operations.
type ScreenshotOptions struct {
	Format   string // "png" or "jpeg"
	Quality  int    // JPEG quality (1-100), only used for jpeg
	FullPage bool   // Capture entire scrollable page
}

// DOMElement represents a DOM node.
type DOMElement struct {
	NodeID      int64
	BackendID   int64
	TagName     string
	Attributes  map[string]string
	TextContent string
	OuterHTML   string
	ChildCount  int
	Visible     bool
	Interactive bool
	X, Y        float64 // Element position for clicking
}

// NetworkRequest represents an HTTP request.
type NetworkRequest struct {
	ID           string
	URL          string
	Method       string
	Status       int
	StatusText   string
	Headers      map[string]string
	ResourceType string
	MimeType     string
	Timestamp    time.Time
	RequestBody  string
	ResponseBody string
}

// ConsoleMessage represents a console log entry.
type ConsoleMessage struct {
	Level     string // "log", "warn", "error", "debug", "info"
	Text      string
	URL       string
	Line      int
	Column    int
	Timestamp time.Time
}

// ConsoleFilter defines which console messages to capture.
type ConsoleFilter struct {
	Levels    []string // Capture only these levels (empty = all)
	Pattern   string   // Only capture messages matching this pattern
	SourceURL string   // Only capture messages from this URL
}

// AuthRequirement represents detected authentication requirements.
type AuthRequirement struct {
	Type     string // "login_form", "http_auth", "session_expired", "auth_wall"
	URL      string
	Selector string // For login forms
	Hint     string // User-friendly hint
}

// Controller provides high-level browser operations.
//
//nolint:interfacebloat // Controller interface requires 20 methods for comprehensive browser automation
type Controller interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	GetPort() int // Returns actual port (for random port allocation)

	// Tab management
	ListTabs(ctx context.Context) ([]Tab, error)
	OpenTab(ctx context.Context, url string) (*Tab, error)
	CloseTab(ctx context.Context, tabID string) error
	SwitchTab(ctx context.Context, tabID string) (*Tab, error)
	Navigate(ctx context.Context, tabID, url string) error
	Reload(ctx context.Context, tabID string, hard bool) error

	// Page interaction
	Screenshot(ctx context.Context, tabID string, opts ScreenshotOptions) ([]byte, error)
	QuerySelector(ctx context.Context, tabID, selector string) (*DOMElement, error)
	QuerySelectorAll(ctx context.Context, tabID, selector string) ([]DOMElement, error)
	Click(ctx context.Context, tabID, selector string) error
	Type(ctx context.Context, tabID, selector, text string, clearField bool) error
	Eval(ctx context.Context, tabID, expression string) (any, error)

	// Monitoring
	GetConsoleLogs(ctx context.Context, tabID string, duration time.Duration) ([]ConsoleMessage, error)
	GetNetworkRequests(ctx context.Context, tabID string, duration time.Duration) ([]NetworkRequest, error)

	// Authentication
	DetectAuth(ctx context.Context, tabID string) (*AuthRequirement, error)
	WaitForLogin(ctx context.Context, tabID string, auth *AuthRequirement) error

	// Cookie management
	GetCookies(ctx context.Context) ([]Cookie, error)
	SetCookies(ctx context.Context, cookies []Cookie) error
}
