package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/valksor/kvelmo/pkg/agent"
)

// WebSocketConnection manages Codex via app-server with WebSocket transport.
// Unlike Claude, Codex HOSTS the WebSocket server and we connect as a client.
type WebSocketConnection struct {
	config   Config
	port     int
	threadID string

	cmd    *exec.Cmd
	cmdMu  sync.Mutex
	cmdErr error

	conn      *websocket.Conn
	connMu    sync.Mutex
	transport *wsTransport

	// State
	connected  atomic.Bool
	closed     atomic.Bool
	closedOnce sync.Once

	// Event channel
	events   chan agent.Event
	eventsMu sync.Mutex

	// Subagent tracker
	subagents *agent.SubagentTracker

	// Pending approval requests
	pendingApprovals   map[string]int64
	pendingApprovalsMu sync.Mutex

	// Track current turn
	turnActive atomic.Bool
}

// wsTransport adapts WebSocket for JSON-RPC transport.
type wsTransport struct {
	conn     *websocket.Conn
	connMu   sync.Mutex
	pending  map[int64]chan *rpcResponse
	pendingM sync.Mutex
	nextID   atomic.Int64

	notificationHandler func(method string, params json.RawMessage)
	requestHandler      func(method string, id int64, params json.RawMessage)

	closed  atomic.Bool
	closeCh chan struct{}
}

// NewWebSocketConnection creates a new WebSocket connection for Codex.
func NewWebSocketConnection(cfg Config) *WebSocketConnection {
	events := make(chan agent.Event, 100)

	return &WebSocketConnection{
		config:           cfg,
		port:             cfg.WebSocketPort,
		events:           events,
		subagents:        agent.NewSubagentTracker(events),
		pendingApprovals: make(map[string]int64),
	}
}

// Connect starts Codex app-server and connects via WebSocket.
func (w *WebSocketConnection) Connect(ctx context.Context) error {
	// Find a free port
	port, err := findFreePort(ctx)
	if err != nil {
		return fmt.Errorf("find free port: %w", err)
	}
	w.port = port

	// Launch Codex app-server
	if err := w.launchCodex(ctx); err != nil {
		return err
	}

	// Connect to Codex with exponential backoff retry
	wsURL := fmt.Sprintf("ws://127.0.0.1:%d", w.port)
	var conn *websocket.Conn
	var lastErr error

	// Retry with exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms (total ~3.1s)
	backoff := 100 * time.Millisecond
	maxAttempts := 5

	for attempt := range maxAttempts {
		if attempt > 0 {
			slog.Debug("retrying codex connection", "attempt", attempt+1, "backoff", backoff)
			time.Sleep(backoff)
			backoff *= 2
		}

		var dialErr error
		conn, _, dialErr = websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
		if dialErr == nil {
			break // Success
		}
		lastErr = dialErr

		// Check if context was cancelled
		if ctx.Err() != nil {
			w.killProcess()

			return fmt.Errorf("connect to codex: %w", ctx.Err())
		}
	}

	if conn == nil {
		w.killProcess()

		return fmt.Errorf("connect to codex after %d attempts: %w", maxAttempts, lastErr)
	}

	w.connMu.Lock()
	w.conn = conn
	w.connMu.Unlock()

	// Create transport
	w.transport = newWsTransport(conn)
	w.transport.notificationHandler = w.handleNotification
	w.transport.requestHandler = w.handleRequest
	go w.transport.readLoop(ctx)

	// Initialize JSON-RPC
	if err := w.initialize(ctx); err != nil {
		_ = w.Close()

		return fmt.Errorf("initialize: %w", err)
	}

	w.connected.Store(true)

	w.events <- agent.Event{
		Type:      agent.EventInit,
		Content:   "Codex WebSocket connected",
		Timestamp: time.Now(),
	}

	return nil
}

// findFreePort finds an available TCP port.
func findFreePort(ctx context.Context) (int, error) {
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close() //nolint:errcheck // Best-effort close

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected address type: %T", listener.Addr())
	}

	return addr.Port, nil
}

