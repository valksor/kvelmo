package library

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestPullURL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
<h1>Hello World</h1>
<p>This is a test page.</p>
</body>
</html>`))
	}))
	defer server.Close()

	page, err := PullURL(context.Background(), server.URL+"/docs/intro", 0, "")
	if err != nil {
		t.Fatalf("PullURL failed: %v", err)
	}

	if page.Title != "Test Page" {
		t.Errorf("Title = %q, want %q", page.Title, "Test Page")
	}

	if page.Path != "docs/intro.md" {
		t.Errorf("Path = %q, want %q", page.Path, "docs/intro.md")
	}

	if page.URL != server.URL+"/docs/intro" {
		t.Errorf("URL = %q, want %q", page.URL, server.URL+"/docs/intro")
	}
}

func TestPullURL_AuthRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := PullURL(context.Background(), server.URL, 0, "")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	// Error should contain helpful message
	if !contains(err.Error(), "authentication required") {
		t.Errorf("error should mention authentication, got: %v", err)
	}
}

func TestPullURL_NonTextContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47})
	}))
	defer server.Close()

	_, err := PullURL(context.Background(), server.URL, 0, "")
	if err == nil {
		t.Fatal("expected error for non-text content")
	}
}

func TestPullURL_SizeLimit(t *testing.T) {
	largeContent := make([]byte, 2000)
	for i := range largeContent {
		largeContent[i] = 'x'
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(largeContent)
	}))
	defer server.Close()

	_, err := PullURL(context.Background(), server.URL, 1000, "")
	if err == nil {
		t.Fatal("expected error for oversized page")
	}
}

func TestIsTextContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"text/plain", true},
		{"text/markdown", true},
		{"application/json", true},
		{"image/png", false},
		{"application/octet-stream", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			got := isTextContentType(tt.contentType)
			if got != tt.want {
				t.Errorf("isTextContentType(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestIsJSRendered(t *testing.T) {
	tests := []struct {
		name string
		html string
		want bool
	}{
		{
			name: "normal html",
			html: `<html><body><h1>Title</h1><p>Content</p></body></html>`,
			want: false,
		},
		{
			name: "react app empty",
			html: `<html><body><div id="root"></div><script src="app.js"></script></body></html>`,
			want: true,
		},
		{
			name: "react app with content",
			html: `<html><body><div id="root"></div><p>Some server-rendered content</p></body></html>`,
			want: false,
		},
		{
			name: "next.js empty",
			html: `<html><body><div id="__next"></div></body></html>`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJSRendered(tt.html)
			if got != tt.want {
				t.Errorf("isJSRendered() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestURLToPath(t *testing.T) {
	tests := []struct {
		inputURL string
		want     string
	}{
		{"https://example.com/", "index.md"},
		{"https://example.com/docs/intro", "docs/intro.md"},
		{"https://example.com/docs/intro.html", "docs/intro.md"},
		{"https://example.com/api/v1/users", "api/v1/users.md"},
		{"https://example.com/search?q=test", "search.md"}, // Query params ignored for cleaner paths
	}

	for _, tt := range tests {
		t.Run(tt.inputURL, func(t *testing.T) {
			u, _ := parseURL(tt.inputURL)
			got := urlToPath(u)
			if got != tt.want {
				t.Errorf("urlToPath(%q) = %q, want %q", tt.inputURL, got, tt.want)
			}
		})
	}
}

func TestExtractTitleFromHTML(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "title tag",
			html: `<html><head><title>Page Title</title></head></html>`,
			want: "Page Title",
		},
		{
			name: "title with suffix",
			html: `<html><head><title>Page Title | Site Name</title></head></html>`,
			want: "Page Title",
		},
		{
			name: "h1 tag",
			html: `<html><body><h1>Main Heading</h1></body></html>`,
			want: "Main Heading",
		},
		{
			name: "h1 with class",
			html: `<html><body><h1 class="title">Styled Heading</h1></body></html>`,
			want: "Styled Heading",
		},
		{
			name: "no title",
			html: `<html><body><p>No heading here</p></body></html>`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitleFromHTML(tt.html)
			if got != tt.want {
				t.Errorf("extractTitleFromHTML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanMarkdown(t *testing.T) {
	input := "# Title\n\n\n\nContent\n\n\n\nMore content\n\n\n"
	want := "# Title\n\nContent\n\nMore content"

	got := cleanMarkdown(input)
	if got != want {
		t.Errorf("cleanMarkdown() =\n%q\nwant\n%q", got, want)
	}
}

// Helper functions.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

func parseURL(s string) (*url.URL, error) {
	return url.Parse(s)
}
