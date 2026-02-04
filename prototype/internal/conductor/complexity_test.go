package conductor

import "testing"

func TestDetectTaskComplexity(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		sourceContent string
		fileCount     int
		taskType      string
		labels        []string
		hasParent     bool
		want          TaskComplexity
	}{
		// Simple tasks
		{
			name:          "simple: short title with update keyword",
			title:         "Update package.json version",
			sourceContent: "Bump version to 2.0.0",
			fileCount:     1,
			taskType:      "chore",
			labels:        nil,
			hasParent:     false,
			want:          ComplexitySimple,
		},
		{
			name:          "simple: fix typo",
			title:         "Fix typo in README",
			sourceContent: "Change 'teh' to 'the'",
			fileCount:     1,
			taskType:      "fix",
			labels:        nil,
			hasParent:     false,
			want:          ComplexitySimple,
		},
		{
			name:          "simple: bump version keyword",
			title:         "Bump version to 1.2.3",
			sourceContent: "Update version number",
			fileCount:     0,
			taskType:      "",
			labels:        nil,
			hasParent:     false,
			want:          ComplexitySimple,
		},
		{
			name:          "simple: rename variable",
			title:         "Rename variable for clarity",
			sourceContent: "Change foo to userCount",
			fileCount:     1,
			taskType:      "fix",
			labels:        nil,
			hasParent:     false,
			want:          ComplexitySimple,
		},
		{
			name:          "simple: chore task type",
			title:         "Clean up old comments",
			sourceContent: "Remove outdated TODO comments",
			fileCount:     1,
			taskType:      "chore",
			labels:        nil,
			hasParent:     false,
			want:          ComplexitySimple,
		},

		// Complex tasks
		{
			name:          "complex: refactor keyword",
			title:         "Refactor authentication module",
			sourceContent: "Restructure the auth flow",
			fileCount:     1,
			taskType:      "feature",
			labels:        nil,
			hasParent:     false,
			want:          ComplexityComplex,
		},
		{
			name:          "complex: migrate keyword",
			title:         "Migrate to new API",
			sourceContent: "Update all endpoints",
			fileCount:     2,
			taskType:      "feature",
			labels:        nil,
			hasParent:     false,
			want:          ComplexityComplex,
		},
		{
			name:          "complex: many files",
			title:         "Update imports",
			sourceContent: "Change import paths",
			fileCount:     5,
			taskType:      "chore",
			labels:        nil,
			hasParent:     false,
			want:          ComplexityComplex,
		},
		{
			name:          "complex: has parent task",
			title:         "Add login button",
			sourceContent: "Simple button addition",
			fileCount:     1,
			taskType:      "feature",
			labels:        nil,
			hasParent:     true,
			want:          ComplexityComplex,
		},
		{
			name:          "complex: architecture label",
			title:         "Update config loader",
			sourceContent: "Small change",
			fileCount:     1,
			taskType:      "fix",
			labels:        []string{"architecture"},
			hasParent:     false,
			want:          ComplexityComplex,
		},
		{
			name:          "complex: breaking-change label",
			title:         "Update API response",
			sourceContent: "Change response format",
			fileCount:     1,
			taskType:      "fix",
			labels:        []string{"breaking-change"},
			hasParent:     false,
			want:          ComplexityComplex,
		},
		{
			name:          "complex: redesign keyword in content",
			title:         "Improve UI",
			sourceContent: "Redesign the dashboard layout",
			fileCount:     1,
			taskType:      "feature",
			labels:        nil,
			hasParent:     false,
			want:          ComplexityComplex,
		},

		// Medium tasks (no strong signals either way)
		{
			name:          "medium: no keywords, no task type",
			title:         "Add error message",
			sourceContent: "Display error when form is invalid",
			fileCount:     2,
			taskType:      "",
			labels:        nil,
			hasParent:     false,
			want:          ComplexityMedium,
		},
		{
			name:          "medium: feature type without simple keywords",
			title:         "Add user profile page",
			sourceContent: "Create profile page component",
			fileCount:     2,
			taskType:      "feature",
			labels:        nil,
			hasParent:     false,
			want:          ComplexityMedium,
		},

		// Edge cases
		{
			name:          "edge: title at boundary (100 chars)",
			title:         "Update pkg file with new version number and some padding text to reach exactly one hundred chars ooo", // exactly 100 chars
			sourceContent: "Short",
			fileCount:     1,
			taskType:      "chore",
			labels:        nil,
			hasParent:     false,
			want:          ComplexitySimple,
		},
		{
			name:          "edge: title over boundary (101 chars)",
			title:         "Update pkg file with new version number and some padding text to reach exactly one hundred chars oooo", // exactly 101 chars
			sourceContent: "Short",
			fileCount:     1,
			taskType:      "chore",
			labels:        nil,
			hasParent:     false,
			want:          ComplexityMedium,
		},
		{
			name:          "edge: case insensitive keyword detection",
			title:         "UPDATE Package Version",
			sourceContent: "BUMP to 2.0",
			fileCount:     1,
			taskType:      "CHORE",
			labels:        nil,
			hasParent:     false,
			want:          ComplexitySimple,
		},
		{
			name:          "edge: complex keyword overrides simple signals",
			title:         "Update and refactor config",
			sourceContent: "Simple change",
			fileCount:     1,
			taskType:      "chore",
			labels:        nil,
			hasParent:     false,
			want:          ComplexityComplex,
		},
		{
			name:          "edge: empty inputs",
			title:         "",
			sourceContent: "",
			fileCount:     0,
			taskType:      "",
			labels:        nil,
			hasParent:     false,
			want:          ComplexityMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectTaskComplexity(tt.title, tt.sourceContent, tt.fileCount, tt.taskType, tt.labels, tt.hasParent)
			if got != tt.want {
				t.Errorf("DetectTaskComplexity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComplexityConstants(t *testing.T) {
	// Verify constants have expected string values (for logging/debugging)
	if ComplexitySimple != "simple" {
		t.Errorf("ComplexitySimple = %q, want %q", ComplexitySimple, "simple")
	}
	if ComplexityMedium != "medium" {
		t.Errorf("ComplexityMedium = %q, want %q", ComplexityMedium, "medium")
	}
	if ComplexityComplex != "complex" {
		t.Errorf("ComplexityComplex = %q, want %q", ComplexityComplex, "complex")
	}
}

func TestThresholds(t *testing.T) {
	// Document and verify threshold values
	if simpleTitleMaxLen != 100 {
		t.Errorf("simpleTitleMaxLen = %d, want 100", simpleTitleMaxLen)
	}
	if simpleContentMaxLen != 500 {
		t.Errorf("simpleContentMaxLen = %d, want 500", simpleContentMaxLen)
	}
	if simpleMaxFiles != 1 {
		t.Errorf("simpleMaxFiles = %d, want 1", simpleMaxFiles)
	}
	if complexMinFiles != 4 {
		t.Errorf("complexMinFiles = %d, want 4", complexMinFiles)
	}
}
