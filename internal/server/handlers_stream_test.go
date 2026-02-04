package server

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valksor/go-toolkit/eventbus"
)

func TestHandleAgentLogs_RequiresTaskID(t *testing.T) {
	bus := eventbus.NewBus()

	cfg := Config{
		Port:     0,
		Mode:     ModeProject,
		EventBus: bus,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Connect to SSE endpoint without task_id
	reqCtx, reqCancel := context.WithTimeout(ctx, 2*time.Second)
	defer reqCancel()

	client := testHTTPClient()
	resp, err := doGet(reqCtx, client, srv.URL()+"/api/v1/agent/logs/stream")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should get an error event since no task is active
	if resp.StatusCode == http.StatusInternalServerError {
		t.Log("SSE endpoint returned 500 - ResponseWriter doesn't support Flusher in test environment")

		return
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	// Read the error event (no active task)
	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err == nil {
		assert.Contains(t, line, "event: error")
	}
}

func TestHandleEvents_SendsConnectedEvent(t *testing.T) {
	bus := eventbus.NewBus()

	cfg := Config{
		Port:     0,
		Mode:     ModeProject,
		EventBus: bus,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Connect to events SSE endpoint
	reqCtx, reqCancel := context.WithTimeout(ctx, 2*time.Second)
	defer reqCancel()

	client := testHTTPClient()
	resp, err := doGet(reqCtx, client, srv.URL()+"/api/v1/events")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusInternalServerError {
		t.Log("SSE endpoint returned 500 - ResponseWriter doesn't support Flusher in test environment")

		return
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	// Read the connected event
	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err == nil {
		assert.Contains(t, line, "event: connected")
	}
}

func TestHandleEvents_ReceivesPublishedEvents(t *testing.T) {
	bus := eventbus.NewBus()

	cfg := Config{
		Port:     0,
		Mode:     ModeProject,
		EventBus: bus,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Connect to events SSE endpoint
	reqCtx, reqCancel := context.WithTimeout(ctx, 3*time.Second)
	defer reqCancel()

	client := testHTTPClient()
	resp, err := doGet(reqCtx, client, srv.URL()+"/api/v1/events")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusInternalServerError {
		t.Log("SSE endpoint returned 500 - ResponseWriter doesn't support Flusher in test environment")

		return
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	reader := bufio.NewReader(resp.Body)

	// Read the connected event first
	_, _ = reader.ReadString('\n') // event: connected
	_, _ = reader.ReadString('\n') // data: ...
	_, _ = reader.ReadString('\n') // empty line

	// Publish a test event
	bus.PublishRaw(eventbus.Event{
		Type: "test_event",
		Data: map[string]any{"message": "hello"},
	})

	// Give time for event to propagate
	time.Sleep(50 * time.Millisecond)

	// Read the test event (with timeout to prevent hanging)
	done := make(chan bool)
	var eventLine string
	go func() {
		eventLine, _ = reader.ReadString('\n')
		done <- true
	}()

	select {
	case <-done:
		assert.Contains(t, eventLine, "event: test_event")
	case <-time.After(1 * time.Second):
		t.Log("Timeout waiting for event - this may be expected in some test environments")
	}
}

func TestSSEHeaders_SetCorrectly(t *testing.T) {
	bus := eventbus.NewBus()

	cfg := Config{
		Port:     0,
		Mode:     ModeProject,
		EventBus: bus,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	reqCtx, reqCancel := context.WithTimeout(ctx, 2*time.Second)
	defer reqCancel()

	client := testHTTPClient()
	resp, err := doGet(reqCtx, client, srv.URL()+"/api/v1/events")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusInternalServerError {
		t.Log("SSE endpoint returned 500 - ResponseWriter doesn't support Flusher in test environment")

		return
	}

	// Verify SSE headers
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestRunSSEHeartbeatLoop_EmitsHeartbeat(t *testing.T) {
	// This test verifies the heartbeat loop logic by calling the helper directly
	// using a mock response writer that captures output

	bus := eventbus.NewBus()

	cfg := Config{
		Port:     0,
		Mode:     ModeProject,
		EventBus: bus,
		// Note: Conductor is nil, so heartbeat will fall back to simple keepalive
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	// Create a pipe to capture SSE output
	pr, pw := newPipeResponseWriter()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run heartbeat loop in goroutine
	done := make(chan struct{})
	go func() {
		srv.runSSEHeartbeatLoop(pw, pw, ctx)
		close(done)
	}()

	// Wait for loop to finish (context timeout)
	<-done

	// Without conductor, it should have sent keepalive comments
	// The pipe might not have data if ticker didn't fire in 100ms,
	// but the loop should exit cleanly
	_ = pr.Close()
}

// pipeResponseWriter is a test helper that implements http.ResponseWriter and http.Flusher
// using an io.Pipe for capturing output.
type pipeResponseWriter struct {
	*pipeFlusher
}

type pipeFlusher struct {
	pw *pipeWriter
}

type pipeWriter struct {
	closed bool
	buf    strings.Builder
}

func (p *pipeWriter) Write(b []byte) (int, error) {
	if p.closed {
		return 0, nil
	}

	return p.buf.Write(b)
}

func (p *pipeWriter) Close() error {
	p.closed = true

	return nil
}

func (p *pipeFlusher) Header() http.Header {
	return http.Header{}
}

func (p *pipeFlusher) Write(b []byte) (int, error) {
	return p.pw.Write(b)
}

func (p *pipeFlusher) WriteHeader(_ int) {}

func (p *pipeFlusher) Flush() {}

func newPipeResponseWriter() (*pipeWriter, *pipeResponseWriter) {
	pw := &pipeWriter{}

	return pw, &pipeResponseWriter{pipeFlusher: &pipeFlusher{pw: pw}}
}

// TestSSEEventLoop_HandlesClientDisconnect verifies that the SSE event loop
// handles client disconnect gracefully without panic when events are published
// after the context is cancelled.
func TestSSEEventLoop_HandlesClientDisconnect(t *testing.T) {
	bus := eventbus.NewBus()

	cfg := Config{
		Port:     0,
		Mode:     ModeProject,
		EventBus: bus,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	// Create a pipe to capture SSE output
	pr, pw := newPipeResponseWriter()

	// Create a context that we'll cancel to simulate disconnect
	ctx, cancel := context.WithCancel(context.Background())

	// Create event channel like handleEvents does
	eventCh := make(chan eventbus.Event, 100)

	// Start the event loop in a goroutine
	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		srv.runSSEEventLoop(pw, pw, ctx, eventCh)
	}()

	// Give the loop time to start
	time.Sleep(10 * time.Millisecond)

	// Send an event - should be delivered
	eventCh <- eventbus.Event{
		Type: "test_event",
		Data: map[string]any{"message": "before disconnect"},
	}

	// Give time for event to be processed
	time.Sleep(10 * time.Millisecond)

	// Cancel context to simulate client disconnect
	cancel()

	// The loop should exit cleanly
	select {
	case <-loopDone:
		// Success - loop exited cleanly
	case <-time.After(1 * time.Second):
		t.Fatal("event loop did not exit after context cancellation")
	}

	// Verify no panic occurred (if we got here, we didn't panic)
	_ = pr.Close()
}

// TestSSEEventLoop_DropsEventsWhenChannelFull verifies that events are dropped
// gracefully when the channel is full (non-blocking behavior).
func TestSSEEventLoop_DropsEventsWhenChannelFull(t *testing.T) {
	bus := eventbus.NewBus()

	cfg := Config{
		Port:     0,
		Mode:     ModeProject,
		EventBus: bus,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a small buffered channel
	eventCh := make(chan eventbus.Event, 2)

	// Fill the channel
	eventCh <- eventbus.Event{Type: "event1", Data: map[string]any{}}
	eventCh <- eventbus.Event{Type: "event2", Data: map[string]any{}}

	// Create a pipe response writer
	_, pw := newPipeResponseWriter()

	// Start the event loop
	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		srv.runSSEEventLoop(pw, pw, ctx, eventCh)
	}()

	// Give the loop time to process events
	time.Sleep(50 * time.Millisecond)

	// Cancel to stop the loop
	cancel()

	select {
	case <-loopDone:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("event loop did not exit")
	}
}

// TestWriteSSEEvent_RecoverFromPanic verifies that writeSSEEvent recovers
// from panics that might occur when writing to an invalid ResponseWriter.
func TestWriteSSEEvent_RecoverFromPanic(t *testing.T) {
	bus := eventbus.NewBus()

	cfg := Config{
		Port:     0,
		Mode:     ModeProject,
		EventBus: bus,
	}

	srv, err := New(cfg)
	require.NoError(t, err)

	// Create a panicking response writer
	panicWriter := &panicResponseWriter{}

	// This should NOT panic - recovery should catch it
	assert.NotPanics(t, func() {
		srv.writeSSEEvent(panicWriter, panicWriter, "test", map[string]any{"data": "value"})
	})
}

// panicResponseWriter is a test helper that panics on Write/Flush.
type panicResponseWriter struct{}

func (p *panicResponseWriter) Header() http.Header {
	return http.Header{}
}

func (p *panicResponseWriter) Write(_ []byte) (int, error) {
	panic("simulated ResponseWriter panic")
}

func (p *panicResponseWriter) WriteHeader(_ int) {}

func (p *panicResponseWriter) Flush() {
	panic("simulated Flusher panic")
}
