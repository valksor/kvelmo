package browser

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// TestConcurrentTabs tests opening and closing multiple tabs concurrently.
func TestConcurrentTabs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx := context.Background()

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

	const numTabs = 10
	var wg sync.WaitGroup
	tabIDs := make(chan string, numTabs)
	errors := make(chan error, numTabs)

	// Open tabs concurrently
	for i := range numTabs {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			url := fmt.Sprintf("https://example.com/%d", index)
			tab, err := controller.OpenTab(ctx, url)
			if err != nil {
				errors <- fmt.Errorf("open tab %d failed: %w", index, err)

				return
			}
			tabIDs <- tab.ID
		}(i)
	}

	wg.Wait()
	close(tabIDs)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Count successful opens
	successCount := 0
	for range tabIDs {
		successCount++
	}

	if successCount < numTabs {
		t.Errorf("only opened %d/%d tabs", successCount, numTabs)
	}

	// List tabs to verify
	tabs, err := controller.ListTabs(ctx)
	if err != nil {
		t.Fatalf("ListTabs() failed: %v", err)
	}

	if len(tabs) < numTabs {
		t.Errorf("ListTabs() returned %d tabs, want at least %d", len(tabs), numTabs)
	}

	// Close tabs concurrently
	var closeWg sync.WaitGroup
	for tabID := range tabIDs {
		closeWg.Add(1)
		go func(id string) {
			defer closeWg.Done()
			if err := controller.CloseTab(ctx, id); err != nil {
				t.Errorf("close tab %s failed: %v", id, err)
			}
		}(tabID)
	}

	closeWg.Wait()
}

// TestConcurrentCloseTab tests closing the same tab from multiple goroutines.
func TestConcurrentCloseTab(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx := context.Background()

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

	// Try to close the same tab from multiple goroutines
	const numGoroutines = 5
	var wg sync.WaitGroup

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = controller.CloseTab(ctx, tab.ID)
		}()
	}

	wg.Wait()

	// Verify tab is closed
	listedTabs, err := controller.ListTabs(ctx)
	if err != nil {
		t.Fatalf("ListTabs() failed: %v", err)
	}

	for _, listedTab := range listedTabs {
		if listedTab.ID == tab.ID {
			t.Error("tab still exists after concurrent close")
		}
	}
}

// TestConcurrentMonitors tests creating monitors for the same tab concurrently.
func TestConcurrentMonitors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx := context.Background()

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

	// Create console monitors concurrently
	const numGoroutines = 10
	var wg sync.WaitGroup

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = controller.GetConsoleLogs(ctx, tab.ID, 10*time.Millisecond)
		}()
	}

	wg.Wait()

	// Create network monitors concurrently
	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = controller.GetNetworkRequests(ctx, tab.ID, 10*time.Millisecond)
		}()
	}

	wg.Wait()
}

// TestConcurrentListAndModify tests listing tabs while modifying them.
func TestConcurrentListAndModify(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx := context.Background()

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

	const numOperations = 20
	var wg sync.WaitGroup

	// Goroutine that opens tabs
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range numOperations / 5 {
				tab, err := controller.OpenTab(ctx, "https://example.com")
				if err != nil {
					t.Errorf("open tab failed: %v", err)
				}
				time.Sleep(10 * time.Millisecond)
				if err := controller.CloseTab(ctx, tab.ID); err != nil {
					t.Errorf("close tab failed: %v", err)
				}
			}
		}()
	}

	// Goroutine that lists tabs
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range numOperations {
			_, err := controller.ListTabs(ctx)
			if err != nil {
				t.Errorf("list tabs failed: %v", err)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	wg.Wait()
}

// TestConcurrentNavigation tests navigating multiple tabs concurrently.
func TestConcurrentNavigation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx := context.Background()

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

	// Open multiple tabs
	const numTabs = 5
	tabs := make([]string, numTabs)
	for i := range numTabs {
		tab, err := controller.OpenTab(ctx, "https://example.com")
		if err != nil {
			t.Fatalf("OpenTab() %d failed: %v", i, err)
		}
		tabs[i] = tab.ID
	}

	// Navigate all tabs concurrently
	var wg sync.WaitGroup
	urls := []string{
		"https://example.com",
		"https://example.org",
		"https://example.net",
	}

	for _, tabID := range tabs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			for _, url := range urls {
				if err := controller.Navigate(ctx, id, url); err != nil {
					t.Errorf("navigate %s to %s failed: %v", id, url, err)
				}
				time.Sleep(50 * time.Millisecond)
				if err := controller.Reload(ctx, id, false); err != nil {
					t.Errorf("reload %s failed: %v", id, err)
				}
			}
		}(tabID)
	}

	wg.Wait()

	// Close all tabs
	for _, tabID := range tabs {
		if err := controller.CloseTab(ctx, tabID); err != nil {
			t.Errorf("close tab %s failed: %v", tabID, err)
		}
	}
}

// TestConcurrentDisconnect tests concurrent disconnect calls.
func TestConcurrentDisconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx := context.Background()

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

	// Call Disconnect from multiple goroutines
	const numGoroutines = 5
	var wg sync.WaitGroup

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = controller.Disconnect()
		}()
	}

	wg.Wait()

	// Verify disconnected
	if controller.IsConnected() {
		t.Error("controller still connected after concurrent disconnect")
	}
}

// TestRaceDetectorTest is specifically for running with go test -race.
// It exercises potential race conditions in monitor management.
func TestRaceDetectorTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race detector test in short mode")
	}

	headless := os.Getenv("CI") != "" || os.Getenv("TEST_BROWSER_HEADLESS") == "true"
	ctx := context.Background()

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

	// Create a tab
	tab, err := controller.OpenTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("OpenTab() failed: %v", err)
	}

	// This pattern is designed to trigger race conditions if they exist
	const numIterations = 10
	for range numIterations {
		var wg sync.WaitGroup

		// Goroutine 1: Get console logs
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = controller.GetConsoleLogs(ctx, tab.ID, 10*time.Millisecond)
		}()

		// Goroutine 2: Get network requests
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = controller.GetNetworkRequests(ctx, tab.ID, 10*time.Millisecond)
		}()

		// Goroutine 3: List tabs
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = controller.ListTabs(ctx)
		}()

		wg.Wait()
	}
}
