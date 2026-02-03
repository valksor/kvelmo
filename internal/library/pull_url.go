package library

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// httpClient is the default HTTP client with sensible timeouts.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}

		return nil
	},
}

// textContentTypes are MIME types considered text content.
var textContentTypes = []string{
	"text/html",
	"text/plain",
	"text/markdown",
	"application/json",
	"application/xml",
	"text/xml",
}

// PullURL fetches a single URL and converts it to markdown.
func PullURL(ctx context.Context, pageURL string, maxPageSize int64, userAgent string) (*CrawledPage, error) {
	if userAgent == "" {
		userAgent = "mehr-library/1.0"
	}

	// Parse and validate URL
	u, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/markdown,text/plain,*/*")

	// Execute request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check for auth errors
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf(`authentication required for %s

To pull auth-protected documentation:
1. Open the URL in your browser (logged in)
2. Use browser's "Save as" → "Webpage, Complete"
3. Run: mehr library pull ./saved-docs/ --name "my-docs"

See: https://valksor.com/docs/mehrhof/library#auth-protected`, pageURL)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !isTextContentType(contentType) {
		return nil, fmt.Errorf("non-text content type: %s", contentType)
	}

	// Read body with size limit
	var reader io.Reader = resp.Body
	if maxPageSize > 0 {
		reader = io.LimitReader(resp.Body, maxPageSize+1)
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if maxPageSize > 0 && int64(len(body)) > maxPageSize {
		return nil, fmt.Errorf("page exceeds max size (%d bytes)", maxPageSize)
	}

	// Check for JS-rendered content
	if isJSRendered(string(body)) {
		return nil, errors.New(`page appears to be JavaScript-rendered. Content may be incomplete.
Tip: Use Chrome to export HTML, then: mehr library pull ./exported-docs/`)
	}

	// Convert to markdown
	content, title, err := htmlToMarkdown(string(body), pageURL)
	if err != nil {
		// If conversion fails, use raw content
		content = string(body)
		title = extractTitleFromHTML(string(body))
	}

	// Derive path from URL
	pagePath := urlToPath(u)

	return &CrawledPage{
		URL:       pageURL,
		Path:      pagePath,
		Title:     title,
		Content:   content,
		SizeBytes: int64(len(content)),
	}, nil
}

// isTextContentType checks if a content type is text-based.
func isTextContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)
	for _, t := range textContentTypes {
		if strings.HasPrefix(contentType, t) {
			return true
		}
	}

	return false
}

// isJSRendered checks if HTML content appears to be primarily JS-rendered.
func isJSRendered(html string) bool {
	// Check for common signs of JS-rendered content
	// Empty body with JS app mount points
	if strings.Contains(html, `<div id="root"></div>`) ||
		strings.Contains(html, `<div id="app"></div>`) ||
		strings.Contains(html, `<div id="__next"></div>`) {
		// Check if there's significant content outside these divs
		// by looking for paragraph or heading tags
		if !strings.Contains(html, "<p>") && !strings.Contains(html, "<h1") &&
			!strings.Contains(html, "<h2") && !strings.Contains(html, "<article") {
			return true
		}
	}

	return false
}

// htmlToMarkdown converts HTML to markdown using the html-to-markdown library.
func htmlToMarkdown(html, _ string) (string, string, error) {
	markdown, err := htmltomarkdown.ConvertString(html)
	if err != nil {
		return "", "", err
	}

	// Clean up excessive whitespace
	markdown = cleanMarkdown(markdown)

	// Extract title
	title := extractTitleFromHTML(html)

	return markdown, title, nil
}

// cleanMarkdown removes excessive whitespace and normalizes the markdown.
func cleanMarkdown(md string) string {
	// Replace multiple newlines with two
	for strings.Contains(md, "\n\n\n") {
		md = strings.ReplaceAll(md, "\n\n\n", "\n\n")
	}

	// Trim leading/trailing whitespace
	md = strings.TrimSpace(md)

	return md
}

// extractTitleFromHTML extracts the title from HTML content.
func extractTitleFromHTML(html string) string {
	// Try <title> tag
	if idx := strings.Index(html, "<title>"); idx != -1 {
		end := strings.Index(html[idx:], "</title>")
		if end != -1 {
			title := html[idx+7 : idx+end]
			title = strings.TrimSpace(title)
			// Remove common suffixes like " | Site Name"
			if pipeIdx := strings.LastIndex(title, " | "); pipeIdx > 0 {
				title = title[:pipeIdx]
			}
			if dashIdx := strings.LastIndex(title, " - "); dashIdx > 0 && dashIdx > len(title)/2 {
				title = title[:dashIdx]
			}

			return title
		}
	}

	// Try <h1> tag
	if idx := strings.Index(html, "<h1"); idx != -1 {
		// Find closing >
		closeTag := strings.Index(html[idx:], ">")
		if closeTag != -1 {
			start := idx + closeTag + 1
			end := strings.Index(html[start:], "</h1>")
			if end != -1 {
				title := html[start : start+end]
				// Strip any inner tags
				title = stripHTMLTags(title)

				return strings.TrimSpace(title)
			}
		}
	}

	return ""
}

// stripHTMLTags removes HTML tags from a string.
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// urlToPath converts a URL to a filesystem-safe path.
func urlToPath(u *url.URL) string {
	path := u.Path

	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// If empty, use index
	if path == "" {
		path = "index"
	}

	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	// Ensure .md extension
	if !strings.HasSuffix(path, ".md") {
		// Remove .html/.htm if present
		path = strings.TrimSuffix(path, ".html")
		path = strings.TrimSuffix(path, ".htm")
		path = path + ".md"
	}

	// Replace any remaining problematic characters
	path = strings.ReplaceAll(path, "?", "-")
	path = strings.ReplaceAll(path, "&", "-")
	path = strings.ReplaceAll(path, "=", "-")

	return path
}
