package commands

import (
	"sync"

	"github.com/valksor/go-mehrhof/internal/taskrunner"
)

// Package-level registry for tracking running parallel tasks.
// This is shared across commands that need to interact with running tasks.
var (
	parallelRegistry     *taskrunner.Registry
	parallelRegistryOnce sync.Once
)

// GetParallelRegistry returns the shared registry for parallel task tracking.
// Creates the registry on first access.
func GetParallelRegistry() *taskrunner.Registry {
	parallelRegistryOnce.Do(func() {
		parallelRegistry = taskrunner.NewRegistry(nil)
	})

	return parallelRegistry
}

// SetParallelRegistry sets the shared registry (used by parallel start).
func SetParallelRegistry(r *taskrunner.Registry) {
	parallelRegistry = r
}
