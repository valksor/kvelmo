package conductor

import (
	"slices"
	"strings"
)

// TaskComplexity represents the detected complexity level of a task.
type TaskComplexity string

const (
	// ComplexitySimple indicates a straightforward task suitable for quick planning.
	ComplexitySimple TaskComplexity = "simple"
	// ComplexityMedium indicates a moderately complex task.
	ComplexityMedium TaskComplexity = "medium"
	// ComplexityComplex indicates a task requiring full planning treatment.
	ComplexityComplex TaskComplexity = "complex"
)

// Thresholds for complexity detection.
const (
	simpleTitleMaxLen   = 100
	simpleContentMaxLen = 500
	simpleMaxFiles      = 1
	complexTitleMinLen  = 300
	complexMinFiles     = 4
)

// simpleActionKeywords indicate straightforward changes.
var simpleActionKeywords = []string{
	"update", "bump", "upgrade",
	"fix typo", "typo",
	"add comment", "remove comment",
	"rename", "remove",
	"change version", "set version",
}

// complexArchitecturalKeywords indicate tasks requiring deeper planning.
var complexArchitecturalKeywords = []string{
	"refactor", "redesign", "migrate",
	"integrate", "replace", "abstract",
	"rewrite", "restructure", "overhaul",
	"architecture", "design pattern",
}

// complexLabels are labels that indicate architectural scope.
var complexLabels = []string{
	"architecture", "breaking-change", "migration",
	"refactor", "redesign", "epic",
}

// simpleTaskTypes are task types suitable for quick planning.
var simpleTaskTypes = []string{
	"fix", "chore", "docs", "style",
}

// DetectTaskComplexity analyzes task metadata to determine appropriate planning complexity.
// It uses heuristics based on title length, content size, keywords, file count,
// task type, labels, and hierarchical context.
//
// The function errs on the side of caution: if signals are mixed or unclear,
// it defaults to medium or complex to ensure adequate planning.
func DetectTaskComplexity(title, sourceContent string, fileCount int, taskType string, labels []string, hasParent bool) TaskComplexity {
	titleLower := strings.ToLower(title)
	contentLower := strings.ToLower(sourceContent)
	combinedText := titleLower + " " + contentLower

	// Check for explicit complexity signals first (complex takes priority)
	if isDefinitelyComplex(combinedText, fileCount, labels, hasParent) {
		return ComplexityComplex
	}

	// Check for simple task signals
	if isDefinitelySimple(title, sourceContent, fileCount, taskType, combinedText) {
		return ComplexitySimple
	}

	// Default to medium for ambiguous cases
	return ComplexityMedium
}

// isDefinitelyComplex returns true if strong complexity signals are present.
func isDefinitelyComplex(combinedText string, fileCount int, labels []string, hasParent bool) bool {
	// Large file count indicates broad scope
	if fileCount >= complexMinFiles {
		return true
	}

	// Subtasks of larger work need full context
	if hasParent {
		return true
	}

	// Architectural keywords indicate need for thorough planning
	for _, keyword := range complexArchitecturalKeywords {
		if strings.Contains(combinedText, keyword) {
			return true
		}
	}

	// Architectural labels indicate scope
	for _, label := range labels {
		labelLower := strings.ToLower(label)
		if slices.Contains(complexLabels, labelLower) {
			return true
		}
	}

	return false
}

// isDefinitelySimple returns true if the task appears straightforward.
func isDefinitelySimple(title, sourceContent string, fileCount int, taskType, combinedText string) bool {
	// Basic size checks
	if len(title) > simpleTitleMaxLen {
		return false
	}
	if len(sourceContent) > simpleContentMaxLen {
		return false
	}
	if fileCount > simpleMaxFiles {
		return false
	}

	// Simple task type is a strong positive signal
	taskTypeLower := strings.ToLower(taskType)
	isSimpleType := slices.Contains(simpleTaskTypes, taskTypeLower)

	// Simple action keywords in text
	hasSimpleKeyword := false
	for _, keyword := range simpleActionKeywords {
		if strings.Contains(combinedText, keyword) {
			hasSimpleKeyword = true

			break
		}
	}

	// Need at least one positive signal (type or keyword) plus size constraints
	return isSimpleType || hasSimpleKeyword
}
