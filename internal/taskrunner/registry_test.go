package taskrunner

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/valksor/go-toolkit/eventbus"
)

func TestNewRegistry(t *testing.T) {
	t.Run("with nil bus", func(t *testing.T) {
		r := NewRegistry(nil)
		if r == nil {
			t.Fatal("expected non-nil registry")
		}
		if r.tasks == nil {
			t.Error("tasks map should be initialized")
		}
	})

	t.Run("with bus", func(t *testing.T) {
		bus := eventbus.NewBus()
		r := NewRegistry(bus)
		if r == nil {
			t.Fatal("expected non-nil registry")
		}
		if r.bus != bus {
			t.Error("bus should be set")
		}
	})
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry(nil)

	task := r.Register("file:test.md")

	if task == nil {
		t.Fatal("expected non-nil task")
	}
	if task.ID == "" {
		t.Error("task ID should not be empty")
	}
	if task.Reference != "file:test.md" {
		t.Errorf("expected reference 'file:test.md', got %q", task.Reference)
	}
	if task.Status != RunStatusPending {
		t.Errorf("expected status pending, got %s", task.Status)
	}
	// Note: StartedAt is set in Start(), not Register()
	if !task.StartedAt.IsZero() {
		t.Error("started time should not be set on register (set on start)")
	}
}

