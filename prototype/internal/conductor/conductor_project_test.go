package conductor

import (
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestFormatWorkUnitAsSource(t *testing.T) {
	tests := []struct {
		name     string
		wu       *provider.WorkUnit
		contains []string
	}{
		{
			name: "full work unit",
			wu: &provider.WorkUnit{
				Title:       "Implement Auth",
				Description: "Build JWT-based authentication",
				Labels:      []string{"backend", "security"},
				Priority:    provider.PriorityHigh,
				Status:      provider.StatusInProgress,
				Assignees: []provider.Person{
					{Name: "Alice", ID: "alice123"},
					{ID: "bob456"},
				},
			},
			contains: []string{
				"# Implement Auth",
				"Build JWT-based authentication",
				"**Labels:** backend, security",
				"**Priority:** high",
				"**Status:** in_progress",
				"**Assignees:** Alice, bob456",
			},
		},
		{
			name: "minimal work unit",
			wu: &provider.WorkUnit{
				Title: "Simple Task",
			},
			contains: []string{
				"# Simple Task",
			},
		},
		{
			name: "work unit with empty assignee names",
			wu: &provider.WorkUnit{
				Title: "Task",
				Assignees: []provider.Person{
					{Name: ""},
					{ID: ""},
				},
			},
			contains: []string{
				"# Task",
			},
		},
		{
			name: "work unit with description only",
			wu: &provider.WorkUnit{
				Title:       "Task",
				Description: "Detailed description here",
			},
			contains: []string{
				"# Task",
				"Detailed description here",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatWorkUnitAsSource(tt.wu)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("formatWorkUnitAsSource() missing %q in:\n%s", expected, result)
				}
			}
		})
	}
}

func TestGenerateQueueID(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		source     string
		wantPrefix string
	}{
		{
			name:       "title provided",
			title:      "My Project",
			source:     "file:test.md",
			wantPrefix: "my-project-",
		},
		{
			name:       "title with special chars",
			title:      "Q1 Features!@#$",
			source:     "",
			wantPrefix: "q1-features-",
		},
		{
			name:       "no title with dir source",
			title:      "",
			source:     "dir:/path/to/specs",
			wantPrefix: "specs-",
		},
		{
			name:       "no title with file source",
			title:      "",
			source:     "file:/path/to/requirements.md",
			wantPrefix: "requirements-",
		},
		{
			name:       "no title with provider source",
			title:      "",
			source:     "github:123",
			wantPrefix: "github-123-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateQueueID(tt.title, tt.source)

			if !strings.HasPrefix(result, tt.wantPrefix) {
				t.Errorf("generateQueueID(%q, %q) = %q, want prefix %q", tt.title, tt.source, result, tt.wantPrefix)
			}

			// Should have timestamp suffix
			parts := strings.Split(result, "-")
			if len(parts) < 3 {
				t.Errorf("generateQueueID() = %q, expected timestamp suffix", result)
			}
		})
	}
}

func TestBuildReorderingPrompt(t *testing.T) {
	queue := &storage.TaskQueue{
		ID:    "test-queue",
		Title: "Test Project",
		Tasks: []*storage.QueuedTask{
			{
				ID:          "task-1",
				Title:       "Setup Database",
				Priority:    1,
				Status:      storage.TaskStatusReady,
				Description: "Create database schema",
			},
			{
				ID:        "task-2",
				Title:     "Create API",
				Priority:  2,
				Status:    storage.TaskStatusBlocked,
				DependsOn: []string{"task-1"},
				Blocks:    []string{"task-3"},
			},
			{
				ID:        "task-3",
				Title:     "Add Frontend",
				Priority:  3,
				Status:    storage.TaskStatusBlocked,
				DependsOn: []string{"task-2"},
			},
		},
	}

	result := buildReorderingPrompt(queue)

	// Check that it contains expected content
	expectedContents := []string{
		"Test Project",
		"task-1",
		"task-2",
		"task-3",
		"Setup Database",
		"Create API",
		"Add Frontend",
		"**Priority**: 1",
		"**Priority**: 2",
		"**Priority**: 3",
		"**Depends on**: task-1",
		"**Blocks**: task-3",
		"## Recommended Order",
		"## Reasoning",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(result, expected) {
			t.Errorf("buildReorderingPrompt() missing %q in result", expected)
		}
	}
}

func TestBuildProjectPlanningPrompt(t *testing.T) {
	title := "Auth System"
	sourceContent := "# Requirements\n\n- User login\n- Password reset"
	customInstructions := "Focus on security"

	result := buildProjectPlanningPrompt(title, sourceContent, customInstructions)

	expectedContents := []string{
		"## Project",
		"Auth System",
		"## Source Content",
		"# Requirements",
		"User login",
		"## Custom Instructions",
		"Focus on security",
		"## Output Format",
		"### task-N: Task Title",
		"**Priority**: N",
		"**Status**: ready OR blocked",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(result, expected) {
			t.Errorf("buildProjectPlanningPrompt() missing %q in result", expected)
		}
	}
}

func TestBuildProjectPlanningPrompt_NoCustomInstructions(t *testing.T) {
	result := buildProjectPlanningPrompt("Test", "content", "")

	if strings.Contains(result, "## Custom Instructions") {
		t.Error("buildProjectPlanningPrompt() should not include Custom Instructions section when empty")
	}
}

func TestProjectPlanOptions(t *testing.T) {
	opts := ProjectPlanOptions{
		Title:              "My Project",
		CustomInstructions: "Be concise",
	}

	if opts.Title != "My Project" {
		t.Errorf("Title = %q, want %q", opts.Title, "My Project")
	}
	if opts.CustomInstructions != "Be concise" {
		t.Errorf("CustomInstructions = %q, want %q", opts.CustomInstructions, "Be concise")
	}
}

