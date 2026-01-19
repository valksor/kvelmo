//go:build !(linux && arm64)

package browser

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBrowserIntegration tests end-to-end browser functionality.
// This is an integration test that requires Chrome to be installed.
// Headless mode is the default. Set TEST_BROWSER_VISIBLE=true to see the browser window.
func TestBrowserIntegration(t *testing.T) {
	// Use headless mode by default
	// Set TEST_BROWSER_VISIBLE=true to see the browser window
	headless := os.Getenv("TEST_BROWSER_VISIBLE") != "true"

	// Use timeout context to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a config with random port (isolated browser)
	cfg := Config{
		Host:     "localhost",
		Port:     0, // Random port
		Headless: headless,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)

	// Ensure cleanup on test exit (success or failure)
	t.Cleanup(func() {
		_ = controller.Disconnect()
	})

	// Test connection
	t.Run("Connect", func(t *testing.T) {
		if err := controller.Connect(ctx); err != nil {
			t.Fatalf("Connect() failed: %v", err)
		}
		if !controller.IsConnected() {
			t.Error("IsConnected() = false, want true")
		}
	})

	// Test opening a tab
	t.Run("OpenTab", func(t *testing.T) {
		tab, err := controller.OpenTab(ctx, "https://example.com")
		if err != nil {
			t.Fatalf("OpenTab() failed: %v", err)
		}
		if tab.ID == "" {
			t.Error("tab.ID is empty")
		}
		if tab.URL != "https://example.com/" {
			t.Errorf("tab.URL = %s, want https://example.com/", tab.URL)
		}
	})

	// Test listing tabs
	t.Run("ListTabs", func(t *testing.T) {
		tabs, err := controller.ListTabs(ctx)
		if err != nil {
			t.Fatalf("ListTabs() failed: %v", err)
		}
		if len(tabs) == 0 {
			t.Error("ListTabs() returned empty list, want at least 1 tab")
		}
	})

	// Test screenshot
	t.Run("Screenshot", func(t *testing.T) {
		tabs, err := controller.ListTabs(ctx)
		if err != nil {
			t.Fatalf("ListTabs() failed: %v", err)
		}
		if len(tabs) == 0 {
			t.Fatal("No tabs available for screenshot")
		}

		tmpDir := t.TempDir()
		screenshotPath := filepath.Join(tmpDir, "screenshot.png")

		data, err := controller.Screenshot(ctx, tabs[0].ID, ScreenshotOptions{
			Format:   "png",
			Quality:  80,
			FullPage: false,
		})
		if err != nil {
			t.Fatalf("Screenshot() failed: %v", err)
		}
		if len(data) == 0 {
			t.Error("Screenshot() returned empty data")
		}

		// Verify we can write the screenshot
		if err := os.WriteFile(screenshotPath, data, 0o644); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}

		// Verify file was created and has content
		info, err := os.Stat(screenshotPath)
		if err != nil {
			t.Fatalf("Stat() failed: %v", err)
		}
		if info.Size() == 0 {
			t.Error("Screenshot file is empty")
		}
	})

	// Test JavaScript evaluation
	t.Run("Eval", func(t *testing.T) {
		t.Skip("Eval has Rod API compatibility issues - to be investigated")

		tabs, err := controller.ListTabs(ctx)
		if err != nil {
			t.Fatalf("ListTabs() failed: %v", err)
		}
		if len(tabs) == 0 {
			t.Fatal("No tabs available for eval")
		}

		result, err := controller.Eval(ctx, tabs[0].ID, "1 + 1")
		if err != nil {
			t.Fatalf("Eval() failed: %v", err)
		}
		if result == nil {
			t.Error("Eval() returned nil, want non-nil")
		}
		// Result should be a number
		if num, ok := result.(float64); ok {
			if num != 2 {
				t.Errorf("Eval() = %v, want 2", result)
			}
		} else {
			t.Errorf("Eval() returned %T, want float64", result)
		}
	})

	// Test disconnection
	t.Run("Disconnect", func(t *testing.T) {
		if err := controller.Disconnect(); err != nil {
			t.Fatalf("Disconnect() failed: %v", err)
		}
		if controller.IsConnected() {
			t.Error("IsConnected() = true after Disconnect(), want false")
		}
	})
}

