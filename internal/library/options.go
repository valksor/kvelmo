package library

import (
	"fmt"
	"time"
)

// ValidateDomainScope returns an error if the domain scope value is invalid.
// Valid values are: "" (empty, defaults to same-host), "same-host", "same-domain".
func ValidateDomainScope(scope string) error {
	switch scope {
	case "", "same-host", "same-domain":
		return nil
	default:
		return fmt.Errorf("invalid domain_scope %q: must be 'same-host' or 'same-domain'", scope)
	}
}

// PullOptions configures a pull operation.
type PullOptions struct {
	// Name is the collection name (auto-generated if empty).
	Name string
	// IncludeMode controls when the collection is included.
	IncludeMode IncludeMode
	// Shared stores in the shared location (available to all projects).
	Shared bool
	// Paths are file glob patterns for auto-include.
	Paths []string
	// Tags are optional labels for the collection.
	Tags []string
	// GitRef is the branch/tag for git sources.
	GitRef string
	// GitPath is the subpath within git repos.
	GitPath string
	// MaxDepth overrides the default crawl depth for URLs.
	MaxDepth int
	// MaxPages overrides the default max pages for URLs.
	MaxPages int
	// DryRun shows what would be pulled without saving.
	DryRun bool
	// NoWait fails immediately if lock is unavailable.
	NoWait bool
	// Force overwrites existing collection without confirmation.
	Force bool

	// Resume-related options (for URL crawling)

	// Continue resumes an interrupted crawl if state exists (default: true when state exists).
	Continue bool
	// ForceRestart ignores existing state and starts fresh.
	ForceRestart bool
	// MaxRetries is the number of retry attempts per URL (default: 3).
	MaxRetries int
	// RetryDelay is the delay between retries (default: 2s).
	RetryDelay time.Duration
	// CheckpointPages is how often to checkpoint by page count (default: 10).
	CheckpointPages int
	// CheckpointTime is how often to checkpoint by time (default: 30s).
	CheckpointTime time.Duration

	// Crawl filtering options (override config defaults)

	// DomainScope controls link filtering: "same-host" (default) or "same-domain".
	DomainScope string
	// VersionFilter enables auto-detection of version path from source URL.
	VersionFilter bool
	// VersionPath is an explicit version path segment to filter links (e.g., "v24").
	VersionPath string
}

// DefaultPullOptions returns PullOptions with sensible defaults.
func DefaultPullOptions() *PullOptions {
	return &PullOptions{
		IncludeMode:     IncludeModeAuto,
		Shared:          false,
		MaxDepth:        3,
		MaxPages:        100,
		Continue:        true, // Resume by default when state exists
		MaxRetries:      3,
		RetryDelay:      2 * time.Second,
		CheckpointPages: 10,
		CheckpointTime:  30 * time.Second,
	}
}

// IncompleteCrawlError indicates an interrupted crawl exists for a collection.
// The caller should decide whether to resume (--continue) or restart (--restart).
type IncompleteCrawlError struct {
	CollectionID string
	Source       string
	Total        int
	Success      int
	Failed       int
	Pending      int
	StartedAt    time.Time
}

func (e *IncompleteCrawlError) Error() string {
	return fmt.Sprintf("incomplete crawl found for %s: %d/%d pages fetched (%d failed, %d pending). "+
		"Use --continue to resume or --restart to start fresh",
		e.CollectionID, e.Success, e.Total, e.Failed, e.Pending)
}

// ListOptions configures a list operation.
type ListOptions struct {
	// SharedOnly shows only shared collections.
	SharedOnly bool
	// ProjectOnly shows only project collections.
	ProjectOnly bool
	// IncludeMode filters by include mode.
	IncludeMode IncludeMode
	// Tag filters by tag.
	Tag string
}

// PullResult describes the outcome of a pull operation.
type PullResult struct {
	// Collection is the created/updated collection.
	Collection *Collection
	// PagesWritten is the number of pages successfully pulled.
	PagesWritten int
	// PagesFailed is the number of pages that failed to pull.
	PagesFailed int
	// PagesSkipped is the number of pages skipped (binary, robots.txt, etc.).
	PagesSkipped int
	// Errors contains per-page errors (non-fatal).
	Errors []error
	// RecoveryHint provides guidance when partial failure occurs.
	RecoveryHint string
	// DryRunURLs contains URLs that would be crawled (for dry-run mode).
	DryRunURLs []string

	// Filtering statistics (for observability)

	// LinksFiltered is the total number of links filtered out during crawl.
	LinksFiltered int
	// DomainFiltered is the count of links filtered by domain scope.
	DomainFiltered int
	// VersionFiltered is the count of links filtered by version path.
	VersionFiltered int
	// RobotsBlocked is the count of pages blocked by robots.txt.
	RobotsBlocked int
}

