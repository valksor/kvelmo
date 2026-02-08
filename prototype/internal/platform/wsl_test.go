package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsWSL(t *testing.T) {
	tests := []struct {
		name            string
		wslDistroName   string
		procVersionData string
		procVersionErr  bool
		want            bool
	}{
		{
			name:          "WSL_DISTRO_NAME set",
			wslDistroName: "Ubuntu",
			want:          true,
		},
		{
			name:            "WSL_DISTRO_NAME unset, /proc/version contains Microsoft",
			wslDistroName:   "",
			procVersionData: "Linux version 5.15.90.1-microsoft-standard-WSL2",
			want:            true,
		},
		{
			name:            "WSL_DISTRO_NAME unset, /proc/version contains microsoft lowercase",
			wslDistroName:   "",
			procVersionData: "linux version 5.15.90.1-microsoft-standard-wsl2",
			want:            true,
		},
		{
			name:            "WSL_DISTRO_NAME unset, /proc/version contains WSL",
			wslDistroName:   "",
			procVersionData: "Linux version 5.15.90.1-custom-WSL2-kernel",
			want:            true,
		},
		{
			name:            "WSL_DISTRO_NAME unset, normal Linux /proc/version",
			wslDistroName:   "",
			procVersionData: "Linux version 6.1.0-generic (build@host) (gcc)",
			want:            false,
		},
		{
			name:           "WSL_DISTRO_NAME unset, /proc/version unreadable",
			wslDistroName:  "",
			procVersionErr: true,
			want:           false,
		},
		{
			name:            "Empty WSL_DISTRO_NAME, check /proc/version",
			wslDistroName:   "",
			procVersionData: "Linux version 5.15.90.1-microsoft-standard-WSL2",
			want:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset detection state
			ResetWSLDetection()

			// Set up environment using t.Setenv (auto-restores after test)
			// First clear any existing value
			if os.Getenv("WSL_DISTRO_NAME") != "" {
				t.Setenv("WSL_DISTRO_NAME", "")
			}

			if tt.wslDistroName != "" {
				t.Setenv("WSL_DISTRO_NAME", tt.wslDistroName)
			}

			// Set up /proc/version mock
			originalPath := procVersionPath
			t.Cleanup(func() { procVersionPath = originalPath })

			if tt.procVersionErr {
				// Point to non-existent file
				procVersionPath = "/nonexistent/proc/version"
			} else if tt.procVersionData != "" {
				// Create temp file with test data
				tmpDir := t.TempDir()
				tmpFile := filepath.Join(tmpDir, "version")
				if err := os.WriteFile(tmpFile, []byte(tt.procVersionData), 0o644); err != nil {
					t.Fatalf("failed to write temp file: %v", err)
				}
				procVersionPath = tmpFile
			}

			got := IsWSL()
			if got != tt.want {
				t.Errorf("IsWSL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWSLCaching(t *testing.T) {
	// Reset detection state
	ResetWSLDetection()

	// Set up environment for WSL detection using t.Setenv
	t.Setenv("WSL_DISTRO_NAME", "TestDistro")

	// First call should detect WSL
	result1 := IsWSL()
	if !result1 {
		t.Error("First IsWSL() call should return true")
	}

	// Modify environment - note: t.Setenv to empty string effectively "unsets"
	// but the cached value should still return true
	t.Setenv("WSL_DISTRO_NAME", "")

	// Second call should still return cached result (true)
	result2 := IsWSL()
	if !result2 {
		t.Error("Second IsWSL() call should return cached true value")
	}

	// Reset and verify it now returns false
	ResetWSLDetection()
	originalPath := procVersionPath
	procVersionPath = "/nonexistent/path"
	t.Cleanup(func() { procVersionPath = originalPath })

	result3 := IsWSL()
	if result3 {
		t.Error("After reset, IsWSL() should return false")
	}
}
