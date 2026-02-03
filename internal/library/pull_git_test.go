package library

import (
	"testing"
)

func TestNormalizeGitURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"git@github.com:user/repo.git", "https://github.com/user/repo"},
		{"https://github.com/user/repo.git", "https://github.com/user/repo"},
		{"https://github.com/user/repo", "https://github.com/user/repo"},
		{"git@gitlab.com:org/project.git", "https://gitlab.com/org/project"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeGitURL(tt.input)
			if got != tt.want {
				t.Errorf("normalizeGitURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectSourceType(t *testing.T) {
	tests := []struct {
		source string
		want   SourceType
	}{
		// URLs
		{"https://react.dev/reference", SourceURL},
		{"https://go.dev/doc/effective_go", SourceURL},
		{"http://example.com/docs", SourceURL},

		// Git repos
		{"git@github.com:user/repo.git", SourceGit},
		{"https://github.com/user/repo.git", SourceGit},
		{"https://github.com/user/repo", SourceGit},
		{"https://gitlab.com/org/project", SourceGit},

		// Git but docs URL (should be URL, not git)
		{"https://github.com/user/repo/docs/", SourceURL},

		// Files
		{"/home/user/docs", SourceFile},
		{"./docs", SourceFile},
		{"../project/docs", SourceFile},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := DetectSourceType(tt.source)
			if got != tt.want {
				t.Errorf("DetectSourceType(%q) = %q, want %q", tt.source, got, tt.want)
			}
		})
	}
}

func TestIsGitHostURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://github.com/user/repo", true},
		{"https://gitlab.com/org/project", true},
		{"https://bitbucket.org/user/repo", true},
		{"https://example.com/repo", false},
		{"https://react.dev/docs", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := isGitHostURL(tt.url)
			if got != tt.want {
				t.Errorf("isGitHostURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestIsDocsURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://github.com/user/repo/docs/intro", true},
		{"https://docs.example.com/guide", true},
		{"https://developer.mozilla.org/en-US/docs/", true},
		{"https://example.com/wiki/Main_Page", true},
		{"https://github.com/user/repo", false},
		{"https://example.com/src/main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := isDocsURL(tt.url)
			if got != tt.want {
				t.Errorf("isDocsURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
