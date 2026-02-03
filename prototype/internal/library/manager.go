package library

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// Manager provides high-level operations for library collections.
// It manages both project-local and shared (global) stores.
type Manager struct {
	projectStore *Store
	sharedStore  *Store
	config       *Config
}

// NewManager creates a new Manager with project and shared stores.
// The repoRoot is used to determine the project-local storage location.
// If repoRoot is empty, only shared storage is available.
func NewManager(ctx context.Context, repoRoot string) (*Manager, error) {
	config := DefaultConfig()

	var projectStore *Store
	if repoRoot != "" {
		var err error
		projectStore, err = NewStore(ctx, repoRoot, false, config.LockTimeout)
		if err != nil {
			return nil, fmt.Errorf("create project store: %w", err)
		}
	}

	sharedStore, err := NewStore(ctx, "", true, config.LockTimeout)
	if err != nil {
		return nil, fmt.Errorf("create shared store: %w", err)
	}

	return &Manager{
		projectStore: projectStore,
		sharedStore:  sharedStore,
		config:       config,
	}, nil
}

// NewManagerWithConfig creates a Manager with custom configuration.
func NewManagerWithConfig(ctx context.Context, repoRoot string, config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var projectStore *Store
	if repoRoot != "" {
		var err error
		projectStore, err = NewStore(ctx, repoRoot, false, config.LockTimeout)
		if err != nil {
			return nil, fmt.Errorf("create project store: %w", err)
		}
	}

	sharedStore, err := NewStore(ctx, "", true, config.LockTimeout)
	if err != nil {
		return nil, fmt.Errorf("create shared store: %w", err)
	}

	return &Manager{
		projectStore: projectStore,
		sharedStore:  sharedStore,
		config:       config,
	}, nil
}

// Pull fetches documentation from a source and stores it as a collection.
func (m *Manager) Pull(ctx context.Context, source string, opts *PullOptions) (*PullResult, error) {
	if opts == nil {
		opts = DefaultPullOptions()
	}

	// Determine source type
	sourceType := DetectSourceType(source)

	// Generate collection ID
	collectionID := GenerateCollectionID(opts.Name, source)

	// Select store
	store := m.projectStore
	location := LocationProject
	if opts.Shared {
		store = m.sharedStore
		location = LocationShared
	}
	if store == nil {
		return nil, fmt.Errorf("no store available (shared=%v)", opts.Shared)
	}

	// Fetch pages based on source type
	var pages []*CrawledPage
	var crawlResult *CrawlResult // Tracks stats for URL crawls
	var err error

	switch sourceType {
	case SourceFile:
		pages, err = PullFile(source, m.config.MaxPageSizeBytes)
	case SourceURL:
		if opts.DryRun {
			return m.pullPreview(ctx, source, opts)
		}
		crawlResult, err = m.pullURL(ctx, source, opts, store, collectionID)
		if crawlResult != nil {
			pages = crawlResult.Pages
		}
	case SourceGit:
		pages, err = PullGit(ctx, source, opts.GitRef, opts.GitPath, m.config.MaxPageSizeBytes)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceType)
	}

	if err != nil {
		return nil, fmt.Errorf("pull %s: %w", sourceType, err)
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages found from source: %s", source)
	}

	// Derive name if not provided
	name := opts.Name
	if name == "" {
		if len(pages) > 0 && pages[0].Title != "" {
			name = pages[0].Title
		} else {
			name = deriveNameFromSource(source)
		}
	}

	// Create collection
	collection := &Collection{
		ID:          collectionID,
		Name:        name,
		Source:      source,
		SourceType:  sourceType,
		IncludeMode: opts.IncludeMode,
		Paths:       opts.Paths,
		Tags:        opts.Tags,
		PulledAt:    time.Now(),
		PageCount:   len(pages),
		TotalSize:   sumPageSizes(pages),
		Location:    string(location),
		GitRef:      opts.GitRef,
		GitPath:     opts.GitPath,
	}

	// Save collection and pages
	if err := store.SaveCollection(collection); err != nil {
		return nil, fmt.Errorf("save collection: %w", err)
	}

	writtenPages, writeErrs := store.WritePages(collectionID, pages)
	if len(writeErrs) > 0 {
		slog.Warn("some pages failed to write", "errors", len(writeErrs), "first_error", writeErrs[0])
	}

	// Save collection metadata (page list with titles)
	meta := &CollectionMeta{
		Version:    "1",
		Source:     source,
		SourceType: sourceType,
		Pages:      writtenPages,
	}
	if err := store.SaveCollectionMeta(collectionID, meta); err != nil {
		return nil, fmt.Errorf("save collection meta: %w", err)
	}

	// Update manifest
	manifest, err := store.LoadManifest()
	if err != nil {
		manifest = NewManifest()
	}

	manifest.AddCollection(collection)

	if err := store.SaveManifest(manifest); err != nil {
		return nil, fmt.Errorf("save manifest: %w", err)
	}

	slog.Info("pulled documentation",
		"id", collectionID,
		"name", collection.Name,
		"pages", len(pages),
		"source", sourceType)

	result := &PullResult{
		Collection:   collection,
		PagesWritten: len(pages),
	}

	// Include crawl stats if available (URL sources)
	if crawlResult != nil {
		result.PagesFailed = crawlResult.Failed
		result.PagesSkipped = crawlResult.Skipped
		result.Errors = crawlResult.Errors
		result.LinksFiltered = crawlResult.LinksFiltered
		result.DomainFiltered = crawlResult.DomainFiltered
		result.VersionFiltered = crawlResult.VersionFiltered
		result.RobotsBlocked = crawlResult.RobotsBlocked
	}

	return result, nil
}

