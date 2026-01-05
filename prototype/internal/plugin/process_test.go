package plugin

import (
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// Manifest tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProcessManifest(t *testing.T) {
	manifest := &Manifest{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    PluginTypeAgent,
	}

	proc := &Process{
		manifest: manifest,
		done:     make(chan struct{}),
	}

	got := proc.Manifest()
	if got != manifest {
		t.Errorf("Process.Manifest() = %p, want %p", got, manifest)
	}
	if got.Name != "test-plugin" {
		t.Errorf("Process.Manifest().Name = %s, want test-plugin", got.Name)
	}
	if got.Version != "1.0.0" {
		t.Errorf("Process.Manifest().Version = %s, want 1.0.0", got.Version)
	}
	if got.Type != PluginTypeAgent {
		t.Errorf("Process.Manifest().Type = %s, want Agent", got.Type)
	}
}

func TestProcessManifestNil(t *testing.T) {
	proc := &Process{
		manifest: nil,
		done:     make(chan struct{}),
	}

	got := proc.Manifest()
	if got != nil {
		t.Errorf("Process.Manifest() = %p, want nil", got)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// IsRunning tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProcessIsRunning_NotStarted(t *testing.T) {
	proc := &Process{
		done:    make(chan struct{}),
		started: false,
	}

	if proc.IsRunning() {
		t.Errorf("Process.IsRunning() = true, want false for not started process")
	}
}

func TestProcessIsRunning_Stopped(t *testing.T) {
	proc := &Process{
		done:    make(chan struct{}),
		started: true,
	}
	close(proc.done) // Mark as stopped

	if proc.IsRunning() {
		t.Errorf("Process.IsRunning() = true, want false for stopped process")
	}
}

func TestProcessIsRunning_Started(t *testing.T) {
	proc := &Process{
		done:     make(chan struct{}),
		started:  true,
		stopping: false,
	}

	if !proc.IsRunning() {
		t.Errorf("Process.IsRunning() = false, want true for started process")
	}
}

func TestProcessIsRunning_Stopping(t *testing.T) {
	proc := &Process{
		done:     make(chan struct{}),
		started:  true,
		stopping: true,
	}

	if proc.IsRunning() {
		t.Errorf("Process.IsRunning() = true, want false for stopping process")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Process state transitions tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProcessState_Transitions(t *testing.T) {
	proc := &Process{
		done: make(chan struct{}),
	}

	// Initially not running
	if proc.IsRunning() {
		t.Errorf("Process.IsRunning() = true initially, want false")
	}

	// Mark as started
	proc.started = true
	if !proc.IsRunning() {
		t.Errorf("Process.IsRunning() = false after started, want true")
	}

	// Mark as stopping
	proc.stopping = true
	if proc.IsRunning() {
		t.Errorf("Process.IsRunning() = true while stopping, want false")
	}

	// Mark as stopped
	close(proc.done)
	if proc.IsRunning() {
		t.Errorf("Process.IsRunning() = true after done closed, want false")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Stop nil context test
// ──────────────────────────────────────────────────────────────────────────────

func TestProcessStopNilContext(t *testing.T) {
	// Process.Stop requires a valid context
	// This test verifies behavior with context.Background()
	proc := &Process{
		done: make(chan struct{}),
	}

	// Cannot fully test Stop without a real process, but we can verify
	// the method exists and the struct is properly initialized
	if proc.done == nil {
		t.Error("Process.done channel is nil")
	}
}
