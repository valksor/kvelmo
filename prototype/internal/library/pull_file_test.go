package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPullFile_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a markdown file
	mdPath := filepath.Join(tmpDir, "readme.md")
	content := "# Test\n\nThis is a test file."
	if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	pages, err := PullFile(mdPath, 0)
	if err != nil {
		t.Fatalf("PullFile failed: %v", err)
	}

	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}

	page := pages[0]
	if page.Title != "Test" {
		t.Errorf("Title = %q, want %q", page.Title, "Test")
	}
	if page.Content != content {
		t.Errorf("Content mismatch")
	}
}

func TestPullFile_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	files := map[string]string{
		"readme.md":         "# Main\n\nreadme.",
		"guide/intro.md":    "# Introduction\n\nIntro guide.",
		"guide/advanced.md": "# Advanced\n\nguide.",
		"data.json":         `{"key": "value"}`,
		".hidden.md":        "# Hidden\n\nShould be skipped.",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	pages, err := PullFile(tmpDir, 0)
	if err != nil {
		t.Fatalf("PullFile failed: %v", err)
	}

	// Should get 4 files (not the hidden one)
	if len(pages) != 4 {
		t.Errorf("expected 4 pages, got %d", len(pages))
		for _, p := range pages {
			t.Logf("  - %s", p.Path)
		}
	}

	// Check that hidden file is not included
	for _, p := range pages {
		if p.Path == ".hidden.md" {
			t.Error("hidden file should not be included")
		}
	}
}

func TestPullFile_BinarySkipped(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a PNG file (binary)
	pngPath := filepath.Join(tmpDir, "image.png")
	pngContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG magic bytes
	if err := os.WriteFile(pngPath, pngContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a markdown file
	mdPath := filepath.Join(tmpDir, "readme.md")
	if err := os.WriteFile(mdPath, []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}

	pages, err := PullFile(tmpDir, 0)
	if err != nil {
		t.Fatalf("PullFile failed: %v", err)
	}

	// Should only get the markdown file
	if len(pages) != 1 {
		t.Errorf("expected 1 page (binary skipped), got %d", len(pages))
	}

	if len(pages) > 0 && pages[0].Path != "readme.md" {
		t.Errorf("expected readme.md, got %s", pages[0].Path)
	}
}

func TestPullFile_SizeLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a large file
	largePath := filepath.Join(tmpDir, "large.md")
	largeContent := make([]byte, 2000)
	for i := range largeContent {
		largeContent[i] = 'x'
	}
	if err := os.WriteFile(largePath, largeContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a small file
	smallPath := filepath.Join(tmpDir, "small.md")
	if err := os.WriteFile(smallPath, []byte("# Small"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pull with 1KB limit
	pages, err := PullFile(tmpDir, 1024)
	if err != nil {
		t.Fatalf("PullFile failed: %v", err)
	}

	// Should only get the small file
	if len(pages) != 1 {
		t.Errorf("expected 1 page (large skipped), got %d", len(pages))
	}
}

func TestIsBinaryContent(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    bool
	}{
		{"text", []byte("hello world"), false},
		{"markdown", []byte("# Heading\n\nParagraph"), false},
		{"null byte", []byte("hello\x00world"), true},
		{"png magic", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, true},
		{"jpeg magic", []byte{0xFF, 0xD8, 0xFF, 0xE0}, true},
		{"pdf magic", []byte{0x25, 0x50, 0x44, 0x46, 0x2D}, true},
		{"empty", []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinaryContent(tt.content)
			if got != tt.want {
				t.Errorf("isBinaryContent(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		filePath string
		want     string
	}{
		{
			name:     "markdown heading",
			content:  "# Hello World\n\nContent here.",
			filePath: "test.md",
			want:     "Hello World",
		},
		{
			name:     "html title",
			content:  "<html><head><title>Page Title</title></head></html>",
			filePath: "test.html",
			want:     "Page Title",
		},
		{
			name:     "no heading",
			content:  "Just some content without a heading.",
			filePath: "my-document.md",
			want:     "My Document",
		},
		{
			name:     "underscore filename",
			content:  "Content",
			filePath: "user_guide.md",
			want:     "User Guide",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.content, tt.filePath)
			if got != tt.want {
				t.Errorf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnsureMarkdownPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"guide/intro.html", "guide/intro.md"},
		{"guide/intro.htm", "guide/intro.md"},
		{"guide/intro.md", "guide/intro.md"},
		{"guide/intro.txt", "guide/intro.txt"},
		{"guide/intro.json", "guide/intro.json"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ensureMarkdownPath(tt.input)
			if got != tt.want {
				t.Errorf("ensureMarkdownPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