func TestRegistry_Start(t *testing.T) {
	r := NewRegistry(nil)
	task := r.Register("file:test.md")

	// Create a mock conductor reference
	mockCond := &mockConductorRef{taskID: "task-123"}
	_, cancel := context.WithCancel(context.Background())

	err := r.Start(task.ID, mockCond, cancel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify task state
	updated := r.Get(task.ID)
	if updated == nil {
		t.Fatal("task should exist")
	}
	if updated.Status != RunStatusRunning {
		t.Errorf("expected status running, got %s", updated.Status)
	}
	if updated.TaskID != "task-123" {
		t.Errorf("expected taskID 'task-123', got %q", updated.TaskID)
	}
}

func TestRegistry_Start_NotFound(t *testing.T) {
	r := NewRegistry(nil)

	err := r.Start("nonexistent", nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestRegistry_Complete(t *testing.T) {
	r := NewRegistry(nil)
	task := r.Register("file:test.md")

	err := r.Complete(task.ID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := r.Get(task.ID)
	if updated.Status != RunStatusCompleted {
		t.Errorf("expected status completed, got %s", updated.Status)
	}
	if updated.FinishedAt.IsZero() {
		t.Error("finished time should be set")
	}
}

func TestRegistry_Complete_WithError(t *testing.T) {
	r := NewRegistry(nil)
	task := r.Register("file:test.md")

	testErr := &mockError{msg: "test error"}
	err := r.Complete(task.ID, testErr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := r.Get(task.ID)
	if updated.Status != RunStatusFailed {
		t.Errorf("expected status failed, got %s", updated.Status)
	}
	if updated.Error == nil || updated.Error.Error() != "test error" {
		t.Errorf("expected error 'test error', got %v", updated.Error)
	}
}

func TestRegistry_Cancel(t *testing.T) {
	r := NewRegistry(nil)
	task := r.Register("file:test.md")

	// Set up cancel func
	cancelled := false
	cancelFunc := func() { cancelled = true }

	mockCond := &mockConductorRef{taskID: "task-123"}
	_ = r.Start(task.ID, mockCond, cancelFunc)

	err := r.Cancel(task.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cancelled {
		t.Error("cancel function should have been called")
	}

	updated := r.Get(task.ID)
	if updated.Status != RunStatusCancelled {
		t.Errorf("expected status cancelled, got %s", updated.Status)
	}
}

func TestRegistry_Cancel_NotRunning(t *testing.T) {
	r := NewRegistry(nil)
	task := r.Register("file:test.md")

	// Complete before cancel
	_ = r.Complete(task.ID, nil)

	err := r.Cancel(task.ID)
	if err == nil {
		t.Error("expected error when cancelling completed task")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry(nil)

	// Non-existent task
	if r.Get("nonexistent") != nil {
		t.Error("expected nil for nonexistent task")
	}

	// Existing task
	task := r.Register("file:test.md")
	got := r.Get(task.ID)
	if got == nil {
		t.Fatal("expected task")
	}
	if got.ID != task.ID {
		t.Errorf("expected ID %q, got %q", task.ID, got.ID)
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry(nil)

	// Empty registry
	list := r.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d tasks", len(list))
	}

	// Add tasks
	r.Register("file:a.md")
	r.Register("file:b.md")
	r.Register("file:c.md")

	list = r.List()
	if len(list) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(list))
	}
}

func TestRegistry_ListRunning(t *testing.T) {
	r := NewRegistry(nil)

	task1 := r.Register("file:a.md")
	task2 := r.Register("file:b.md")
	task3 := r.Register("file:c.md")

	// At this point, all 3 are pending
	// ListRunning includes pending + running status
	running := r.ListRunning()
	if len(running) != 3 {
		t.Errorf("expected 3 pending/running tasks, got %d", len(running))
	}

	// Start task1 and task2
	mockCond := &mockConductorRef{taskID: "t"}
	_, cancel := context.WithCancel(context.Background())
	_ = r.Start(task1.ID, mockCond, cancel)
	_ = r.Start(task2.ID, mockCond, cancel)

	// Complete task2
	_ = r.Complete(task2.ID, nil)

	// Now: task1=running, task2=completed, task3=pending
	// ListRunning should return task1 and task3 (running + pending)
	running = r.ListRunning()
	if len(running) != 2 {
		t.Errorf("expected 2 running/pending tasks, got %d", len(running))
	}

	// Verify task2 (completed) is not in the list
	for _, task := range running {
		if task.ID == task2.ID {
			t.Error("completed task should not be in ListRunning()")
		}
	}

	// Also verify task3 stays pending
	for _, task := range running {
		if task.ID == task3.ID && task.Status != RunStatusPending {
			t.Errorf("task3 should be pending, got %s", task.Status)
		}
	}
}

func TestRegistry_Count(t *testing.T) {
	r := NewRegistry(nil)

	if r.Count() != 0 {
		t.Errorf("expected count 0, got %d", r.Count())
	}

	r.Register("file:a.md")
	r.Register("file:b.md")

	if r.Count() != 2 {
		t.Errorf("expected count 2, got %d", r.Count())
	}
}

func TestRegistry_CountRunning(t *testing.T) {
	r := NewRegistry(nil)

	task1 := r.Register("file:a.md")
	task2 := r.Register("file:b.md")

	if r.CountRunning() != 0 {
		t.Errorf("expected 0 running, got %d", r.CountRunning())
	}

	mockCond := &mockConductorRef{taskID: "t"}
	_, cancel := context.WithCancel(context.Background())
	_ = r.Start(task1.ID, mockCond, cancel)

	if r.CountRunning() != 1 {
		t.Errorf("expected 1 running, got %d", r.CountRunning())
	}

	_ = r.Start(task2.ID, mockCond, cancel)

	if r.CountRunning() != 2 {
		t.Errorf("expected 2 running, got %d", r.CountRunning())
	}
}

func TestRegistry_AddNote(t *testing.T) {
	r := NewRegistry(nil)
	task := r.Register("file:test.md")

	mockCond := &mockConductorRef{taskID: "task-123"}
	_, cancel := context.WithCancel(context.Background())
	_ = r.Start(task.ID, mockCond, cancel)

	ctx := context.Background()
	err := r.AddNote(ctx, task.ID, "test note")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockCond.lastNote != "test note" {
		t.Errorf("expected note 'test note', got %q", mockCond.lastNote)
	}
}

func TestRegistry_AddNote_NotRunning(t *testing.T) {
	r := NewRegistry(nil)
	task := r.Register("file:test.md")

	ctx := context.Background()
	err := r.AddNote(ctx, task.ID, "test note")
	if err == nil {
		t.Error("expected error for non-running task")
	}
}

func TestRegistry_SetWorktreePath(t *testing.T) {
	r := NewRegistry(nil)
	task := r.Register("file:test.md")

	err := r.SetWorktreePath(task.ID, "/path/to/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := r.Get(task.ID)
	if updated.WorktreePath != "/path/to/worktree" {
		t.Errorf("expected worktree path '/path/to/worktree', got %q", updated.WorktreePath)
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry(nil)

	var wg sync.WaitGroup
	const n = 100

	// Concurrent registration
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Register("file:test.md")
		}()
	}

	wg.Wait()

	if r.Count() != n {
		t.Errorf("expected %d tasks, got %d", n, r.Count())
	}
}

func TestRunningTask_Duration(t *testing.T) {
	task := &RunningTask{
		StartedAt: time.Now().Add(-5 * time.Second),
	}

	// Running task (no finish time)
	dur := task.Duration()
	if dur < 5*time.Second {
		t.Errorf("expected duration >= 5s, got %v", dur)
	}

	// Completed task
	task.FinishedAt = task.StartedAt.Add(3 * time.Second)
	dur = task.Duration()
	if dur != 3*time.Second {
		t.Errorf("expected duration 3s, got %v", dur)
	}
}

func TestRunStatus_Constants(t *testing.T) {
	tests := []struct {
		status RunStatus
		want   string
	}{
		{RunStatusPending, "pending"},
		{RunStatusRunning, "running"},
		{RunStatusCompleted, "completed"},
		{RunStatusFailed, "failed"},
		{RunStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("expected %q, got %q", tt.want, string(tt.status))
		}
	}
}

// Mock types for testing.

type mockConductorRef struct {
	taskID   string
	lastNote string
}

func (m *mockConductorRef) GetTaskID() string {
	return m.taskID
}

func (m *mockConductorRef) AddNote(ctx context.Context, message string) error {
	m.lastNote = message

	return nil
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