// pullURL fetches pages from a URL, optionally crawling for multiple pages.
// Supports resume/continuation for interrupted crawls.
// Returns the full CrawlResult to preserve stats (RobotsBlocked, etc.).
func (m *Manager) pullURL(ctx context.Context, rootURL string, opts *PullOptions, store *Store, collectionID string) (*CrawlResult, error) {
	crawlConfig := DefaultCrawlConfig()

	if m.config.MaxCrawlPages > 0 {
		crawlConfig.MaxPages = m.config.MaxCrawlPages
	}
	if m.config.MaxCrawlDepth > 0 {
		crawlConfig.MaxDepth = m.config.MaxCrawlDepth
	}
	if opts.MaxDepth > 0 {
		crawlConfig.MaxDepth = opts.MaxDepth
	}
	if opts.MaxPages > 0 {
		crawlConfig.MaxPages = opts.MaxPages
	}

	// Apply crawl filtering from library config (defaults)
	if m.config.DomainScope != "" {
		crawlConfig.DomainScope = m.config.DomainScope
	}
	if m.config.VersionPath != "" {
		crawlConfig.VersionPath = m.config.VersionPath
	} else if m.config.VersionFilter {
		// Auto-detect version from source URL
		if parsed, err := url.Parse(rootURL); err == nil {
			crawlConfig.VersionPath = DetectVersionFromPath(parsed.Path)
		}
	}

	// Apply crawl filtering from pull options (overrides config)
	if opts.DomainScope != "" {
		crawlConfig.DomainScope = opts.DomainScope
	}
	if opts.VersionPath != "" {
		crawlConfig.VersionPath = opts.VersionPath
	} else if opts.VersionFilter {
		// Auto-detect version from source URL
		if parsed, err := url.Parse(rootURL); err == nil {
			crawlConfig.VersionPath = DetectVersionFromPath(parsed.Path)
		}
	}

	// Create state manager for resume capability
	stateManager := NewCrawlStateManager(store, collectionID)

	// Check for incomplete crawl (unless we're forcing restart or continuing)
	if stateManager.HasIncompleteState() && !opts.Continue && !opts.ForceRestart {
		state, loadErr := stateManager.LoadState()
		if loadErr == nil {
			stats := state.GetStats()

			return nil, &IncompleteCrawlError{
				CollectionID: collectionID,
				Source:       rootURL,
				Total:        stats.Total,
				Success:      stats.Success,
				Failed:       stats.Failed,
				Pending:      stats.Pending,
				StartedAt:    state.StartedAt,
			}
		}
	}

	// Create crawler with state manager and store for incremental writes
	crawler := NewCrawler(crawlConfig)
	crawler.SetStateManager(stateManager)
	crawler.SetStore(store)

	// Build resume options
	resumeOpts := &ResumeOptions{
		Continue:        opts.Continue,
		ForceRestart:    opts.ForceRestart,
		MaxRetries:      opts.MaxRetries,
		RetryDelay:      opts.RetryDelay,
		CheckpointPages: opts.CheckpointPages,
		CheckpointTime:  opts.CheckpointTime,
	}
	if resumeOpts.MaxRetries == 0 {
		resumeOpts.MaxRetries = 3
	}
	if resumeOpts.RetryDelay == 0 {
		resumeOpts.RetryDelay = 2 * time.Second
	}
	if resumeOpts.CheckpointPages == 0 {
		resumeOpts.CheckpointPages = 10
	}
	if resumeOpts.CheckpointTime == 0 {
		resumeOpts.CheckpointTime = 30 * time.Second
	}

	result, err := crawler.CrawlWithResume(ctx, rootURL, resumeOpts)
	if err != nil {
		return nil, err
	}

	if len(result.Errors) > 0 {
		slog.Warn("some pages failed to fetch",
			"failed", result.Failed,
			"skipped", result.Skipped,
			"first_error", result.Errors[0])
	}

	// Delete state file on successful completion
	if stateManager.HasIncompleteState() {
		if delErr := stateManager.DeleteState(); delErr != nil {
			slog.Warn("failed to delete crawl state after completion", "error", delErr)
		}
	}

	return result, nil
}

