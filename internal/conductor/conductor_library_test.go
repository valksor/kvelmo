package conductor

import (
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestLibrarySystemStruct(t *testing.T) {
	// Test that LibrarySystem struct is properly defined and can hold nil values
	sys := &LibrarySystem{
		manager: nil,
		config:  nil,
	}

	// Verify nil fields are accessible without panic
	if sys.manager != nil {
		t.Error("manager should be nil")
	}
	if sys.config != nil {
		t.Error("config should be nil")
	}
}

func TestLibrarySettingsConfig(t *testing.T) {
	// Test that LibrarySettings can be used in config
	cfg := &storage.LibrarySettings{
		AutoIncludeMax:    5,
		MaxPagesPerPrompt: 30,
		MaxCrawlPages:     200,
		MaxCrawlDepth:     5,
		MaxPageSizeBytes:  2 << 20, // 2MB
		LockTimeout:       "15s",
		MaxTokenBudget:    10000,
	}

	if cfg.AutoIncludeMax != 5 {
		t.Errorf("AutoIncludeMax = %d, want 5", cfg.AutoIncludeMax)
	}
	if cfg.MaxTokenBudget != 10000 {
		t.Errorf("MaxTokenBudget = %d, want 10000", cfg.MaxTokenBudget)
	}
}
