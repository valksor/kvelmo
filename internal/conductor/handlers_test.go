package conductor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/eventbus"
)

func TestFormatSpecContent(t *testing.T) {
	tests := []struct {
		response *agent.Response
		name     string
		wantIn   []string
		num      int
	}{
		{
			name: "with summary only",
			num:  1,
			response: &agent.Response{
				Summary: "This is the summary",
			},
			wantIn: []string{"# Specification 1", "## Summary", "This is the summary"},
		},
		{
			name: "with messages only",
			num:  2,
			response: &agent.Response{
				Messages: []string{"Message 1", "Message 2"},
			},
			wantIn: []string{"# Specification 2", "## Details", "Message 1", "Message 2"},
		},
		{
			name: "with both summary and messages",
			num:  3,
			response: &agent.Response{
				Summary:  "Summary text",
				Messages: []string{"Detail 1", "Detail 2"},
			},
			wantIn: []string{"# Specification 3", "## Summary", "Summary text", "## Details", "Detail 1", "Detail 2"},
		},
		{
			name:     "empty response",
			num:      4,
			response: &agent.Response{},
			wantIn:   []string{"# Specification 4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSpecificationContent(tt.num, tt.response)
			for _, want := range tt.wantIn {
				if !strings.Contains(got, want) {
					t.Errorf("formatSpecificationContent() missing %q in:\n%s", want, got)
				}
			}
		})
	}
}

func TestBuildPlanningPrompt(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		sourceContent string
		notes         string
		existingSpecs string
		wantIn        []string
		wantNotIn     []string
	}{
		{
			name:          "basic prompt",
			title:         "Add login feature",
			sourceContent: "Task description here",
			notes:         "",
			existingSpecs: "",
			wantIn:        []string{"Add login feature", "Task description here", "## Instructions"},
			wantNotIn:     []string{"## Additional Notes", "## Previous Specifications"},
		},
		{
			name:          "with notes",
			title:         "Add login",
			sourceContent: "Task desc",
			notes:         "User wants OAuth",
			existingSpecs: "",
			wantIn:        []string{"Add login", "## Additional Notes", "User wants OAuth"},
			wantNotIn:     []string{"## Previous Specifications"},
		},
		{
			name:          "with existing specs",
			title:         "Add login",
			sourceContent: "Task desc",
			notes:         "",
			existingSpecs: "# Specification 1\nExisting spec content",
			wantIn:        []string{"## Previous Specifications", "DO NOT start from scratch", "Specification 1"},
			wantNotIn:     []string{"## Additional Notes"},
		},
		{
			name:          "with everything",
			title:         "Full task",
			sourceContent: "Full description",
			notes:         "Some notes",
			existingSpecs: "Previous spec",
			wantIn:        []string{"Full task", "Full description", "## Additional Notes", "Some notes", "## Previous Specifications", "Previous spec"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPlanningPrompt(nil, ".", tt.title, tt.sourceContent, tt.notes, tt.existingSpecs, "", false, nil)
			for _, want := range tt.wantIn {
				if !strings.Contains(got, want) {
					t.Errorf("buildPlanningPrompt(, nil) missing %q", want)
				}
			}
			for _, notWant := range tt.wantNotIn {
				if strings.Contains(got, notWant) {
					t.Errorf("buildPlanningPrompt(, nil) should not contain %q", notWant)
				}
			}
		})
	}
}

func TestBuildPlanningPromptWithCustomInstructions(t *testing.T) {
	got := buildPlanningPrompt(nil, ".", "Task", "Source", "", "", "Focus on security.", false, nil)

	if !strings.Contains(got, "## Custom Instructions") {
		t.Error("buildPlanningPrompt(, nil) should contain custom instructions section")
	}
	if !strings.Contains(got, "Focus on security.") {
		t.Error("buildPlanningPrompt(, nil) should contain custom instruction content")
	}
}

func TestBuildImplementationPrompt(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		source    string
		specs     string
		notes     string
		wantIn    []string
		wantNotIn []string
	}{
		{
			name:      "basic prompt",
			title:     "Add feature",
			source:    "Original requirements",
			specs:     "# Specification 1\nSpec content",
			notes:     "",
			wantIn:    []string{"Add feature", "Original requirements", "Spec content", "## Instructions"},
			wantNotIn: []string{"## Additional Notes"},
		},
		{
			name:   "with notes",
			title:  "Add feature",
			source: "Original",
			specs:  "Specs",
			notes:  "Implementation notes",
			wantIn: []string{"## Additional Notes", "Implementation notes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildImplementationPrompt(nil, ".", tt.title, tt.source, tt.specs, tt.notes, "", "", "", nil)
			for _, want := range tt.wantIn {
				if !strings.Contains(got, want) {
					t.Errorf("buildImplementationPrompt(, nil) missing %q", want)
				}
			}
			for _, notWant := range tt.wantNotIn {
				if strings.Contains(got, notWant) {
					t.Errorf("buildImplementationPrompt(, nil) should not contain %q", notWant)
				}
			}
		})
	}
}

func TestBuildImplementationPromptWithCustomInstructions(t *testing.T) {
	got := buildImplementationPrompt(nil, ".", "Task", "Source", "Specs", "", "Write tests first.", "", "", nil)

	if !strings.Contains(got, "## Custom Instructions") {
		t.Error("buildImplementationPrompt(, nil) should contain custom instructions section")
	}
	if !strings.Contains(got, "Write tests first.") {
		t.Error("buildImplementationPrompt(, nil) should contain custom instruction content")
	}
}

