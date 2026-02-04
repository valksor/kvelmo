package views

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		contains []string // substrings that should be present
		excludes []string // substrings that should NOT be present (XSS)
	}{
		{
			name:     "simple paragraph",
			markdown: "Hello world",
			contains: []string{"<p>Hello world</p>"},
		},
		{
			name:     "bold text",
			markdown: "**bold text**",
			contains: []string{"<strong>bold text</strong>"},
		},
		{
			name:     "italic text",
			markdown: "*italic*",
			contains: []string{"<em>italic</em>"},
		},
		{
			name:     "code block",
			markdown: "```go\nfunc main() {}\n```",
			contains: []string{"<code", "func main()"},
		},
		{
			name:     "inline code",
			markdown: "Use `fmt.Println`",
			contains: []string{"<code>fmt.Println</code>"},
		},
		{
			name:     "link",
			markdown: "[example](https://example.com)",
			contains: []string{`<a href="https://example.com"`, ">example</a>"},
		},
		{
			name:     "header",
			markdown: "# Title",
			contains: []string{"<h1", "Title", "</h1>"},
		},
		{
			name:     "list",
			markdown: "- Item 1\n- Item 2",
			contains: []string{"<ul>", "<li>Item 1</li>", "<li>Item 2</li>", "</ul>"},
		},
		{
			name:     "task list",
			markdown: "- [ ] Todo\n- [x] Done",
			contains: []string{"<li>", "Todo", "Done"}, // GFM extension renders task lists
		},
		{
			name:     "table",
			markdown: "| A | B |\n|---|---|\n| 1 | 2 |",
			contains: []string{"<table>", "<th>A</th>", "<td>1</td>"},
		},
		{
			name:     "xss script tag stripped",
			markdown: "<script>alert('xss')</script>",
			excludes: []string{"<script>", "alert"},
		},
		{
			name:     "xss onclick stripped",
			markdown: `<a href="#" onclick="alert('xss')">Click</a>`,
			contains: []string{"Click"},                  // Text preserved
			excludes: []string{"onclick", "alert('xss'"}, // XSS attributes stripped
		},
		{
			name:     "xss img onerror stripped",
			markdown: `<img src="x" onerror="alert('xss')">`,
			excludes: []string{"onerror"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderMarkdown(tt.markdown)
			if err != nil {
				t.Fatalf("RenderMarkdown() error = %v", err)
			}

			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("RenderMarkdown() = %q, want to contain %q", got, want)
				}
			}

			for _, exclude := range tt.excludes {
				if strings.Contains(got, exclude) {
					t.Errorf("RenderMarkdown() = %q, should NOT contain %q (XSS)", got, exclude)
				}
			}
		})
	}
}

func TestExtractShortDescription(t *testing.T) {
	tests := []struct {
		name    string
		content string
		maxLen  int
		want    string
	}{
		{
			name:    "simple text",
			content: "This is a simple description.",
			maxLen:  100,
			want:    "This is a simple description.",
		},
		{
			name:    "strips header and gets first paragraph",
			content: "# Title\n\nThis is the description paragraph.\n\nAnother paragraph.",
			maxLen:  100,
			want:    "This is the description paragraph.",
		},
		{
			name:    "strips multiple headers",
			content: "# Title\n## Subtitle\n\nActual content here.",
			maxLen:  100,
			want:    "Actual content here.",
		},
		{
			name:    "truncates long text",
			content: "This is a very long description that should be truncated at the word boundary.",
			maxLen:  30,
			want:    "This is a very long...",
		},
		{
			name:    "strips bold",
			content: "This has **bold text** inside.",
			maxLen:  100,
			want:    "This has bold text inside.",
		},
		{
			name:    "strips italic",
			content: "This has *italic* and _also italic_ text.",
			maxLen:  100,
			want:    "This has italic and also italic text.",
		},
		{
			name:    "strips inline code",
			content: "Use the `fmt.Println` function.",
			maxLen:  100,
			want:    "Use the fmt.Println function.",
		},
		{
			name:    "strips links",
			content: "Check out [this link](https://example.com) for more.",
			maxLen:  100,
			want:    "Check out this link for more.",
		},
		{
			name:    "skips code blocks",
			content: "Intro text.\n\n```go\ncode here\n```\n\nMore text.",
			maxLen:  100,
			want:    "Intro text.",
		},
		{
			name:    "handles frontmatter",
			content: "---\ntitle: Test\n---\n\nActual description.",
			maxLen:  100,
			want:    "Actual description.",
		},
		{
			name:    "horizontal rule after frontmatter",
			content: "---\ntitle: Test\n---\n\nDescription.\n\n---\n\nMore content.",
			maxLen:  100,
			want:    "Description.",
		},
		{
			name:    "horizontal rule without frontmatter",
			content: "# Title\n\n---\n\nDescription after rule.",
			maxLen:  100,
			want:    "Description after rule.",
		},
		{
			name:    "asterisk horizontal rule",
			content: "# Title\n\n***\n\nDescription after rule.",
			maxLen:  100,
			want:    "Description after rule.",
		},
		{
			name:    "underscore horizontal rule",
			content: "# Title\n\n___\n\nDescription after rule.",
			maxLen:  100,
			want:    "Description after rule.",
		},
		{
			name:    "horizontal rule with spaces",
			content: "# Title\n\n- - -\n\nDescription after rule.",
			maxLen:  100,
			want:    "Description after rule.",
		},
		{
			name:    "handles list items",
			content: "# Title\n\n- First item\n- Second item",
			maxLen:  100,
			want:    "First item Second item",
		},
		{
			name:    "empty content",
			content: "",
			maxLen:  100,
			want:    "",
		},
		{
			name:    "only headers",
			content: "# Title\n## Subtitle",
			maxLen:  100,
			want:    "",
		},
		{
			name:    "collapses whitespace",
			content: "Text with   multiple    spaces.",
			maxLen:  100,
			want:    "Text with multiple spaces.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractShortDescription(tt.content, tt.maxLen)
			if got != tt.want {
				t.Errorf("ExtractShortDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripInlineMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bold double asterisk", "**bold**", "bold"},
		{"bold underscore", "__bold__", "bold"},
		{"italic asterisk", "*italic*", "italic"},
		{"italic underscore", "_italic_", "italic"},
		{"inline code", "`code`", "code"},
		{"link", "[text](url)", "text"},
		{"image", "![alt](http://example.com/img.png)", "alt"},
		{"strikethrough", "~~deleted~~", "deleted"},
		{"combined", "**bold** and *italic*", "bold and italic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripInlineMarkdown(tt.input)
			if got != tt.want {
				t.Errorf("stripInlineMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsHorizontalRule(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"three dashes", "---", true},
		{"three asterisks", "***", true},
		{"three underscores", "___", true},
		{"dashes with spaces", "- - -", true},
		{"asterisks with spaces", "* * *", true},
		{"underscores with spaces", "_ _ _", true},
		{"many dashes", "----------", true},
		{"two dashes", "--", false},
		{"two asterisks", "**", false},
		{"text", "hello", false},
		{"mixed chars", "-*-", false},
		{"empty", "", false},
		{"header marker", "###", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHorizontalRule(tt.input)
			if got != tt.want {
				t.Errorf("isHorizontalRule(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
