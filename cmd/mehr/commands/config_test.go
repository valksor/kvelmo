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

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/paths"
)

// Note: TestConfigCommand_Structure is in common_test.go

func TestConfigCommand_LongDescription(t *testing.T) {
	if configCmd.Long == "" {
		t.Error("Long description is empty")
	}
}

func TestConfigCommand_ShortDescription(t *testing.T) {
	expected := "Manage configuration"
	if configCmd.Short != expected {
		t.Errorf("Short = %q, want %q", configCmd.Short, expected)
	}
}

func TestConfigValidateCommand_Properties(t *testing.T) {
	// Check validate subcommand is properly configured
	if configValidateCmd.Use != "validate" {
		t.Errorf("Use = %q, want %q", configValidateCmd.Use, "validate")
	}

	if configValidateCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if configValidateCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if configValidateCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestConfigValidateCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{
			name:         "strict flag",
			flagName:     "strict",
			defaultValue: "false",
		},
		{
			name:         "format flag",
			flagName:     "format",
			defaultValue: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := configValidateCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			// Check default value
			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestConfigValidateCommand_ShortDescription(t *testing.T) {
	expected := "Validate configuration files"
	if configValidateCmd.Short != expected {
		t.Errorf("Short = %q, want %q", configValidateCmd.Short, expected)
	}
}

func TestConfigValidateCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"Validate workspace configuration",
		"config.yaml",
		"YAML syntax",
		"circular dependencies",
	}

	for _, substr := range contains {
		if !containsString(configValidateCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestConfigValidateCommand_ExamplesContains(t *testing.T) {
	examples := []string{
		"mehr config validate",
		"--strict",
		"--format json",
	}

	for _, example := range examples {
		if !containsString(configValidateCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestConfigCommand_RegisteredInRoot(t *testing.T) {
	// Verify configCmd is a subcommand of rootCmd
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "config" {
			found = true

			break
		}
	}
	if !found {
		t.Error("config command not registered in root command")
	}
}

func TestConfigValidateCommand_RegisteredInConfig(t *testing.T) {
	// Verify configValidateCmd is a subcommand of configCmd
	found := false
	for _, cmd := range configCmd.Commands() {
		if cmd.Use == "validate" {
			found = true

			break
		}
	}
	if !found {
		t.Error("validate subcommand not registered in config command")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Unit tests for config.go utility functions
// ─────────────────────────────────────────────────────────────────────────────

// TestDetectProjectType tests the detectProjectType function.
func TestDetectProjectType(t *testing.T) {
	tests := []struct {
		name         string
		setupFile    string
		fileContent  string
		expectedType string
	}{
		{
			name:         "go project - go.mod",
			setupFile:    "go.mod",
			fileContent:  "module example.com\n\ngo 1.21\n",
			expectedType: "go",
		},
		{
			name:         "node project - package.json",
			setupFile:    "package.json",
			fileContent:  `{"name": "test", "version": "1.0.0"}`,
			expectedType: "node",
		},
		{
			name:         "python project - pyproject.toml",
			setupFile:    "pyproject.toml",
			fileContent:  `[project]\nname = "test"`,
			expectedType: "python",
		},
		{
			name:         "python project - requirements.txt",
			setupFile:    "requirements.txt",
			fileContent:  `requests==2.28.0`,
			expectedType: "python",
		},
		{
			name:         "python project - setup.py",
			setupFile:    "setup.py",
			fileContent:  `from setuptools import setup\nsetup()`,
			expectedType: "python",
		},
		{
			name:         "python project - setup.cfg",
			setupFile:    "setup.cfg",
			fileContent:  `[metadata]\nname = test`,
			expectedType: "python",
		},
		{
			name:         "php project - composer.json",
			setupFile:    "composer.json",
			fileContent:  `{"name": "test/project"}`,
			expectedType: "php",
		},
		{
			name:         "no project markers",
			setupFile:    "",
			fileContent:  "",
			expectedType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.setupFile != "" {
				fullPath := filepath.Join(tmpDir, tt.setupFile)
				if err := os.WriteFile(fullPath, []byte(tt.fileContent), 0o644); err != nil {
					t.Fatalf("write file: %v", err)
				}
			}

			result := detectProjectType(tmpDir)
			if result != tt.expectedType {
				t.Errorf("detectProjectType() = %q, want %q", result, tt.expectedType)
			}
		})
	}
}

// TestApplyProjectCustomizations tests the applyProjectCustomizations function.
func TestApplyProjectCustomizations(t *testing.T) {
	tests := []struct {
		name        string
		projectType string
		checkConfig func(t *testing.T, cfg *storage.WorkspaceConfig)
	}{
		{
			name:        "go project",
			projectType: "go",
			checkConfig: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Agent.Default != "claude" {
					t.Errorf("Agent.Default = %q, want 'claude'", cfg.Agent.Default)
				}
				if cfg.Git.CommitPrefix != "[{key}]" {
					t.Errorf("Git.CommitPrefix = %q, want '[{key}]'", cfg.Git.CommitPrefix)
				}
				if cfg.Git.BranchPattern != "{type}/{key}--{slug}" {
					t.Errorf("Git.BranchPattern = %q, want '{type}/{key}--{slug}'", cfg.Git.BranchPattern)
				}
			},
		},
		{
			name:        "node project",
			projectType: "node",
			checkConfig: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Agent.Default != "claude" {
					t.Errorf("Agent.Default = %q, want 'claude'", cfg.Agent.Default)
				}
				if cfg.Git.CommitPrefix != "feat({key}):" {
					t.Errorf("Git.CommitPrefix = %q, want 'feat({key}):'", cfg.Git.CommitPrefix)
				}
			},
		},
		{
			name:        "python project",
			projectType: "python",
			checkConfig: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Agent.Default != "claude" {
					t.Errorf("Agent.Default = %q, want 'claude'", cfg.Agent.Default)
				}
				if cfg.Git.CommitPrefix != "[{key}]" {
					t.Errorf("Git.CommitPrefix = %q, want '[{key}]'", cfg.Git.CommitPrefix)
				}
			},
		},
		{
			name:        "php project",
			projectType: "php",
			checkConfig: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				if cfg.Agent.Default != "claude" {
					t.Errorf("Agent.Default = %q, want 'claude'", cfg.Agent.Default)
				}
				if cfg.Git.CommitPrefix != "[{key}]" {
					t.Errorf("Git.CommitPrefix = %q, want '[{key}]'", cfg.Git.CommitPrefix)
				}
			},
		},
		{
			name:        "unknown project type",
			projectType: "unknown",
			checkConfig: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				// Unknown types should not modify config
				// Just verify it doesn't crash
				if cfg == nil {
					t.Error("config should not be nil")
				}
			},
		},
		{
			name:        "empty project type",
			projectType: "",
			checkConfig: func(t *testing.T, cfg *storage.WorkspaceConfig) {
				t.Helper()
				// Empty type should not modify config
				if cfg == nil {
					t.Error("config should not be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := storage.NewDefaultWorkspaceConfig()
			applyProjectCustomizations(cfg, tt.projectType)
			tt.checkConfig(t, cfg)
		})
	}
}

func TestRunConfigInit_NewWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	origForce := configInitForce
	origProject := configInitProject

	defer func() {
		configInitForce = origForce
		configInitProject = origProject
	}()

	configInitForce = false
	configInitProject = ""

	t.Chdir(tmpDir)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runConfigInit(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runConfigInit() error = %v", err)
	}

	if !strings.Contains(output, "Creating new configuration") {
		t.Errorf("output missing 'Creating new configuration'\nGot:\n%s", output)
	}

	if !strings.Contains(output, "Configuration created successfully") {
		t.Errorf("output missing 'Configuration created successfully'\nGot:\n%s", output)
	}

	// Verify the config file was created
	configPath := filepath.Join(tmpDir, ".mehrhof", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}

func TestRunConfigInit_ExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	origForce := configInitForce
	origProject := configInitProject

	defer func() {
		configInitForce = origForce
		configInitProject = origProject
	}()

	configInitForce = false
	configInitProject = ""

	// Create workspace with existing config
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir

	ws, err := storage.OpenWorkspace(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("OpenWorkspace: %v", err)
	}

	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	t.Chdir(tmpDir)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runConfigInit(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if runErr != nil {
		t.Fatalf("runConfigInit() error = %v", runErr)
	}

	if !strings.Contains(output, "WARNING") {
		t.Errorf("output missing 'WARNING'\nGot:\n%s", output)
	}

	if !strings.Contains(output, "already exists") {
		t.Errorf("output missing 'already exists'\nGot:\n%s", output)
	}
}

func TestRunConfigInit_WithProjectDetection(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	origForce := configInitForce
	origProject := configInitProject

	defer func() {
		configInitForce = origForce
		configInitProject = origProject
	}()

	configInitForce = false
	configInitProject = ""

	// Create a go.mod so detection picks up "go"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("WriteFile go.mod: %v", err)
	}

	t.Chdir(tmpDir)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runConfigInit(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runConfigInit() error = %v", err)
	}

	if !strings.Contains(output, "Detected project type:") {
		t.Errorf("output missing 'Detected project type:'\nGot:\n%s", output)
	}
}
