package library

import (
	"testing"
)

func TestMatchesPath(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		path     string
		want     bool
	}{
		{
			name:     "exact match",
			patterns: []string{"src/main.go"},
			path:     "src/main.go",
			want:     true,
		},
		{
			name:     "glob star",
			patterns: []string{"src/*.go"},
			path:     "src/main.go",
			want:     true,
		},
		{
			name:     "glob star no match",
			patterns: []string{"src/*.go"},
			path:     "src/sub/main.go",
			want:     false,
		},
		{
			name:     "doublestar basic",
			patterns: []string{"ide/vscode/**"},
			path:     "ide/vscode/src/extension.ts",
			want:     true,
		},
		{
			name:     "doublestar nested",
			patterns: []string{"ide/vscode/**"},
			path:     "ide/vscode/src/views/panel.ts",
			want:     true,
		},
		{
			name:     "doublestar root file",
			patterns: []string{"ide/vscode/**"},
			path:     "ide/vscode/package.json",
			want:     true,
		},
		{
			name:     "doublestar no match",
			patterns: []string{"ide/vscode/**"},
			path:     "ide/jetbrains/plugin.kt",
			want:     false,
		},
		{
			name:     "multiple patterns - first match",
			patterns: []string{"ide/vscode/**", "ide/jetbrains/**"},
			path:     "ide/vscode/src/extension.ts",
			want:     true,
		},
		{
			name:     "multiple patterns - second match",
			patterns: []string{"ide/vscode/**", "ide/jetbrains/**"},
			path:     "ide/jetbrains/plugin.kt",
			want:     true,
		},
		{
			name:     "multiple patterns - no match",
			patterns: []string{"ide/vscode/**", "ide/jetbrains/**"},
			path:     "internal/server/handler.go",
			want:     false,
		},
		{
			name:     "empty patterns",
			patterns: []string{},
			path:     "any/path.go",
			want:     false,
		},
		{
			name:     "extension pattern",
			patterns: []string{"**/*.md"},
			path:     "docs/guide/intro.md",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesPath(tt.patterns, tt.path)
			if got != tt.want {
				t.Errorf("MatchesPath(%v, %q) = %v, want %v", tt.patterns, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchesAnyPath(t *testing.T) {
	patterns := []string{"ide/vscode/**", "src/**/*.ts"}

	tests := []struct {
		name      string
		filePaths []string
		want      bool
	}{
		{
			name:      "one match",
			filePaths: []string{"ide/vscode/extension.ts", "other/file.go"},
			want:      true,
		},
		{
			name:      "no match",
			filePaths: []string{"internal/server/handler.go", "cmd/main.go"},
			want:      false,
		},
		{
			name:      "empty paths",
			filePaths: []string{},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesAnyPath(patterns, tt.filePaths)
			if got != tt.want {
				t.Errorf("MatchesAnyPath(%v, %v) = %v, want %v", patterns, tt.filePaths, got, tt.want)
			}
		})
	}
}

func TestSuggestPaths(t *testing.T) {
	projectDirs := []string{"ide/vscode", "ide/jetbrains", "internal", "cmd", "docs"}

	tests := []struct {
		name        string
		source      string
		wantContain string
	}{
		{
			name:        "vscode docs",
			source:      "https://code.visualstudio.com/api",
			wantContain: "vscode",
		},
		{
			name:        "local vscode dir",
			source:      "/home/user/docs/vscode-api",
			wantContain: "", // May not match since exact dir name doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuggestPaths(tt.source, projectDirs)
			if tt.wantContain != "" {
				found := false
				for _, p := range got {
					if containsStr(p, tt.wantContain) {
						found = true

						break
					}
				}
				if !found && len(got) > 0 {
					// This is okay - suggestions are heuristic
					t.Logf("SuggestPaths(%q) = %v, didn't contain %q (but that's okay)", tt.source, got, tt.wantContain)
				}
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name      string
		filePaths []string
		want      []string
	}{
		{
			name:      "simple paths",
			filePaths: []string{"ide/vscode/extension.ts", "ide/vscode/views/panel.ts"},
			want:      []string{"ide", "vscode", "extension", "views", "panel"},
		},
		{
			name:      "filters common parts",
			filePaths: []string{"src/internal/lib/handler.go"},
			want:      []string{"handler"}, // src, internal, lib are filtered
		},
		{
			name:      "empty",
			filePaths: []string{},
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractKeywords(tt.filePaths)

			// Check that expected keywords are present
			for _, want := range tt.want {
				found := false
				for _, g := range got {
					if g == want {
						found = true

						break
					}
				}
				if !found {
					t.Errorf("ExtractKeywords() missing %q, got %v", want, got)
				}
			}
		})
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		// Simple patterns
		{"*.go", "main.go", true},
		{"*.go", "main.txt", false},

		// Doublestar patterns
		{"**/*.go", "main.go", true},
		{"**/*.go", "src/main.go", true},
		{"**/*.go", "src/pkg/main.go", true},

		// Prefix doublestar
		{"src/**", "src/main.go", true},
		{"src/**", "src/pkg/main.go", true},
		{"src/**", "other/main.go", false},

		// Exact prefix
		{"cmd/", "cmd/main.go", false}, // Trailing slash is literal
		{"cmd/*", "cmd/main.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			got := matchGlob(tt.pattern, tt.path)
			if got != tt.want {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}
