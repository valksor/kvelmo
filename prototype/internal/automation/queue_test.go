package automation

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestNewJobQueue(t *testing.T) {
	cfg := QueueConfig{
		MaxWorkers: 2,
		JobTimeout: 5 * time.Minute,
	}

	q := NewJobQueue(cfg)

	if q == nil {
		t.Fatal("Expected queue to be created")
	}

	status := q.Status()
	if status.Workers != 2 {
		t.Errorf("Expected Workers 2, got %d", status.Workers)
	}
}

func TestJobQueue_Enqueue(t *testing.T) {
	cfg := QueueConfig{
		MaxWorkers: 1,
		JobTimeout: 1 * time.Minute,
	}

	q := NewJobQueue(cfg)

	event := &WebhookEvent{
		ID:       "test-123",
		Provider: "github",
		Type:     EventTypeIssueOpened,
	}
	job := &WebhookJob{
		Event:        event,
		WorkflowType: WorkflowTypeIssueFix,
		MaxAttempts:  3,
	}

	err := q.Enqueue(job)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	status := q.Status()
	if status.PendingJobs != 1 {
		t.Errorf("Expected 1 pending job, got %d", status.PendingJobs)
	}
}

func TestJobQueue_CancelJob(t *testing.T) {
	cfg := QueueConfig{
		MaxWorkers: 1,
		JobTimeout: 1 * time.Minute,
	}

	q := NewJobQueue(cfg)

	event := &WebhookEvent{
		ID:       "test-123",
		Provider: "github",
		Type:     EventTypeIssueOpened,
	}
	job := &WebhookJob{
		Event:        event,
		WorkflowType: WorkflowTypeIssueFix,
		MaxAttempts:  3,
	}

	err := q.Enqueue(job)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Cancel the job.
	err = q.CancelJob(job.ID)
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	// Get the job and check status.
	cancelledJob, ok := q.GetJob(job.ID)
	if !ok {
		t.Fatal("Expected job to be found")
	}

	if cancelledJob.Status != JobStatusCancelled {
		t.Errorf("Expected status %v, got %v", JobStatusCancelled, cancelledJob.Status)
	}
}

func TestJobQueue_GetJob(t *testing.T) {
	cfg := QueueConfig{
		MaxWorkers: 1,
		JobTimeout: 1 * time.Minute,
	}

	q := NewJobQueue(cfg)

	event := &WebhookEvent{
		ID:       "test-123",
		Provider: "github",
		Type:     EventTypeIssueOpened,
	}
	job := &WebhookJob{
		Event:        event,
		WorkflowType: WorkflowTypeIssueFix,
		MaxAttempts:  3,
	}

	err := q.Enqueue(job)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Get existing job.
	retrieved, ok := q.GetJob(job.ID)
	if !ok {
		t.Fatal("Expected job to be found")
	}

	if retrieved.ID != job.ID {
		t.Errorf("Expected job ID %s, got %s", job.ID, retrieved.ID)
	}

	// Get non-existent job.
	_, ok = q.GetJob("nonexistent")
	if ok {
		t.Error("Expected job to not be found")
	}
}

func TestJobQueue_ListJobs(t *testing.T) {
	cfg := QueueConfig{
		MaxWorkers: 1,
		JobTimeout: 1 * time.Minute,
	}

	q := NewJobQueue(cfg)

	// Enqueue multiple jobs.
	for i := range 5 {
		event := &WebhookEvent{
			ID:       "test-" + strconv.Itoa(int(rune('A'+i))),
			Provider: "github",
			Type:     EventTypeIssueOpened,
		}
		job := &WebhookJob{
			Event:        event,
			WorkflowType: WorkflowTypeIssueFix,
			MaxAttempts:  3,
		}
		err := q.Enqueue(job)
		if err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}
	}

	// List all jobs.
	jobs := q.ListJobs(nil)
	if len(jobs) != 5 {
		t.Errorf("Expected 5 jobs, got %d", len(jobs))
	}

	// List by status.
	pendingStatus := JobStatusPending
	statusJobs := q.ListJobs(&pendingStatus)
	if len(statusJobs) != 5 {
		t.Errorf("Expected 5 pending jobs, got %d", len(statusJobs))
	}
}

func TestJobQueue_Status(t *testing.T) {
	cfg := QueueConfig{
		MaxWorkers: 2,
		JobTimeout: 1 * time.Minute,
	}

	q := NewJobQueue(cfg)

	// Check initial status.
	status := q.Status()
	if status.PendingJobs != 0 || status.RunningJobs != 0 || status.CompletedJobs != 0 || status.FailedJobs != 0 {
		t.Error("Expected all stats to be 0 initially")
	}

	// Enqueue some jobs.
	for i := range 3 {
		event := &WebhookEvent{
			ID:       "test-" + strconv.Itoa(int(rune('A'+i))),
			Provider: "github",
			Type:     EventTypeIssueOpened,
		}
		job := &WebhookJob{
			Event:        event,
			WorkflowType: WorkflowTypeIssueFix,
			MaxAttempts:  3,
		}
		err := q.Enqueue(job)
		if err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}
	}

	status = q.Status()
	if status.PendingJobs != 3 {
		t.Errorf("Expected 3 pending jobs, got %d", status.PendingJobs)
	}
}

func TestJobQueue_StartStop(t *testing.T) {
	processedJobs := make(chan string, 10)

	//nolint:unparam // test handler always succeeds
	handler := func(_ context.Context, job *WebhookJob) error {
		processedJobs <- job.ID

		return nil
	}

	cfg := QueueConfig{
		MaxWorkers: 2,
		JobTimeout: 1 * time.Minute,
	}

	q := NewJobQueue(cfg)

	// Enqueue jobs before starting.
	for i := range 3 {
		event := &WebhookEvent{
			ID:       "test-" + strconv.Itoa(int(rune('A'+i))),
			Provider: "github",
			Type:     EventTypeIssueOpened,
		}
		job := &WebhookJob{
			Event:        event,
			WorkflowType: WorkflowTypeIssueFix,
			MaxAttempts:  3,
		}
		err := q.Enqueue(job)
		if err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}
	}

	// Start the queue.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		q.Start(ctx, handler)
	}()

	// Wait for jobs to be processed.
	processed := 0
	timeout := time.After(5 * time.Second)

	for processed < 3 {
		select {
		case <-processedJobs:
			processed++
		case <-timeout:
			t.Fatalf("Timeout waiting for jobs, processed %d of 3", processed)
		}
	}

	// Stop the queue.
	cancel()
	wg.Wait()

	// Verify stats.
	status := q.Status()
	if status.CompletedJobs != 3 {
		t.Errorf("Expected 3 completed jobs, got %d", status.CompletedJobs)
	}
}

func TestJobQueue_DefaultConfig(t *testing.T) {
	// Test with zero config values.
	cfg := QueueConfig{}
	q := NewJobQueue(cfg)

	status := q.Status()
	if status.Workers != 1 {
		t.Errorf("Expected default Workers 1, got %d", status.Workers)
	}
}
