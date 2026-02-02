package browser

import (
	"testing"
	"time"
)

// TestWebSocketMonitor tests WebSocket frame monitoring functionality.
func TestWebSocketMonitor(t *testing.T) {
	t.Run("NewWebSocketMonitor", func(t *testing.T) {
		mon := NewWebSocketMonitor()
		if mon.connections == nil {
			t.Error("connections map is not initialized")
		}
		if len(mon.GetFrames()) != 0 {
			t.Error("new monitor should have no frames")
		}
		if len(mon.GetConnections()) != 0 {
			t.Error("new monitor should have no connections")
		}
	})

	t.Run("GetFrames", func(t *testing.T) {
		mon := NewWebSocketMonitor()

		// Simulate frames being added via event handlers
		mon.mu.Lock()
		mon.frames = []WebSocketFrame{
			{ConnectionID: "ws-1", Direction: "sent", Data: "hello", Timestamp: time.Now()},
			{ConnectionID: "ws-1", Direction: "received", Data: "world", Timestamp: time.Now()},
			{ConnectionID: "ws-2", Direction: "sent", Data: "ping", Timestamp: time.Now()},
		}
		mon.mu.Unlock()

		frames := mon.GetFrames()
		if len(frames) != 3 {
			t.Errorf("got %d frames, want 3", len(frames))
		}

		// Verify returned slice is a copy (modifying it doesn't affect monitor)
		frames[0].Data = "modified"
		original := mon.GetFrames()
		if original[0].Data == "modified" {
			t.Error("GetFrames() should return a copy, not a reference")
		}
	})

	t.Run("GetConnections", func(t *testing.T) {
		mon := NewWebSocketMonitor()

		now := time.Now()
		mon.mu.Lock()
		mon.connections["ws-1"] = &WebSocketConnection{
			ID:        "ws-1",
			URL:       "ws://localhost:8080/chat",
			Status:    "open",
			CreatedAt: now,
		}
		mon.connections["ws-2"] = &WebSocketConnection{
			ID:        "ws-2",
			URL:       "ws://localhost:8080/events",
			Status:    "closed",
			CreatedAt: now,
			ClosedAt:  now.Add(5 * time.Second),
		}
		mon.mu.Unlock()

		connections := mon.GetConnections()
		if len(connections) != 2 {
			t.Errorf("got %d connections, want 2", len(connections))
		}

		// Verify status values
		statusMap := make(map[string]string)
		for _, conn := range connections {
			statusMap[conn.ID] = conn.Status
		}
		if statusMap["ws-1"] != "open" {
			t.Errorf("ws-1 status = %q, want 'open'", statusMap["ws-1"])
		}
		if statusMap["ws-2"] != "closed" {
			t.Errorf("ws-2 status = %q, want 'closed'", statusMap["ws-2"])
		}
	})

	t.Run("GetFramesByConnection", func(t *testing.T) {
		mon := NewWebSocketMonitor()

		mon.mu.Lock()
		mon.frames = []WebSocketFrame{
			{ConnectionID: "ws-1", Direction: "sent", Data: "msg1"},
			{ConnectionID: "ws-2", Direction: "sent", Data: "msg2"},
			{ConnectionID: "ws-1", Direction: "received", Data: "msg3"},
			{ConnectionID: "ws-2", Direction: "received", Data: "msg4"},
			{ConnectionID: "ws-1", Direction: "sent", Data: "msg5"},
		}
		mon.mu.Unlock()

		tests := []struct {
			name         string
			connectionID string
			wantCount    int
		}{
			{"ws-1 frames", "ws-1", 3},
			{"ws-2 frames", "ws-2", 2},
			{"non-existent connection", "ws-99", 0},
			{"empty connection ID", "", 0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				frames := mon.GetFramesByConnection(tt.connectionID)
				if len(frames) != tt.wantCount {
					t.Errorf("got %d frames, want %d", len(frames), tt.wantCount)
				}

				// Verify all returned frames belong to correct connection
				for _, frame := range frames {
					if frame.ConnectionID != tt.connectionID {
						t.Errorf("frame ConnectionID = %q, want %q", frame.ConnectionID, tt.connectionID)
					}
				}
			})
		}
	})

	t.Run("GetFramesByPattern", func(t *testing.T) {
		mon := NewWebSocketMonitor()

		mon.mu.Lock()
		mon.frames = []WebSocketFrame{
			{ConnectionID: "ws-1", Data: `{"type":"message","text":"hello"}`},
			{ConnectionID: "ws-1", Data: `{"type":"ping"}`},
			{ConnectionID: "ws-1", Data: `{"type":"message","text":"goodbye"}`},
			{ConnectionID: "ws-2", Data: `{"type":"error","code":500}`},
		}
		mon.mu.Unlock()

		tests := []struct {
			name      string
			pattern   string
			wantCount int
		}{
			{"match message type", "message", 2},
			{"match ping type", "ping", 1},
			{"match error type", "error", 1},
			{"no match", "foobar", 0},
			{"case insensitive - upper", "MESSAGE", 2},
			{"partial match", "hell", 1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				frames := mon.GetFramesByPattern(tt.pattern)
				if len(frames) != tt.wantCount {
					t.Errorf("got %d frames, want %d", len(frames), tt.wantCount)
				}
			})
		}
	})

	t.Run("ErrorFrames", func(t *testing.T) {
		mon := NewWebSocketMonitor()

		mon.mu.Lock()
		mon.frames = []WebSocketFrame{
			{ConnectionID: "ws-1", Direction: "sent", Data: "hello"},
			{ConnectionID: "ws-1", Direction: "error", Error: "connection reset"},
			{ConnectionID: "ws-1", Direction: "received", Data: "world"},
		}
		mon.mu.Unlock()

		frames := mon.GetFrames()
		if len(frames) != 3 {
			t.Fatalf("got %d frames, want 3", len(frames))
		}

		errorFrame := frames[1]
		if errorFrame.Direction != "error" {
			t.Errorf("Direction = %q, want 'error'", errorFrame.Direction)
		}
		if errorFrame.Error != "connection reset" {
			t.Errorf("Error = %q, want 'connection reset'", errorFrame.Error)
		}
	})

	t.Run("ConnectionStatusTransitions", func(t *testing.T) {
		mon := NewWebSocketMonitor()

		now := time.Now()

		// Simulate lifecycle: connecting -> open -> closed
		mon.mu.Lock()
		mon.connections["ws-1"] = &WebSocketConnection{
			ID:        "ws-1",
			URL:       "ws://localhost:8080/chat",
			Status:    "connecting",
			CreatedAt: now,
		}
		mon.mu.Unlock()

		conns := mon.GetConnections()
		if conns[0].Status != "connecting" {
			t.Errorf("initial status = %q, want 'connecting'", conns[0].Status)
		}

		mon.mu.Lock()
		mon.connections["ws-1"].Status = "open"
		mon.mu.Unlock()

		conns = mon.GetConnections()
		if conns[0].Status != "open" {
			t.Errorf("after handshake status = %q, want 'open'", conns[0].Status)
		}

		mon.mu.Lock()
		mon.connections["ws-1"].Status = "closed"
		mon.connections["ws-1"].ClosedAt = now.Add(10 * time.Second)
		mon.mu.Unlock()

		conns = mon.GetConnections()
		if conns[0].Status != "closed" {
			t.Errorf("after close status = %q, want 'closed'", conns[0].Status)
		}
		if conns[0].ClosedAt.IsZero() {
			t.Error("ClosedAt should be set after closing")
		}
	})

	t.Run("ThreadSafety", func(t *testing.T) {
		mon := NewWebSocketMonitor()

		done := make(chan bool)
		for range 10 {
			go func() {
				mon.mu.Lock()
				mon.frames = append(mon.frames, WebSocketFrame{
					ConnectionID: "ws-1",
					Direction:    "sent",
					Data:         "message",
					Timestamp:    time.Now(),
				})
				mon.mu.Unlock()
				done <- true
			}()
		}

		// Wait for all goroutines
		for range 10 {
			<-done
		}

		frames := mon.GetFrames()
		if len(frames) != 10 {
			t.Errorf("got %d frames, want 10", len(frames))
		}
	})
}
