package automation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/valksor/go-toolkit/eventbus"
)

var (
	// ErrQueueClosed is returned when trying to enqueue on a closed queue.
	ErrQueueClosed = errors.New("queue is closed")

	// ErrJobNotFound is returned when a job is not found.
	ErrJobNotFound = errors.New("job not found")

	// ErrJobNotPending is returned when trying to cancel a non-pending job.
	ErrJobNotPending = errors.New("job is not pending")
)

// JobHandler processes a single webhook job.
type JobHandler func(ctx context.Context, job *WebhookJob) error

// JobQueue manages webhook jobs with configurable concurrency.
//
//nolint:containedctx // Long-running service requires stored context for graceful shutdown
type JobQueue struct {
	mu sync.RWMutex

	// Job storage.
	jobs    map[string]*WebhookJob
	pending []*WebhookJob

	// Worker configuration.
	maxWorkers int
	jobTimeout time.Duration

	// Running state.
	running  map[string]*WebhookJob
	workerWg sync.WaitGroup
	jobCh    chan *WebhookJob
	ctx      context.Context
	cancel   context.CancelFunc
	closed   bool

	// Readiness signal.
	ready chan struct{}

	// Event publishing.
	eventBus *eventbus.Bus

	// Statistics.
	stats QueueStatus
}

// QueueConfig holds configuration for the job queue.
type QueueConfig struct {
	MaxWorkers int           // Number of concurrent workers (default: 1)
	JobTimeout time.Duration // Timeout per job (default: 30m)
	EventBus   *eventbus.Bus // Optional event bus for publishing events
}

// NewJobQueue creates a new job queue with the given configuration.
func NewJobQueue(cfg QueueConfig) *JobQueue {
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 1
	}
	if cfg.JobTimeout <= 0 {
		cfg.JobTimeout = 30 * time.Minute
	}

	return &JobQueue{
		jobs:       make(map[string]*WebhookJob),
		pending:    make([]*WebhookJob, 0),
		running:    make(map[string]*WebhookJob),
		maxWorkers: cfg.MaxWorkers,
		jobTimeout: cfg.JobTimeout,
		eventBus:   cfg.EventBus,
		ready:      make(chan struct{}),
		stats: QueueStatus{
			Enabled: true,
			Workers: cfg.MaxWorkers,
		},
	}
}

// Start begins processing jobs with the given handler.
// This method blocks until Stop is called or the context is cancelled.
func (q *JobQueue) Start(ctx context.Context, handler JobHandler) {
	q.mu.Lock()
	if q.running == nil {
		q.running = make(map[string]*WebhookJob)
	}
	q.ctx, q.cancel = context.WithCancel(ctx)
	q.jobCh = make(chan *WebhookJob, q.maxWorkers*2)
	q.stats.Running = true
	q.mu.Unlock()

	// Start workers.
	for range q.maxWorkers {
		q.workerWg.Add(1)
		go q.worker(handler) //nolint:contextcheck // Worker uses q.ctx from struct
	}

	// Dispatch pending jobs.
	go q.dispatcher()

	// Periodically clean up terminal jobs older than retention TTL.
	go q.cleanupLoop()

	// Signal readiness — workers and dispatcher are initialized.
	close(q.ready)

	// Wait for context cancellation.
	<-q.ctx.Done()

	// Graceful shutdown.
	q.mu.Lock()
	q.stats.Running = false
	q.mu.Unlock()
}

// Ready returns a channel that is closed when workers and the dispatcher are initialized.
// Callers can select on this to wait for the queue to be ready before enqueuing.
func (q *JobQueue) Ready() <-chan struct{} {
	return q.ready
}

// Stop gracefully shuts down the queue, waiting for running jobs to complete.
func (q *JobQueue) Stop(timeout time.Duration) error {
	q.mu.Lock()
	if q.cancel == nil {
		q.mu.Unlock()

		return nil
	}
	q.cancel()
	q.closed = true
	q.mu.Unlock()

	// Wait for workers with timeout.
	done := make(chan struct{})
	go func() {
		q.workerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return errors.New("timeout waiting for workers to stop")
	}
}