// pullPreview returns what would be crawled without actually fetching content.
func (m *Manager) pullPreview(ctx context.Context, rootURL string, opts *PullOptions) (*PullResult, error) {
	crawlConfig := DefaultCrawlConfig()

	if opts.MaxDepth > 0 {
		crawlConfig.MaxDepth = opts.MaxDepth
	}
	if opts.MaxPages > 0 {
		crawlConfig.MaxPages = opts.MaxPages
	}

	// Apply crawl filtering from library config (defaults)
	if m.config.DomainScope != "" {
		crawlConfig.DomainScope = m.config.DomainScope
	}
	if m.config.VersionPath != "" {
		crawlConfig.VersionPath = m.config.VersionPath
	} else if m.config.VersionFilter {
		// Auto-detect version from source URL
		if parsed, err := url.Parse(rootURL); err == nil {
			crawlConfig.VersionPath = DetectVersionFromPath(parsed.Path)
		}
	}

	// Apply crawl filtering from pull options (overrides config)
	if opts.DomainScope != "" {
		crawlConfig.DomainScope = opts.DomainScope
	}
	if opts.VersionPath != "" {
		crawlConfig.VersionPath = opts.VersionPath
	} else if opts.VersionFilter {
		// Auto-detect version from source URL
		if parsed, err := url.Parse(rootURL); err == nil {
			crawlConfig.VersionPath = DetectVersionFromPath(parsed.Path)
		}
	}

	crawler := NewCrawler(crawlConfig)
	urls, err := crawler.Preview(ctx, rootURL)
	if err != nil {
		return nil, err
	}

	return &PullResult{
		Collection: &Collection{
			ID:   GenerateCollectionID(opts.Name, rootURL),
			Name: opts.Name,
		},
		PagesWritten: 0,
		DryRunURLs:   urls,
	}, nil
}

// List returns all collections from the specified stores.
func (m *Manager) List(_ context.Context, opts *ListOptions) ([]*Collection, error) {
	if opts == nil {
		opts = &ListOptions{}
	}

	var result []*Collection

	// Include project collections unless SharedOnly
	if !opts.SharedOnly && m.projectStore != nil {
		manifest, err := m.projectStore.LoadManifest()
		if err == nil && manifest != nil {
			for _, coll := range manifest.Collections {
				if matchesListFilter(coll, opts) {
					result = append(result, coll)
				}
			}
		}
	}

	// Include shared collections unless ProjectOnly
	if !opts.ProjectOnly && m.sharedStore != nil {
		manifest, err := m.sharedStore.LoadManifest()
		if err == nil && manifest != nil {
			for _, coll := range manifest.Collections {
				if matchesListFilter(coll, opts) {
					result = append(result, coll)
				}
			}
		}
	}

	return result, nil
}

// matchesListFilter checks if a collection matches the list filter options.
func matchesListFilter(coll *Collection, opts *ListOptions) bool {
	if opts.IncludeMode != "" && coll.IncludeMode != opts.IncludeMode {
		return false
	}
	if opts.Tag != "" {
		hasTag := false
		for _, t := range coll.Tags {
			if t == opts.Tag {
				hasTag = true

				break
			}
		}
		if !hasTag {
			return false
		}
	}

	return true
}

