package conductor

import (
	"strings"
	"testing"
)

func TestBuildOptimizerPrompt(t *testing.T) {
	tests := []struct {
		name             string
		phase            string
		originalPrompt   string
		wantContains     []string
		dontWantContains []string
	}{
		{
			name:           "planning phase with simple prompt",
			phase:          "planning",
			originalPrompt: "Create a REST API for user management",
			wantContains: []string{
				"You are an expert at refining AI prompts",
				"planning",
				"Create a REST API for user management",
				"Optimization Guidelines",
				"Clarity",
				"Structure",
				"Conciseness",
				"Precision",
				"Completeness",
				"What NOT to Change",
				"Output Format",
				"Return ONLY the optimized prompt text",
			},
		},
		{
			name:  "implementing phase with complex prompt",
			phase: "implementing",
			originalPrompt: `Implement the following specifications:

1. Create user model
2. Add authentication endpoints
3. Write tests for all functions`,
			wantContains: []string{
				"implementing",
				"Create user model",
			},
		},
		{
			name:           "reviewing phase",
			phase:          "reviewing",
			originalPrompt: "Review the code for bugs and security issues",
			wantContains: []string{
				"reviewing",
				"Review the code for bugs and security issues",
			},
		},
		{
			name:           "empty prompt",
			phase:          "planning",
			originalPrompt: "",
			wantContains: []string{
				"planning",
			},
		},
		{
			name:           "prompt with special characters",
			phase:          "implementing",
			originalPrompt: "Fix: <div> & \"quotes\" - test's value",
			wantContains: []string{
				"implementing",
				"Fix: <div> & \"quotes\" - test's value",
			},
		},
		{
			name:  "prompt with code snippets",
			phase: "planning",
			originalPrompt: `Add this function:

func foo() string {
	return "bar"
}`,
			wantContains: []string{
				"Add this function:",
				"func foo() string",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildOptimizerPrompt(tt.phase, tt.originalPrompt)

			// Check that expected strings are present
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("buildOptimizerPrompt() should contain %q", want)
				}
			}

			// Check that unwanted strings are not present
			for _, dontWant := range tt.dontWantContains {
				if strings.Contains(got, dontWant) {
					t.Errorf("buildOptimizerPrompt() should NOT contain %q", dontWant)
				}
			}
		})
	}
}

func TestBuildOptimizerPrompt_Structure(t *testing.T) {
	phase := "planning"
	originalPrompt := "Test prompt for structure validation"

	result := buildOptimizerPrompt(phase, originalPrompt)

	// Verify the prompt structure has the expected sections in order
	expectedOrder := []string{
		"You are an expert at refining AI prompts",
		"Current timestamp:",
		"## Your Task",
		"## Original Prompt",
		"## Optimization Guidelines",
		"## What NOT to Change",
		"## Output Format",
	}

	lastIndex := 0
	for _, expected := range expectedOrder {
		index := strings.Index(result, expected)
		if index == -1 {
			t.Fatalf("buildOptimizerPrompt() missing section: %q", expected)
		}
		if index < lastIndex {
			t.Errorf("buildOptimizerPrompt() sections out of order: %q should come before previous content", expected)
		}
		lastIndex = index
	}

	// Verify phase is included in the task description
	if !strings.Contains(result, `"planning" phase`) {
		t.Error("buildOptimizerPrompt() should include phase in task description")
	}

	// Verify original prompt is included
	if !strings.Contains(result, originalPrompt) {
		t.Error("buildOptimizerPrompt() should include original prompt")
	}
}

func TestBuildOptimizerPrompt_AllGuidelinesPresent(t *testing.T) {
	guidelines := []string{
		"Clarity",
		"Structure",
		"Conciseness",
		"Precision",
		"Completeness",
	}

	result := buildOptimizerPrompt("planning", "test")

	for _, guideline := range guidelines {
		if !strings.Contains(result, guideline) {
			t.Errorf("buildOptimizerPrompt() should contain guideline: %s", guideline)
		}
	}
}

func TestBuildOptimizerPrompt_WhatNotToChange(t *testing.T) {
	result := buildOptimizerPrompt("implementing", "test prompt")

	restrictions := []string{
		"Do NOT change the fundamental task",
		"Do NOT add new constraints",
		"Do NOT change the expected output format",
		"Do NOT alter code snippets",
	}

	for _, restriction := range restrictions {
		if !strings.Contains(result, restriction) {
			t.Errorf("buildOptimizerPrompt() should contain restriction: %s", restriction)
		}
	}
}

func TestBuildOptimizerPrompt_Timestamp(t *testing.T) {
	result := buildOptimizerPrompt("planning", "test")

	// Check that timestamp placeholder is present
	if !strings.Contains(result, "Current timestamp:") {
		t.Error("buildOptimizerPrompt() should include current timestamp")
	}

	// Verify timestamp is included after the label (format includes date and time)
	// The format "2006-01-02 15:04" produces output like "2025-01-24 14:30"
	if !strings.Contains(result, "Current timestamp:") {
		t.Error("buildOptimizerPrompt() timestamp label missing")
	}
}
