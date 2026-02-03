package library

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
	"golang.org/x/net/publicsuffix"
)

// Crawler crawls documentation sites.
type Crawler struct {
	client       *http.Client
	config       *CrawlConfig
	robotsMap    map[string]*robotstxt.RobotsData
	robotsMu     sync.Mutex
	stateManager *CrawlStateManager // For resume capability
	store        *Store             // For incremental page writes
}

// NewCrawler creates a new crawler with the given configuration.
func NewCrawler(config *CrawlConfig) *Crawler {
	if config == nil {
		config = DefaultCrawlConfig()
	}

	client := &http.Client{
		Timeout: config.RequestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}

			return nil
		},
	}

	return &Crawler{
		client:    client,
		config:    config,
		robotsMap: make(map[string]*robotstxt.RobotsData),
	}
}

// Crawl discovers and fetches pages from a documentation site.
// Strategy: try sitemap.xml first, then fall back to link following.
func (c *Crawler) Crawl(ctx context.Context, rootURL string) (*CrawlResult, error) {
	// Parse root URL
	u, err := url.Parse(rootURL)
	if err != nil {
		return nil, fmt.Errorf("invalid root URL: %w", err)
	}

	// Set base URL if not specified
	if c.config.BaseURL == "" {
		c.config.BaseURL = rootURL
	}

	// Create context with total timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.TotalTimeout)
	defer cancel()

	// Discover URLs to crawl
	urls, discoverErr := c.discoverURLs(ctx, u)
	if discoverErr != nil {
		slog.Warn("URL discovery failed, falling back to single page", "error", discoverErr)
		urls = []string{rootURL}
	}

	result := &CrawlResult{
		TotalDiscovered: len(urls),
	}

	// Check if truncating due to limit
	if len(urls) > c.config.MaxPages {
		result.Truncated = true
		urls = urls[:c.config.MaxPages]
	}

	// Crawl pages with rate limiting
	for i, pageURL := range urls {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Rate limiting
		if i > 0 && c.config.DelayBetween > 0 {
			time.Sleep(c.config.DelayBetween)
		}

		// Check robots.txt
		if c.config.RespectRobotsTxt && !c.isAllowed(ctx, pageURL) {
			slog.Debug("blocked by robots.txt", "url", pageURL)
			result.Skipped++
			result.RobotsBlocked++

			continue
		}

		// Fetch page
		page, err := PullURL(ctx, pageURL, 0, c.config.UserAgent) // No size limit during crawl
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", pageURL, err))

			continue
		}

		result.Pages = append(result.Pages, page)
	}

	return result, nil
}

// SetStateManager sets the state manager for resume capability.
func (c *Crawler) SetStateManager(sm *CrawlStateManager) {
	c.stateManager = sm
}

// SetStore sets the store for incremental page writes.
func (c *Crawler) SetStore(store *Store) {
	c.store = store
}

// ResumeOptions configures the resume-capable crawl behavior.
type ResumeOptions struct {
	// Continue resumes an interrupted crawl if state exists.
	Continue bool
	// ForceRestart ignores existing state and starts fresh.
	ForceRestart bool
	// MaxRetries is the number of retry attempts for failed URLs (default: 3).
	MaxRetries int
	// RetryDelay is the delay between retries (default: 2s).
	RetryDelay time.Duration
	// CheckpointPages is how often to save state by page count (default: 10).
	CheckpointPages int
	// CheckpointTime is how often to save state by time (default: 30s).
	CheckpointTime time.Duration
}

// DefaultResumeOptions returns sensible defaults for resume options.
func DefaultResumeOptions() *ResumeOptions {
	return &ResumeOptions{
		Continue:        true,
		MaxRetries:      3,
		RetryDelay:      2 * time.Second,
		CheckpointPages: 10,
		CheckpointTime:  30 * time.Second,
	}
}

