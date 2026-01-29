package conductor

import (
	"os"
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

func TestReadResearchSource(t *testing.T) {
	t.Run("basic directory structure", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test files
		if err := os.WriteFile(tmpDir+"/README.md", []byte("# Test Project"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(tmpDir+"/tasks", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tmpDir+"/tasks/README.md", []byte("# Tasks"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tmpDir+"/config.yaml", []byte("key: value"), 0o644); err != nil {
			t.Fatal(err)
		}

		c := &Conductor{}
		manifest, err := c.readResearchSource(tmpDir)
		if err != nil {
			t.Fatalf("readResearchSource() error = %v", err)
		}

		if manifest.BasePath != tmpDir {
			t.Errorf("BasePath = %q, want %q", manifest.BasePath, tmpDir)
		}

		if manifest.FileCount != 3 {
			t.Errorf("FileCount = %d, want 3", manifest.FileCount)
		}

		if len(manifest.EntryPoints) == 0 {
			t.Error("EntryPoints should not be empty, expected at least README.md")
		}

		// Check entry points contain README.md
		hasReadme := false
		hasTasksReadme := false
		for _, ep := range manifest.EntryPoints {
			if strings.HasSuffix(ep, "README.md") {
				if strings.Contains(ep, "tasks") {
					hasTasksReadme = true
				} else {
					hasReadme = true
				}
			}
		}
		if !hasReadme {
			t.Error("EntryPoints should contain root README.md")
		}
		if !hasTasksReadme {
			t.Error("EntryPoints should contain tasks/README.md")
		}

		// Check categorization
		docsFiles, ok := manifest.ByCategory["docs"]
		if !ok {
			t.Error("ByCategory should have 'docs' key")
		} else if len(docsFiles) != 2 { // README.md and tasks/README.md
			t.Errorf("ByCategory['docs'] = %d, want 2", len(docsFiles))
		}

		configFiles, ok := manifest.ByCategory["config"]
		if !ok {
			t.Error("ByCategory should have 'config' key")
		} else if len(configFiles) != 1 {
			t.Errorf("ByCategory['config'] = %d, want 1", len(configFiles))
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		c := &Conductor{}
		manifest, err := c.readResearchSource(tmpDir)
		if err != nil {
			t.Fatalf("readResearchSource() error = %v", err)
		}

		if manifest.FileCount != 0 {
			t.Errorf("FileCount = %d, want 0", manifest.FileCount)
		}

		if len(manifest.EntryPoints) != 0 {
			t.Errorf("EntryPoints = %d, want 0", len(manifest.EntryPoints))
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		c := &Conductor{}
		_, err := c.readResearchSource("/nonexistent/path/12345")

		if err == nil {
			t.Error("readResearchSource() should return error for non-existent path")
		}
	})

	t.Run("file instead of directory", func(t *testing.T) {
		tmpFile := t.TempDir() + "/test.md"
		if err := os.WriteFile(tmpFile, []byte("# Test"), 0o644); err != nil {
			t.Fatal(err)
		}

		c := &Conductor{}
		_, err := c.readResearchSource(tmpFile)

		if err == nil {
			t.Error("readResearchSource() should return error for file path")
		}
	})

	t.Run("skips hidden files and common exclusions", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create hidden file
		if err := os.WriteFile(tmpDir+"/.hidden", []byte("hidden"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create node_modules directory
		if err := os.MkdirAll(tmpDir+"/node_modules/pkg", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tmpDir+"/node_modules/pkg/index.js", []byte("code"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create a normal file
		if err := os.WriteFile(tmpDir+"/visible.md", []byte("# Visible"), 0o644); err != nil {
			t.Fatal(err)
		}

		c := &Conductor{}
		manifest, err := c.readResearchSource(tmpDir)
		if err != nil {
			t.Fatalf("readResearchSource() error = %v", err)
		}

		if manifest.FileCount != 1 {
			t.Errorf("FileCount = %d, want 1 (hidden and node_modules should be excluded)", manifest.FileCount)
		}
	})
}

func TestBuildResearchPlanningPrompt(t *testing.T) {
	manifest := &ResearchManifest{
		BasePath:  "/workspace/docs",
		FileCount: 5,
		Structure: []DirEntry{
			{Path: "README.md", Name: "README.md", Type: "file", Size: 100, Category: "docs"},
			{Path: "tasks", Name: "tasks", Type: "dir", Size: 0, Category: ""},
		},
		EntryPoints: []string{
			"/workspace/docs/README.md",
			"/workspace/docs/tasks/README.md",
		},
		ByCategory: map[string][]string{
			"docs": {"/workspace/docs/README.md", "/workspace/docs/tasks/README.md"},
		},
	}

	result := buildResearchPlanningPrompt("Test Project", manifest, "Preserve existing structure")

	expectedContents := []string{
		"Test Project",
		"/workspace/docs",
		"5 files",
		"## Detected Entry Points",
		"/workspace/docs/README.md",
		"/workspace/docs/tasks/README.md",
		"## Directory Structure",
		"## Research Instructions",
		"Read, Glob, and Grep tools",
		"Preserve existing structure",
		"## Output Format",
		"### task-N: Task Title",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(result, expected) {
			t.Errorf("buildResearchPlanningPrompt() missing %q in result", expected)
		}
	}
}

func TestBuildResearchPlanningPrompt_NoEntryPoints(t *testing.T) {
	manifest := &ResearchManifest{
		BasePath:    "/workspace/docs",
		FileCount:   2,
		Structure:   []DirEntry{},
		EntryPoints: []string{},
		ByCategory:  map[string][]string{},
	}

	result := buildResearchPlanningPrompt("Test", manifest, "")

	if strings.Contains(result, "## Detected Entry Points") {
		t.Error("buildResearchPlanningPrompt() should not include Entry Points section when empty")
	}
}

func TestBuildResearchPlanningPrompt_NoCustomInstructions(t *testing.T) {
	manifest := &ResearchManifest{
		BasePath:   "/test",
		FileCount:  1,
		Structure:  []DirEntry{},
		ByCategory: map[string][]string{},
	}

	result := buildResearchPlanningPrompt("Test", manifest, "")

	if strings.Contains(result, "## Custom Instructions") {
		t.Error("buildResearchPlanningPrompt() should not include Custom Instructions section when empty")
	}
}

func TestTopologicalSortWithParents(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []*storage.QueuedTask
		wantErr bool
		check   func(t *testing.T, sorted []*storage.QueuedTask)
	}{
		{
			name: "no dependencies or parents",
			tasks: []*storage.QueuedTask{
				{ID: "task-3", Priority: 3},
				{ID: "task-1", Priority: 1},
				{ID: "task-2", Priority: 2},
			},
			wantErr: false,
			check: func(t *testing.T, sorted []*storage.QueuedTask) {
				t.Helper()
				// Should sort by priority when no deps
				if sorted[0].ID != "task-1" {
					t.Errorf("expected task-1 first (priority 1), got %s", sorted[0].ID)
				}
			},
		},
		{
			name: "parent-child relationships",
			tasks: []*storage.QueuedTask{
				{ID: "task-2", Priority: 2, ParentID: "task-1"},
				{ID: "task-1", Priority: 1},
				{ID: "task-3", Priority: 3, ParentID: "task-1"},
			},
			wantErr: false,
			check: func(t *testing.T, sorted []*storage.QueuedTask) {
				t.Helper()
				// task-1 must come before its children
				task1Idx := -1
				task2Idx := -1
				task3Idx := -1
				for i, task := range sorted {
					switch task.ID {
					case "task-1":
						task1Idx = i
					case "task-2":
						task2Idx = i
					case "task-3":
						task3Idx = i
					}
				}
				if task1Idx >= task2Idx {
					t.Errorf("task-1 should come before task-2 (parent before child)")
				}
				if task1Idx >= task3Idx {
					t.Errorf("task-1 should come before task-3 (parent before child)")
				}
			},
		},
		{
			name: "depends-on relationships",
			tasks: []*storage.QueuedTask{
				{ID: "task-3", Priority: 3, DependsOn: []string{"task-2"}},
				{ID: "task-1", Priority: 1},
				{ID: "task-2", Priority: 2, DependsOn: []string{"task-1"}},
			},
			wantErr: false,
			check: func(t *testing.T, sorted []*storage.QueuedTask) {
				t.Helper()
				// Order should be: task-1 -> task-2 -> task-3
				if sorted[0].ID != "task-1" {
					t.Errorf("expected task-1 first, got %s", sorted[0].ID)
				}
				if sorted[1].ID != "task-2" {
					t.Errorf("expected task-2 second, got %s", sorted[1].ID)
				}
				if sorted[2].ID != "task-3" {
					t.Errorf("expected task-3 third, got %s", sorted[2].ID)
				}
			},
		},
		{
			name: "mixed parent and depends-on",
			tasks: []*storage.QueuedTask{
				{ID: "task-1", Priority: 1},
				{ID: "task-2", Priority: 2, ParentID: "task-1"},
				{ID: "task-3", Priority: 3, ParentID: "task-1", DependsOn: []string{"task-2"}},
			},
			wantErr: false,
			check: func(t *testing.T, sorted []*storage.QueuedTask) {
				t.Helper()
				// task-1 must come first (parent of both)
				// task-2 must come before task-3 (dependency)
				task1Idx := -1
				task2Idx := -1
				task3Idx := -1
				for i, task := range sorted {
					switch task.ID {
					case "task-1":
						task1Idx = i
					case "task-2":
						task2Idx = i
					case "task-3":
						task3Idx = i
					}
				}
				if task1Idx >= task2Idx || task1Idx >= task3Idx {
					t.Errorf("task-1 should come before both children")
				}
				if task2Idx >= task3Idx {
					t.Errorf("task-2 should come before task-3 (dependency)")
				}
			},
		},
		{
			name: "circular dependency",
			tasks: []*storage.QueuedTask{
				{ID: "task-1", DependsOn: []string{"task-2"}},
				{ID: "task-2", DependsOn: []string{"task-1"}},
			},
			wantErr: true,
		},
		{
			name: "circular parent-child",
			tasks: []*storage.QueuedTask{
				{ID: "task-1", ParentID: "task-2"},
				{ID: "task-2", ParentID: "task-1"},
			},
			wantErr: true,
		},
		{
			name: "nested subtasks",
			tasks: []*storage.QueuedTask{
				{ID: "task-1", Priority: 1},
				{ID: "task-2", Priority: 2, ParentID: "task-1"},
				{ID: "task-3", Priority: 3, ParentID: "task-2"}, // Nested (grandchild of task-1)
			},
			wantErr: false,
			check: func(t *testing.T, sorted []*storage.QueuedTask) {
				t.Helper()
				// Order should be: task-1 -> task-2 -> task-3
				if sorted[0].ID != "task-1" {
					t.Errorf("expected task-1 first, got %s", sorted[0].ID)
				}
				if sorted[1].ID != "task-2" {
					t.Errorf("expected task-2 second, got %s", sorted[1].ID)
				}
				if sorted[2].ID != "task-3" {
					t.Errorf("expected task-3 third, got %s", sorted[2].ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted, err := topologicalSortWithParents(tt.tasks)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(sorted) != len(tt.tasks) {
				t.Errorf("sorted length = %d, want %d", len(sorted), len(tt.tasks))
			}

			if tt.check != nil {
				tt.check(t, sorted)
			}
		})
	}
}

func TestValidateSubmitSelection_MissingParents(t *testing.T) {
	queue := &storage.TaskQueue{
		ID: "test-queue",
		Tasks: []*storage.QueuedTask{
			{ID: "task-1", Status: storage.TaskStatusReady},
			{ID: "task-2", Status: storage.TaskStatusReady, ParentID: "task-1"},
			{ID: "task-3", Status: storage.TaskStatusReady, ParentID: "task-nonexistent"},
		},
	}

	// Select only task-3 which has a non-existent parent
	selected := []*storage.QueuedTask{queue.Tasks[2]}
	opts := SubmitOptions{TaskIDs: []string{"task-3"}}

	err := validateSubmitSelection(queue, selected, opts)
	if err == nil {
		t.Error("expected error for missing parent")
	}
	if !strings.Contains(err.Error(), "missing parents") {
		t.Errorf("error should mention missing parents, got: %v", err)
	}
}

func TestValidateSubmitSelection_ParentInSelection(t *testing.T) {
	queue := &storage.TaskQueue{
		ID: "test-queue",
		Tasks: []*storage.QueuedTask{
			{ID: "task-1", Status: storage.TaskStatusReady},
			{ID: "task-2", Status: storage.TaskStatusReady, ParentID: "task-1"},
		},
	}

	// Select both parent and child - should be valid
	selected := queue.Tasks
	opts := SubmitOptions{TaskIDs: []string{"task-1", "task-2"}}

	err := validateSubmitSelection(queue, selected, opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSubmitSelection_ParentAlreadySubmitted(t *testing.T) {
	queue := &storage.TaskQueue{
		ID: "test-queue",
		Tasks: []*storage.QueuedTask{
			{ID: "task-1", Status: storage.TaskStatusSubmitted, ExternalID: "EXT-1"},
			{ID: "task-2", Status: storage.TaskStatusReady, ParentID: "task-1"},
		},
	}

	// Select only child - parent already submitted, should be valid
	selected := []*storage.QueuedTask{queue.Tasks[1]}
	opts := SubmitOptions{TaskIDs: []string{"task-2"}}

	err := validateSubmitSelection(queue, selected, opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBuildProjectPlanningPrompt_IncludesParentField(t *testing.T) {
	result := buildProjectPlanningPrompt("Test", "content", "")

	expectedContents := []string{
		"**Parent**: task-X (if this is a subtask)",
		"**Parent**: Hierarchical grouping",
		"**Depends on**: Execution ordering",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(result, expected) {
			t.Errorf("buildProjectPlanningPrompt() missing %q in result", expected)
		}
	}
}
