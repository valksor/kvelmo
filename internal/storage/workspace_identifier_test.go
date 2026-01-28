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
		name      string
		path      string
		prefixLen int // Should be "local-" + 10 hex chars = 16 total
	}{
		{
			name:      "Simple path",
			path:      "/home/user/projects/myproject",
			prefixLen: 16, // "local-" + 10 hex chars
		},
		{
			name:      "Relative path",
			path:      "../myproject",
			prefixLen: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashPathToFallbackID(tt.path)
			assert.Equal(t, "local-", result[:6])
			assert.Len(t, result, tt.prefixLen)
			// Verify it's hex after the prefix
			for _, c := range result[6:] {
				assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
					"Expected hex character, got: %c", c)
			}
		})
	}
}
