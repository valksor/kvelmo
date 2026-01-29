// Package schema provides JSON Schema-based extraction for project plans.
package schema

import (
	"github.com/valksor/go-mehrhof/internal/agent"
)

// Extractor extracts structured data from text using an LLM.
type Extractor struct {
	agent agent.Agent
}

// NewExtractor creates a new LLM-based extractor.
func NewExtractor(a agent.Agent) *Extractor {
	return &Extractor{agent: a}
}
