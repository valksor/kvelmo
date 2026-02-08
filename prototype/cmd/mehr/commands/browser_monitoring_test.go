//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// Console Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserConsoleCommand_Properties(t *testing.T) {
	if browserConsoleCmd.Use != "console" {
		t.Errorf("Use = %q, want %q", browserConsoleCmd.Use, "console")
	}

	if browserConsoleCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserConsoleCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if browserConsoleCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserConsoleCommand_ShortDescription(t *testing.T) {
	expected := "Capture console logs"
	if browserConsoleCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserConsoleCmd.Short, expected)
	}
}

func TestBrowserConsoleCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"duration flag", "duration", "d", "1"},
		{"level flag", "level", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := browserConsoleCmd.Flags().Lookup(tt.flagName)
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

func TestBrowserConsoleCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "console" {
			found = true

			break
		}
	}
	if !found {
		t.Error("console command not registered in browser command")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Network Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserNetworkCommand_Properties(t *testing.T) {
	if browserNetworkCmd.Use != "network" {
		t.Errorf("Use = %q, want %q", browserNetworkCmd.Use, "network")
	}

	if browserNetworkCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserNetworkCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if browserNetworkCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserNetworkCommand_ShortDescription(t *testing.T) {
	expected := "Capture network requests"
	if browserNetworkCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserNetworkCmd.Short, expected)
	}
}

func TestBrowserNetworkCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"duration flag", "duration", "d", "3"},
		{"type flag", "type", "", ""},
		{"body flag", "body", "", "false"},
		{"max-body-size flag", "max-body-size", "", "1048576"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := browserNetworkCmd.Flags().Lookup(tt.flagName)
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

func TestBrowserNetworkCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "network" {
			found = true

			break
		}
	}
	if !found {
		t.Error("network command not registered in browser command")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Cookies Export Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserCookiesExportCommand_Properties(t *testing.T) {
	if browserCookiesExportCmd.Use != "export" {
		t.Errorf("Use = %q, want %q", browserCookiesExportCmd.Use, "export")
	}

	if browserCookiesExportCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserCookiesExportCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserCookiesExportCommand_ShortDescription(t *testing.T) {
	expected := "Export cookies to file"
	if browserCookiesExportCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserCookiesExportCmd.Short, expected)
	}
}

func TestBrowserCookiesExportCommand_RegisteredInCookies(t *testing.T) {
	found := false
	for _, cmd := range browserCookiesCmd.Commands() {
		if cmd.Name() == "export" {
			found = true

			break
		}
	}
	if !found {
		t.Error("export command not registered in cookies command")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Cookies Import Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserCookiesImportCommand_Properties(t *testing.T) {
	if browserCookiesImportCmd.Use != "import" {
		t.Errorf("Use = %q, want %q", browserCookiesImportCmd.Use, "import")
	}

	if browserCookiesImportCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if browserCookiesImportCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestBrowserCookiesImportCommand_ShortDescription(t *testing.T) {
	expected := "Import cookies from file"
	if browserCookiesImportCmd.Short != expected {
		t.Errorf("Short = %q, want %q", browserCookiesImportCmd.Short, expected)
	}
}

func TestBrowserCookiesImportCommand_RegisteredInCookies(t *testing.T) {
	found := false
	for _, cmd := range browserCookiesCmd.Commands() {
		if cmd.Name() == "import" {
			found = true

			break
		}
	}
	if !found {
		t.Error("import command not registered in cookies command")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Cookies Command Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserCookiesCommand_Properties(t *testing.T) {
	if browserCookiesCmd.Use != "cookies <subcommand>" {
		t.Errorf("Use = %q, want %q", browserCookiesCmd.Use, "cookies <subcommand>")
	}

	if browserCookiesCmd.Short == "" {
		t.Error("Short description is empty")
	}
}

func TestBrowserCookiesCommand_HasSubcommands(t *testing.T) {
	expectedSubcommands := []string{"export", "import"}

	for _, sub := range expectedSubcommands {
		found := false
		for _, cmd := range browserCookiesCmd.Commands() {
			if cmd.Name() == sub {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("Missing subcommand %q", sub)
		}
	}
}

func TestBrowserCookiesCommand_RegisteredInBrowser(t *testing.T) {
	found := false
	for _, cmd := range browserCmd.Commands() {
		if cmd.Name() == "cookies" {
			found = true

			break
		}
	}
	if !found {
		t.Error("cookies command not registered in browser command")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Long Description Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBrowserConsoleCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"console",
		"logs",
		"duration",
	}

	for _, substr := range contains {
		if !containsString(browserConsoleCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestBrowserNetworkCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"network",
		"requests",
		"body",
	}

	for _, substr := range contains {
		if !containsString(browserNetworkCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}
