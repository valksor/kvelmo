package worker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: checkLocalOrigin,
}

// checkLocalOrigin validates that WebSocket connections come from localhost only.
// No Origin header (CLI client) or localhost origins are allowed.
func checkLocalOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // No origin = non-browser client (CLI)
	}

	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		origin == "http://localhost" ||
		origin == "http://127.0.0.1"
}

// WebSocketWorker manages a Claude CLI connection via WebSocket.
// Per flow_v2.md: "We act as WebSocket SERVER, Claude CLI connects to us.".
//
// Deprecated: WebSocket worker functionality has been moved to pkg/agent/claude/websocket.go
// and is now accessed through the agent.Agent interface.
// Use pool.AddAgentWorker() instead.
type WebSocketWorker struct {
	ID        string
	Port      int
	SessionID string

	server   *http.Server
	listener net.Listener
	conn     *websocket.Conn
	connMu   sync.Mutex

	cmd    *exec.Cmd
	cmdMu  sync.Mutex
	cmdErr error

	// Message channels
	incoming chan IncomingMessage
	outgoing chan OutgoingMessage
	events   chan Event

	// State
	ready     chan struct{}
	readyOnce sync.Once
	status    WorkerStatus
	statusMu  sync.RWMutex

	// Current job
	currentJob   *Job
	currentJobMu sync.RWMutex

	// Permission handler callback
	permissionHandler func(req ControlRequest) bool
}

