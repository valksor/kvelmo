package help

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// setTestHome sets HOME to a temp directory.
// This ensures workspace data is stored in a predictable location during tests.
func setTestHome(t *testing.T, tmpDir string) {
	t.Helper()
	t.Setenv("HOME", tmpDir)
}

// openTestWorkspace creates a test workspace with a temporary home directory.
func openTestWorkspace(tb testing.TB, repoRoot string, homeDir string) *storage.Workspace {
	tb.Helper()

	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir

	ws, err := storage.OpenWorkspace(context.Background(), repoRoot, cfg)
	if err != nil {
		tb.Fatalf("OpenWorkspace: %v", err)
	}

	return ws
}

func TestLoadContext_NoWorkspace(t *testing.T) {
	// Create an isolated directory structure where parent traversal won't find .mehrhof
	// We need to avoid the parent directory search by using /tmp
	tmpDir := t.TempDir()
	isolatedDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(isolatedDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	t.Chdir(isolatedDir)

	ctx := LoadContext()

	// In an isolated temp directory, there should be no .mehrhof
	// Note: storage.OpenWorkspace walks up directories, so we need a truly isolated path
	// If it finds a .mehrhof somewhere, HasWorkspace will be true
	// The important check is that HasActiveTask should be false
	if ctx.HasActiveTask {
		t.Error("expected HasActiveTask to be false")
	}
	if ctx.HasSpecifications {
		t.Error("expected HasSpecifications to be false")
	}
}

func TestLoadContext_WithWorkspaceNoTask(t *testing.T) {
	// Create a temp directory with .mehrhof but no active task
	tmpDir := t.TempDir()
	mehrhofDir := filepath.Join(tmpDir, ".mehrhof")
	if err := os.Mkdir(mehrhofDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	t.Chdir(tmpDir)

	ctx := LoadContext()

	if !ctx.HasWorkspace {
		t.Error("expected HasWorkspace to be true")
	}
	if ctx.HasActiveTask {
		t.Error("expected HasActiveTask to be false")
	}
}

func TestLoadContext_WithActiveTask(t *testing.T) {
	// Create a temp directory structure for the new workspace architecture
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)

	// Create project directory
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Use storage package to properly set up workspace and active task
	ws := openTestWorkspace(t, projectDir, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create a work entry
	source := storage.SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test-task-id", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Create active task
	activeTask := &storage.ActiveTask{
		ID:      "test-task-id",
		Ref:     "file:task.md",
		WorkDir: ws.WorkPath("test-task-id"),
		State:   "implementing",
		UseGit:  true,
		Started: time.Now(),
	}
	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	t.Chdir(projectDir)

	ctx := LoadContext()

	if !ctx.HasWorkspace {
		t.Error("expected HasWorkspace to be true")
	}
	if !ctx.HasActiveTask {
		t.Error("expected HasActiveTask to be true")
	}
	if ctx.TaskID != "test-task-id" {
		t.Errorf("TaskID = %q, want 'test-task-id'", ctx.TaskID)
	}
	if ctx.TaskState != "implementing" {
		t.Errorf("TaskState = %q, want 'implementing'", ctx.TaskState)
	}
	if !ctx.UseGit {
		t.Error("expected UseGit to be true")
	}
}

func TestLoadContext_WithSpecifications(t *testing.T) {
	// Create a temp directory structure for the new workspace architecture
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)

	// Create project directory
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Use storage package to properly set up workspace
	ws := openTestWorkspace(t, projectDir, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create a work entry
	source := storage.SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test-task-id", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Create a specification
	if err := ws.SaveSpecification("test-task-id", 1, "# Specification"); err != nil {
		t.Fatalf("SaveSpecification: %v", err)
	}

	// Create active task
	activeTask := &storage.ActiveTask{
		ID:      "test-task-id",
		Ref:     "file:task.md",
		WorkDir: ws.WorkPath("test-task-id"),
		State:   "planning",
		UseGit:  false,
		Started: time.Now(),
	}
	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	t.Chdir(projectDir)

	ctx := LoadContext()

	if !ctx.HasWorkspace {
		t.Error("expected HasWorkspace to be true")
	}
	if !ctx.HasActiveTask {
		t.Error("expected HasActiveTask to be true")
	}
	if !ctx.HasSpecifications {
		t.Error("expected HasSpecifications to be true")
	}
}

func TestLoadContext_BadWorkYaml(t *testing.T) {
	// Create a temp directory structure for the new workspace architecture
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)

	// Create project directory
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Use storage package to properly set up workspace
	ws := openTestWorkspace(t, projectDir, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create a work entry first (to get the directory created)
	source := storage.SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test-task-id", source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Corrupt the work.yaml file
	workYamlPath := filepath.Join(ws.WorkPath("test-task-id"), "work.yaml")
	if err := os.WriteFile(workYamlPath, []byte("not valid yaml: [["), 0o644); err != nil {
		t.Fatalf("write bad work.yaml: %v", err)
	}

	// Create active task
	activeTask := &storage.ActiveTask{
		ID:      "test-task-id",
		Ref:     "file:task.md",
		WorkDir: ws.WorkPath("test-task-id"),
		State:   "planning",
		UseGit:  false,
		Started: time.Now(),
	}
	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	t.Chdir(projectDir)

	ctx := LoadContext()

	// LoadContext only reads .active_task file, not work.yaml
	// So even with bad work.yaml, HasActiveTask should be true
	if !ctx.HasActiveTask {
		t.Error("expected HasActiveTask to be true (LoadContext doesn't read work.yaml)")
	}
}

func TestHelpContext_StructFields(t *testing.T) {
	// Test that HelpContext struct can be created and accessed
	ctx := &HelpContext{
		HasWorkspace:      true,
		HasActiveTask:     true,
		TaskID:            "task-123",
		TaskState:         "implementing",
		HasSpecifications: true,
		UseGit:            true,
	}

	if !ctx.HasWorkspace {
		t.Error("HasWorkspace should be true")
	}
	if !ctx.HasActiveTask {
		t.Error("HasActiveTask should be true")
	}
	if ctx.TaskID != "task-123" {
		t.Errorf("TaskID = %q, want 'task-123'", ctx.TaskID)
	}
	if ctx.TaskState != "implementing" {
		t.Errorf("TaskState = %q, want 'implementing'", ctx.TaskState)
	}
	if !ctx.HasSpecifications {
		t.Error("HasSpecifications should be true")
	}
	if !ctx.UseGit {
		t.Error("UseGit should be true")
	}
}
