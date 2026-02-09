//go:build !testbinary
// +build !testbinary

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// setTestHome sets HOME to a temp directory for testing.
// This ensures workspace data is stored in a temp location during tests.
func setTestHome(t *testing.T, tmpDir string) {
	t.Helper()
	t.Setenv("HOME", tmpDir)
}

func TestInitCommand(t *testing.T) {
	tests := []struct {
		name            string
		wantFiles       []string
		wantInOutput    []string
		wantNotInOutput []string
		setupGit        bool
		runTwice        bool
	}{
		{
			name:     "creates workspace structure",
			setupGit: false,
			wantFiles: []string{
				".mehrhof",             // workspace directory
				".mehrhof/config.yaml", // config file
				".mehrhof/.env",        // env template (created in .mehrhof)
			},
			wantInOutput: []string{
				"Created config file",
				"Created .env template",
				"Workspace initialized",
			},
		},
		{
			name:     "idempotent - can run twice",
			setupGit: false,
			runTwice: true,
			wantFiles: []string{
				".mehrhof",
				".mehrhof/config.yaml",
			},
			wantInOutput: []string{
				"Config file already exists",
				"Workspace initialized",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			setTestHome(t, tmpDir)

			tc := NewTestContext(t)

			// Create a git repo if requested
			if tt.setupGit {
				tc.WithGit()
			}

			// Add init command to root
			rootCmd := &cobra.Command{
				Use:   "mehr",
				Short: "Test command",
			}
			rootCmd.SetOut(tc.StdoutBuf)
			rootCmd.SetErr(tc.StderrBuf)
			rootCmd.AddCommand(initCmd)

			// Execute init command (possibly twice)
			rootCmd.SetArgs([]string{"init"})
			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("First Execute: %v", err)
			}

			if tt.runTwice {
				// Reset output and run again
				tc.ResetOutput()
				rootCmd.SetArgs([]string{"init"})
				err = rootCmd.Execute()
				if err != nil {
					t.Fatalf("Second Execute: %v", err)
				}
			}

			// Check expected files exist
			for _, file := range tt.wantFiles {
				fullPath := filepath.Join(tc.TmpDir, file)
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					t.Errorf("file not created: %s", file)
				}
			}

			// Check output
			output := tc.StdoutString()
			for _, want := range tt.wantInOutput {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nGot: %s", want, output)
				}
			}

			for _, notWant := range tt.wantNotInOutput {
				if strings.Contains(output, notWant) {
					t.Errorf("output should not contain %q\nGot: %s", notWant, output)
				}
			}
		})
	}
}

func TestInitCommand_WithGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)

	tc := NewTestContext(t)

	// Initialize git repo first
	tc.WithGit()

	rootCmd := &cobra.Command{
		Use:   "mehr",
		Short: "Test command",
	}
	rootCmd.SetOut(tc.StdoutBuf)
	rootCmd.AddCommand(initCmd)

	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Check .mehrhof was created
	tc.AssertFileExists(".mehrhof/config.yaml")

	// Check .env template was created in .mehrhof/
	tc.AssertFileExists(".mehrhof/.env")

	// Check .env has content
	content, err := os.ReadFile(filepath.Join(tc.TmpDir, ".mehrhof", ".env"))
	if err != nil {
		t.Fatalf("Read .env: %v", err)
	}
	envContent := string(content)
	if !strings.Contains(envContent, "# Mehrhof environment variables") {
		t.Errorf(".env template missing expected content")
	}
	if !strings.Contains(envContent, "ANTHROPIC_API_KEY") {
		t.Errorf(".env template missing ANTHROPIC_API_KEY example")
	}
}

func TestInitCommand_Twice(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)

	tc := NewTestContext(t)

	rootCmd := &cobra.Command{
		Use:   "mehr",
		Short: "Test command",
	}
	rootCmd.SetOut(tc.StdoutBuf)
	rootCmd.AddCommand(initCmd) // Add init command

	// First run
	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("First Execute: %v", err)
	}

	firstOutput := tc.StdoutString()

	// Reset and run again
	tc.ResetOutput()
	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Second Execute: %v", err)
	}

	secondOutput := tc.StdoutString()

	// The first run should create, the second should say already exists
	if !strings.Contains(firstOutput, "Created config file") {
		t.Errorf("First run should create config file, got: %s", firstOutput)
	}
	if !strings.Contains(secondOutput, "Config file already exists") {
		t.Errorf("Second run should say config exists, got: %s", secondOutput)
	}
}

