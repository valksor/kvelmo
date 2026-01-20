package conductor

import (
	"strings"
	"testing"
)

func TestParseSimplifiedSpecifications(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantCount    int
		wantFirstNum int
	}{
		{
			name: "single specification",
			content: `--- specification-1.md ---
This is the specification content.
--- end ---`,
			wantCount:    1,
			wantFirstNum: 1,
		},
		{
			name: "multiple specifications",
			content: `--- specification-1.md ---
First spec content.
--- end ---
--- specification-2.md ---
Second spec content.
--- end ---`,
			wantCount:    2,
			wantFirstNum: 1,
		},
		{
			name: "specification with newlines",
			content: `--- specification-1.md ---
Line 1
Line 2
Line 3
--- end ---`,
			wantCount:    1,
			wantFirstNum: 1,
		},
		{
			name:         "no specification markers - fallback",
			content:      `This is just plain text without markers.`,
			wantCount:    1,
			wantFirstNum: 1,
		},
		{
			name: "specification number 10",
			content: `--- specification-10.md ---
Content for spec 10.
--- end ---`,
			wantCount:    1,
			wantFirstNum: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			specs := parseSimplifiedSpecifications(tt.content)

			if len(specs) != tt.wantCount {
				t.Errorf("parseSimplifiedSpecifications() got %d specs, want %d", len(specs), tt.wantCount)

				return
			}

			if len(specs) > 0 && specs[0].Number != tt.wantFirstNum {
				t.Errorf("parseSimplifiedSpecifications() first spec number = %d, want %d", specs[0].Number, tt.wantFirstNum)
			}
		})
	}
}