// IncomingMessage represents a message from Claude CLI.
// Based on flow_v2.md WebSocket protocol.
type IncomingMessage struct {
	Type string `json:"type"`

	// system/init fields
	SessionID    string   `json:"session_id,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Tools        []Tool   `json:"tools,omitempty"`

	// stream_event fields
	Content string `json:"content,omitempty"`
	Delta   string `json:"delta,omitempty"`

	// assistant fields
	Message *AssistantMessage `json:"message,omitempty"`

	// control_request fields
	ControlRequest *ControlRequest `json:"control_request,omitempty"`

	// result fields
	Success bool   `json:"success,omitempty"`
	Error   string `json:"error,omitempty"`
}

// OutgoingMessage represents a message to Claude CLI.
type OutgoingMessage struct {
	Type string `json:"type"`

	// user message fields
	Message   *UserMessage `json:"message,omitempty"`
	SessionID string       `json:"session_id,omitempty"`

	// control_response fields
	ControlRequestID string `json:"control_request_id,omitempty"`
	Approved         bool   `json:"approved,omitempty"`
}

// Tool represents a tool available to Claude.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AssistantMessage represents an assistant response.
type AssistantMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// UserMessage represents a user message.
type UserMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ControlRequest represents a permission request from Claude.
type ControlRequest struct {
	ID     string          `json:"id"`
	Tool   string          `json:"tool"`
	Input  json.RawMessage `json:"input"`
	Action string          `json:"action,omitempty"`
}

// NewWebSocketWorker creates a new WebSocket-based worker.
func NewWebSocketWorker(id string, port int) *WebSocketWorker {
	return &WebSocketWorker{
		ID:       id,
		Port:     port,
		incoming: make(chan IncomingMessage, 100),
		outgoing: make(chan OutgoingMessage, 100),
		events:   make(chan Event, 100),
		ready:    make(chan struct{}),
		status:   StatusDisconnected,
		permissionHandler: func(req ControlRequest) bool {
			// Default: auto-approve safe tools
			safeTols := map[string]bool{
				"read_file": true, "glob": true, "grep": true,
				"list_dir": true, "search": true,
			}

			return safeTols[req.Tool]
		},
	}
}

// Start starts the WebSocket server and launches Claude CLI.
func (w *WebSocketWorker) Start(ctx context.Context) error {
	// Create listener first to ensure port is available
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", w.Port)) //nolint:noctx // Context cancellation handled via server shutdown
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", w.Port, err)
	}
	w.listener = listener

	// Get actual port (in case 0 was specified)
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		w.Port = tcpAddr.Port
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", w.handleConnection)

	w.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start HTTP server
	go func() {
		if err := w.server.Serve(listener); err != http.ErrServerClosed {
			w.setStatus(StatusDisconnected)
		}
	}()

	// Start outgoing message sender
	go w.sendLoop(ctx)

	// Launch Claude CLI
	if err := w.launchClaude(ctx); err != nil {
		_ = w.Stop()

		return err
	}

	// Wait for connection or timeout
	select {
	case <-w.ready:
		w.setStatus(StatusAvailable)

		return nil
	case <-time.After(30 * time.Second):
		_ = w.Stop()

		return errors.New("timeout waiting for Claude CLI connection")
	case <-ctx.Done():
		_ = w.Stop()

		return ctx.Err()
	}
}

// launchClaude launches the Claude CLI process.
func (w *WebSocketWorker) launchClaude(ctx context.Context) error {
	w.cmdMu.Lock()
	defer w.cmdMu.Unlock()

	// Build command based on flow_v2.md specification
	w.cmd = exec.CommandContext(ctx, "claude",
		"--sdk-url", fmt.Sprintf("ws://127.0.0.1:%d", w.Port),
		"--print",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
	)

	// Capture stderr for debugging
	stderr, err := w.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := w.cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}

	// Log stderr in background
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) != "" {
				w.events <- Event{
					Type:    "worker_stderr",
					Content: line,
				}
			}
		}
	}()

	// Wait for process completion in background
	go func() {
		err := w.cmd.Wait()
		w.cmdMu.Lock()
		w.cmdErr = err
		w.cmdMu.Unlock()

		if err != nil {
			w.events <- Event{
				Type:    "worker_error",
				Content: fmt.Sprintf("Claude CLI exited: %v", err),
			}
		}
		w.setStatus(StatusDisconnected)
	}()

	return nil
}

// handleConnection handles incoming WebSocket connections.
func (w *WebSocketWorker) handleConnection(rw http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		return
	}

	w.connMu.Lock()
	w.conn = conn
	w.connMu.Unlock()

	// Signal ready
	w.readyOnce.Do(func() {
		close(w.ready)
	})

	// Read messages
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			w.setStatus(StatusDisconnected)

			return
		}

		// Handle NDJSON - may have multiple JSON objects per message
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			var msg IncomingMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}

			w.handleIncomingMessage(msg)
		}
	}
}

// handleIncomingMessage processes an incoming message from Claude.
func (w *WebSocketWorker) handleIncomingMessage(msg IncomingMessage) {
	switch msg.Type {
	case "system/init":
		w.SessionID = msg.SessionID
		w.setStatus(StatusAvailable)
		w.events <- Event{
			Type:    "worker_ready",
			Content: fmt.Sprintf("Worker %s ready (session: %s)", w.ID, w.SessionID),
		}

	case "stream_event":
		// Token-by-token streaming output
		content := msg.Content
		if content == "" {
			content = msg.Delta
		}
		w.events <- Event{
			Type:    "stream",
			Content: content,
		}

	case "assistant":
		// Full assistant response
		if msg.Message != nil {
			w.events <- Event{
				Type:    "assistant",
				Content: msg.Message.Content,
			}
		}

	case "control_request":
		// Permission request - evaluate and respond
		if msg.ControlRequest != nil {
			approved := w.permissionHandler(*msg.ControlRequest)
			w.outgoing <- OutgoingMessage{
				Type:             "control_response",
				ControlRequestID: msg.ControlRequest.ID,
				Approved:         approved,
			}
		}

	case "result":
		// Task complete
		w.currentJobMu.Lock()
		job := w.currentJob
		w.currentJob = nil
		w.currentJobMu.Unlock()

		if job != nil {
			if msg.Success {
				w.events <- Event{
					Type:  "job_completed",
					JobID: job.ID,
				}
			} else {
				w.events <- Event{
					Type:    "job_failed",
					JobID:   job.ID,
					Content: msg.Error,
				}
			}
		}
		w.setStatus(StatusAvailable)

	case "keep_alive":
		// Heartbeat - no action needed
	}
}

// sendLoop sends outgoing messages to Claude.
func (w *WebSocketWorker) sendLoop(ctx context.Context) {
	for {
		select {
		case msg := <-w.outgoing:
			w.connMu.Lock()
			if w.conn != nil {
				data, err := json.Marshal(msg)
				if err != nil {
					continue
				}
				// Send as NDJSON
				data = append(data, '\n')
				_ = w.conn.WriteMessage(websocket.TextMessage, data)
			}
			w.connMu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// SendPrompt sends a user prompt to Claude.
func (w *WebSocketWorker) SendPrompt(prompt string) error {
	if w.SessionID == "" {
		return errors.New("worker not ready (no session)")
	}

	w.setStatus(StatusWorking)

	w.outgoing <- OutgoingMessage{
		Type:      "user",
		SessionID: w.SessionID,
		Message: &UserMessage{
			Role:    "user",
			Content: prompt,
		},
	}

	return nil
}

// ExecuteJob executes a job and returns a stream of events.
func (w *WebSocketWorker) ExecuteJob(job *Job) (<-chan Event, error) {
	if w.Status() != StatusAvailable {
		return nil, fmt.Errorf("worker not available (status: %s)", w.Status())
	}

	w.currentJobMu.Lock()
	w.currentJob = job
	w.currentJobMu.Unlock()

	job.WorkerID = w.ID
	job.Status = JobStatusInProgress

	// Send prompt
	if err := w.SendPrompt(job.Prompt); err != nil {
		return nil, err
	}

	// Return filtered event stream
	filtered := make(chan Event, 100)
	go func() {
		defer close(filtered)
		for event := range w.events {
			filtered <- event
			if event.Type == "job_completed" || event.Type == "job_failed" {
				return
			}
		}
	}()

	return filtered, nil
}

// SetPermissionHandler sets the permission evaluation callback.
func (w *WebSocketWorker) SetPermissionHandler(handler func(req ControlRequest) bool) {
	w.permissionHandler = handler
}

// Events returns the event channel.
func (w *WebSocketWorker) Events() <-chan Event {
	return w.events
}

// Stop stops the worker.
func (w *WebSocketWorker) Stop() error {
	// Kill Claude process
	w.cmdMu.Lock()
	if w.cmd != nil && w.cmd.Process != nil {
		_ = w.cmd.Process.Kill()
	}
	w.cmdMu.Unlock()

	// Close WebSocket connection
	w.connMu.Lock()
	if w.conn != nil {
		_ = w.conn.Close()
	}
	w.connMu.Unlock()

	// Stop HTTP server
	if w.server != nil {
		_ = w.server.Close()
	}

	w.setStatus(StatusDisconnected)

	return nil
}

// Status returns the worker's current status.
func (w *WebSocketWorker) Status() WorkerStatus {
	w.statusMu.RLock()
	defer w.statusMu.RUnlock()

	return w.status
}

// IsAvailable returns true if the worker can accept jobs.
func (w *WebSocketWorker) IsAvailable() bool {
	return w.Status() == StatusAvailable
}

// IsWorking returns true if the worker is currently executing a job.
func (w *WebSocketWorker) IsWorking() bool {
	return w.Status() == StatusWorking
}

func (w *WebSocketWorker) setStatus(s WorkerStatus) {
	w.statusMu.Lock()
	defer w.statusMu.Unlock()
	w.status = s
}

// CurrentJob returns the job currently being executed.
func (w *WebSocketWorker) CurrentJob() *Job {
	w.currentJobMu.RLock()
	defer w.currentJobMu.RUnlock()

	return w.currentJob
}