// CrawlWithResume performs a crawl with resume and incremental write support.
// If an incomplete crawl state exists and opts.Continue is true, it resumes.
// Pages are written to disk immediately as fetched (not batched at end).
func (c *Crawler) CrawlWithResume(ctx context.Context, rootURL string, opts *ResumeOptions) (*CrawlResult, error) {
	if opts == nil {
		opts = DefaultResumeOptions()
	}

	// Parse root URL
	u, err := url.Parse(rootURL)
	if err != nil {
		return nil, fmt.Errorf("invalid root URL: %w", err)
	}

	if c.config.BaseURL == "" {
		c.config.BaseURL = rootURL
	}

	// Create context with total timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.TotalTimeout)
	defer cancel()

	// Check for existing state (resume capability)
	var urls []string
	resuming := false

	if c.stateManager != nil && c.stateManager.HasIncompleteState() && opts.Continue && !opts.ForceRestart {
		// Resume from existing state
		state, loadErr := c.stateManager.LoadState()
		if loadErr != nil {
			slog.Warn("failed to load crawl state, starting fresh", "error", loadErr)
		} else {
			urls = state.GetPendingURLs()
			resuming = true
			stats := state.GetStats()
			slog.Info("resuming interrupted crawl",
				"pending", stats.Pending,
				"already_processed", stats.Success,
				"total", stats.Total)
		}
	}

	// If not resuming, perform discovery
	if !resuming {
		var discoverErr error
		urls, discoverErr = c.discoverURLs(ctx, u)
		if discoverErr != nil {
			slog.Warn("URL discovery failed, falling back to single page", "error", discoverErr)
			urls = []string{rootURL}
		}

		// Initialize state
		if c.stateManager != nil {
			c.stateManager.InitState(rootURL, c.config, urls)
			if saveErr := c.stateManager.SaveState(); saveErr != nil {
				slog.Warn("failed to save initial crawl state", "error", saveErr)
			}
		}
	}

	result := &CrawlResult{
		TotalDiscovered: len(urls),
	}

	// Apply page limit (only for new crawls, not resume)
	if !resuming && len(urls) > c.config.MaxPages {
		result.Truncated = true
		urls = urls[:c.config.MaxPages]
	}

	// Crawl pages with rate limiting and incremental writes
	processedCount := 0
	for _, pageURL := range urls {
		select {
		case <-ctx.Done():
			// Save state on context cancellation
			if c.stateManager != nil {
				_ = c.stateManager.SaveState()
			}

			return result, ctx.Err()
		default:
		}

		// Rate limiting
		if processedCount > 0 && c.config.DelayBetween > 0 {
			time.Sleep(c.config.DelayBetween)
		}

		// Check robots.txt
		if c.config.RespectRobotsTxt && !c.isAllowed(ctx, pageURL) {
			slog.Debug("blocked by robots.txt", "url", pageURL)
			result.Skipped++
			result.RobotsBlocked++
			if c.stateManager != nil {
				c.stateManager.MarkSkipped(pageURL, "disallowed by robots.txt")
			}

			continue
		}

		// Fetch page with retries
		page, fetchErr := c.fetchWithRetry(ctx, pageURL, opts.MaxRetries, opts.RetryDelay)
		if fetchErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", pageURL, fetchErr))
			if c.stateManager != nil {
				c.stateManager.MarkFailed(pageURL, fetchErr, opts.MaxRetries)
			}

			continue
		}

		// Write page to disk immediately (incremental writes)
		if c.store != nil && c.stateManager != nil {
			collectionID := c.stateManager.GetState().CollectionID
			if writeErr := c.store.WritePage(collectionID, page.Path, page.Content); writeErr != nil {
				slog.Warn("failed to write page", "path", page.Path, "error", writeErr)
			}
		}

		result.Pages = append(result.Pages, page)
		processedCount++

		// Mark success and checkpoint
		if c.stateManager != nil {
			c.stateManager.MarkSuccess(pageURL, page.Path, ComputeContentHash(page.Content))

			if c.stateManager.ShouldCheckpoint(processedCount, opts.CheckpointPages, opts.CheckpointTime) {
				if saveErr := c.stateManager.SaveState(); saveErr != nil {
					slog.Warn("failed to checkpoint crawl state", "error", saveErr)
				}
			}
		}
	}

	// Mark crawl as completed
	if c.stateManager != nil {
		c.stateManager.SetPhase(PhaseCompleted)
		_ = c.stateManager.SaveState()
	}

	return result, nil
}