// launchCodex starts the Codex app-server process.
func (w *WebSocketConnection) launchCodex(ctx context.Context) error {
	w.cmdMu.Lock()
	defer w.cmdMu.Unlock()

	args := []string{
		"app-server",
		"--listen", fmt.Sprintf("ws://127.0.0.1:%d", w.port),
		// Multi-agent mode configured via ~/.codex/config.toml, not CLI flags
	}

	// Add configured arguments
	args = append(args, w.config.Args...)

	w.cmd = exec.CommandContext(ctx, w.config.Command[0], args...)

	if w.config.WorkDir != "" {
		w.cmd.Dir = w.config.WorkDir
	}

	// Set environment
	for k, v := range w.config.Environment {
		w.cmd.Env = append(w.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Capture stderr
	stderr, err := w.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := w.cmd.Start(); err != nil {
		return fmt.Errorf("start codex: %w", err)
	}

	// Log stderr in background
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				slog.Debug("codex stderr", "output", line)
			}
		}
	}()

	// Monitor process
	go func() {
		err := w.cmd.Wait()
		w.cmdMu.Lock()
		w.cmdErr = err
		w.cmdMu.Unlock()
		w.connected.Store(false)
	}()

	return nil
}

// initialize performs the JSON-RPC initialization handshake.
func (w *WebSocketConnection) initialize(ctx context.Context) error {
	// Step 1: initialize
	_, err := w.transport.Call(ctx, "initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    "kvelmo",
			"title":   "kvelmo",
			"version": "1.0.0",
		},
		"capabilities": map[string]any{
			"experimentalApi": false,
		},
	})
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	// Step 2: initialized notification
	if err := w.transport.Notify("initialized", map[string]any{}); err != nil {
		return fmt.Errorf("initialized: %w", err)
	}

	// Step 3: thread/start
	result, err := w.transport.Call(ctx, "thread/start", map[string]any{
		"model":          w.config.Model,
		"cwd":            w.config.WorkDir,
		"approvalPolicy": "always",
		"sandbox":        "workspace-write",
	})
	if err != nil {
		return fmt.Errorf("thread/start: %w", err)
	}

	// Extract thread ID
	var threadResult struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(result, &threadResult); err != nil {
		return fmt.Errorf("parse thread result: %w", err)
	}

	w.threadID = threadResult.Thread.ID
	slog.Debug("codex thread started", "threadId", w.threadID)

	return nil
}

// handleNotification processes incoming notifications.
func (w *WebSocketConnection) handleNotification(method string, params json.RawMessage) {
	switch method {
	case "item/agentMessage/delta":
		var delta struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(params, &delta); err == nil && delta.Text != "" {
			w.emitEvent(agent.Event{
				Type:      agent.EventStream,
				Content:   delta.Text,
				Timestamp: time.Now(),
			})
		}

	case "item/started":
		var item struct {
			ItemID string `json:"itemId"`
			Type   string `json:"type"`
		}
		if err := json.Unmarshal(params, &item); err == nil {
			toolName := item.Type
			switch item.Type {
			case "commandExecution":
				toolName = "Bash"
			case "fileChange":
				toolName = "Edit"
			}
			w.emitEvent(agent.Event{
				Type:      agent.EventToolUse,
				Content:   toolName,
				Timestamp: time.Now(),
			})
		}

	case "item/completed":
		var item struct {
			ItemID string `json:"itemId"`
			Type   string `json:"type"`
		}
		if err := json.Unmarshal(params, &item); err == nil {
			w.emitEvent(agent.Event{
				Type:      agent.EventToolResult,
				Content:   item.Type + " completed",
				Timestamp: time.Now(),
			})
		}

	case "turn/completed":
		w.turnActive.Store(false)
		w.emitEvent(agent.Event{
			Type:      agent.EventComplete,
			Timestamp: time.Now(),
		})

	case "turn/failed":
		w.turnActive.Store(false)
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(params, &failure)
		w.emitEvent(agent.Event{
			Type:      agent.EventError,
			Error:     failure.Error,
			Timestamp: time.Now(),
		})
	}
}

