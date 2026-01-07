package wrike

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple filename",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "filename with slashes",
			input:    "path/to/document.pdf",
			expected: "path_to_document.pdf",
		},
		{
			name:     "filename with backslashes",
			input:    "path\\to\\document.pdf",
			expected: "path_to_document.pdf",
		},
		{
			name:     "filename with mixed slashes",
			input:    "path/to\\file.pdf",
			expected: "path_to_file.pdf",
		},
		{
			name:     "empty filename",
			input:    "",
			expected: "attachment",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "attachment",
		},
		{
			name:     "filename with leading/trailing spaces",
			input:    "  document.pdf  ",
			expected: "document.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNextAvailablePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Test non-existent path
	t.Run("path doesn't exist", func(t *testing.T) {
		path := filepath.Join(tmpDir, "newfile.txt")
		result := NextAvailablePath(path)
		if result != path {
			t.Errorf("NextAvailablePath(%q) = %q, want %q", path, result, path)
		}
	})

	// Test existing file - should add _1
	t.Run("file exists", func(t *testing.T) {
		path := filepath.Join(tmpDir, "exists.txt")
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}

		result := NextAvailablePath(path)
		expected := filepath.Join(tmpDir, "exists_1.txt")
		if result != expected {
			t.Errorf("NextAvailablePath(%q) = %q, want %q", path, result, expected)
		}
	})

	// Test multiple existing files - should find next available number
	t.Run("multiple files exist", func(t *testing.T) {
		basePath := filepath.Join(tmpDir, "count.txt")
		for i := range 3 {
			path := basePath
			if i > 0 {
				path = filepath.Join(tmpDir, "count_1.txt")
			}
			if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		result := NextAvailablePath(basePath)
		expected := filepath.Join(tmpDir, "count_2.txt")
		if result != expected {
			t.Errorf("NextAvailablePath(%q) = %q, want %q", basePath, result, expected)
		}
	})
}

func TestReplaceAttachmentTokens(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		attachments map[string]string
		expected    string
	}{
		{
			name:        "no tokens",
			text:        "This is plain text",
			attachments: map[string]string{"ABC123": "/path/to/file.pdf"},
			expected:    "This is plain text",
		},
		{
			name:        "single token replaced",
			text:        "See attachment://ABC123 for details",
			attachments: map[string]string{"ABC123": "/path/to/file.pdf"},
			expected:    "See /path/to/file.pdf for details",
		},
		{
			name: "multiple tokens replaced",
			text: "attachment://ABC123 and attachment://DEF456",
			attachments: map[string]string{
				"ABC123": "/path/one.pdf",
				"DEF456": "/path/two.pdf",
			},
			expected: "/path/one.pdf and /path/two.pdf",
		},
		{
			name:        "missing token stays as-is",
			text:        "See attachment:://NOTFOUND for details",
			attachments: map[string]string{"ABC123": "/path/to/file.pdf"},
			expected:    "See attachment:://NOTFOUND for details",
		},
		{
			name:        "empty text",
			text:        "",
			attachments: map[string]string{},
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplaceAttachmentTokens(tt.text, tt.attachments)
			if result != tt.expected {
				t.Errorf("ReplaceAttachmentTokens() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHTMLToText(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "plain text",
			html:     "Just plain text",
			expected: "Just plain text",
		},
		{
			name:     "br to newline",
			html:     "Line 1<br>Line 2<br/>Line 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "strip script tags",
			html:     "Before<script>console.log('test');</script>After",
			expected: "BeforeAfter",
		},
		{
			name:     "strip style tags",
			html:     "Before<style>.class { color: red; }</style>After",
			expected: "BeforeAfter",
		},
		{
			name:     "strip html tags",
			html:     "<p>Hello <strong>world</strong>!</p>",
			expected: "Hello world!",
		},
		{
			name:     "normalize multiple newlines",
			html:     "Line 1\n\n\n\nLine 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "complex html",
			html:     "<h1>Title</h1><p>Paragraph<br>Next line</p>",
			expected: "TitleParagraph\nNext line",
		},
		{
			name:     "empty html",
			html:     "",
			expected: "",
		},
		{
			name:     "html with entities",
			html:     "<p>Text &nbsp; here</p>",
			expected: "Text &nbsp; here", // We don't decode HTML entities (would need html.UnescapeString)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToText(tt.html)
			if result != tt.expected {
				t.Errorf("HTMLToText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestResolveAuthor(t *testing.T) {
	tests := []struct {
		name     string
		comment  Comment
		expected string
	}{
		{
			name: "author name present",
			comment: Comment{
				AuthorName: "John Doe",
				AuthorID:   "123",
			},
			expected: "John Doe",
		},
		{
			name: "only author ID",
			comment: Comment{
				AuthorID: "123",
			},
			expected: "123",
		},
		{
			name:     "empty comment",
			comment:  Comment{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveAuthor(tt.comment)
			if result != tt.expected {
				t.Errorf("ResolveAuthor() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"PDF", "document.pdf", "application/pdf"},
		{"PNG", "image.png", "image/png"},
		{"JPEG", "photo.jpg", "image/jpeg"},
		{"JPEG alt", "photo.jpeg", "image/jpeg"},
		{"GIF", "anim.gif", "image/gif"},
		{"Text", "file.txt", "text/plain"},
		{"HTML", "page.html", "text/html"},
		{"HTML alt", "page.htm", "text/html"},
		{"JSON", "data.json", "application/json"},
		{"XML", "data.xml", "application/xml"},
		{"ZIP", "archive.zip", "application/zip"},
		{"Unknown", "file.xyz", "application/octet-stream"},
		{"No extension", "README", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("GetContentType(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

// Integration test (would need mock Wrike client for full testing).
func TestDownloadAttachmentIntegration(t *testing.T) {
	t.Skip("needs mock Wrike client")

	// Example structure for when we have a mock:
	// ctx := context.Background()
	// mockClient := NewMockClient()
	// attachment := Attachment{ID: "ATT123", Name: "test.pdf"}
	//
	// tmpDir := t.TempDir()
	// path, err := DownloadAttachment(ctx, mockClient, attachment, tmpDir)
	//
	// if err != nil {
	//     t.Fatal(err)
	// }
	// if _, err := os.Stat(path); os.IsNotExist(err) {
	//     t.Errorf("file not created: %s", path)
	// }
}

func TestDownloadAttachmentSanitizePath(t *testing.T) {
	// Test that the function sanitizes paths correctly
	attachment := Attachment{
		ID:   "ATT123",
		Name: "path/to/file with spaces.pdf",
	}

	sanitized := SanitizeFilename(attachment.Name)
	if !strings.Contains(sanitized, "_") {
		t.Error("expected underscores in sanitized name (from slashes)")
	}
	if strings.Contains(sanitized, "/") {
		t.Error("sanitized name should not contain slashes")
	}
	// Spaces are preserved in sanitization (only slashes are replaced)
	if !strings.Contains(sanitized, " ") {
		t.Error("sanitized name should preserve spaces")
	}
}
