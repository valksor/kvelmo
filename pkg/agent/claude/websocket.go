package claude

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
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/valksor/kvelmo/pkg/agent"
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

// WebSocketConnection manages a Claude CLI connection via WebSocket.
// Per flow_v2.md: "We act as WebSocket SERVER, Claude CLI connects to us.".
type WebSocketConnection struct {
	config    Config
	port      int
	sessionID string

	server   *http.Server
	listener net.Listener
	conn     *websocket.Conn
	connMu   sync.Mutex

	cmd    *exec.Cmd
	cmdMu  sync.Mutex
	cmdErr error

	// Message channels
	outgoing chan outgoingMessage
	events   chan agent.Event

	// State
	ready      chan struct{}
	readyOnce  sync.Once
	connected  atomic.Bool
	closed     atomic.Bool
	closedOnce sync.Once
}

// Incoming message types from Claude CLI.
type incomingMessage struct {
	Type string `json:"type"`

	// system/init fields
	SessionID    string   `json:"session_id,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Tools        []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"tools,omitempty"`

	// stream_event fields
	Content string `json:"content,omitempty"`
	Delta   string `json:"delta,omitempty"`

	// assistant fields
	Message *struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message,omitempty"`

	// control_request fields
	ControlRequest *struct {
		ID     string          `json:"id"`
		Tool   string          `json:"tool"`
		Input  json.RawMessage `json:"input"`
		Action string          `json:"action,omitempty"`
	} `json:"control_request,omitempty"`

	// result fields
	Success bool   `json:"success,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Outgoing message types to Claude CLI.
type outgoingMessage struct {
	Type string `json:"type"`

	// user message fields
	Message *struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message,omitempty"`
	SessionID string `json:"session_id,omitempty"`

	// control_response fields
	ControlRequestID string `json:"control_request_id,omitempty"`
	Approved         bool   `json:"approved,omitempty"`
}

// NewWebSocketConnection creates a new WebSocket connection for Claude.
func NewWebSocketConnection(cfg Config) *WebSocketConnection {
	return &WebSocketConnection{
		config:   cfg,
		port:     cfg.WebSocketPort,
		outgoing: make(chan outgoingMessage, 100),
		events:   make(chan agent.Event, 100),
		ready:    make(chan struct{}),
	}
}

// Connect starts the WebSocket server and launches Claude CLI.
func (w *WebSocketConnection) Connect(ctx context.Context) error {
	// Create listener
	addr := fmt.Sprintf("127.0.0.1:%d", w.port)
	listener, err := net.Listen("tcp", addr) //nolint:noctx // Context cancellation handled via server shutdown
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	w.listener = listener

	// Get actual port (if 0 was specified)
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		w.port = tcpAddr.Port
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
			w.connected.Store(false)
		}
	}()

	// Start outgoing message sender
	go w.sendLoop(ctx)

	// Launch Claude CLI
	if err := w.launchClaude(ctx); err != nil {
		_ = w.Close()

		return err
	}

	// Wait for connection or timeout
	select {
	case <-w.ready:
		w.connected.Store(true)

		return nil
	case <-time.After(30 * time.Second):
		_ = w.Close()

		return errors.New("timeout waiting for Claude CLI connection")
	case <-ctx.Done():
		_ = w.Close()

		return ctx.Err()
	}
}

// launchClaude starts the Claude CLI process with --sdk-url.
func (w *WebSocketConnection) launchClaude(ctx context.Context) error {
	w.cmdMu.Lock()
	defer w.cmdMu.Unlock()

	args := w.buildArgs()
	w.cmd = exec.CommandContext(ctx, w.config.Command[0], args...)

	if w.config.WorkDir != "" {
		w.cmd.Dir = w.config.WorkDir
	}

	// Set environment
	for k, v := range w.config.Environment {
		w.cmd.Env = append(w.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

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
				w.events <- agent.Event{
					Type:      agent.EventError,
					Content:   line,
					Timestamp: time.Now(),
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
		w.connected.Store(false)
	}()

	return nil
}

// buildArgs constructs CLI arguments for Claude with WebSocket.
func (w *WebSocketConnection) buildArgs() []string {
	args := []string{
		"--sdk-url", fmt.Sprintf("ws://127.0.0.1:%d", w.port),
		"--print",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
	}

	// Add configured arguments
	args = append(args, w.config.Args...)

	// Add model if specified
	if w.config.Model != "" {
		args = append(args, "--model", w.config.Model)
	}

	return args
}

// handleConnection handles incoming WebSocket connections from Claude CLI.
func (w *WebSocketConnection) handleConnection(rw http.ResponseWriter, r *http.Request) {
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
			w.connected.Store(false)

			return
		}

		// Handle NDJSON - may have multiple JSON objects per message
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			var msg incomingMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}

			w.handleIncomingMessage(msg)
		}
	}
}

