//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// Source Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserSourceCommand_Properties(t *testing.T) {
	if browserSourceCmd.Use != "source" {
		t.Errorf("Use = %q, want %q", browserSourceCmd.Use, "source")
	}

	if browserSourceCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserSourceCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if browserSourceCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserSourceCommand_ShortDescription(t *testing.T) {
	expected := "Get page HTML source"
	if browserSourceCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserSourceCmd.Short, expected)
	}
}

func TestBrowserSourceCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"output flag", "output", "o", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := browserSourceCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag %q not found", tt.flagName)

				return
			}

			if tt.shorthand != "" && flag.Shorthand != tt.shorthand {
				t.Errorf("Shorthand = %q, want %q", flag.Shorthand, tt.shorthand)
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("Default = %q, want %q", flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestBrowserSourceCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "source" {
			found = true

			break
		}
	}
	if !found {
		t.Error("source command not registered in browser command")
	}
}

func TestBrowserSourceCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"HTML",
		"source",
		"page",
	}

	for _, substr := range contains {
		if !containsString(browserSourceCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Scripts Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserScriptsCommand_Properties(t *testing.T) {
	if browserScriptsCmd.Use != "scripts" {
		t.Errorf("Use = %q, want %q", browserScriptsCmd.Use, "scripts")
	}

	if browserScriptsCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserScriptsCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if browserScriptsCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserScriptsCommand_ShortDescription(t *testing.T) {
	expected := "List loaded JavaScript sources"
	if browserScriptsCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserScriptsCmd.Short, expected)
	}
}

func TestBrowserScriptsCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"url flag", "url", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := browserScriptsCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag %q not found", tt.flagName)

				return
			}

			if tt.shorthand != "" && flag.Shorthand != tt.shorthand {
				t.Errorf("Shorthand = %q, want %q", flag.Shorthand, tt.shorthand)
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("Default = %q, want %q", flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestBrowserScriptsCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "scripts" {
			found = true

			break
		}
	}
	if !found {
		t.Error("scripts command not registered in browser command")
	}
}

func TestBrowserScriptsCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"JavaScript",
		"sources",
		"URL",
	}

	for _, substr := range contains {
		if !containsString(browserScriptsCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// WebSocket Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserWebSocketCommand_Properties(t *testing.T) {
	if browserWebSocketCmd.Use != "websocket" {
		t.Errorf("Use = %q, want %q", browserWebSocketCmd.Use, "websocket")
	}

	if browserWebSocketCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserWebSocketCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if browserWebSocketCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserWebSocketCommand_ShortDescription(t *testing.T) {
	expected := "Monitor WebSocket connections"
	if browserWebSocketCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserWebSocketCmd.Short, expected)
	}
}

func TestBrowserWebSocketCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"duration flag", "duration", "d", "5"},
		{"url flag", "url", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := browserWebSocketCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag %q not found", tt.flagName)

				return
			}

			if tt.shorthand != "" && flag.Shorthand != tt.shorthand {
				t.Errorf("Shorthand = %q, want %q", flag.Shorthand, tt.shorthand)
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("Default = %q, want %q", flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestBrowserWebSocketCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "websocket" {
			found = true

			break
		}
	}
	if !found {
		t.Error("websocket command not registered in browser command")
	}
}

func TestBrowserWebSocketCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"WebSocket",
		"connections",
		"frames",
	}

	for _, substr := range contains {
		if !containsString(browserWebSocketCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Coverage Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserCoverageCommand_Properties(t *testing.T) {
	if browserCoverageCmd.Use != "coverage" {
		t.Errorf("Use = %q, want %q", browserCoverageCmd.Use, "coverage")
	}

	if browserCoverageCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserCoverageCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if browserCoverageCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserCoverageCommand_ShortDescription(t *testing.T) {
	expected := "Track CSS/JS code coverage"
	if browserCoverageCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserCoverageCmd.Short, expected)
	}
}

func TestBrowserCoverageCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"duration flag", "duration", "d", "5"},
		{"js flag", "js", "", "true"},
		{"css flag", "css", "", "true"},
		{"detail flag", "detail", "", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := browserCoverageCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag %q not found", tt.flagName)

				return
			}

			if tt.shorthand != "" && flag.Shorthand != tt.shorthand {
				t.Errorf("Shorthand = %q, want %q", flag.Shorthand, tt.shorthand)
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("Default = %q, want %q", flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestBrowserCoverageCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "coverage" {
			found = true

			break
		}
	}
	if !found {
		t.Error("coverage command not registered in browser command")
	}
}

func TestBrowserCoverageCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"JavaScript",
		"CSS",
		"coverage",
	}

	for _, substr := range contains {
		if !containsString(browserCoverageCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Styles Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserStylesCommand_Properties(t *testing.T) {
	if browserStylesCmd.Use != "styles --selector <css>" {
		t.Errorf("Use = %q, want %q", browserStylesCmd.Use, "styles --selector <css>")
	}

	if browserStylesCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserStylesCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if browserStylesCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserStylesCommand_ShortDescription(t *testing.T) {
	expected := "Inspect CSS styles on an element"
	if browserStylesCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserStylesCmd.Short, expected)
	}
}

func TestBrowserStylesCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"selector flag", "selector", "", ""},
		{"computed flag", "computed", "", "true"},
		{"matched flag", "matched", "", "false"},
		{"inherited flag", "inherited", "", "false"},
		{"filter flag", "filter", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := browserStylesCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag %q not found", tt.flagName)

				return
			}

			if tt.shorthand != "" && flag.Shorthand != tt.shorthand {
				t.Errorf("Shorthand = %q, want %q", flag.Shorthand, tt.shorthand)
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("Default = %q, want %q", flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestBrowserStylesCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "styles" {
			found = true

			break
		}
	}
	if !found {
		t.Error("styles command not registered in browser command")
	}
}

func TestBrowserStylesCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"CSS",
		"styles",
		"selector",
	}

	for _, substr := range contains {
		if !containsString(browserStylesCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper Function Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 512, "512 B"},
		{"exactly 1KB boundary", 1023, "1023 B"},
		{"just over 1KB", 1024, "1.0 KB"},
		{"multiple KB", 5120, "5.0 KB"},
		{"fractional KB", 1536, "1.5 KB"},
		{"exactly 1MB boundary", 1024*1024 - 1, "1024.0 KB"},
		{"just over 1MB", 1024 * 1024, "1.0 MB"},
		{"multiple MB", 5 * 1024 * 1024, "5.0 MB"},
		{"fractional MB", 1536 * 1024, "1.5 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.expected)
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"exact match lowercase", "hello world", "world", true},
		{"exact match uppercase", "HELLO WORLD", "WORLD", true},
		{"mixed case match", "Hello World", "world", true},
		{"reverse mixed case", "hello world", "WORLD", true},
		{"not found", "hello world", "foo", false},
		{"empty substring", "hello world", "", true},
		{"empty string", "", "foo", false},
		{"both empty", "", "", true},
		{"partial match", "JavaScript", "script", true},
		{"no match different case", "javascript", "PYTHON", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsIgnoreCase(tt.s, tt.substr)
			if got != tt.expected {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.expected)
			}
		})
	}
}
