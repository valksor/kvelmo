package log

import (
	"testing"
)

func TestTaskID(t *testing.T) {
	attr := TaskID("task-123")
	if attr.Key != "task_id" {
		t.Errorf("TaskID().Key = %q, want %q", attr.Key, "task_id")
	}
	if attr.Value.String() != "task-123" {
		t.Errorf("TaskID().Value = %q, want %q", attr.Value.String(), "task-123")
	}
}

func TestState(t *testing.T) {
	attrs := State("idle", "planning")
	if len(attrs) != 2 {
		t.Errorf("State() returned %d attrs, want 2", len(attrs))
	}

	foundFrom, foundTo := false, false
	for _, attr := range attrs {
		if attr.Key == "from" && attr.Value.String() == "idle" {
			foundFrom = true
		}
		if attr.Key == "to" && attr.Value.String() == "planning" {
			foundTo = true
		}
	}

	if !foundFrom {
		t.Error("State() missing 'from' attribute")
	}
	if !foundTo {
		t.Error("State() missing 'to' attribute")
	}
}
