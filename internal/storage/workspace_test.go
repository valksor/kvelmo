package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// openTestWorkspace opens a workspace for testing with a temporary home directory.
// This prevents tests from polluting the real ~/.mehrhof directory.
func openTestWorkspace(tb testing.TB, repoRoot string) *Workspace {
	tb.Helper()
	homeDir := tb.TempDir()
	cfg := NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir
	ws, err := OpenWorkspace(context.Background(), repoRoot, cfg)
	if err != nil {
		tb.Fatalf("OpenWorkspace: %v", err)
	}

	return ws
}

func TestOpenWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	ws := openTestWorkspace(t, tmpDir)

	if ws.Root() != tmpDir {
		t.Errorf("Root() = %q, want %q", ws.Root(), tmpDir)
	}

	// TaskRoot is in project directory
	expectedTaskRoot := filepath.Join(tmpDir, ".mehrhof")
	if ws.TaskRoot() != expectedTaskRoot {
		t.Errorf("TaskRoot() = %q, want %q", ws.TaskRoot(), expectedTaskRoot)
	}

	// WorkRoot is in home directory (not in project directory anymore)
	// With openTestWorkspace, it's in a temporary home directory
	// Just verify it ends with /work
	if !strings.HasSuffix(ws.WorkRoot(), string(filepath.Separator)+"work") {
		t.Errorf("WorkRoot() = %q, want suffix %s", ws.WorkRoot(), string(filepath.Separator)+"work")
	}
}

func TestWorkspaceConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	expected := filepath.Join(tmpDir, ".mehrhof", "config.yaml")
	if ws.ConfigPath() != expected {
		t.Errorf("ConfigPath() = %q, want %q", ws.ConfigPath(), expected)
	}
}

func TestHasConfig(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Initially no config
	if ws.HasConfig() {
		t.Error("HasConfig() = true, want false (no config exists)")
	}

	// Create config directory and file
	if err := os.MkdirAll(ws.TaskRoot(), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", ws.TaskRoot(), err)
	}
	if err := os.WriteFile(ws.ConfigPath(), []byte("test: true"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", ws.ConfigPath(), err)
	}

	if !ws.HasConfig() {
		t.Error("HasConfig() = false, want true (config exists)")
	}
}

func TestNewDefaultWorkspaceConfig(t *testing.T) {
	cfg := NewDefaultWorkspaceConfig()

	if cfg.Git.AutoCommit != true {
		t.Errorf("Git.AutoCommit = %v, want true", cfg.Git.AutoCommit)
	}
	if cfg.Git.CommitPrefix != "[{key}]" {
		t.Errorf("Git.CommitPrefix = %q, want %q", cfg.Git.CommitPrefix, "[{key}]")
	}
	if cfg.Git.BranchPattern != "{type}/{key}--{slug}" {
		t.Errorf("Git.BranchPattern = %q, want %q", cfg.Git.BranchPattern, "{type}/{key}--{slug}")
	}
	if cfg.Agent.Default != "claude" {
		t.Errorf("Agent.Default = %q, want %q", cfg.Agent.Default, "claude")
	}
	if cfg.Agent.Timeout != 300 {
		t.Errorf("Agent.Timeout = %d, want 300", cfg.Agent.Timeout)
	}
	if cfg.Agent.MaxRetries != 3 {
		t.Errorf("Agent.MaxRetries = %d, want 3", cfg.Agent.MaxRetries)
	}
	if cfg.Workflow.AutoInit != true {
		t.Errorf("Workflow.AutoInit = %v, want true", cfg.Workflow.AutoInit)
	}
	if cfg.Workflow.SessionRetentionDays != 30 {
		t.Errorf("Workflow.SessionRetentionDays = %d, want 30", cfg.Workflow.SessionRetentionDays)
	}
	if cfg.Env == nil {
		t.Error("Env is nil, want initialized map")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	cfg := NewDefaultWorkspaceConfig()
	cfg.Git.CommitPrefix = "[custom]"
	cfg.Agent.Timeout = 600

	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.Git.CommitPrefix != "[custom]" {
		t.Errorf("loaded Git.CommitPrefix = %q, want %q", loaded.Git.CommitPrefix, "[custom]")
	}
	if loaded.Agent.Timeout != 600 {
		t.Errorf("loaded Agent.Timeout = %d, want 600", loaded.Agent.Timeout)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// No config file exists, should return defaults
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Agent.Default != "claude" {
		t.Errorf("default Agent.Default = %q, want %q", cfg.Agent.Default, "claude")
	}
}

func TestEnsureInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized failed: %v", err)
	}

	// Check work directory exists
	if _, err := os.Stat(ws.WorkRoot()); os.IsNotExist(err) {
		t.Error("work directory was not created")
	}

	// Check .mehrhof directory exists
	if _, err := os.Stat(ws.TaskRoot()); os.IsNotExist(err) {
		t.Error(".mehrhof directory was not created")
	}

	// Note: EnsureInitialized does NOT update .gitignore
	// .gitignore is only updated by the init command via UpdateGitignore()
}

func TestUpdateGitignore(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Test with no existing .gitignore
	if err := ws.UpdateGitignore(); err != nil {
		t.Fatalf("UpdateGitignore failed: %v", err)
	}

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	content := string(data)
	// Only .mehrhof/.env should be added (work and active_task are in home dir now)
	if !contains(content, ".mehrhof/.env") {
		t.Error(".gitignore does not contain .mehrhof/.env")
	}
}

func TestUpdateGitignoreExisting(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Create existing .gitignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", gitignorePath, err)
	}

	if err := ws.UpdateGitignore(); err != nil {
		t.Fatalf("UpdateGitignore failed: %v", err)
	}

	data, _ := os.ReadFile(gitignorePath)
	content := string(data)

	if !contains(content, "node_modules/") {
		t.Error("existing .gitignore content was lost")
	}
	if !contains(content, ".mehrhof/.env") {
		t.Error(".gitignore does not contain .mehrhof/.env")
	}
}

func TestUpdateGitignoreIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Call UpdateGitignore twice - second call should not duplicate entries
	if err := ws.UpdateGitignore(); err != nil {
		t.Fatalf("UpdateGitignore (1st) failed: %v", err)
	}

	if err := ws.UpdateGitignore(); err != nil {
		t.Fatalf("UpdateGitignore (2nd) failed: %v", err)
	}

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	data, _ := os.ReadFile(gitignorePath)
	lines := strings.Split(string(data), "\n")

	// Count exact line matches for ".mehrhof/.env" (the only entry now)
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == ".mehrhof/.env" {
			count++
		}
	}
	if count != 1 {
		t.Errorf(".mehrhof/.env appears %d times as exact line in .gitignore, want 1", count)
	}
}

func TestActiveTaskPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// ActiveTaskPath is in the workspace data directory (home)
	path := ws.ActiveTaskPath()
	if !strings.HasSuffix(path, ".active_task") {
		t.Errorf("ActiveTaskPath() = %q, want suffix .active_task", path)
	}
}

func TestHasActiveTask(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	if ws.HasActiveTask() {
		t.Error("HasActiveTask() = true, want false (no active task)")
	}

	// Create active task file (need to create parent dir first)
	if err := os.MkdirAll(filepath.Dir(ws.ActiveTaskPath()), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(ws.ActiveTaskPath(), []byte("id: test"), 0o644); err != nil {
		t.Fatalf("WriteFile active task: %v", err)
	}

	if !ws.HasActiveTask() {
		t.Error("HasActiveTask() = false, want true")
	}
}

func TestSaveAndLoadActiveTask(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	active := &ActiveTask{
		ID:      "test123",
		Ref:     "file:task.md",
		WorkDir: ".mehrhof/work/test123",
		State:   "planning",
		Branch:  "task/test123",
		UseGit:  true,
		Started: time.Now(),
	}

	if err := ws.SaveActiveTask(active); err != nil {
		t.Fatalf("SaveActiveTask failed: %v", err)
	}

	loaded, err := ws.LoadActiveTask()
	if err != nil {
		t.Fatalf("LoadActiveTask failed: %v", err)
	}

	if loaded.ID != active.ID {
		t.Errorf("loaded ID = %q, want %q", loaded.ID, active.ID)
	}
	if loaded.Ref != active.Ref {
		t.Errorf("loaded Ref = %q, want %q", loaded.Ref, active.Ref)
	}
	if loaded.State != active.State {
		t.Errorf("loaded State = %q, want %q", loaded.State, active.State)
	}
	if loaded.UseGit != active.UseGit {
		t.Errorf("loaded UseGit = %v, want %v", loaded.UseGit, active.UseGit)
	}
}

func TestClearActiveTask(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create active task file (need to create parent dir first)
	if err := os.MkdirAll(filepath.Dir(ws.ActiveTaskPath()), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(ws.ActiveTaskPath(), []byte("id: test"), 0o644); err != nil {
		t.Fatalf("WriteFile active task: %v", err)
	}

	if err := ws.ClearActiveTask(); err != nil {
		t.Fatalf("ClearActiveTask failed: %v", err)
	}

	if ws.HasActiveTask() {
		t.Error("active task still exists after clear")
	}

	// Clear non-existent should not error
	if err := ws.ClearActiveTask(); err != nil {
		t.Errorf("ClearActiveTask on non-existent failed: %v", err)
	}
}

func TestUpdateActiveTaskState(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	active := &ActiveTask{
		ID:    "test123",
		State: "idle",
	}
	if err := ws.SaveActiveTask(active); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	if err := ws.UpdateActiveTaskState("planning"); err != nil {
		t.Fatalf("UpdateActiveTaskState failed: %v", err)
	}

	loaded, _ := ws.LoadActiveTask()
	if loaded.State != "planning" {
		t.Errorf("State = %q, want %q", loaded.State, "planning")
	}
}

func TestWorkPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// WorkPath is in the workspace data directory (home)
	path := ws.WorkPath("abc123")
	if !strings.HasSuffix(path, "/abc123") {
		t.Errorf("WorkPath() = %q, want suffix /abc123", path)
	}
}

func TestWorkExists(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	if ws.WorkExists("nonexistent") {
		t.Error("WorkExists() = true for non-existent work")
	}

	// Create work directory
	if err := os.MkdirAll(ws.WorkPath("test123"), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", ws.WorkPath("test123"), err)
	}

	if !ws.WorkExists("test123") {
		t.Error("WorkExists() = false for existing work")
	}
}

func TestGenerateTaskID(t *testing.T) {
	id1 := GenerateTaskID()
	id2 := GenerateTaskID()

	if id1 == "" {
		t.Error("GenerateTaskID returned empty string")
	}
	if len(id1) != 8 {
		t.Errorf("GenerateTaskID length = %d, want 8", len(id1))
	}
	if id1 == id2 {
		t.Error("GenerateTaskID returned duplicate IDs")
	}
}

func TestCreateWork(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: "Test task content",
		ReadAt:  time.Now(),
	}

	work, err := ws.CreateWork("test123", source)
	if err != nil {
		t.Fatalf("CreateWork failed: %v", err)
	}

	if work.Metadata.ID != "test123" {
		t.Errorf("work ID = %q, want %q", work.Metadata.ID, "test123")
	}

	// Check directories were created
	if !ws.WorkExists("test123") {
		t.Error("work directory was not created")
	}

	specsDir := ws.SpecificationsDir("test123")
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		t.Error("specifications directory was not created")
	}

	sessionsDir := ws.SessionsDir("test123")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		t.Error("sessions directory was not created")
	}

	// Check notes.md was created
	notesPath := ws.NotesPath("test123")
	if _, err := os.Stat(notesPath); os.IsNotExist(err) {
		t.Error("notes.md was not created")
	}
}

