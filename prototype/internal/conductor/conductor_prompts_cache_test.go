package conductor

import (
	"strings"
	"sync"
	"testing"
)

func TestPromptCacheInitialization(t *testing.T) {
	tests := []struct {
		name   string
		getter func() string
	}{
		{"spec validation", getSpecValidationInstructions},
		{"quality gate", getQualityGateInstructions},
		{"error recovery", getErrorRecoverySection},
		{"unknowns defaults", func() string { return getUnknownsSection(true) }},
		{"unknowns ask", func() string { return getUnknownsSection(false) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.getter()
			if s == "" {
				t.Errorf("cached %s should not be empty", tt.name)
			}
		})
	}
}

func TestPromptCacheConsistency(t *testing.T) {
	tests := []struct {
		name    string
		cached  func() string
		compute func() string
	}{
		{
			name:    "spec validation",
			cached:  getSpecValidationInstructions,
			compute: computeSpecValidationInstructions,
		},
		{
			name:    "quality gate",
			cached:  getQualityGateInstructions,
			compute: computeQualityGateInstructions,
		},
		{
			name:    "error recovery",
			cached:  getErrorRecoverySection,
			compute: computeErrorRecoverySection,
		},
		{
			name:    "unknowns defaults",
			cached:  func() string { return getUnknownsSection(true) },
			compute: func() string { return computeUnknownsSection(true) },
		},
		{
			name:    "unknowns ask",
			cached:  func() string { return getUnknownsSection(false) },
			compute: func() string { return computeUnknownsSection(false) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cached := tt.cached()
			computed := tt.compute()
			if cached != computed {
				t.Errorf("cached and computed %s should be identical", tt.name)
			}
		})
	}
}

func TestPromptCacheThreadSafety(t *testing.T) {
	var wg sync.WaitGroup
	const goroutines = 100

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = getSpecValidationInstructions()
			_ = getQualityGateInstructions()
			_ = getErrorRecoverySection()
			_ = getUnknownsSection(true)
			_ = getUnknownsSection(false)
		}()
	}
	wg.Wait()
}

func TestUnknownsSectionVariants(t *testing.T) {
	defaults := getUnknownsSection(true)
	ask := getUnknownsSection(false)

	if defaults == ask {
		t.Error("unknowns sections for defaults and ask should be different")
	}

	// Defaults variant should mention providing default answers
	if !strings.Contains(strings.ToLower(defaults), "default") {
		t.Error("defaults variant should mention providing defaults")
	}

	// Ask variant should mention asking the user
	if !strings.Contains(strings.ToLower(ask), "ask") && !strings.Contains(strings.ToLower(ask), "user") {
		t.Error("ask variant should mention asking the user")
	}
}
