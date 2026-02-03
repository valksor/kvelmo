// Package library provides documentation pull and management for AI agent context.
// It allows pulling docs from URLs (with crawling), local files/directories, and git repos,
// storing them as collections that can be auto-included based on file path patterns.
package library

import (
	"time"
)

// SourceType indicates how the collection was obtained.
type SourceType string

const (
	// SourceURL indicates the collection was pulled from a URL (may involve crawling).
	SourceURL SourceType = "url"
	// SourceFile indicates the collection was pulled from local files or directories.
	SourceFile SourceType = "file"
	// SourceGit indicates the collection was cloned from a git repository.
	SourceGit SourceType = "git"
)

// IncludeMode controls when a collection is included in agent prompts.
type IncludeMode string

const (
	// IncludeModeAuto includes the collection when edited files match its path patterns.
	IncludeModeAuto IncludeMode = "auto"
	// IncludeModeExplicit includes the collection only when explicitly requested via --library flag.
	IncludeModeExplicit IncludeMode = "explicit"
	// IncludeModeAlways includes the collection in all prompts regardless of file context.
	IncludeModeAlways IncludeMode = "always"
)

// Collection represents a named group of documentation pages.
type Collection struct {
	// ID is the unique identifier for the collection (slug derived from name/source).
	ID string `yaml:"id"`
	// Name is the human-readable name for the collection.
	Name string `yaml:"name"`
	// Source is the original URL, file path, or git repo from which docs were pulled.
	Source string `yaml:"source"`
	// SourceType indicates how the collection was obtained.
	SourceType SourceType `yaml:"source_type"`
	// IncludeMode controls when the collection is included in prompts.
	IncludeMode IncludeMode `yaml:"include_mode"`
	// Paths are glob patterns for auto-include (e.g., "ide/vscode/**").
	Paths []string `yaml:"paths,omitempty"`
	// Tags are optional labels for organizing collections.
	Tags []string `yaml:"tags,omitempty"`
	// PulledAt is when the collection was last pulled.
	PulledAt time.Time `yaml:"pulled_at"`
	// PageCount is the number of pages in the collection.
	PageCount int `yaml:"page_count"`
	// TotalSize is the total size in bytes of all pages.
	TotalSize int64 `yaml:"total_size_bytes"`
	// Location indicates where this collection is stored ("project" or "shared").
	Location string `yaml:"location"`
	// GitRef is the branch/tag for git sources.
	GitRef string `yaml:"git_ref,omitempty"`
	// GitPath is the subpath within the git repo.
	GitPath string `yaml:"git_path,omitempty"`
}

// Page represents a single document page within a collection.
type Page struct {
	// Path is the relative path within the collection (e.g., "extension-guides/overview.md").
	Path string `yaml:"path"`
	// SourceURL is the original URL for URL-sourced pages.
	SourceURL string `yaml:"source_url,omitempty"`
	// Title is the page title extracted from content or URL.
	Title string `yaml:"title,omitempty"`
	// SizeBytes is the size of the page content.
	SizeBytes int64 `yaml:"size_bytes"`
}

// Manifest is the index of all collections in a store.
type Manifest struct {
	// Version is the manifest format version.
	Version string `yaml:"version"`
	// Collections is the list of all collections.
	Collections []*Collection `yaml:"collections"`
}

// NewManifest creates a new empty manifest.
func NewManifest() *Manifest {
	return &Manifest{
		Version:     "1",
		Collections: make([]*Collection, 0),
	}
}

// GetCollection returns a collection by ID, or nil if not found.
func (m *Manifest) GetCollection(id string) *Collection {
	for _, c := range m.Collections {
		if c.ID == id {
			return c
		}
	}

	return nil
}

// AddCollection adds or replaces a collection in the manifest.
func (m *Manifest) AddCollection(c *Collection) {
	for i, existing := range m.Collections {
		if existing.ID == c.ID {
			m.Collections[i] = c

			return
		}
	}
	m.Collections = append(m.Collections, c)
}

