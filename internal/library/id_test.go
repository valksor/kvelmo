package library

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"VS Code Extension API", "vs-code-extension-api"},
		{"React Docs", "react-docs"},
		{"hello--world", "hello-world"},
		{"  spaces  ", "spaces"},
		{"UPPERCASE", "uppercase"},
		{"with.dots.and-hyphens", "with-dots-and-hyphens"},
		{"special!@#$%chars", "special-chars"},
		{"", "collection"},
		{"a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestShortHash(t *testing.T) {
	// Should return 6 characters
	h := shortHash("test")
	if len(h) != 6 {
		t.Errorf("shortHash length = %d, want 6", len(h))
	}

	// Should be deterministic
	h2 := shortHash("test")
	if h != h2 {
		t.Errorf("shortHash not deterministic: %q != %q", h, h2)
	}

	// Different inputs should give different hashes
	h3 := shortHash("other")
	if h == h3 {
		t.Errorf("different inputs produced same hash: %q", h)
	}
}

func TestGenerateCollectionID(t *testing.T) {
	tests := []struct {
		name     string
		nameArg  string
		source   string
		wantLen  bool // Just check it's reasonable length
		contains string
	}{
		{
			name:     "with name",
			nameArg:  "VS Code API",
			source:   "https://code.visualstudio.com/api",
			contains: "vs-code-api",
		},
		{
			name:     "URL only",
			nameArg:  "",
			source:   "https://react.dev/reference",
			contains: "react",
		},
		{
			name:     "git URL",
			nameArg:  "",
			source:   "git@github.com:user/repo.git",
			contains: "repo",
		},
		{
			name:     "local path",
			nameArg:  "",
			source:   "/home/user/docs/project",
			contains: "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateCollectionID(tt.nameArg, tt.source)
			if got == "" {
				t.Error("GenerateCollectionID returned empty string")
			}
			if len(got) > 60 {
				t.Errorf("ID too long: %d chars", len(got))
			}
			if tt.contains != "" {
				// The contains check is case-insensitive since slugify lowercases
				if !containsIgnoreCase(got, tt.contains) {
					t.Errorf("ID %q should contain %q", got, tt.contains)
				}
			}
		})
	}
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstr(s, substr))))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

func TestDeriveIDFromURL(t *testing.T) {
	tests := []struct {
		url      string
		contains string
	}{
		{"https://code.visualstudio.com/api", "code-visualstudio"},
		{"https://react.dev/reference/hooks", "react-dev"},
		{"https://go.dev/doc/effective_go", "go-dev"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := deriveIDFromSource(tt.url)
			if !containsIgnoreCase(got, tt.contains) {
				t.Errorf("deriveIDFromSource(%q) = %q, want to contain %q", tt.url, got, tt.contains)
			}
		})
	}
}

func TestIsValidID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"valid-id", true},
		{"also-valid-123", true},
		{"a", true},
		{"", false},
		{"-starts-with-hyphen", false},
		{"ends-with-hyphen-", false},
		{"has spaces", false},
		{"HAS_UPPER", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := IsValidID(tt.id)
			if got != tt.valid {
				t.Errorf("IsValidID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}