func TestLoadAndSaveWork(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: "Test content",
	}

	work, _ := ws.CreateWork("test123", source)
	work.Metadata.Title = "Test Task"

	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork failed: %v", err)
	}

	loaded, err := ws.LoadWork("test123")
	if err != nil {
		t.Fatalf("LoadWork failed: %v", err)
	}

	if loaded.Metadata.Title != "Test Task" {
		t.Errorf("loaded Title = %q, want %q", loaded.Metadata.Title, "Test Task")
	}
	if loaded.Source.Content != "Test content" {
		t.Errorf("loaded Source.Content = %q, want %q", loaded.Source.Content, "Test content")
	}
}

func TestDeleteWork(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	if err := ws.DeleteWork("test123"); err != nil {
		t.Fatalf("DeleteWork failed: %v", err)
	}

	if ws.WorkExists("test123") {
		t.Error("work still exists after delete")
	}
}

func TestListWorks(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Initially empty
	works, err := ws.ListWorks()
	if err != nil {
		t.Fatalf("ListWorks failed: %v", err)
	}
	if len(works) != 0 {
		t.Errorf("ListWorks returned %d works, want 0", len(works))
	}

	// Create some works
	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("task1", source); err != nil {
		t.Fatalf("CreateWork(task1): %v", err)
	}
	if _, err := ws.CreateWork("task2", source); err != nil {
		t.Fatalf("CreateWork(task2): %v", err)
	}

	works, err = ws.ListWorks()
	if err != nil {
		t.Fatalf("ListWorks failed: %v", err)
	}
	if len(works) != 2 {
		t.Errorf("ListWorks returned %d works, want 2", len(works))
	}
}

// TestCreateWork_SaveInProject verifies that CreateWork respects save_in_project config.
func TestCreateWork_SaveInProject(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Configure to save in project
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	cfg.Storage.SaveInProject = true
	cfg.Storage.ProjectDir = "tickets"
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("TASK-123", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Verify work directory created in project, not global storage
	projectWorkDir := filepath.Join(tmpDir, "tickets", "TASK-123")
	if _, err := os.Stat(projectWorkDir); err != nil {
		t.Errorf("Work directory not created in project: %s (error: %v)", projectWorkDir, err)
	}

	// Verify work.yaml exists in project location
	workYaml := filepath.Join(projectWorkDir, "work.yaml")
	if _, err := os.Stat(workYaml); err != nil {
		t.Errorf("work.yaml not found in project: %s (error: %v)", workYaml, err)
	}

	// Verify notes.md exists in project location
	notesMd := filepath.Join(projectWorkDir, "notes.md")
	if _, err := os.Stat(notesMd); err != nil {
		t.Errorf("notes.md not found in project: %s (error: %v)", notesMd, err)
	}
}

// TestLoadSaveWork_SaveInProject verifies LoadWork and SaveWork respect save_in_project config.
func TestLoadSaveWork_SaveInProject(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Configure to save in project
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	cfg.Storage.SaveInProject = true
	cfg.Storage.ProjectDir = "tickets"
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Create and modify work
	source := SourceInfo{Type: "file", Ref: "task.md"}
	work, err := ws.CreateWork("TASK-456", source)
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}
	work.Metadata.Title = "Updated Title"
	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	// Load and verify
	loaded, err := ws.LoadWork("TASK-456")
	if err != nil {
		t.Fatalf("LoadWork: %v", err)
	}
	if loaded.Metadata.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", loaded.Metadata.Title, "Updated Title")
	}
}

// TestListWorks_SaveInProject verifies ListWorks returns tasks from project location.
func TestListWorks_SaveInProject(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Configure to save in project
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	cfg.Storage.SaveInProject = true
	cfg.Storage.ProjectDir = "tickets"
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Create tasks
	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("TASK-1", source); err != nil {
		t.Fatalf("CreateWork(TASK-1): %v", err)
	}
	if _, err := ws.CreateWork("TASK-2", source); err != nil {
		t.Fatalf("CreateWork(TASK-2): %v", err)
	}

	// List and verify
	works, err := ws.ListWorks()
	if err != nil {
		t.Fatalf("ListWorks: %v", err)
	}
	if len(works) != 2 {
		t.Errorf("ListWorks returned %d tasks, want 2", len(works))
	}
}

// TestDeleteWork_SaveInProject verifies DeleteWork removes from project location.
func TestDeleteWork_SaveInProject(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Configure to save in project
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	cfg.Storage.SaveInProject = true
	cfg.Storage.ProjectDir = "tickets"
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Create and delete work
	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("TASK-789", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	if err := ws.DeleteWork("TASK-789"); err != nil {
		t.Fatalf("DeleteWork: %v", err)
	}

	// Verify deleted from project location
	projectWorkDir := filepath.Join(tmpDir, "tickets", "TASK-789")
	if _, err := os.Stat(projectWorkDir); !os.IsNotExist(err) {
		t.Errorf("Work directory still exists: %s", projectWorkDir)
	}

	// Verify WorkExists returns false
	if ws.WorkExists("TASK-789") {
		t.Error("WorkExists returned true after delete")
	}
}

// TestWorkExists_SaveInProject verifies WorkExists checks project location.
func TestWorkExists_SaveInProject(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Configure to save in project
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	cfg.Storage.SaveInProject = true
	cfg.Storage.ProjectDir = "tickets"
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Initially doesn't exist
	if ws.WorkExists("TASK-ABC") {
		t.Error("WorkExists returned true before create")
	}

	// Create work
	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("TASK-ABC", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Now exists
	if !ws.WorkExists("TASK-ABC") {
		t.Error("WorkExists returned false after create")
	}
}

func TestNotesPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// NotesPath is in the workspace data directory (home)
	path := ws.NotesPath("test123")
	if !strings.HasSuffix(path, "/test123/notes.md") {
		t.Errorf("NotesPath() = %q, want suffix /test123/notes.md", path)
	}
}

func TestAppendAndReadNotes(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	if err := ws.AppendNote("test123", "First note", "planning"); err != nil {
		t.Fatalf("AppendNote failed: %v", err)
	}

	if err := ws.AppendNote("test123", "Second note", "implementing"); err != nil {
		t.Fatalf("AppendNote failed: %v", err)
	}

	content, err := ws.ReadNotes("test123")
	if err != nil {
		t.Fatalf("ReadNotes failed: %v", err)
	}

	if !contains(content, "First note") {
		t.Error("notes do not contain 'First note'")
	}
	if !contains(content, "Second note") {
		t.Error("notes do not contain 'Second note'")
	}
	if !contains(content, "[planning]") {
		t.Error("notes do not contain state tag '[planning]'")
	}
}

func TestSpecsDir(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// SpecsDir is in the workspace data directory (home)
	path := ws.SpecificationsDir("test123")
	if !strings.HasSuffix(path, "/test123/specifications") {
		t.Errorf("SpecsDir() = %q, want suffix /test123/specifications", path)
	}
}

func TestSpecPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	cfg := NewDefaultWorkspaceConfig()

	// SpecPath is in the workspace data directory (home)
	path := ws.SpecificationPath("test123", 1, cfg)
	if !strings.HasSuffix(path, "/test123/specifications/specification-1.md") {
		t.Errorf("SpecPath() = %q, want suffix /test123/specifications/specification-1.md", path)
	}

	// Custom pattern test
	cfg.Specification.FilenamePattern = "SPEC-{n}.md"
	path = ws.SpecificationPath("test123", 1, cfg)
	if !strings.HasSuffix(path, "/test123/specifications/SPEC-1.md") {
		t.Errorf("SpecPath(SPEC) = %q, want suffix /test123/specifications/SPEC-1.md", path)
	}
}

func TestSaveAndLoadSpec(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	content := "# Spec 1\n\nThis is the first spec."
	if err := ws.SaveSpecification("test123", 1, content); err != nil {
		t.Fatalf("SaveSpec failed: %v", err)
	}

	loaded, err := ws.LoadSpecification("test123", 1)
	if err != nil {
		t.Fatalf("LoadSpec failed: %v", err)
	}

	if loaded != content {
		t.Errorf("loaded spec = %q, want %q", loaded, content)
	}
}

func TestListSpecs(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}
	// Initially empty
	specifications, err := ws.ListSpecifications("test123")
	if err != nil {
		t.Fatalf("ListSpecifications failed: %v", err)
	}
	if len(specifications) != 0 {
		t.Errorf("ListSpecifications returned %d specifications, want 0", len(specifications))
	}

	// Add some specifications
	if err := ws.SaveSpecification("test123", 1, "Specification 1"); err != nil {
		t.Fatalf("SaveSpecification(1): %v", err)
	}
	if err := ws.SaveSpecification("test123", 3, "Specification 3"); err != nil {
		t.Fatalf("SaveSpecification(3): %v", err)
	}
	if err := ws.SaveSpecification("test123", 2, "Specification 2"); err != nil {
		t.Fatalf("SaveSpecification(2): %v", err)
	}

	specifications, err = ws.ListSpecifications("test123")
	if err != nil {
		t.Fatalf("ListSpecifications failed: %v", err)
	}
	if len(specifications) != 3 {
		t.Errorf("ListSpecifications returned %d specifications, want 3", len(specifications))
	}

	// Should be sorted
	if specifications[0] != 1 || specifications[1] != 2 || specifications[2] != 3 {
		t.Errorf("specifications not sorted: %v", specifications)
	}
}

func TestNextSpecNumber(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	// First spec should be 1
	num, err := ws.NextSpecificationNumber("test123")
	if err != nil {
		t.Fatalf("NextSpecNumber failed: %v", err)
	}
	if num != 1 {
		t.Errorf("NextSpecNumber = %d, want 1", num)
	}

	// After adding spec 1, next should be 2
	if err := ws.SaveSpecification("test123", 1, "Spec 1"); err != nil {
		t.Fatalf("SaveSpec(1): %v", err)
	}
	num, _ = ws.NextSpecificationNumber("test123")
	if num != 2 {
		t.Errorf("NextSpecNumber = %d, want 2", num)
	}

	// After adding spec 5, next should be 6
	if err := ws.SaveSpecification("test123", 5, "Spec 5"); err != nil {
		t.Fatalf("SaveSpec(5): %v", err)
	}
	num, _ = ws.NextSpecificationNumber("test123")
	if num != 6 {
		t.Errorf("NextSpecNumber = %d, want 6", num)
	}
}

func TestProjectSpecsDir(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	cfg := NewDefaultWorkspaceConfig()

	// Default: ProjectSpecificationsDir is in the project .mehrhof directory (no /specifications/ subdir)
	path := ws.ProjectSpecificationsDir("test123", cfg)
	if !strings.HasSuffix(path, ".mehrhof/work/test123") {
		t.Errorf("ProjectSpecificationsDir() = %q, want suffix .mehrhof/work/test123", path)
	}

	// Custom ProjectDir: should use that directory
	cfg.Storage.ProjectDir = "tickets"
	path = ws.ProjectSpecificationsDir("test123", cfg)
	if !strings.HasSuffix(path, "tickets/test123") {
		t.Errorf("ProjectSpecificationsDir(tickets) = %q, want suffix tickets/test123", path)
	}
}

func TestProjectSpecPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	cfg := NewDefaultWorkspaceConfig()

	// Default pattern: specification-{n}.md
	path := ws.ProjectSpecificationPath("test123", 1, cfg)
	if !strings.HasSuffix(path, ".mehrhof/work/test123/specification-1.md") {
		t.Errorf("ProjectSpecificationPath() = %q, want suffix .mehrhof/work/test123/specification-1.md", path)
	}

	// Custom pattern: SPEC-{n}.md
	cfg.Specification.FilenamePattern = "SPEC-{n}.md"
	cfg.Storage.ProjectDir = "tickets"
	path = ws.ProjectSpecificationPath("test123", 1, cfg)
	if !strings.HasSuffix(path, "tickets/test123/SPEC-1.md") {
		t.Errorf("ProjectSpecificationPath(SPEC) = %q, want suffix tickets/test123/SPEC-1.md", path)
	}
}

func TestSaveSpecificationInProject(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Enable project-local saving (mutually exclusive - saves to project ONLY)
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	cfg.Storage.SaveInProject = true
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	content := "# Spec 1\n\nThis is the first spec."
	if err := ws.SaveSpecification("test123", 1, content); err != nil {
		t.Fatalf("SaveSpecification failed: %v", err)
	}

	// Reload config after save
	cfg, err = ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig after save: %v", err)
	}

	// Verify project-local storage exists (mutually exclusive: project ONLY)
	projectPath := ws.ProjectSpecificationPath("test123", 1, cfg)
	if _, err := os.Stat(projectPath); err != nil {
		t.Errorf("Project-local spec not found: %s, error: %v", projectPath, err)
	}

	// Verify internal storage does NOT exist (mutually exclusive)
	internalPath := ws.SpecificationPath("test123", 1, cfg)
	if _, err := os.Stat(internalPath); err == nil {
		t.Errorf("Internal spec should NOT exist when save_in_project=true: %s", internalPath)
	}

	// Verify content matches
	loaded, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("ReadFile(projectPath) failed: %v", err)
	}
	if string(loaded) != content {
		t.Errorf("Project spec content mismatch, got %q, want %q", string(loaded), content)
	}

	// Verify LoadSpecification returns correct content
	loadedViaAPI, err := ws.LoadSpecification("test123", 1)
	if err != nil {
		t.Fatalf("LoadSpecification failed: %v", err)
	}
	if loadedViaAPI != content {
		t.Errorf("LoadSpecification content mismatch, got %q, want %q", loadedViaAPI, content)
	}
}

func TestSaveSpecificationNotInProject(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Ensure project-local saving is disabled (default)
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Storage.SaveInProject {
		t.Skip("SaveInProject should be false by default")
	}
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	content := "# Spec 1\n\nThis is the first spec."
	if err := ws.SaveSpecification("test123", 1, content); err != nil {
		t.Fatalf("SaveSpecification failed: %v", err)
	}

	// Verify internal storage exists
	internalPath := ws.SpecificationPath("test123", 1, cfg)
	if _, err := os.Stat(internalPath); err != nil {
		t.Errorf("Internal spec not found: %s, error: %v", internalPath, err)
	}

	// Verify project-local storage does NOT exist
	projectPath := ws.ProjectSpecificationPath("test123", 1, cfg)
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		t.Errorf("Project spec should not exist when SaveInProject=false, path: %s", projectPath)
	}
}

func TestSaveSpecificationConfigLoadFailure(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Enable project-local saving
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	cfg.Storage.SaveInProject = true
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	// Corrupt the config file to simulate a config load failure
	configPath := ws.ConfigPath()
	if err := os.WriteFile(configPath, []byte("invalid: yaml: content: [unclosed"), 0o644); err != nil {
		t.Fatalf("Failed to corrupt config: %v", err)
	}

	// SaveSpecification should still succeed (internal spec is saved)
	// but should log a warning and use defaults
	content := "# Spec 1\n\nThis is the first spec."
	if err := ws.SaveSpecification("test123", 1, content); err != nil {
		t.Errorf("SaveSpecification should succeed even with config load failure, got: %v", err)
	}

	// Use default config for path verification (config was corrupted)
	defaultCfg := NewDefaultWorkspaceConfig()

	// Verify internal storage exists
	internalPath := ws.SpecificationPath("test123", 1, defaultCfg)
	if _, err := os.Stat(internalPath); err != nil {
		t.Errorf("Internal spec should exist even with config load failure: %v", err)
	}

	// Verify project-local storage does NOT exist (defaults don't enable project save)
	projectPath := ws.ProjectSpecificationPath("test123", 1, defaultCfg)
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		t.Errorf("Project spec should not exist when using defaults, path: %s", projectPath)
	}
}

func TestSaveSpecificationProjectWriteFails(t *testing.T) {
	// Skip test when running as root - root can write to read-only directories
	// which would make this test pass for the wrong reason.
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Enable project-local saving (mutually exclusive - project is ONLY storage)
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	cfg.Storage.SaveInProject = true
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	// Make the project directory read-only to simulate a write failure
	// (default project_dir is .mehrhof/work when save_in_project=true)
	projectDir := filepath.Join(ws.Root(), ".mehrhof", "work", "test123")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	// Make directory read-only
	if err := os.Chmod(projectDir, 0o444); err != nil {
		t.Fatalf("Failed to chmod directory: %v", err)
	}
	// Ensure we can restore permissions even if test fails
	defer func() {
		_ = os.Chmod(projectDir, 0o755)
	}()

	// With mutually exclusive storage, when save_in_project=true and project write fails,
	// the operation should fail (project is the ONLY storage location)
	content := "# Spec 1\n\nThis is the first spec."
	if err := ws.SaveSpecification("test123", 1, content); err == nil {
		t.Errorf("SaveSpecification should fail when project write fails and save_in_project=true")
	}
}

func TestInvalidTaskIDRejected(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}

	invalidTaskIDs := []struct {
		name    string
		taskID  string
		wantErr bool
	}{
		{"empty string", "", true},
		{"path traversal", "../etc/passwd", true},
		{"backslash traversal", "..\\windows\\system32", true},
		{"absolute path unix", "/etc/passwd", true},
		{"absolute path windows", "C:\\Windows\\System32", true},
		{"mixed traversal", "foo/../bar", true},
		{"special characters", "foo@bar#baz", true},
		{"spaces", "foo bar", true},
		{"null character", "foo\x00bar", true},
		{"valid - alphanumeric", "abc123", false},
		{"valid - with hyphen", "abc-123", false},
		{"valid - with underscore", "abc_123", false},
		{"valid - mixed", "ABC-123_def", false},
	}

	for _, tt := range invalidTaskIDs {
		t.Run(tt.name, func(t *testing.T) {
			// Create work directory for this test case so valid IDs can be saved
			if !tt.wantErr {
				if _, err := ws.CreateWork(tt.taskID, source); err != nil {
					t.Fatalf("CreateWork(%q): %v", tt.taskID, err)
				}
			}

			err := ws.SaveSpecification(tt.taskID, 1, "content")
			if (err != nil) != tt.wantErr {
				// For valid IDs, check if the error is specifically about validation
				// (not directory not found)
				if tt.wantErr {
					t.Errorf("SaveSpecification(%q) error = %v, wantErr %v", tt.taskID, err, tt.wantErr)
				} else {
					// For valid IDs, we expect no validation error
					// Directory not found is ok for this test - we're testing validation
					if err != nil && !strings.Contains(err.Error(), "invalid task ID") {
						t.Errorf("SaveSpecification(%q) should not fail validation, got: %v", tt.taskID, err)
					}
				}
			}
		})
	}
}

func TestGatherSpecsContent(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	if err := ws.SaveSpecification("test123", 1, "Content of spec 1"); err != nil {
		t.Fatalf("SaveSpec(1): %v", err)
	}
	if err := ws.SaveSpecification("test123", 2, "Content of spec 2"); err != nil {
		t.Fatalf("SaveSpec(2): %v", err)
	}

	content, err := ws.GatherSpecificationsContent("test123")
	if err != nil {
		t.Fatalf("GatherSpecsContent failed: %v", err)
	}

	if !contains(content, "Specification 1") {
		t.Error("gathered content does not contain Specification 1")
	}
	if !contains(content, "Content of spec 1") {
		t.Error("gathered content does not contain spec 1 content")
	}
	if !contains(content, "Specification 2") {
		t.Error("gathered content does not contain Specification 2")
	}
}

func TestGetLatestSpecContent(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	// No specifications
	content, num, err := ws.GetLatestSpecificationContent("test123")
	if err != nil {
		t.Fatalf("GetLatestSpecificationContent failed: %v", err)
	}
	if content != "" || num != 0 {
		t.Errorf("expected empty content and 0, got %q and %d", content, num)
	}

	// Add specifications
	if err := ws.SaveSpecification("test123", 1, "First specification"); err != nil {
		t.Fatalf("SaveSpecification(1): %v", err)
	}
	if err := ws.SaveSpecification("test123", 3, "Third specification"); err != nil {
		t.Fatalf("SaveSpecification(3): %v", err)
	}

	content, num, err = ws.GetLatestSpecificationContent("test123")
	if err != nil {
		t.Fatalf("GetLatestSpecificationContent failed: %v", err)
	}
	if num != 3 {
		t.Errorf("latest specification number = %d, want 3", num)
	}
	if content != "Third specification" {
		t.Errorf("latest specification content = %q, want %q", content, "Third specification")
	}
}

func TestSessionsDir(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// SessionsDir is in the workspace data directory (home)
	path := ws.SessionsDir("test123")
	if !strings.HasSuffix(path, "/test123/sessions") {
		t.Errorf("SessionsDir() = %q, want suffix /test123/sessions", path)
	}
}

func TestSessionPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// SessionPath is in the workspace data directory (home)
	path := ws.SessionPath("test123", "session.yaml")
	if !strings.HasSuffix(path, "/test123/sessions/session.yaml") {
		t.Errorf("SessionPath() = %q, want suffix /test123/sessions/session.yaml", path)
	}
}

func TestCreateSession(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	session, filename, err := ws.CreateSession("test123", "planning", "claude", "planning")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session.Metadata.Type != "planning" {
		t.Errorf("session type = %q, want %q", session.Metadata.Type, "planning")
	}
	if session.Metadata.Agent != "claude" {
		t.Errorf("session agent = %q, want %q", session.Metadata.Agent, "claude")
	}
	if filename == "" {
		t.Error("filename is empty")
	}
	if !contains(filename, "planning") {
		t.Error("filename does not contain session type")
	}
}

func TestLoadAndSaveSession(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	session, filename, _ := ws.CreateSession("test123", "planning", "claude", "planning")

	// Add an exchange
	session.Exchanges = append(session.Exchanges, Exchange{
		Role:      "user",
		Timestamp: time.Now(),
		Content:   "Test message",
	})

	if err := ws.SaveSession("test123", filename, session); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	loaded, err := ws.LoadSession("test123", filename)
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}

	if len(loaded.Exchanges) != 1 {
		t.Errorf("loaded exchanges count = %d, want 1", len(loaded.Exchanges))
	}
	if loaded.Exchanges[0].Content != "Test message" {
		t.Errorf("loaded exchange content = %q, want %q", loaded.Exchanges[0].Content, "Test message")
	}
}