// Show returns a collection by name or ID.
func (m *Manager) Show(_ context.Context, nameOrID string) (*Collection, error) {
	// Try project store first
	if m.projectStore != nil {
		if coll, err := m.projectStore.GetCollection(nameOrID); err == nil {
			return coll, nil
		}
	}

	// Try shared store
	if m.sharedStore != nil {
		if coll, err := m.sharedStore.GetCollection(nameOrID); err == nil {
			return coll, nil
		}
	}

	return nil, fmt.Errorf("collection not found: %s", nameOrID)
}

// ShowPage returns a specific page's metadata and content from a collection.
func (m *Manager) ShowPage(_ context.Context, collectionID, pagePath string) (*Page, string, error) {
	// Try project store first
	if m.projectStore != nil {
		content, err := m.projectStore.ReadPage(collectionID, pagePath)
		if err == nil {
			page := m.findPageMeta(m.projectStore, collectionID, pagePath)

			return page, content, nil
		}
	}

	// Try shared store
	if m.sharedStore != nil {
		content, err := m.sharedStore.ReadPage(collectionID, pagePath)
		if err == nil {
			page := m.findPageMeta(m.sharedStore, collectionID, pagePath)

			return page, content, nil
		}
	}

	return nil, "", fmt.Errorf("page not found: %s/%s", collectionID, pagePath)
}

// findPageMeta looks up page metadata from the collection's meta file.
func (m *Manager) findPageMeta(store *Store, collectionID, pagePath string) *Page {
	meta, err := store.LoadCollectionMeta(collectionID)
	if err != nil {
		return &Page{Path: pagePath}
	}
	for _, p := range meta.Pages {
		if p.Path == pagePath {
			return p
		}
	}

	return &Page{Path: pagePath}
}

// ListPages returns all pages in a collection.
func (m *Manager) ListPages(_ context.Context, collectionID string) ([]string, error) {
	// Try project store first
	if m.projectStore != nil {
		if paths, err := m.projectStore.ListPageFiles(collectionID); err == nil && len(paths) > 0 {
			return paths, nil
		}
	}

	// Try shared store
	if m.sharedStore != nil {
		if paths, err := m.sharedStore.ListPageFiles(collectionID); err == nil && len(paths) > 0 {
			return paths, nil
		}
	}

	return nil, fmt.Errorf("collection not found: %s", collectionID)
}

// Remove deletes a collection from the appropriate store.
func (m *Manager) Remove(_ context.Context, nameOrID string, _ bool) error {
	// Try to find and remove from project store
	if m.projectStore != nil {
		if _, err := m.projectStore.GetCollection(nameOrID); err == nil {
			return m.removeFromStore(m.projectStore, nameOrID)
		}
	}

	// Try shared store
	if m.sharedStore != nil {
		if _, err := m.sharedStore.GetCollection(nameOrID); err == nil {
			return m.removeFromStore(m.sharedStore, nameOrID)
		}
	}

	return fmt.Errorf("collection not found: %s", nameOrID)
}

// removeFromStore removes a collection from a specific store.
func (m *Manager) removeFromStore(store *Store, collectionID string) error {
	// Delete collection directory
	if err := store.DeleteCollection(collectionID); err != nil {
		return fmt.Errorf("delete collection: %w", err)
	}

	// Update manifest
	manifest, err := store.LoadManifest()
	if err != nil {
		//nolint:nilerr // Collection was deleted; manifest update is best-effort
		return nil
	}

	manifest.RemoveCollection(collectionID)

	if err := store.SaveManifest(manifest); err != nil {
		slog.Warn("failed to update manifest after deletion", "error", err)
	}

	return nil
}

// Update refreshes a collection from its original source.
// By default, updates auto-continue any interrupted crawls.
func (m *Manager) Update(ctx context.Context, nameOrID string) (*PullResult, error) {
	return m.UpdateWithOptions(ctx, nameOrID, true, false)
}

