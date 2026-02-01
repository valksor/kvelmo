//go:build !testbinary
// +build !testbinary

package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/security"
)

func TestScanCommand_Properties(t *testing.T) {
	if scanCmd.Use != "scan" {
		t.Errorf("Use = %q, want %q", scanCmd.Use, "scan")
	}

	if scanCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if scanCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if scanCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestScanCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "dir flag",
			flagName:     "dir",
			shorthand:    "d",
			defaultValue: ".",
		},
		{
			name:         "scanners flag",
			flagName:     "scanners",
			shorthand:    "s",
			defaultValue: "[]",
		},
		{
			name:         "output flag",
			flagName:     "output",
			shorthand:    "o",
			defaultValue: "",
		},
		{
			name:         "format flag",
			flagName:     "format",
			shorthand:    "",
			defaultValue: "text",
		},
		{
			name:         "fail-level flag",
			flagName:     "fail-level",
			shorthand:    "",
			defaultValue: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := scanCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			if tt.shorthand != "" {
				shorthand := scanCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestScanCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "scan" {
			found = true

			break
		}
	}
	if !found {
		t.Error("scan command not registered in root command")
	}
}

func TestScanCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"security",
		"gosec",
		"gitleaks",
		"semgrep",
	}

	for _, substr := range contains {
		if !containsString(scanCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestValidateScanDirectory(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "valid directory",
			setup: func(t *testing.T) string {
				t.Helper()

				return t.TempDir()
			},
			wantErr: false,
		},
		{
			name: "file instead of directory",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				filePath := filepath.Join(dir, "notadir.txt")
				if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
					t.Fatalf("write file: %v", err)
				}

				return filePath
			},
			wantErr: true,
		},
		{
			name: "non-existent directory",
			setup: func(t *testing.T) string {
				t.Helper()

				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			err := validateScanDirectory(dir)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateScanDirectory(%q) error = %v, wantErr %v", dir, err, tt.wantErr)
			}
		})
	}
}

func TestSaveOutput(t *testing.T) {
	t.Run("writes file with content", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "report.txt")
		content := "scan results here"

		err := saveOutput(path, content)
		if err != nil {
			t.Fatalf("saveOutput() error = %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		if string(got) != content {
			t.Errorf("file content = %q, want %q", string(got), content)
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "sub", "dir", "report.txt")

		err := saveOutput(path, "content")
		if err != nil {
			t.Fatalf("saveOutput() error = %v", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("file was not created")
		}
	})
}

func TestRegisterDefaultScanners(t *testing.T) {
	registry := security.NewScannerRegistry()
	registerDefaultScanners(registry)

	expectedScanners := []string{
		"gosec",
		"gitleaks",
		"govulncheck",
		"semgrep",
		"npm-audit",
		"eslint-security",
		"bandit",
		"pip-audit",
	}

	registered := registry.List()

	for _, name := range expectedScanners {
		found := false
		for _, r := range registered {
			if r == name {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("scanner %q not registered (registered: %v)", name, registered)
		}
	}
}

func TestRunScan_InvalidFormat(t *testing.T) {
	orig := scanFormat
	defer func() { scanFormat = orig }()

	scanFormat = "xml"

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runScan(cmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid format, got nil")
	}

	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "invalid format")
	}
}

func TestRunScan_InvalidFailLevel(t *testing.T) {
	origFormat := scanFormat
	defer func() { scanFormat = origFormat }()

	origLevel := scanFailLevel
	defer func() { scanFailLevel = origLevel }()

	scanFormat = "text"
	scanFailLevel = "extreme"

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runScan(cmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid fail-level, got nil")
	}

	if !strings.Contains(err.Error(), "invalid fail-level") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "invalid fail-level")
	}
}
