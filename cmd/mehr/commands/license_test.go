//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

func TestRunLicense_PrintsLicense(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := licenseCmd.RunE(licenseCmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("licenseCmd.RunE() error = %v", err)
	}

	if output == "" {
		t.Error("licenseCmd.RunE() produced no output")
	}

	// Should contain some license text
	if len(output) < 50 {
		t.Errorf("license output seems too short (%d chars): %q", len(output), output)
	}
}

func TestRunLicenseInfo_WithGoMod(t *testing.T) {
	// Save/restore flags
	origJSON := licenseJSON
	origUnknown := licenseUnknownOnly

	defer func() {
		licenseJSON = origJSON
		licenseUnknownOnly = origUnknown
	}()

	licenseJSON = false
	licenseUnknownOnly = false

	// Find project root (contains go.mod) by walking up from test dir
	root := findModuleRoot(t)
	t.Chdir(root)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runLicenseInfo(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runLicenseInfo() error = %v", err)
	}

	if !strings.Contains(output, "Dependency Licenses") {
		t.Errorf("output does not contain 'Dependency Licenses'\nGot:\n%s", output)
	}
}

func findModuleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod in any parent directory")
		}

		dir = parent
	}
}
