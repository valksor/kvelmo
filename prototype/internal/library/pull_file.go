package library

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// textExtensions are file extensions considered text/documentation.
var textExtensions = map[string]bool{
	".md":       true,
	".markdown": true,
	".txt":      true,
	".rst":      true,
	".html":     true,
	".htm":      true,
	".json":     true,
	".yaml":     true,
	".yml":      true,
	".xml":      true,
	".adoc":     true,
	".asciidoc": true,
}

// binaryMagicBytes are file signatures for common binary formats.
var binaryMagicBytes = [][]byte{
	{0x89, 0x50, 0x4E, 0x47}, // PNG
	{0xFF, 0xD8, 0xFF},       // JPEG
	{0x47, 0x49, 0x46},       // GIF
	{0x25, 0x50, 0x44, 0x46}, // PDF
	{0x50, 0x4B, 0x03, 0x04}, // ZIP/JAR
	{0x1F, 0x8B},             // GZIP
}

// PullFile pulls documentation from a local file or directory.
func PullFile(sourcePath string, maxPageSize int64) ([]*CrawledPage, error) {
	sourcePath = filepath.Clean(sourcePath)

	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("stat source: %w", err)
	}

	if info.IsDir() {
		return pullDirectory(sourcePath, maxPageSize)
	}

	page, err := pullSingleFile(sourcePath, "", maxPageSize)
	if err != nil {
		return nil, err
	}
	if page == nil {
		return nil, fmt.Errorf("file %s is not a text file", sourcePath)
	}

	return []*CrawledPage{page}, nil
}

// pullDirectory recursively pulls all text files from a directory.
func pullDirectory(rootPath string, maxPageSize int64) ([]*CrawledPage, error) {
	var pages []*CrawledPage

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}

		page, err := pullSingleFile(path, relPath, maxPageSize)
		if err != nil {
			// Non-fatal: skip files that fail to read
			return nil //nolint:nilerr // Intentionally skip problematic files
		}
		if page != nil {
			pages = append(pages, page)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}

	return pages, nil
}

// pullSingleFile reads a single file and returns a CrawledPage.
// Returns nil, nil if the file is binary or should be skipped (not an error condition).
func pullSingleFile(filePath, relPath string, maxPageSize int64) (*CrawledPage, error) {
	// Check extension first
	ext := strings.ToLower(filepath.Ext(filePath))
	if !textExtensions[ext] {
		// Unknown extension - check content
		if isBinaryFile(filePath) {
			return nil, nil //nolint:nilnil // Binary files are skipped, not errors
		}
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	// Skip files that are too large
	if maxPageSize > 0 && info.Size() > maxPageSize {
		return nil, nil //nolint:nilnil // Oversized files are skipped, not errors
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Double-check for binary content
	if isBinaryContent(content) {
		return nil, nil //nolint:nilnil // Binary content is skipped, not errors
	}

	// Use relative path or filename
	if relPath == "" {
		relPath = filepath.Base(filePath)
	}

	// Normalize path to use forward slashes and .md extension
	relPath = filepath.ToSlash(relPath)
	if ext != ".md" && ext != ".markdown" {
		// Keep original extension but ensure it's recognizable
		relPath = ensureMarkdownPath(relPath)
	}

	// Extract title from content
	title := extractTitle(string(content), filePath)

	return &CrawledPage{
		Path:      relPath,
		Title:     title,
		Content:   string(content),
		SizeBytes: info.Size(),
	}, nil
}

// isBinaryFile checks if a file appears to be binary by reading its first bytes.
func isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true // Assume binary if we can't read
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return true
	}

	return isBinaryContent(buf[:n])
}

// isBinaryContent checks if content appears to be binary.
func isBinaryContent(content []byte) bool {
	// Check magic bytes
	for _, magic := range binaryMagicBytes {
		if len(content) >= len(magic) {
			matches := true
			for i, b := range magic {
				if content[i] != b {
					matches = false

					break
				}
			}
			if matches {
				return true
			}
		}
	}

	// Check for null bytes (common in binary files)
	for _, b := range content {
		if b == 0 {
			return true
		}
	}

	return false
}

// ensureMarkdownPath ensures the path has a recognizable extension.
// For HTML files, changes to .md; for others, keeps original extension.
func ensureMarkdownPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".html" || ext == ".htm" {
		return strings.TrimSuffix(path, ext) + ".md"
	}

	return path
}

// extractTitle extracts a title from content or derives it from the file path.
func extractTitle(content, filePath string) string {
	// Try to find markdown heading
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}

	// Try HTML title
	if idx := strings.Index(content, "<title>"); idx != -1 {
		if end := strings.Index(content[idx:], "</title>"); end != -1 {
			return content[idx+7 : idx+end]
		}
	}

	// Fall back to filename
	base := filepath.Base(filePath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	// Convert dashes/underscores to spaces and title case
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")

	return strings.Title(name) //nolint:staticcheck // Title is fine for simple conversion
}
