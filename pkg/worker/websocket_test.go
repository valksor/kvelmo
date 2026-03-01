package worker

import (
	"testing"
)

func TestNewWebSocketWorker(t *testing.T) {
	w := NewWebSocketWorker("worker-1", 0)
	if w == nil {
		t.Fatal("NewWebSocketWorker returned nil")
	}
	if w.ID != "worker-1" {
		t.Errorf("ID = %q, want worker-1", w.ID)
	}
}

func TestWebSocketWorker_Status_Initial(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	if w.Status() != StatusDisconnected {
		t.Errorf("Status() = %q, want %q", w.Status(), StatusDisconnected)
	}
}

func TestWebSocketWorker_IsAvailable_False(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	if w.IsAvailable() {
		t.Error("IsAvailable() should be false for new (disconnected) worker")
	}
}

func TestWebSocketWorker_IsWorking_False(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	if w.IsWorking() {
		t.Error("IsWorking() should be false for new worker")
	}
}

func TestWebSocketWorker_CurrentJob_Nil(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	if w.CurrentJob() != nil {
		t.Error("CurrentJob() should return nil for new worker")
	}
}

func TestWebSocketWorker_Events(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	ch := w.Events()
	if ch == nil {
		t.Error("Events() should return non-nil channel")
	}
}

func TestWebSocketWorker_SetPermissionHandler(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	called := false
	w.SetPermissionHandler(func(_ ControlRequest) bool {
		called = true

		return true
	})
	// Permission handler is set — call it to verify
	result := w.permissionHandler(ControlRequest{})
	if !called {
		t.Error("SetPermissionHandler: new handler was not called")
	}
	if !result {
		t.Error("SetPermissionHandler: handler should return true")
	}
}

func TestWebSocketWorker_Stop_Unstarted(t *testing.T) {
	// Stop on an unstarted worker should not panic
	w := NewWebSocketWorker("w1", 0)
	if err := w.Stop(); err != nil {
		t.Errorf("Stop() on unstarted worker returned error: %v", err)
	}
	if w.Status() != StatusDisconnected {
		t.Errorf("Status() after Stop = %q, want %q", w.Status(), StatusDisconnected)
	}
}

func TestWebSocketWorker_setStatus(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	w.setStatus(StatusAvailable)
	if w.Status() != StatusAvailable {
		t.Errorf("setStatus(Available): Status() = %q, want %q", w.Status(), StatusAvailable)
	}
	w.setStatus(StatusWorking)
	if !w.IsWorking() {
		t.Error("setStatus(Working): IsWorking() should be true")
	}
}

func TestHandleIncomingMessage_SystemInit(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	w.handleIncomingMessage(IncomingMessage{
		Type:      "system/init",
		SessionID: "sess-abc",
	})
	if w.SessionID != "sess-abc" {
		t.Errorf("SessionID = %q, want sess-abc", w.SessionID)
	}
	if w.Status() != StatusAvailable {
		t.Errorf("Status = %q, want %q after system/init", w.Status(), StatusAvailable)
	}
	select {
	case ev := <-w.events:
		if ev.Type != "worker_ready" {
			t.Errorf("event type = %q, want worker_ready", ev.Type)
		}
	default:
		t.Error("expected worker_ready event")
	}
}

func TestHandleIncomingMessage_StreamEvent_Content(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	w.handleIncomingMessage(IncomingMessage{
		Type:    "stream_event",
		Content: "hello token",
	})
	select {
	case ev := <-w.events:
		if ev.Type != "stream" {
			t.Errorf("event type = %q, want stream", ev.Type)
		}
		if ev.Content != "hello token" {
			t.Errorf("Content = %q, want hello token", ev.Content)
		}
	default:
		t.Error("expected stream event")
	}
}

