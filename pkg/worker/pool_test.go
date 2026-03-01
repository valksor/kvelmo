package worker

import (
	"testing"
	"time"
)

func newTestPool(t *testing.T) *Pool {
	t.Helper()
	pool := NewPool(DefaultPoolConfig())
	if err := pool.Start(); err != nil {
		t.Fatalf("pool.Start() error = %v", err)
	}
	t.Cleanup(func() { _ = pool.Stop() })

	return pool
}

func TestSubmit_CreatesJob(t *testing.T) {
	pool := newTestPool(t)

	job, err := pool.Submit(JobTypePlan, "wt-1", "write a plan")
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if job == nil {
		t.Fatal("Submit() = nil job")
	}
	if job.ID == "" {
		t.Error("job.ID should not be empty")
	}
	if job.Type != JobTypePlan {
		t.Errorf("job.Type = %q, want %q", job.Type, JobTypePlan)
	}
	if job.WorktreeID != "wt-1" {
		t.Errorf("job.WorktreeID = %q, want wt-1", job.WorktreeID)
	}
	if job.Prompt != "write a plan" {
		t.Errorf("job.Prompt = %q, want 'write a plan'", job.Prompt)
	}
}

func TestSubmit_JobQueuedStatus(t *testing.T) {
	// Pool with no workers — job stays queued initially
	pool := NewPool(PoolConfig{MaxWorkers: 5, BasePort: 8765})
	if err := pool.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pool.Stop() }()

	job, err := pool.Submit(JobTypePlan, "wt-1", "test")
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	// No workers available, should be queued
	if job.Status != JobStatusQueued {
		t.Errorf("job.Status = %q, want %q", job.Status, JobStatusQueued)
	}
}

func TestGetJob(t *testing.T) {
	pool := newTestPool(t)

	job, err := pool.Submit(JobTypeImplement, "wt-1", "implement X")
	if err != nil {
		t.Fatal(err)
	}

	got := pool.GetJob(job.ID)
	if got == nil {
		t.Fatal("GetJob() = nil, want job")
	}
	if got.ID != job.ID {
		t.Errorf("GetJob().ID = %q, want %q", got.ID, job.ID)
	}
}

func TestGetJob_NotFound(t *testing.T) {
	pool := newTestPool(t)
	if got := pool.GetJob("no-such-id"); got != nil {
		t.Errorf("GetJob(missing) = %v, want nil", got)
	}
}

func TestListJobs_Empty(t *testing.T) {
	pool := newTestPool(t)
	jobs := pool.ListJobs()
	if len(jobs) != 0 {
		t.Errorf("ListJobs() empty = %d, want 0", len(jobs))
	}
}