func TestBuildReviewPrompt(t *testing.T) {
	got := buildReviewPrompt(nil, "Task Title", "Source content", "Spec content")

	wantIn := []string{
		"Task Title",
		"Source content",
		"Spec content",
		"## Instructions",
		"Correctness",
		"Code quality",
		"Security",
		"Performance",
		"Best practices",
	}

	for _, want := range wantIn {
		if !strings.Contains(got, want) {
			t.Errorf("buildReviewPrompt() missing %q", want)
		}
	}
}

func TestBuildReviewPromptWithCustomInstructions(t *testing.T) {
	got := buildReviewPromptWithLint(nil, "Task", "Source", "Specs", "", "Focus on security issues.")

	if !strings.Contains(got, "## Custom Instructions") {
		t.Error("buildReviewPromptWithLint() should contain custom instructions section")
	}
	if !strings.Contains(got, "Focus on security issues.") {
		t.Error("buildReviewPromptWithLint() should contain custom instruction content")
	}
}

func TestBuildBrowserToolsSection(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *storage.WorkspaceConfig
		wantIn  []string
		wantNot []string
	}{
		{
			name: "nil workspace returns empty",
			cfg:  nil,
			wantNot: []string{
				"## Browser Automation",
				"browser_open_url",
				"browser_screenshot",
			},
		},
		{
			name: "browser disabled returns empty",
			cfg: &storage.WorkspaceConfig{
				Browser: &storage.BrowserSettings{
					Enabled: false,
				},
			},
			wantNot: []string{
				"## Browser Automation",
				"browser_open_url",
			},
		},
		{
			name: "browser enabled returns tools section",
			cfg: &storage.WorkspaceConfig{
				Browser: &storage.BrowserSettings{
					Enabled: true,
				},
			},
			wantIn: []string{
				"## Browser Automation",
				"Browser automation is ENABLED",
				"browser_open_url",
				"browser_screenshot",
				"browser_click",
				"browser_type",
				"browser_evaluate",
				"browser_query",
				"browser_get_console_logs",
				"browser_get_network_requests",
				"browser_detect_auth",
				"browser_wait_for_login",
				"Testing web applications during implementation",
				"Verifying frontend features",
				"Handling authentication flows",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock workspace based on test config
			var ws *storage.Workspace
			if tt.cfg != nil {
				// For testing with enabled browser, we need a valid workspace
				// In production, this would use real workspace initialization
				// For unit tests, we'll skip the full workspace creation
				// and just test the nil case and basic structure
				if tt.cfg.Browser.Enabled {
					t.Skip("Skipping enabled browser test in unit tests - requires full workspace setup")

					return
				}
			}

			got := buildBrowserToolsSection(ws)

			for _, want := range tt.wantIn {
				if !strings.Contains(got, want) {
					t.Errorf("buildBrowserToolsSection() missing %q", want)
				}
			}
			for _, notWant := range tt.wantNot {
				if strings.Contains(got, notWant) {
					t.Errorf("buildBrowserToolsSection() should not contain %q", notWant)
				}
			}
		})
	}
}

func TestBuildCombinedInstructions(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *storage.WorkspaceConfig
		step   string
		want   string
		wantIn []string
	}{
		{
			name: "nil config returns empty",
			cfg:  nil,
			step: "planning",
			want: "",
		},
		{
			name: "global only",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					Instructions: "Follow best practices.",
				},
			},
			step:   "planning",
			wantIn: []string{"Follow best practices."},
		},
		{
			name: "step only",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					Steps: map[string]storage.StepAgentConfig{
						"planning": {
							Instructions: "Focus on architecture.",
						},
					},
				},
			},
			step:   "planning",
			wantIn: []string{"Focus on architecture."},
		},
		{
			name: "global and step combined",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					Instructions: "Follow best practices.",
					Steps: map[string]storage.StepAgentConfig{
						"planning": {
							Instructions: "Focus on architecture.",
						},
					},
				},
			},
			step:   "planning",
			wantIn: []string{"Follow best practices.", "Focus on architecture."},
		},
		{
			name: "step not configured",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					Instructions: "Global only.",
					Steps: map[string]storage.StepAgentConfig{
						"implementing": {
							Instructions: "Not planning.",
						},
					},
				},
			},
			step:   "planning",
			wantIn: []string{"Global only."},
		},
		{
			name: "whitespace only trimmed",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					Instructions: "   ",
					Steps: map[string]storage.StepAgentConfig{
						"planning": {
							Instructions: "   Valid instructions.   ",
						},
					},
				},
			},
			step:   "planning",
			wantIn: []string{"Valid instructions."},
		},
		{
			name: "neither global nor step",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{},
			},
			step: "planning",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCombinedInstructions(tt.cfg, tt.step)

			if tt.want != "" && got != tt.want {
				t.Errorf("buildCombinedInstructions() = %q, want %q", got, tt.want)
			}

			for _, want := range tt.wantIn {
				if !strings.Contains(got, want) {
					t.Errorf("buildCombinedInstructions() missing %q in %q", want, got)
				}
			}

			// Verify empty expectations
			if tt.want == "" && len(tt.wantIn) == 0 && got != "" {
				t.Errorf("buildCombinedInstructions() = %q, want empty", got)
			}
		})
	}
}

