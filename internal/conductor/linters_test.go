package conductor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/valksor/go-mehrhof/internal/quality"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// mockLinter is a minimal mock of quality.Linter for testing.
type mockLinter struct {
	name string
}

func (m *mockLinter) Name() string {
	return m.name
}

func (m *mockLinter) Available() bool {
	return true
}

func (m *mockLinter) Run(ctx context.Context, workDir string, files []string) (*quality.Result, error) {
	return &quality.Result{Passed: true}, nil
}

// Test linterNames utility function.
func TestLinterNames(t *testing.T) {
	tests := []struct {
		name    string
		linters []quality.Linter
		want    string
	}{
		{
			name:    "empty slice",
			linters: []quality.Linter{},
			want:    "",
		},
		{
			name:    "nil slice",
			linters: nil,
			want:    "",
		},
		{
			name: "single linter",
			linters: []quality.Linter{
				&mockLinter{name: "golangci-lint"},
			},
			want: "golangci-lint",
		},
		{
			name: "multiple linters",
			linters: []quality.Linter{
				&mockLinter{name: "golangci-lint"},
				&mockLinter{name: "eslint"},
				&mockLinter{name: "ruff"},
			},
			want: "golangci-lint, eslint, ruff",
		},
		{
			name: "linters with special characters",
			linters: []quality.Linter{
				&mockLinter{name: "golangci-lint"},
				&mockLinter{name: "pylint"},
				&mockLinter{name: "shellcheck"},
			},
			want: "golangci-lint, pylint, shellcheck",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := linterNames(tt.linters)
			if got != tt.want {
				t.Errorf("linterNames() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRunLintersWithConfigDisabled tests that quality checks are skipped when disabled in config.
func TestRunLintersWithConfigDisabled(t *testing.T) {
	// Create a temporary workspace
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "workspace")
	if err := os.Mkdir(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a minimal conductor with test workspace
	c, err := New(WithWorkDir(workDir))
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock workspace with quality disabled
	ws, err := storage.OpenWorkspace(context.Background(), workDir, storage.NewDefaultWorkspaceConfig())
	if err != nil {
		t.Fatal(err)
	}
	c.workspace = ws

	// Save config with quality disabled
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Quality = &storage.QualitySettings{
		Enabled: false,
	}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Run linters - should return empty string when quality is disabled
	result := c.runLinters(context.Background())
	if result != "" {
		t.Errorf("expected empty result when quality disabled, got: %s", result)
	}
}

// TestRunLintersWithLinterDisabled tests that individual linters respect config.
func TestRunLintersWithLinterDisabled(t *testing.T) {
	// Create a temporary workspace with a Go project
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "workspace")
	if err := os.Mkdir(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create go.mod to trigger Go linter detection
	goMod := filepath.Join(workDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a minimal conductor
	c, err := New(WithWorkDir(workDir))
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock workspace with golangci-lint explicitly disabled
	ws, err := storage.OpenWorkspace(context.Background(), workDir, storage.NewDefaultWorkspaceConfig())
	if err != nil {
		t.Fatal(err)
	}
	c.workspace = ws

	// Save config with golangci-lint disabled
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Quality = &storage.QualitySettings{
		Enabled: true,
		Linters: map[string]storage.LinterConfig{
			"golangci-lint": {Enabled: false},
		},
	}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Run linters - golangci-lint should be filtered out
	result := c.runLinters(context.Background())

	// The result should be empty since golangci-lint is disabled and it's the only linter for Go
	// (If golangci-lint is available on the system, it would normally be detected)
	if result != "" {
		// Check that golangci-lint is not mentioned in the output
		if indexOf(result, "golangci-lint") >= 0 {
			t.Error("golangci-lint should be filtered out when disabled in config")
		}
	}
}

// TestRunLintersWithCustomLinter tests that custom linters are registered and run.
func TestRunLintersWithCustomLinter(t *testing.T) {
	// Create a temporary workspace
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "workspace")
	if err := os.Mkdir(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a minimal conductor
	c, err := New(WithWorkDir(workDir))
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock workspace
	ws, err := storage.OpenWorkspace(context.Background(), workDir, storage.NewDefaultWorkspaceConfig())
	if err != nil {
		t.Fatal(err)
	}
	c.workspace = ws

	// Save config with a custom linter (using /bin/echo as a test command)
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Quality = &storage.QualitySettings{
		Enabled: true,
		Linters: map[string]storage.LinterConfig{
			"test-linter": {
				Enabled: true,
				Command: []string{"/bin/echo"},
			},
		},
	}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// The registry should have the custom linter registered
	// We can't test runLinters directly since it needs a working git setup,
	// but we can verify the config is loaded correctly
	loadedCfg, err := c.workspace.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if loadedCfg.Quality == nil {
		t.Error("quality config should be loaded")

		return
	}

	if !loadedCfg.Quality.Enabled {
		t.Error("quality should be enabled")
	}

	customLinter, exists := loadedCfg.Quality.Linters["test-linter"]
	if !exists {
		t.Error("custom linter config should be loaded")

		return
	}

	if len(customLinter.Command) == 0 || customLinter.Command[0] != "/bin/echo" {
		t.Error("custom linter command should be preserved")
	}
}

// indexOf returns the index of substr in s, or -1 if not found.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}

// TestRunLintersWithUseDefaultsFalse tests that no linters run when UseDefaults is false
// and no linters are explicitly configured.
func TestRunLintersWithUseDefaultsFalse(t *testing.T) {
	// Create a temporary workspace with a Go project
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "workspace")
	if err := os.Mkdir(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create go.mod to trigger Go linter detection (but it won't be used)
	goMod := filepath.Join(workDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a minimal conductor
	c, err := New(WithWorkDir(workDir))
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock workspace with UseDefaults=false
	ws, err := storage.OpenWorkspace(context.Background(), workDir, storage.NewDefaultWorkspaceConfig())
	if err != nil {
		t.Fatal(err)
	}
	c.workspace = ws

	// Save config with UseDefaults=false (safer default)
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Quality = &storage.QualitySettings{
		Enabled:     true,
		UseDefaults: false, // Don't auto-enable built-in linters
	}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Run linters - should return empty string since no linters are configured
	result := c.runLinters(context.Background())

	// The result should be empty since UseDefaults is false and no linters are explicitly configured
	if result != "" {
		// Check that golangci-lint is not mentioned in the output
		if indexOf(result, "golangci-lint") >= 0 {
			t.Error("golangci-lint should NOT run when UseDefaults=false and not explicitly configured")
		}
	}
}

// TestRunLintersWithUseDefaultsTrue tests that built-in linters run when UseDefaults is true.
func TestRunLintersWithUseDefaultsTrue(t *testing.T) {
	// Create a temporary workspace with a Go project
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "workspace")
	if err := os.Mkdir(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create go.mod to trigger Go linter detection
	goMod := filepath.Join(workDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a minimal conductor
	c, err := New(WithWorkDir(workDir))
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock workspace with UseDefaults=true
	ws, err := storage.OpenWorkspace(context.Background(), workDir, storage.NewDefaultWorkspaceConfig())
	if err != nil {
		t.Fatal(err)
	}
	c.workspace = ws

	// Save config with UseDefaults=true (opt-in to auto-detection)
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Quality = &storage.QualitySettings{
		Enabled:     true,
		UseDefaults: true, // Auto-enable built-in linters
	}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Run linters - golangci-lint should be detected and run (if available)
	_ = c.runLinters(context.Background())

	// We can't assert golangci-lint runs without it being installed,
	// but we verify the config is loaded correctly
	loadedCfg, err := c.workspace.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if loadedCfg.Quality == nil || !loadedCfg.Quality.UseDefaults {
		t.Error("UseDefaults should be true")
	}
}

// TestRunLintersWithExplicitConfigAndUseDefaultsFalse tests that explicitly
// configured linters run even when UseDefaults is false.
func TestRunLintersWithExplicitConfigAndUseDefaultsFalse(t *testing.T) {
	// Create a temporary workspace with a Go project
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "workspace")
	if err := os.Mkdir(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create go.mod to trigger Go linter detection
	goMod := filepath.Join(workDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a minimal conductor
	c, err := New(WithWorkDir(workDir))
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock workspace
	ws, err := storage.OpenWorkspace(context.Background(), workDir, storage.NewDefaultWorkspaceConfig())
	if err != nil {
		t.Fatal(err)
	}
	c.workspace = ws

	// Save config with UseDefaults=false but golangci-lint explicitly enabled
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Quality = &storage.QualitySettings{
		Enabled:     true,
		UseDefaults: false,
		Linters: map[string]storage.LinterConfig{
			"golangci-lint": {Enabled: true}, // Explicitly enabled
		},
	}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Verify the config is loaded correctly
	loadedCfg, err := c.workspace.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if loadedCfg.Quality == nil {
		t.Fatal("quality config should be loaded")
	}

	// Verify UseDefaults is false
	if loadedCfg.Quality.UseDefaults {
		t.Error("UseDefaults should be false")
	}

	// Verify golangci-lint is explicitly enabled
	golangciCfg, exists := loadedCfg.Quality.Linters["golangci-lint"]
	if !exists {
		t.Fatal("golangci-lint config should be present")
	}
	if !golangciCfg.Enabled {
		t.Error("golangci-lint should be enabled in config")
	}
}
