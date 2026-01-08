//go:build !no_browser
// +build !no_browser

package browser

import (
	"testing"
	"time"
)

// TestConfig tests browser configuration.
func TestConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		cfg := DefaultConfig()

		if cfg.Host == "" {
			t.Error("Host should not be empty")
		}
		if cfg.Port != 0 {
			t.Error("Port should be 0 (random) by default")
		}
		if cfg.RemoteDebug {
			t.Error("RemoteDebug should be false by default")
		}
		if cfg.Headless {
			t.Error("Headless should be false by default")
		}
		if !cfg.IgnoreCertErrors {
			t.Error("IgnoreCertErrors should be true by default (for local dev)")
		}
		if cfg.Timeout == 0 {
			t.Error("Timeout should not be 0")
		}
		if cfg.ScreenshotDir == "" {
			t.Error("ScreenshotDir should not be empty")
		}
	})

	t.Run("CustomConfig", func(t *testing.T) {
		cfg := Config{
			Host:             "192.168.1.1",
			Port:             9222,
			RemoteDebug:      true,
			Headless:         true,
			IgnoreCertErrors: false, // Set to false for strict cert validation
			Timeout:          60 * time.Second,
			ScreenshotDir:    "/tmp/screenshots",
			UserDataDir:      "/tmp/chrome-profile",
		}

		if cfg.Host != "192.168.1.1" {
			t.Errorf("Host = %s, want 192.168.1.1", cfg.Host)
		}
		if cfg.Port != 9222 {
			t.Errorf("Port = %d, want 9222", cfg.Port)
		}
		if !cfg.RemoteDebug {
			t.Error("RemoteDebug should be true")
		}
		if !cfg.Headless {
			t.Error("Headless should be true")
		}
		if cfg.IgnoreCertErrors {
			t.Error("IgnoreCertErrors should be false in this custom config")
		}
		if cfg.Timeout != 60*time.Second {
			t.Errorf("Timeout = %v, want 60s", cfg.Timeout)
		}
		if cfg.ScreenshotDir != "/tmp/screenshots" {
			t.Errorf("ScreenshotDir = %s, want /tmp/screenshots", cfg.ScreenshotDir)
		}
		if cfg.UserDataDir != "/tmp/chrome-profile" {
			t.Errorf("UserDataDir = %s, want /tmp/chrome-profile", cfg.UserDataDir)
		}
	})
}

// TestTab tests tab structure.
func TestTab(t *testing.T) {
	tab := Tab{
		ID:    "tab-123",
		Title: "Example Domain",
		URL:   "https://example.com",
	}

	if tab.ID == "" {
		t.Error("Tab ID should not be empty")
	}
	if tab.Title == "" {
		t.Error("Tab Title should not be empty")
	}
	if tab.URL == "" {
		t.Error("Tab URL should not be empty")
	}
}

// TestScreenshotOptions tests screenshot options.
func TestScreenshotOptions(t *testing.T) {
	t.Run("ZeroOptions", func(t *testing.T) {
		opts := ScreenshotOptions{}

		// Zero values are valid defaults
		if opts.FullPage {
			t.Error("FullPage should be false by default")
		}
	})

	t.Run("CustomOptions", func(t *testing.T) {
		opts := ScreenshotOptions{
			Format:   "jpeg",
			Quality:  90,
			FullPage: true,
		}

		if opts.Format != "jpeg" {
			t.Errorf("Format = %s, want jpeg", opts.Format)
		}
		if opts.Quality != 90 {
			t.Errorf("Quality = %d, want 90", opts.Quality)
		}
		if !opts.FullPage {
			t.Error("FullPage should be true")
		}
	})
}