// RemoveCollection removes a collection by ID. Returns true if found and removed.
func (m *Manifest) RemoveCollection(id string) bool {
	for i, c := range m.Collections {
		if c.ID == id {
			m.Collections = append(m.Collections[:i], m.Collections[i+1:]...)

			return true
		}
	}

	return false
}

// CollectionMeta contains metadata stored alongside collection pages.
type CollectionMeta struct {
	// Version is the metadata format version.
	Version string `yaml:"version"`
	// Source is the original source URL/path.
	Source string `yaml:"source"`
	// SourceType indicates how the collection was obtained.
	SourceType SourceType `yaml:"source_type"`
	// CrawlConfig holds crawler settings for URL sources.
	CrawlConfig *CrawlConfig `yaml:"crawl_config,omitempty"`
	// Pages is the list of pages in this collection.
	Pages []*Page `yaml:"pages"`
}

// NewCollectionMeta creates a new collection metadata instance.
func NewCollectionMeta(source string, sourceType SourceType) *CollectionMeta {
	return &CollectionMeta{
		Version:    "1",
		Source:     source,
		SourceType: sourceType,
		Pages:      make([]*Page, 0),
	}
}

// CrawlConfig holds configuration for the web crawler.
type CrawlConfig struct {
	// MaxDepth is the maximum link depth to crawl (default: 3).
	MaxDepth int `yaml:"max_depth"`
	// MaxPages is the maximum number of pages to crawl (default: 100).
	MaxPages int `yaml:"max_pages"`
	// BaseURL is the URL prefix to stay within during crawling.
	BaseURL string `yaml:"base_url"`
	// DelayBetween is the delay between requests (default: 500ms).
	DelayBetween time.Duration `yaml:"delay_between,omitempty"`
	// RequestTimeout is the timeout for each request (default: 30s).
	RequestTimeout time.Duration `yaml:"request_timeout,omitempty"`
	// TotalTimeout is the total crawl timeout (default: 10m).
	TotalTimeout time.Duration `yaml:"total_timeout,omitempty"`
	// RespectRobotsTxt controls whether to respect robots.txt (default: true).
	RespectRobotsTxt bool `yaml:"respect_robots_txt"`
	// UserAgent is the User-Agent header for requests.
	UserAgent string `yaml:"user_agent,omitempty"`

	// Link filtering options

	// DomainScope controls which domains are followed: "same-host" (default) or "same-domain".
	DomainScope string `yaml:"domain_scope,omitempty"`
	// VersionPath restricts crawling to URLs containing this version segment (e.g., "v24").
	VersionPath string `yaml:"version_path,omitempty"`

	// Security options

	// BlockPrivateIPs blocks crawling to private/internal IP addresses (127.x, 10.x, 192.168.x, etc.)
	// for SSRF protection. Default: true for remote sources.
	BlockPrivateIPs bool `yaml:"block_private_ips"`
}

// DefaultCrawlConfig returns a CrawlConfig with sensible defaults.
func DefaultCrawlConfig() *CrawlConfig {
	return &CrawlConfig{
		MaxDepth:         3,
		MaxPages:         100,
		DelayBetween:     500 * time.Millisecond,
		RequestTimeout:   30 * time.Second,
		TotalTimeout:     10 * time.Minute,
		RespectRobotsTxt: true,
		UserAgent:        "mehr-library/1.0",
		BlockPrivateIPs:  true,
	}
}

// StorageLocation indicates where a collection is stored.
type StorageLocation string

const (
	// LocationProject stores collections namespaced to the current project.
	LocationProject StorageLocation = "project"
	// LocationShared stores collections available to all projects.
	LocationShared StorageLocation = "shared"
)

// CrawlPhase indicates the current phase of a crawl operation.
type CrawlPhase string

