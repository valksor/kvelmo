package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUrlToProjectID(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTPS URL",
			url:      "https://github.com/user/repo.git",
			expected: "github.com-user-repo",
		},
		{
			name:     "SSH URL",
			url:      "git@github.com:user/repo.git",
			expected: "github.com-user-repo",
		},
		{
			name:     "Nested groups",
			url:      "https://gitlab.com/group/subgroup/project.git",
			expected: "gitlab.com-group-subgroup-project",
		},
		{
			name:     "Without .git suffix",
			url:      "https://github.com/user/repo",
			expected: "github.com-user-repo",
		},
		{
			name:     "Deeply nested",
			url:      "https://gitlab.com/group/subgroup/subsubgroup/project.git",
			expected: "gitlab.com-group-subgroup-subsubgroup-project",
		},
		{
			name:     "HTTPS URL with token userinfo",
			url:      "https://ghp_secret123@github.com/user/repo.git",
			expected: "github.com-user-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := urlToProjectID(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashPathToFallbackID(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedPrefix string // Directory name (sanitized) before the hash
	}{
		{
			name:           "Simple path",
			path:           "/home/user/projects/myproject",
			expectedPrefix: "myproject-",
		},
		{
			name:           "Path with spaces",
			path:           "/home/user/My Project",
			expectedPrefix: "my-project-",
		},
		{
			name:           "Path with dots",
			path:           "/home/user/project.v2",
			expectedPrefix: "project-v2-",
		},
		{
			name:           "Hidden directory",
			path:           "/home/user/.hidden",
			expectedPrefix: "hidden-",
		},
		{
			name:           "Special characters",
			path:           "/home/user/project@2.0!test",
			expectedPrefix: "project2-0test-", // @ and ! removed, . becomes -
		},
		{
			name:           "Uppercase",
			path:           "/home/user/MyProject",
			expectedPrefix: "myproject-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashPathToFallbackID(tt.path)

			// Should start with sanitized directory name
			assert.True(t, len(result) > len(tt.expectedPrefix),
				"Result %q should be longer than prefix %q", result, tt.expectedPrefix)
			assert.Equal(t, tt.expectedPrefix, result[:len(tt.expectedPrefix)],
				"Result should start with %q, got %q", tt.expectedPrefix, result)

			// Extract hash suffix (after the prefix)
			hashSuffix := result[len(tt.expectedPrefix):]
			assert.Len(t, hashSuffix, 6, "Hash suffix should be 6 hex chars")

			// Verify suffix is hex
			for _, c := range hashSuffix {
				assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
					"Expected hex character, got: %c", c)
			}
		})
	}
}

func TestSanitizeForPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple lowercase",
			input:    "myproject",
			expected: "myproject",
		},
		{
			name:     "Uppercase to lowercase",
			input:    "MyProject",
			expected: "myproject",
		},
		{
			name:     "Spaces to dashes",
			input:    "my project",
			expected: "my-project",
		},
		{
			name:     "Dots to dashes",
			input:    "my.project.v2",
			expected: "my-project-v2",
		},
		{
			name:     "Special chars removed",
			input:    "project@2.0!test#123",
			expected: "project2-0test123", // @ ! # removed, . becomes -
		},
		{
			name:     "Multiple dashes collapsed",
			input:    "my---project",
			expected: "my-project",
		},
		{
			name:     "Leading/trailing dashes trimmed",
			input:    "-my-project-",
			expected: "my-project",
		},
		{
			name:     "Hidden directory (leading dot)",
			input:    ".hidden",
			expected: "hidden",
		},
		{
			name:     "Empty after sanitization",
			input:    "...",
			expected: "workspace",
		},
		{
			name:     "Underscores preserved",
			input:    "my_project_v2",
			expected: "my_project_v2",
		},
		{
			name:     "Numbers preserved",
			input:    "project123",
			expected: "project123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeForPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
