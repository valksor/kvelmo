package provider

import (
	"errors"
	"net/url"
	"path"
	"strings"
)

// ErrInvalidRemoteURL is returned when the git remote URL cannot be parsed.
var ErrInvalidRemoteURL = errors.New("invalid git remote URL")

// DetectProviderFromURL parses a URL to determine the provider.
// Returns the provider name (github, gitlab, bitbucket, azuredevops) or empty string if unknown.
func DetectProviderFromURL(rawURL string) string {
	switch {
	case strings.Contains(rawURL, "github.com"):
		return "github"
	case strings.Contains(rawURL, "gitlab.com"):
		return "gitlab"
	case strings.Contains(rawURL, "bitbucket.org"):
		return "bitbucket"
	case strings.Contains(rawURL, "dev.azure.com"), strings.Contains(rawURL, "azure.com"), strings.Contains(rawURL, "visualstudio.com"):
		return "azuredevops"
	default:
		return ""
	}
}

// ParseOwnerRepoFromURL extracts owner and repo from a git remote URL.
// Handles both HTTPS and SSH formats:
//   - https://github.com/owner/repo.git → ("owner", "repo", nil)
//   - git@github.com:owner/repo.git → ("owner", "repo", nil)
//   - https://gitlab.com/group/subgroup/project.git → ("group/subgroup", "project", nil)
//   - https://github.com:443/owner/repo.git → ("owner", "repo", nil)
//
// Strips .git suffix if present. Returns error if owner or repo is empty.
func ParseOwnerRepoFromURL(rawURL string) (string, string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", "", ErrInvalidRemoteURL
	}

	// Handle SSH format: git@github.com:owner/repo.git
	if strings.HasPrefix(rawURL, "git@") {
		return parseSSHURL(rawURL)
	}

	// Handle HTTPS format: https://github.com/owner/repo.git
	return parseHTTPSURL(rawURL)
}

// parseSSHURL handles SSH-style git URLs: git@github.com:owner/repo.git.
func parseSSHURL(rawURL string) (string, string, error) {
	// Format: git@host:path
	colonIdx := strings.Index(rawURL, ":")
	if colonIdx == -1 {
		return "", "", ErrInvalidRemoteURL
	}

	pathPart := rawURL[colonIdx+1:]
	pathPart = strings.TrimSuffix(pathPart, ".git")

	return splitOwnerRepo(pathPart)
}

// parseHTTPSURL handles HTTPS-style git URLs: https://github.com/owner/repo.git
func parseHTTPSURL(rawURL string) (string, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", ErrInvalidRemoteURL
	}

	pathPart := strings.TrimPrefix(u.Path, "/")
	pathPart = strings.TrimSuffix(pathPart, ".git")

	return splitOwnerRepo(pathPart)
}

// splitOwnerRepo splits a path like "owner/repo" or "group/subgroup/project" into owner and repo.
// For nested paths, all segments except the last become the owner.
func splitOwnerRepo(pathPart string) (string, string, error) {
	pathPart = path.Clean(pathPart)
	if pathPart == "" || pathPart == "." {
		return "", "", ErrInvalidRemoteURL
	}

	// Split into segments
	segments := strings.Split(pathPart, "/")
	if len(segments) < 2 {
		return "", "", ErrInvalidRemoteURL
	}

	// Last segment is repo, everything else is owner
	repo := segments[len(segments)-1]
	owner := strings.Join(segments[:len(segments)-1], "/")

	if owner == "" || repo == "" {
		return "", "", ErrInvalidRemoteURL
	}

	return owner, repo, nil
}