func TestHandleIncomingMessage_StreamEvent_Delta(t *testing.T) {
	// When Content is empty, should fall back to Delta
	w := NewWebSocketWorker("w1", 0)
	w.handleIncomingMessage(IncomingMessage{
		Type:  "stream_event",
		Delta: "delta-token",
	})
	select {
	case ev := <-w.events:
		if ev.Content != "delta-token" {
			t.Errorf("Content = %q, want delta-token", ev.Content)
		}
	default:
		t.Error("expected stream event for delta")
	}
}

func TestHandleIncomingMessage_Assistant(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	w.handleIncomingMessage(IncomingMessage{
		Type: "assistant",
		Message: &AssistantMessage{
			Role:    "assistant",
			Content: "I can help",
		},
	})
	select {
	case ev := <-w.events:
		if ev.Type != "assistant" {
			t.Errorf("event type = %q, want assistant", ev.Type)
		}
		if ev.Content != "I can help" {
			t.Errorf("Content = %q, want I can help", ev.Content)
		}
	default:
		t.Error("expected assistant event")
	}
}

func TestHandleIncomingMessage_ControlRequest(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	approved := false
	w.permissionHandler = func(req ControlRequest) bool {
		approved = true

		return true
	}
	w.handleIncomingMessage(IncomingMessage{
		Type: "control_request",
		ControlRequest: &ControlRequest{
			ID:   "req-1",
			Tool: "Bash",
		},
	})
	if !approved {
		t.Error("permissionHandler should have been called")
	}
	select {
	case out := <-w.outgoing:
		if out.Type != "control_response" {
			t.Errorf("outgoing type = %q, want control_response", out.Type)
		}
		if out.ControlRequestID != "req-1" {
			t.Errorf("ControlRequestID = %q, want req-1", out.ControlRequestID)
		}
		if !out.Approved {
			t.Error("Approved should be true")
		}
	default:
		t.Error("expected control_response outgoing message")
	}
}

func TestHandleIncomingMessage_Result_Success(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	job := &Job{ID: "job-1"}
	w.currentJobMu.Lock()
	w.currentJob = job
	w.currentJobMu.Unlock()

	w.handleIncomingMessage(IncomingMessage{
		Type:    "result",
		Success: true,
	})
	select {
	case ev := <-w.events:
		if ev.Type != "job_completed" {
			t.Errorf("event type = %q, want job_completed", ev.Type)
		}
		if ev.JobID != "job-1" {
			t.Errorf("JobID = %q, want job-1", ev.JobID)
		}
	default:
		t.Error("expected job_completed event")
	}
	if w.Status() != StatusAvailable {
		t.Errorf("Status = %q, want %q after result", w.Status(), StatusAvailable)
	}
}

func TestHandleIncomingMessage_Result_Failure(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	job := &Job{ID: "job-2"}
	w.currentJobMu.Lock()
	w.currentJob = job
	w.currentJobMu.Unlock()

	w.handleIncomingMessage(IncomingMessage{
		Type:    "result",
		Success: false,
		Error:   "timeout exceeded",
	})
	select {
	case ev := <-w.events:
		if ev.Type != "job_failed" {
			t.Errorf("event type = %q, want job_failed", ev.Type)
		}
		if ev.Content != "timeout exceeded" {
			t.Errorf("Content = %q, want timeout exceeded", ev.Content)
		}
	default:
		t.Error("expected job_failed event")
	}
}

func TestHandleIncomingMessage_KeepAlive(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	w.handleIncomingMessage(IncomingMessage{Type: "keep_alive"})
	select {
	case ev := <-w.events:
		t.Errorf("keep_alive should not produce events, got %q", ev.Type)
	default:
		// expected: no event
	}
}

func TestWebSocketWorker_SendPrompt_NoSession(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	err := w.SendPrompt("hello")
	if err == nil {
		t.Error("SendPrompt() with empty SessionID should return error")
	}
}

func TestWebSocketWorker_ExecuteJob_NotAvailable(t *testing.T) {
	w := NewWebSocketWorker("w1", 0)
	// Status is Disconnected → not Available
	_, err := w.ExecuteJob(&Job{ID: "j1"})
	if err == nil {
		t.Error("ExecuteJob() with unavailable worker should return error")
	}
}
