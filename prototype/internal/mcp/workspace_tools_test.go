package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/paths"
)

func TestResolveWorkspaceRoot(t *testing.T) {
	ctx := context.Background()

	// Test in current directory
	root, err := resolveWorkspaceRoot(ctx)
	if err != nil {
		t.Fatalf("resolveWorkspaceRoot failed: %v", err)
	}

	if root == "" {
		t.Error("Got empty root path")
	}

	// Check if root is absolute path
	if !filepath.IsAbs(root) {
		t.Errorf("Root is not absolute path: got %s", root)
	}
}

func TestWorkspaceToolsRegistration(t *testing.T) {
	registry := NewToolRegistry(nil)
	RegisterWorkspaceTools(registry)

	tools := registry.ListTools()

	// Check that workspace tools are registered
	expectedTools := []string{
		"workspace_get_active_task",
		"workspace_list_tasks",
		"workspace_get_specs",
		"workspace_get_sessions",
		"workspace_get_notes",
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Tool %s not registered", expected)
		}
	}
}

func TestWorkspaceGetActiveTask(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()
	ctx := context.Background()

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = t.TempDir()
	_, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	if err != nil {
		t.Fatalf("Failed to open workspace: %v", err)
	}

	// Test with no active task
	registry := NewToolRegistry(nil)
	RegisterWorkspaceTools(registry)

	// Change to temp directory
	t.Chdir(tmpDir)

	result, err := registry.CallTool(ctx, "workspace_get_active_task", map[string]interface{}{})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result.IsError {
		t.Errorf("Tool returned error: %s", result.Content[0].Text)
	}

	// Should indicate no active task
	if result.Content[0].Text == "" {
		t.Error("Expected non-empty response about no active task")
	}
}

func TestWorkspaceListTasks(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = t.TempDir()
	_, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	if err != nil {
		t.Fatalf("Failed to open workspace: %v", err)
	}

	registry := NewToolRegistry(nil)
	RegisterWorkspaceTools(registry)

	// Change to temp directory
	t.Chdir(tmpDir)

	result, err := registry.CallTool(ctx, "workspace_list_tasks", map[string]interface{}{})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result.IsError {
		t.Errorf("Tool returned error: %s", result.Content[0].Text)
	}

	// Should return empty task list
	if result.Content[0].Text == "" {
		t.Error("Expected non-empty response")
	}
}

func TestWorkspaceGetNotes(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	ctx := context.Background()

	// Set global home dir override so MCP tools find the same workspace
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = homeDir
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	if err != nil {
		t.Fatalf("Failed to open workspace: %v", err)
	}

	// Create a task with notes
	taskID := "test-task-123"
	workDir := ws.WorkRoot()
	taskWorkDir := filepath.Join(workDir, taskID)

	if err := os.MkdirAll(taskWorkDir, 0o755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	notesFile := filepath.Join(taskWorkDir, "notes.md")
	notesContent := "# Test Notes\n\nSome note content"
	if err := os.WriteFile(notesFile, []byte(notesContent), 0o644); err != nil {
		t.Fatalf("Failed to write notes: %v", err)
	}

	registry := NewToolRegistry(nil)
	RegisterWorkspaceTools(registry)

	// Change to temp directory
	t.Chdir(tmpDir)

	result, err := registry.CallTool(ctx, "workspace_get_notes", map[string]interface{}{
		"task_id": taskID,
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result.IsError {
		t.Errorf("Tool returned error: %s", result.Content[0].Text)
	}

	// Check if notes content is in the result
	resultText := result.Content[0].Text
	if resultText == "" {
		t.Error("Expected non-empty notes content")
	}
}
