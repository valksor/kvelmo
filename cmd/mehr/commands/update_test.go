//go:build !testbinary
// +build !testbinary

package commands

import (
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/update"
)

func TestUpdateCommand_Properties(t *testing.T) {
	if updateCmd.Use != "update" {
		t.Errorf("Use = %q, want %q", updateCmd.Use, "update")
	}

	if updateCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if updateCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if updateCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestUpdateCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "nightly flag",
			flagName:     "nightly",
			shorthand:    "n",
			defaultValue: "false",
		},
		{
			name:         "check flag",
			flagName:     "check",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "yes flag",
			flagName:     "yes",
			shorthand:    "y",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := updateCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			if tt.shorthand != "" {
				shorthand := updateCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestUpdateCommand_ShortDescription(t *testing.T) {
	expected := "Update Mehrhof to the latest version"
	if updateCmd.Short != expected {
		t.Errorf("Short = %q, want %q", updateCmd.Short, expected)
	}
}

func TestUpdateCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"latest version",
		"GitHub releases",
		"nightly",
	}

	for _, substr := range contains {
		if !containsString(updateCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestUpdateCommand_DocumentsUpdateProcess(t *testing.T) {
	steps := []string{
		"latest release",
		"Downloads the checksums file",
		"Downloads the binary",
		"Verifies SHA256 checksum",
		"Replaces the current binary",
	}

	for _, step := range steps {
		if !containsString(updateCmd.Long, step) {
			t.Errorf("Long description does not document step %q", step)
		}
	}
}

func TestUpdateCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "update" {
			found = true

			break
		}
	}
	if !found {
		t.Error("update command not registered in root command")
	}
}

func TestUpdateCommand_NightlyFlagShorthand(t *testing.T) {
	flag := updateCmd.Flags().Lookup("nightly")
	if flag == nil {
		t.Fatal("nightly flag not found")

		return
	}
	if flag.Shorthand != "n" {
		t.Errorf("nightly flag shorthand = %q, want 'n'", flag.Shorthand)
	}
}

func TestUpdateCommand_YesFlagShorthand(t *testing.T) {
	flag := updateCmd.Flags().Lookup("yes")
	if flag == nil {
		t.Fatal("yes flag not found")

		return
	}
	if flag.Shorthand != "y" {
		t.Errorf("yes flag shorthand = %q, want 'y'", flag.Shorthand)
	}
}

// TestGetReleaseURLs tests the getReleaseURLs function.
func TestGetReleaseURLs(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "standard version",
			version: "v1.0.0",
		},
		{
			name:    "version without v prefix",
			version: "1.0.0",
		},
		{
			name:    "nightly version",
			version: "v1.0.0-beta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &update.UpdateStatus{
				LatestVersion: tt.version,
			}

			checksumsURL, signatureURL := getReleaseURLs(status)

			// Verify checksums URL
			if !strings.Contains(checksumsURL, "checksums.txt") {
				t.Errorf("checksumsURL = %q, want to contain 'checksums.txt'", checksumsURL)
			}
			if !strings.Contains(checksumsURL, tt.version) {
				t.Errorf("checksumsURL = %q, want to contain version %q", checksumsURL, tt.version)
			}

			// Verify signature URL
			if !strings.Contains(signatureURL, "checksums.txt.minisig") {
				t.Errorf("signatureURL = %q, want to contain 'checksums.txt.minisig'", signatureURL)
			}
			if !strings.Contains(signatureURL, tt.version) {
				t.Errorf("signatureURL = %q, want to contain version %q", signatureURL, tt.version)
			}
		})
	}
}