// UpdateResult describes the outcome of an update operation.
type UpdateResult struct {
	// Collection is the updated collection.
	Collection *Collection
	// PagesAdded is the count of new pages.
	PagesAdded int
	// PagesRemoved is the count of removed pages.
	PagesRemoved int
	// PagesUpdated is the count of updated pages.
	PagesUpdated int
}

// DocContext represents documentation content ready for prompt injection.
type DocContext struct {
	// Pages contains the selected page contents.
	Pages []*PageContent
	// TotalTokens is the estimated token count for all pages.
	TotalTokens int
	// Truncated indicates if some pages were truncated to fit token limits.
	Truncated bool
}

// PageContent represents a page's content for prompt injection.
type PageContent struct {
	// CollectionID is the unique collection identifier.
	CollectionID string
	// CollectionName is the human-readable collection name.
	CollectionName string
	// Path is the relative path within the collection.
	Path string
	// Title is the page title.
	Title string
	// Content is the markdown content of the page.
	Content string
	// TokenCount is the estimated token count.
	TokenCount int
	// Score is the relevance score (for ranked selection).
	Score float64
}

// CrawledPage represents a page fetched during crawling.
type CrawledPage struct {
	// URL is the source URL of the page.
	URL string
	// Path is the relative path for storage.
	Path string
	// Title is the extracted page title.
	Title string
	// Content is the markdown content.
	Content string
	// SizeBytes is the content size.
	SizeBytes int64
	// Error is set if this page failed to fetch.
	Error error
}

// ToPage converts a CrawledPage to a Page for storage.
func (cp *CrawledPage) ToPage() *Page {
	return &Page{
		Path:      cp.Path,
		SourceURL: cp.URL,
		Title:     cp.Title,
		SizeBytes: cp.SizeBytes,
	}
}

// CrawlResult describes the outcome of a crawl operation.
type CrawlResult struct {
	// Pages contains successfully crawled pages.
	Pages []*CrawledPage
	// Skipped is the count of pages skipped (binary, disallowed, etc.).
	Skipped int
	// Failed is the count of pages that errored.
	Failed int
	// Errors contains per-page errors (non-fatal).
	Errors []error
	// Truncated indicates crawl was stopped due to page limit.
	Truncated bool
	// TotalDiscovered is the total URLs discovered (before limit).
	TotalDiscovered int

	// Filtering statistics

	// LinksFiltered is the total number of links filtered out.
	LinksFiltered int
	// DomainFiltered is the count of links filtered by domain scope.
	DomainFiltered int
	// VersionFiltered is the count of links filtered by version path.
	VersionFiltered int
	// RobotsBlocked is the count of pages blocked by robots.txt.
	RobotsBlocked int
}

// Config holds library configuration settings (loaded from workspace config).
type Config struct {
	// AutoIncludeMax is the max collections to auto-include per prompt (default: 3).
	AutoIncludeMax int
	// MaxPagesPerPrompt is the max pages from a single collection (default: 20).
	MaxPagesPerPrompt int
	// MaxCrawlPages is the default max pages per crawl (default: 100).
	MaxCrawlPages int
	// MaxCrawlDepth is the default max crawl depth (default: 3).
	MaxCrawlDepth int
	// MaxPageSizeBytes is the max size per page (default: 1MB).
	MaxPageSizeBytes int64
	// LockTimeout is the file lock timeout (default: 10s).
	LockTimeout time.Duration
	// MaxTokenBudget is the total token budget for library context (default: 8000).
	MaxTokenBudget int

	// Crawl filtering defaults

	// DomainScope controls link filtering: "same-host" (default) or "same-domain".
	DomainScope string
	// VersionFilter enables auto-detection of version path from source URL.
	VersionFilter bool
	// VersionPath is an explicit version path segment to filter links.
	VersionPath string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		AutoIncludeMax:    3,
		MaxPagesPerPrompt: 20,
		MaxCrawlPages:     100,
		MaxCrawlDepth:     3,
		MaxPageSizeBytes:  1 << 20, // 1MB
		LockTimeout:       10 * time.Second,
		MaxTokenBudget:    8000,
	}
}