// TestMonitorLifecycle tests monitor creation and cleanup.
func TestMonitorLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping monitor lifecycle test in short mode")
	}

	headless := os.Getenv("TEST_BROWSER_VISIBLE") != "true"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  10 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	// Open a tab
	tab, err := controller.OpenTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("OpenTab() failed: %v", err)
	}

	// Create console monitor
	consoleLogs, err := controller.GetConsoleLogs(ctx, tab.ID, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("GetConsoleLogs() failed: %v", err)
	}
	if consoleLogs == nil {
		t.Fatal("GetConsoleLogs() returned nil")
	}

	// Create network monitor
	networkReqs, err := controller.GetNetworkRequests(ctx, tab.ID, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("GetNetworkRequests() failed: %v", err)
	}
	if networkReqs == nil {
		t.Fatal("GetNetworkRequests() returned nil")
	}

	// Close tab - monitors should be cleaned up
	if err := controller.CloseTab(ctx, tab.ID); err != nil {
		t.Fatalf("CloseTab() failed: %v", err)
	}

	// Verify we can't get logs from closed tab
	_, err = controller.GetConsoleLogs(ctx, tab.ID, 100*time.Millisecond)
	if err == nil {
		t.Error("Expected error when getting logs from closed tab, got nil")
	}
}

// TestMultipleTabs tests operations on multiple tabs.
func TestMultipleTabs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multiple tabs test in short mode")
	}

	headless := os.Getenv("TEST_BROWSER_VISIBLE") != "true"
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  10 * time.Second,
	}

	controller := NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	// Open multiple tabs
	tabs := []string{}
	urls := []string{
		"https://example.com",
		"https://example.org",
		"https://example.net",
	}

	for _, url := range urls {
		tab, err := controller.OpenTab(ctx, url)
		if err != nil {
			t.Fatalf("OpenTab(%s) failed: %v", url, err)
		}
		tabs = append(tabs, tab.ID)
	}

	// List all tabs
	allTabs, err := controller.ListTabs(ctx)
	if err != nil {
		t.Fatalf("ListTabs() failed: %v", err)
	}
	if len(allTabs) < len(urls) {
		t.Errorf("ListTabs() returned %d tabs, want at least %d", len(allTabs), len(urls))
	}

	// Close all tabs
	for _, tabID := range tabs {
		if err := controller.CloseTab(ctx, tabID); err != nil {
			t.Errorf("CloseTab(%s) failed: %v", tabID, err)
		}
	}
}

// TestNavigationTests tests navigation and reload operations.
func TestNavigationTests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping navigation test in short mode")
	}

	headless := os.Getenv("TEST_BROWSER_VISIBLE") != "true"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  10 * time.Second,
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

	// Test navigation
	if err := controller.Navigate(ctx, tab.ID, "https://example.org"); err != nil {
		t.Fatalf("Navigate() failed: %v", err)
	}

	// Test reload
	if err := controller.Reload(ctx, tab.ID, false); err != nil {
		t.Fatalf("Reload() failed: %v", err)
	}

	// Test switching tabs
	tabs, err := controller.ListTabs(ctx)
	if err != nil {
		t.Fatalf("ListTabs() failed: %v", err)
	}
	if len(tabs) > 0 {
		switchedTab, err := controller.SwitchTab(ctx, tabs[0].ID)
		if err != nil {
			t.Fatalf("SwitchTab() failed: %v", err)
		}
		if switchedTab.ID != tabs[0].ID {
			t.Errorf("SwitchTab() returned tab %s, want %s", switchedTab.ID, tabs[0].ID)
		}
	}
}

