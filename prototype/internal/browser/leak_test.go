//go:build !(linux && arm64)

package browser

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// TestNoGoroutineLeaks verifies that browser operations don't leak goroutines.
func TestNoGoroutineLeaks(t *testing.T) {
	skipBrowserIntegration(t)

	initial := runtime.NumGoroutine()

	// Create a controller
	cfg := Config{
		Host:     "localhost",
		Port:     0, // Random isolated browser
		Headless: true,
		Timeout:  30 * time.Second,
	}
	ctrl := NewController(cfg)
	ctx := context.Background()

	// Connect and perform operations
	connectOrSkip(t, ctrl, ctx)

	// Create and destroy multiple tabs with monitors
	for range 5 {
		tab, err := ctrl.OpenTab(ctx, "https://example.com")
		if err != nil {
			t.Fatalf("failed to open tab: %v", err)
		}

		// Create monitors (they start goroutines)
		_, _ = ctrl.GetConsoleLogs(ctx, tab.ID, 50*time.Millisecond)
		_, _ = ctrl.GetNetworkRequests(ctx, tab.ID, 50*time.Millisecond)

		// Close tab (should clean up monitors)
		if err := ctrl.CloseTab(ctx, tab.ID); err != nil {
			t.Fatalf("failed to close tab: %v", err)
		}
	}

	// Disconnect (should clean up all resources)
	if err := ctrl.Disconnect(); err != nil {
		t.Fatalf("failed to disconnect: %v", err)
	}

	// Give time for cleanup goroutines to finish
	time.Sleep(500 * time.Millisecond)

	final := runtime.NumGoroutine()
	leaked := final - initial

	// Allow small margin for test infrastructure goroutines
	if leaked > 3 {
		t.Fatalf("goroutine leak detected: %d -> %d (leaked %d)",
			initial, final, leaked)
	}

	t.Logf("goroutine check: %d -> %d (delta: %d)", initial, final, leaked)
}