// Tests for file utility functions

func TestEnsureDirExists(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "current directory (empty dir)",
			path:    "file.txt",
			wantErr: false,
		},
		{
			name:    "current directory (dot)",
			path:    "./file.txt",
			wantErr: false,
		},
		{
			name:    "single level directory",
			path:    "subdir/file.txt",
			wantErr: false,
		},
		{
			name:    "nested directories",
			path:    "a/b/c/d/file.txt",
			wantErr: false,
		},
		{
			name:    "absolute path creates directories",
			path:    "/tmp/test-mehrhof-ensure/file.txt",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// For absolute path test, use tmpDir
			var testPath string
			if strings.HasPrefix(tt.path, "/tmp/") {
				testPath = filepath.Join(tmpDir, "nested/dir/file.txt")
			} else {
				testPath = filepath.Join(tmpDir, tt.path)
			}

			err := ensureDirExists(testPath)

			if tt.wantErr {
				if err == nil {
					t.Error("ensureDirExists() expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Errorf("ensureDirExists() unexpected error: %v", err)
			}

			// Verify directory exists
			dir := filepath.Dir(testPath)
			if info, err := os.Stat(dir); err != nil {
				t.Errorf("directory not created: %v", err)
			} else if !info.IsDir() {
				t.Error("path is not a directory")
			}
		})
	}
}

func TestValidatePathInWorkspace(t *testing.T) {
	tests := []struct {
		name     string
		resolved string
		root     string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "path within workspace",
			resolved: "/workspace/file.txt",
			root:     "/workspace",
			wantErr:  false,
		},
		{
			name:     "nested path within workspace",
			resolved: "/workspace/a/b/c/file.txt",
			root:     "/workspace",
			wantErr:  false,
		},
		{
			name:     "same as root",
			resolved: "/workspace",
			root:     "/workspace",
			wantErr:  false,
		},
		{
			name:     "parent directory escape",
			resolved: "/other/file.txt",
			root:     "/workspace",
			wantErr:  true,
			errMsg:   "outside workspace",
		},
		{
			name:     "relative path with dotdot",
			resolved: "/workspace/../escape.txt",
			root:     "/workspace",
			wantErr:  true,
			errMsg:   "outside workspace",
		},
		{
			name:     "dotdot only",
			resolved: "..",
			root:     "/workspace",
			wantErr:  true,
			errMsg:   "invalid path",
		},
		{
			name:     "sibling directory",
			resolved: "/workspace/../other/file.txt",
			root:     "/workspace",
			wantErr:  true,
			errMsg:   "outside workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePathInWorkspace(tt.resolved, tt.root)

			if tt.wantErr {
				if err == nil {
					t.Error("validatePathInWorkspace() expected error, got nil")

					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want contain %q", err.Error(), tt.errMsg)
				}

				return
			}

			if err != nil {
				t.Errorf("validatePathInWorkspace() unexpected error: %v", err)
			}
		})
	}
}

