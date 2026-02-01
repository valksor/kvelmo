//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// captureOutput captures stdout during command execution.
func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	return buf.String()
}

func TestGenerateSecretCommand(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantInOutput    []string
		wantNotInOutput []string
		validateSecret  func(t *testing.T, secret string)
	}{
		{
			name: "generates cryptographically secure secret",
			args: []string{"generate-secret"},
			wantInOutput: []string{
				"# Add this to your environment or CI/CD secrets:",
				"export MEHRHOF_STATE_SECRET=",
				"# Or in GitHub Actions / GitLab CI:",
				"MEHRHOF_STATE_SECRET:",
			},
			validateSecret: func(t *testing.T, secret string) {
				t.Helper()
				// Secret should be base64 encoded (decode should succeed)
				_, err := base64.StdEncoding.DecodeString(secret)
				if err != nil {
					t.Errorf("generated secret is not valid base64: %v", err)
				}

				// Secret should be at least 32 characters (base64 of 32 bytes)
				if len(secret) < 32 {
					t.Errorf("generated secret too short: got %d, want >= 32", len(secret))
				}

				// Secret should not contain whitespace (except newlines in output)
				secret = strings.TrimSpace(secret)
				if strings.ContainsAny(secret, " \t\r") {
					t.Error("generated secret contains whitespace")
				}
			},
		},
		{
			name: "contains proper export format",
			args: []string{"generate-secret"},
			wantInOutput: []string{
				"export MEHRHOF_STATE_SECRET",
			},
		},
		{
			name: "contains CI/CD format",
			args: []string{"generate-secret"},
			wantInOutput: []string{
				"MEHRHOF_STATE_SECRET:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output string
			var execErr error

			// Capture output
			output = captureOutput(func() {
				rootCmd := &cobra.Command{Use: "mehr"}
				rootCmd.SetArgs(tt.args)
				rootCmd.AddCommand(generateSecretCmd)
				execErr = rootCmd.Execute()
			})

			if execErr != nil {
				t.Fatalf("Execute: %v", execErr)
			}

			// Check output contains expected strings
			for _, want := range tt.wantInOutput {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nGot:\n%s", want, output)
				}
			}

			// Check output doesn't contain unwanted strings
			for _, notWant := range tt.wantNotInOutput {
				if strings.Contains(output, notWant) {
					t.Errorf("output should not contain %q\nGot:\n%s", notWant, output)
				}
			}

			// Extract and validate the secret
			if tt.validateSecret != nil {
				// Find the secret in the export line
				lines := strings.Split(output, "\n")
				var secret string
				for _, line := range lines {
					if strings.HasPrefix(line, "export MEHRHOF_STATE_SECRET=") {
						// Extract secret between quotes
						start := strings.Index(line, `"`)
						if start != -1 {
							end := strings.Index(line[start+1:], `"`)
							if end != -1 {
								secret = line[start+1 : start+1+end]
							}
						}
					} else if strings.HasPrefix(line, "MEHRHOF_STATE_SECRET:") {
						// Extract from a CI / CD format
						parts := strings.SplitN(line, ":", 2)
						if len(parts) == 2 {
							secret = strings.TrimSpace(parts[1])
						}
					}
				}

				if secret != "" {
					tt.validateSecret(t, secret)
				}
			}
		})
	}
}

func TestGenerateSecretCommand_Uniqueness(t *testing.T) {
	// Generate multiple secrets and verify they're different
	var output1, output2 string

	output1 = captureOutput(func() {
		rootCmd := &cobra.Command{Use: "mehr"}
		rootCmd.SetArgs([]string{"generate-secret"})
		rootCmd.AddCommand(generateSecretCmd)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})

	output2 = captureOutput(func() {
		rootCmd := &cobra.Command{Use: "mehr"}
		rootCmd.SetArgs([]string{"generate-secret"})
		rootCmd.AddCommand(generateSecretCmd)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})

	// Extract secrets from output
	extractSecret := func(output string) string {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "export MEHRHOF_STATE_SECRET=") {
				start := strings.Index(line, `"`)
				if start != -1 {
					end := strings.Index(line[start+1:], `"`)
					if end != -1 {
						return line[start+1 : start+1+end]
					}
				}
			}
		}

		return ""
	}

	secret1 := extractSecret(output1)
	secret2 := extractSecret(output2)

	if secret1 == "" {
		t.Fatal("could not extract first secret from output")
	}
	if secret2 == "" {
		t.Fatal("could not extract second secret from output")
	}

	// Secrets should be different (cryptographically random)
	if secret1 == secret2 {
		t.Error("generated secrets are identical - randomness may be compromised")
	}
}

func TestGenerateSecretCommand_Length(t *testing.T) {
	output := captureOutput(func() {
		rootCmd := &cobra.Command{Use: "mehr"}
		rootCmd.SetArgs([]string{"generate-secret"})
		rootCmd.AddCommand(generateSecretCmd)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})

	// Extract secret
	extractSecret := func(output string) string {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "export MEHRHOF_STATE_SECRET=") {
				start := strings.Index(line, `"`)
				if start != -1 {
					end := strings.Index(line[start+1:], `"`)
					if end != -1 {
						return line[start+1 : start+1+end]
					}
				}
			}
		}

		return ""
	}

	secret := extractSecret(output)
	if secret == "" {
		t.Fatal("could not extract secret from output")
	}

	// 32 bytes base64 encoded = 44 characters (with padding)
	expectedLen := base64.StdEncoding.EncodedLen(32)
	if len(secret) != expectedLen {
		t.Errorf("secret length: got %d, want %d", len(secret), expectedLen)
	}

	// Should decode to exactly 32 bytes
	decoded, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		t.Fatalf("failed to decode secret: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("decoded secret length: got %d, want 32", len(decoded))
	}
}

// TestGenerateSecretCommand_Integration tests the command is registered correctly.
func TestGenerateSecretCommand_Integration(t *testing.T) {
	// Verify the command is registered
	if generateSecretCmd == nil {
		t.Fatal("generateSecretCmd is not registered")
	}

	// Check command properties
	if generateSecretCmd.Use != "generate-secret" {
		t.Errorf("command Use: got %q, want %q", generateSecretCmd.Use, "generate-secret")
	}

	// Check that it has a RunE function
	if generateSecretCmd.RunE == nil {
		t.Error("command RunE is nil")
	}

	// Verify the command can be added to a root command
	rootCmd := &cobra.Command{Use: "mehr"}
	rootCmd.AddCommand(generateSecretCmd)

	// Find the command in the root command
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "generate-secret" {
			found = true

			break
		}
	}

	if !found {
		t.Error("generate-secret command not found in root command")
	}
}

// Example_generateSecretCommand provides an example usage.
func Example_generateSecretCommand() {
	// This example shows how to use the generate-secret command programmatically
	rootCmd := &cobra.Command{Use: "mehr"}
	rootCmd.AddCommand(generateSecretCmd)

	// In actual usage, you would execute:
	// rootCmd.SetArgs([]string{"generate-secret"})
	// rootCmd.Execute()
	fmt.Println("mehr generate-secret")
	// Output:
	// mehr generate-secret
}
