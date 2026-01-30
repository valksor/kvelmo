// Package wrike provides Wrike-specific utilities for handling files and attachments.
package wrike

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// attachmentPattern matches attachment://ID tokens in text.
	attachmentPattern = regexp.MustCompile(`attachment://([A-Za-z0-9]+)`)

	// htmlPatterns for HTML to text conversion.
	htmlPatternBR       = regexp.MustCompile(`(?i)<br\s*/?>`)
	htmlPatternScript   = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
	htmlPatternStyle    = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style>`)
	htmlPatternTags     = regexp.MustCompile(`(?s)<[^>]+>`)
	htmlPatternNewlines = regexp.MustCompile(`\n{3,}`)
)

// SanitizeFilename cleans a filename for safe filesystem use.
// Replaces slashes and backslashes with underscores.
func SanitizeFilename(name string) string {
	cleaned := strings.ReplaceAll(name, "/", "_")
	cleaned = strings.ReplaceAll(cleaned, "\\", "_")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "attachment"
	}

	return cleaned
}

// NextAvailablePath finds the next available path that doesn't exist.
// If the destination exists, appends _1, _2, etc.
func NextAvailablePath(dest string) string {
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest
	}

	ext := filepath.Ext(dest)
	stem := dest[:len(dest)-len(ext)]
	counter := 1
	for {
		candidate := fmt.Sprintf("%s_%d%s", stem, counter, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
		counter++
	}
}

// DownloadAttachment downloads a single attachment and saves it to the destination directory.
// Returns the path where the file was saved.
func DownloadAttachment(ctx context.Context, client *Client, attachment Attachment, destDir string) (string, error) {
	if attachment.ID == "" {
		return "", errors.New("attachment missing ID")
	}

	// Sanitize filename
	rawName := attachment.Name
	if rawName == "" {
		rawName = attachment.ID
	}
	filename := SanitizeFilename(rawName)

	// Create destination directory
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	// Find available path (handle duplicates)
	destPath := filepath.Join(destDir, filename)
	destPath = NextAvailablePath(destPath)

	// Download attachment
	reader, contentDisposition, err := client.DownloadAttachment(ctx, attachment.ID)
	if err != nil {
		return "", fmt.Errorf("download attachment %s: %w", attachment.ID, err)
	}
	defer func() { _ = reader.Close() }()

	// Try to get filename from Content-Disposition header
	if contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err == nil {
			if filename := params["filename"]; filename != "" {
				sanitized := SanitizeFilename(filename)
				destPath = filepath.Join(destDir, sanitized)
				destPath = NextAvailablePath(destPath)
			}
		}
	}

	// Write to file
	file, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, reader); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return destPath, nil
}

// DownloadAttachments downloads multiple attachments and saves them to the destination directory.
// Returns a map of attachment ID to local file path.
func DownloadAttachments(ctx context.Context, client *Client, attachments []Attachment, destDir string) (map[string]string, error) {
	paths := make(map[string]string)

	for _, attachment := range attachments {
		if attachment.ID == "" {
			continue
		}

		localPath, err := DownloadAttachment(ctx, client, attachment, destDir)
		if err != nil {
			// Log but continue - partial success is better than total failure
			continue
		}

		paths[attachment.ID] = localPath
	}

	return paths, nil
}

// ReplaceAttachmentTokens replaces attachment://ID tokens with local file paths.
func ReplaceAttachmentTokens(text string, attachments map[string]string) string {
	if text == "" {
		return ""
	}

	result := attachmentPattern.ReplaceAllStringFunc(text, func(match string) string {
		// Extract ID from match (strip "attachment://" prefix)
		id := strings.TrimPrefix(match, "attachment://")
		if localPath, ok := attachments[id]; ok {
			return localPath
		}

		return match // Keep original if not found
	})

	return result
}

// HTMLToText converts HTML content to plain text.
// Handles <br>, removes <script> and <style> tags, strips other tags, normalizes whitespace.
func HTMLToText(html string) string {
	if html == "" {
		return ""
	}

	// Convert <br> to newlines
	text := htmlPatternBR.ReplaceAllString(html, "\n")

	// Remove script blocks
	text = htmlPatternScript.ReplaceAllString(text, "")

	// Remove style blocks
	text = htmlPatternStyle.ReplaceAllString(text, "")

	// Remove all remaining HTML tags
	text = htmlPatternTags.ReplaceAllString(text, "")

	// Normalize excessive newlines (3+ to 2)
	text = htmlPatternNewlines.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// ResolveAuthor extracts author name from a comment.
// Handles various field names and nested structures.
func ResolveAuthor(comment Comment) string {
	if comment.AuthorName != "" {
		return comment.AuthorName
	}

	// Could be extended to handle createdBy/author/user fields
	// For now, just return the ID if name is not available
	if comment.AuthorID != "" {
		return comment.AuthorID
	}

	return ""
}

// DownloadAttachmentBytes downloads an attachment and returns it as bytes.
// Useful for tests or when you need the attachment in memory.
func DownloadAttachmentBytes(ctx context.Context, client *Client, attachmentID string) ([]byte, string, error) {
	reader, _, err := client.DownloadAttachment(ctx, attachmentID)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = reader.Close() }()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, "", fmt.Errorf("read attachment: %w", err)
	}

	return buf.Bytes(), "", nil
}

// GetContentType detects the content type of a file based on its extension.
func GetContentType(filename string) string {
	ext := filepath.Ext(filename)
	switch strings.ToLower(ext) {
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".csv":
		return "text/csv"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}
