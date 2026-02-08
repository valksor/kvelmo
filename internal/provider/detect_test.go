package provider

import "testing"

// TestDetectProviderFromURL tests provider detection from URLs.
func TestDetectProviderFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		provider string
	}{
		// GitHub
		{"GitHub HTTPS", "https://github.com/owner/repo", "github"},
		{"GitHub SSH", "git@github.com:owner/repo.git", "github"},
		{"GitHub PR URL", "https://github.com/owner/repo/pull/123", "github"},
		{"GitHub with git://", "git://github.com/owner/repo.git", "github"},

		// GitLab
		{"GitLab HTTPS", "https://gitlab.com/owner/repo", "gitlab"},
		{"GitLab SSH", "git@gitlab.com:owner/repo.git", "gitlab"},
		{"GitLab MR URL", "https://gitlab.com/owner/repo/merge_requests/123", "gitlab"},

		// Bitbucket
		{"Bitbucket HTTPS", "https://bitbucket.org/owner/repo", "bitbucket"},
		{"Bitbucket SSH", "git@bitbucket.org:owner/repo.git", "bitbucket"},
		{"Bitbucket PR URL", "https://bitbucket.org/owner/repo/pull-requests/123", "bitbucket"},

		// Azure DevOps
		{"Azure DevOps", "https://dev.azure.com/org/project/_git/repo", "azuredevops"},
		{"Azure DevOps PR", "https://dev.azure.com/org/project/_git/repo/pullrequest/123", "azuredevops"},
		{"Visual Studio", "https://visualstudio.com/org/project/_git/repo", "azuredevops"},
		{"Azure.com", "https://azure.com/org/project/_git/repo", "azuredevops"},

		// Unknown providers
		{"Unknown provider", "https://unknown.com/repo", ""},
		{"Empty URL", "", ""},
		{"Local path", "/path/to/local/repo", ""},
		{"File URL", "file:///path/to/repo", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectProviderFromURL(tt.url)
			if result != tt.provider {
				t.Errorf("DetectProviderFromURL(%q) = %q, want %q", tt.url, result, tt.provider)
			}
		})
	}
}

// TestDetectProviderFromURLCaseInsensitive tests that detection is case-sensitive (by design).
// URLs are expected to be lowercase; this documents that behavior.
func TestDetectProviderFromURLCaseSensitive(t *testing.T) {
	tests := []struct {
		url      string
		provider string
	}{
		// Uppercase domains should NOT match (by design - URLs are expected to be lowercase)
		{"https://GitHub.com/owner/repo", ""},               // Won't match
		{"https://GITLAB.com/owner/repo", ""},               // Won't match
		{"https://BITBUCKET.ORG/owner/repo", ""},            // Won't match
		{"https://DEV.AZURE.COM/org/project/_git/repo", ""}, // Won't match
		// Mixed case within lowercase domains should match
		{"https://github.com/Owner/Repo", "github"}, // Path case doesn't matter
		{"https://gitlab.com/Owner/Repo", "gitlab"}, // Path case doesn't matter
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := DetectProviderFromURL(tt.url)
			if result != tt.provider {
				t.Errorf("DetectProviderFromURL(%q) = %q, want %q", tt.url, result, tt.provider)
			}
		})
	}
}

// TestDetectProviderFromURLSubdomain tests that subdomains don't affect detection.
func TestDetectProviderFromURLSubdomain(t *testing.T) {
	tests := []struct {
		url      string
		provider string
	}{
		// GitHub Enterprise (should still detect as github for simplicity)
		{"https://github.example.com/owner/repo", ""},
		{"https://api.github.com/owner/repo", "github"},
		{"https://gist.github.com/owner/repo", "github"},

		// GitLab self-hosted (should still detect as gitlab for simplicity)
		{"https://gitlab.example.com/owner/repo", ""},
		{"https://api.gitlab.com/owner/repo", "gitlab"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := DetectProviderFromURL(tt.url)
			if result != tt.provider {
				t.Errorf("DetectProviderFromURL(%q) = %q, want %q", tt.url, result, tt.provider)
			}
		})
	}
}

