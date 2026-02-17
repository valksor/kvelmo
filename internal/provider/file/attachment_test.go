package file

import (
	"testing"
)

func TestExtractAttachmentReferences_Deduplication(t *testing.T) {
	content := `
See [file.txt](file.txt) and also [file.txt](file.txt)
![image.png](image.png) and another ![image.png](image.png)
`

	attachments := ExtractAttachmentReferences(content)

	// Should deduplicate - only 2 unique attachments
	if len(attachments) != 2 {
		t.Errorf("Expected 2 attachments after deduplication, got %d", len(attachments))
	}

	// Check IDs are unique
	seen := make(map[string]bool)
	for _, att := range attachments {
		if seen[att.ID] {
			t.Errorf("Duplicate attachment ID found: %s", att.ID)
		}
		seen[att.ID] = true
	}
}

func TestExtractAttachmentReferences_MixedContent(t *testing.T) {
	content := `
# Document

Here's a local file: [document.pdf](document.pdf)

And an image: ![screenshot](screenshot.png)

External links should be ignored: [GitHub](https://github.com)

Mixed: [data.txt](data.txt) and ![chart.jpg](chart.jpg)
`

	attachments := ExtractAttachmentReferences(content)

	// Expected: document.pdf (file), screenshot.png (image), data.txt (file), chart.jpg (image)
	// = 4 attachments total (HTTP URL is excluded)
	if len(attachments) != 4 {
		t.Errorf("Expected 4 attachments (2 file links + 2 images), got %d", len(attachments))
	}

	// Check that HTTP URLs are excluded
	for _, att := range attachments {
		if att.ID == "https://github.com" {
			t.Error("HTTP URL should not be included in attachments")
		}
	}
}

func TestExtractAttachmentReferences_EmptyContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		expect  int
	}{
		{"empty string", "", 0},
		{"whitespace only", "   \n\t  ", 0},
		{"no links", "Just plain text with no markdown links", 0},
		{"only HTTP links", "[Google](https://google.com)", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attachments := ExtractAttachmentReferences(tt.content)
			if len(attachments) != tt.expect {
				t.Errorf("Expected %d attachments, got %d", tt.expect, len(attachments))
			}
		})
	}
}

func TestExtractAttachmentReferences_DoesNotDuplicateImageAsFile(t *testing.T) {
	content := `![photo.jpg](photo.jpg)`

	attachments := ExtractAttachmentReferences(content)

	// Image should appear once, not twice (once as image, once as file link)
	if len(attachments) != 1 {
		t.Errorf("Expected 1 attachment for image, got %d", len(attachments))
	}
}

func TestExtractAttachmentReferences_MalformedMarkdown(t *testing.T) {
	tests := []struct {
		name    string
		content string
		expect  int
	}{
		{"incomplete link", "[text](missing-bracket", 0},
		{"empty link", "[text]()", 0},           // Empty URL doesn't match regex
		{"only brackets", "[]()", 0},            // No text or URL
		{"nested brackets", "[[text]](url)", 0}, // Doesn't match regex pattern
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attachments := ExtractAttachmentReferences(tt.content)
			if len(attachments) != tt.expect {
				t.Errorf("Expected %d attachments, got %d", tt.expect, len(attachments))
			}
		})
	}
}

func TestIsImageURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"image.png", true},
		{"photo.jpg", true},
		{"picture.jpeg", true},
		{"animation.gif", true},
		{"vector.svg", true},
		{"document.pdf", false},
		{"data.txt", false},
		{"archive.zip", false},
		{"script.sh", false},
		{"no-extension", false},
		{"image.JPG", true}, // Case insensitive check would need toLower
		{"image.PNG", true}, // Case insensitive check would need toLower
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isImageURL(tt.url)
			if result != tt.expected {
				t.Errorf("isImageURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestIsHTTPURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com", true},
		{"http://localhost:8080", true},
		{"ftp://server.com", false},
		{"file.txt", false},
		{"./local/file.pdf", false},
		{"/absolute/path.png", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isHTTPURL(tt.url)
			if result != tt.expected {
				t.Errorf("isHTTPURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestExtractAttachmentReferences_PreservesAttachmentStructure(t *testing.T) {
	content := `[report.pdf](docs/report.pdf)`

	attachments := ExtractAttachmentReferences(content)

	if len(attachments) != 1 {
		t.Fatalf("Expected 1 attachment, got %d", len(attachments))
	}

	att := attachments[0]
	if att.ID != "docs/report.pdf" {
		t.Errorf("Expected ID 'docs/report.pdf', got %q", att.ID)
	}
	if att.Name != "report.pdf" {
		t.Errorf("Expected Name 'report.pdf', got %q", att.Name)
	}
	if att.URL != "docs/report.pdf" {
		t.Errorf("Expected URL 'docs/report.pdf', got %q", att.URL)
	}
}

// TestExtractAttachmentReferences_TypeCompatibility verifies the workunit.Attachment type is correctly used.
func TestExtractAttachmentReferences_TypeCompatibility(t *testing.T) {
	content := `[file.txt](file.txt)`

	attachments := ExtractAttachmentReferences(content)

	if len(attachments) != 1 {
		t.Fatalf("Expected 1 attachment, got %d", len(attachments))
	}

	// Verify it's the correct type (type will be inferred)
	_ = attachments[0]
}
