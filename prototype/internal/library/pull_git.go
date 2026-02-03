package library

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PullGit clones a git repository and extracts documentation files.
func PullGit(ctx context.Context, repoURL, ref, subpath string, maxPageSize int64) ([]*CrawledPage, error) {
	// Create temp directory for clone
	tmpDir, err := os.MkdirTemp("", "mehr-library-git-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Clone repository (shallow for efficiency)
	if err := gitClone(ctx, repoURL, ref, tmpDir); err != nil {
		return nil, fmt.Errorf("git clone: %w", err)
	}

	// Determine source path
	sourcePath := tmpDir
	if subpath != "" {
		sourcePath = filepath.Join(tmpDir, subpath)
		// Verify path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("subpath %q not found in repository", subpath)
		}
	}

	// Pull files from cloned directory
	pages, err := PullFile(sourcePath, maxPageSize)
	if err != nil {
		return nil, fmt.Errorf("pull files: %w", err)
	}

	// Update source URL on pages to reference the git repo
	for _, p := range pages {
		if subpath != "" {
			p.URL = fmt.Sprintf("%s/blob/%s/%s/%s", normalizeGitURL(repoURL), ref, subpath, p.Path)
		} else {
			p.URL = fmt.Sprintf("%s/blob/%s/%s", normalizeGitURL(repoURL), ref, p.Path)
		}
	}

	return pages, nil
}

// gitClone performs a shallow git clone.
func gitClone(ctx context.Context, repoURL, ref, destDir string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	args := []string{"clone", "--depth=1"}

	// Add branch/tag if specified
	if ref != "" {
		args = append(args, "--branch", ref)
	}

	args = append(args, repoURL, destDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = nil // Suppress output
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		// Check if it's a context timeout
		if ctx.Err() != nil {
			return errors.New("clone timed out after 5 minutes")
		}

		return err
	}

	return nil
}

// normalizeGitURL converts git URLs to browsable HTTPS URLs.
func normalizeGitURL(repoURL string) string {
	// Handle git@host:user/repo.git format
	if strings.HasPrefix(repoURL, "git@") {
		repoURL = strings.TrimPrefix(repoURL, "git@")
		repoURL = strings.Replace(repoURL, ":", "/", 1)
		repoURL = "https://" + repoURL
	}

	// Remove .git suffix
	repoURL = strings.TrimSuffix(repoURL, ".git")

	return repoURL
}

// DetectSourceType determines the source type from a string.
func DetectSourceType(source string) SourceType {
	// HTTP(S) URLs
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// Check if it's a git URL (github, gitlab, etc.)
		if isGitHostURL(source) && !isDocsURL(source) {
			return SourceGit
		}

		return SourceURL
	}

	// Git SSH URLs
	if strings.HasPrefix(source, "git@") {
		return SourceGit
	}

	// Local paths with .git
	if strings.HasSuffix(source, ".git") {
		return SourceGit
	}

	// Default to file
	return SourceFile
}

// isGitHostURL checks if a URL is from a known git hosting service.
func isGitHostURL(urlStr string) bool {
	hosts := []string{
		"github.com",
		"gitlab.com",
		"bitbucket.org",
		"codeberg.org",
		"gitea.com",
		"sr.ht",
	}

	for _, host := range hosts {
		if strings.Contains(urlStr, host) {
			return true
		}
	}

	return false
}

// isDocsURL checks if a URL appears to be a documentation site rather than a repo.
func isDocsURL(urlStr string) bool {
	docsPatterns := []string{
		"/docs/",
		"/documentation/",
		"/wiki/",
		"docs.",
		"developer.",
		"api.",
		"help.",
		"manual.",
		"guide.",
	}

	urlLower := strings.ToLower(urlStr)
	for _, pattern := range docsPatterns {
		if strings.Contains(urlLower, pattern) {
			return true
		}
	}

	return false
}