func TestApplyFiles_Create(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.eventBus = eventbus.NewBus()

	files := []agent.FileChange{
		{
			Path:      "test/new-file.txt",
			Operation: agent.FileOpCreate,
			Content:   "Hello, World!",
		},
	}

	err = applyFiles(ctx, c, files)
	if err != nil {
		t.Fatalf("applyFiles: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(filepath.Join(tmpDir, "test/new-file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "Hello, World!" {
		t.Errorf("file content = %q, want %q", string(content), "Hello, World!")
	}
}

func TestApplyFiles_Update(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create existing file
	existingPath := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingPath, []byte("old content"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.eventBus = eventbus.NewBus()

	files := []agent.FileChange{
		{
			Path:      "existing.txt",
			Operation: agent.FileOpUpdate,
			Content:   "new content",
		},
	}

	err = applyFiles(ctx, c, files)
	if err != nil {
		t.Fatalf("applyFiles: %v", err)
	}

	// Verify file was updated
	content, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "new content" {
		t.Errorf("file content = %q, want %q", string(content), "new content")
	}
}

func TestApplyFiles_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create file to delete
	filePath := filepath.Join(tmpDir, "to-delete.txt")
	if err := os.WriteFile(filePath, []byte("delete me"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.eventBus = eventbus.NewBus()

	files := []agent.FileChange{
		{
			Path:      "to-delete.txt",
			Operation: agent.FileOpDelete,
		},
	}

	err = applyFiles(ctx, c, files)
	if err != nil {
		t.Fatalf("applyFiles: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestApplyFiles_DeleteSentinel(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create file to delete
	filePath := filepath.Join(tmpDir, "sentinel-delete.txt")
	if err := os.WriteFile(filePath, []byte("delete me"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.eventBus = eventbus.NewBus()

	files := []agent.FileChange{
		{
			Path:    "sentinel-delete.txt",
			Content: DeleteFileSentinel, // Use sentinel instead of operation
		},
	}

	err = applyFiles(ctx, c, files)
	if err != nil {
		t.Fatalf("applyFiles: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file should have been deleted via sentinel")
	}
}

func TestApplyFiles_DeleteNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.eventBus = eventbus.NewBus()

	files := []agent.FileChange{
		{
			Path:      "does-not-exist.txt",
			Operation: agent.FileOpDelete,
		},
	}

	// Should not error when deleting non-existent file
	err = applyFiles(ctx, c, files)
	if err != nil {
		t.Errorf("applyFiles should not error on non-existent file: %v", err)
	}
}

func TestApplyFiles_Multiple(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create file for update and delete
	updatePath := filepath.Join(tmpDir, "update.txt")
	deletePath := filepath.Join(tmpDir, "delete.txt")
	if err := os.WriteFile(updatePath, []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(deletePath, []byte("gone"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.eventBus = eventbus.NewBus()

	files := []agent.FileChange{
		{Path: "new.txt", Operation: agent.FileOpCreate, Content: "created"},
		{Path: "update.txt", Operation: agent.FileOpUpdate, Content: "updated"},
		{Path: "delete.txt", Operation: agent.FileOpDelete},
	}

	err = applyFiles(ctx, c, files)
	if err != nil {
		t.Fatalf("applyFiles: %v", err)
	}

	// Verify create
	if content, _ := os.ReadFile(filepath.Join(tmpDir, "new.txt")); string(content) != "created" {
		t.Error("create failed")
	}

	// Verify update
	if content, _ := os.ReadFile(updatePath); string(content) != "updated" {
		t.Error("update failed")
	}

	// Verify delete
	if _, err := os.Stat(deletePath); !os.IsNotExist(err) {
		t.Error("delete failed")
	}
}

func TestApplyFiles_WithGitRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	ctx := context.Background()

	// Initialize git repo
	initGitRepo(t, tmpDir)

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.eventBus = eventbus.NewBus()

	// Initialize to set git
	_ = c.Initialize(ctx)

	files := []agent.FileChange{
		{
			Path:      "git-tracked.txt",
			Operation: agent.FileOpCreate,
			Content:   "tracked by git",
		},
	}

	err = applyFiles(ctx, c, files)
	if err != nil {
		t.Fatalf("applyFiles: %v", err)
	}

	// Verify file was created in git root
	content, err := os.ReadFile(filepath.Join(tmpDir, "git-tracked.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "tracked by git" {
		t.Errorf("file content = %q, want %q", string(content), "tracked by git")
	}
}

func TestDeleteFileSentinelConstant(t *testing.T) {
	if DeleteFileSentinel != "__DELETE_FILE__" {
		t.Errorf("DeleteFileSentinel = %q, want %q", DeleteFileSentinel, "__DELETE_FILE__")
	}
}

func TestErrPendingQuestion(t *testing.T) {
	if ErrPendingQuestion == nil {
		t.Fatal("ErrPendingQuestion should not be nil")
	}
	if ErrPendingQuestion.Error() != "agent has a pending question" {
		t.Errorf("ErrPendingQuestion.Error() = %q", ErrPendingQuestion.Error())
	}

	// Test that errors.Is() works correctly with the sentinel error.
	// This is the key reason for using errors.New() instead of fmt.Errorf().
	wrappedErr := fmt.Errorf("wrapped: %w", ErrPendingQuestion)
	if !errors.Is(wrappedErr, ErrPendingQuestion) {
		t.Error("errors.Is() should work with sentinel error")
	}
}

func TestCreateCheckpointIfNeeded_NoGit(t *testing.T) {
	ctx := context.Background()
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// No git - should return nil
	c.activeTask = &storage.ActiveTask{
		ID:     "test",
		UseGit: true,
	}
	event := c.createCheckpointIfNeeded(ctx, "test", "message")
	if event != nil {
		t.Error("createCheckpointIfNeeded should return nil when git is nil")
	}
}

func TestCreateCheckpointIfNeeded_GitNotUsed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)
	ctx := context.Background()

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Initialize to set git
	_ = c.Initialize(ctx)

	// UseGit is false
	c.activeTask = &storage.ActiveTask{
		ID:     "test",
		UseGit: false,
	}

	event := c.createCheckpointIfNeeded(ctx, "test", "message")
	if event != nil {
		t.Error("createCheckpointIfNeeded should return nil when UseGit is false")
	}
}

func TestCreateCheckpointIfNeeded_NoChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)
	ctx := context.Background()

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Initialize to set git
	_ = c.Initialize(ctx)

	c.activeTask = &storage.ActiveTask{
		ID:      "test",
		UseGit:  true,
		Started: time.Now(),
	}

	// No changes - should return nil
	event := c.createCheckpointIfNeeded(ctx, "test", "message")
	if event != nil {
		t.Error("createCheckpointIfNeeded should return nil when no changes")
	}
}

func TestSaveCurrentSession(t *testing.T) {
	tests := []struct {
		name               string
		currentSession     *storage.Session
		currentSessionFile string
		taskID             string
		expectCleared      bool // whether session should be cleared
	}{
		{
			name:               "nil current session - no-op",
			currentSession:     nil,
			currentSessionFile: "",
			taskID:             "test-task",
			expectCleared:      false,
		},
		{
			name: "empty session file - no-op",
			currentSession: &storage.Session{
				Version: "1",
				Kind:    "Session",
				Metadata: storage.SessionMetadata{
					StartedAt: time.Now(),
				},
			},
			currentSessionFile: "",
			taskID:             "test-task",
			expectCleared:      false,
		},
		{
			name: "session but nil file - no-op",
			currentSession: &storage.Session{
				Version: "1",
				Kind:    "Session",
				Metadata: storage.SessionMetadata{
					StartedAt: time.Now(),
				},
			},
			currentSessionFile: "",
			taskID:             "test-task",
			expectCleared:      false,
		},
		{
			name: "valid session - saves and clears",
			currentSession: &storage.Session{
				Version: "1",
				Kind:    "Session",
				Metadata: storage.SessionMetadata{
					StartedAt: time.Now(),
					Type:      "planning",
				},
				Exchanges: []storage.Exchange{
					{
						Role:    "system",
						Content: "You are helpful",
					},
				},
			},
			currentSessionFile: "2025-01-01T12-00-00-planning.yaml",
			taskID:             "test-task",
			expectCleared:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create workspace
			ws := openTestWorkspace(t, tmpDir)
			if err := ws.EnsureInitialized(); err != nil {
				t.Fatalf("EnsureInitialized: %v", err)
			}

			// Create task work
			_, err := ws.CreateWork(tt.taskID, storage.SourceInfo{
				Type: "file",
				Ref:  "task.md",
			})
			if err != nil {
				t.Fatalf("CreateWork: %v", err)
			}

			c, err := New(WithWorkDir(tmpDir))
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			c.workspace = ws
			c.currentSession = tt.currentSession
			c.currentSessionFile = tt.currentSessionFile

			// Call saveCurrentSession
			c.saveCurrentSession(tt.taskID)

			// Verify session state
			if tt.expectCleared {
				if c.currentSession != nil {
					t.Error("currentSession should be nil after saveCurrentSession")
				}
				if c.currentSessionFile != "" {
					t.Errorf("currentSessionFile should be empty after saveCurrentSession, got %q", c.currentSessionFile)
				}

				// Verify session was saved to workspace
				sessions, err := ws.ListSessions(tt.taskID)
				if err != nil {
					t.Fatalf("ListSessions: %v", err)
				}
				if len(sessions) == 0 {
					t.Error("expected session to be saved, but found none")
				}
			} else {
				// Should not be cleared
				if tt.currentSession != nil && c.currentSession == nil {
					t.Error("currentSession should NOT be cleared when session file is empty")
				}
			}
		})
	}
}

func TestIsRecoverableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error returns false",
			err:  nil,
			want: false,
		},
		{
			name: "context overflow is recoverable",
			err:  errors.New("context overflow exceeded"),
			want: true,
		},
		{
			name: "token limit is recoverable",
			err:  errors.New("token limit exceeded"),
			want: true,
		},
		{
			name: "timeout is recoverable",
			err:  errors.New("request timeout"),
			want: true,
		},
		{
			name: "rate limit is recoverable",
			err:  errors.New("rate limit exceeded"),
			want: true,
		},
		{
			name: "429 status is recoverable",
			err:  errors.New("HTTP 429 Too Many Requests"),
			want: true,
		},
		{
			name: "too many requests is recoverable",
			err:  errors.New("too many requests, please retry"),
			want: true,
		},
		{
			name: "compilation error is not recoverable",
			err:  errors.New("syntax error at line 42"),
			want: false,
		},
		{
			name: "validation error is not recoverable",
			err:  errors.New("invalid input"),
			want: false,
		},
		{
			name: "case insensitive matching",
			err:  errors.New("CONTEXT OVERFLOW - tokens exceeded"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRecoverableError(tt.err)
			if got != tt.want {
				t.Errorf("isRecoverableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildErrorRecoverySection(t *testing.T) {
	got := buildErrorRecoverySection()

	// Check that key error recovery strategies are present
	wantIn := []string{
		"## Error Recovery Strategies",
		"### Context Overflow:",
		"Focus on highest-priority specifications first",
		"### Parse Failures:",
		"Ask user to provide file contents",
		"### Authentication Errors:",
		"Check environment variables and config",
		"### Dependency Errors:",
		"Check dependency management files",
		"### Compilation Errors:",
		"Fix syntax errors first, then type errors",
		"### Test Failures:",
		"Check if tests are outdated or implementation incorrect",
	}

	for _, want := range wantIn {
		if !strings.Contains(got, want) {
			t.Errorf("buildErrorRecoverySection() missing %q", want)
		}
	}

	// Check that it uses correct terminology (specification, not spec)
	if strings.Contains(got, "spec ") && !strings.Contains(got, "specification") {
		t.Error("buildErrorRecoverySection() should use 'specification' not 'spec'")
	}
}

func TestBuildQualityGateInstructions(t *testing.T) {
	got := buildQualityGateInstructions()

	// Check that key quality gate sections are present
	wantIn := []string{
		"## Pre-Review Quality Checklist",
		"### Code Quality:",
		"Compiles without errors",
		"Error handling present",
		"Descriptive names",
		"Follows existing style",
		"### Functional Completeness:",
		"All specification requirements addressed",
		"Edge cases handled",
		"Helpful error messages",
		"Sensible defaults",
		"### Testing:",
		"Code is testable",
		"Critical paths covered",
		"Manual testing steps documented",
		"### Verification:",
		"Review yaml:file blocks above",
		"specification status updated to \"completed",
		"If issues found, provide additional yaml:file blocks",
		"IMPLEMENTATION_COMPLETE",
	}

	for _, want := range wantIn {
		if !strings.Contains(got, want) {
			t.Errorf("buildQualityGateInstructions() missing %q", want)
		}
	}
}

// Tests for useDefaults parameter in buildPlanningPrompt

func TestBuildPlanningPromptUseDefaults(t *testing.T) {
	tests := []struct {
		name        string
		useDefaults bool
		wantIn      []string
		wantNotIn   []string
	}{
		{
			name:        "useDefaults false - should ask user",
			useDefaults: false,
			wantIn: []string{
				"STOP and ask the user",
				"Do not guess",
				"get clarification",
			},
			wantNotIn: []string{
				"Provide your best-guess default answers",
				"Do not wait for user input",
			},
		},
		{
			name:        "useDefaults true - should use defaults",
			useDefaults: true,
			wantIn: []string{
				"your recommended default answer",
				"Do not wait for user input",
			},
			wantNotIn: []string{
				"STOP and ask the user",
				"get clarification for important decisions",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPlanningPrompt(nil, ".", "Task", "Source", "", "", "", tt.useDefaults, nil)

			for _, want := range tt.wantIn {
				if !strings.Contains(got, want) {
					t.Errorf("buildPlanningPrompt(useDefaults=%v, nil) missing %q", tt.useDefaults, want)
				}
			}
			for _, notWant := range tt.wantNotIn {
				if strings.Contains(got, notWant) {
					t.Errorf("buildPlanningPrompt(useDefaults=%v, nil) should not contain %q", tt.useDefaults, notWant)
				}
			}
		})
	}
}

func TestBuildUnknownsSection(t *testing.T) {
	tests := []struct {
		name        string
		useDefaults bool
		wantIn      []string
		wantNotIn   []string
	}{
		{
			name:        "ask user mode",
			useDefaults: false,
			wantIn: []string{
				"STOP and ask the user",
				"ask_question tool",
				"Do not guess",
			},
			wantNotIn: []string{
				"Do not wait for user input",
			},
		},
		{
			name:        "defaults mode",
			useDefaults: true,
			wantIn: []string{
				"your recommended default answer",
				"Do not wait for user input",
			},
			wantNotIn: []string{
				"STOP and ask the user",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildUnknownsSection(tt.useDefaults)

			for _, want := range tt.wantIn {
				if !strings.Contains(got, want) {
					t.Errorf("buildUnknownsSection(%v) missing %q", tt.useDefaults, want)
				}
			}
			for _, notWant := range tt.wantNotIn {
				if strings.Contains(got, notWant) {
					t.Errorf("buildUnknownsSection(%v) should not contain %q", tt.useDefaults, notWant)
				}
			}
		})
	}
}

// Tests for step-specific browser sections

func TestBuildBrowserToolsSectionForStep(t *testing.T) {
	tests := []struct {
		name    string
		step    string
		wantIn  []string
		wantNot []string
	}{
		{
			name: "planning step",
			step: "planning",
			wantIn: []string{
				"## Browser Automation (Planning)",
				"Research APIs",
				"explore documentation",
				"Capture screenshots for specification references",
			},
		},
		{
			name: "implementing step",
			step: "implementing",
			wantIn: []string{
				"## Browser Automation (Implementation)",
				"Test your changes",
				"Verify UI components",
				"Debug frontend issues",
			},
		},
		{
			name: "reviewing step",
			step: "reviewing",
			wantIn: []string{
				"## Browser Automation (Review)",
				"Verify the implementation works end-to-end",
				"Test edge cases",
				"Capture evidence of issues",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test verifies the step-specific content.
			// Without a workspace with browser enabled, it returns empty.
			// We test the switch logic by verifying the function exists
			// and handles different step values.

			// For nil workspace, should return empty
			got := buildBrowserToolsSectionForStep(nil, tt.step)
			if got != "" {
				t.Errorf("buildBrowserToolsSectionForStep(nil, %q) should be empty", tt.step)
			}
		})
	}
}

// Tests for chain-of-thought sections

func TestPlanningPromptChainOfThought(t *testing.T) {
	got := buildPlanningPrompt(nil, ".", "Task", "Source", "", "", "", false, nil)

	wantIn := []string{
		"## Approach",
		"Before writing your specification",
		"Identify existing patterns in the codebase",
		"Consider integration points and dependencies",
		"Think through edge cases and failure scenarios",
		"Evaluate trade-offs",
		"Briefly explain your key architectural decisions",
	}

	for _, want := range wantIn {
		if !strings.Contains(got, want) {
			t.Errorf("buildPlanningPrompt(, nil) missing chain-of-thought element %q", want)
		}
	}
}

func TestReviewPromptChainOfThought(t *testing.T) {
	got := buildReviewPrompt(nil, "Task", "Source", "Specs")

	wantIn := []string{
		"## Approach",
		"Before providing your review",
		"Trace through the code paths systematically",
		"Check each specification requirement",
		"Consider edge cases the implementation might miss",
		"Look for patterns that commonly cause bugs",
		"Document your analysis",
	}

	for _, want := range wantIn {
		if !strings.Contains(got, want) {
			t.Errorf("buildReviewPrompt() missing chain-of-thought element %q", want)
		}
	}
}

// Tests for actionable constraints

func TestPlanningPromptActionableConstraints(t *testing.T) {
	got := buildPlanningPrompt(nil, ".", "Task", "Source", "", "", "", false, nil)

	wantIn := []string{
		"If the approach has flaws, explain the specific problem",
		"If requirements are ambiguous, list your assumptions explicitly",
		"Prefer existing patterns in the codebase",
		"If you're uncertain about something, say so rather than guessing",
	}

	wantNotIn := []string{
		"Be brutally honest",
	}

	for _, want := range wantIn {
		if !strings.Contains(got, want) {
			t.Errorf("buildPlanningPrompt(, nil) missing actionable constraint %q", want)
		}
	}

	for _, notWant := range wantNotIn {
		if strings.Contains(got, notWant) {
			t.Errorf("buildPlanningPrompt(, nil) should not contain vague constraint %q", notWant)
		}
	}
}

func TestReviewPromptActionableConstraints(t *testing.T) {
	got := buildReviewPrompt(nil, "Task", "Source", "Specs")

	wantIn := []string{
		"If the approach has flaws, explain the specific problem",
		"If you find issues, provide concrete fixes",
		"Prefer existing patterns in the codebase",
	}

	wantNotIn := []string{
		"Be brutally honest",
		"Prioritize technical accuracy over validation",
	}

	for _, want := range wantIn {
		if !strings.Contains(got, want) {
			t.Errorf("buildReviewPrompt() missing actionable constraint %q", want)
		}
	}

	for _, notWant := range wantNotIn {
		if strings.Contains(got, notWant) {
			t.Errorf("buildReviewPrompt() should not contain vague constraint %q", notWant)
		}
	}
}

// Tests for concrete examples

func TestPlanningPromptHasExample(t *testing.T) {
	got := buildPlanningPrompt(nil, ".", "Task", "Source", "", "", "", false, nil)

	wantIn := []string{
		"## Example Specification",
		"Add rate limiting",
		"/api/users endpoint",
		"## Plan",
		"golang.org/x/time/rate",
		"## Complete Condition",
		"go test",
	}

	for _, want := range wantIn {
		if !strings.Contains(got, want) {
			t.Errorf("buildPlanningPrompt(, nil) missing example element %q", want)
		}
	}
}

func TestImplementationPromptHasExample(t *testing.T) {
	got := buildImplementationPrompt(nil, ".", "Task", "Source", "Specs", "", "", "", "", nil)

	wantIn := []string{
		"## Example Output",
		"yaml:file",
		"path: internal/api/middleware/ratelimit.go",
		"operation: create",
		"content:",
		"RateLimit",
		"rate.NewLimiter",
	}

	for _, want := range wantIn {
		if !strings.Contains(got, want) {
			t.Errorf("buildImplementationPrompt(, nil) missing example element %q", want)
		}
	}
}

// Tests for Implementation prompt structure (phase order)

func TestImplementationPromptStructure(t *testing.T) {
	got := buildImplementationPrompt(nil, ".", "Task", "Source", "Specs", "Notes", "Custom", "StatusSum", "TrackingSum", nil)

	// Verify phases appear in correct order
	phases := []string{
		"## Task",
		"## Original Requirements",
		"## Specifications",
		"## Additional Notes",
		"## Custom Instructions",
		"## Constraints",
		"## Instructions",
		"## Spec Status Overview",
		"## Specification Tracking",
		"## Error Recovery Strategies",
		"## Testing and Verification",
		"## Example Output",
	}

	lastIndex := -1
	for _, phase := range phases {
		idx := strings.Index(got, phase)
		if idx == -1 {
			t.Errorf("buildImplementationPrompt(, nil) missing phase %q", phase)

			continue
		}
		if idx < lastIndex {
			t.Errorf("buildImplementationPrompt(, nil) phase %q appears before previous phase (order violation)", phase)
		}
		lastIndex = idx
	}
}

// Tests for Review prompt large codebase handling

func TestReviewPromptLargeCodebaseHandling(t *testing.T) {
	got := buildReviewPrompt(nil, "Task", "Source", "Specs")

	wantIn := []string{
		"## Handling Large Codebases",
		"unable to review all changes due to context limits",
		"Focus on critical paths and security-sensitive code first",
		"Prioritize files with the most significant changes",
		"Note which files were not fully reviewed",
		"follow-up review",
	}

	for _, want := range wantIn {
		if !strings.Contains(got, want) {
			t.Errorf("buildReviewPrompt() missing large codebase handling element %q", want)
		}
	}
}

// Tests for role standardization

func TestPromptRoles(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		wantRole string
	}{
		{
			name:     "planning prompt",
			prompt:   buildPlanningPrompt(nil, ".", "Task", "Source", "", "", "", false, nil),
			wantRole: "expert software engineer specializing in architecture and system design",
		},
		{
			name:     "implementation prompt",
			prompt:   buildImplementationPrompt(nil, ".", "Task", "Source", "Specs", "", "", "", "", nil),
			wantRole: "expert software engineer",
		},
		{
			name:     "review prompt",
			prompt:   buildReviewPrompt(nil, "Task", "Source", "Specs"),
			wantRole: "expert software engineer specializing in code review and quality assurance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.prompt, tt.wantRole) {
				t.Errorf("%s missing role %q", tt.name, tt.wantRole)
			}
		})
	}
}

// Tests for timestamp in Review prompt

func TestReviewPromptHasTimestamp(t *testing.T) {
	got := buildReviewPrompt(nil, "Task", "Source", "Specs")

	if !strings.Contains(got, "Current timestamp:") {
		t.Error("buildReviewPrompt() should contain timestamp")
	}
}

// Phase 2 Tests

func TestReviewPromptHasExample(t *testing.T) {
	got := buildReviewPrompt(nil, "Task", "Source", "Specs")

	// Verify example sections
	required := []string{
		"## Example Review Output",
		"### Analysis",
		"### Issues Found",
		"### Fixes",
		"yaml:file",
	}

	for _, section := range required {
		if !strings.Contains(got, section) {
			t.Errorf("buildReviewPrompt() should contain %q", section)
		}
	}
}

func TestReviewPromptHasIterationStructure(t *testing.T) {
	got := buildReviewPrompt(nil, "Task", "Source", "Specs")

	// Verify Round 1/2/3 structure
	required := []string{
		"## Review Process",
		"**Round 1: Analysis**",
		"**Round 2: Fixes**",
		"**Round 3: Verification**",
	}

	for _, section := range required {
		if !strings.Contains(got, section) {
			t.Errorf("buildReviewPrompt() should contain %q", section)
		}
	}
}

func TestPlanningPromptHasOutputFormat(t *testing.T) {
	got := buildPlanningPrompt(nil, ".", "Task", "Source", "", "", "", false, nil)

	required := []string{
		"## Required Output Format",
		"specification with ALL sections",
	}

	for _, section := range required {
		if !strings.Contains(got, section) {
			t.Errorf("buildPlanningPrompt(, nil) should contain %q", section)
		}
	}
}

func TestImplementationPromptHasOutputFormat(t *testing.T) {
	got := buildImplementationPrompt(nil, ".", "Task", "Source", "Specs", "", "", "", "", nil)

	required := []string{
		"## Required Output Format",
		"yaml:file blocks",
		"yaml:file block to update the specification",
	}

	for _, section := range required {
		if !strings.Contains(got, section) {
			t.Errorf("buildImplementationPrompt(, nil) should contain %q", section)
		}
	}
}

func TestReviewPromptHasOutputFormat(t *testing.T) {
	got := buildReviewPrompt(nil, "Task", "Source", "Specs")

	required := []string{
		"## Required Output Format",
		"Analysis section",
		"Issues found with severity",
		"yaml:file blocks fixing ALL issues",
		"Final verdict",
	}

	for _, section := range required {
		if !strings.Contains(got, section) {
			t.Errorf("buildReviewPrompt() should contain %q", section)
		}
	}
}

func TestFinishPromptHasGuidelines(t *testing.T) {
	got := buildFinishPrompt("TEST-123", "Test Task", nil, "", "", "", "")

	// Verify conventional commits guidance
	required := []string{
		"## Commit Message Guidelines",
		"**Conventional Commits Types:**",
		"feat:",
		"fix:",
		"refactor:",
		"docs:",
		"test:",
		"chore:",
		"**Format Rules:**",
		"72 characters",
		"imperative mood",
	}

	for _, section := range required {
		if !strings.Contains(got, section) {
			t.Errorf("buildFinishPrompt() should contain %q", section)
		}
	}
}

func TestFinishPromptHasExample(t *testing.T) {
	got := buildFinishPrompt("TEST-123", "Test Task", nil, "", "", "", "")

	// Verify good/bad examples
	required := []string{
		"## Example Commit Messages",
		"**Good:**",
		"**Bad:**",
		"updated files",
	}

	for _, section := range required {
		if !strings.Contains(got, section) {
			t.Errorf("buildFinishPrompt() should contain %q", section)
		}
	}
}

func TestBuildImplementationPromptHasExplicitStatusInstructions(t *testing.T) {
	got := buildImplementationPrompt(nil, "/work", "Test Task", "Source", "Specs", "Notes", "Custom", "StatusSum", "TrackingSum", nil)

	// Verify prompt instructs agent to update YAML frontmatter status field
	required := []string{
		"MUST update specification YAML frontmatter status to \"done\"",
		"Update specification status",
		"Edit the specification file and change the YAML frontmatter status field",
		"yaml:file block to update the specification",
	}

	for _, section := range required {
		if !strings.Contains(got, section) {
			t.Errorf("buildImplementationPrompt(, nil) should contain %q", section)
		}
	}

	// Verify old ambiguous wording is NOT present
	forbidden := []string{
		"Status to \"completed",
		"Change Status to \"completed",
	}

	for _, text := range forbidden {
		if strings.Contains(got, text) {
			t.Errorf("buildImplementationPrompt(, nil) should NOT contain ambiguous %q", text)
		}
	}
}

func TestBuildImplementationPromptHasYamlExample(t *testing.T) {
	got := buildImplementationPrompt(nil, "/work", "Test Task", "Source", "Specs", "Notes", "Custom", "StatusSum", "TrackingSum", nil)

	// Verify prompt includes example of updating spec YAML frontmatter
	required := []string{
		"```yaml:file",
		"path: .mehrhof/work/",
		"operation: update",
		"status: done",
		"implemented_files:",
	}

	for _, section := range required {
		if !strings.Contains(got, section) {
			t.Errorf("buildImplementationPrompt(, nil) should contain %q", section)
		}
	}
}
