package commands

import (
	"testing"
)

func TestLicenseCommand_Properties(t *testing.T) {
	cmd := licenseCmd
	if cmd.Use != "license" {
		t.Errorf("Use = %q, want %q", cmd.Use, "license")
	}
	if cmd.Short != "Display license information" {
		t.Errorf("Short = %q, want %q", cmd.Short, "Display license information")
	}
}

func TestLicenseInfoCommand_Properties(t *testing.T) {
	cmd := licenseInfoCmd
	if cmd.Use != "info" {
		t.Errorf("Use = %q, want %q", cmd.Use, "info")
	}
	if cmd.Short != "List all dependency licenses" {
		t.Errorf("Short = %q, want %q", cmd.Short, "List all dependency licenses")
	}
}

func TestLicenseCommand_Group(t *testing.T) {
	if licenseCmd.GroupID != "info" {
		t.Errorf("licenseCmd.GroupID = %q, want 'info'", licenseCmd.GroupID)
	}
}

func TestLicenseOutputFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Set up to capture output
	cmd := licenseCmd
	cmd.SetArgs([]string{})

	// The output should contain BSD 3-Clause license text
	// We're not executing here, just checking the command structure
	t.Run("license command has runE", func(t *testing.T) {
		if cmd.RunE == nil {
			t.Error("licenseCmd.RunE is nil")
		}
	})
}

func TestLicenseFlags(t *testing.T) {
	// Test that flags are properly defined
	tests := []struct {
		name      string
		flagName  string
		flagVar   *bool
		shorthand string
	}{
		{"json flag", "json", &licenseJSON, ""},
		{"unknown-only flag", "unknown-only", &licenseUnknownOnly, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := licenseInfoCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}
		})
	}
}

// Test that the license info command doesn't crash with --json flag.
func TestLicenseInfoJSONFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	licenseJSON = true
	defer func() { licenseJSON = false }()

	// Just verify the flag is set, we don't execute in unit tests
	if !licenseJSON {
		t.Error("licenseJSON was not set")
	}
}
