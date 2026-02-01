//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"

	"github.com/valksor/go-mehrhof/internal/taskrunner"
)

func TestGetParallelRegistry(t *testing.T) {
	registry := GetParallelRegistry()
	if registry == nil {
		t.Fatal("GetParallelRegistry() returned nil")
	}

	// Should return the same instance (singleton)
	registry2 := GetParallelRegistry()
	if registry != registry2 {
		t.Error("GetParallelRegistry() should return the same instance on repeated calls")
	}
}

func TestSetParallelRegistry(t *testing.T) {
	// Save original
	original := GetParallelRegistry()
	defer SetParallelRegistry(original)

	// Create and set a new registry
	custom := taskrunner.NewRegistry(nil)
	SetParallelRegistry(custom)

	// Verify it's returned by Get
	got := GetParallelRegistry()
	if got != custom {
		t.Error("GetParallelRegistry() should return the registry set by SetParallelRegistry()")
	}
}