// TestDOMElement tests DOM element structure.
func TestDOMElement(t *testing.T) {
	elem := DOMElement{
		NodeID:    123,
		BackendID: 456,
		TagName:   "div",
		Attributes: map[string]string{
			"id":    "test-id",
			"class": "test-class",
		},
		TextContent: "Hello, World!",
		OuterHTML:   "<div id=\"test-id\">Hello, World!</div>",
		ChildCount:  0,
		Visible:     true,
		X:           100.5,
		Y:           200.5,
	}

	if elem.NodeID == 0 {
		t.Error("NodeID should not be 0")
	}
	if elem.BackendID == 0 {
		t.Error("BackendID should not be 0")
	}
	if elem.TagName == "" {
		t.Error("TagName should not be empty")
	}
	if elem.Attributes == nil {
		t.Error("Attributes map should not be nil")
	}
	if elem.TextContent == "" {
		t.Error("TextContent should not be empty")
	}
	if elem.OuterHTML == "" {
		t.Error("OuterHTML should not be empty")
	}
	if !elem.Visible {
		t.Error("Visible should be true")
	}
	if elem.X == 0 {
		t.Error("X should not be 0")
	}
	if elem.Y == 0 {
		t.Error("Y should not be 0")
	}
}

// TestNetworkRequest tests network request structure.
func TestNetworkRequest(t *testing.T) {
	req := NetworkRequest{
		ID:         "req-123",
		URL:        "https://api.example.com/data",
		Method:     "GET",
		Status:     200,
		StatusText: "OK",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "Test/1.0",
		},
		ResourceType: "XHR",
		MimeType:     "application/json",
		Timestamp:    time.Now(),
		RequestBody:  `{"test": true}`,
		ResponseBody: `{"result": "success"}`,
	}

	if req.ID == "" {
		t.Error("ID should not be empty")
	}
	if req.URL == "" {
		t.Error("URL should not be empty")
	}
	if req.Method == "" {
		t.Error("Method should not be empty")
	}
	if req.Status == 0 {
		t.Error("Status should not be 0")
	}
	if req.Headers == nil {
		t.Error("Headers should not be nil")
	}
	if req.ResourceType == "" {
		t.Error("ResourceType should not be empty")
	}
	if req.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

// TestConsoleMessage tests console message structure.
func TestConsoleMessage(t *testing.T) {
	msg := ConsoleMessage{
		Level:     "error",
		Text:      "Uncaught Error: Something went wrong",
		URL:       "https://example.com/app.js",
		Line:      42,
		Column:    10,
		Timestamp: time.Now(),
	}

	if msg.Level == "" {
		t.Error("Level should not be empty")
	}
	if msg.Text == "" {
		t.Error("Text should not be empty")
	}
	if msg.URL == "" {
		t.Error("URL should not be empty")
	}
	if msg.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

// TestConsoleFilter tests console filter structure.
func TestConsoleFilter(t *testing.T) {
	filter := ConsoleFilter{
		Levels:    []string{"error", "warn"},
		Pattern:   "API",
		SourceURL: "https://example.com",
	}

	if filter.Levels == nil {
		t.Error("Levels should not be nil")
	}
	if len(filter.Levels) != 2 {
		t.Errorf("got %d levels, want 2", len(filter.Levels))
	}
	if filter.Pattern == "" {
		t.Error("Pattern should not be empty")
	}
	if filter.SourceURL == "" {
		t.Error("SourceURL should not be empty")
	}

	// Test with empty filter
	emptyFilter := ConsoleFilter{}
	if emptyFilter.Levels != nil {
		t.Error("Levels should be nil for empty filter")
	}
	if emptyFilter.Pattern != "" {
		t.Error("Pattern should be empty for empty filter")
	}
	if emptyFilter.SourceURL != "" {
		t.Error("SourceURL should be empty for empty filter")
	}
}

// TestAuthRequirement tests authentication requirement structure.
func TestAuthRequirement(t *testing.T) {
	auth := AuthRequirement{
		Type:     "login_form",
		URL:      "https://example.com/login",
		Selector: "#password",
		Hint:     "Please enter your credentials",
	}

	if auth.Type == "" {
		t.Error("Type should not be empty")
	}
	if auth.URL == "" {
		t.Error("URL should not be empty")
	}
	if auth.Hint == "" {
		t.Error("Hint should not be empty")
	}

	// Test different auth types
	types := []string{
		"login_form",
		"http_auth",
		"session_expired",
		"auth_wall",
	}

	for _, authType := range types {
		auth := AuthRequirement{
			Type: authType,
			URL:  "https://example.com",
			Hint: "Authentication required",
		}

		if auth.Type != authType {
			t.Errorf("Type = %s, want %s", auth.Type, authType)
		}
	}
}
