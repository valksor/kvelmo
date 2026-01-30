// Package taskrunner provides in-memory tracking and parallel execution of multiple tasks.
//
// The Registry maintains state for running tasks across goroutines, while the Runner
// coordinates parallel task execution with worker pools and graceful cancellation.
//
// Key features:
//   - Thread-safe task registration and status updates
//   - Per-task conductor instances for isolated execution
//   - Event-driven status notifications via eventbus
//   - Graceful cancellation support via context
package taskrunner

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/valksor/go-toolkit/eventbus"
)

// RunStatus represents the execution status of a running task.
type RunStatus string

const (
	// RunStatusPending indicates the task is queued but not yet started.
	RunStatusPending RunStatus = "pending"
	// RunStatusRunning indicates the task is actively executing.
	RunStatusRunning RunStatus = "running"
	// RunStatusCompleted indicates the task finished successfully.
	RunStatusCompleted RunStatus = "completed"
	// RunStatusFailed indicates the task encountered an error.
	RunStatusFailed RunStatus = "failed"
	// RunStatusCancelled indicates the task was cancelled by user or timeout.
	RunStatusCancelled RunStatus = "cancelled"
)

// Event types for task runner notifications.
const (
	TypeTaskRegistered   eventbus.Type = "task_registered"
	TypeTaskStarted      eventbus.Type = "task_started"
	TypeTaskCompleted    eventbus.Type = "task_completed"
	TypeTaskFailed       eventbus.Type = "task_failed"
	TypeTaskCancelled    eventbus.Type = "task_cancelled"
	TypeTaskProgressNote eventbus.Type = "task_progress_note"
)

// ConductorRef is an interface for conductor operations needed by the registry.
// This avoids circular imports with the conductor package.
type ConductorRef interface {
	// AddNote adds a note to the task's notes.md file.
	AddNote(ctx context.Context, message string) error
	// GetTaskID returns the active task ID.
	GetTaskID() string
}

// RunningTask represents a task currently executing in a goroutine.
type RunningTask struct {
	// ID is a short unique identifier for the running task (e.g., "abc123").
	ID string

	// Reference is the task reference string (e.g., "file:a.md", "github:123").
	Reference string

	// Status is the current execution status.
	Status RunStatus

	// StartedAt is when the task began execution.
	StartedAt time.Time

	// FinishedAt is when the task completed (zero if still running).
	FinishedAt time.Time

	// Conductor is the task's conductor instance for communication.
	// This allows sending notes/messages to specific running tasks.
	Conductor ConductorRef

	// Cancel is the cancellation function for this task's context.
	Cancel context.CancelFunc

	// Error holds the error if the task failed.
	Error error

	// WorktreePath is the path to the task's worktree (if using worktrees).
	WorktreePath string

	// TaskID is the mehrhof task ID (different from the running task ID).
	TaskID string
}

// Duration returns the elapsed time for this task.
func (t *RunningTask) Duration() time.Duration {
	if t.FinishedAt.IsZero() {
		return time.Since(t.StartedAt)
	}

	return t.FinishedAt.Sub(t.StartedAt)
}

// IsTerminal returns true if the task has reached a terminal state.
func (t *RunningTask) IsTerminal() bool {
	return t.Status == RunStatusCompleted ||
		t.Status == RunStatusFailed ||
		t.Status == RunStatusCancelled
}

// Registry tracks running tasks across goroutines.
// It is thread-safe and can be accessed from multiple goroutines.
type Registry struct {
	mu    sync.RWMutex
	tasks map[string]*RunningTask
	bus   *eventbus.Bus
}

// NewRegistry creates a new task registry.
// If bus is nil, events will not be published.
func NewRegistry(bus *eventbus.Bus) *Registry {
	return &Registry{
		tasks: make(map[string]*RunningTask),
		bus:   bus,
	}
}

// Register creates a new running task entry in pending state.
// Returns the RunningTask with a generated ID.
func (r *Registry) Register(ref string) *RunningTask {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.generateID()
	task := &RunningTask{
		ID:        id,
		Reference: ref,
		Status:    RunStatusPending,
	}

	r.tasks[id] = task

	// Publish event
	if r.bus != nil {
		r.bus.PublishRaw(eventbus.Event{
			Type:      TypeTaskRegistered,
			Timestamp: time.Now(),
			Data: map[string]any{
				"running_task_id": id,
				"reference":       ref,
			},
		})
	}

	return task
}

// Start marks a task as running with the given conductor and cancel function.
func (r *Registry) Start(id string, cond ConductorRef, cancel context.CancelFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return fmt.Errorf("running task %q not found", id)
	}

	task.Status = RunStatusRunning
	task.StartedAt = time.Now()
	task.Conductor = cond
	task.Cancel = cancel

	// Get task ID from conductor if available
	if cond != nil {
		task.TaskID = cond.GetTaskID()
	}

	// Publish event
	if r.bus != nil {
		r.bus.PublishRaw(eventbus.Event{
			Type:      TypeTaskStarted,
			Timestamp: time.Now(),
			Data: map[string]any{
				"running_task_id": id,
				"reference":       task.Reference,
				"task_id":         task.TaskID,
			},
		})
	}

	return nil
}