// fetchWithRetry attempts to fetch a URL with retries on transient errors.
func (c *Crawler) fetchWithRetry(ctx context.Context, pageURL string, maxRetries int, retryDelay time.Duration) (*CrawledPage, error) {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if retryDelay <= 0 {
		retryDelay = 2 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		page, err := PullURL(ctx, pageURL, 0, c.config.UserAgent)
		if err == nil {
			return page, nil
		}

		lastErr = err

		// Don't retry on certain errors
		if isNonRetryableError(err) {
			return nil, err
		}

		slog.Debug("retrying failed fetch",
			"url", pageURL,
			"attempt", attempt+1,
			"max", maxRetries+1,
			"error", err)
	}

	return nil, fmt.Errorf("after %d attempts: %w", maxRetries+1, lastErr)
}

// isNonRetryableError checks if an error should not be retried.
func isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	return strings.Contains(errStr, "authentication required") ||
		strings.Contains(errStr, "non-text content type") ||
		strings.Contains(errStr, "HTTP 404") ||
		strings.Contains(errStr, "HTTP 403") ||
		strings.Contains(errStr, "HTTP 401")
}

// Preview returns the list of URLs that would be crawled (dry-run).
func (c *Crawler) Preview(ctx context.Context, rootURL string) ([]string, error) {
	u, err := url.Parse(rootURL)
	if err != nil {
		return nil, fmt.Errorf("invalid root URL: %w", err)
	}

	if c.config.BaseURL == "" {
		c.config.BaseURL = rootURL
	}

	urls, err := c.discoverURLs(ctx, u)
	if err != nil {
		return []string{rootURL}, nil //nolint:nilerr // Fallback to single page on discovery error
	}

	// Apply limit
	if len(urls) > c.config.MaxPages {
		urls = urls[:c.config.MaxPages]
	}

	return urls, nil
}

// discoverURLs discovers pages to crawl using sitemap or link following.
func (c *Crawler) discoverURLs(ctx context.Context, rootURL *url.URL) ([]string, error) {
	// Try sitemap.xml first
	sitemapURLs := []string{
		rootURL.Scheme + "://" + rootURL.Host + "/sitemap.xml",
		rootURL.Scheme + "://" + rootURL.Host + "/sitemap_index.xml",
	}

	for _, sitemapURL := range sitemapURLs {
		urls, err := c.fetchSitemap(ctx, sitemapURL)
		if err == nil && len(urls) > 0 {
			// Filter to base URL
			filtered := c.filterToBase(urls)
			if len(filtered) > 0 {
				slog.Info("discovered URLs from sitemap", "count", len(filtered), "sitemap", sitemapURL)

				return filtered, nil
			}
		}
	}

	// Fall back to link following
	slog.Info("no sitemap found, using link following", "root", rootURL.String())

	return c.followLinks(ctx, rootURL.String())
}

// fetchSitemap fetches and parses a sitemap.xml file.
func (c *Crawler) fetchSitemap(ctx context.Context, sitemapURL string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sitemapURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return nil, err
	}

	return parseSitemap(body)
}

// Sitemap structures for XML parsing.
type sitemapIndex struct {
	Sitemaps []sitemapLoc `xml:"sitemap"`
}

type sitemapLoc struct {
	Loc string `xml:"loc"`
}

type urlset struct {
	URLs []urlLoc `xml:"url"`
}

type urlLoc struct {
	Loc string `xml:"loc"`
}

// parseSitemap parses sitemap XML content.
func parseSitemap(data []byte) ([]string, error) {
	var urls []string

	// Try as sitemap index first
	var idx sitemapIndex
	if err := xml.Unmarshal(data, &idx); err == nil && len(idx.Sitemaps) > 0 {
		for _, s := range idx.Sitemaps {
			urls = append(urls, s.Loc)
		}

		return urls, nil
	}

	// Try as urlset
	var us urlset
	if err := xml.Unmarshal(data, &us); err == nil && len(us.URLs) > 0 {
		for _, u := range us.URLs {
			urls = append(urls, u.Loc)
		}

		return urls, nil
	}

	return nil, errors.New("could not parse sitemap")
}

