package conductor

import (
	"context"
	"fmt"
	"time"

	"github.com/valksor/go-mehrhof/internal/library"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// LibrarySystem holds the library documentation system components.
type LibrarySystem struct {
	manager *library.Manager
	config  *storage.LibrarySettings
}

// InitializeLibrary initializes the library system from workspace config.
func (c *Conductor) InitializeLibrary(ctx context.Context) error {
	cfg, err := c.workspace.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Library is always available, config just customizes behavior
	libConfig := library.DefaultConfig()

	// Apply user config overrides if present
	if cfg != nil && cfg.Library != nil {
		if cfg.Library.AutoIncludeMax > 0 {
			libConfig.AutoIncludeMax = cfg.Library.AutoIncludeMax
		}
		if cfg.Library.MaxPagesPerPrompt > 0 {
			libConfig.MaxPagesPerPrompt = cfg.Library.MaxPagesPerPrompt
		}
		if cfg.Library.MaxCrawlPages > 0 {
			libConfig.MaxCrawlPages = cfg.Library.MaxCrawlPages
		}
		if cfg.Library.MaxCrawlDepth > 0 {
			libConfig.MaxCrawlDepth = cfg.Library.MaxCrawlDepth
		}
		if cfg.Library.MaxPageSizeBytes > 0 {
			libConfig.MaxPageSizeBytes = cfg.Library.MaxPageSizeBytes
		}
		if cfg.Library.LockTimeout != "" {
			if d, err := time.ParseDuration(cfg.Library.LockTimeout); err == nil {
				libConfig.LockTimeout = d
			}
		}
		if cfg.Library.MaxTokenBudget > 0 {
			libConfig.MaxTokenBudget = cfg.Library.MaxTokenBudget
		}

		// Crawl filtering settings
		if cfg.Library.DomainScope != "" {
			libConfig.DomainScope = cfg.Library.DomainScope
		}
		libConfig.VersionFilter = cfg.Library.VersionFilter
		if cfg.Library.VersionPath != "" {
			libConfig.VersionPath = cfg.Library.VersionPath
		}
	}

	// Create manager using workspace
	manager, err := library.NewManagerFromWorkspace(ctx, c.workspace)
	if err != nil {
		return fmt.Errorf("create library manager: %w", err)
	}

	// Store library settings for reference
	var libSettings *storage.LibrarySettings
	if cfg != nil {
		libSettings = cfg.Library
	}

	c.library = &LibrarySystem{
		manager: manager,
		config:  libSettings,
	}

	return nil
}

// GetLibrary returns the library manager.
func (c *Conductor) GetLibrary() *library.Manager {
	if c.library == nil {
		return nil
	}

	return c.library.manager
}

// GetLibraryError returns an actionable error message if library failed to initialize.
// Returns nil if library is available or was never configured.
func (c *Conductor) GetLibraryError() error {
	if c.library != nil {
		return nil // Library is working
	}
	if c.libraryInitErr != nil {
		return fmt.Errorf("library failed to initialize: %w. Check permissions on ~/.valksor/mehrhof/library/", c.libraryInitErr)
	}
	// Library was never configured (no error, just not enabled)
	return nil
}

// GetLibraryContextForPaths returns library documentation relevant to the given file paths.
// This is used for auto-include functionality.
func (c *Conductor) GetLibraryContextForPaths(ctx context.Context, filePaths []string) (string, error) {
	if c.library == nil {
		return "", nil
	}

	maxTokens := 0 // Use config default
	if c.library.config != nil && c.library.config.MaxTokenBudget > 0 {
		maxTokens = c.library.config.MaxTokenBudget
	}

	docs, err := c.library.manager.GetDocsForPaths(ctx, filePaths, maxTokens)
	if err != nil {
		return "", fmt.Errorf("get docs for paths: %w", err)
	}

	return library.FormatDocsForPrompt(docs), nil
}

// GetLibraryContextExplicit returns library documentation for explicitly named collections.
func (c *Conductor) GetLibraryContextExplicit(ctx context.Context, collectionNames []string) (string, error) {
	if c.library == nil {
		return "", nil
	}

	maxTokens := 0 // Use config default
	if c.library.config != nil && c.library.config.MaxTokenBudget > 0 {
		maxTokens = c.library.config.MaxTokenBudget
	}

	docs, err := c.library.manager.GetExplicitDocs(ctx, collectionNames, maxTokens)
	if err != nil {
		return "", fmt.Errorf("get explicit docs: %w", err)
	}

	return library.FormatDocsForPrompt(docs), nil
}

// GetLibraryContextForQuery returns library documentation matching a search query.
func (c *Conductor) GetLibraryContextForQuery(ctx context.Context, query string) (string, error) {
	if c.library == nil {
		return "", nil
	}

	maxTokens := 0 // Use config default
	if c.library.config != nil && c.library.config.MaxTokenBudget > 0 {
		maxTokens = c.library.config.MaxTokenBudget
	}

	docs, err := c.library.manager.GetDocsForQuery(ctx, query, maxTokens)
	if err != nil {
		return "", fmt.Errorf("get docs for query: %w", err)
	}

	return library.FormatDocsForPrompt(docs), nil
}

// getLibraryContextForWorkingDir returns library documentation relevant to the working directory.
// It scans the working directory for common project subdirectories to infer relevant collections.
func (c *Conductor) getLibraryContextForWorkingDir(ctx context.Context, workingDir string) (string, error) {
	if c.library == nil {
		return "", nil
	}

	// Get common subdirectories from the working directory for path matching
	projectDirs := c.inferProjectDirectories(workingDir)

	// Get library context for inferred paths
	return c.GetLibraryContextForPaths(ctx, projectDirs)
}

// inferProjectDirectories extracts likely relevant project directories from the working directory.
// This is a heuristic for auto-include when we don't have explicit file paths.
func (c *Conductor) inferProjectDirectories(workingDir string) []string {
	// For now, return the working directory itself
	// The library's path matching will use glob patterns to find relevant collections
	return []string{workingDir}
}