// handleIncomingMessage processes messages from Claude CLI.
func (w *WebSocketConnection) handleIncomingMessage(msg incomingMessage) {
	switch msg.Type {
	case "system/init":
		w.sessionID = msg.SessionID
		w.connected.Store(true)
		w.events <- agent.Event{
			Type:      agent.EventInit,
			Content:   "Session initialized: " + w.sessionID,
			Timestamp: time.Now(),
		}

	case "stream_event":
		content := msg.Content
		if content == "" {
			content = msg.Delta
		}
		w.events <- agent.Event{
			Type:      agent.EventStream,
			Content:   content,
			Timestamp: time.Now(),
		}

	case "assistant":
		if msg.Message != nil {
			w.events <- agent.Event{
				Type:      agent.EventAssistant,
				Content:   msg.Message.Content,
				Timestamp: time.Now(),
			}
		}

	case "control_request":
		if msg.ControlRequest != nil {
			var input map[string]any
			_ = json.Unmarshal(msg.ControlRequest.Input, &input)

			req := agent.PermissionRequest{
				ID:     msg.ControlRequest.ID,
				Tool:   msg.ControlRequest.Tool,
				Input:  input,
				Action: msg.ControlRequest.Action,
			}

			// Auto-handle with permission handler
			if w.config.PermissionHandler != nil {
				approved := w.config.PermissionHandler(req)
				_ = w.HandlePermission(req.ID, approved)
			} else {
				// Send event for external handling
				w.events <- agent.Event{
					Type:              agent.EventPermission,
					PermissionRequest: &req,
					Timestamp:         time.Now(),
				}
			}
		}

	case "result":
		if msg.Success {
			w.events <- agent.Event{
				Type:      agent.EventComplete,
				Timestamp: time.Now(),
			}
		} else {
			w.events <- agent.Event{
				Type:      agent.EventError,
				Error:     msg.Error,
				Timestamp: time.Now(),
			}
		}

	case "keep_alive":
		// Heartbeat - no action needed
	}
}

// sendLoop sends outgoing messages to Claude CLI.
func (w *WebSocketConnection) sendLoop(ctx context.Context) {
	for {
		select {
		case msg := <-w.outgoing:
			w.connMu.Lock()
			if w.conn != nil {
				data, err := json.Marshal(msg)
				if err == nil {
					data = append(data, '\n')
					_ = w.conn.WriteMessage(websocket.TextMessage, data)
				}
			}
			w.connMu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// Connected returns true if connected.
func (w *WebSocketConnection) Connected() bool {
	return w.connected.Load()
}

// SendPrompt sends a user prompt and returns the event stream.
func (w *WebSocketConnection) SendPrompt(ctx context.Context, prompt string) (<-chan agent.Event, error) {
	if w.sessionID == "" {
		return nil, errors.New("not connected (no session)")
	}

	w.outgoing <- outgoingMessage{
		Type:      "user",
		SessionID: w.sessionID,
		Message: &struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			Role:    "user",
			Content: prompt,
		},
	}

	// Return filtered event stream
	filtered := make(chan agent.Event, 100)
	go func() {
		defer close(filtered)
		for {
			select {
			case event, ok := <-w.events:
				if !ok {
					return
				}
				filtered <- event
				if event.Type == agent.EventComplete || event.Type == agent.EventError {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return filtered, nil
}

// HandlePermission sends a permission response.
func (w *WebSocketConnection) HandlePermission(requestID string, approved bool) error {
	w.outgoing <- outgoingMessage{
		Type:             "control_response",
		ControlRequestID: requestID,
		Approved:         approved,
	}

	return nil
}

// Close stops the connection.
func (w *WebSocketConnection) Close() error {
	w.closedOnce.Do(func() {
		w.closed.Store(true)
		w.connected.Store(false)

		// Kill Claude process
		w.cmdMu.Lock()
		if w.cmd != nil && w.cmd.Process != nil {
			_ = w.cmd.Process.Kill()
		}
		w.cmdMu.Unlock()

		// Close WebSocket
		w.connMu.Lock()
		if w.conn != nil {
			_ = w.conn.Close()
		}
		w.connMu.Unlock()

		// Stop HTTP server
		if w.server != nil {
			_ = w.server.Close()
		}

		close(w.events)
	})

	return nil
}
