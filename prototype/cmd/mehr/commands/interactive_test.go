//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestInteractiveCommand_Properties verifies the interactive command properties.
func TestInteractiveCommand_Properties(t *testing.T) {
	if interactiveCmd.Use != "interactive" {
		t.Errorf("Use = %q, want %q", interactiveCmd.Use, "interactive")
	}

	if interactiveCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if interactiveCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if interactiveCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

// TestInteractiveCommand_Aliases verifies the interactive command aliases.
func TestInteractiveCommand_Aliases(t *testing.T) {
	expectedAliases := []string{"i", "repl"}

	if len(interactiveCmd.Aliases) != len(expectedAliases) {
		t.Fatalf("Aliases = %v (%d), want %v (%d)", interactiveCmd.Aliases, len(interactiveCmd.Aliases), expectedAliases, len(expectedAliases))
	}

	for i, alias := range interactiveCmd.Aliases {
		if alias != expectedAliases[i] {
			t.Errorf("Aliases[%d] = %q, want %q", i, alias, expectedAliases[i])
		}
	}
}

// TestInteractiveCommand_Flags verifies the interactive command flags.
func TestInteractiveCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "no-history flag",
			flagName:     "no-history",
			shorthand:    "",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := interactiveCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

// TestInteractiveCommand_HasSubcommands verifies interactive has expected subcommands.
func TestInteractiveCommand_HasSubcommands(t *testing.T) {
	// Interactive command should not have subcommands
	// It's a direct command that enters REPL mode
	if len(interactiveCmd.Commands()) > 0 {
		t.Errorf("interactive command should not have subcommands, has %d", len(interactiveCmd.Commands()))
	}
}

// TestInteractiveCommand_CommandPath verifies the command path.
func TestInteractiveCommand_CommandPath(t *testing.T) {
	// Verify the command is accessible from root
	path := interactiveCmd.CommandPath()
	expected := "mehr interactive"

	if path != expected {
		t.Errorf("CommandPath = %q, want %q", path, expected)
	}
}

// TestInteractiveRepeatableAsAlias verifies the command can be called via aliases.
func TestInteractiveRepeatableAsAlias(t *testing.T) {
	// Create a test root command with interactive subcommand
	testRoot := &cobra.Command{
		Use: "test",
	}
	testRoot.AddCommand(interactiveCmd)

	// Verify i alias works
	aliases := interactiveCmd.Aliases
	if !stringSliceContains(aliases, "i") {
		t.Error("interactive command should have 'i' alias")
	}

	// Verify repl alias works
	if !stringSliceContains(aliases, "repl") {
		t.Error("interactive command should have 'repl' alias")
	}
}

// stringSliceContains is a helper to check if a string slice contains a value.
func stringSliceContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}

	return false
}
