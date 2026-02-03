package library

import (
	"net/url"
	"path/filepath"
	"strings"
)

// MatchesPath checks if any of the patterns match the given file path.
// Patterns use glob syntax (e.g., "ide/vscode/**", "*.md").
func MatchesPath(patterns []string, filePath string) bool {
	if len(patterns) == 0 {
		return false
	}

	// Normalize path separators
	filePath = filepath.ToSlash(filePath)

	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)
		if matchGlob(pattern, filePath) {
			return true
		}
	}

	return false
}

// matchGlob matches a file path against a glob pattern.
// Supports ** for recursive matching and * for single-level matching.
func matchGlob(pattern, path string) bool {
	// Handle ** patterns
	if strings.Contains(pattern, "**") {
		return matchDoublestar(pattern, path)
	}

	// Use filepath.Match for simple patterns
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false
	}

	return matched
}

// matchDoublestar handles ** glob patterns.
func matchDoublestar(pattern, path string) bool {
	// Split pattern at **
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		// Multiple ** not supported, fall back to prefix/suffix match
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "**"))
	}

	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	// Path must start with prefix
	if prefix != "" && !strings.HasPrefix(path, prefix) {
		// Check if prefix matches (allowing for path separator)
		if !strings.HasPrefix(path, prefix+"/") && path != prefix {
			return false
		}
	}

	// If no suffix, just need prefix match
	if suffix == "" {
		return true
	}

	// Path must end with suffix (or match suffix pattern)
	remainder := path
	if prefix != "" {
		remainder = strings.TrimPrefix(path, prefix)
		remainder = strings.TrimPrefix(remainder, "/")
	}

	// Check if any part of remainder matches suffix
	if strings.HasSuffix(remainder, suffix) {
		return true
	}

	// Try matching suffix as a pattern
	matched, _ := filepath.Match(suffix, filepath.Base(path))

	return matched
}

// MatchesAnyPath checks if any of the file paths match the collection's patterns.
func MatchesAnyPath(patterns []string, filePaths []string) bool {
	for _, fp := range filePaths {
		if MatchesPath(patterns, fp) {
			return true
		}
	}

	return false
}

// SuggestPaths attempts to suggest path patterns from a source URL or path.
// This uses heuristics based on common documentation site patterns.
// Returns empty slice if no confident suggestion can be made.
func SuggestPaths(source string, projectDirs []string) []string {
	// Try URL-based suggestions
	if u, err := url.Parse(source); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return suggestPathsFromURL(u, projectDirs)
	}

	// Git URL suggestions
	if strings.HasPrefix(source, "git@") || strings.Contains(source, ".git") {
		return suggestPathsFromGitURL(source, projectDirs)
	}

	// Local path suggestions
	return suggestPathsFromLocalPath(source, projectDirs)
}

// suggestPathsFromURL suggests patterns based on URL.
func suggestPathsFromURL(u *url.URL, projectDirs []string) []string {
	host := strings.ToLower(u.Host)

	// Known documentation sites with common project mappings
	mappings := map[string][]string{
		"code.visualstudio.com": {"ide/vscode/**", "vscode/**"},
		"react.dev":             {"react/**", "src/**/*.tsx", "src/**/*.jsx"},
		"go.dev":                {"**/*.go"},
		"docs.python.org":       {"**/*.py"},
		"nodejs.org":            {"**/*.js", "**/*.ts"},
		"docs.rs":               {"**/*.rs"},
		"developer.mozilla.org": {"**/*.html", "**/*.css", "**/*.js"},
		"plugins.jetbrains.com": {"ide/jetbrains/**"},
		"www.jetbrains.com":     {"ide/jetbrains/**"},
	}

	// Check for known host
	for knownHost, patterns := range mappings {
		if strings.Contains(host, knownHost) || host == knownHost {
			return filterExistingDirs(patterns, projectDirs)
		}
	}

	// Try to extract meaningful path component
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for _, part := range pathParts {
		part = strings.ToLower(part)
		if part == "docs" || part == "api" || part == "reference" || part == "guide" {
			continue
		}
		// Look for matching directory
		if containsDir(projectDirs, part) {
			return []string{part + "/**"}
		}
	}

	return nil
}

