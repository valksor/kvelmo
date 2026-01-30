package taskrunner

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valksor/go-toolkit/eventbus"
)

func TestNewRunner(t *testing.T) {
	t.Run("default parallelism", func(t *testing.T) {
		r := NewRunner(nil, 0, nil)
		if r.maxParallel != 1 {
			t.Errorf("expected maxParallel 1 (default), got %d", r.maxParallel)
		}
	})

	t.Run("custom parallelism", func(t *testing.T) {
		r := NewRunner(nil, 5, nil)
		if r.maxParallel != 5 {
			t.Errorf("expected maxParallel 5, got %d", r.maxParallel)
		}
	})

	t.Run("with registry and bus", func(t *testing.T) {
		reg := NewRegistry(nil)
		bus := eventbus.NewBus()
		r := NewRunner(reg, 3, bus)

		if r.registry != reg {
			t.Error("registry not set")
		}
		if r.bus != bus {
			t.Error("bus not set")
		}
	})
}

func TestRunner_Run_NoFactory(t *testing.T) {
	reg := NewRegistry(nil)
	runner := NewRunner(reg, 2, nil)

	opts := RunOptions{
		// No ConductorFactory set
	}

	_, err := runner.Run(context.Background(), []string{"file:a.md"}, opts)
	if err == nil {
		t.Error("expected error when ConductorFactory is nil")
	}
	if !errors.Is(err, errors.New("ConductorFactory is required")) && err.Error() != "ConductorFactory is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunner_Run_Empty(t *testing.T) {
	reg := NewRegistry(nil)
	runner := NewRunner(reg, 2, nil)

	results, err := runner.Run(context.Background(), []string{}, RunOptions{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestRunner_Run_SingleTask(t *testing.T) {
	reg := NewRegistry(nil)
	runner := NewRunner(reg, 2, nil)

	factory := &mockConductorFactory{
		conductors: make(map[string]*mockTaskConductor),
	}

	opts := RunOptions{
		ConductorFactory: factory,
	}

	results, err := runner.Run(context.Background(), []string{"file:test.md"}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Error != nil {
		t.Errorf("unexpected task error: %v", results[0].Error)
	}
	if results[0].Reference != "file:test.md" {
		t.Errorf("expected reference 'file:test.md', got %q", results[0].Reference)
	}
}

func TestRunner_Run_MultipleTasks(t *testing.T) {
	reg := NewRegistry(nil)
	runner := NewRunner(reg, 3, nil)

	factory := &mockConductorFactory{
		conductors: make(map[string]*mockTaskConductor),
	}

	refs := []string{"file:a.md", "file:b.md", "file:c.md"}
	opts := RunOptions{
		ConductorFactory: factory,
	}

	results, err := runner.Run(context.Background(), refs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify all conductors were called
	factory.mu.Lock()
	if len(factory.conductors) != 3 {
		t.Errorf("expected 3 conductors created, got %d", len(factory.conductors))
	}
	factory.mu.Unlock()
}

func TestRunner_Run_Parallelism(t *testing.T) {
	reg := NewRegistry(nil)
	runner := NewRunner(reg, 2, nil) // Only 2 parallel

	var activeCount atomic.Int32
	var maxActive atomic.Int32

	factory := &mockConductorFactory{
		conductors: make(map[string]*mockTaskConductor),
		beforeRun: func() {
			current := activeCount.Add(1)
			for {
				oldMax := maxActive.Load()
				if current <= oldMax {
					break
				}
				if maxActive.CompareAndSwap(oldMax, current) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond) // Simulate work
		},
		afterRun: func() {
			activeCount.Add(-1)
		},
	}

	refs := []string{"file:a.md", "file:b.md", "file:c.md", "file:d.md"}
	opts := RunOptions{
		ConductorFactory: factory,
	}

	_, err := runner.Run(context.Background(), refs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// MaxActive should not exceed 2 (our parallelism limit)
	if maxActive.Load() > 2 {
		t.Errorf("expected max parallelism <= 2, got %d", maxActive.Load())
	}
}

func TestRunner_Run_Timeout(t *testing.T) {
	reg := NewRegistry(nil)
	runner := NewRunner(reg, 1, nil)

	factory := &mockConductorFactory{
		conductors: make(map[string]*mockTaskConductor),
		delay:      500 * time.Millisecond, // Longer than timeout
	}

	opts := RunOptions{
		ConductorFactory: factory,
		Timeout:          100 * time.Millisecond,
	}

	results, err := runner.Run(context.Background(), []string{"file:test.md"}, opts)
	if err == nil {
		t.Error("expected timeout error")
	}

	// Task should have context error
	if len(results) > 0 && results[0].Error == nil {
		t.Error("expected task to have error due to timeout")
	}
}

func TestRunner_Run_StopOnError(t *testing.T) {
	reg := NewRegistry(nil)
	runner := NewRunner(reg, 1, nil) // Sequential to control order

	var tasksStarted atomic.Int32

	factory := &mockConductorFactory{
		conductors: make(map[string]*mockTaskConductor),
		beforeRun: func() {
			tasksStarted.Add(1)
		},
		failRef: "file:a.md", // First task will fail
	}

	refs := []string{"file:a.md", "file:b.md", "file:c.md"}
	opts := RunOptions{
		ConductorFactory: factory,
		StopOnError:      true,
	}

	_, err := runner.Run(context.Background(), refs, opts)
	if err == nil {
		t.Error("expected error due to task failure")
	}

	// With sequential execution and StopOnError, later tasks might not start
	// (depending on timing, some may have already been queued)
}

func TestRunner_Run_Callbacks(t *testing.T) {
	reg := NewRegistry(nil)
	runner := NewRunner(reg, 2, nil)

	factory := &mockConductorFactory{
		conductors: make(map[string]*mockTaskConductor),
	}

	var startedRefs []string
	var completedRefs []string
	var mu sync.Mutex

	refs := []string{"file:a.md", "file:b.md"}
	opts := RunOptions{
		ConductorFactory: factory,
		OnTaskStart: func(runningID, ref string) {
			mu.Lock()
			startedRefs = append(startedRefs, ref)
			mu.Unlock()
		},
		OnTaskComplete: func(result TaskResult) {
			mu.Lock()
			completedRefs = append(completedRefs, result.Reference)
			mu.Unlock()
		},
	}

	_, err := runner.Run(context.Background(), refs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(startedRefs) != 2 {
		t.Errorf("expected 2 started callbacks, got %d", len(startedRefs))
	}
	if len(completedRefs) != 2 {
		t.Errorf("expected 2 completed callbacks, got %d", len(completedRefs))
	}
}

func TestRunner_CancelAll(t *testing.T) {
	reg := NewRegistry(nil)
	runner := NewRunner(reg, 3, nil)

	// Register and start some tasks
	task1 := reg.Register("file:a.md")
	task2 := reg.Register("file:b.md")
	task3 := reg.Register("file:c.md")

	var cancel1Called, cancel2Called bool
	cancel1 := func() { cancel1Called = true }
	cancel2 := func() { cancel2Called = true }

	mock := &mockConductorRef{taskID: "t"}
	_ = reg.Start(task1.ID, mock, cancel1)
	_ = reg.Start(task2.ID, mock, cancel2)
	_ = reg.Complete(task3.ID, nil) // Already completed

	count := runner.CancelAll()
	if count != 2 {
		t.Errorf("expected 2 cancelled, got %d", count)
	}

	if !cancel1Called || !cancel2Called {
		t.Error("cancel functions should have been called")
	}
}

func TestRunner_GetRegistry(t *testing.T) {
	reg := NewRegistry(nil)
	runner := NewRunner(reg, 2, nil)

	if runner.GetRegistry() != reg {
		t.Error("GetRegistry should return the registry")
	}
}

func TestRunner_SetMaxParallel(t *testing.T) {
	runner := NewRunner(nil, 2, nil)

	runner.SetMaxParallel(5)
	if runner.maxParallel != 5 {
		t.Errorf("expected maxParallel 5, got %d", runner.maxParallel)
	}

	// Invalid value should be ignored
	runner.SetMaxParallel(0)
	if runner.maxParallel != 5 {
		t.Errorf("expected maxParallel to remain 5, got %d", runner.maxParallel)
	}
}

func TestTaskResult_Fields(t *testing.T) {
	result := TaskResult{
		RunningTaskID: "run-123",
		Reference:     "file:test.md",
		TaskID:        "task-456",
		Error:         nil,
		Duration:      5 * time.Second,
		WorktreePath:  "/path/to/worktree",
	}

	if result.RunningTaskID != "run-123" {
		t.Errorf("unexpected RunningTaskID: %s", result.RunningTaskID)
	}
	if result.Reference != "file:test.md" {
		t.Errorf("unexpected Reference: %s", result.Reference)
	}
	if result.TaskID != "task-456" {
		t.Errorf("unexpected TaskID: %s", result.TaskID)
	}
	if result.Duration != 5*time.Second {
		t.Errorf("unexpected Duration: %v", result.Duration)
	}
	if result.WorktreePath != "/path/to/worktree" {
		t.Errorf("unexpected WorktreePath: %s", result.WorktreePath)
	}
}

// Mock types for testing.

type mockConductorFactory struct {
	mu         sync.Mutex
	conductors map[string]*mockTaskConductor
	failRef    string        // Reference that should fail
	delay      time.Duration // Delay to simulate work
	beforeRun  func()        // Called before running
	afterRun   func()        // Called after running
}

func (f *mockConductorFactory) Create(ctx context.Context, ref string, worktree bool) (TaskConductor, error) {
	cond := &mockTaskConductor{
		factory:      f,
		ref:          ref,
		shouldFail:   ref == f.failRef,
		delay:        f.delay,
		worktreePath: "",
	}
	if worktree {
		cond.worktreePath = "/worktrees/" + ref
	}

	f.mu.Lock()
	f.conductors[ref] = cond
	f.mu.Unlock()

	return cond, nil
}

type mockTaskConductor struct {
	factory      *mockConductorFactory
	ref          string
	taskID       string
	shouldFail   bool
	delay        time.Duration
	worktreePath string
	closed       bool
}

func (c *mockTaskConductor) GetTaskID() string {
	return c.taskID
}

func (c *mockTaskConductor) AddNote(ctx context.Context, message string) error {
	return nil
}

func (c *mockTaskConductor) Start(ctx context.Context, ref string) error {
	c.taskID = "task-" + ref

	return nil
}

func (c *mockTaskConductor) Plan(ctx context.Context) error {
	if c.factory.beforeRun != nil {
		c.factory.beforeRun()
	}

	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			if c.factory.afterRun != nil {
				c.factory.afterRun()
			}

			return ctx.Err()
		}
	}

	if c.shouldFail {
		if c.factory.afterRun != nil {
			c.factory.afterRun()
		}

		return errors.New("mock failure")
	}

	return nil
}

func (c *mockTaskConductor) Implement(ctx context.Context) error {
	if c.factory.afterRun != nil {
		defer c.factory.afterRun()
	}

	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (c *mockTaskConductor) GetWorktreePath() string {
	return c.worktreePath
}

func (c *mockTaskConductor) Close() error {
	c.closed = true

	return nil
}
