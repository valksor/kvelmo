package library

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// nonAlphanumeric matches characters that are not letters, numbers, or hyphens.
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9-]+`)
	// multipleHyphens matches consecutive hyphens.
	multipleHyphens = regexp.MustCompile(`-+`)
)

// GenerateCollectionID creates a unique, URL-safe identifier for a collection.
// If name is provided, it slugifies the name and adds a short hash suffix for uniqueness.
// If name is empty, it derives an ID from the source.
func GenerateCollectionID(name, source string) string {
	if name != "" {
		slug := slugify(name)
		// Add short hash of source for uniqueness when same name used for different sources
		hash := shortHash(source)

		return slug + "-" + hash
	}

	// Derive from source
	return deriveIDFromSource(source)
}

// slugify converts a string to a URL-safe slug.
func slugify(s string) string {
	// Lowercase
	s = strings.ToLower(s)

	// Replace non-alphanumeric with hyphens
	s = nonAlphanumeric.ReplaceAllString(s, "-")

	// Collapse multiple hyphens
	s = multipleHyphens.ReplaceAllString(s, "-")

	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")

	// Limit length
	if len(s) > 50 {
		s = s[:50]
		// Don't end with a hyphen
		s = strings.TrimRight(s, "-")
	}

	if s == "" {
		return "collection"
	}

	return s
}

// shortHash returns a 6-character hash of the input string.
func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))

	return hex.EncodeToString(h[:])[:6]
}

// deriveIDFromSource creates an ID from the source URL or path.
func deriveIDFromSource(source string) string {
	// Try to parse as URL
	if u, err := url.Parse(source); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return deriveIDFromURL(u)
	}

	// Git URL patterns
	if strings.HasPrefix(source, "git@") || strings.Contains(source, ".git") {
		return deriveIDFromGitURL(source)
	}

	// Treat as file path
	return deriveIDFromPath(source)
}

// deriveIDFromURL creates an ID from an HTTP(S) URL.
func deriveIDFromURL(u *url.URL) string {
	// Use host + meaningful path segments
	parts := []string{u.Host}

	// Add path segments, excluding common prefixes
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for _, p := range pathParts {
		if p != "" && p != "docs" && p != "documentation" && p != "api" {
			parts = append(parts, p)
			if len(parts) >= 3 {
				break
			}
		}
	}

	combined := strings.Join(parts, "-")

	return slugify(combined)
}

// deriveIDFromGitURL creates an ID from a git URL.
func deriveIDFromGitURL(source string) string {
	// Handle git@github.com:user/repo.git format
	if strings.HasPrefix(source, "git@") {
		source = strings.TrimPrefix(source, "git@")
		source = strings.Replace(source, ":", "/", 1)
	}

	// Remove .git suffix
	source = strings.TrimSuffix(source, ".git")

	// Parse remaining as URL or path
	if u, err := url.Parse("https://" + source); err == nil {
		return deriveIDFromURL(u)
	}

	return slugify(filepath.Base(source))
}

// deriveIDFromPath creates an ID from a file path.
func deriveIDFromPath(source string) string {
	// Use the last 1-2 meaningful path components
	source = filepath.Clean(source)
	parts := strings.Split(source, string(filepath.Separator))

	var meaningful []string
	for i := len(parts) - 1; i >= 0 && len(meaningful) < 2; i-- {
		p := parts[i]
		if p != "" && p != "." && p != ".." {
			meaningful = append([]string{p}, meaningful...)
		}
	}

	if len(meaningful) == 0 {
		return "local-docs"
	}

	combined := strings.Join(meaningful, "-")
	slug := slugify(combined)

	// Add hash for uniqueness (paths can have same basename)
	hash := shortHash(source)

	return slug + "-" + hash
}

// IsValidID checks if a string is a valid collection ID.
func IsValidID(id string) bool {
	if id == "" || len(id) > 60 {
		return false
	}
	// Must be lowercase alphanumeric with hyphens, not starting/ending with hyphen
	if strings.HasPrefix(id, "-") || strings.HasSuffix(id, "-") {
		return false
	}
	// After removing hyphens, should only contain lowercase alphanumeric
	withoutHyphens := strings.ReplaceAll(id, "-", "")
	cleaned := nonAlphanumeric.ReplaceAllString(withoutHyphens, "")

	return cleaned == withoutHyphens
}
