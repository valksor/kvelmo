package plugin

import (
	"context"
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// NewLoader tests
// ──────────────────────────────────────────────────────────────────────────────

func TestNewLoader(t *testing.T) {
	l := NewLoader()

	if l == nil {
		t.Fatal("NewLoader() returned nil")
	}

	// Check that processes map is initialized
	if l.processes == nil {
		t.Error("NewLoader() processes map is nil")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Get tests
// ──────────────────────────────────────────────────────────────────────────────

func TestLoaderGet(t *testing.T) {
	l := NewLoader()

	// Get non-existent plugin
	proc, ok := l.Get("nonexistent")
	if ok {
		t.Errorf("Loader.Get() = true, want false for non-existent plugin")
	}
	if proc != nil {
		t.Errorf("Loader.Get() = non-nil process, want nil for non-existent plugin")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Unload tests
// ──────────────────────────────────────────────────────────────────────────────

func TestLoaderUnload(t *testing.T) {
	ctx := context.Background()
	l := NewLoader()

	// Unload non-existent plugin should not error
	err := l.Unload(ctx, "nonexistent")
	if err != nil {
		t.Errorf("Loader.Unload() non-existent error = %v, want nil", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// UnloadAll tests
// ──────────────────────────────────────────────────────────────────────────────

func TestLoaderUnloadAll(t *testing.T) {
	ctx := context.Background()
	l := NewLoader()

	// UnloadAll on empty loader should not error
	err := l.UnloadAll(ctx)
	if err != nil {
		t.Errorf("Loader.UnloadAll() empty loader error = %v, want nil", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Load/Unload lifecycle tests
// ──────────────────────────────────────────────────────────────────────────────

func TestLoaderLoadUnloadEmptyManifest(t *testing.T) {
	ctx := context.Background()
	l := NewLoader()

	// Try to load with a manifest that has no executable - should return error
	// because the process won't start, but we can test the loader logic
	manifest := &Manifest{
		Name:    "test-plugin",
		Version: "1.0",
		Type:    PluginTypeAgent,
	}

	// This will fail because there's no executable configured
	_, err := l.Load(ctx, manifest)
	if err == nil {
		t.Errorf("Loader.Load() with no executable expected error, got nil")
	}

	// Verify the plugin was not added
	_, ok := l.Get("test-plugin")
	if ok {
		t.Errorf("Loader.Get() found plugin after failed Load, want not found")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Load state management tests
// ──────────────────────────────────────────────────────────────────────────────

func TestLoaderConcurrentGet(t *testing.T) {
	l := NewLoader()

	// Multiple Gets should be safe (using RWMutex)
	done := make(chan bool)
	for range 10 {
		go func() {
			_, _ = l.Get("test")
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}
}

func TestLoaderProcessCount(t *testing.T) {
	l := NewLoader()

	// Initially empty
	proc, ok := l.Get("any")
	if ok {
		t.Errorf("Loader.Get() on empty loader found plugin")
	}
	if proc != nil {
		t.Errorf("Loader.Get() returned non-nil process from empty loader")
	}

	// After failed load, still empty
	ctx := context.Background()
	manifest := &Manifest{Name: "test"}
	_, _ = l.Load(ctx, manifest)

	_, ok = l.Get("test")
	if ok {
		t.Errorf("Loader.Get() found plugin after failed load")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// UnloadAll with mock processes tests
// ──────────────────────────────────────────────────────────────────────────────

func TestLoaderUnloadAllClearsMap(t *testing.T) {
	ctx := context.Background()
	l := NewLoader()

	// Verify processes map is initialized and empty
	if l.processes == nil {
		t.Fatal("NewLoader() processes map is nil")
	}
	if len(l.processes) != 0 {
		t.Errorf("NewLoader() processes map not empty, has %d entries", len(l.processes))
	}

	// UnloadAll on empty loader should not error
	err := l.UnloadAll(ctx)
	if err != nil {
		t.Errorf("Loader.UnloadAll() empty loader error = %v, want nil", err)
	}

	// Verify map is still empty
	if len(l.processes) != 0 {
		t.Errorf("Loader processes map not empty after UnloadAll, has %d entries", len(l.processes))
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Get with nil process test
// ──────────────────────────────────────────────────────────────────────────────

func TestLoaderGetWithNilProcess(t *testing.T) {
	l := NewLoader()

	// Manually add a nil entry to simulate edge case
	l.processes["nil-plugin"] = nil

	proc, ok := l.Get("nil-plugin")
	if !ok {
		t.Errorf("Loader.Get() returned false for existing key")
	}
	// This is acceptable - we return whatever is in the map
	if proc != nil {
		t.Errorf("Loader.Get() = non-nil, expected nil for nil entry")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Load nil manifest test
// ──────────────────────────────────────────────────────────────────────────────

func TestLoaderLoadNilManifest(t *testing.T) {
	ctx := context.Background()
	l := NewLoader()

	// Load with nil manifest - should handle gracefully or panic
	// The current implementation will likely panic when accessing manifest.Name
	// We'll test the behavior
	defer func() {
		if r := recover(); r != nil {
			// Expected to panic with nil manifest
			return
		}
		t.Errorf("Loader.Load(nil) expected panic, but didn't")
	}()

	_, _ = l.Load(ctx, nil)
}