// Enqueue adds a job to the queue.
func (q *JobQueue) Enqueue(job *WebhookJob) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	// Set defaults.
	if job.ID == "" {
		job.ID = generateJobID()
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 1
	}
	job.Status = JobStatusPending
	job.CreatedAt = time.Now()

	// Store and queue.
	q.jobs[job.ID] = job
	q.pending = append(q.pending, job)
	q.stats.PendingJobs++

	slog.Info("automation.job.enqueued",
		"job_id", job.ID,
		"provider", job.Event.Provider,
		"workflow", job.WorkflowType,
		"priority", job.Priority,
		"pending_count", q.stats.PendingJobs,
	)

	// Publish event.
	q.publishEvent("job.enqueued", job)

	return nil
}

// CancelJob cancels a pending job.
func (q *JobQueue) CancelJob(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, exists := q.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	if job.Status != JobStatusPending {
		return ErrJobNotPending
	}

	job.Status = JobStatusCancelled
	now := time.Now()
	job.CompletedAt = &now
	q.stats.PendingJobs--
	q.stats.CancelledJobs++

	// Remove from pending queue.
	for i, j := range q.pending {
		if j.ID == id {
			q.pending = append(q.pending[:i], q.pending[i+1:]...)

			break
		}
	}

	slog.Info("job cancelled", "job_id", id)
	q.publishEvent("job.cancelled", job)

	return nil
}

// RetryJob resets a failed job and re-enqueues it for processing.
func (q *JobQueue) RetryJob(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	job, exists := q.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	if job.Status != JobStatusFailed {
		return fmt.Errorf("only failed jobs can be retried (current status: %s)", job.Status)
	}

	// Reset job for retry.
	job.Status = JobStatusPending
	job.Attempts = 0
	job.Error = ""
	job.CompletedAt = nil
	job.Result = nil
	job.RetryAfter = time.Time{} // Clear backoff for immediate dispatch

	q.pending = append(q.pending, job)
	q.stats.PendingJobs++
	q.stats.FailedJobs--

	slog.Info("job retried", "job_id", id)
	q.publishEvent("job.retried", job)

	return nil
}

// GetJob returns a job by ID.
func (q *JobQueue) GetJob(id string) (*WebhookJob, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	job, exists := q.jobs[id]

	return job, exists
}

// ListJobs returns jobs, optionally filtered by status.
func (q *JobQueue) ListJobs(status *JobStatus) []*WebhookJob {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []*WebhookJob
	for _, job := range q.jobs {
		if status == nil || job.Status == *status {
			result = append(result, job)
		}
	}

	return result
}

// Status returns the current queue status.
func (q *JobQueue) Status() QueueStatus {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return q.stats
}

// dispatcher moves jobs from pending to the job channel.
func (q *JobQueue) dispatcher() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			close(q.jobCh)

			return
		case <-ticker.C:
			q.dispatchNext()
		}
	}
}

// dispatchNext sends the next pending job to workers if available.
func (q *JobQueue) dispatchNext() {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if we can dispatch more jobs.
	if len(q.running) >= q.maxWorkers || len(q.pending) == 0 {
		return
	}

	// Get highest priority job that is ready to dispatch.
	bestIdx := -1
	bestPriority := -1
	now := time.Now()
	for i, job := range q.pending {
		// Skip jobs in backoff period (not yet ready for retry).
		if !job.RetryAfter.IsZero() && now.Before(job.RetryAfter) {
			continue
		}
		if job.Priority > bestPriority {
			bestPriority = job.Priority
			bestIdx = i
		}
	}

	if bestIdx < 0 {
		return // No dispatchable jobs
	}

	// Remove from pending and send to workers.
	job := q.pending[bestIdx]
	q.pending = append(q.pending[:bestIdx], q.pending[bestIdx+1:]...)
	q.stats.PendingJobs--

	// Non-blocking send (channel should have capacity).
	select {
	case q.jobCh <- job:
	default:
		// Channel full, put job back at end of queue to preserve ordering.
		q.pending = append(q.pending, job)
		q.stats.PendingJobs++
		slog.Warn("worker channel full, job re-queued", "job_id", job.ID)
	}
}

// worker processes jobs from the channel.
func (q *JobQueue) worker(handler JobHandler) {
	defer q.workerWg.Done()

	for job := range q.jobCh {
		q.processJob(handler, job)
	}
}

