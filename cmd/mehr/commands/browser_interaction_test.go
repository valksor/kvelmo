//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// Screenshot Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserScreenshotCommand_Properties(t *testing.T) {
	if browserScreenshotCmd.Use != "screenshot [url]" {
		t.Errorf("Use = %q, want %q", browserScreenshotCmd.Use, "screenshot [url]")
	}

	if browserScreenshotCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserScreenshotCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if browserScreenshotCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserScreenshotCommand_ShortDescription(t *testing.T) {
	expected := "Capture screenshot"
	if browserScreenshotCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserScreenshotCmd.Short, expected)
	}
}

func TestBrowserScreenshotCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"format flag", "format", "f", "png"},
		{"output flag", "output", "o", ""},
		{"full-page flag", "full-page", "F", "false"},
		{"quality flag", "quality", "", "80"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := browserScreenshotCmd.Flags().Lookup(tt.flagName)
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

func TestBrowserScreenshotCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "screenshot" {
			found = true

			break
		}
	}
	if !found {
		t.Error("screenshot command not registered in browser command")
	}
}

func TestBrowserScreenshotCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"screenshot",
		"URL",
		"tab",
	}

	for _, substr := range contains {
		if !containsString(browserScreenshotCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// DOM Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserDOMCommand_Properties(t *testing.T) {
	if browserDOMCmd.Use != "dom --selector <css>" {
		t.Errorf("Use = %q, want %q", browserDOMCmd.Use, "dom --selector <css>")
	}

	if browserDOMCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserDOMCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if browserDOMCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserDOMCommand_ShortDescription(t *testing.T) {
	expected := "Query DOM elements"
	if browserDOMCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserDOMCmd.Short, expected)
	}
}

func TestBrowserDOMCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"selector flag", "selector", "", ""},
		{"all flag", "all", "", "false"},
		{"html flag", "html", "", "false"},
		{"computed flag", "computed", "", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := browserDOMCmd.Flags().Lookup(tt.flagName)
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

func TestBrowserDOMCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "dom" {
			found = true

			break
		}
	}
	if !found {
		t.Error("dom command not registered in browser command")
	}
}

func TestBrowserDOMCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"DOM",
		"CSS",
		"selector",
	}

	for _, substr := range contains {
		if !containsString(browserDOMCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Click Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserClickCommand_Properties(t *testing.T) {
	if browserClickCmd.Use != "click --selector <css>" {
		t.Errorf("Use = %q, want %q", browserClickCmd.Use, "click --selector <css>")
	}

	if browserClickCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserClickCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if browserClickCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserClickCommand_ShortDescription(t *testing.T) {
	expected := "Click an element"
	if browserClickCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserClickCmd.Short, expected)
	}
}

func TestBrowserClickCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"selector flag", "selector", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := browserClickCmd.Flags().Lookup(tt.flagName)
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

func TestBrowserClickCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "click" {
			found = true

			break
		}
	}
	if !found {
		t.Error("click command not registered in browser command")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Type Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserTypeCommand_Properties(t *testing.T) {
	if browserTypeCmd.Use != "type --selector <css> <text>" {
		t.Errorf("Use = %q, want %q", browserTypeCmd.Use, "type --selector <css> <text>")
	}

	if browserTypeCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserTypeCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if browserTypeCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserTypeCommand_ShortDescription(t *testing.T) {
	expected := "Type text into an element"
	if browserTypeCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserTypeCmd.Short, expected)
	}
}

func TestBrowserTypeCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"selector flag", "selector", "", ""},
		{"clear flag", "clear", "", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := browserTypeCmd.Flags().Lookup(tt.flagName)
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

func TestBrowserTypeCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "type" {
			found = true

			break
		}
	}
	if !found {
		t.Error("type command not registered in browser command")
	}
}

func TestBrowserTypeCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"text",
		"input",
		"selector",
	}

	for _, substr := range contains {
		if !containsString(browserTypeCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Eval Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserEvalCommand_Properties(t *testing.T) {
	if browserEvalCmd.Use != "eval <expression>" {
		t.Errorf("Use = %q, want %q", browserEvalCmd.Use, "eval <expression>")
	}

	if browserEvalCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserEvalCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if browserEvalCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserEvalCommand_ShortDescription(t *testing.T) {
	expected := "Evaluate JavaScript"
	if browserEvalCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserEvalCmd.Short, expected)
	}
}

func TestBrowserEvalCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "eval" {
			found = true

			break
		}
	}
	if !found {
		t.Error("eval command not registered in browser command")
	}
}

func TestBrowserEvalCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"JavaScript",
		"expression",
	}

	for _, substr := range contains {
		if !containsString(browserEvalCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

// Verify eval requires at least 1 argument.
func TestBrowserEvalCommand_RequiresArg(t *testing.T) {
	if browserEvalCmd.Args == nil {
		t.Fatal("Args validation not configured")
	}

	// Test with no args (should fail)
	err := browserEvalCmd.Args(browserEvalCmd, []string{})
	if err == nil {
		t.Error("Expected error with 0 args, got nil")
	}

	// Test with 1 arg (should pass)
	err = browserEvalCmd.Args(browserEvalCmd, []string{"document.title"})
	if err != nil {
		t.Errorf("Expected no error with 1 arg, got %v", err)
	}
}
