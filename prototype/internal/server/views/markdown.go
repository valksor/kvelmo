package views

import (
	"bytes"
	"regexp"
	"strings"
	"unicode"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	mdParser  goldmark.Markdown
	sanitizer *bluemonday.Policy
)

func init() {
	// Initialize goldmark with GitHub Flavored Markdown extensions
	mdParser = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Allow raw HTML, we sanitize with bluemonday
		),
	)

	// Initialize sanitizer with safe defaults for user-generated content
	sanitizer = bluemonday.UGCPolicy()
	// Allow code highlighting classes
	sanitizer.AllowAttrs("class").Matching(regexp.MustCompile(`^language-[\w-]+$`)).OnElements("code")
}

// RenderMarkdown converts markdown to sanitized HTML.
// Returns sanitized HTML that is safe to embed in templates.
func RenderMarkdown(markdown string) (string, error) {
	var buf bytes.Buffer
	if err := mdParser.Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}

	return sanitizer.Sanitize(buf.String()), nil
}

// ExtractShortDescription extracts a short plain-text description from markdown content.
// It strips markdown syntax, skips headers, and returns the first meaningful paragraph
// truncated to maxLen characters with ellipsis if needed.
func ExtractShortDescription(content string, maxLen int) string {
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	var paragraphLines []string
	inCodeBlock := false
	inFrontmatter := false
	frontmatterEnded := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track frontmatter state and skip horizontal rules
		if isHorizontalRule(trimmed) {
			if !frontmatterEnded {
				// Only --- can be a frontmatter delimiter
				if trimmed == "---" {
					if i == 0 {
						// Start of frontmatter
						inFrontmatter = true

						continue
					}
					if inFrontmatter {
						// End of frontmatter
						inFrontmatter = false
						frontmatterEnded = true

						continue
					}
				}
			}
			// Skip all horizontal rules (---, ***, ___, - - -, etc.)
			continue
		}
		if inFrontmatter {
			continue
		}

		// Track code block state
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock

			continue
		}
		if inCodeBlock {
			continue
		}

		// Skip headers
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Skip empty lines (but stop if we have content)
		if trimmed == "" {
			if len(paragraphLines) > 0 {
				break // End of first paragraph
			}

			continue
		}

		// Skip list markers, blockquotes for cleaner preview
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") ||
			strings.HasPrefix(trimmed, "> ") || regexp.MustCompile(`^\d+\.\s`).MatchString(trimmed) {
			// Include list content but strip marker
			trimmed = regexp.MustCompile(`^[-*>]\s+|^\d+\.\s+`).ReplaceAllString(trimmed, "")
		}

		paragraphLines = append(paragraphLines, trimmed)
	}

	if len(paragraphLines) == 0 {
		return ""
	}

	// Join lines with spaces
	text := strings.Join(paragraphLines, " ")

	// Strip inline markdown syntax
	text = stripInlineMarkdown(text)

	// Collapse multiple spaces
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Truncate if needed
	if len(text) > maxLen {
		// Find last space before maxLen to avoid cutting words
		cutoff := maxLen - 3 // Room for "..."
		for cutoff > 0 && !unicode.IsSpace(rune(text[cutoff])) {
			cutoff--
		}
		if cutoff <= 0 {
			cutoff = maxLen - 3
		}
		text = strings.TrimSpace(text[:cutoff]) + "..."
	}

	return text
}

// stripInlineMarkdown removes common inline markdown syntax.
func stripInlineMarkdown(text string) string {
	// Bold: **text** or __text__
	text = regexp.MustCompile(`\*\*(.+?)\*\*`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`__(.+?)__`).ReplaceAllString(text, "$1")

	// Italic: *text* or _text_
	text = regexp.MustCompile(`\*(.+?)\*`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`_(.+?)_`).ReplaceAllString(text, "$1")

	// Inline code: `text`
	text = regexp.MustCompile("`(.+?)`").ReplaceAllString(text, "$1")

	// Images: ![alt](url) - MUST come before links since links pattern matches ![]() partially
	text = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`).ReplaceAllString(text, "$1")

	// Links: [text](url)
	text = regexp.MustCompile(`\[(.+?)\]\(.+?\)`).ReplaceAllString(text, "$1")

	// Strikethrough: ~~text~~
	text = regexp.MustCompile(`~~(.+?)~~`).ReplaceAllString(text, "$1")

	return text
}

// isHorizontalRule returns true if the line is a markdown horizontal rule.
// Markdown allows ---, ***, and ___ (3+ of same char, optionally with spaces).
func isHorizontalRule(s string) bool {
	s = strings.ReplaceAll(s, " ", "") // Remove spaces
	if len(s) < 3 {
		return false
	}

	return (strings.Count(s, "-") == len(s)) ||
		(strings.Count(s, "*") == len(s)) ||
		(strings.Count(s, "_") == len(s))
}
