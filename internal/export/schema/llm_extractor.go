// Package schema provides JSON Schema-based extraction for project plans.
package schema

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ExtractPlan extracts a project plan from content using the schema.
// It sends the content to the LLM with the schema and parses the JSON response.
func (e *Extractor) ExtractPlan(ctx context.Context, content string) (*ParsedPlan, error) {
	if e.agent == nil {
		return nil, errors.New("extractor: no agent configured")
	}

	prompt := e.buildPrompt(content)

	response, err := e.agent.Run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM extraction failed: %w", err)
	}

	// Extract JSON from the response
	jsonText := extractJSON(response.Summary)
	if jsonText == "" {
		return nil, errors.New("no JSON found in LLM response")
	}

	// Parse the JSON response
	var plan ParsedPlan
	if err := json.Unmarshal([]byte(jsonText), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse extracted JSON: %w", err)
	}

	return &plan, nil
}

// buildPrompt constructs the extraction prompt with schema and content.
func (e *Extractor) buildPrompt(content string) string {
	return "Extract the project plan from the following content.\n" +
		"Return ONLY a valid JSON object matching this schema (no markdown, no explanation):\n\n" +
		string(TaskSchema) + "\n\n" +
		"Content to extract from:\n" +
		content + "\n\n" +
		"IMPORTANT: Return ONLY the JSON object. Do not include markdown formatting.\n" +
		"If you cannot find certain fields, return empty arrays/strings for those fields.\n" +
		"The \"id\" and \"title\" fields are required for each task."
}

// extractJSON extracts JSON from a response that might contain markdown formatting.
// It handles both plain JSON and markdown code blocks.
func extractJSON(response string) string {
	// Try to extract JSON from markdown code block
	start := 0
	end := len(response)

	// Look for ```json or ``` opening
	if idx := findIndex(response, "```json", "```"); idx >= 0 {
		// Check if this is actually ```json (not just ```)
		if idx+7 <= len(response) && response[idx:idx+7] == "```json" {
			start = idx + 7
			if start < len(response) && response[start] == '\n' {
				start++
			}
		} else {
			// Just ```
			start = idx + 3
			if start < len(response) && response[start] == '\n' {
				start++
			}
		}
	} else if idx := findIndex(response, "```"); idx >= 0 {
		start = idx + 3
		if start < len(response) && response[start] == '\n' {
			start++
		}
	}

	// Look for closing ```
	// Search in the original string starting from 'start'
	if idx := findIndex(response[start:], "```"); idx >= 0 {
		// idx is relative to response[start:], so add start to get absolute position
		end = start + idx
	}

	jsonText := response[start:end]

	return trimWhitespace(jsonText)
}

// findIndex returns the first index of any of the given substrings, or -1 if not found.
func findIndex(s string, substrs ...string) int {
	for _, substr := range substrs {
		if idx := indexOf(s, substr); idx >= 0 {
			return idx
		}
	}

	return -1
}

// indexOf returns the index of substr in s, or -1 if not found.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}

// trimWhitespace removes leading and trailing whitespace from a string.
func trimWhitespace(s string) string {
	start := 0
	end := len(s)

	// Find first non-whitespace character
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Find last non-whitespace character
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
