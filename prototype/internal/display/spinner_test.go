package display

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewSpinner(t *testing.T) {
	s := NewSpinner("Test message")

	if s == nil {
		t.Fatal("NewSpinner() returned nil")
	}

	if s.message != "Test message" {
		t.Errorf("NewSpinner() message = %q, want %q", s.message, "Test message")
	}

	if len(s.frames) == 0 {
		t.Error("NewSpinner() frames should be set")
	}

	if s.delay != 80*time.Millisecond {
		t.Errorf("NewSpinner() delay = %v, want %v", s.delay, 80*time.Millisecond)
	}

	if s.running {
		t.Error("NewSpinner() running should be false initially")
	}
}

func TestSpinnerStartStop(t *testing.T) {
	s := NewSpinner("Starting...")

	// Start the spinner
	s.Start()

	// Give the goroutine a moment to start
	time.Sleep(10 * time.Millisecond)

	if !s.running {
		t.Error("Start() should set running to true")
	}

	// Stop the spinner
	s.Stop()

	if s.running {
		t.Error("Stop() should set running to false")
	}
}

func TestSpinnerMultipleStarts(t *testing.T) {
	s := NewSpinner("Test")

	// Start multiple times
	s.Start()
	time.Sleep(10 * time.Millisecond)
	s.Start()
	time.Sleep(10 * time.Millisecond)
	s.Start()

	// Should not cause multiple goroutines
	// Stop should work without hanging
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		s.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		// Success - stopped cleanly
	case <-ctx.Done():
		t.Fatal("Stop() hung - likely multiple goroutines running")
	}
}

func TestSpinnerStopWithSuccess(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner("Loading...")
	s.writer = &buf

	s.Start()
	time.Sleep(10 * time.Millisecond)
	s.StopWithSuccess("Done!")

	output := buf.String()

	if output == "" {
		t.Error("StopWithSuccess() should write output")
	}

	// Should contain the success message
	if !strings.Contains(output, "Done!") {
		t.Errorf("StopWithSuccess() output should contain success message, got: %s", output)
	}
}

func TestSpinnerStopWithError(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner("Loading...")
	s.writer = &buf

	s.Start()
	time.Sleep(10 * time.Millisecond)
	s.StopWithError("Failed!")

	output := buf.String()

	if output == "" {
		t.Error("StopWithError() should write output")
	}

	// Should contain the error message
	if !strings.Contains(output, "Failed!") {
		t.Errorf("StopWithError() output should contain error message, got: %s", output)
	}
}

func TestSpinnerStopWithWarning(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner("Loading...")
	s.writer = &buf

	s.Start()
	time.Sleep(10 * time.Millisecond)
	s.StopWithWarning("Warning!")

	output := buf.String()

	if output == "" {
		t.Error("StopWithWarning() should write output")
	}

	// Should contain the warning message
	if !strings.Contains(output, "Warning!") {
		t.Errorf("StopWithWarning() output should contain warning message, got: %s", output)
	}
}

func TestSpinnerUpdateMessage(t *testing.T) {
	s := NewSpinner("Original message")

	s.UpdateMessage("New message")

	if s.message != "New message" {
		t.Errorf("UpdateMessage() = %q, want %q", s.message, "New message")
	}
}

func TestSpinnerUpdateMessageThreadSafe(t *testing.T) {
	s := NewSpinner("Initial")
	s.Start()

	// Concurrent updates
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.UpdateMessage("Message" + string(rune('0'+n%10)))
		}(i)
	}

	wg.Wait()
	s.Stop()

	// Should not panic or deadlock
}

func TestSpinnerStopWhenNotRunning(t *testing.T) {
	s := NewSpinner("Test")

	// Stop without starting - should not panic
	s.Stop()

	if s.running {
		t.Error("running should be false after Stop() when not started")
	}
}

func TestSpinnerStartAlreadyRunning(t *testing.T) {
	s := NewSpinner("Test")

	s.Start()
	time.Sleep(10 * time.Millisecond)

	// Start again while running - should not create extra goroutine
	oldDoneCh := s.doneCh
	s.Start()

	if s.doneCh != oldDoneCh {
		t.Error("Start() while running should not recreate channels")
	}

	s.Stop()
}

func TestSpinnerDefaultWriter(t *testing.T) {
	s := NewSpinner("Test")

	if s.writer == nil {
		t.Error("NewSpinner() should set default writer to os.Stdout")
	}
}

func TestSpinnerFramesSequence(t *testing.T) {
	s := NewSpinner("Test")

	// Verify default frames are set
	expectedFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	if len(s.frames) != len(expectedFrames) {
		t.Errorf("Expected %d frames, got %d", len(expectedFrames), len(s.frames))
	}

	for i, frame := range expectedFrames {
		if s.frames[i] != frame {
			t.Errorf("Frame %d = %q, want %q", i, s.frames[i], frame)
		}
	}
}