func TestSubmitOptions(t *testing.T) {
	opts := SubmitOptions{
		Provider:   "wrike",
		CreateEpic: true,
		Labels:     []string{"q1", "feature"},
		DryRun:     false,
	}

	if opts.Provider != "wrike" {
		t.Errorf("Provider = %q, want %q", opts.Provider, "wrike")
	}
	if !opts.CreateEpic {
		t.Error("CreateEpic should be true")
	}
	if len(opts.Labels) != 2 {
		t.Errorf("Labels length = %d, want 2", len(opts.Labels))
	}
}

func TestAutoReorderResult(t *testing.T) {
	result := AutoReorderResult{
		OldOrder:  []string{"task-3", "task-1", "task-2"},
		NewOrder:  []string{"task-1", "task-2", "task-3"},
		Reasoning: "Reordered based on dependencies",
	}

	if len(result.OldOrder) != 3 {
		t.Errorf("OldOrder length = %d, want 3", len(result.OldOrder))
	}
	if len(result.NewOrder) != 3 {
		t.Errorf("NewOrder length = %d, want 3", len(result.NewOrder))
	}
	if result.Reasoning == "" {
		t.Error("Reasoning should not be empty")
	}
}

func TestSubmittedTask(t *testing.T) {
	task := SubmittedTask{
		LocalID:     "task-1",
		ExternalID:  "PROJ-123",
		ExternalURL: "https://jira.example.com/browse/PROJ-123",
		Title:       "Implement Feature",
	}

	if task.LocalID != "task-1" {
		t.Errorf("LocalID = %q, want %q", task.LocalID, "task-1")
	}
	if task.ExternalID != "PROJ-123" {
		t.Errorf("ExternalID = %q, want %q", task.ExternalID, "PROJ-123")
	}
	if task.Title != "Implement Feature" {
		t.Errorf("Title = %q, want %q", task.Title, "Implement Feature")
	}
}

func TestSubmitResult(t *testing.T) {
	result := SubmitResult{
		Epic: &SubmittedItem{
			ExternalID:  "EPIC-1",
			ExternalURL: "https://example.com/epic/1",
			Title:       "Q1 Epic",
		},
		Tasks: []*SubmittedTask{
			{LocalID: "task-1", ExternalID: "EXT-1"},
			{LocalID: "task-2", ExternalID: "EXT-2"},
		},
		DryRun: false,
	}

	if result.Epic == nil {
		t.Error("Epic should not be nil")
	}
	if result.Epic.Title != "Q1 Epic" {
		t.Errorf("Epic.Title = %q, want %q", result.Epic.Title, "Q1 Epic")
	}
	if len(result.Tasks) != 2 {
		t.Errorf("Tasks length = %d, want 2", len(result.Tasks))
	}
}

func TestProjectPlanResult(t *testing.T) {
	result := ProjectPlanResult{
		Queue: &storage.TaskQueue{ID: "test-queue"},
		Tasks: []*storage.QueuedTask{
			{ID: "task-1", Title: "Task 1"},
		},
		Questions: []string{"What is the scope?"},
		Blockers:  []string{"Need API access"},
	}

	if result.Queue == nil {
		t.Error("Queue should not be nil")
	}
	if len(result.Tasks) != 1 {
		t.Errorf("Tasks length = %d, want 1", len(result.Tasks))
	}
	if len(result.Questions) != 1 {
		t.Errorf("Questions length = %d, want 1", len(result.Questions))
	}
	if len(result.Blockers) != 1 {
		t.Errorf("Blockers length = %d, want 1", len(result.Blockers))
	}
}

func TestProjectAutoOptions(t *testing.T) {
	opts := ProjectAutoOptions{
		ProjectPlanOptions: ProjectPlanOptions{
			Title: "Test Project",
		},
		SubmitOptions: SubmitOptions{
			Provider: "github",
		},
	}

	if opts.Title != "Test Project" {
		t.Errorf("Title = %q, want %q", opts.Title, "Test Project")
	}
	if opts.Provider != "github" {
		t.Errorf("Provider = %q, want %q", opts.Provider, "github")
	}
}

func TestProjectAutoResult(t *testing.T) {
	result := ProjectAutoResult{
		Queue:          &storage.TaskQueue{ID: "q1"},
		TasksPlanned:   5,
		TasksSubmitted: 5,
		TasksCompleted: 3,
		FailedAt:       "implement-task-4",
	}

	if result.TasksPlanned != 5 {
		t.Errorf("TasksPlanned = %d, want 5", result.TasksPlanned)
	}
	if result.TasksSubmitted != 5 {
		t.Errorf("TasksSubmitted = %d, want 5", result.TasksSubmitted)
	}
	if result.TasksCompleted != 3 {
		t.Errorf("TasksCompleted = %d, want 3", result.TasksCompleted)
	}
	if result.FailedAt != "implement-task-4" {
		t.Errorf("FailedAt = %q, want %q", result.FailedAt, "implement-task-4")
	}
}

func TestSubmittedWorkUnit(t *testing.T) {
	wu := submittedWorkUnit{
		ID:    "ext-123",
		URL:   "https://example.com/task/123",
		Title: "Test Task",
	}

	if wu.ID != "ext-123" {
		t.Errorf("ID = %q, want %q", wu.ID, "ext-123")
	}
	if wu.URL != "https://example.com/task/123" {
		t.Errorf("URL = %q, want %q", wu.URL, "https://example.com/task/123")
	}
	if wu.Title != "Test Task" {
		t.Errorf("Title = %q, want %q", wu.Title, "Test Task")
	}
}