func TestListJobs_AfterSubmit(t *testing.T) {
	pool := newTestPool(t)

	if _, err := pool.Submit(JobTypePlan, "wt-1", "prompt1"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Submit(JobTypeImplement, "wt-1", "prompt2"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Submit(JobTypeReview, "wt-1", "prompt3"); err != nil {
		t.Fatal(err)
	}

	jobs := pool.ListJobs()
	if len(jobs) != 3 {
		t.Errorf("ListJobs() len = %d, want 3", len(jobs))
	}
}

func TestListQueuedJobs(t *testing.T) {
	// Pool with no workers — all jobs stay queued
	pool := NewPool(PoolConfig{MaxWorkers: 5, BasePort: 8765})
	if err := pool.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pool.Stop() }()

	if _, err := pool.Submit(JobTypePlan, "wt-1", "plan it"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Submit(JobTypeImplement, "wt-1", "implement it"); err != nil {
		t.Fatal(err)
	}

	queued := pool.ListQueuedJobs()
	if len(queued) != 2 {
		t.Errorf("ListQueuedJobs() len = %d, want 2", len(queued))
	}
	for _, j := range queued {
		if j.Status != JobStatusQueued && j.Status != JobStatusInProgress {
			t.Errorf("job %q status = %q, want queued or in_progress", j.ID, j.Status)
		}
	}
}

func TestStats_Empty(t *testing.T) {
	pool := newTestPool(t)
	stats := pool.Stats()

	if stats.TotalWorkers != 0 {
		t.Errorf("TotalWorkers = %d, want 0", stats.TotalWorkers)
	}
	if stats.AvailableWorkers != 0 {
		t.Errorf("AvailableWorkers = %d, want 0", stats.AvailableWorkers)
	}
	if stats.QueuedJobs != 0 {
		t.Errorf("QueuedJobs = %d, want 0", stats.QueuedJobs)
	}
}

func TestStats_WithWorkersAndJobs(t *testing.T) {
	pool := newTestPool(t)

	pool.AddWorker()
	pool.AddWorker()

	stats := pool.Stats()
	if stats.TotalWorkers != 2 {
		t.Errorf("TotalWorkers = %d, want 2", stats.TotalWorkers)
	}
	if stats.AvailableWorkers != 2 {
		t.Errorf("AvailableWorkers = %d, want 2", stats.AvailableWorkers)
	}
}

func TestAddWorkerWithAgent_Fields(t *testing.T) {
	pool := newTestPool(t)

	w := pool.AddWorkerWithAgent("claude")
	if w == nil {
		t.Fatal("AddWorkerWithAgent() = nil")
	}
	if w.AgentName != "claude" {
		t.Errorf("w.AgentName = %q, want claude", w.AgentName)
	}
	if w.Agent != nil {
		t.Error("AddWorkerWithAgent should not set Agent (not connected)")
	}
}

func TestSimulatedJobExecution(t *testing.T) {
	pool := newTestPool(t)

	// Add a simulated worker (nil agent)
	pool.AddWorker()

	job, err := pool.Submit(JobTypeImplement, "wt-1", "implement the feature")
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	stream := pool.Stream(job.ID)

	// Collect events with timeout
	var events []Event
	deadline := time.After(5 * time.Second)
	done := false
	for !done {
		select {
		case ev, ok := <-stream:
			if !ok {
				done = true
			} else {
				events = append(events, ev)
			}
		case <-deadline:
			t.Fatal("timed out waiting for simulated job to complete")
		}
	}

	// Job should be done
	got := pool.GetJob(job.ID)
	if got.Status != JobStatusDone {
		t.Errorf("job.Status = %q, want %q", got.Status, JobStatusDone)
	}
	if got.Result == "" {
		t.Error("job.Result should not be empty after simulated completion")
	}
	if got.CompletedAt == nil {
		t.Error("job.CompletedAt should be set")
	}

	// Should have received at least stream and completed events
	if len(events) == 0 {
		t.Error("expected at least one event from simulated job")
	}
}

func TestStreamClosesOnCompletion(t *testing.T) {
	pool := newTestPool(t)
	pool.AddWorker()

	job, err := pool.Submit(JobTypePlan, "wt-1", "plan the feature")
	if err != nil {
		t.Fatal(err)
	}

	stream := pool.Stream(job.ID)
	if stream == nil {
		t.Fatal("Stream() returned nil for active job")
	}

	// Drain until closed
	deadline := time.After(5 * time.Second)
	closed := false
	for !closed {
		select {
		case _, ok := <-stream:
			if !ok {
				closed = true
			}
		case <-deadline:
			t.Fatal("stream not closed within timeout")
		}
	}

	if !closed {
		t.Error("stream should be closed after job completion")
	}
}

func TestSubmitWithOptions(t *testing.T) {
	pool := newTestPool(t)

	opts := &JobOptions{
		WorkDir:     "/tmp/test",
		Environment: map[string]string{"KEY": "val"},
		Metadata:    map[string]any{"meta": "data"},
	}

	job, err := pool.SubmitWithOptions(JobTypePlan, "wt-2", "plan it", opts)
	if err != nil {
		t.Fatalf("SubmitWithOptions() error = %v", err)
	}

	if job.WorkDir != "/tmp/test" {
		t.Errorf("job.WorkDir = %q, want /tmp/test", job.WorkDir)
	}
	if job.Environment["KEY"] != "val" {
		t.Errorf("job.Environment[KEY] = %q, want val", job.Environment["KEY"])
	}
	if job.Metadata["meta"] != "data" {
		t.Errorf("job.Metadata[meta] = %q, want data", job.Metadata["meta"])
	}
}