func TestParseSimplifiedCode(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantFiles []string
		wantError bool
	}{
		{
			name: "single file",
			content: `--- main.go ---
package main

func main() {
	println("hello")
}
--- end ---`,
			wantCount: 1,
			wantFiles: []string{"main.go"},
			wantError: false,
		},
		{
			name: "multiple files",
			content: `--- main.go ---
package main
--- end ---
--- utils/helper.go ---
package utils
--- end ---`,
			wantCount: 2,
			wantFiles: []string{"main.go", "utils/helper.go"},
			wantError: false,
		},
		{
			name:      "no file markers",
			content:   `Just plain text without any markers.`,
			wantCount: 0,
			wantFiles: []string{},
			wantError: true,
		},
		{
			name: "file with spaces in path",
			content: `--- path/to/my file.go ---
content
--- end ---`,
			wantCount: 1,
			wantFiles: []string{"path/to/my file.go"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := parseSimplifiedCode(tt.content)

			if (err != nil) != tt.wantError {
				t.Errorf("parseSimplifiedCode() error = %v, wantError %v", err, tt.wantError)

				return
			}

			if len(files) != tt.wantCount {
				t.Errorf("parseSimplifiedCode() got %d files, want %d", len(files), tt.wantCount)

				return
			}

			for _, wantFile := range tt.wantFiles {
				if _, ok := files[wantFile]; !ok {
					t.Errorf("parseSimplifiedCode() missing file %s", wantFile)
				}
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PR Review Parsing Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestParsePRReviewEmpty tests parsing empty review content.
func TestParsePRReviewEmpty(t *testing.T) {
	review := parsePRReview("")

	if review == nil {
		t.Fatal("review is nil")
	}

	if review.Summary != "" {
		t.Errorf("summary: got %q, want empty", review.Summary)
	}

	if len(review.Issues) != 0 {
		t.Errorf("issues: got %d, want 0", len(review.Issues))
	}
}

// TestParseReviewIssuesDirect tests parseReviewIssues directly.
func TestParseReviewIssuesDirect(t *testing.T) {
	content := `## Issues

- [CRITICAL] [main.go:42] Test issue
- [HIGH] [util.go:10] Another issue
`

	issues := parseReviewIssues(content)
	t.Logf("Got %d issues", len(issues))
	for i, issue := range issues {
		t.Logf("Issue %d: file=%s line=%d severity=%s message=%s", i, issue.File, issue.Line, issue.Severity, issue.Message)
	}

	if len(issues) != 2 {
		t.Fatalf("got %d issues, want 2", len(issues))
	}
}

// TestParsePRReviewWithContent tests parsing review with content.
func TestParsePRReviewWithContent(t *testing.T) {
	content := `## Summary

This PR adds a new feature.

## Issues

### correctness

- [CRITICAL] [main.go:42] Missing error handling
- [HIGH] [util.go:10] Inefficient string concatenation
`

	review := parsePRReview(content)

	if review.Summary == "" {
		t.Error("summary is empty")
	}

	if len(review.Issues) != 2 {
		t.Fatalf("issues: got %d, want 2", len(review.Issues))
	}

	if review.Issues[0].Severity != "critical" {
		t.Errorf("issue 0 severity: got %s, want critical", review.Issues[0].Severity)
	}

	if review.Issues[0].File != "main.go" {
		t.Errorf("issue 0 file: got %s, want main.go", review.Issues[0].File)
	}

	if review.Issues[0].Line != 42 {
		t.Errorf("issue 0 line: got %d, want 42", review.Issues[0].Line)
	}
}

// TestExtractReviewSection tests extracting markdown sections.
func TestExtractReviewSection(t *testing.T) {
	content := `## Summary

This is the summary.

## Issues

Some issues here.

## Conclusion

Final thoughts.`

	section := extractReviewSection(content, "Summary")
	if section != "This is the summary." {
		t.Errorf("section: got %q, want %q", section, "This is the summary.")
	}
}

// TestExtractOverallAssessment tests extracting overall assessment.
func TestExtractOverallAssessment(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "approved",
			content:  "This PR looks good and is approved",
			expected: "approved",
		},
		{
			name:     "changes requested",
			content:  "This PR needs changes before merging",
			expected: "changes_requested",
		},
		{
			name:     "comment",
			content:  "Just leaving some comments",
			expected: "comment",
		},
		{
			name:     "no issues",
			content:  "The code has no issues",
			expected: "approved",
		},
		{
			name:     "empty",
			content:  "Just some text",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractOverallAssessment(tt.content)
			if result != tt.expected {
				t.Errorf("extractOverallAssessment(%q) = %q, want %q", tt.content, result, tt.expected)
			}
		})
	}
}

// TestParseReviewIssueLine tests parsing various issue line formats.
func TestParseReviewIssueLine(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		expectIssue    bool
		expectFile     string
		expectLine     int
		expectMessage  string
		expectSeverity string
	}{
		{
			name:           "format 1 - brackets",
			line:           "[CRITICAL] [main.go:42] Missing error handling",
			expectIssue:    true,
			expectFile:     "main.go",
			expectLine:     42,
			expectMessage:  "Missing error handling",
			expectSeverity: "critical",
		},
		{
			name:           "format 2 - bold",
			line:           "[HIGH] [util.go:10] Inefficient string concatenation",
			expectIssue:    true,
			expectFile:     "util.go",
			expectLine:     10,
			expectMessage:  "Inefficient string concatenation",
			expectSeverity: "high",
		},
		{
			name:           "format 3 - no line number",
			line:           "[MEDIUM] file.go Missing validation",
			expectIssue:    true,
			expectFile:     "file.go",
			expectLine:     0,
			expectMessage:  "Missing validation",
			expectSeverity: "medium",
		},
		{
			name:           "format 4 - bold no line",
			line:           "**CRITICAL** `file.go` Description text",
			expectIssue:    true,
			expectFile:     "file.go",
			expectLine:     0,
			expectMessage:  "Description text",
			expectSeverity: "critical",
		},
		{
			name:           "fallback - no brackets or bold",
			line:           "just plain text",
			expectIssue:    true,
			expectFile:     "",
			expectLine:     0,
			expectMessage:  "just plain text",
			expectSeverity: "medium",
		},
		{
			name:        "invalid - empty",
			line:        "",
			expectIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := parseReviewIssueLine(tt.line)

			if tt.expectIssue {
				if issue == nil {
					t.Fatal("expected issue, got nil")
				}

				if issue.File != tt.expectFile {
					t.Errorf("file: got %s, want %s", issue.File, tt.expectFile)
				}

				if issue.Line != tt.expectLine {
					t.Errorf("line: got %d, want %d", issue.Line, tt.expectLine)
				}

				if issue.Message != tt.expectMessage {
					t.Errorf("message: got %s, want %s", issue.Message, tt.expectMessage)
				}

				if issue.Severity != tt.expectSeverity {
					t.Errorf("severity: got %s, want %s", issue.Severity, tt.expectSeverity)
				}
			} else {
				if issue != nil {
					t.Errorf("expected no issue, got %+v", issue)
				}
			}
		})
	}
}

