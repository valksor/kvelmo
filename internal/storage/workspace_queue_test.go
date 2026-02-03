package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanOutputPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	path := ws.PlanOutputPath("my-queue")

	// Should end with queues/my-queue/plan.md
	expected := filepath.Join("queues", "my-queue", "plan.md")
	if !filepath.IsAbs(path) {
		t.Errorf("PlanOutputPath() = %q, want absolute path", path)
	}
	if !pathEndsWith(path, expected) {
		t.Errorf("PlanOutputPath() = %q, want path ending with %q", path, expected)
	}
}

func TestSavePlanOutput(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	queueID := "test-queue-123"
	content := "## Tasks\n\n### task-1: Test task\n- **Priority**: 1\n- **Status**: ready"

	err := ws.SavePlanOutput(queueID, content)
	if err != nil {
		t.Fatalf("SavePlanOutput() error = %v", err)
	}

	// Verify file was created with correct content
	planPath := ws.PlanOutputPath(queueID)
	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", planPath, err)
	}

	if string(data) != content {
		t.Errorf("SavePlanOutput() content = %q, want %q", string(data), content)
	}
}

func TestSavePlanOutput_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	queueID := "new-queue-456"
	content := "Test content"

	// Directory shouldn't exist yet
	queueDir := filepath.Dir(ws.PlanOutputPath(queueID))
	if _, err := os.Stat(queueDir); !os.IsNotExist(err) {
		t.Fatalf("Queue directory already exists: %s", queueDir)
	}

	err := ws.SavePlanOutput(queueID, content)
	if err != nil {
		t.Fatalf("SavePlanOutput() error = %v", err)
	}

	// Directory should now exist
	if _, err := os.Stat(queueDir); os.IsNotExist(err) {
		t.Errorf("Queue directory was not created: %s", queueDir)
	}
}

func TestSavePlanOutput_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	queueID := "empty-queue"
	content := ""

	err := ws.SavePlanOutput(queueID, content)
	if err != nil {
		t.Fatalf("SavePlanOutput() error = %v", err)
	}

	// Verify file was created (even with empty content)
	planPath := ws.PlanOutputPath(queueID)
	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", planPath, err)
	}

	if string(data) != "" {
		t.Errorf("SavePlanOutput() content = %q, want empty", string(data))
	}
}

// pathEndsWith checks if path ends with the expected suffix.
func pathEndsWith(path, expected string) bool {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(path)
	normalizedExpected := filepath.ToSlash(expected)

	return len(normalizedPath) >= len(normalizedExpected) &&
		normalizedPath[len(normalizedPath)-len(normalizedExpected):] == normalizedExpected
}
