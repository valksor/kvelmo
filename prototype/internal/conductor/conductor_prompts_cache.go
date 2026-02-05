package conductor

import "sync"

// Prompt caching for static sections.
//
// These sections never change between invocations:
// - Spec validation instructions
// - Quality gate instructions
// - Error recovery section
// - Unknowns section (two variants: useDefaults true/false)
//
// Using sync.Once ensures:
// 1. Thread-safe initialization
// 2. Lazy evaluation (only computed when first needed)
// 3. Single computation per program lifetime

var (
	initPromptCacheOnce sync.Once

	// Cached static prompt sections.
	cachedSpecValidation   string
	cachedQualityGate      string
	cachedErrorRecovery    string
	cachedUnknownsDefaults string // useDefaults=true
	cachedUnknownsAsk      string // useDefaults=false
)

// initPromptCache computes static prompt sections once.
func initPromptCache() {
	initPromptCacheOnce.Do(func() {
		cachedSpecValidation = computeSpecValidationInstructions()
		cachedQualityGate = computeQualityGateInstructions()
		cachedErrorRecovery = computeErrorRecoverySection()
		cachedUnknownsDefaults = computeUnknownsSection(true)
		cachedUnknownsAsk = computeUnknownsSection(false)
	})
}

// getSpecValidationInstructions returns cached spec validation instructions.
func getSpecValidationInstructions() string {
	initPromptCache()

	return cachedSpecValidation
}

// getQualityGateInstructions returns cached quality gate instructions.
func getQualityGateInstructions() string {
	initPromptCache()

	return cachedQualityGate
}

// getErrorRecoverySection returns cached error recovery section.
func getErrorRecoverySection() string {
	initPromptCache()

	return cachedErrorRecovery
}

// getUnknownsSection returns the appropriate cached unknowns section.
func getUnknownsSection(useDefaults bool) string {
	initPromptCache()
	if useDefaults {
		return cachedUnknownsDefaults
	}

	return cachedUnknownsAsk
}
