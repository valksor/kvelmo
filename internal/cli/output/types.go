// Package output provides shared JSON output types for CLI commands.
//
// These types standardize the JSON output format across different commands,
// making it easier for scripts and tools to parse CLI output consistently.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// TokenUsage represents token consumption metrics for an operation.
// This is used across cost tracking, session reporting, and usage summaries.
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CachedTokens int `json:"cached_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens"`
}

// CostMetrics represents token usage with associated monetary cost.
type CostMetrics struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	CachedTokens int     `json:"cached_tokens,omitempty"`
	CostUSD      float64 `json:"cost_usd"`
}

// StepCost represents cost metrics for a workflow step (planning, implementing, reviewing).
type StepCost struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CachedTokens int     `json:"cached_tokens,omitempty"`
	TotalTokens  int     `json:"total_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	Calls        int     `json:"calls"`
}

// TaskSummary provides a minimal task representation for listings.
type TaskSummary struct {
	TaskID   string   `json:"task_id"`
	Title    string   `json:"title"`
	State    string   `json:"state"`
	Labels   []string `json:"labels,omitempty"`
	IsActive bool     `json:"is_active,omitempty"`
}

// SpecificationSummary provides counts of specifications by status.
type SpecificationSummary struct {
	Draft        int `json:"draft"`
	Ready        int `json:"ready"`
	Implementing int `json:"implementing"`
	Done         int `json:"done"`
}

// Checkpoint represents a version checkpoint in task history.
type Checkpoint struct {
	Number    int    `json:"number"`
	Message   string `json:"message"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
}

// Session represents an agent session record.
type Session struct {
	Kind         string `json:"kind"`
	StartTime    string `json:"start_time"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

// Specification represents a task specification/step.
type Specification struct {
	Number           int      `json:"number"`
	Title            string   `json:"title"`
	Status           string   `json:"status"`
	CreatedAt        string   `json:"created_at,omitempty"`
	CompletedAt      string   `json:"completed_at,omitempty"`
	ImplementedFiles []string `json:"implemented_files,omitempty"`
}

// WriteJSON writes the value as indented JSON to stdout.
// This provides consistent JSON formatting across all CLI commands.
func WriteJSON(v any) error {
	return WriteJSONTo(os.Stdout, v)
}

// WriteJSONTo writes the value as indented JSON to the given writer.
func WriteJSONTo(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}