func TestListSessionFiles(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Create multiple sessions
	if _, _, err := ws.CreateSession("test123", "planning", "claude", "planning"); err != nil {
		t.Fatalf("CreateSession planning: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	if _, _, err := ws.CreateSession("test123", "implementing", "claude", "implementing"); err != nil {
		t.Fatalf("CreateSession implementing: %v", err)
	}

	files, err := ws.ListSessionFiles("test123")
	if err != nil {
		t.Fatalf("ListSessionFiles failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("ListSessionFiles count = %d, want 2", len(files))
	}
}

func TestGetLatestSessionFile(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// No sessions yet
	latest := ws.GetLatestSessionFile("test123")
	if latest != "" {
		t.Errorf("GetLatestSessionFile expected empty, got %q", latest)
	}

	// Create a single session and verify we get it back
	_, file1, _ := ws.CreateSession("test123", "planning", "claude", "planning")

	latest = ws.GetLatestSessionFile("test123")
	if latest != file1 {
		t.Errorf("GetLatestSessionFile = %q, want %q", latest, file1)
	}

	// Create another session - GetLatestSessionFile returns last in sorted order
	// (which may be file1 or file2 if created in same second)
	_, file2, _ := ws.CreateSession("test123", "implementing", "claude", "implementing")

	latest = ws.GetLatestSessionFile("test123")
	// Should return one of the files (last alphabetically)
	if latest != file1 && latest != file2 {
		t.Errorf("GetLatestSessionFile = %q, want one of [%q, %q]", latest, file1, file2)
	}
}

func TestTranscriptsDir(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// TranscriptsDir should be within the workspace work directory
	got := ws.TranscriptsDir("test123")
	if !strings.HasSuffix(got, "/test123/transcripts") {
		t.Errorf("TranscriptsDir = %q, want suffix /test123/transcripts", got)
	}
}

func TestTranscriptPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// TranscriptPath should be within the workspace work directory
	got := ws.TranscriptPath("test123", "2024-01-15T10-30-00-planning.log")
	if !strings.HasSuffix(got, "/test123/transcripts/2024-01-15T10-30-00-planning.log") {
		t.Errorf("TranscriptPath = %q, want suffix /test123/transcripts/2024-01-15T10-30-00-planning.log", got)
	}
}

func TestSaveAndLoadTranscript(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Save transcript
	content := "This is the full agent output\nWith multiple lines\nAnd content"
	err := ws.SaveTranscript("test123", "2024-01-15T10-30-00-planning.log", content)
	if err != nil {
		t.Fatalf("SaveTranscript failed: %v", err)
	}

	// Verify file exists
	transcriptPath := ws.TranscriptPath("test123", "2024-01-15T10-30-00-planning.log")
	if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
		t.Fatal("transcript file not created")
	}

	// Load transcript
	loaded, err := ws.LoadTranscript("test123", "2024-01-15T10-30-00-planning.log")
	if err != nil {
		t.Fatalf("LoadTranscript failed: %v", err)
	}
	if loaded != content {
		t.Errorf("LoadTranscript content = %q, want %q", loaded, content)
	}
}

func TestListTranscripts(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// No transcripts yet
	files, err := ws.ListTranscripts("test123")
	if err != nil {
		t.Fatalf("ListTranscripts failed: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("ListTranscripts count = %d, want 0", len(files))
	}

	// Save some transcripts
	_ = ws.SaveTranscript("test123", "2024-01-15T10-30-00-planning.log", "content1")
	_ = ws.SaveTranscript("test123", "2024-01-15T10-31-00-answer.log", "content2")
	_ = ws.SaveTranscript("test123", "2024-01-15T10-32-00-implementation.log", "content3")

	files, err = ws.ListTranscripts("test123")
	if err != nil {
		t.Fatalf("ListTranscripts failed: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("ListTranscripts count = %d, want 3", len(files))
	}

	// Verify files are sorted (timestamp order)
	if files[0] != "2024-01-15T10-30-00-planning.log" {
		t.Errorf("first file = %q, want planning.log", files[0])
	}
	if files[1] != "2024-01-15T10-31-00-answer.log" {
		t.Errorf("second file = %q, want answer.log", files[1])
	}
}

func TestGetSourceContent(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Test single file source (embedded content for backwards compat)
	source := SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: "Single file content",
	}
	if _, err := ws.CreateWork("test1", source); err != nil {
		t.Fatalf("CreateWork(test1): %v", err)
	}

	content, err := ws.GetSourceContent("test1")
	if err != nil {
		t.Fatalf("GetSourceContent failed: %v", err)
	}
	if content != "Single file content" {
		t.Errorf("content = %q, want %q", content, "Single file content")
	}

	// Test directory source with actual files (new hybrid storage)
	source2 := SourceInfo{
		Type:  "directory",
		Ref:   "tasks/",
		Files: []string{"source/file1.md", "source/file2.md"},
	}
	if _, err := ws.CreateWork("test2", source2); err != nil {
		t.Fatalf("CreateWork(test2): %v", err)
	}

	// Write actual source files
	workPath := ws.WorkPath("test2")
	if err := os.WriteFile(filepath.Join(workPath, "source", "file1.md"), []byte("Content 1"), 0o644); err != nil {
		t.Fatalf("write file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workPath, "source", "file2.md"), []byte("Content 2"), 0o644); err != nil {
		t.Fatalf("write file2: %v", err)
	}

	content, err = ws.GetSourceContent("test2")
	if err != nil {
		t.Fatalf("GetSourceContent failed: %v", err)
	}
	if !contains(content, "file1.md") {
		t.Error("content does not contain file1.md")
	}
	if !contains(content, "Content 1") {
		t.Error("content does not contain Content 1")
	}
	if !contains(content, "file2.md") {
		t.Error("content does not contain file2.md")
	}
}

func TestPendingQuestionPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// PendingQuestionPath should be within the workspace work directory
	got := ws.PendingQuestionPath("test123")
	if !strings.HasSuffix(got, "/test123/pending_question.yaml") {
		t.Errorf("PendingQuestionPath() = %q, want suffix /test123/pending_question.yaml", got)
	}
}

func TestHasPendingQuestion(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	if ws.HasPendingQuestion("test123") {
		t.Error("HasPendingQuestion() = true, want false")
	}

	// Create pending question
	q := &PendingQuestion{
		Question: "Test question?",
		Phase:    "planning",
		AskedAt:  time.Now(),
	}
	if err := ws.SavePendingQuestion("test123", q); err != nil {
		t.Fatalf("SavePendingQuestion: %v", err)
	}

	if !ws.HasPendingQuestion("test123") {
		t.Error("HasPendingQuestion() = false, want true")
	}
}

func TestSaveAndLoadPendingQuestion(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	q := &PendingQuestion{
		Question: "What should we do?",
		Options: []QuestionOption{
			{Label: "A", Description: "Option A"},
			{Label: "B", Description: "Option B"},
		},
		Phase:   "planning",
		AskedAt: time.Now(),
	}

	if err := ws.SavePendingQuestion("test123", q); err != nil {
		t.Fatalf("SavePendingQuestion failed: %v", err)
	}

	loaded, err := ws.LoadPendingQuestion("test123")
	if err != nil {
		t.Fatalf("LoadPendingQuestion failed: %v", err)
	}

	if loaded.Question != q.Question {
		t.Errorf("loaded question = %q, want %q", loaded.Question, q.Question)
	}
	if len(loaded.Options) != 2 {
		t.Errorf("loaded options count = %d, want 2", len(loaded.Options))
	}
	if loaded.Phase != "planning" {
		t.Errorf("loaded phase = %q, want %q", loaded.Phase, "planning")
	}
}

func TestClearPendingQuestion(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	q := &PendingQuestion{Question: "Test?", Phase: "planning", AskedAt: time.Now()}
	if err := ws.SavePendingQuestion("test123", q); err != nil {
		t.Fatalf("SavePendingQuestion: %v", err)
	}

	if err := ws.ClearPendingQuestion("test123"); err != nil {
		t.Fatalf("ClearPendingQuestion failed: %v", err)
	}

	if ws.HasPendingQuestion("test123") {
		t.Error("pending question still exists after clear")
	}

	// Clear non-existent should not error
	if err := ws.ClearPendingQuestion("test123"); err != nil {
		t.Errorf("ClearPendingQuestion on non-existent failed: %v", err)
	}
}

// Type constructor tests

func TestNewActiveTask(t *testing.T) {
	task := NewActiveTask("task123", "file:test.md", ".mehrhof/work/task123")

	if task.ID != "task123" {
		t.Errorf("ID = %q, want %q", task.ID, "task123")
	}
	if task.Ref != "file:test.md" {
		t.Errorf("Ref = %q, want %q", task.Ref, "file:test.md")
	}
	if task.WorkDir != ".mehrhof/work/task123" {
		t.Errorf("WorkDir = %q, want %q", task.WorkDir, ".mehrhof/work/task123")
	}
	if task.State != "idle" {
		t.Errorf("State = %q, want %q", task.State, "idle")
	}
	if task.UseGit != false {
		t.Errorf("UseGit = %v, want false", task.UseGit)
	}
	if task.Started.IsZero() {
		t.Error("Started should not be zero")
	}
}

func TestNewTaskWork(t *testing.T) {
	source := SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: "Test content",
	}
	work := NewTaskWork("work123", source)

	if work.Version != "1" {
		t.Errorf("Version = %q, want %q", work.Version, "1")
	}
	if work.Metadata.ID != "work123" {
		t.Errorf("Metadata.ID = %q, want %q", work.Metadata.ID, "work123")
	}
	if work.Source.Type != "file" {
		t.Errorf("Source.Type = %q, want %q", work.Source.Type, "file")
	}
	if work.Metadata.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestNewSession(t *testing.T) {
	session := NewSession("planning", "claude", "planning")

	if session.Version != "1" {
		t.Errorf("Version = %q, want %q", session.Version, "1")
	}
	if session.Kind != "Session" {
		t.Errorf("Kind = %q, want %q", session.Kind, "Session")
	}
	if session.Metadata.Type != "planning" {
		t.Errorf("Metadata.Type = %q, want %q", session.Metadata.Type, "planning")
	}
	if session.Metadata.Agent != "claude" {
		t.Errorf("Metadata.Agent = %q, want %q", session.Metadata.Agent, "claude")
	}
	if session.Metadata.State != "planning" {
		t.Errorf("Metadata.State = %q, want %q", session.Metadata.State, "planning")
	}
	if session.Exchanges == nil {
		t.Error("Exchanges should not be nil")
	}
}

// Spec parsing tests

func TestParseSpec(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Test spec without frontmatter
	content := "# Spec Title\n\nThis is the spec content."
	if err := ws.SaveSpecification("test123", 1, content); err != nil {
		t.Fatalf("SaveSpec: %v", err)
	}

	spec, err := ws.ParseSpecification("test123", 1)
	if err != nil {
		t.Fatalf("ParseSpec: %v", err)
	}

	if spec.Number != 1 {
		t.Errorf("Number = %d, want 1", spec.Number)
	}
	if spec.Title != "Spec Title" {
		t.Errorf("Title = %q, want %q", spec.Title, "Spec Title")
	}
	if spec.Status != SpecificationStatusDraft {
		t.Errorf("Status = %q, want %q", spec.Status, SpecificationStatusDraft)
	}
}

func TestParseSpecWithFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Test spec with frontmatter
	content := `---
title: "Frontmatter Title"
status: ready
---

# Content Title

This is the spec content.`
	if err := ws.SaveSpecification("test123", 2, content); err != nil {
		t.Fatalf("SaveSpec: %v", err)
	}

	spec, err := ws.ParseSpecification("test123", 2)
	if err != nil {
		t.Fatalf("ParseSpec: %v", err)
	}

	// Note: ParseSpec extracts title from markdown heading (# Content Title),
	// which overwrites any frontmatter title. This is intentional - frontmatter
	// stores metadata like status, while visible title comes from the document.
	if spec.Title != "Content Title" {
		t.Errorf("Title = %q, want %q (from heading, not frontmatter)", spec.Title, "Content Title")
	}
	if spec.Status != "ready" {
		t.Errorf("Status = %q, want %q", spec.Status, "ready")
	}
}