// filterToBase filters URLs to only those within the base URL prefix.
func (c *Crawler) filterToBase(urls []string) []string {
	var filtered []string
	for _, u := range urls {
		if strings.HasPrefix(u, c.config.BaseURL) {
			filtered = append(filtered, u)
		}
	}

	return filtered
}

// followLinks discovers URLs by following links from a page.
func (c *Crawler) followLinks(ctx context.Context, startURL string) ([]string, error) {
	seen := make(map[string]bool)
	var result []string
	queue := []struct {
		url   string
		depth int
	}{{startURL, 0}}

	for len(queue) > 0 && len(result) < c.config.MaxPages {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		item := queue[0]
		queue = queue[1:]

		if seen[item.url] || item.depth > c.config.MaxDepth {
			continue
		}
		seen[item.url] = true

		// Rate limiting
		if len(result) > 0 && c.config.DelayBetween > 0 {
			time.Sleep(c.config.DelayBetween)
		}

		// Fetch page
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", c.config.UserAgent)

		resp, err := c.client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()

			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
		_ = resp.Body.Close()
		if err != nil {
			continue
		}

		// Add this URL to results
		result = append(result, item.url)

		// Extract links if we haven't reached max depth
		if item.depth < c.config.MaxDepth {
			links := c.extractLinks(string(body), item.url)
			for _, link := range links {
				if !seen[link] && strings.HasPrefix(link, c.config.BaseURL) {
					queue = append(queue, struct {
						url   string
						depth int
					}{link, item.depth + 1})
				}
			}
		}
	}

	return result, nil
}

// linkRegex matches href attributes in HTML.
var linkRegex = regexp.MustCompile(`href=["']([^"']+)["']`)

// versionPathRegex matches common version path segments like /v1/, /v24/, /v1.2.3/.
var versionPathRegex = regexp.MustCompile(`/v\d+(?:\.\d+)*/?`)

// extractLinks extracts links from HTML content with domain and version filtering.
// When config is nil, defaults to same-host filtering with no version filtering.
func (c *Crawler) extractLinks(html, baseURL string) []string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}

	var links []string
	seen := make(map[string]bool)

	matches := linkRegex.FindAllStringSubmatch(html, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		href := match[1]

		// Skip fragments, javascript, mailto
		if strings.HasPrefix(href, "#") ||
			strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "mailto:") {
			continue
		}

		// Resolve relative URLs
		linkURL, err := url.Parse(href)
		if err != nil {
			slog.Debug("skipped unparseable link", "href", href, "error", err, "base", baseURL)

			continue
		}

		resolved := base.ResolveReference(linkURL)

		// Apply domain scope filtering
		if !c.isAllowedDomain(resolved, base) {
			scope := "same-host"
			if c.config != nil && c.config.DomainScope != "" {
				scope = c.config.DomainScope
			}
			slog.Debug("filtered link: domain scope", "link", resolved.String(), "base_host", base.Host, "scope", scope)

			continue
		}

		// Apply version path filtering
		if c.config != nil && c.config.VersionPath != "" {
			if !containsVersionPath(resolved.Path, c.config.VersionPath) {
				slog.Debug("filtered link: version mismatch", "link", resolved.String(), "required_version", c.config.VersionPath)

				continue
			}
		}

		// Skip non-HTTP(S)
		if resolved.Scheme != "http" && resolved.Scheme != "https" {
			continue
		}

		// Normalize URL
		resolved.Fragment = ""
		normalizedURL := resolved.String()

		// Skip duplicates
		if seen[normalizedURL] {
			continue
		}
		seen[normalizedURL] = true

		links = append(links, normalizedURL)
	}

	return links
}

// isAllowedDomain checks if link is within the allowed domain scope.
func (c *Crawler) isAllowedDomain(link, base *url.URL) bool {
	if c.config == nil {
		// Default: same-host
		return link.Host == base.Host
	}

	switch c.config.DomainScope {
	case "same-domain":
		return extractRootDomain(link.Host) == extractRootDomain(base.Host)
	default: // "same-host" or empty
		return link.Host == base.Host
	}
}

