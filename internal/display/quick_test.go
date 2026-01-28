package display

import (
	"strings"
	"testing"
)

func TestQuickTaskSuccess(t *testing.T) {
	result := QuickTaskSuccess("task-123", "Build new feature", "queue-abc")

	if result == "" {
		t.Fatal("QuickTaskSuccess() returned empty string")
	}

	// Check that task ID is included
	if !strings.Contains(result, "task-123") {
		t.Errorf("QuickTaskSuccess() should contain task ID, got: %s", result)
	}

	// Check that title is included
	if !strings.Contains(result, "Build new feature") {
		t.Errorf("QuickTaskSuccess() should contain title, got: %s", result)
	}

	// Check that queue ID is included
	if !strings.Contains(result, "queue-abc") {
		t.Errorf("QuickTaskSuccess() should contain queue ID, got: %s", result)
	}

	// Check for the checkmark
	if !strings.Contains(result, "✓") {
		t.Errorf("QuickTaskSuccess() should contain success indicator, got: %s", result)
	}
}

func TestQuickTaskListEmpty(t *testing.T) {
	result := QuickTaskList([]QuickTaskItem{})

	if result != "No tasks found." {
		t.Errorf("QuickTaskList() with empty list = %q, want %q", result, "No tasks found.")
	}
}

func TestQuickTaskListSingle(t *testing.T) {
	tasks := []QuickTaskItem{
		{ID: "task-1", Title: "First task", Labels: []string{"bug"}, NoteCount: 2},
	}

	result := QuickTaskList(tasks)

	// Should contain headers
	if !strings.Contains(result, "ID") || !strings.Contains(result, "Title") {
		t.Errorf("QuickTaskList() should contain table headers, got: %s", result)
	}

	// Should contain task data
	if !strings.Contains(result, "task-1") {
		t.Errorf("QuickTaskList() should contain task ID, got: %s", result)
	}

	if !strings.Contains(result, "First task") {
		t.Errorf("QuickTaskList() should contain title, got: %s", result)
	}

	if !strings.Contains(result, "bug") {
		t.Errorf("QuickTaskList() should contain labels, got: %s", result)
	}

	if !strings.Contains(result, "2") {
		t.Errorf("QuickTaskList() should contain note count, got: %s", result)
	}
}

func TestQuickTaskListMultiple(t *testing.T) {
	tasks := []QuickTaskItem{
		{ID: "task-1", Title: "First task", Labels: []string{}, NoteCount: 0},
		{ID: "task-2", Title: "Second task", Labels: []string{"feature"}, NoteCount: 0},
		{ID: "task-3", Title: "Third task", Labels: []string{"bug", "urgent"}, NoteCount: 5},
	}

	result := QuickTaskList(tasks)

	// All task IDs should be present
	if !strings.Contains(result, "task-1") {
		t.Error("Missing task-1")
	}
	if !strings.Contains(result, "task-2") {
		t.Error("Missing task-2")
	}
	if !strings.Contains(result, "task-3") {
		t.Error("Missing task-3")
	}

	// Empty labels should show "-"
	if !strings.Contains(result, "-") {
		t.Error("Empty labels should show '-'")
	}
}

func TestQuickTaskListLongTitleTruncated(t *testing.T) {
	longTitle := "This is a very long title that should be truncated because it exceeds the maximum length"
	tasks := []QuickTaskItem{
		{ID: "task-1", Title: longTitle, Labels: []string{}, NoteCount: 0},
	}

	result := QuickTaskList(tasks)

	// Title should be truncated
	if !strings.Contains(result, "...") {
		t.Errorf("Long title should be truncated, got: %s", result)
	}

	// Original long title should NOT be present
	if strings.Contains(result, longTitle) {
		t.Errorf("Long title should be truncated, but full title found in: %s", result)
	}
}

func TestOptimizedTaskChangeTitleChanged(t *testing.T) {
	result := OptimizedTaskChange(OptimizedTaskResult{
		OriginalTitle:  "Fix bug",
		OptimizedTitle: "Fix critical authentication bug",
	})

	// Should show transition arrow
	if !strings.Contains(result, "→") {
		t.Errorf("OptimizedTaskChange() should show transition for changed title, got: %s", result)
	}

	if !strings.Contains(result, "Fix bug") {
		t.Errorf("Should contain original title, got: %s", result)
	}

	if !strings.Contains(result, "Fix critical authentication bug") {
		t.Errorf("Should contain optimized title, got: %s", result)
	}
}

