package security

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNewToolManager(t *testing.T) {
	tests := []struct {
		name         string
		toolsDir     string
		autoDownload bool
		wantErr      bool
	}{
		{
			name:         "default directory",
			toolsDir:     "",
			autoDownload: true,
			wantErr:      false,
		},
		{
			name:         "custom directory",
			toolsDir:     "/tmp/test-tools",
			autoDownload: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, err := NewToolManager(tt.toolsDir, tt.autoDownload)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewToolManager() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tm == nil {
				t.Fatal("NewToolManager() returned nil ToolManager")
			}

			if tm.autoDownload != tt.autoDownload {
				t.Errorf("autoDownload = %v, want %v", tm.autoDownload, tt.autoDownload)
			}

			if tm.toolsDir == "" {
				t.Error("toolsDir should not be empty")
			}
		})
	}
}

func TestToolManager_GetCachedTool(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tm, err := NewToolManager(tmpDir, false)
	if err != nil {
		t.Fatalf("NewToolManager() error = %v", err)
	}

	spec := ToolSpec{
		Name:       "test-tool",
		BinaryName: "test-tool",
	}

	// Test with empty cache
	path, err := tm.getCachedTool(spec)
	if err == nil {
		t.Error("getCachedTool() should return error when tool not in cache")
	}

	if path != "" {
		t.Errorf("getCachedTool() path = %v, want empty", path)
	}

	// Test with cached tool
	versionDir := filepath.Join(tmpDir, "test-tool-1.0.0-"+runtime.GOOS+"-"+runtime.GOARCH)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	binaryPath := filepath.Join(versionDir, "test-tool")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	if err := os.WriteFile(binaryPath, []byte("#! test binary"), 0o755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	path, err = tm.getCachedTool(spec)
	if err != nil {
		t.Errorf("getCachedTool() error = %v, want nil", err)
	}

	if path != binaryPath {
		t.Errorf("getCachedTool() path = %v, want %v", path, binaryPath)
	}
}

func TestToolManager_Warnings(t *testing.T) {
	tm, err := NewToolManager("", false)
	if err != nil {
		t.Fatalf("NewToolManager() error = %v", err)
	}

	// Initially no warnings
	if tm.HasWarnings() {
		t.Error("HasWarnings() = true, want false")
	}

	// Add a warning
	tm.addWarning("test warning")

	if !tm.HasWarnings() {
		t.Error("HasWarnings() = false, want true")
	}

	warnings := tm.GetWarnings()
	if len(warnings) != 1 {
		t.Errorf("GetWarnings() length = %d, want 1", len(warnings))
	}

	if warnings[0] != "test warning" {
		t.Errorf("GetWarnings()[0] = %v, want 'test warning'", warnings[0])
	}

	// Clear warnings
	tm.ClearWarnings()

	if tm.HasWarnings() {
		t.Error("HasWarnings() = true after ClearWarnings(), want false")
	}
}

func TestToolManager_EnsureTool(t *testing.T) {
	t.Run("tool in PATH is used", func(t *testing.T) {
		// This test assumes 'go' is in PATH
		tm, err := NewToolManager("", false)
		if err != nil {
			t.Fatalf("NewToolManager() error = %v", err)
		}

		spec := ToolSpec{
			Name:       "go",
			Repository: "golang/go",
			BinaryName: "go",
		}

		ctx := context.Background()
		path, err := tm.EnsureTool(ctx, spec)
		if err != nil {
			t.Errorf("EnsureTool() error = %v, want nil (go should be in PATH)", err)
		}

		if path == "" {
			t.Error("EnsureTool() returned empty path when go should be in PATH")
		}
	})

	t.Run("auto-download disabled returns error for missing tool", func(t *testing.T) {
		tm, err := NewToolManager("", false)
		if err != nil {
			t.Fatalf("NewToolManager() error = %v", err)
		}

		spec := ToolSpec{
			Name:       "nonexistent-tool-xyz",
			Repository: "example/nonexistent",
			BinaryName: "nonexistent-tool-xyz",
		}

		ctx := context.Background()
		_, err = tm.EnsureTool(ctx, spec)

		if err == nil {
			t.Error("EnsureTool() error = nil, want error (tool doesn't exist and auto-download disabled)")
		}
	})

	t.Run("cached tool is used", func(t *testing.T) {
		tmpDir := t.TempDir()
		tm, err := NewToolManager(tmpDir, false)
		if err != nil {
			t.Fatalf("NewToolManager() error = %v", err)
		}

		// Create a fake cached tool
		versionDir := filepath.Join(tmpDir, "fake-tool-1.0.0-"+runtime.GOOS+"-"+runtime.GOARCH)
		if err := os.MkdirAll(versionDir, 0o755); err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		binaryPath := filepath.Join(versionDir, "fake-tool")
		if runtime.GOOS == "windows" {
			binaryPath += ".exe"
		}

		if err := os.WriteFile(binaryPath, []byte("#! fake binary"), 0o755); err != nil {
			t.Fatalf("Failed to create test binary: %v", err)
		}

		spec := ToolSpec{
			Name:       "fake-tool",
			BinaryName: "fake-tool",
		}

		ctx := context.Background()
		path, err := tm.EnsureTool(ctx, spec)
		if err != nil {
			t.Errorf("EnsureTool() error = %v, want nil", err)
		}

		if path != binaryPath {
			t.Errorf("EnsureTool() path = %v, want %v", path, binaryPath)
		}
	})
}

func TestGetInstallCommand(t *testing.T) {
	tests := []struct {
		name     string
		spec     ToolSpec
		expected string
	}{
		{
			name:     "gitleaks",
			spec:     ToolSpec{Name: "gitleaks"},
			expected: "github.com/zricethezav/gitleaks/v8/cmd/gitleaks@latest",
		},
		{
			name:     "gosec",
			spec:     ToolSpec{Name: "gosec"},
			expected: "github.com/securego/gosec/v2/cmd/gosec@latest",
		},
		{
			name:     "govulncheck",
			spec:     ToolSpec{Name: "govulncheck"},
			expected: "golang.org/x/vuln/cmd/govulncheck@latest",
		},
		{
			name:     "unknown tool",
			spec:     ToolSpec{Name: "unknown", Repository: "example/unknown"},
			expected: "example/unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstallCommand(tt.spec)
			if result != tt.expected {
				t.Errorf("getInstallCommand() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSkippedResult(t *testing.T) {
	duration := 100 * time.Millisecond
	result := skippedResult("test-scanner", duration)

	if result.Scanner != "test-scanner" {
		t.Errorf("skippedResult() Scanner = %v, want 'test-scanner'", result.Scanner)
	}

	if result.Status != ScanStatusSkipped {
		t.Errorf("skippedResult() Status = %v, want ScanStatusSkipped", result.Status)
	}

	if result.Duration != duration {
		t.Errorf("skippedResult() Duration = %v, want %v", result.Duration, duration)
	}

	if len(result.Findings) != 0 {
		t.Errorf("skippedResult() Findings length = %v, want 0", len(result.Findings))
	}

	if result.Summary.Total != 0 {
		t.Errorf("skippedResult() Summary.Total = %v, want 0", result.Summary.Total)
	}
}

func TestToolManager_Concurrency(t *testing.T) {
	tm, err := NewToolManager("", true)
	if err != nil {
		t.Fatalf("NewToolManager() error = %v", err)
	}

	// Test concurrent warning access
	done := make(chan bool)
	for range 10 {
		go func() {
			tm.addWarning("concurrent warning")
			tm.HasWarnings()
			tm.GetWarnings()
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	// Verify no race conditions
	warnings := tm.GetWarnings()
	if len(warnings) != 10 {
		t.Errorf("Expected 10 warnings, got %d", len(warnings))
	}
}