// TestParseReviewIssues tests parsing issues from a full review.
func TestParseReviewIssues(t *testing.T) {
	content := `## Issues

### Security

- [CRITICAL] [main.go:42] SQL injection vulnerability
- [HIGH] [auth.go:10] Missing authentication

### Performance

- [MEDIUM] [util.go:100] Inefficient loop
- [LOW] [cache.go:1] Missing cache invalidation
`

	issues := parseReviewIssues(content)

	if len(issues) != 4 {
		t.Fatalf("got %d issues, want 4", len(issues))
	}

	// Find the SQL injection issue
	var sqlIssue *ReviewIssue
	for i := range issues {
		if strings.Contains(issues[i].Message, "SQL injection") {
			sqlIssue = &issues[i]

			break
		}
	}

	if sqlIssue == nil {
		t.Fatal("SQL injection issue not found")
	}

	if sqlIssue.Severity != "critical" {
		t.Errorf("SQL injection severity: got %s, want critical", sqlIssue.Severity)
	}

	if sqlIssue.Category != "security" {
		t.Errorf("SQL injection category: got %s, want security", sqlIssue.Category)
	}
}

// TestInferCategoryFromSeverity tests category inference.
func TestInferCategoryFromSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		message  string
		expected string
	}{
		{
			name:     "critical security",
			severity: "critical",
			message:  "SQL injection vulnerability",
			expected: "security",
		},
		{
			name:     "critical correctness",
			severity: "critical",
			message:  "Null pointer dereference",
			expected: "correctness",
		},
		{
			name:     "high security keyword",
			severity: "high",
			message:  "Authentication bypass issue",
			expected: "security",
		},
		{
			name:     "medium performance",
			severity: "medium",
			message:  "Slow database query",
			expected: "performance",
		},
		{
			name:     "low performance",
			severity: "low",
			message:  "Inefficient algorithm",
			expected: "performance",
		},
		{
			name:     "default correctness",
			severity: "medium",
			message:  "Just a bug",
			expected: "correctness",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferCategoryFromSeverity(tt.severity, tt.message)
			if result != tt.expected {
				t.Errorf("inferCategoryFromSeverity(%q, %q) = %q, want %q",
					tt.severity, tt.message, result, tt.expected)
			}
		})
	}
}

// TestCleanReviewMarkdown tests markdown cleaning.
func TestCleanReviewMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bold",
			input:    "**bold text**",
			expected: "bold text",
		},
		{
			name:     "code",
			input:    "`code text`",
			expected: "code text",
		},
		{
			name:     "extra whitespace",
			input:    "text    with    spaces",
			expected: "text with spaces",
		},
		{
			name:     "mixed",
			input:    "**bold** `code` text    here",
			expected: "bold code text here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanReviewMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("cleanReviewMarkdown(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGenerateReviewIssueID tests ID generation for review issues.
func TestGenerateReviewIssueID(t *testing.T) {
	id1 := generateReviewIssueID("test.go", "message", 42)
	id2 := generateReviewIssueID("test.go", "message", 42)

	if id1 != id2 {
		t.Errorf("IDs not stable: %s != %s", id1, id2)
	}

	id3 := generateReviewIssueID("test.go", "different", 42)
	if id1 == id3 {
		t.Error("different message produced same ID")
	}
}