func TestOptimizedTaskChangeTitleUnchanged(t *testing.T) {
	result := OptimizedTaskChange(OptimizedTaskResult{
		OriginalTitle:  "Same title",
		OptimizedTitle: "Same title",
	})

	if !strings.Contains(result, "(unchanged)") {
		t.Errorf("OptimizedTaskChange() with unchanged title should show '(unchanged)', got: %s", result)
	}

	// Should NOT show transition arrow
	if strings.Contains(result, "→") {
		t.Errorf("OptimizedTaskChange() with unchanged title should not show arrow, got: %s", result)
	}
}

func TestOptimizedTaskChangeDescriptionChanged(t *testing.T) {
	result := OptimizedTaskChange(OptimizedTaskResult{
		OriginalTitle:      "Title",
		OptimizedTitle:     "Title",
		DescriptionChanged: true,
	})

	if !strings.Contains(result, "Description: enhanced") {
		t.Errorf("Should show enhanced description, got: %s", result)
	}
}

func TestOptimizedTaskChangeAddedLabels(t *testing.T) {
	result := OptimizedTaskChange(OptimizedTaskResult{
		OriginalTitle:  "Title",
		OptimizedTitle: "Title",
		AddedLabels:    []string{"bug", "urgent", "security"},
	})

	if !strings.Contains(result, "Added labels:") {
		t.Errorf("Should show added labels section, got: %s", result)
	}

	if !strings.Contains(result, "bug") {
		t.Error("Missing 'bug' label")
	}

	if !strings.Contains(result, "urgent") {
		t.Error("Missing 'urgent' label")
	}

	if !strings.Contains(result, "security") {
		t.Error("Missing 'security' label")
	}
}

func TestOptimizedTaskChangeImprovements(t *testing.T) {
	result := OptimizedTaskChange(OptimizedTaskResult{
		OriginalTitle:  "Title",
		OptimizedTitle: "Title",
		Improvements:   []string{"Better clarity", "More specific", "Added context"},
	})

	if !strings.Contains(result, "Improvements:") {
		t.Errorf("Should show improvements section, got: %s", result)
	}

	// All improvements should be bulleted
	for _, imp := range []string{"Better clarity", "More specific", "Added context"} {
		if !strings.Contains(result, imp) {
			t.Errorf("Missing improvement: %s", imp)
		}
	}
}

func TestQuickTaskMenu(t *testing.T) {
	result := QuickTaskMenu()

	if result == "" {
		t.Fatal("QuickTaskMenu() returned empty string")
	}

	// Check for all menu options
	expectedOptions := []string{
		"[d]iscuss",
		"[o]ptimize",
		"[s]ubmit",
		"[tart]", // part of [start]
		"[x]exit",
	}

	for _, option := range expectedOptions {
		if !strings.Contains(result, option) {
			t.Errorf("QuickTaskMenu() should contain %q, got: %s", option, result)
		}
	}

	// Should mention "What next?"
	if !strings.Contains(result, "What next?") {
		t.Errorf("QuickTaskMenu() should contain 'What next?', got: %s", result)
	}
}

func TestFormatSubmitSuccess(t *testing.T) {
	result := FormatSubmitSuccess("github", "PR-123", "https://github.com/user/repo/pull/123")

	// Check basic content
	if !strings.Contains(result, "✓ Submitted:") {
		t.Error("Should show submitted indicator")
	}

	if !strings.Contains(result, "github") {
		t.Error("Should contain provider name")
	}

	if !strings.Contains(result, "PR-123") {
		t.Error("Should contain external ID")
	}

	if !strings.Contains(result, "https://github.com/user/repo/pull/123") {
		t.Error("Should contain URL")
	}
}

func TestFormatSubmitSuccessNoURL(t *testing.T) {
	result := FormatSubmitSuccess("jira", "TICKET-456", "")

	// URL should not be mentioned
	if strings.Contains(result, "URL:") {
		t.Error("Should not show URL line when URL is empty")
	}

	// External ID should still be present
	if !strings.Contains(result, "TICKET-456") {
		t.Error("Should contain external ID")
	}
}

func TestFormatExportSuccess(t *testing.T) {
	result := FormatExportSuccess("/path/to/task.md")

	if !strings.Contains(result, "✓ Exported to:") {
		t.Error("Should show exported indicator")
	}

	if !strings.Contains(result, "/path/to/task.md") {
		t.Error("Should contain filename")
	}

	// Should contain command hint
	if !strings.Contains(result, "mehr start") {
		t.Error("Should contain command hint")
	}

	if !strings.Contains(result, "file:/path/to/task.md") {
		t.Error("Should contain file reference with filename")
	}
}
