package conductor

import (
	"context"
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
)

// openResetTestWorkspace creates a test workspace with a temporary home directory.
func openResetTestWorkspace(tb testing.TB, repoRoot string) *storage.Workspace {
	tb.Helper()
	homeDir := tb.TempDir()
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir
	ws, err := storage.OpenWorkspace(context.Background(), repoRoot, cfg)
	if err != nil {
		tb.Fatalf("OpenWorkspace: %v", err)
	}

	// Initialize workspace directories
	if err := ws.EnsureInitialized(); err != nil {
		tb.Fatalf("EnsureInitialized: %v", err)
	}

	return ws
}

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

func TestResetState_WithActiveTask(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace with test home directory
	ws := openResetTestWorkspace(t, tmpDir)

	// Create conductor with workspace and machine
	eventBus := eventbus.NewBus()
	c := &Conductor{
		workspace: ws,
		machine:   workflow.NewMachine(eventBus),
		eventBus:  eventBus,
	}

	// Set up an active task in a non-idle state
	c.activeTask = &storage.ActiveTask{
		ID:    "test-task",
		State: "planning", // Should be reset to "idle"
	}

	// Reset state
	err := c.ResetState(context.Background())
	if err != nil {
		t.Fatalf("ResetState() error = %v", err)
	}

	// Verify state was reset to idle
	if c.activeTask.State != "idle" {
		t.Errorf("activeTask.State = %q, want %q", c.activeTask.State, "idle")
	}
}

func TestResetState_FromDifferentStates(t *testing.T) {
	tests := []struct {
		name         string
		initialState string
	}{
		{"from planning", "planning"},
		{"from implementing", "implementing"},
		{"from reviewing", "reviewing"},
		{"from waiting", "waiting"},
		{"from failed", "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			ws := openResetTestWorkspace(t, tmpDir)

			eventBus := eventbus.NewBus()
			c := &Conductor{
				workspace: ws,
				machine:   workflow.NewMachine(eventBus),
				eventBus:  eventBus,
			}

			c.activeTask = &storage.ActiveTask{
				ID:    "test-task",
				State: tt.initialState,
			}

			err := c.ResetState(context.Background())
			if err != nil {
				t.Fatalf("ResetState() error = %v", err)
			}

			if c.activeTask.State != "idle" {
				t.Errorf("from %q: activeTask.State = %q, want %q",
					tt.initialState, c.activeTask.State, "idle")
			}
		})
	}
}

func TestResetState_MachineReset(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openResetTestWorkspace(t, tmpDir)

	eventBus := eventbus.NewBus()
	machine := workflow.NewMachine(eventBus)
	c := &Conductor{
		workspace: ws,
		machine:   machine,
		eventBus:  eventBus,
	}

	// Set active task
	c.activeTask = &storage.ActiveTask{
		ID:    "test-task",
		State: "planning",
	}

	err := c.ResetState(context.Background())
	if err != nil {
		t.Fatalf("ResetState() error = %v", err)
	}

	// Verify machine state is idle after reset
	if machine.State() != workflow.StateIdle {
		t.Errorf("machine state = %q, want %q", machine.State(), workflow.StateIdle)
	}
}