// handleRequest processes incoming requests.
func (w *WebSocketConnection) handleRequest(method string, id int64, params json.RawMessage) {
	switch method {
	case "item/commandExecution/requestApproval":
		w.handleCommandApproval(id, params)
	case "item/fileChange/requestApproval":
		w.handleFileChangeApproval(id, params)
	case "item/mcpToolCall/requestApproval":
		_ = w.transport.Respond(id, map[string]any{"decision": "accept"})
	default:
		_ = w.transport.Respond(id, map[string]any{"decision": "accept"})
	}
}

func (w *WebSocketConnection) handleCommandApproval(id int64, params json.RawMessage) {
	var req struct {
		ItemID  string   `json:"itemId"`
		Command []string `json:"command"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		slog.Warn("rejecting malformed command approval request", "error", err)
		_ = w.transport.Respond(id, map[string]any{"decision": "reject"})

		return
	}

	requestID := uuid.NewString()
	w.pendingApprovalsMu.Lock()
	w.pendingApprovals[requestID] = id
	w.pendingApprovalsMu.Unlock()

	command := ""
	if len(req.Command) > 0 {
		command = req.Command[0]
	}

	permReq := agent.PermissionRequest{
		ID:    requestID,
		Tool:  "Bash",
		Input: map[string]any{"command": command},
	}

	if w.config.PermissionHandler != nil {
		approved := w.config.PermissionHandler(permReq)
		_ = w.HandlePermission(requestID, approved)
	} else {
		w.emitEvent(agent.Event{
			Type:              agent.EventPermission,
			PermissionRequest: &permReq,
			Timestamp:         time.Now(),
		})
	}
}

func (w *WebSocketConnection) handleFileChangeApproval(id int64, params json.RawMessage) {
	var req struct {
		ItemID  string `json:"itemId"`
		Changes []struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		slog.Warn("rejecting malformed file change approval request", "error", err)
		_ = w.transport.Respond(id, map[string]any{"decision": "reject"})

		return
	}

	requestID := uuid.NewString()
	w.pendingApprovalsMu.Lock()
	w.pendingApprovals[requestID] = id
	w.pendingApprovalsMu.Unlock()

	paths := make([]string, len(req.Changes))
	for i, ch := range req.Changes {
		paths[i] = ch.Path
	}

	permReq := agent.PermissionRequest{
		ID:    requestID,
		Tool:  "Edit",
		Input: map[string]any{"paths": paths},
	}

	if w.config.PermissionHandler != nil {
		approved := w.config.PermissionHandler(permReq)
		_ = w.HandlePermission(requestID, approved)
	} else {
		w.emitEvent(agent.Event{
			Type:              agent.EventPermission,
			PermissionRequest: &permReq,
			Timestamp:         time.Now(),
		})
	}
}

func (w *WebSocketConnection) emitEvent(event agent.Event) {
	w.eventsMu.Lock()
	ch := w.events
	w.eventsMu.Unlock()

	select {
	case ch <- event:
	default:
	}
}

// Connected returns true if connected.
func (w *WebSocketConnection) Connected() bool {
	return w.connected.Load()
}

// SendPrompt sends a prompt via JSON-RPC.
func (w *WebSocketConnection) SendPrompt(ctx context.Context, prompt string) (<-chan agent.Event, error) {
	if !w.connected.Load() {
		return nil, errors.New("not connected")
	}

	if w.threadID == "" {
		return nil, errors.New("no thread started")
	}

	// Create new event channel
	w.eventsMu.Lock()
	w.events = make(chan agent.Event, 100)
	ch := w.events
	w.eventsMu.Unlock()
	w.subagents.SetEventChannel(ch)

	// Start turn
	w.turnActive.Store(true)
	_, err := w.transport.Call(ctx, "turn/start", map[string]any{
		"threadId": w.threadID,
		"message": map[string]any{
			"role":    "user",
			"content": prompt,
		},
	})
	if err != nil {
		w.turnActive.Store(false)

		return nil, fmt.Errorf("turn/start: %w", err)
	}

	// Return filtered channel
	filtered := make(chan agent.Event, 100)
	go func() {
		defer close(filtered)
		for {
			select {
			case event, ok := <-ch:
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

// HandlePermission responds to a permission request.
func (w *WebSocketConnection) HandlePermission(requestID string, approved bool) error {
	w.pendingApprovalsMu.Lock()
	rpcID, ok := w.pendingApprovals[requestID]
	if ok {
		delete(w.pendingApprovals, requestID)
	}
	w.pendingApprovalsMu.Unlock()

	if !ok {
		return fmt.Errorf("no pending approval for %s", requestID)
	}

	decision := "accept"
	if !approved {
		decision = "reject"
	}

	return w.transport.Respond(rpcID, map[string]any{"decision": decision})
}

func (w *WebSocketConnection) killProcess() {
	w.cmdMu.Lock()
	defer w.cmdMu.Unlock()

	if w.cmd != nil && w.cmd.Process != nil {
		_ = w.cmd.Process.Kill()
	}
}

// Close stops the connection.
func (w *WebSocketConnection) Close() error {
	w.closedOnce.Do(func() {
		w.closed.Store(true)
		w.connected.Store(false)

		if w.transport != nil {
			_ = w.transport.Close()
		}

		w.connMu.Lock()
		if w.conn != nil {
			_ = w.conn.Close()
		}
		w.connMu.Unlock()

		w.killProcess()

		close(w.events)
	})

	return nil
}

// --- WebSocket Transport ---

func newWsTransport(conn *websocket.Conn) *wsTransport {
	return &wsTransport{
		conn:    conn,
		pending: make(map[int64]chan *rpcResponse),
		closeCh: make(chan struct{}),
	}
}

func (t *wsTransport) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.closeCh:
			return
		default:
		}

		_, data, err := t.conn.ReadMessage()
		if err != nil {
			if !t.closed.Load() {
				slog.Debug("codex ws read error", "error", err)
			}

			return
		}

		var msg rpcMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		t.dispatch(msg)
	}
}

func (t *wsTransport) dispatch(msg rpcMessage) {
	// Response
	if msg.ID != nil && msg.Method == "" {
		t.pendingM.Lock()
		ch, ok := t.pending[*msg.ID]
		if ok {
			delete(t.pending, *msg.ID)
		}
		t.pendingM.Unlock()

		if ok {
			ch <- &rpcResponse{
				ID:     *msg.ID,
				Result: msg.Result,
				Error:  msg.Error,
			}
		}

		return
	}

	// Request
	if msg.ID != nil && msg.Method != "" {
		if t.requestHandler != nil {
			t.requestHandler(msg.Method, *msg.ID, msg.Params)
		}

		return
	}

	// Notification
	if msg.Method != "" {
		if t.notificationHandler != nil {
			t.notificationHandler(msg.Method, msg.Params)
		}
	}
}

func (t *wsTransport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if t.closed.Load() {
		return nil, errors.New("transport closed")
	}

	id := t.nextID.Add(1)
	req := rpcRequest{
		JsonRpc: "2.0",
		Method:  method,
		ID:      &id,
		Params:  params,
	}

	respCh := make(chan *rpcResponse, 1)
	t.pendingM.Lock()
	t.pending[id] = respCh
	t.pendingM.Unlock()

	defer func() {
		t.pendingM.Lock()
		delete(t.pending, id)
		t.pendingM.Unlock()
	}()

	if err := t.write(req); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	timeout := 60 * time.Second
	if method == "thread/start" || method == "thread/resume" {
		timeout = 120 * time.Second
	}

	select {
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		return resp.Result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("rpc timeout: %s", method)
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-t.closeCh:
		return nil, errors.New("transport closed")
	}
}

func (t *wsTransport) Notify(method string, params any) error {
	if t.closed.Load() {
		return errors.New("transport closed")
	}

	msg := rpcRequest{
		JsonRpc: "2.0",
		Method:  method,
		Params:  params,
	}

	return t.write(msg)
}

func (t *wsTransport) Respond(id int64, result any) error {
	if t.closed.Load() {
		return errors.New("transport closed")
	}

	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}

	return t.write(msg)
}

func (t *wsTransport) write(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	t.connMu.Lock()
	defer t.connMu.Unlock()

	return t.conn.WriteMessage(websocket.TextMessage, data)
}

func (t *wsTransport) Close() error {
	if t.closed.Swap(true) {
		return nil
	}
	close(t.closeCh)

	t.pendingM.Lock()
	for id, ch := range t.pending {
		close(ch)
		delete(t.pending, id)
	}
	t.pendingM.Unlock()

	return nil
}
