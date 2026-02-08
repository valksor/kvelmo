//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// Command Property Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestLibraryCommand_Properties(t *testing.T) {
	if libraryCmd.Use != "library [list|show|search|pull|remove|update]" {
		t.Errorf("Use = %q, want %q", libraryCmd.Use, "library [list|show|search|pull|remove|update]")
	}

	if libraryCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if libraryCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if libraryCmd.RunE != nil {
		t.Error("RunE should be nil for parent command")
	}
}

func TestLibraryCommand_HasSubcommands(t *testing.T) {
	expectedSubcommands := []string{"pull", "list", "show", "remove", "update"}

	for _, sub := range expectedSubcommands {
		found := false
		for _, cmd := range libraryCmd.Commands() {
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

func TestLibraryPullCommand_Properties(t *testing.T) {
	if libraryPullCmd.Use != "pull <source>" {
		t.Errorf("Use = %q, want %q", libraryPullCmd.Use, "pull <source>")
	}

	if libraryPullCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if libraryPullCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if libraryPullCmd.Args == nil {
		t.Error("Args not set")
	}

	if libraryPullCmd.RunE == nil {
		t.Error("RunE not set")
	}

	// Args should require exactly 1 arg (source)
	if err := libraryPullCmd.Args(libraryPullCmd, []string{}); err == nil {
		t.Error("Args validation should require exactly 1 argument")
	}
}

func TestLibraryListCommand_Properties(t *testing.T) {
	if libraryListCmd.Use != "list" {
		t.Errorf("Use = %q, want %q", libraryListCmd.Use, "list")
	}

	if libraryListCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if libraryListCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestLibraryShowCommand_Properties(t *testing.T) {
	if libraryShowCmd.Use != "show <name> [page]" {
		t.Errorf("Use = %q, want %q", libraryShowCmd.Use, "show <name> [page]")
	}

	if libraryShowCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if libraryShowCmd.Args == nil {
		t.Error("Args not set")
	}

	if libraryShowCmd.RunE == nil {
		t.Error("RunE not set")
	}

	// Args should require 1-2 args
	if err := libraryShowCmd.Args(libraryShowCmd, []string{}); err == nil {
		t.Error("Args validation should require at least 1 argument")
	}

	if err := libraryShowCmd.Args(libraryShowCmd, []string{"name", "page", "extra"}); err == nil {
		t.Error("Args validation should reject more than 2 arguments")
	}
}

func TestLibraryRemoveCommand_Properties(t *testing.T) {
	if libraryRemoveCmd.Use != "remove <name>" {
		t.Errorf("Use = %q, want %q", libraryRemoveCmd.Use, "remove <name>")
	}

	if libraryRemoveCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if libraryRemoveCmd.Args == nil {
		t.Error("Args not set")
	}

	if libraryRemoveCmd.RunE == nil {
		t.Error("RunE not set")
	}

	// Args should require exactly 1 arg
	if err := libraryRemoveCmd.Args(libraryRemoveCmd, []string{}); err == nil {
		t.Error("Args validation should require exactly 1 argument")
	}
}

func TestLibraryUpdateCommand_Properties(t *testing.T) {
	if libraryUpdateCmd.Use != "update [name]" {
		t.Errorf("Use = %q, want %q", libraryUpdateCmd.Use, "update [name]")
	}

	if libraryUpdateCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if libraryUpdateCmd.RunE == nil {
		t.Error("RunE not set")
	}

	// Args should allow 0-1 args
	if err := libraryUpdateCmd.Args(libraryUpdateCmd, []string{}); err != nil {
		t.Errorf("Args validation should allow 0 args: %v", err)
	}

	if err := libraryUpdateCmd.Args(libraryUpdateCmd, []string{"name"}); err != nil {
		t.Errorf("Args validation should allow 1 arg: %v", err)
	}

	if err := libraryUpdateCmd.Args(libraryUpdateCmd, []string{"name", "extra"}); err == nil {
		t.Error("Args validation should reject more than 1 argument")
	}
}

func TestLibraryCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "library" {
			found = true

			break
		}
	}
	if !found {
		t.Error("library command not registered in root command")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Short Description Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestLibraryCommand_ShortDescription(t *testing.T) {
	expected := "Manage documentation library collections"
	if libraryCmd.Short != expected {
		t.Errorf("Short = %q, want %q", libraryCmd.Short, expected)
	}
}

func TestLibraryPullCommand_ShortDescription(t *testing.T) {
	expected := "Pull documentation from a source"
	if libraryPullCmd.Short != expected {
		t.Errorf("Short = %q, want %q", libraryPullCmd.Short, expected)
	}
}

func TestLibraryListCommand_ShortDescription(t *testing.T) {
	expected := "List documentation collections"
	if libraryListCmd.Short != expected {
		t.Errorf("Short = %q, want %q", libraryListCmd.Short, expected)
	}
}

func TestLibraryShowCommand_ShortDescription(t *testing.T) {
	expected := "Show collection details or page content"
	if libraryShowCmd.Short != expected {
		t.Errorf("Short = %q, want %q", libraryShowCmd.Short, expected)
	}
}

func TestLibraryRemoveCommand_ShortDescription(t *testing.T) {
	expected := "Remove a documentation collection"
	if libraryRemoveCmd.Short != expected {
		t.Errorf("Short = %q, want %q", libraryRemoveCmd.Short, expected)
	}
}

func TestLibraryUpdateCommand_ShortDescription(t *testing.T) {
	expected := "Update documentation from source"
	if libraryUpdateCmd.Short != expected {
		t.Errorf("Short = %q, want %q", libraryUpdateCmd.Short, expected)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Long Description Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestLibraryCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"documentation",
		"collections",
		"pull",
		"URLs",
	}

	for _, substr := range contains {
		if !containsString(libraryCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestLibraryPullCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"URL",
		"File",
		"Git",
		"auto",
		"explicit",
		"always",
	}

	for _, substr := range contains {
		if !containsString(libraryPullCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Flag Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestLibraryPullCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"name flag", "name", "n", ""},
		{"mode flag", "mode", "m", "auto"},
		{"shared flag", "shared", "", "false"},
		{"paths flag", "paths", "p", "[]"},
		{"tag flag", "tag", "t", "[]"},
		{"max-depth flag", "max-depth", "", "0"},
		{"max-pages flag", "max-pages", "", "0"},
		{"dry-run flag", "dry-run", "", "false"},
		{"git-ref flag", "git-ref", "", ""},
		{"git-path flag", "git-path", "", ""},
		{"force flag", "force", "", "false"},
		{"continue flag", "continue", "", "false"},
		{"restart flag", "restart", "", "false"},
		{"domain-scope flag", "domain-scope", "", ""},
		{"version-filter flag", "version-filter", "", "false"},
		{"version flag", "version", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := libraryPullCmd.Flags().Lookup(tt.flagName)
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

func TestLibraryListCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"shared flag", "shared", "", "false"},
		{"project flag", "project", "", "false"},
		{"verbose flag", "verbose", "v", "false"},
		{"tag flag", "tag", "", ""},
		{"mode flag", "mode", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := libraryListCmd.Flags().Lookup(tt.flagName)
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

func TestLibraryRemoveCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"force flag", "force", "f", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := libraryRemoveCmd.Flags().Lookup(tt.flagName)
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

func TestLibraryUpdateCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{"full flag", "full", "", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := libraryUpdateCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("Default = %q, want %q", flag.DefValue, tt.defaultValue)
			}
		})
	}
}