func TestSaveSpecWithMeta(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	spec := &Specification{
		Number:  3,
		Title:   "Test Spec",
		Status:  SpecificationStatusReady,
		Content: "# Test Spec\n\nContent here.",
	}

	if err := ws.SaveSpecificationWithMeta("test123", spec); err != nil {
		t.Fatalf("SaveSpecWithMeta: %v", err)
	}

	// Load and verify
	loaded, err := ws.ParseSpecification("test123", 3)
	if err != nil {
		t.Fatalf("ParseSpec: %v", err)
	}

	if loaded.Title != "Test Spec" {
		t.Errorf("loaded Title = %q, want %q", loaded.Title, "Test Spec")
	}
	if loaded.Status != SpecificationStatusReady {
		t.Errorf("loaded Status = %q, want %q", loaded.Status, SpecificationStatusReady)
	}
}

func TestUpdateSpecStatus(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Create spec
	spec := &Specification{
		Number:  1,
		Title:   "Updateable Spec",
		Status:  SpecificationStatusDraft,
		Content: "# Updateable Spec\n\nContent.",
	}
	if err := ws.SaveSpecificationWithMeta("test123", spec); err != nil {
		t.Fatalf("SaveSpecWithMeta: %v", err)
	}

	// Update to implementing
	if err := ws.UpdateSpecificationStatus("test123", 1, SpecificationStatusImplementing); err != nil {
		t.Fatalf("UpdateSpecStatus: %v", err)
	}

	loaded, _ := ws.ParseSpecification("test123", 1)
	if loaded.Status != SpecificationStatusImplementing {
		t.Errorf("Status = %q, want %q", loaded.Status, SpecificationStatusImplementing)
	}

	// Update to done (should set CompletedAt)
	if err := ws.UpdateSpecificationStatus("test123", 1, SpecificationStatusDone); err != nil {
		t.Fatalf("UpdateSpecStatus to done: %v", err)
	}

	loaded, _ = ws.ParseSpecification("test123", 1)
	if loaded.Status != SpecificationStatusDone {
		t.Errorf("Status = %q, want %q", loaded.Status, SpecificationStatusDone)
	}
}

func TestListSpecsWithStatus(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Create specifications with different statuses
	specifications := []*Specification{
		{Number: 1, Status: SpecificationStatusDraft, Content: "# Specification 1"},
		{Number: 2, Status: SpecificationStatusReady, Content: "# Specification 2"},
		{Number: 3, Status: SpecificationStatusDone, Content: "# Specification 3"},
	}

	for _, s := range specifications {
		if err := ws.SaveSpecificationWithMeta("test123", s); err != nil {
			t.Fatalf("SaveSpecificationWithMeta(%d): %v", s.Number, err)
		}
	}

	result, err := ws.ListSpecificationsWithStatus("test123")
	if err != nil {
		t.Fatalf("ListSpecsWithStatus: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("result length = %d, want 3", len(result))
	}

	// Check statuses are loaded
	statusMap := make(map[int]string)
	for _, s := range result {
		statusMap[s.Number] = s.Status
	}

	if statusMap[1] != SpecificationStatusDraft {
		t.Errorf("spec 1 status = %q, want %q", statusMap[1], SpecificationStatusDraft)
	}
	if statusMap[2] != SpecificationStatusReady {
		t.Errorf("spec 2 status = %q, want %q", statusMap[2], SpecificationStatusReady)
	}
	if statusMap[3] != SpecificationStatusDone {
		t.Errorf("spec 3 status = %q, want %q", statusMap[3], SpecificationStatusDone)
	}
}

func TestGetSpecsSummary(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Create specifications with various statuses
	specifications := []*Specification{
		{Number: 1, Status: SpecificationStatusDraft, Content: "# S1"},
		{Number: 2, Status: SpecificationStatusDraft, Content: "# S2"},
		{Number: 3, Status: SpecificationStatusReady, Content: "# S3"},
		{Number: 4, Status: SpecificationStatusDone, Content: "# S4"},
	}

	for _, s := range specifications {
		if err := ws.SaveSpecificationWithMeta("test123", s); err != nil {
			t.Fatalf("SaveSpecificationWithMeta(%d): %v", s.Number, err)
		}
	}

	summary, err := ws.GetSpecificationsSummary("test123")
	if err != nil {
		t.Fatalf("GetSpecsSummary: %v", err)
	}

	if summary[SpecificationStatusDraft] != 2 {
		t.Errorf("draft count = %d, want 2", summary[SpecificationStatusDraft])
	}
	if summary[SpecificationStatusReady] != 1 {
		t.Errorf("ready count = %d, want 1", summary[SpecificationStatusReady])
	}
	if summary[SpecificationStatusDone] != 1 {
		t.Errorf("done count = %d, want 1", summary[SpecificationStatusDone])
	}
}

// Plan tests

func TestPlannedRoot(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	expected := filepath.Join(tmpDir, ".mehrhof", "planned")
	if ws.PlannedRoot() != expected {
		t.Errorf("PlannedRoot() = %q, want %q", ws.PlannedRoot(), expected)
	}
}

func TestPlannedPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	expected := filepath.Join(tmpDir, ".mehrhof", "planned", "plan123")
	if ws.PlannedPath("plan123") != expected {
		t.Errorf("PlannedPath() = %q, want %q", ws.PlannedPath("plan123"), expected)
	}
}

func TestGeneratePlanID(t *testing.T) {
	id1 := GeneratePlanID()
	if id1 == "" {
		t.Error("GeneratePlanID returned empty string")
	}

	// Should be timestamp format YYYY-MM-DD-HHMMSS
	if len(id1) != 17 { // 2006-01-02-150405
		t.Errorf("GeneratePlanID length = %d, want 17", len(id1))
	}
}

func TestCreatePlan(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	plan, err := ws.CreatePlan("test-plan", "Initial seed idea")
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}

	if plan.ID != "test-plan" {
		t.Errorf("ID = %q, want %q", plan.ID, "test-plan")
	}
	if plan.Seed != "Initial seed idea" {
		t.Errorf("Seed = %q, want %q", plan.Seed, "Initial seed idea")
	}
	if plan.Version != "1" {
		t.Errorf("Version = %q, want %q", plan.Version, "1")
	}

	// Check plan directory was created
	planPath := ws.PlannedPath("test-plan")
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		t.Error("plan directory was not created")
	}

	// Check plan-history.md was created
	historyPath := filepath.Join(planPath, "plan-history.md")
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		t.Error("plan-history.md was not created")
	}

	// Verify history contains seed
	data, _ := os.ReadFile(historyPath)
	if !contains(string(data), "Initial seed idea") {
		t.Error("history file does not contain seed")
	}
}

func TestSaveAndLoadPlan(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	plan, err := ws.CreatePlan("save-test", "")
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}

	plan.Title = "Updated Title"
	if err := ws.SavePlan(plan); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	loaded, err := ws.LoadPlan("save-test")
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}

	if loaded.Title != "Updated Title" {
		t.Errorf("loaded Title = %q, want %q", loaded.Title, "Updated Title")
	}
}

func TestAppendPlanHistory(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	if _, err := ws.CreatePlan("history-test", ""); err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}

	// Add user message
	if err := ws.AppendPlanHistory("history-test", "user", "User message"); err != nil {
		t.Fatalf("AppendPlanHistory (user): %v", err)
	}

	// Add assistant message
	if err := ws.AppendPlanHistory("history-test", "assistant", "Assistant response"); err != nil {
		t.Fatalf("AppendPlanHistory (assistant): %v", err)
	}

	// Load and verify
	plan, _ := ws.LoadPlan("history-test")
	if len(plan.History) != 2 {
		t.Fatalf("History length = %d, want 2", len(plan.History))
	}

	if plan.History[0].Role != "user" {
		t.Errorf("History[0].Role = %q, want %q", plan.History[0].Role, "user")
	}
	if plan.History[0].Content != "User message" {
		t.Errorf("History[0].Content = %q, want %q", plan.History[0].Content, "User message")
	}
	if plan.History[1].Role != "assistant" {
		t.Errorf("History[1].Role = %q, want %q", plan.History[1].Role, "assistant")
	}

	// Check markdown file was updated
	historyPath := filepath.Join(ws.PlannedPath("history-test"), "plan-history.md")
	data, _ := os.ReadFile(historyPath)
	content := string(data)

	if !contains(content, "User message") {
		t.Error("history markdown does not contain user message")
	}
	if !contains(content, "Assistant response") {
		t.Error("history markdown does not contain assistant response")
	}
}

func TestListPlans(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Initially empty
	plans, err := ws.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("ListPlans returned %d plans, want 0", len(plans))
	}

	// Create some plans
	if _, err := ws.CreatePlan("plan-a", ""); err != nil {
		t.Fatalf("CreatePlan(a): %v", err)
	}
	if _, err := ws.CreatePlan("plan-b", ""); err != nil {
		t.Fatalf("CreatePlan(b): %v", err)
	}

	plans, err = ws.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("ListPlans returned %d plans, want 2", len(plans))
	}
}

func TestDeletePlan(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	if _, err := ws.CreatePlan("delete-me", ""); err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}

	if err := ws.DeletePlan("delete-me"); err != nil {
		t.Fatalf("DeletePlan: %v", err)
	}

	// Verify deleted
	if _, err := os.Stat(ws.PlannedPath("delete-me")); !os.IsNotExist(err) {
		t.Error("plan directory still exists after delete")
	}

	plans, _ := ws.ListPlans()
	if len(plans) != 0 {
		t.Error("plan still appears in list after delete")
	}
}

func TestSpecStatusConstants(t *testing.T) {
	if SpecificationStatusDraft != "draft" {
		t.Errorf("SpecificationStatusDraft = %q, want %q", SpecificationStatusDraft, "draft")
	}
	if SpecificationStatusReady != "ready" {
		t.Errorf("SpecificationStatusReady = %q, want %q", SpecificationStatusReady, "ready")
	}
	if SpecificationStatusImplementing != "implementing" {
		t.Errorf("SpecificationStatusImplementing = %q, want %q", SpecificationStatusImplementing, "implementing")
	}
	if SpecificationStatusDone != "done" {
		t.Errorf("SpecificationStatusDone = %q, want %q", SpecificationStatusDone, "done")
	}
}

