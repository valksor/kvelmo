package storage

import (
	"testing"
)

func TestAddLabel(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	taskID := "test-task-001"
	// CreateWork sets up the directory structure
	_, err := ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	if err != nil {
		t.Fatal(err)
	}

	if err := ws.AddLabel(taskID, "priority:high"); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}

	updated, err := ws.LoadWork(taskID)
	if err != nil {
		t.Fatal(err)
	}

	if len(updated.Metadata.Labels) != 1 {
		t.Errorf("expected 1 label, got %d", len(updated.Metadata.Labels))
	}
	if updated.Metadata.Labels[0] != "priority:high" {
		t.Errorf("expected priority:high, got %s", updated.Metadata.Labels[0])
	}
}

func TestAddLabel_Duplicate(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	taskID := "test-task-dup"
	_, err := ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	if err != nil {
		t.Fatal(err)
	}
	// Add initial label
	if err := ws.AddLabel(taskID, "priority:high"); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}

	// Adding duplicate should not error
	if err := ws.AddLabel(taskID, "priority:high"); err != nil {
		t.Fatalf("AddLabel duplicate failed: %v", err)
	}

	updated, _ := ws.LoadWork(taskID)
	if len(updated.Metadata.Labels) != 1 {
		t.Errorf("expected 1 label (no duplicate), got %d", len(updated.Metadata.Labels))
	}
}

func TestRemoveLabel(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	taskID := "test-task-002"
	_, err := ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	if err != nil {
		t.Fatal(err)
	}
	// Add initial labels
	if err := ws.AddLabel(taskID, "priority:high"); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}
	if err := ws.AddLabel(taskID, "type:bug"); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}

	if err := ws.RemoveLabel(taskID, "priority:high"); err != nil {
		t.Fatalf("RemoveLabel failed: %v", err)
	}

	updated, _ := ws.LoadWork(taskID)
	if len(updated.Metadata.Labels) != 1 {
		t.Errorf("expected 1 label after removal, got %d", len(updated.Metadata.Labels))
	}
	if updated.Metadata.Labels[0] != "type:bug" {
		t.Errorf("expected type:bug, got %s", updated.Metadata.Labels[0])
	}
}

func TestRemoveLabel_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	taskID := "test-task-remove-non"
	_, err := ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	if err != nil {
		t.Fatal(err)
	}
	// Add initial label
	if err := ws.AddLabel(taskID, "priority:high"); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}

	// Removing non-existent label should not error
	if err := ws.RemoveLabel(taskID, "type:bug"); err != nil {
		t.Fatalf("RemoveLabel non-existent failed: %v", err)
	}

	updated, _ := ws.LoadWork(taskID)
	if len(updated.Metadata.Labels) != 1 {
		t.Errorf("expected 1 label (unchanged), got %d", len(updated.Metadata.Labels))
	}
}

func TestSetLabels(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	taskID := "test-task-003"
	_, err := ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	if err != nil {
		t.Fatal(err)
	}

	newLabels := []string{"priority:high", "team:backend"}
	if err := ws.SetLabels(taskID, newLabels); err != nil {
		t.Fatalf("SetLabels failed: %v", err)
	}

	updated, _ := ws.LoadWork(taskID)
	if len(updated.Metadata.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(updated.Metadata.Labels))
	}
}

func TestSetLabels_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	taskID := "test-task-clear"
	_, err := ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	if err != nil {
		t.Fatal(err)
	}
	// Add initial labels
	if err := ws.AddLabel(taskID, "priority:high"); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}
	if err := ws.AddLabel(taskID, "type:bug"); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}

	// Clear all labels
	if err := ws.SetLabels(taskID, []string{}); err != nil {
		t.Fatalf("SetLabels clear failed: %v", err)
	}

	updated, _ := ws.LoadWork(taskID)
	if len(updated.Metadata.Labels) != 0 {
		t.Errorf("expected 0 labels (cleared), got %d", len(updated.Metadata.Labels))
	}
}

func TestGetLabels(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	taskID := "test-task-get"
	_, err := ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	if err != nil {
		t.Fatal(err)
	}
	// Add labels
	if err := ws.AddLabel(taskID, "priority:high"); err != nil {
		t.Fatal(err)
	}
	if err := ws.AddLabel(taskID, "team:backend"); err != nil {
		t.Fatal(err)
	}

	labels, err := ws.GetLabels(taskID)
	if err != nil {
		t.Fatalf("GetLabels failed: %v", err)
	}

	if len(labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(labels))
	}
}

func TestGetLabels_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	taskID := "test-task-empty"
	_, err := ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	if err != nil {
		t.Fatal(err)
	}

	labels, err := ws.GetLabels(taskID)
	if err != nil {
		t.Fatalf("GetLabels empty failed: %v", err)
	}

	if len(labels) != 0 {
		t.Errorf("expected 0 labels, got %d", len(labels))
	}
}
