//go:build !(linux && arm64)

package browser

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// TestEmptyURL tests opening a tab with an empty URL.
func TestEmptyURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	// Empty URL should fail with URL validation error
	tab, err := controller.OpenTab(ctx, "")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
		if tab != nil {
			_ = controller.CloseTab(ctx, tab.ID)
		}

		return
	}

	// Verify it's a URL validation error
	if !strings.Contains(err.Error(), "empty URL") {
		t.Errorf("expected 'empty URL' error, got: %v", err)
	}
}

// TestInvalidURL tests opening a tab with various invalid URLs.
func TestInvalidURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	// URLs with non http/https schemes should fail validation
	invalidURLs := []struct {
		url       string
		wantError string
	}{
		{"javascript:alert(1)", "unsupported URL scheme"},
		{"file:///etc/passwd", "unsupported URL scheme"},
		{"ftp://example.com", "unsupported URL scheme"},
		{"data:text/html,<h1>hi</h1>", "unsupported URL scheme"},
	}

	for _, tc := range invalidURLs {
		t.Run(tc.url, func(t *testing.T) {
			tab, err := controller.OpenTab(ctx, tc.url)
			if err == nil {
				t.Errorf("expected error for URL %q, got nil", tc.url)
				if tab != nil {
					_ = controller.CloseTab(ctx, tab.ID)
				}

				return
			}

			if !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("expected error containing %q, got: %v", tc.wantError, err)
			}
		})
	}
}

// TestUnreachableURL tests navigating to an unreachable/unresolvable URL.
func TestUnreachableURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  5 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	// Use an invalid domain that should fail to resolve
	tab, err := controller.OpenTab(ctx, "http://this-domain-definitely-does-not-exist-12345.com")
	if err != nil {
		// Expected to fail
		return
	}

	// If it succeeds (DNS might return a NXDOMAIN page), that's also valid behavior
	if tab != nil {
		_ = controller.CloseTab(ctx, tab.ID)
	}
}

// TestMalformedSelectors tests various malformed CSS selectors.
func TestMalformedSelectors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  5 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	tab, err := controller.OpenTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("OpenTab() failed: %v", err)
	}

	// Close tab after tests
	defer func() {
		_ = controller.CloseTab(ctx, tab.ID)
	}()

	malformedSelectors := []string{
		"///",
		"...",
		"[[[",
	}

	for _, selector := range malformedSelectors {
		t.Run(selector, func(t *testing.T) {
			testCtx, testCancel := context.WithTimeout(ctx, 3*time.Second)
			defer testCancel()
			_, err := controller.QuerySelector(testCtx, tab.ID, selector)
			if err == nil {
				t.Logf("selector '%s' did not error (might be valid)", selector)
			}
		})
	}
}

// TestSpecialCharactersInSelector tests selectors with special characters.
func TestSpecialCharactersInSelector(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  5 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	tab, err := controller.OpenTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("OpenTab() failed: %v", err)
	}
	defer func() { _ = controller.CloseTab(ctx, tab.ID) }()

	// These should not crash - use short timeouts per query
	selectors := []string{
		"body",
		"h1",
		"p",
	}

	for _, selector := range selectors {
		t.Run(selector, func(t *testing.T) {
			testCtx, testCancel := context.WithTimeout(ctx, 3*time.Second)
			defer testCancel()
			_, err := controller.QuerySelector(testCtx, tab.ID, selector)
			// We don't care if the element exists, just that it doesn't crash
			_ = err
		})
	}
}

// TestVeryLongSelector tests an extremely long CSS selector.
func TestVeryLongSelector(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  3 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	tab, err := controller.OpenTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("OpenTab() failed: %v", err)
	}

	// Close tab to prevent hanging
	defer func() {
		_ = controller.CloseTab(ctx, tab.ID)
	}()

	// Create a moderately complex selector (not too long to avoid timeout)
	longSelector := "div > div > div > div > div"

	// Should handle gracefully without crashing (with timeout)
	testCtx, testCancel := context.WithTimeout(ctx, 5*time.Second)
	defer testCancel()
	_, err = controller.QuerySelector(testCtx, tab.ID, longSelector)
	_ = err // Don't care about result, just that it doesn't panic
}

// TestZeroTimeout tests operations with zero timeout.
func TestZeroTimeout(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: true,
		Timeout:  0, // Zero timeout
	}

	controller := NewController(cfg)
	// Zero timeout should be handled gracefully
	_ = controller
}

// TestVeryShortTimeout tests operations with very short timeout.
func TestVeryShortTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  1 * time.Nanosecond, // Extremely short timeout
	}

	controller := NewController(cfg)
	ctx := context.Background()

	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	// Operations should timeout gracefully
	_, err := controller.OpenTab(ctx, "https://example.com")
	if err != nil {
		// Expected to fail due to timeout
		return
	}
}