func TestGetEnvForAgent(t *testing.T) {
	tests := []struct {
		env       map[string]string
		want      map[string]string
		name      string
		agentName string
	}{
		{
			name: "filters claude vars and strips prefix",
			env: map[string]string{
				"CLAUDE_ANTHROPIC_API_KEY": "sk-ant-xxx",
				"CLAUDE_MAX_TOKENS":        "4096",
				"OPENAI_API_KEY":           "sk-openai-xxx",
				"UNRELATED_VAR":            "value",
			},
			agentName: "claude",
			want: map[string]string{
				"ANTHROPIC_API_KEY": "sk-ant-xxx",
				"MAX_TOKENS":        "4096",
			},
		},
		{
			name: "filters openai vars",
			env: map[string]string{
				"CLAUDE_ANTHROPIC_API_KEY": "sk-ant-xxx",
				"OPENAI_API_KEY":           "sk-openai-xxx",
				"OPENAI_MODEL":             "gpt-4",
			},
			agentName: "openai",
			want: map[string]string{
				"API_KEY": "sk-openai-xxx",
				"MODEL":   "gpt-4",
			},
		},
		{
			name: "case insensitive agent name",
			env: map[string]string{
				"CLAUDE_FOO": "bar",
			},
			agentName: "Claude",
			want: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:      "empty env returns empty map",
			env:       map[string]string{},
			agentName: "claude",
			want:      map[string]string{},
		},
		{
			name: "no matching prefix returns empty map",
			env: map[string]string{
				"OPENAI_API_KEY": "sk-xxx",
			},
			agentName: "claude",
			want:      map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &WorkspaceConfig{Env: tt.env}
			got := cfg.GetEnvForAgent(tt.agentName)

			if len(got) != len(tt.want) {
				t.Errorf("GetEnvForAgent() returned %d vars, want %d", len(got), len(tt.want))
			}

			for k, wantV := range tt.want {
				if gotV, ok := got[k]; !ok {
					t.Errorf("GetEnvForAgent() missing key %q", k)
				} else if gotV != wantV {
					t.Errorf("GetEnvForAgent()[%q] = %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}

// Helper function.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

func TestActiveTaskStruct(t *testing.T) {
	now := time.Now()
	task := ActiveTask{
		ID:           "task-789",
		Ref:          "github:owner/repo#123",
		WorkDir:      "/work/task-789",
		State:        "planning",
		Branch:       "feature/task-789",
		UseGit:       true,
		WorktreePath: "/worktrees/task-789",
		Started:      now,
	}

	if task.ID != "task-789" {
		t.Errorf("ID = %q, want %q", task.ID, "task-789")
	}
	if task.Branch != "feature/task-789" {
		t.Errorf("Branch = %q, want %q", task.Branch, "feature/task-789")
	}
	if task.WorktreePath != "/worktrees/task-789" {
		t.Errorf("WorktreePath = %q, want %q", task.WorktreePath, "/worktrees/task-789")
	}
}

func TestTaskWorkStruct(t *testing.T) {
	now := time.Now()
	work := TaskWork{
		Version: "1",
		Metadata: WorkMetadata{
			ID:          "task-abc",
			Title:       "Test Task",
			CreatedAt:   now,
			UpdatedAt:   now,
			ExternalKey: "FEAT-123",
			TaskType:    "feature",
			Slug:        "test-task",
		},
		Source: SourceInfo{
			Type:    "directory",
			Ref:     ".mehrhof/plans/my-plan",
			ReadAt:  now,
			Content: "# Plan",
		},
		Git: GitInfo{
			Branch:        "feature/FEAT-123--test-task",
			BaseBranch:    "main",
			CommitPrefix:  "[FEAT-123]",
			BranchPattern: "{type}/{key}--{slug}",
		},
		Agent: AgentInfo{
			Name:   "claude",
			Source: "workspace",
		},
	}

	if work.Metadata.ExternalKey != "FEAT-123" {
		t.Errorf("Metadata.ExternalKey = %q, want %q", work.Metadata.ExternalKey, "FEAT-123")
	}
	if work.Git.CommitPrefix != "[FEAT-123]" {
		t.Errorf("Git.CommitPrefix = %q, want %q", work.Git.CommitPrefix, "[FEAT-123]")
	}
	if work.Agent.Name != "claude" {
		t.Errorf("Agent.Name = %q, want %q", work.Agent.Name, "claude")
	}
}

func TestSourceInfoStruct(t *testing.T) {
	now := time.Now()
	source := SourceInfo{
		Type:   "directory",
		Ref:    ".mehrhof/plans/my-plan",
		ReadAt: now,
		Files:  []string{"source/task.md", "source/notes.md"},
	}

	if len(source.Files) != 2 {
		t.Errorf("Files length = %d, want 2", len(source.Files))
	}
	if source.Files[0] != "source/task.md" {
		t.Errorf("Files[0] = %q, want %q", source.Files[0], "source/task.md")
	}
}

func TestSessionStruct(t *testing.T) {
	now := time.Now()
	session := Session{
		Version: "1",
		Kind:    "Session",
		Metadata: SessionMetadata{
			StartedAt: now,
			EndedAt:   now.Add(time.Hour),
			Type:      "implementation",
			Agent:     "claude",
			State:     "implementing",
		},
		Usage: &UsageInfo{
			InputTokens:  1000,
			OutputTokens: 500,
			CachedTokens: 200,
			CostUSD:      0.05,
		},
		Exchanges: []Exchange{
			{
				Role:      "user",
				Timestamp: now,
				Content:   "Implement the feature",
			},
			{
				Role:      "agent",
				Timestamp: now.Add(time.Minute),
				Content:   "Done!",
				FilesChanged: []FileChange{
					{Path: "main.go", Operation: "update"},
				},
			},
		},
	}

	if session.Usage.InputTokens != 1000 {
		t.Errorf("Usage.InputTokens = %d, want 1000", session.Usage.InputTokens)
	}
	if len(session.Exchanges) != 2 {
		t.Errorf("Exchanges length = %d, want 2", len(session.Exchanges))
	}
	if len(session.Exchanges[1].FilesChanged) != 1 {
		t.Errorf("FilesChanged length = %d, want 1", len(session.Exchanges[1].FilesChanged))
	}
}

func TestCheckpointStruct(t *testing.T) {
	now := time.Now()
	cp := Checkpoint{
		ID:        "cp-001",
		Commit:    "abc123def456",
		Message:   "Checkpoint after planning",
		State:     "planning",
		CreatedAt: now,
	}

	if cp.ID != "cp-001" {
		t.Errorf("ID = %q, want %q", cp.ID, "cp-001")
	}
	if cp.Commit != "abc123def456" {
		t.Errorf("Commit = %q, want %q", cp.Commit, "abc123def456")
	}
}

func TestSpecificationStruct(t *testing.T) {
	now := time.Now()
	spec := Specification{
		Number:      1,
		Title:       "Feature Specification",
		Description: "Implement the feature",
		Status:      SpecificationStatusReady,
		CreatedAt:   now,
		UpdatedAt:   now,
		Sections:    []string{"Overview", "Requirements", "Implementation"},
		Content:     "# Feature Specification\n\n## Overview\n...",
	}

	if spec.Number != 1 {
		t.Errorf("Number = %d, want 1", spec.Number)
	}
	if spec.Status != "ready" {
		t.Errorf("Status = %q, want %q", spec.Status, "ready")
	}
	if len(spec.Sections) != 3 {
		t.Errorf("Sections length = %d, want 3", len(spec.Sections))
	}
}

func TestNoteStruct(t *testing.T) {
	now := time.Now()
	note := Note{
		Timestamp: now,
		Content:   "This is a note",
		State:     "idle",
	}

	if note.Content != "This is a note" {
		t.Errorf("Content = %q, want %q", note.Content, "This is a note")
	}
	if note.State != "idle" {
		t.Errorf("State = %q, want %q", note.State, "idle")
	}
}

func TestNotesFileStruct(t *testing.T) {
	notes := NotesFile{
		Notes: []Note{
			{Content: "Note 1"},
			{Content: "Note 2"},
		},
	}

	if len(notes.Notes) != 2 {
		t.Errorf("Notes length = %d, want 2", len(notes.Notes))
	}
}

func TestAgentInfoStruct(t *testing.T) {
	info := AgentInfo{
		Name:      "claude",
		Source:    "workspace",
		InlineEnv: map[string]string{"API_KEY": "secret"},
		Steps: map[string]StepAgentInfo{
			"planning": {
				Name:      "glm",
				Source:    "cli-step",
				InlineEnv: map[string]string{"MODEL": "glm-4"},
			},
		},
	}

	if info.Name != "claude" {
		t.Errorf("Name = %q, want %q", info.Name, "claude")
	}
	if info.Steps["planning"].Name != "glm" {
		t.Errorf("Steps[planning].Name = %q, want %q", info.Steps["planning"].Name, "glm")
	}
}

// Tests for custom work directory configuration

func TestOpenWorkspace_WorkRoot(t *testing.T) {
	homeDir := t.TempDir()

	tests := []struct {
		name            string
		cfg             *WorkspaceConfig
		wantWorkRootSfx string // expected suffix of WorkRoot
	}{
		{
			name:            "nil config uses default",
			cfg:             nil,
			wantWorkRootSfx: "/work",
		},
		{
			name:            "empty storage uses default",
			cfg:             &WorkspaceConfig{Storage: StorageSettings{HomeDir: homeDir}},
			wantWorkRootSfx: "/work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			ws, err := OpenWorkspace(context.Background(), tmpDir, tt.cfg)
			if err != nil {
				t.Fatalf("OpenWorkspace: %v", err)
			}

			// WorkRoot is in ~/.valksor/mehrhof/workspaces/<project-id>/work (fixed path)
			if !strings.HasSuffix(ws.WorkRoot(), tt.wantWorkRootSfx) {
				t.Errorf("WorkRoot() = %q, want suffix %q", ws.WorkRoot(), tt.wantWorkRootSfx)
			}

			// Verify TaskRoot is always .mehrhof in project directory
			expectedTaskRoot := filepath.Join(tmpDir, ".mehrhof")
			if ws.TaskRoot() != expectedTaskRoot {
				t.Errorf("TaskRoot() = %q, want %q", ws.TaskRoot(), expectedTaskRoot)
			}
		})
	}
}

func TestCreateWork_WorkPath(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	cfg := &WorkspaceConfig{
		Storage: StorageSettings{HomeDir: homeDir},
	}

	ws, err := OpenWorkspace(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("OpenWorkspace: %v", err)
	}

	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md", Content: "test"}
	work, err := ws.CreateWork("task123", source)
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Verify work was created in fixed location within workspace data dir
	workPath := ws.WorkPath("task123")
	if !strings.HasSuffix(workPath, "/work/task123") {
		t.Errorf("WorkPath() = %q, want suffix /work/task123", workPath)
	}

	// Verify work directory actually exists
	if _, err := os.Stat(workPath); os.IsNotExist(err) {
		t.Error("work directory was not created")
	}

	// Verify we can load the work back
	loaded, err := ws.LoadWork("task123")
	if err != nil {
		t.Fatalf("LoadWork: %v", err)
	}
	if loaded.Metadata.ID != work.Metadata.ID {
		t.Error("loaded work does not match")
	}
}

func TestEnsureInitialized_WorkDir(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	cfg := &WorkspaceConfig{
		Storage: StorageSettings{HomeDir: homeDir},
	}

	ws, err := OpenWorkspace(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("OpenWorkspace: %v", err)
	}

	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Verify work directory was created within workspace data dir
	workRoot := ws.WorkRoot()
	if _, err := os.Stat(workRoot); os.IsNotExist(err) {
		t.Error("work directory was not created")
	}
	if !strings.HasSuffix(workRoot, "/work") {
		t.Errorf("WorkRoot() = %q, want suffix /work", workRoot)
	}
}

func TestListWorks_WorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &WorkspaceConfig{
		Storage: StorageSettings{},
	}

	ws, err := OpenWorkspace(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("OpenWorkspace: %v", err)
	}
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create several tasks
	source := SourceInfo{Type: "file", Ref: "task.md"}
	for i := 1; i <= 3; i++ {
		if _, err := ws.CreateWork(fmt.Sprintf("task%d", i), source); err != nil {
			t.Fatalf("CreateWork: %v", err)
		}
	}

	// List and verify
	works, err := ws.ListWorks()
	if err != nil {
		t.Fatalf("ListWorks: %v", err)
	}

	if len(works) != 3 {
		t.Errorf("ListWorks() returned %d, want 3", len(works))
	}
}

func TestDeleteWork_WorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &WorkspaceConfig{
		Storage: StorageSettings{},
	}

	ws, err := OpenWorkspace(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("OpenWorkspace: %v", err)
	}
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create a task
	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test-task", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Verify it exists
	if !ws.WorkExists("test-task") {
		t.Fatal("work should exist after creation")
	}

	// Delete it
	if err := ws.DeleteWork("test-task"); err != nil {
		t.Fatalf("DeleteWork: %v", err)
	}

	// Verify it's gone
	if ws.WorkExists("test-task") {
		t.Error("work should not exist after deletion")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Usage tracking tests (AddUsage, FlushUsage, etc.)
// ──────────────────────────────────────────────────────────────────────────────

func TestAddUsage_SingleCall(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	taskID := "test-usage-1"
	source := SourceInfo{Type: "file", Ref: "test.md"}
	if _, err := ws.CreateWork(taskID, source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Add usage
	if err := ws.AddUsage(taskID, "planning", 1000, 500, 100, 0.01); err != nil {
		t.Fatalf("AddUsage: %v", err)
	}

	// Flush to persist
	if err := ws.FlushUsage(); err != nil {
		t.Fatalf("FlushUsage: %v", err)
	}

	// Load work and verify
	work, err := ws.LoadWork(taskID)
	if err != nil {
		t.Fatalf("LoadWork: %v", err)
	}

	if work.Costs.TotalInputTokens != 1000 {
		t.Errorf("TotalInputTokens = %d, want 1000", work.Costs.TotalInputTokens)
	}
	if work.Costs.TotalOutputTokens != 500 {
		t.Errorf("TotalOutputTokens = %d, want 500", work.Costs.TotalOutputTokens)
	}
	if work.Costs.TotalCachedTokens != 100 {
		t.Errorf("TotalCachedTokens = %d, want 100", work.Costs.TotalCachedTokens)
	}
	if work.Costs.TotalCostUSD != 0.01 {
		t.Errorf("TotalCostUSD = %f, want 0.01", work.Costs.TotalCostUSD)
	}

	// Check ByStep stats
	if work.Costs.ByStep == nil {
		t.Fatal("ByStep should not be nil")
	}
	stepStats, ok := work.Costs.ByStep["planning"]
	if !ok {
		t.Fatal("ByStep should contain 'planning' stats")
	}
	if stepStats.InputTokens != 1000 {
		t.Errorf("stepStats.InputTokens = %d, want 1000", stepStats.InputTokens)
	}
	if stepStats.OutputTokens != 500 {
		t.Errorf("stepStats.OutputTokens = %d, want 500", stepStats.OutputTokens)
	}
	if stepStats.CachedTokens != 100 {
		t.Errorf("stepStats.CachedTokens = %d, want 100", stepStats.CachedTokens)
	}
	if stepStats.CostUSD != 0.01 {
		t.Errorf("stepStats.CostUSD = %f, want 0.01", stepStats.CostUSD)
	}
	if stepStats.Calls != 1 {
		t.Errorf("stepStats.Calls = %d, want 1", stepStats.Calls)
	}
}

func TestAddUsage_MultipleCallsAccumulate(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	taskID := "test-usage-2"
	source := SourceInfo{Type: "file", Ref: "test.md"}
	if _, err := ws.CreateWork(taskID, source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Add usage multiple times
	for i := range 5 {
		if err := ws.AddUsage(taskID, "planning", 1000, 500, 100, 0.01); err != nil {
			t.Fatalf("AddUsage %d: %v", i, err)
		}
	}

	// Flush to persist
	if err := ws.FlushUsage(); err != nil {
		t.Fatalf("FlushUsage: %v", err)
	}

	// Load work and verify accumulation
	work, err := ws.LoadWork(taskID)
	if err != nil {
		t.Fatalf("LoadWork: %v", err)
	}

	if work.Costs.TotalInputTokens != 5000 { // 5 * 1000
		t.Errorf("TotalInputTokens = %d, want 5000", work.Costs.TotalInputTokens)
	}
	if work.Costs.TotalOutputTokens != 2500 { // 5 * 500
		t.Errorf("TotalOutputTokens = %d, want 2500", work.Costs.TotalOutputTokens)
	}
	if work.Costs.TotalCostUSD != 0.05 { // 5 * 0.01
		t.Errorf("TotalCostUSD = %f, want 0.05", work.Costs.TotalCostUSD)
	}

	// Check ByStep stats
	stepStats := work.Costs.ByStep["planning"]
	if stepStats.Calls != 5 {
		t.Errorf("stepStats.Calls = %d, want 5", stepStats.Calls)
	}
}

func TestAddUsage_MultipleSteps(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	taskID := "test-usage-3"
	source := SourceInfo{Type: "file", Ref: "test.md"}
	if _, err := ws.CreateWork(taskID, source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Add usage for different steps
	if err := ws.AddUsage(taskID, "planning", 1000, 500, 100, 0.01); err != nil {
		t.Fatalf("AddUsage planning: %v", err)
	}
	if err := ws.AddUsage(taskID, "implementing", 2000, 1000, 200, 0.03); err != nil {
		t.Fatalf("AddUsage implementing: %v", err)
	}
	if err := ws.AddUsage(taskID, "reviewing", 500, 300, 50, 0.005); err != nil {
		t.Fatalf("AddUsage reviewing: %v", err)
	}

	// Flush to persist
	if err := ws.FlushUsage(); err != nil {
		t.Fatalf("FlushUsage: %v", err)
	}

	// Load work and verify
	work, err := ws.LoadWork(taskID)
	if err != nil {
		t.Fatalf("LoadWork: %v", err)
	}

	// Check totals
	if work.Costs.TotalInputTokens != 3500 { // 1000 + 2000 + 500
		t.Errorf("TotalInputTokens = %d, want 3500", work.Costs.TotalInputTokens)
	}
	if work.Costs.TotalOutputTokens != 1800 { // 500 + 1000 + 300
		t.Errorf("TotalOutputTokens = %d, want 1800", work.Costs.TotalOutputTokens)
	}

	// Check each step
	if len(work.Costs.ByStep) != 3 {
		t.Errorf("ByStep has %d entries, want 3", len(work.Costs.ByStep))
	}

	// Verify planning step
	planningStats := work.Costs.ByStep["planning"]
	if planningStats.InputTokens != 1000 {
		t.Errorf("planning.InputTokens = %d, want 1000", planningStats.InputTokens)
	}

	// Verify implementing step
	implementingStats := work.Costs.ByStep["implementing"]
	if implementingStats.InputTokens != 2000 {
		t.Errorf("implementing.InputTokens = %d, want 2000", implementingStats.InputTokens)
	}

	// Verify reviewing step
	reviewingStats := work.Costs.ByStep["reviewing"]
	if reviewingStats.InputTokens != 500 {
		t.Errorf("reviewing.InputTokens = %d, want 500", reviewingStats.InputTokens)
	}
}

func TestAddUsage_AutoFlushOnThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	taskID := "test-usage-4"
	source := SourceInfo{Type: "file", Ref: "test.md"}
	if _, err := ws.CreateWork(taskID, source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Add enough usage to trigger auto-flush (threshold is 100 calls)
	// The auto-flush should happen when totalCalls >= defaultUsageFlushThreshold
	for i := range 100 {
		if err := ws.AddUsage(taskID, "planning", 100, 50, 10, 0.001); err != nil {
			t.Fatalf("AddUsage %d: %v", i, err)
		}
	}

	// Data should be persisted even without explicit flush
	work, err := ws.LoadWork(taskID)
	if err != nil {
		t.Fatalf("LoadWork: %v", err)
	}

	if work.Costs.TotalInputTokens != 10000 { // 100 * 100
		t.Errorf("TotalInputTokens = %d, want 10000", work.Costs.TotalInputTokens)
	}
}

func TestFlushUsage_EmptyBuffer(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Flush with no usage data should not error
	if err := ws.FlushUsage(); err != nil {
		t.Errorf("FlushUsage with empty buffer: %v", err)
	}
}

func TestAddUsage_MultipleTasks(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	taskID1 := "test-task-1"
	taskID2 := "test-task-2"
	source := SourceInfo{Type: "file", Ref: "test.md"}

	if _, err := ws.CreateWork(taskID1, source); err != nil {
		t.Fatalf("CreateWork task1: %v", err)
	}
	if _, err := ws.CreateWork(taskID2, source); err != nil {
		t.Fatalf("CreateWork task2: %v", err)
	}

	// Add usage for both tasks
	if err := ws.AddUsage(taskID1, "planning", 1000, 500, 100, 0.01); err != nil {
		t.Fatalf("AddUsage task1: %v", err)
	}
	if err := ws.AddUsage(taskID2, "planning", 2000, 1000, 200, 0.02); err != nil {
		t.Fatalf("AddUsage task2: %v", err)
	}

	// Flush
	if err := ws.FlushUsage(); err != nil {
		t.Fatalf("FlushUsage: %v", err)
	}

	// Verify task1
	work1, err := ws.LoadWork(taskID1)
	if err != nil {
		t.Fatalf("LoadWork task1: %v", err)
	}
	if work1.Costs.TotalInputTokens != 1000 {
		t.Errorf("task1 TotalInputTokens = %d, want 1000", work1.Costs.TotalInputTokens)
	}

	// Verify task2
	work2, err := ws.LoadWork(taskID2)
	if err != nil {
		t.Fatalf("LoadWork task2: %v", err)
	}
	if work2.Costs.TotalInputTokens != 2000 {
		t.Errorf("task2 TotalInputTokens = %d, want 2000", work2.Costs.TotalInputTokens)
	}
}

func TestAddUsage_InvalidTask(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Try to add usage for non-existent task
	// This should not error immediately (usage is buffered)
	// but will error on flush when trying to load the work
	err := ws.AddUsage("nonexistent-task", "planning", 1000, 500, 100, 0.01)
	if err != nil {
		t.Fatalf("AddUsage should not error immediately: %v", err)
	}

	// Flush should error because task doesn't exist
	err = ws.FlushUsage()
	if err == nil {
		t.Error("FlushUsage should error when task doesn't exist")
	}
}

func TestValidateSpecification(t *testing.T) {
	tests := []struct {
		name            string
		specContent     string
		wantValid       bool
		wantErrors      int
		wantWarnings    int
		errorContains   []string
		warningContains []string
	}{
		{
			name: "valid specification",
			specContent: `## Request
Implement a feature

## Plan
1. First step
2. Second step

## Context
path/to/file:1-10: description

## Unknowns
0. None

## Complete Condition
- manual: Check the feature works
- run: make test

## Status
planned 2024-01-01 12:00
`,
			wantValid:       true,
			wantErrors:      0,
			wantWarnings:    0,
			errorContains:   []string{},
			warningContains: []string{},
		},
		{
			name: "missing required sections",
			specContent: `## Request
Implement a feature

## Plan
1. First step

## Context
path/to/file:1-10: description
`,
			wantValid:    false,
			wantErrors:   3,
			wantWarnings: 0,
			errorContains: []string{
				"Missing: ## Unknowns",
				"Missing: ## Complete Condition",
				"Missing: ## Status",
			},
			warningContains: []string{},
		},
		{
			name: "plan with only one step",
			specContent: `## Request
Implement a feature

## Plan
1. Only step

## Context
path/to/file:1-10: description

## Unknowns
0. None

## Complete Condition
- manual: Check it
- run: make test

## Status
planned 2024-01-01 12:00
`,
			wantValid:       true,
			wantErrors:      0,
			wantWarnings:    1,
			errorContains:   []string{},
			warningContains: []string{"Plan should have at least 2 steps"},
		},
		{
			name: "unknowns with user input required",
			specContent: `## Request
Implement a feature

## Plan
1. First step
2. Second step

## Context
path/to/file:1-10: description

## Unknowns
1. What should we do?
   user input required

## Complete Condition
- manual: Check it
- run: make test

## Status
planned 2024-01-01 12:00
`,
			wantValid:       true,
			wantErrors:      0,
			wantWarnings:    1,
			errorContains:   []string{},
			warningContains: []string{"Unknowns should have default answers"},
		},
		{
			name: "missing manual validation",
			specContent: `## Request
Implement a feature

## Plan
1. First step
2. Second step

## Context
path/to/file:1-10: description

## Unknowns
0. None

## Complete Condition
- run: make test

## Status
planned 2024-01-01 12:00
`,
			wantValid:       true,
			wantErrors:      0,
			wantWarnings:    1,
			errorContains:   []string{},
			warningContains: []string{"Complete Condition should include manual validation step"},
		},
		{
			name: "missing run validation",
			specContent: `## Request
Implement a feature

## Plan
1. First step
2. Second step

## Context
path/to/file:1-10: description

## Unknowns
0. None

## Complete Condition
- manual: Check it

## Status
planned 2024-01-01 12:00
`,
			wantValid:       true,
			wantErrors:      0,
			wantWarnings:    1,
			errorContains:   []string{},
			warningContains: []string{"Complete Condition should include run validation step"},
		},
		{
			name: "case insensitive section matching",
			specContent: `## request
Implement a feature

## plan
1. First step
2. Second step

## context
path/to/file:1-10: description

## unknowns
0. None

## complete condition
- manual: Check it
- run: make test

## status
planned 2024-01-01 12:00
`,
			wantValid:       true,
			wantErrors:      0,
			wantWarnings:    0,
			errorContains:   []string{},
			warningContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			ws := openTestWorkspace(t, tmpDir)

			// Create a test task
			taskID := "test-task"
			source := SourceInfo{
				Type:    "file",
				Ref:     "test.md",
				Content: "Test task",
			}
			_, err := ws.CreateWork(taskID, source)
			if err != nil {
				t.Fatalf("CreateWork: %v", err)
			}

			// Save specification
			specNum := 1
			if err := ws.SaveSpecification(taskID, specNum, tt.specContent); err != nil {
				t.Fatalf("SaveSpecification: %v", err)
			}

			// Validate
			result, err := ws.ValidateSpecification(taskID, specNum)
			if err != nil {
				t.Fatalf("ValidateSpecification: %v", err)
			}

			// Check validity
			if result.IsValid != tt.wantValid {
				t.Errorf("ValidateSpecification() IsValid = %v, want %v", result.IsValid, tt.wantValid)
			}

			// Check error count
			if len(result.Errors) != tt.wantErrors {
				t.Errorf("ValidateSpecification() Errors count = %d, want %d", len(result.Errors), tt.wantErrors)
			}

			// Check warning count
			if len(result.Warnings) != tt.wantWarnings {
				t.Errorf("ValidateSpecification() Warnings count = %d, want %d", len(result.Warnings), tt.wantWarnings)
			}

			// Check error content
			for _, expectedErr := range tt.errorContains {
				found := false
				for _, err := range result.Errors {
					if strings.Contains(err, expectedErr) {
						found = true

						break
					}
				}
				if !found {
					t.Errorf("ValidateSpecification() Errors should contain %q", expectedErr)
				}
			}

			// Check warning content
			for _, expectedWarn := range tt.warningContains {
				found := false
				for _, warn := range result.Warnings {
					if strings.Contains(warn, expectedWarn) {
						found = true

						break
					}
				}
				if !found {
					t.Errorf("ValidateSpecification() Warnings should contain %q", expectedWarn)
				}
			}
		})
	}
}

// TestSpecification_ComponentField tests the Component field in specification frontmatter.
func TestSpecification_ComponentField(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create work directory first
	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	tests := []struct {
		name              string
		component         string
		expectedComponent string
	}{
		{
			name:              "backend component",
			component:         "backend",
			expectedComponent: "backend",
		},
		{
			name:              "frontend component",
			component:         "frontend",
			expectedComponent: "frontend",
		},
		{
			name:              "tests component",
			component:         "tests",
			expectedComponent: "tests",
		},
		{
			name:              "empty component",
			component:         "",
			expectedComponent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := `---
title: Test Spec
status: ready
component: ` + tt.component + `
---
# Test Specification Content
`
			if err := ws.SaveSpecification("test123", 1, content); err != nil {
				t.Fatalf("SaveSpecification failed: %v", err)
			}

			// Parse the specification
			spec, err := ws.ParseSpecification("test123", 1)
			if err != nil {
				t.Fatalf("ParseSpecification failed: %v", err)
			}

			if spec.Component != tt.expectedComponent {
				t.Errorf("Component = %q, want %q", spec.Component, tt.expectedComponent)
			}
		})
	}
}

// TestAgentAliasConfig_ComponentsField tests the Components field in agent alias config.
func TestAgentAliasConfig_ComponentsField(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Load config
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Add agent alias with components
	if cfg.Agents == nil {
		cfg.Agents = make(map[string]AgentAliasConfig)
	}

	cfg.Agents["backend-agent"] = AgentAliasConfig{
		Extends:    "claude",
		Components: []string{"backend", "api"},
	}

	cfg.Agents["frontend-agent"] = AgentAliasConfig{
		Extends:    "claude",
		Components: []string{"frontend", "ui"},
	}

	// Save config
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Reload and verify
	loadedCfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig (reload) failed: %v", err)
	}

	// Verify backend agent
	backendAgent, ok := loadedCfg.Agents["backend-agent"]
	if !ok {
		t.Fatal("backend-agent alias not found")
	}
	if backendAgent.Extends != "claude" {
		t.Errorf("backend-agent.Extends = %q, want %q", backendAgent.Extends, "claude")
	}
	expectedComponents := []string{"backend", "api"}
	if len(backendAgent.Components) != len(expectedComponents) {
		t.Fatalf("backend-agent.Components length = %d, want %d", len(backendAgent.Components), len(expectedComponents))
	}
	for i, comp := range backendAgent.Components {
		if comp != expectedComponents[i] {
			t.Errorf("backend-agent.Components[%d] = %q, want %q", i, comp, expectedComponents[i])
		}
	}

	// Verify frontend agent
	frontendAgent, ok := loadedCfg.Agents["frontend-agent"]
	if !ok {
		t.Fatal("frontend-agent alias not found")
	}
	expectedComponents = []string{"frontend", "ui"}
	if len(frontendAgent.Components) != len(expectedComponents) {
		t.Fatalf("frontend-agent.Components length = %d, want %d", len(frontendAgent.Components), len(expectedComponents))
	}
	for i, comp := range frontendAgent.Components {
		if comp != expectedComponents[i] {
			t.Errorf("frontend-agent.Components[%d] = %q, want %q", i, comp, expectedComponents[i])
		}
	}
}

func TestOpenWorkspace_CodeDir(t *testing.T) {
	tests := []struct {
		name         string
		setupCodeDir func(t *testing.T, projectRoot string) string // returns code_dir value
		wantErr      bool
		errContains  string
		checkRoot    func(t *testing.T, ws *Workspace, projectRoot string)
	}{
		{
			name: "empty code_dir defaults to project root",
			setupCodeDir: func(t *testing.T, projectRoot string) string {
				t.Helper()

				return ""
			},
			checkRoot: func(t *testing.T, ws *Workspace, projectRoot string) {
				t.Helper()
				if ws.CodeRoot() != projectRoot {
					t.Errorf("CodeRoot() = %q, want %q (same as Root)", ws.CodeRoot(), projectRoot)
				}
				if ws.CodeRoot() != ws.Root() {
					t.Error("CodeRoot() should equal Root() when code_dir is empty")
				}
			},
		},
		{
			name: "absolute code_dir path",
			setupCodeDir: func(t *testing.T, projectRoot string) string {
				t.Helper()
				codeDir := t.TempDir()

				return codeDir
			},
			checkRoot: func(t *testing.T, ws *Workspace, projectRoot string) {
				t.Helper()
				if ws.CodeRoot() == projectRoot {
					t.Error("CodeRoot() should differ from Root() when code_dir is set")
				}
			},
		},
		{
			name: "relative code_dir path",
			setupCodeDir: func(t *testing.T, projectRoot string) string {
				t.Helper()
				// Create a sibling directory
				codeDir := filepath.Join(filepath.Dir(projectRoot), "code-target")
				if err := os.MkdirAll(codeDir, 0o755); err != nil {
					t.Fatalf("MkdirAll: %v", err)
				}
				// Relative path from project root to sibling
				rel, err := filepath.Rel(projectRoot, codeDir)
				if err != nil {
					t.Fatalf("Rel: %v", err)
				}

				return rel
			},
			checkRoot: func(t *testing.T, ws *Workspace, projectRoot string) {
				t.Helper()
				if ws.CodeRoot() == projectRoot {
					t.Error("CodeRoot() should differ from Root()")
				}
				// Should end with code-target
				if !strings.HasSuffix(ws.CodeRoot(), "code-target") {
					t.Errorf("CodeRoot() = %q, want suffix 'code-target'", ws.CodeRoot())
				}
			},
		},
		{
			name: "nonexistent code_dir returns error",
			setupCodeDir: func(t *testing.T, projectRoot string) string {
				t.Helper()

				return "/nonexistent/path/that/does/not/exist"
			},
			wantErr:     true,
			errContains: "code_dir",
		},
		{
			name: "code_dir is a file not a directory",
			setupCodeDir: func(t *testing.T, projectRoot string) string {
				t.Helper()
				filePath := filepath.Join(t.TempDir(), "not-a-dir.txt")
				if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}

				return filePath
			},
			wantErr:     true,
			errContains: "not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectRoot := t.TempDir()
			homeDir := t.TempDir()

			cfg := NewDefaultWorkspaceConfig()
			cfg.Storage.HomeDir = homeDir
			cfg.Project.CodeDir = tt.setupCodeDir(t, projectRoot)

			ws, err := OpenWorkspace(context.Background(), projectRoot, cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tt.checkRoot(t, ws, projectRoot)
		})
	}
}

func TestWorkspace_CodeAbsolutePath_UsesCodeRoot(t *testing.T) {
	projectRoot := t.TempDir()
	codeDir := t.TempDir()
	homeDir := t.TempDir()

	cfg := NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir
	cfg.Project.CodeDir = codeDir

	ws, err := OpenWorkspace(context.Background(), projectRoot, cfg)
	if err != nil {
		t.Fatalf("OpenWorkspace: %v", err)
	}

	// Relative paths should resolve against codeRoot
	result := ws.CodeAbsolutePath("src/main.go")
	expected := filepath.Join(codeDir, "src/main.go")
	if result != expected {
		t.Errorf("CodeAbsolutePath(relative) = %q, want %q", result, expected)
	}

	// Absolute paths should be returned as-is
	absPath := "/some/absolute/path"
	if ws.CodeAbsolutePath(absPath) != absPath {
		t.Errorf("CodeAbsolutePath(absolute) = %q, want %q", ws.CodeAbsolutePath(absPath), absPath)
	}
}

func TestProjectSettings_EnvExpansion(t *testing.T) {
	projectRoot := t.TempDir()
	codeDir := t.TempDir()

	// Set env var
	t.Setenv("TEST_CODE_DIR", codeDir)

	// Create config with env var reference
	ws := openTestWorkspace(t, projectRoot)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	cfg := NewDefaultWorkspaceConfig()
	cfg.Project.CodeDir = "${TEST_CODE_DIR}"
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if loaded.Project.CodeDir != codeDir {
		t.Errorf("Project.CodeDir = %q, want %q (expanded from env)", loaded.Project.CodeDir, codeDir)
	}
}