const (
	// PhaseDiscovery is when URLs are being discovered via sitemap/links.
	PhaseDiscovery CrawlPhase = "discovery"
	// PhaseFetching is when pages are being fetched.
	PhaseFetching CrawlPhase = "fetching"
	// PhaseCompleted indicates successful crawl completion.
	PhaseCompleted CrawlPhase = "completed"
	// PhaseFailed indicates the crawl failed.
	PhaseFailed CrawlPhase = "failed"
)

// URL status constants for tracking individual URL processing.
const (
	URLStatusPending = "pending"
	URLStatusSuccess = "success"
	URLStatusFailed  = "failed"
	URLStatusSkipped = "skipped"
)

// CrawlState tracks the progress of an in-progress crawl for resume capability.
// Stored as .crawl-state.yaml in the collection directory during active crawls.
type CrawlState struct {
	// Version is the state format version for forward compatibility.
	Version string `yaml:"version"`
	// CollectionID is the target collection being crawled.
	CollectionID string `yaml:"collection_id"`
	// Source is the original source URL.
	Source string `yaml:"source"`
	// Config holds the crawl configuration used.
	Config *CrawlConfig `yaml:"config"`
	// DiscoveredURLs is the complete list of URLs found during discovery.
	DiscoveredURLs []string `yaml:"discovered_urls"`
	// ProcessedURLs tracks status of each URL.
	ProcessedURLs map[string]URLStatus `yaml:"processed_urls"`
	// StartedAt is when the crawl began.
	StartedAt time.Time `yaml:"started_at"`
	// LastUpdatedAt is when state was last persisted.
	LastUpdatedAt time.Time `yaml:"last_updated_at"`
	// Phase indicates current crawl phase.
	Phase CrawlPhase `yaml:"phase"`
}

// URLStatus tracks the processing status of a single URL.
type URLStatus struct {
	// Status indicates the outcome: pending, success, failed, skipped.
	Status string `yaml:"status"`
	// PagePath is the stored path (if successful).
	PagePath string `yaml:"page_path,omitempty"`
	// Error contains the error message (if failed).
	Error string `yaml:"error,omitempty"`
	// ProcessedAt is when this URL was processed.
	ProcessedAt time.Time `yaml:"processed_at,omitempty"`
	// RetryCount tracks retry attempts.
	RetryCount int `yaml:"retry_count,omitempty"`
	// ContentHash is SHA256 of content for detecting changes.
	ContentHash string `yaml:"content_hash,omitempty"`
}

// NewCrawlState creates a new crawl state for a collection.
func NewCrawlState(collectionID, source string, config *CrawlConfig) *CrawlState {
	return &CrawlState{
		Version:       "1",
		CollectionID:  collectionID,
		Source:        source,
		Config:        config,
		ProcessedURLs: make(map[string]URLStatus),
		StartedAt:     time.Now(),
		LastUpdatedAt: time.Now(),
		Phase:         PhaseDiscovery,
	}
}

// GetPendingURLs returns URLs that haven't been successfully processed.
func (s *CrawlState) GetPendingURLs() []string {
	var pending []string
	for _, url := range s.DiscoveredURLs {
		status := s.ProcessedURLs[url]
		if status.Status == URLStatusPending || status.Status == URLStatusFailed {
			pending = append(pending, url)
		}
	}

	return pending
}

// CrawlStats holds crawl progress statistics.
type CrawlStats struct {
	Total   int
	Success int
	Failed  int
	Pending int
	Skipped int
}

// GetStats returns crawl progress statistics.
func (s *CrawlState) GetStats() CrawlStats {
	stats := CrawlStats{
		Total: len(s.DiscoveredURLs),
	}
	for _, status := range s.ProcessedURLs {
		switch status.Status {
		case URLStatusSuccess:
			stats.Success++
		case URLStatusFailed:
			stats.Failed++
		case URLStatusPending:
			stats.Pending++
		case URLStatusSkipped:
			stats.Skipped++
		}
	}

	return stats
}
