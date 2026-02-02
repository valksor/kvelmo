package browser

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// WebSocketMonitor monitors WebSocket connections and frames for a browser tab.
type WebSocketMonitor struct {
	connections map[string]*WebSocketConnection
	frames      []WebSocketFrame
	mu          sync.RWMutex
	cancel      context.CancelFunc
	done        chan struct{}
	wg          sync.WaitGroup
}

// NewWebSocketMonitor creates a new WebSocket monitor.
func NewWebSocketMonitor() *WebSocketMonitor {
	return &WebSocketMonitor{
		connections: make(map[string]*WebSocketConnection),
	}
}

// Start begins monitoring WebSocket events for a page.
func (m *WebSocketMonitor) Start(ctx context.Context, page *rod.Page) error {
	// Network domain must be enabled for WebSocket events.
	enableNetwork := proto.NetworkEnable{}
	if err := enableNetwork.Call(page); err != nil {
		slog.Warn("CDP NetworkEnable failed for WebSocket monitor", "error", err)

		return fmt.Errorf("enable CDP network events: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.done = make(chan struct{})

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.listenForEvents(ctx, page)
	}()

	return nil
}

// Stop stops the WebSocket monitor and cleans up resources.
func (m *WebSocketMonitor) Stop() error {
	if m.cancel != nil {
		m.cancel()
	}

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		slog.Warn("websocket monitor goroutine leak detected", "timeout", "5s")

		return errors.New("websocket monitor stop timeout after 5s")
	}
}

// listenForEvents listens for WebSocket CDP events.
func (m *WebSocketMonitor) listenForEvents(ctx context.Context, page *rod.Page) {
	defer close(m.done)

	page.Context(ctx).EachEvent(
		func(e *proto.NetworkWebSocketCreated) {
			m.mu.Lock()
			m.connections[string(e.RequestID)] = &WebSocketConnection{
				ID:        string(e.RequestID),
				URL:       e.URL,
				Status:    "connecting",
				CreatedAt: time.Now(),
			}
			m.mu.Unlock()
		},
		func(e *proto.NetworkWebSocketHandshakeResponseReceived) {
			m.mu.Lock()
			if conn, exists := m.connections[string(e.RequestID)]; exists {
				conn.Status = "open"
			}
			m.mu.Unlock()
		},
		func(e *proto.NetworkWebSocketFrameSent) {
			m.mu.Lock()
			m.frames = append(m.frames, WebSocketFrame{
				ConnectionID: string(e.RequestID),
				Direction:    "sent",
				Data:         e.Response.PayloadData,
				Opcode:       int(e.Response.Opcode),
				Timestamp:    time.Now(),
			})
			m.mu.Unlock()
		},
		func(e *proto.NetworkWebSocketFrameReceived) {
			m.mu.Lock()
			m.frames = append(m.frames, WebSocketFrame{
				ConnectionID: string(e.RequestID),
				Direction:    "received",
				Data:         e.Response.PayloadData,
				Opcode:       int(e.Response.Opcode),
				Timestamp:    time.Now(),
			})
			m.mu.Unlock()
		},
		func(e *proto.NetworkWebSocketFrameError) {
			m.mu.Lock()
			m.frames = append(m.frames, WebSocketFrame{
				ConnectionID: string(e.RequestID),
				Direction:    "error",
				Error:        e.ErrorMessage,
				Timestamp:    time.Now(),
			})
			if conn, exists := m.connections[string(e.RequestID)]; exists {
				conn.Status = "error"
			}
			m.mu.Unlock()
		},
		func(e *proto.NetworkWebSocketClosed) {
			m.mu.Lock()
			if conn, exists := m.connections[string(e.RequestID)]; exists {
				conn.Status = "closed"
				conn.ClosedAt = time.Now()
			}
			m.mu.Unlock()
		},
	)()

	<-ctx.Done()
}

// GetFrames returns all captured WebSocket frames.
func (m *WebSocketMonitor) GetFrames() []WebSocketFrame {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]WebSocketFrame, len(m.frames))
	copy(result, m.frames)

	return result
}

// GetConnections returns all tracked WebSocket connections.
func (m *WebSocketMonitor) GetConnections() []WebSocketConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]WebSocketConnection, 0, len(m.connections))
	for _, conn := range m.connections {
		result = append(result, *conn)
	}

	return result
}

// GetFramesByConnection returns frames for a specific connection.
func (m *WebSocketMonitor) GetFramesByConnection(connectionID string) []WebSocketFrame {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []WebSocketFrame{}
	for _, frame := range m.frames {
		if frame.ConnectionID == connectionID {
			result = append(result, frame)
		}
	}

	return result
}

// GetFramesByPattern returns frames matching a data pattern.
func (m *WebSocketMonitor) GetFramesByPattern(pattern string) []WebSocketFrame {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []WebSocketFrame{}
	for _, frame := range m.frames {
		if contains(frame.Data, pattern) {
			result = append(result, frame)
		}
	}

	return result
}
