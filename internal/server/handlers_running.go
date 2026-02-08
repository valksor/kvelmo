package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/registration"
	"github.com/valksor/go-mehrhof/internal/taskrunner"
	"github.com/valksor/go-toolkit/eventbus"
)

// parallelStartRequest represents a request to start multiple tasks in parallel.
type parallelStartRequest struct {
	References  []string `json:"references"`   // Task references to start
	MaxWorkers  int      `json:"max_workers"`  // Max parallel workers (default: 2)
	UseWorktree bool     `json:"use_worktree"` // Create worktree for each task
}

// handleRunningTaskStream streams events for a specific running task via SSE.
// GET /api/v1/running/{id}/stream.
func (s *Server) handleRunningTaskStream(w http.ResponseWriter, r *http.Request) {
	// Disable write timeout for SSE (allows indefinite streaming during long agent operations)
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		slog.Debug("could not disable write deadline for SSE", "error", err)
	}

	// Set CORS header first
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Check if response writer supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")

		return
	}

	// Extract task ID from path
	// Path is /api/v1/running/{id}/stream
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/running/")
	taskID := strings.TrimSuffix(path, "/stream")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}

	registry := s.getTaskRegistry()
	if registry == nil {
		s.writeError(w, http.StatusServiceUnavailable, "no parallel tasks running")

		return
	}

	// Check if task exists
	task := registry.Get(taskID)
	if task == nil {
		s.writeError(w, http.StatusNotFound, "task not found: "+taskID)

		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// If no event bus, just keep connection alive with heartbeats
	if s.config.EventBus == nil {
		s.writeSSEEvent(w, flusher, "connected", map[string]string{
			"status":  "connected",
			"task_id": taskID,
		})
		s.runSSEHeartbeatLoop(w, flusher, r.Context())

		return
	}

	// Use channel-based event delivery to prevent panic on client disconnect.
	// The callback only writes to a channel (never panics), while the main loop
	// checks ctx.Done() before writing to ResponseWriter.
	eventCh := make(chan eventbus.Event, 100)
	subID := s.config.EventBus.SubscribeAll(func(e eventbus.Event) {
		// Filter events for this task by checking the Data field
		// Event.Data is already map[string]any
		if id, found := e.Data["id"]; found && id == taskID {
			select {
			case eventCh <- e:
			default:
				// Channel full, drop event (non-blocking to prevent callback from hanging)
			}
		}
	})
	defer s.config.EventBus.Unsubscribe(subID)
	defer close(eventCh)

	// Send initial task state
	s.writeSSEEvent(w, flusher, "connected", map[string]any{
		"status":    "connected",
		"task_id":   taskID,
		"reference": task.Reference,
		"state":     string(task.Status),
	})

	// Run combined event + heartbeat loop (blocks until client disconnects)
	s.runSSEEventLoop(w, flusher, r.Context(), eventCh)
}

// handleParallelStart starts multiple tasks in parallel.
// POST /api/v1/parallel.
func (s *Server) handleParallelStart(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req parallelStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if len(req.References) == 0 {
		s.writeError(w, http.StatusBadRequest, "at least one reference is required")

		return
	}

	maxWorkers := req.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 2 // Default to 2 workers
	}

	// Validate: if > 1 worker and not using worktrees, we might have conflicts
	if maxWorkers > 1 && !req.UseWorktree {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "worktrees required for parallel execution",
			"message": "Set use_worktree=true or reduce max_workers to 1",
		})

		return
	}

	// Get or create registry
	registry := s.getOrCreateTaskRegistry()

	// Create runner
	runner := taskrunner.NewRunner(registry, maxWorkers, s.config.EventBus)

	// Create factory adapter for conductor creation
	factory := &serverConductorFactory{
		server:      s,
		useWorktree: req.UseWorktree,
	}

	// Start tasks in background with a detached context.
	// We use Background() because the HTTP request completes before tasks finish.
	// Parallel tasks are long-running and tracked by the registry.
	//nolint:contextcheck // Intentional: background tasks outlive the HTTP request
	go func() {
		ctx := context.Background()
		opts := taskrunner.RunOptions{
			RequireWorktree:  req.UseWorktree,
			ConductorFactory: factory,
		}

		_, _ = runner.Run(ctx, req.References, opts)
		// Results and errors are tracked per-task in the registry
	}()

	// Return immediately with task IDs
	var taskIDs []string
	for _, task := range registry.List() {
		taskIDs = append(taskIDs, task.ID)
	}

	s.writeJSON(w, http.StatusAccepted, map[string]any{
		"success":     true,
		"message":     "parallel execution started",
		"task_count":  len(req.References),
		"max_workers": maxWorkers,
		"task_ids":    taskIDs,
	})
}

// getTaskRegistry returns the shared task registry, or nil if not set.
func (s *Server) getTaskRegistry() *taskrunner.Registry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.taskRegistry
}

// getOrCreateTaskRegistry returns the task registry, creating it if needed.
func (s *Server) getOrCreateTaskRegistry() *taskrunner.Registry {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.taskRegistry == nil {
		s.taskRegistry = taskrunner.NewRegistry(s.config.EventBus)
	}

	return s.taskRegistry
}

// serverConductorFactory adapts server to create conductors for parallel tasks.
type serverConductorFactory struct {
	server      *Server
	useWorktree bool
}

// Create creates a new conductor for a task.
func (f *serverConductorFactory) Create(ctx context.Context, _ string, worktree bool) (taskrunner.TaskConductor, error) {
	// Build conductor options
	opts := []conductor.Option{
		conductor.WithWorkDir(f.server.config.WorkspaceRoot),
	}

	// Enable worktree if requested
	if worktree || f.useWorktree {
		opts = append(opts, conductor.WithUseWorktree(true))
	}

	// Create a new conductor for this task
	cond, err := conductor.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("create conductor: %w", err)
	}

	// Register standard providers and agents
	registration.RegisterStandardProviders(cond)
	if err := registration.RegisterStandardAgents(cond); err != nil {
		return nil, fmt.Errorf("register agents: %w", err)
	}

	// Initialize the conductor
	if err := cond.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize conductor: %w", err)
	}

	// Wrap in adapter to satisfy TaskConductor interface
	return taskrunner.NewConductorAdapter(cond), nil
}