// Complete marks a task as completed or failed based on the error.
func (r *Registry) Complete(id string, err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return fmt.Errorf("running task %q not found", id)
	}

	task.FinishedAt = time.Now()
	task.Error = err

	eventType := TypeTaskCompleted
	if err != nil {
		task.Status = RunStatusFailed
		eventType = TypeTaskFailed
	} else {
		task.Status = RunStatusCompleted
	}

	// Publish event
	if r.bus != nil {
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		r.bus.PublishRaw(eventbus.Event{
			Type:      eventType,
			Timestamp: time.Now(),
			Data: map[string]any{
				"running_task_id": id,
				"reference":       task.Reference,
				"task_id":         task.TaskID,
				"error":           errStr,
				"duration_ms":     task.Duration().Milliseconds(),
			},
		})
	}

	return nil
}

// Cancel cancels a running task.
func (r *Registry) Cancel(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return fmt.Errorf("running task %q not found", id)
	}

	if task.IsTerminal() {
		return fmt.Errorf("task %q already finished with status %s", id, task.Status)
	}

	// Call cancel function if available
	if task.Cancel != nil {
		task.Cancel()
	}

	task.Status = RunStatusCancelled
	task.FinishedAt = time.Now()

	// Publish event
	if r.bus != nil {
		r.bus.PublishRaw(eventbus.Event{
			Type:      TypeTaskCancelled,
			Timestamp: time.Now(),
			Data: map[string]any{
				"running_task_id": id,
				"reference":       task.Reference,
				"task_id":         task.TaskID,
				"duration_ms":     task.Duration().Milliseconds(),
			},
		})
	}

	return nil
}

// Get returns a running task by ID.
// Returns nil if not found.
func (r *Registry) Get(id string) *RunningTask {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]
	if !ok {
		return nil
	}

	// Return a copy to avoid data races
	taskCopy := *task

	return &taskCopy
}

// GetByTaskID returns a running task by its mehrhof task ID.
// Returns nil if not found.
func (r *Registry) GetByTaskID(taskID string) *RunningTask {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, task := range r.tasks {
		if task.TaskID == taskID {
			taskCopy := *task

			return &taskCopy
		}
	}

	return nil
}

// List returns all running tasks (both active and completed).
func (r *Registry) List() []*RunningTask {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*RunningTask, 0, len(r.tasks))
	for _, task := range r.tasks {
		taskCopy := *task
		result = append(result, &taskCopy)
	}

	return result
}

// ListRunning returns only actively running tasks.
func (r *Registry) ListRunning() []*RunningTask {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*RunningTask
	for _, task := range r.tasks {
		if task.Status == RunStatusRunning || task.Status == RunStatusPending {
			taskCopy := *task
			result = append(result, &taskCopy)
		}
	}

	return result
}

// Count returns the total number of tasks in the registry.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tasks)
}

// CountRunning returns the number of actively running tasks.
func (r *Registry) CountRunning() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, task := range r.tasks {
		if task.Status == RunStatusRunning {
			count++
		}
	}

	return count
}

// Remove removes a task from the registry.
// Only completed/failed/cancelled tasks can be removed.
func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return fmt.Errorf("running task %q not found", id)
	}

	if !task.IsTerminal() {
		return fmt.Errorf("cannot remove task %q with status %s (must be terminal)", id, task.Status)
	}

	delete(r.tasks, id)

	return nil
}

// Clear removes all terminal tasks from the registry.
// Running tasks are not affected.
func (r *Registry) Clear() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for id, task := range r.tasks {
		if task.IsTerminal() {
			delete(r.tasks, id)
			count++
		}
	}

	return count
}

// WaitAll blocks until all registered tasks complete.
// Returns the first error encountered, or nil if all succeeded.
// Respects context cancellation.
func (r *Registry) WaitAll(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			running := r.ListRunning()
			if len(running) == 0 {
				// All done - check for errors
				for _, task := range r.List() {
					if task.Error != nil {
						return task.Error
					}
				}

				return nil
			}
		}
	}
}

// AddNote sends a note to a specific running task.
func (r *Registry) AddNote(ctx context.Context, id string, message string) error {
	r.mu.RLock()
	task, ok := r.tasks[id]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("running task %q not found", id)
	}

	if task.Conductor == nil {
		return fmt.Errorf("task %q has no conductor (not yet started?)", id)
	}

	if task.IsTerminal() {
		return fmt.Errorf("cannot add note to finished task %q", id)
	}

	// Publish event for UI updates
	if r.bus != nil {
		r.bus.PublishRaw(eventbus.Event{
			Type:      TypeTaskProgressNote,
			Timestamp: time.Now(),
			Data: map[string]any{
				"running_task_id": id,
				"message":         message,
			},
		})
	}

	return task.Conductor.AddNote(ctx, message)
}

// SetWorktreePath updates the worktree path for a running task.
func (r *Registry) SetWorktreePath(id string, path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return fmt.Errorf("running task %q not found", id)
	}

	task.WorktreePath = path

	return nil
}

// generateID creates a short unique identifier.
// Uses base64url encoding of 4 random bytes = 6 characters.
func (r *Registry) generateID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%x", time.Now().UnixNano()&0xFFFFFF)
	}

	// Use base64url encoding (URL-safe, no padding)
	id := base64.RawURLEncoding.EncodeToString(b)

	// Ensure uniqueness
	for r.tasks[id] != nil {
		if _, err := rand.Read(b); err != nil {
			return fmt.Sprintf("%x%d", time.Now().UnixNano()&0xFFFFFF, len(r.tasks))
		}
		id = base64.RawURLEncoding.EncodeToString(b)
	}

	return id
}
