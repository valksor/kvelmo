package conductor

import (
	"testing"

	"github.com/valksor/go-mehrhof/internal/agent/orchestration"
)

// TestExtractFinalOutput tests the extractFinalOutput helper method.
func TestExtractFinalOutput(t *testing.T) {
	tests := []struct {
		name           string
		finalOutput    string
		stepOutputs    map[string]string
		expectedOutput string
	}{
		{
			name:        "consensus mode with final output",
			finalOutput: "This is the synthesized final output from consensus.",
			stepOutputs: map[string]string{
				"agent-1": "Agent 1 output",
				"agent-2": "Agent 2 output",
			},
			expectedOutput: "This is the synthesized final output from consensus.",
		},
		{
			name:        "sequential mode uses last step",
			finalOutput: "",
			stepOutputs: map[string]string{
				"step-1": "First step output",
				"step-2": "Second step output",
				"step-3": "Third step output",
			},
			expectedOutput: "Third step output",
		},
		{
			name:           "empty results",
			finalOutput:    "",
			stepOutputs:    map[string]string{},
			expectedOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Conductor{}
			result := &orchestration.PipelineResult{
				FinalOutput: tt.finalOutput,
				StepResults: make(map[string]*orchestration.StepResult),
			}

			for stepName, output := range tt.stepOutputs {
				result.StepResults[stepName] = &orchestration.StepResult{
					StepName: stepName,
					Output:   output,
				}
			}

			output := c.extractFinalOutput(result)
			if output != tt.expectedOutput {
				t.Errorf("extractFinalOutput() = %q, want %q", output, tt.expectedOutput)
			}
		})
	}
}

// TestLogOrchestrationResult tests that logging doesn't panic.
func TestLogOrchestrationResult(t *testing.T) {
	c := &Conductor{}

	// Create a mock result
	result := &orchestration.PipelineResult{
		StepResults: map[string]*orchestration.StepResult{
			"agent-1": {
				StepName:   "agent-1",
				AgentName:  "test-agent",
				Output:     "Test output",
				TokenUsage: 1000,
				Duration:   1000000, // 1ms
			},
		},
		FinalOutput: "Synthesized output",
		Consensus:   0.8,
		Duration:    2000000, // 2ms
	}

	// This should not panic
	c.logOrchestrationResult("planning", result)
}
