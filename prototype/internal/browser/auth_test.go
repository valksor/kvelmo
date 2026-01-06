package browser

import (
	"strings"
	"testing"
)

// TestAuthDetection tests authentication detection functionality.
func TestAuthDetection(t *testing.T) {
	// Note: These tests require a browser connection, so they're more like
	// integration tests. Here we test the logic that can be tested without Chrome.

	t.Run("isAuthURL", func(t *testing.T) {
		tests := []struct {
			name     string
			url      string
			expected bool
		}{
			{"login path", "https://example.com/login", true},
			{"signin path", "https://example.com/signin", true},
			{"auth path", "https://example.com/auth", true},
			{"session path", "https://example.com/session", true},
			{"oauth path", "https://example.com/oauth", true},
			{"regular path", "https://example.com/home", false},
			{"root path", "https://example.com/", false},
			{"with query", "https://example.com/login?redirect=/home", true},
			{"mixed case", "https://example.com/LogIn", true},
			{"subpath", "https://example.com/user/login", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isAuthURL(tt.url)
				if result != tt.expected {
					t.Errorf("isAuthURL(%q) = %v, want %v", tt.url, result, tt.expected)
				}
			})
		}
	})

	t.Run("AuthRequirementTypes", func(t *testing.T) {
		tests := []struct {
			name     string
			auth     *AuthRequirement
			expected bool
		}{
			{
				name: "login form",
				auth: &AuthRequirement{
					Type:     "login_form",
					URL:      "https://example.com/login",
					Selector: "#password",
					Hint:     "Please login",
				},
				expected: true,
			},
			{
				name: "HTTP auth",
				auth: &AuthRequirement{
					Type: "http_auth",
					URL:  "https://example.com",
					Hint: "HTTP Authentication required",
				},
				expected: true,
			},
			{
				name: "session expired",
				auth: &AuthRequirement{
					Type: "session_expired",
					URL:  "https://example.com",
					Hint: "Session expired",
				},
				expected: true,
			},
			{
				name: "auth wall",
				auth: &AuthRequirement{
					Type: "auth_wall",
					URL:  "https://example.com",
					Hint: "Authentication required",
				},
				expected: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.auth.Type == "" {
					t.Error("AuthRequirement.Type is empty")
				}
				if tt.auth.URL == "" {
					t.Error("AuthRequirement.URL is empty")
				}
				if tt.auth.Hint == "" {
					t.Error("AuthRequirement.Hint is empty")
				}
			})
		}
	})
}

// TestDisplayHelpers tests the display formatting helper functions.
// Note: These functions now conditionally apply ANSI codes based on TTY detection.
// In non-TTY mode (CI, tests), they return plain text.
func TestDisplayHelpers(t *testing.T) {
	t.Run("displayBold", func(t *testing.T) {
		result := displayBold("test")
		// In non-TTY mode, returns plain text; in TTY mode, has ANSI codes
		if result != "test" && !strings.Contains(result, "\033[") {
			t.Errorf("displayBold() returned unexpected result: %q", result)
		}
	})

	t.Run("displayMuted", func(t *testing.T) {
		result := displayMuted("test")
		// In non-TTY mode, returns plain text; in TTY mode, has ANSI codes
		if result != "test" && !strings.Contains(result, "\033[") {
			t.Errorf("displayMuted() returned unexpected result: %q", result)
		}
	})

	t.Run("displayInfo", func(t *testing.T) {
		result := displayInfo("test")
		// Should contain arrow (always) and possibly ANSI codes
		if !strings.Contains(result, "→") && !strings.Contains(result, "\033[") {
			t.Errorf("displayInfo() should contain arrow or ANSI codes, got: %q", result)
		}
		if len(result) == 0 {
			t.Error("displayInfo() result should not be empty")
		}
	})

	t.Run("displaySuccess", func(t *testing.T) {
		result := displaySuccess("test")
		// In non-TTY mode, returns plain text; in TTY mode, has ANSI codes
		if result != "test" && !strings.Contains(result, "\033[") {
			t.Errorf("displaySuccess() returned unexpected result: %q", result)
		}
	})

	t.Run("displayWarning", func(t *testing.T) {
		result := displayWarning("test")
		// In non-TTY mode, returns plain text; in TTY mode, has ANSI codes
		if result != "test" && !strings.Contains(result, "\033[") {
			t.Errorf("displayWarning() returned unexpected result: %q", result)
		}
	})
}

// TestContains tests the contains helper function.
func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"simple match", "hello world", "world", true},
		{"simple no match", "hello world", "foo", false},
		{"case insensitive", "Hello World", "hello", true},
		{"empty substring", "hello", "", true},
		{"empty string", "", "test", false},
		{"both empty", "", "", true},
		{"substring same as string", "test", "test", true},
		{"first char match", "hello", "h", true},
		{"last char match", "hello", "o", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}