// UpdateWithOptions refreshes a collection with explicit resume control.
// If continueInterrupted is true, incomplete crawls are auto-resumed.
// If fullRefresh is true, all pages are re-fetched regardless of state.
func (m *Manager) UpdateWithOptions(ctx context.Context, nameOrID string, continueInterrupted, fullRefresh bool) (*PullResult, error) {
	// Find the collection
	coll, err := m.Show(ctx, nameOrID)
	if err != nil {
		return nil, err
	}

	// Determine which store it's in
	isShared := coll.Location == string(LocationShared)

	// Build options with resume settings
	opts := DefaultPullOptions()
	opts.Name = coll.Name
	opts.IncludeMode = coll.IncludeMode
	opts.Paths = coll.Paths
	opts.Tags = coll.Tags
	opts.Shared = isShared
	opts.GitRef = coll.GitRef
	opts.GitPath = coll.GitPath
	opts.Continue = continueInterrupted
	opts.ForceRestart = fullRefresh

	return m.Pull(ctx, coll.Source, opts)
}

// GetStore returns the appropriate store based on the shared flag.
func (m *Manager) GetStore(shared bool) *Store {
	if shared {
		return m.sharedStore
	}

	return m.projectStore
}

// Config returns the manager's configuration.
func (m *Manager) Config() *Config {
	return m.config
}

// Helper functions

func sumPageSizes(pages []*CrawledPage) int64 {
	var total int64
	for _, p := range pages {
		total += p.SizeBytes
	}

	return total
}

func deriveNameFromSource(source string) string {
	sourceType := DetectSourceType(source)

	switch sourceType {
	case SourceURL, SourceGit:
		// For URLs, parse and derive a readable name
		if u, err := url.Parse(source); err == nil {
			return deriveIDFromURL(u)
		}

		return slugify(source)
	case SourceFile:
		// For files, just use the base name
		return filepath.Base(filepath.Clean(source))
	default:
		return slugify(source)
	}
}

// ResolveCollectionID finds a collection by name or ID and returns the canonical ID.
func (m *Manager) ResolveCollectionID(ctx context.Context, nameOrID string) (string, error) {
	coll, err := m.Show(ctx, nameOrID)
	if err != nil {
		return "", err
	}

	return coll.ID, nil
}

// GetCrawlState returns the crawl state for a collection if one exists.
func (m *Manager) GetCrawlState(collectionID string) (*CrawlState, error) {
	// Try project store first
	if m.projectStore != nil {
		stateManager := NewCrawlStateManager(m.projectStore, collectionID)
		if stateManager.HasIncompleteState() {
			return stateManager.LoadState()
		}
	}

	// Try shared store
	if m.sharedStore != nil {
		stateManager := NewCrawlStateManager(m.sharedStore, collectionID)
		if stateManager.HasIncompleteState() {
			return stateManager.LoadState()
		}
	}

	return nil, fmt.Errorf("no crawl state found for: %s", collectionID)
}

// ListIncompleteCollections returns collections with interrupted crawls.
func (m *Manager) ListIncompleteCollections(_ context.Context) ([]*CrawlState, error) {
	var result []*CrawlState

	// Check project store
	if m.projectStore != nil {
		ids, err := m.projectStore.ListIncompleteCollections()
		if err == nil {
			for _, id := range ids {
				stateManager := NewCrawlStateManager(m.projectStore, id)
				if state, loadErr := stateManager.LoadState(); loadErr == nil {
					result = append(result, state)
				}
			}
		}
	}

	// Check shared store
	if m.sharedStore != nil {
		ids, err := m.sharedStore.ListIncompleteCollections()
		if err == nil {
			for _, id := range ids {
				stateManager := NewCrawlStateManager(m.sharedStore, id)
				if state, loadErr := stateManager.LoadState(); loadErr == nil {
					result = append(result, state)
				}
			}
		}
	}

	return result, nil
}

// NewManagerFromWorkspace creates a Manager using workspace storage paths.
func NewManagerFromWorkspace(ctx context.Context, ws *storage.Workspace) (*Manager, error) {
	config := DefaultConfig()

	var projectStore *Store
	if ws != nil {
		projectStore = NewStoreWithRoot(filepath.Join(ws.Root(), "library"), config.LockTimeout)
	}

	sharedStore, err := NewStore(ctx, "", true, config.LockTimeout)
	if err != nil {
		return nil, fmt.Errorf("create shared store: %w", err)
	}

	return &Manager{
		projectStore: projectStore,
		sharedStore:  sharedStore,
		config:       config,
	}, nil
}
