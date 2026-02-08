package storage

import (
	"net/url"
	"strings"
)

// SanitizeRemoteURL removes embedded credentials from git remote URLs.
// Example: https://token@github.com/org/repo.git -> https://github.com/org/repo.git
func SanitizeRemoteURL(remoteURL string) string {
	if remoteURL == "" {
		return ""
	}

	// Standard URLs (https://, ssh://, etc.)
	if strings.Contains(remoteURL, "://") {
		parsed, err := url.Parse(remoteURL)
		if err != nil {
			return remoteURL
		}

		if parsed.User != nil {
			parsed.User = nil
		}

		return parsed.String()
	}

	// SCP-like or schemeless URLs that may include userinfo:
	// user@host:path/repo.git or user@host/path/repo.git
	at := strings.Index(remoteURL, "@")
	if at <= 0 {
		return remoteURL
	}

	userPart := remoteURL[:at]
	hostPart := remoteURL[at+1:]

	// Keep the common SSH form git@host:path unchanged.
	if strings.EqualFold(userPart, "git") {
		return remoteURL
	}

	if hostPart == "" {
		return remoteURL
	}

	// Strip credentials - hostPart is valid whether it's a URL path
	// (contains : or /) or a project ID (uses dashes)
	return hostPart
}