// TestParseOwnerRepoFromURL tests owner/repo extraction from git URLs.
func TestParseOwnerRepoFromURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		// HTTPS format
		{
			name:          "github https",
			url:           "https://github.com/owner/repo.git",
			expectedOwner: "owner",
			expectedRepo:  "repo",
		},
		{
			name:          "github https no .git",
			url:           "https://github.com/owner/repo",
			expectedOwner: "owner",
			expectedRepo:  "repo",
		},
		{
			name:          "github https with port",
			url:           "https://github.com:443/owner/repo.git",
			expectedOwner: "owner",
			expectedRepo:  "repo",
		},
		{
			name:          "gitlab https",
			url:           "https://gitlab.com/group/project.git",
			expectedOwner: "group",
			expectedRepo:  "project",
		},
		{
			name:          "gitlab https subgroup",
			url:           "https://gitlab.com/group/subgroup/project.git",
			expectedOwner: "group/subgroup",
			expectedRepo:  "project",
		},
		{
			name:          "gitlab https nested subgroups",
			url:           "https://gitlab.com/org/team/subsystem/project.git",
			expectedOwner: "org/team/subsystem",
			expectedRepo:  "project",
		},

		// SSH format
		{
			name:          "github ssh",
			url:           "git@github.com:owner/repo.git",
			expectedOwner: "owner",
			expectedRepo:  "repo",
		},
		{
			name:          "github ssh no .git",
			url:           "git@github.com:owner/repo",
			expectedOwner: "owner",
			expectedRepo:  "repo",
		},
		{
			name:          "gitlab ssh",
			url:           "git@gitlab.com:group/project.git",
			expectedOwner: "group",
			expectedRepo:  "project",
		},
		{
			name:          "gitlab ssh subgroup",
			url:           "git@gitlab.com:group/subgroup/project.git",
			expectedOwner: "group/subgroup",
			expectedRepo:  "project",
		},

		// Edge cases
		{
			name:          "bitbucket https",
			url:           "https://bitbucket.org/workspace/repo.git",
			expectedOwner: "workspace",
			expectedRepo:  "repo",
		},
		{
			name:          "trailing slash stripped",
			url:           "https://github.com/owner/repo/",
			expectedOwner: "owner",
			expectedRepo:  "repo",
		},
		{
			name:          "whitespace trimmed",
			url:           "  https://github.com/owner/repo.git  ",
			expectedOwner: "owner",
			expectedRepo:  "repo",
		},

		// Error cases
		{
			name:        "empty url",
			url:         "",
			expectError: true,
		},
		{
			name:        "whitespace only",
			url:         "   ",
			expectError: true,
		},
		{
			name:        "no path",
			url:         "https://github.com",
			expectError: true,
		},
		{
			name:        "only owner",
			url:         "https://github.com/owner",
			expectError: true,
		},
		{
			name:        "ssh no colon",
			url:         "git@github.com/owner/repo.git",
			expectError: true,
		},
		{
			name:        "ssh empty path",
			url:         "git@github.com:",
			expectError: true,
		},
		{
			name:        "malformed ssh",
			url:         "git@github.com:.git",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseOwnerRepoFromURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseOwnerRepoFromURL(%q) expected error, got owner=%q repo=%q", tt.url, owner, repo)
				}

				return
			}

			if err != nil {
				t.Errorf("ParseOwnerRepoFromURL(%q) unexpected error: %v", tt.url, err)

				return
			}

			if owner != tt.expectedOwner {
				t.Errorf("ParseOwnerRepoFromURL(%q) owner = %q, want %q", tt.url, owner, tt.expectedOwner)
			}

			if repo != tt.expectedRepo {
				t.Errorf("ParseOwnerRepoFromURL(%q) repo = %q, want %q", tt.url, repo, tt.expectedRepo)
			}
		})
	}
}
