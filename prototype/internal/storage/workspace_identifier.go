package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/valksor/go-mehrhof/internal/vcs"
	"github.com/valksor/go-toolkit/paths"
)

// Path configuration for mehrhof.
var pathsConfig = &paths.Config{
	Vendor:   ".valksor",
	ToolName: "mehrhof",
	LocalDir: ".mehrhof",
}

const (
	// MehrhofHomeDir is the base directory in user's home for all mehrhof data.
	MehrhofHomeDir = ".valksor/mehrhof"
	// WorkspacesDir is the subdirectory for workspace data.
	WorkspacesDir = "workspaces"
)

// GenerateProjectID creates a deterministic project identifier from git remote.
// Format: "github.com-user-repo" or "local-pathhash" for local repos.
func GenerateProjectID(ctx context.Context, repoRoot string) (string, error) {
	// Try to get git remote for remote repositories
	git, err := vcs.New(ctx, repoRoot)
	if err == nil {
		// Get default remote name
		remote, err := git.GetDefaultRemote(ctx)
		if err == nil && remote != "" {
			// Get remote URL
			remoteURL, err := git.RemoteURL(ctx, remote)
			if err == nil && remoteURL != "" {
				// Convert URL to project ID
				return urlToProjectID(remoteURL), nil
			}
		}
	}

	// Fallback: use hash of repo path for local/non-git repos
	return hashPathToFallbackID(repoRoot), nil
}

// urlToProjectID converts a git remote URL to a project ID.
// Examples:
//
//	https://github.com/user/repo.git -> github.com-user-repo
//	git@github.com:user/repo.git -> github.com-user-repo
//	https://gitlab.com/group/subgroup/project.git -> gitlab.com-group-subgroup-project
func urlToProjectID(url string) string {
	// Remove protocol
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "git@")
	url = strings.TrimPrefix(url, "ssh://")

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Replace : with / (for SSH URLs like git@github.com:user/repo)
	url = strings.ReplaceAll(url, ":", "/")

	// Split by / and take host + path parts
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		// Malformed URL, fall back to hash
		hash := sha256.Sum256([]byte(url))

		return fmt.Sprintf("unknown-%x", hash)[:16]
	}

	// Join with dashes, filter empty parts
	var result []string
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}

	return strings.Join(result, "-")
}

// hashPathToFallbackID creates a project ID from a filesystem path.
// Used for local repos without git remotes.
// Format: "{dirname}-{hash6}" e.g., "my-app-a3d4f2".
func hashPathToFallbackID(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Extract directory name and sanitize for filesystem
	dirName := sanitizeForPath(filepath.Base(absPath))

	// Hash the full path for uniqueness (6 hex chars = 16.7M combinations)
	hash := sha256.Sum256([]byte(absPath))
	hashStr := hex.EncodeToString(hash[:3]) // 3 bytes = 6 hex chars

	return fmt.Sprintf("%s-%s", dirName, hashStr)
}

// sanitizeForPath makes a string safe for use in filesystem paths.
// Converts to lowercase, replaces unsafe characters with dashes, and collapses multiple dashes.
func sanitizeForPath(s string) string {
	s = strings.ToLower(s)

	// Replace unsafe characters
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		if r == ' ' || r == '.' {
			return '-'
		}

		return -1 // Remove other characters
	}, s)

	// Collapse multiple dashes
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	// Trim leading/trailing dashes
	s = strings.Trim(s, "-")

	// Fallback if empty after sanitization
	if s == "" {
		s = "workspace"
	}

	return s
}

// GetMehrhofHomeDir returns the mehrhof home directory path.
// Example: /home/user/.valksor/mehrhof.
func GetMehrhofHomeDir() (string, error) {
	return pathsConfig.GlobalDir()
}

// GetGlobalWorkspaceRoot returns the global mehrhof directory in user's home.
// Example: /home/user/.valksor/mehrhof.
func GetGlobalWorkspaceRoot() (string, error) {
	return GetMehrhofHomeDir()
}

// GetGlobalWorkspaceRootWithOverride returns the global mehrhof directory,
// using the provided override if set, otherwise using the user's home directory.
// This allows tests and custom configurations to use a different base directory.
func GetGlobalWorkspaceRootWithOverride(homeDirOverride string) (string, error) {
	if homeDirOverride != "" {
		// Set up temporary override
		restore := paths.SetHomeDirForTesting(homeDirOverride)
		defer restore()
	}

	return pathsConfig.GlobalDir()
}

// GetWorkspaceDataDir returns the workspace data directory for a specific project.
// This is where work/ and .active_task are stored.
// Example: /home/user/.valksor/mehrhof/workspaces/github.com-user-repo.
// The homeDirOverride parameter allows specifying a custom home directory (e.g., for tests).
// Pass empty string to use the default user home directory.
func GetWorkspaceDataDir(ctx context.Context, repoRoot string, homeDirOverride string) (string, error) {
	projectID, err := GenerateProjectID(ctx, repoRoot)
	if err != nil {
		return "", fmt.Errorf("generate project ID: %w", err)
	}

	globalRoot, err := GetGlobalWorkspaceRootWithOverride(homeDirOverride)
	if err != nil {
		return "", err
	}

	return filepath.Join(globalRoot, WorkspacesDir, projectID), nil
}