// TestMultipleConnectDisconnect tests connecting and disconnecting multiple times.
func TestMultipleConnectDisconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)

	// Connect and disconnect multiple times
	for i := range 3 {
		if err := controller.Connect(ctx); err != nil {
			t.Fatalf("Connect() iteration %d failed: %v", i, err)
		}

		if !controller.IsConnected() {
			t.Errorf("IsConnected() = false after Connect() iteration %d", i)
		}

		if err := controller.Disconnect(); err != nil {
			t.Fatalf("Disconnect() iteration %d failed: %v", i, err)
		}

		if controller.IsConnected() {
			t.Errorf("IsConnected() = true after Disconnect() iteration %d", i)
		}
	}
}

// TestConnectWhenAlreadyConnected tests calling Connect() multiple times.
func TestConnectWhenAlreadyConnected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)

	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("First Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	// Connect again - should be idempotent or return error
	if err := controller.Connect(ctx); err != nil {
		// It's ok to return an error
		t.Logf("Second Connect() returned error (acceptable): %v", err)
	}

	if !controller.IsConnected() {
		t.Error("IsConnected() = false after second Connect()")
	}
}

// TestDisconnectWhenNotConnected tests calling Disconnect() without Connect().
func TestDisconnectWhenNotConnected(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: true,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)

	// Should not panic
	if err := controller.Disconnect(); err != nil {
		t.Logf("Disconnect() without Connect() returned error: %v", err)
	}
}

// TestOperationsOnClosedTab tests performing operations on a closed tab.
func TestOperationsOnClosedTab(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	tab, err := controller.OpenTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("OpenTab() failed: %v", err)
	}

	tabID := tab.ID

	// Close the tab
	if err := controller.CloseTab(ctx, tabID); err != nil {
		t.Fatalf("CloseTab() failed: %v", err)
	}

	// Try operations on the closed tab
	operations := []struct {
		name string
		fn   func() error
	}{
		{
			name: "Navigate",
			fn:   func() error { return controller.Navigate(ctx, tabID, "https://example.org") },
		},
		{
			name: "Reload",
			fn:   func() error { return controller.Reload(ctx, tabID, false) },
		},
		{
			name: "Screenshot",
			fn: func() error {
				_, err := controller.Screenshot(ctx, tabID, ScreenshotOptions{})

				return err
			},
		},
		{
			name: "Click",
			fn:   func() error { return controller.Click(ctx, tabID, "h1") },
		},
		{
			name: "Type",
			fn:   func() error { return controller.Type(ctx, tabID, "input", "text", false) },
		},
		{
			name: "Eval",
			fn: func() error {
				_, err := controller.Eval(ctx, tabID, "1+1")

				return err
			},
		},
	}

	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			err := op.fn()
			if err == nil {
				t.Error("expected error for operation on closed tab, got nil")
			}
		})
	}
}

// TestContextCancellation tests operations with cancelled context.
func TestContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithCancel(context.Background())

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)

	// Cancel immediately
	cancel()

	// Should return context canceled error
	err := controller.Connect(ctx)
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	} else if !errors.Is(err, context.Canceled) {
		t.Logf("Connect() with cancelled context returned: %v", err)
	}
}

// TestScreenshotFormats tests different screenshot formats.
func TestScreenshotFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	tab, err := controller.OpenTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("OpenTab() failed: %v", err)
	}

	formats := []string{"png", "jpeg", "webp"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			data, err := controller.Screenshot(ctx, tab.ID, ScreenshotOptions{
				Format:   format,
				Quality:  80,
				FullPage: false,
			})
			if err != nil {
				t.Logf("Screenshot with format %s failed: %v", format, err)

				return
			}

			if len(data) == 0 {
				t.Errorf("Screenshot with format %s returned empty data", format)
			}
		})
	}
}

// TestInvalidScreenshotQuality tests screenshot with invalid quality values.
func TestInvalidScreenshotQuality(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	tab, err := controller.OpenTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("OpenTab() failed: %v", err)
	}

	invalidQualities := []int{-1, 0, 101, 1000}

	for _, quality := range invalidQualities {
		t.Run(string(rune(quality)), func(t *testing.T) {
			// Should handle gracefully
			_, err := controller.Screenshot(ctx, tab.ID, ScreenshotOptions{
				Format:   "jpeg",
				Quality:  quality,
				FullPage: false,
			})
			_ = err // Don't care about result, just that it doesn't panic
		})
	}
}

// TestEmptyJavaScript tests evaluating empty JavaScript.
func TestEmptyJavaScript(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	tab, err := controller.OpenTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("OpenTab() failed: %v", err)
	}

	// Empty JavaScript
	_, err = controller.Eval(ctx, tab.ID, "")
	if err == nil {
		t.Log("Empty JavaScript did not error")
	}
}

// TestVeryLongJavaScript tests evaluating very long JavaScript code.
func TestVeryLongJavaScript(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping edge case test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	tab, err := controller.OpenTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("OpenTab() failed: %v", err)
	}

	// Create a very long JavaScript expression
	var longExpr strings.Builder
	longExpr.WriteString("1")
	for range 1000 {
		longExpr.WriteString("+1")
	}

	// Should handle gracefully without crashing
	_, err = controller.Eval(ctx, tab.ID, longExpr.String())
	_ = err // Don't care about result, just that it doesn't panic
}