// TestErrorHandling tests error handling for invalid operations.
func TestErrorHandling(t *testing.T) {
	ctx := context.Background()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: true,
		Timeout:  30 * time.Second,
	}

	controller := NewController(cfg)

	// Test operations when not connected
	t.Run("NotConnected", func(t *testing.T) {
		_, err := controller.ListTabs(ctx)
		if err == nil {
			t.Error("ListTabs() should fail when not connected")
		}

		_, err = controller.OpenTab(ctx, "https://example.com")
		if err == nil {
			t.Error("OpenTab() should fail when not connected")
		}

		err = controller.Navigate(ctx, "nonexistent", "https://example.com")
		if err == nil {
			t.Error("Navigate() should fail for nonexistent tab")
		}

		err = controller.CloseTab(ctx, "nonexistent")
		if err == nil {
			t.Error("CloseTab() should fail for nonexistent tab")
		}
	})

	// Connect for further tests
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	// Test operations on non-existent tabs
	t.Run("NonExistentTab", func(t *testing.T) {
		err := controller.Navigate(ctx, "faketabid", "https://example.com")
		if err == nil {
			t.Error("Navigate() should fail for nonexistent tab")
		}

		err = controller.Reload(ctx, "faketabid", false)
		if err == nil {
			t.Error("Reload() should fail for nonexistent tab")
		}

		_, err = controller.Screenshot(ctx, "faketabid", ScreenshotOptions{})
		if err == nil {
			t.Error("Screenshot() should fail for nonexistent tab")
		}

		_, err = controller.Eval(ctx, "faketabid", "1+1")
		if err == nil {
			t.Error("Eval() should fail for nonexistent tab")
		}

		_, err = controller.QuerySelector(ctx, "faketabid", "body")
		if err == nil {
			t.Error("QuerySelector() should fail for nonexistent tab")
		}

		err = controller.Click(ctx, "faketabid", "button")
		if err == nil {
			t.Error("Click() should fail for nonexistent tab")
		}

		err = controller.Type(ctx, "faketabid", "input", "text", false)
		if err == nil {
			t.Error("Type() should fail for nonexistent tab")
		}
	})
}

// TestDOMOperations tests DOM interaction.
func TestDOMOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DOM operations test in short mode")
	}

	headless := os.Getenv("TEST_BROWSER_VISIBLE") != "true"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  10 * time.Second,
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

	// Test query selector
	elem, err := controller.QuerySelector(ctx, tab.ID, "h1")
	if err != nil {
		t.Fatalf("QuerySelector() failed: %v", err)
	}
	if elem == nil {
		t.Fatal("QuerySelector() returned nil, expected h1 element")
	}

	// Test query selector all
	elems, err := controller.QuerySelectorAll(ctx, tab.ID, "p")
	if err != nil {
		t.Fatalf("QuerySelectorAll() failed: %v", err)
	}
	if elems == nil {
		t.Fatal("QuerySelectorAll() returned nil")
	}

	// Test that invalid selector fails
	testCtx, testCancel := context.WithTimeout(ctx, 5*time.Second)
	defer testCancel()
	_, err = controller.QuerySelector(testCtx, tab.ID, "///invalid-selector")
	if err == nil {
		t.Error("QuerySelector() should fail for invalid selector")
	}
}

// TestReconnect tests disconnecting and reconnecting.
func TestReconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping reconnect test in short mode")
	}

	headless := os.Getenv("TEST_BROWSER_VISIBLE") != "true"
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "localhost",
		Port:     0,
		Headless: headless,
		Timeout:  10 * time.Second,
	}

	controller := NewController(cfg)

	// First connection
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("First Connect() failed: %v", err)
	}
	if !controller.IsConnected() {
		t.Error("IsConnected() = false after first Connect()")
	}

	// Disconnect
	if err := controller.Disconnect(); err != nil {
		t.Fatalf("Disconnect() failed: %v", err)
	}
	if controller.IsConnected() {
		t.Error("IsConnected() = true after Disconnect()")
	}

	// Reconnect
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Second Connect() failed: %v", err)
	}
	if !controller.IsConnected() {
		t.Error("IsConnected() = false after second Connect()")
	}

	// Verify we can open a tab after reconnect
	tab, err := controller.OpenTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("OpenTab() after reconnect failed: %v", err)
	}

	// Clean up
	_ = controller.CloseTab(ctx, tab.ID)
	_ = controller.Disconnect()
}
