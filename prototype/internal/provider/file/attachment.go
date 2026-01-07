package file

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/valksor/go-mehrhof/internal/provider"
)

var (
	// imageRegex matches ![alt](path) for images.
	imageRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	// linkRegex matches [link](path) for file links.
	linkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	// httpRegex matches HTTP/HTTPS URLs.
	httpRegex = regexp.MustCompile(`^https?://`)
)

// ExtractAttachmentReferences parses markdown for local file links and image references.
// Returns attachments for non-HTTP URLs found in the markdown content.
func ExtractAttachmentReferences(content string) []provider.Attachment {
	var attachments []provider.Attachment
	seen := make(map[string]bool)

	// Extract images
	for _, match := range imageRegex.FindAllStringSubmatch(content, -1) {
		if len(match) < 3 {
			continue
		}
		url := match[2]
		if !isHTTPURL(url) && !seen[url] {
			seen[url] = true
			attachments = append(attachments, provider.Attachment{
				ID:   url,
				Name: filepath.Base(url),
				URL:  url,
			})
		}
	}

	// Extract file links (excluding HTTP URLs and images)
	for _, match := range linkRegex.FindAllStringSubmatch(content, -1) {
		if len(match) < 3 {
			continue
		}
		url := match[2]
		if !isHTTPURL(url) && !isImageURL(url) && !seen[url] {
			seen[url] = true
			attachments = append(attachments, provider.Attachment{
				ID:   url,
				Name: filepath.Base(url),
				URL:  url,
			})
		}
	}

	return attachments
}

// isHTTPURL checks if a URL is an HTTP/HTTPS URL.
func isHTTPURL(url string) bool {
	return httpRegex.MatchString(url)
}

// isImageURL checks if a URL has an image extension (case-insensitive).
func isImageURL(url string) bool {
	ext := strings.ToLower(filepath.Ext(url))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".bmp":
		return true
	default:
		return false
	}
}
