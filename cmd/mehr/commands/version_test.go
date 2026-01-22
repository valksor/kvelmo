//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/valksor/go-toolkit/cli"
	"github.com/valksor/go-toolkit/version"
)

func TestVersionCommand(t *testing.T) {
	// Save original values
	origVersion, origCommit, origBuildTime := version.Version, version.Commit, version.BuildTime
	defer func() {
		version.Set(origVersion, origCommit, origBuildTime)
	}()

	// Set test values
	version.Set("1.2.3", "abc123", "2024-01-15T10:30:00Z")

	tests := []struct {
		name            string
		args            []string
		wantInOutput    []string
		wantNotInOutput []string
	}{
		{
			name: "shows version information",
			args: []string{"version"},
			wantInOutput: []string{
				"mehr 1.2.3",
				"by Valksor",
				"Commit: abc123",
				"Built:  2024-01-15T10:30:00Z",
				"Go:",
			},
		},
		{
			name: "contains all fields",
			args: []string{"version"},
			wantInOutput: []string{
				"mehr",
				"by Valksor",
				"Commit:",
				"Built:",
				"Go:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			// Create test root command
			rootCmd := &cobra.Command{
				Use:   "mehr",
				Short: "Test command",
			}
			rootCmd.SetOut(stdout)
			rootCmd.SetErr(stderr)
			rootCmd.AddCommand(cli.NewVersionCommand("mehr"))

			// Execute command
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}

			// Check stdout contains expected strings
			output := stdout.String()
			for _, want := range tt.wantInOutput {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nGot:\n%s", want, output)
				}
			}

			// Check stdout doesn't contain unwanted strings
			for _, notWant := range tt.wantNotInOutput {
				if strings.Contains(output, notWant) {
					t.Errorf("output should not contain %q\nGot:\n%s", notWant, output)
				}
			}
		})
	}
}

func TestVersionCommand_DefaultValues(t *testing.T) {
	// Set default build values
	version.Set("dev", "none", "unknown")

	stdout := &bytes.Buffer{}
	rootCmd := &cobra.Command{
		Use:   "mehr",
		Short: "Test command",
	}
	rootCmd.SetOut(stdout)
	rootCmd.AddCommand(cli.NewVersionCommand("mehr"))

	rootCmd.SetArgs([]string{"version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "mehr dev") {
		t.Errorf("expected 'mehr dev' in output, got: %s", output)
	}
	if !strings.Contains(output, "Commit: none") {
		t.Errorf("expected 'Commit: none' in output, got: %s", output)
	}
	if !strings.Contains(output, "Built:  unknown") {
		t.Errorf("expected 'Built:  unknown' in output, got: %s", output)
	}
}

func TestVersionCommand_GoVersion(t *testing.T) {
	stdout := &bytes.Buffer{}
	rootCmd := &cobra.Command{
		Use:   "mehr",
		Short: "Test command",
	}
	rootCmd.SetOut(stdout)
	rootCmd.AddCommand(cli.NewVersionCommand("mehr"))

	rootCmd.SetArgs([]string{"version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Go:") {
		t.Errorf("expected 'Go:' in output, got: %s", output)
	}
	// Go version should start with "go1."
	if !strings.Contains(output, "go1.") {
		t.Errorf("expected Go version to contain 'go1.' in output, got: %s", output)
	}
}
