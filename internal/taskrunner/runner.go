package taskrunner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/valksor/go-toolkit/eventbus"
)

// TaskResult holds the outcome of a single task execution.
type TaskResult struct {
	// RunningTaskID is the registry ID for this task.
	RunningTaskID string

	// Reference is the original task reference.
	Reference string

	// TaskID is the mehrhof task ID (assigned after start).
	TaskID string

	// Error is nil on success, or the error that caused failure.
	Error error

	// Duration is how long the task took to execute.
	Duration time.Duration

	// WorktreePath is the path to the task's worktree (if any).
	WorktreePath string
}

// RunOptions configures parallel task execution.
type RunOptions struct {
	// MaxParallel is the maximum number of concurrent tasks.
	// 0 or negative means unlimited.
	MaxParallel int

	// RequireWorktree enforces that parallel execution uses worktrees.
	// This prevents file conflicts between parallel tasks.
	RequireWorktree bool

	// StopOnError halts all remaining tasks when one fails.
	// If false, all tasks run to completion regardless of failures.
	StopOnError bool

	// Timeout is the maximum duration for all tasks combined.
	// 0 means no timeout.
	Timeout time.Duration

	// OnTaskStart is called when a task begins execution.
	OnTaskStart func(runningID, ref string)

	// OnTaskComplete is called when a task finishes (success or failure).
	OnTaskComplete func(result TaskResult)

	// ConductorFactory creates conductor instances for each task.
	// This is required and must be set before calling Run().
	ConductorFactory ConductorFactory
}

// ConductorFactory creates new conductor instances for parallel tasks.
// Each task needs its own conductor to maintain isolated state.
type ConductorFactory interface {
	// Create creates a new conductor instance for the given task reference.
	// The conductor should be fully initialized and ready to start the task.
	Create(ctx context.Context, ref string, worktree bool) (TaskConductor, error)
}

// TaskConductor defines the conductor operations needed by the runner.
// This interface avoids circular imports with the conductor package.
type TaskConductor interface {
	ConductorRef

	// Start begins the task from the given reference.
	Start(ctx context.Context, ref string) error

	// Plan runs the planning phase.
	Plan(ctx context.Context) error

	// Implement runs the implementation phase.
	Implement(ctx context.Context) error

	// GetWorktreePath returns the worktree path if using worktrees.
	GetWorktreePath() string

	// Close performs cleanup for the conductor.
	Close() error
}

// Runner coordinates parallel task execution.
type Runner struct {
	registry    *Registry
	bus         *eventbus.Bus
	maxParallel int
}

// NewRunner creates a new task runner.
func NewRunner(registry *Registry, maxParallel int, bus *eventbus.Bus) *Runner {
	if maxParallel <= 0 {
		maxParallel = 1
	}

	return &Runner{
		registry:    registry,
		maxParallel: maxParallel,
		bus:         bus,
	}
}

// Run executes multiple tasks in parallel.
// It respects the maxParallel limit and returns when all tasks complete.
func (r *Runner) Run(ctx context.Context, refs []string, opts RunOptions) ([]TaskResult, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	if opts.ConductorFactory == nil {
		return nil, errors.New("ConductorFactory is required")
	}

	// Note: RequireWorktree is enforced by the ConductorFactory.Create() method
	// which receives the worktree flag and configures the conductor accordingly.

	// Apply timeout if specified
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Determine parallelism
	maxParallel := opts.MaxParallel
	if maxParallel <= 0 {
		maxParallel = r.maxParallel
	}
	if maxParallel > len(refs) {
		maxParallel = len(refs)
	}

	// Create semaphore channel to limit concurrency
	sem := make(chan struct{}, maxParallel)

	// Result channel
	resultCh := make(chan TaskResult, len(refs))

	// Stop signal for early termination
	stopCtx, stopCancel := context.WithCancel(ctx)
	defer stopCancel()

	// WaitGroup for all goroutines
	var wg sync.WaitGroup

	// Launch workers
	for _, ref := range refs {
		// Register task in registry
		runningTask := r.registry.Register(ref)

		wg.Add(1)
		go func(ref string, runningID string) {
			defer wg.Done()

			// Acquire semaphore slot
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-stopCtx.Done():
				// Cancelled before we could start
				result := TaskResult{
					RunningTaskID: runningID,
					Reference:     ref,
					Error:         stopCtx.Err(),
				}
				_ = r.registry.Complete(runningID, stopCtx.Err())
				resultCh <- result

				return
			}

			// Execute the task
			result := r.executeTask(stopCtx, runningID, ref, opts)
			resultCh <- result

			// Notify callback
			if opts.OnTaskComplete != nil {
				opts.OnTaskComplete(result)
			}

			// Stop others on error if configured
			if result.Error != nil && opts.StopOnError {
				stopCancel()
			}
		}(ref, runningTask.ID)
	}

	// Wait for all workers in a separate goroutine
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	var results []TaskResult
	for result := range resultCh {
		results = append(results, result)
	}

	// Check for any errors
	var errs []error
	for _, result := range results {
		if result.Error != nil {
			errs = append(errs, fmt.Errorf("%s: %w", result.Reference, result.Error))
		}
	}

	if len(errs) > 0 {
		return results, errors.Join(errs...)
	}

	return results, nil
}