// processJob executes a single job.
func (q *JobQueue) processJob(handler JobHandler, job *WebhookJob) {
	// Mark as running.
	q.mu.Lock()
	job.Status = JobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	job.Attempts++
	q.running[job.ID] = job
	q.stats.RunningJobs++
	runningCount := q.stats.RunningJobs // Capture under lock
	pendingCount := q.stats.PendingJobs // Capture under lock
	q.mu.Unlock()

	slog.Info("automation.job.dispatched",
		"job_id", job.ID,
		"attempt", job.Attempts,
		"workflow", job.WorkflowType,
		"running_count", runningCount,
		"pending_count", pendingCount,
	)
	q.publishEvent("job.started", job)

	// Execute with timeout.
	ctx, cancel := context.WithTimeout(q.ctx, q.jobTimeout)
	defer cancel()

	start := time.Now()
	err := handler(ctx, job)
	duration := time.Since(start)

	// Update job state.
	q.mu.Lock()
	delete(q.running, job.ID)
	q.stats.RunningJobs--
	completedAt := time.Now()
	job.CompletedAt = &completedAt

	if err != nil {
		job.Error = err.Error()
		if job.CanRetry() {
			// Re-queue for retry with exponential backoff.
			job.Status = JobStatusPending
			job.CompletedAt = nil
			exp := min(max(job.Attempts-1, 0), 20)                    // Cap exponent to prevent overflow: 2^20 * 30s ≈ 9.7h, safely above 10min cap
			backoff := time.Duration(1<<uint(exp)) * 30 * time.Second //nolint:gosec // exp clamped to [0,20]
			if backoff > 10*time.Minute {
				backoff = 10 * time.Minute
			}
			job.RetryAfter = time.Now().Add(backoff)
			q.pending = append(q.pending, job)
			q.stats.PendingJobs++
			slog.Warn("job failed, will retry",
				"job_id", job.ID,
				"attempt", job.Attempts,
				"retry_after", job.RetryAfter,
				"error", err,
			)
		} else {
			job.Status = JobStatusFailed
			job.Result = &JobResult{
				Success:      false,
				ErrorMessage: err.Error(),
				Duration:     duration,
			}
			q.stats.FailedJobs++
			slog.Error("job failed",
				"job_id", job.ID,
				"attempts", job.Attempts,
				"error", err,
			)
		}
	} else {
		job.Status = JobStatusCompleted
		if job.Result == nil {
			job.Result = &JobResult{Success: true, Duration: duration}
		} else {
			job.Result.Success = true
			job.Result.Duration = duration
		}
		q.stats.CompletedJobs++
		slog.Info("automation.job.completed",
			"job_id", job.ID,
			"workflow", job.WorkflowType,
			"attempts", job.Attempts,
			"duration_ms", duration.Milliseconds(),
		)
	}
	q.mu.Unlock()

	// Publish completion event.
	switch job.Status {
	case JobStatusCompleted:
		q.publishEvent("job.completed", job)
	case JobStatusFailed:
		q.publishEvent("job.failed", job)
	case JobStatusCancelled:
		q.publishEvent("job.cancelled", job)
	case JobStatusPending, JobStatusRunning:
		// Job still in progress, no completion event needed.
	}
}

// publishEvent publishes a job event to the event bus.
func (q *JobQueue) publishEvent(eventType string, job *WebhookJob) {
	if q.eventBus == nil {
		return
	}

	q.eventBus.PublishRaw(eventbus.Event{
		Type: eventbus.Type("automation." + eventType),
		Data: map[string]any{
			"job_id":        job.ID,
			"status":        job.Status,
			"workflow_type": job.WorkflowType,
			"provider":      job.Event.Provider,
			"attempts":      job.Attempts,
		},
	})
}

// generateJobID creates a unique job ID.
func generateJobID() string {
	return fmt.Sprintf("job-%d", time.Now().UnixNano())
}

// jobRetentionTTL is how long terminal jobs are kept before cleanup.
const jobRetentionTTL = 24 * time.Hour

// cleanupLoop periodically removes terminal jobs older than the retention TTL.
func (q *JobQueue) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			q.cleanupStaleJobs()
		}
	}
}

// cleanupStaleJobs removes completed/failed/cancelled jobs older than jobRetentionTTL.
func (q *JobQueue) cleanupStaleJobs() {
	q.mu.Lock()
	defer q.mu.Unlock()

	cutoff := time.Now().Add(-jobRetentionTTL)

	for id, job := range q.jobs {
		if job.IsTerminal() && job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
			delete(q.jobs, id)
		}
	}
}