func TestInitCommand_ASCFlag(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)

	tc := NewTestContext(t)

	rootCmd := &cobra.Command{
		Use:   "mehr",
		Short: "Test command",
	}
	rootCmd.SetOut(tc.StdoutBuf)
	rootCmd.SetErr(tc.StderrBuf)
	rootCmd.AddCommand(initCmd)

	// Run init with --asc flag
	rootCmd.SetArgs([]string{"init", "--asc"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Check output mentions ASC
	output := tc.StdoutString()
	if !strings.Contains(output, "ASC-compatible") {
		t.Errorf("output should mention ASC-compatible, got: %s", output)
	}

	// Read and verify config
	configPath := filepath.Join(tc.TmpDir, ".mehrhof", "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Read config: %v", err)
	}

	configStr := string(content)

	// Verify ASC settings are present
	// Note: YAML serializes with single quotes for strings containing special chars
	expectedSettings := []string{
		"branch_pattern: '{type}/{key}/{slug}'",
		"commit_prefix: '{type}({key}):'",
		"project_dir: tickets",
		"filename_pattern: SPEC-{n}.md",
		"filename_pattern: CODERABBIT-{n}.txt",
	}

	for _, expected := range expectedSettings {
		if !strings.Contains(configStr, expected) {
			t.Errorf("config missing ASC setting: %s\nGot:\n%s", expected, configStr)
		}
	}
}

func TestInitCommand_ASCFlagOnExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)

	tc := NewTestContext(t)

	rootCmd := &cobra.Command{
		Use:   "mehr",
		Short: "Test command",
	}
	rootCmd.SetOut(tc.StdoutBuf)
	rootCmd.SetErr(tc.StderrBuf)
	rootCmd.AddCommand(initCmd)

	// First init without --asc
	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("First Execute: %v", err)
	}

	// Reset and run with --asc on existing config
	tc.ResetOutput()
	rootCmd.SetArgs([]string{"init", "--asc"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Second Execute with --asc: %v", err)
	}

	output := tc.StdoutString()
	if !strings.Contains(output, "Config file already exists") {
		t.Errorf("should mention existing config, got: %s", output)
	}
	if !strings.Contains(output, "ASC-compatible") {
		t.Errorf("should mention ASC-compatible was applied, got: %s", output)
	}

	// Verify ASC settings were applied
	configPath := filepath.Join(tc.TmpDir, ".mehrhof", "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Read config: %v", err)
	}

	if !strings.Contains(string(content), "branch_pattern: '{type}/{key}/{slug}'") {
		t.Errorf("--asc flag should update existing config with ASC settings")
	}
}

func TestInitCommand_ASCFlagHidden(t *testing.T) {
	// Verify the --asc flag is hidden from help output
	help := initCmd.Flags().FlagUsages()
	if strings.Contains(help, "--asc") {
		t.Error("--asc flag should be hidden from help output")
	}
}

func TestCreateEnvTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	err := createEnvTemplate(envPath)
	if err != nil {
		t.Fatalf("createEnvTemplate: %v", err)
	}

	// Check a file exists
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Fatal(".env file was not created")
	}

	// Check file permissions
	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("Stat .env: %v", err)
	}
	// 0o600 = rw------- (user read/write only)
	if info.Mode().Perm() != 0o600 {
		t.Errorf(".env permissions = %o, want %o", info.Mode().Perm(), 0o600)
	}

	// Check content
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Read .env: %v", err)
	}

	contentStr := string(content)
	expectedStrings := []string{
		"# Mehrhof environment variables",
		"ANTHROPIC_API_KEY",
		"GLM_API_KEY",
		"GITHUB_TOKEN",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf(".env missing expected string: %s", expected)
		}
	}
}