// executeTask runs a single task with its own conductor.
func (r *Runner) executeTask(ctx context.Context, runningID, ref string, opts RunOptions) TaskResult {
	result := TaskResult{
		RunningTaskID: runningID,
		Reference:     ref,
	}

	startTime := time.Now()

	// Create task-specific context with cancellation
	taskCtx, taskCancel := context.WithCancel(ctx)
	defer taskCancel()

	// Notify start callback
	if opts.OnTaskStart != nil {
		opts.OnTaskStart(runningID, ref)
	}

	// Create conductor for this task
	cond, err := opts.ConductorFactory.Create(taskCtx, ref, opts.RequireWorktree)
	if err != nil {
		result.Error = fmt.Errorf("create conductor: %w", err)
		result.Duration = time.Since(startTime)
		_ = r.registry.Complete(runningID, result.Error)

		return result
	}
	defer func() {
		if cond != nil {
			_ = cond.Close()
		}
	}()

	// Register conductor with registry
	// We wrap cond to satisfy ConductorRef interface
	if err := r.registry.Start(runningID, cond, taskCancel); err != nil {
		result.Error = fmt.Errorf("register conductor: %w", err)
		result.Duration = time.Since(startTime)
		_ = r.registry.Complete(runningID, result.Error)

		return result
	}

	// Update worktree path in registry
	if worktreePath := cond.GetWorktreePath(); worktreePath != "" {
		_ = r.registry.SetWorktreePath(runningID, worktreePath)
		result.WorktreePath = worktreePath
	}

	// Start the task
	if err := cond.Start(taskCtx, ref); err != nil {
		result.Error = fmt.Errorf("start task: %w", err)
		result.Duration = time.Since(startTime)
		_ = r.registry.Complete(runningID, result.Error)

		return result
	}

	// Get task ID
	result.TaskID = cond.GetTaskID()

	// Run planning phase
	if err := cond.Plan(taskCtx); err != nil {
		result.Error = fmt.Errorf("plan: %w", err)
		result.Duration = time.Since(startTime)
		_ = r.registry.Complete(runningID, result.Error)

		return result
	}

	// Run implementation phase
	if err := cond.Implement(taskCtx); err != nil {
		result.Error = fmt.Errorf("implement: %w", err)
		result.Duration = time.Since(startTime)
		_ = r.registry.Complete(runningID, result.Error)

		return result
	}

	// Success
	result.Duration = time.Since(startTime)
	_ = r.registry.Complete(runningID, nil)

	return result
}

// CancelAll cancels all running tasks in the registry.
func (r *Runner) CancelAll() int {
	running := r.registry.ListRunning()
	count := 0
	for _, task := range running {
		if err := r.registry.Cancel(task.ID); err == nil {
			count++
		}
	}

	return count
}

// GetRegistry returns the underlying registry.
func (r *Runner) GetRegistry() *Registry {
	return r.registry
}

// SetMaxParallel updates the maximum parallelism for future runs.
func (r *Runner) SetMaxParallel(n int) {
	if n > 0 {
		r.maxParallel = n
	}
}