// extractRootDomain returns the registrable domain (e.g., "example.com" from "docs.example.com").
// Uses the Public Suffix List to correctly handle TLDs like .co.uk, .com.au.
func extractRootDomain(host string) string {
	// Handle localhost and IP addresses
	if host == "localhost" || isIPAddress(host) {
		return host
	}

	// Use publicsuffix to get the registrable domain
	domain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		// Fallback: return the host as-is
		return host
	}

	return domain
}

// isIPAddress checks if host is an IP address (v4 or v6).
// Uses net.ParseIP for proper validation instead of string matching.
func isIPAddress(host string) bool {
	// Handle IPv6 with brackets (e.g., "[::1]:8080")
	if strings.HasPrefix(host, "[") {
		if idx := strings.Index(host, "]"); idx != -1 {
			host = host[1:idx]
		}
	} else {
		// Handle IPv4 with port (e.g., "192.168.1.1:8080")
		// For IPv6 without brackets, colons are part of the address
		if strings.Count(host, ":") == 1 {
			// Exactly one colon = IPv4 with port
			host = host[:strings.LastIndex(host, ":")]
		}
	}

	return net.ParseIP(host) != nil
}

// validateURL checks if a URL is safe to crawl.
// It validates the scheme and optionally blocks private/internal IPs for SSRF protection.
func (c *Crawler) validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http/https schemes
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme %q: only http and https are allowed", u.Scheme)
	}

	// Block private/internal IPs for SSRF protection
	if c.config.BlockPrivateIPs {
		hostname := u.Hostname()
		if ip := net.ParseIP(hostname); ip != nil {
			if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				return fmt.Errorf("private/internal IP address not allowed: %s", hostname)
			}
		}
	}

	return nil
}

// containsVersionPath checks if the URL path contains the specified version segment.
func containsVersionPath(path, version string) bool {
	// Handle version with or without leading/trailing slashes
	cleanVersion := strings.Trim(version, "/")

	// Check for /version/ in the middle
	if strings.Contains(path, "/"+cleanVersion+"/") {
		return true
	}

	// Check for /version at the end
	if strings.HasSuffix(path, "/"+cleanVersion) {
		return true
	}

	// Check for version as the first path segment (e.g., /v24/docs)
	if strings.HasPrefix(path, "/"+cleanVersion+"/") {
		return true
	}

	return false
}

// DetectVersionFromPath extracts a version segment from a URL path.
// Returns empty string if no version pattern is found.
// Examples:
//
//	"/docs/v24/intro" -> "v24"
//	"/api/v1.2.3/ref" -> "v1.2.3"
//	"/guide/latest"   -> ""
func DetectVersionFromPath(path string) string {
	match := versionPathRegex.FindString(path)
	if match != "" {
		return strings.Trim(match, "/")
	}

	return ""
}

// isAllowed checks if a URL is allowed by robots.txt.
func (c *Crawler) isAllowed(ctx context.Context, pageURL string) bool {
	u, err := url.Parse(pageURL)
	if err != nil {
		return false
	}

	robotsKey := u.Scheme + "://" + u.Host

	c.robotsMu.Lock()
	robots, ok := c.robotsMap[robotsKey]
	c.robotsMu.Unlock()

	if !ok {
		// Fetch robots.txt
		robotsURL := robotsKey + "/robots.txt"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
		if err != nil {
			return true // Allow if we can't fetch robots.txt
		}

		req.Header.Set("User-Agent", c.config.UserAgent)

		resp, err := c.client.Do(req)
		if err != nil {
			return true
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return true // No robots.txt = allow all
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
		if err != nil {
			return true
		}

		robots, err = robotstxt.FromBytes(body)
		if err != nil {
			return true
		}

		c.robotsMu.Lock()
		c.robotsMap[robotsKey] = robots
		c.robotsMu.Unlock()
	}

	// Check if our user agent is allowed
	return robots.TestAgent(u.Path, c.config.UserAgent)
}