// suggestPathsFromGitURL suggests patterns from git URL.
func suggestPathsFromGitURL(source string, projectDirs []string) []string {
	// Extract repo name
	source = strings.TrimSuffix(source, ".git")

	// Handle git@host:user/repo format
	if strings.HasPrefix(source, "git@") {
		parts := strings.Split(source, "/")
		if len(parts) > 0 {
			repoName := parts[len(parts)-1]
			if containsDir(projectDirs, repoName) {
				return []string{repoName + "/**"}
			}
		}
	}

	// Handle https://host/user/repo format
	if u, err := url.Parse(source); err == nil {
		pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(pathParts) > 0 {
			repoName := pathParts[len(pathParts)-1]
			if containsDir(projectDirs, repoName) {
				return []string{repoName + "/**"}
			}
		}
	}

	return nil
}

// suggestPathsFromLocalPath suggests patterns from local file path.
func suggestPathsFromLocalPath(source string, projectDirs []string) []string {
	// Use the directory name as pattern
	source = filepath.Clean(source)
	base := filepath.Base(source)

	if base != "." && base != "" {
		// Check if this directory exists in project
		if containsDir(projectDirs, base) {
			return []string{base + "/**"}
		}
	}

	// Use parent directory if source is a file
	dir := filepath.Dir(source)
	if dir != "." && dir != "" {
		base = filepath.Base(dir)
		if containsDir(projectDirs, base) {
			return []string{base + "/**"}
		}
	}

	return nil
}

// filterExistingDirs filters patterns to only those matching existing project directories.
func filterExistingDirs(patterns []string, projectDirs []string) []string {
	if len(projectDirs) == 0 {
		return patterns
	}

	var filtered []string
	for _, pattern := range patterns {
		// Extract directory from pattern
		dir := strings.Split(pattern, "/")[0]
		dir = strings.TrimSuffix(dir, "**")

		if containsDir(projectDirs, dir) {
			filtered = append(filtered, pattern)
		}
	}

	if len(filtered) == 0 {
		return patterns // Return original if no matches
	}

	return filtered
}

// containsDir checks if a directory name exists in the project directories list.
func containsDir(dirs []string, name string) bool {
	name = strings.ToLower(name)
	for _, d := range dirs {
		d = strings.ToLower(filepath.Base(d))
		if d == name {
			return true
		}
	}

	return false
}

// ExtractKeywords extracts keywords from file paths for content matching.
func ExtractKeywords(filePaths []string) []string {
	seen := make(map[string]bool)
	var keywords []string

	for _, fp := range filePaths {
		// Normalize and split path
		fp = filepath.ToSlash(fp)
		parts := strings.Split(fp, "/")

		for _, part := range parts {
			// Skip common non-meaningful parts
			part = strings.ToLower(part)
			if isCommonPathPart(part) {
				continue
			}

			// Remove extension
			if ext := filepath.Ext(part); ext != "" {
				part = strings.TrimSuffix(part, ext)
			}

			// Skip if too short or already seen
			if len(part) < 3 || seen[part] {
				continue
			}

			seen[part] = true
			keywords = append(keywords, part)
		}
	}

	return keywords
}

// isCommonPathPart returns true for common path parts that aren't meaningful keywords.
func isCommonPathPart(s string) bool {
	common := map[string]bool{
		"src": true, "lib": true, "pkg": true, "cmd": true,
		"internal": true, "test": true, "tests": true,
		"spec": true, "specs": true, "dist": true, "build": true,
		"node_modules": true, "vendor": true, ".": true, "..": true,
	}

	return common[s]
}
