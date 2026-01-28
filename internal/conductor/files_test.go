package conductor

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNormalizeAgentPath(t *testing.T) {
	tests := []struct {
		name      string
		agentPath string
		root      string
		want      string
	}{
		{
			name:      "relative path",
			agentPath: "hello.go",
			root:      "/tmp",
			want:      "hello.go",
		},
		{
			name:      "dot-slash prefix",
			agentPath: "./hello.go",
			root:      "/tmp",
			want:      "hello.go",
		},
		{
			name:      "dot-slash with subdirectory",
			agentPath: "./subdir/hello.go",
			root:      "/tmp",
			want:      "subdir/hello.go",
		},
		{
			name:      "absolute path in root",
			agentPath: "/tmp/hello.go",
			root:      "/tmp",
			want:      "hello.go",
		},
		{
			name:      "absolute path in subdirectory",
			agentPath: "/tmp/foo/hello.go",
			root:      "/tmp",
			want:      "foo/hello.go",
		},
		{
			name:      "absolute path in nested subdirectory",
			agentPath: "/tmp/foo/bar/hello.go",
			root:      "/tmp",
			want:      "foo/bar/hello.go",
		},
		{
			name:      "absolute path outside root - returns as-is",
			agentPath: "/other/hello.go",
			root:      "/tmp",
			want:      "/other/hello.go",
		},
		{
			name:      "absolute path with dot-slash prefix",
			agentPath: "./tmp/hello.go",
			root:      "/tmp",
			want:      "tmp/hello.go",
		},
		{
			name:      "empty path",
			agentPath: "",
			root:      "/tmp",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// On Windows, convert paths to use backslashes for root
			// but keep the agentPath format consistent
			root := tt.root
			if runtime.GOOS == "windows" {
				root = filepath.FromSlash(tt.root)
				// For absolute paths in agentPath, also convert
				if strings.HasPrefix(tt.agentPath, "/") {
					tt.agentPath = filepath.FromSlash(tt.agentPath)
				}
			}

			got := normalizeAgentPath(tt.agentPath, root)
			want := tt.want
			if runtime.GOOS == "windows" {
				want = filepath.FromSlash(tt.want)
			}

			if got != want {
				t.Errorf("normalizeAgentPath(%q, %q) = %q, want %q", tt.agentPath, tt.root, got, want)
			}
		})
	}
}

func TestNormalizeAgentPathRealWorldCases(t *testing.T) {
	// Test the actual bug case: agent returns /home/user/e2e/file.md
	// when working directory is /home/user/e2e
	tests := []struct {
		name      string
		agentPath string
		root      string
		want      string
	}{
		{
			name:      "bug case: absolute path same as root",
			agentPath: "/home/daviszalitis/e2e/hello.md",
			root:      "/home/daviszalitis/e2e",
			want:      "hello.md",
		},
		{
			name:      "bug case: absolute path with dot prefix",
			agentPath: "./home/daviszalitis/e2e/hello.md",
			root:      "/home/daviszalitis/e2e",
			want:      "home/daviszalitis/e2e/hello.md",
		},
		{
			name:      "relative path should pass through",
			agentPath: "hello.md",
			root:      "/home/daviszalitis/e2e",
			want:      "hello.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAgentPath(tt.agentPath, tt.root)
			if got != tt.want {
				t.Errorf("normalizeAgentPath(%q, %q) = %q, want %q", tt.agentPath, tt.root, got, tt.want)
			}
		})
	}
}
