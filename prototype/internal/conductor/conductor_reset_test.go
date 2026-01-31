package conductor

import (
	"context"
	"testing"
)

func TestResetState_NoActiveTask(t *testing.T) {
	// Create a minimal conductor without an active task
	c := &Conductor{}

	err := c.ResetState(context.Background())
	if err == nil {
		t.Error("ResetState() should return error when no active task")
	}
	if err.Error() != "no active task" {
		t.Errorf("ResetState() error = %q, want %q", err.Error(), "no active task")
	}
}
