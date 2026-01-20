package conductor

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SimplifiedSpec represents a parsed simplified specification.
type SimplifiedSpec struct {
	Number  int
	Content string
}

// parseSimplifiedSpecifications extracts simplified specs from agent response.
// Expected format: --- specification-N.md ---\n[content]\n--- end ---.
func parseSimplifiedSpecifications(content string) []SimplifiedSpec {
	pattern := regexp.MustCompile(`(?s)---\s+specification-(\d+)\.md\s*---\s*\n(.*?)\n---\s+end\s*---`)
	matches := pattern.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		// Fallback: treat entire response as single spec
		return []SimplifiedSpec{{Number: 1, Content: strings.TrimSpace(content)}}
	}

	var specs []SimplifiedSpec
	for _, match := range matches {
		num, _ := strconv.Atoi(match[1])
		specs = append(specs, SimplifiedSpec{
			Number:  num,
			Content: strings.TrimSpace(match[2]),
		})
	}

	return specs
}

// parseSimplifiedCode extracts simplified code files from agent response.
// Expected format: --- path/to/file.ext ---\n[code]\n--- end ---.
func parseSimplifiedCode(content string) (map[string]string, error) {
	pattern := regexp.MustCompile(`(?s)---\s+(.+?)\s+---\s*\n(.*?)\n---\s+end\s*---`)
	matches := pattern.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return nil, errors.New("no simplified files found in response")
	}

	files := make(map[string]string)
	for _, match := range matches {
		filePath := strings.TrimSpace(match[1])
		files[filePath] = match[2]
	}

	return files, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// PR Review Parsing
// ─────────────────────────────────────────────────────────────────────────────

// parsePRReview parses an agent response into a structured ParsedReview.
func parsePRReview(content string) *ParsedReview {
	if content == "" {
		return &ParsedReview{
			Issues: []ReviewIssue{},
		}
	}

	review := &ParsedReview{
		Issues: []ReviewIssue{},
	}

	// Extract summary
	review.Summary = extractReviewSection(content, "Summary", "Overview", "Assessment")
	review.Overall = extractOverallAssessment(content)

	// Parse issues
	review.Issues = parseReviewIssues(content)

	// If we didn't find a summary but have issues, create one
	if review.Summary == "" && len(review.Issues) > 0 {
		review.Summary = fmt.Sprintf("Found %d issue(s) requiring attention", len(review.Issues))
	}

	return review
}

// extractReviewSection extracts content from a markdown section.
func extractReviewSection(content string, sectionHeaders ...string) string {
	for _, header := range sectionHeaders {
		// Try to find the section header
		pattern := regexp.MustCompile(`##\s*` + regexp.QuoteMeta(header))
		loc := pattern.FindStringIndex(content)
		if len(loc) < 2 {
			continue
		}

		// Find the end of the section
		sectionStart := loc[1] // End of the match

		// Find next ## header or end of content
		nextHeader := strings.Index(content[sectionStart:], "\n## ")
		end := len(content)
		if nextHeader != -1 {
			end = sectionStart + nextHeader
		}

		section := strings.TrimSpace(content[sectionStart:end])

		return section
	}

	return ""
}

// extractOverallAssessment extracts the overall assessment from the review.
func extractOverallAssessment(content string) string {
	content = strings.ToLower(content)

	// Look for assessment keywords
	if strings.Contains(content, "approved") || strings.Contains(content, "looks good") || strings.Contains(content, "no issues") {
		return "approved"
	}
	if strings.Contains(content, "changes requested") || strings.Contains(content, "needs changes") {
		return "changes_requested"
	}
	if strings.Contains(content, "comment") {
		return "comment"
	}

	return ""
}

// parseReviewIssues extracts issues from the review content.
func parseReviewIssues(content string) []ReviewIssue {
	var issues []ReviewIssue

	// Find the Issues section
	issuesSection := extractReviewSection(content, "Issues", "Findings", "Problems")
	if issuesSection == "" {
		return issues
	}

	// Parse each line
	lines := strings.Split(issuesSection, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip category headers
		if strings.HasPrefix(line, "###") || strings.HasPrefix(line, "##") {
			continue
		}

		// Skip non-issue lines
		if !strings.Contains(line, "[") && !strings.Contains(line, "**") {
			continue
		}

		issue := parseReviewIssueLine(line)
		if issue != nil && issue.Message != "" {
			issues = append(issues, *issue)
		}
	}

	return issues
}

// parseReviewIssueLine parses a single issue line.
// Examples:
// - [CRITICAL] [main.go:42] Missing error handling
// - **HIGH** `util.go:10` Inefficient string concatenation
// - [MEDIUM] file.go Missing validation
// - **CRITICAL** `file.go`:42: Description
// - [HIGH] file.go Description (no line).
func parseReviewIssueLine(line string) *ReviewIssue {
	// Trim leading bullet point markers (- or * with space)
	// Be careful not to strip ** (bold markdown) at the start
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		line = strings.TrimSpace(line[2:])
	}

	issue := &ReviewIssue{
		Category: "general", // Default category
		Severity: "medium",  // Default severity
	}

	// Extract severity and location
	severityPatterns := map[string]string{
		"CRITICAL": "critical",
		"HIGH":     "high",
		"MEDIUM":   "medium",
		"LOW":      "low",
		"BLOCKER":  "critical",
		"TRIVIAL":  "low",
	}

	// Define patterns using string concatenation to avoid backslash issues in raw strings
	// [SEVERITY] [file:line] Message
	pattern1 := "^\\[([A-Z]+)\\]\\s+\\[([^\\]]+)\\]\\s+(.+)"
	// **SEVERITY** file:line Message
	pattern2 := "\\*\\*([A-Z]+)\\*\\*\\s+([^:\\s]+):\\s*(\\d+):\\s*(.+)"
	// **SEVERITY** file Description (no line)
	pattern3 := "\\*\\*([A-Z]+)\\*\\*\\s+([^\\s]+)\\s+(.+)"
	// [SEVERITY] file Description (no line)
	pattern4 := "^\\[([A-Z]+)\\]\\s+([^\\s]+)\\s+(.+)"

	patterns := []struct {
		pattern     string
		severityIdx int
		locationIdx int
		messageFrom int
	}{
		{pattern1, 1, 2, 3},
		{pattern2, 1, 2, 4},
		{pattern3, 1, 2, 3},
		{pattern4, 1, 2, 3},
	}

	var severityOrCategory, location, message string

	for _, pat := range patterns {
		re := regexp.MustCompile(pat.pattern)
		matches := re.FindStringSubmatch(line)
		if len(matches) > pat.severityIdx && matches[pat.severityIdx] != "" {
			severityOrCategory = matches[pat.severityIdx]
			if pat.locationIdx > 0 && len(matches) > pat.locationIdx {
				location = matches[pat.locationIdx]
			}
			if pat.messageFrom > 0 && len(matches) > pat.messageFrom {
				message = matches[pat.messageFrom]
			}

			break
		}
	}

	// If no pattern matched, try fallback parsing
	if severityOrCategory == "" {
		// Try to find any severity/category keyword at the start
		re := regexp.MustCompile(`^\[?([A-Z]+)\]?\s*`)
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 && matches[1] != "" {
			severityOrCategory = matches[1]
		}

		// For message, take everything after the first bracket/keyword
		remaining := line
		if idx := strings.Index(line, "]"); idx != -1 {
			remaining = strings.TrimSpace(line[idx+1:])
		} else if idx := strings.Index(line, "**"); idx != -1 && idx+3 < len(line) {
			// Find closing **
			if endIdx := strings.Index(line[idx+2:], "**"); endIdx != -1 {
				remaining = strings.TrimSpace(line[idx+3+endIdx+2:])
			}
		}
		message = remaining
	}

	// Process severity/category
	if severityOrCategory != "" {
		severityOrCategory = strings.ToUpper(severityOrCategory)
		if sev, ok := severityPatterns[severityOrCategory]; ok {
			issue.Severity = sev
		} else {
			// It's a category - use medium as default severity
			issue.Category = strings.ToLower(severityOrCategory)
		}
	}

	// Process location (extract file and line number)
	if location != "" {
		// Handle `file:line` format
		if strings.Contains(location, ":") {
			parts := strings.SplitN(location, ":", 2)
			if len(parts) == 2 {
				issue.File = strings.Trim(parts[0], "`[]")
				if lineNum, err := strconv.Atoi(strings.Trim(parts[1], "`[]")); err == nil {
					issue.Line = lineNum
				}
			}
		} else {
			// Location is just a file path
			issue.File = strings.Trim(location, "`[]")
			issue.Line = 0
		}
	}

	// Clean up message
	if message != "" {
		message = cleanReviewMarkdown(message)
		issue.Message = message
	} else {
		// Fallback: extract message from line directly
		issue.Message = cleanReviewMarkdown(line)
	}

	// Validate that we have at least a message
	if issue.Message == "" {
		return nil
	}

	// Infer category from severity if not set
	if issue.Category == "general" {
		issue.Category = inferCategoryFromSeverity(issue.Severity, issue.Message)
	}

	// Generate ID if not provided
	if issue.ID == "" {
		issue.ID = generateReviewIssueID(issue.File, issue.Message, issue.Line)
	}

	return issue
}

// generateReviewIssueID creates a stable unique ID for a review issue.
// Uses the proper SHA256-based GenerateIssueID function from pr_review_state.go.
func generateReviewIssueID(file, message string, line int) string {
	return GenerateIssueID(file, message, line)
}

// inferCategoryFromSeverity infers a category from severity and message content.
func inferCategoryFromSeverity(severity, message string) string {
	if severity == "critical" || severity == "blocker" {
		if strings.Contains(strings.ToLower(message), "security") ||
			strings.Contains(strings.ToLower(message), "injection") ||
			strings.Contains(strings.ToLower(message), "vulnerability") {
			return "security"
		}

		return "correctness"
	}

	messageLower := strings.ToLower(message)

	// Security keywords
	securityKeywords := []string{
		"security", "injection", "xss", "csrf", "authentication", "authorization",
		"vulnerability", "exploit", "hack", "credential", "token",
	}
	for _, kw := range securityKeywords {
		if strings.Contains(messageLower, kw) {
			return "security"
		}
	}

	// Performance keywords
	performanceKeywords := []string{
		"slow", "inefficient", "performance", "optimize", "memory leak",
		"complexity", "o(n²)", "quadratic",
	}
	for _, kw := range performanceKeywords {
		if strings.Contains(messageLower, kw) {
			return "performance"
		}
	}

	return "correctness"
}

// cleanReviewMarkdown removes markdown formatting artifacts.
func cleanReviewMarkdown(text string) string {
	// Remove bold markers
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "*", "")

	// Remove code markers
	text = strings.ReplaceAll(text, "`", "")

	// Remove extra whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}
