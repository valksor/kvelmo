package conductor

import (
	"strings"
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

// TestCreateAdHocParallelConfig tests ad-hoc parallel config creation.
func TestCreateAdHocParallelConfig(t *testing.T) {
	tests := []struct {
		name          string
		parallelCount string
		wantMode      orchestration.OrchestratorMode
		wantAgents    int
	}{
		{
			name:          "numeric 2",
			parallelCount: "2",
			wantMode:      orchestration.ModeParallel,
			wantAgents:    2,
		},
		{
			name:          "numeric 4",
			parallelCount: "4",
			wantMode:      orchestration.ModeParallel,
			wantAgents:    4,
		},
		{
			name:          "comma-separated agents",
			parallelCount: "claude,gemini",
			wantMode:      orchestration.ModeParallel,
			wantAgents:    2,
		},
		{
			name:          "three comma-separated",
			parallelCount: "claude, gemini, gpt",
			wantMode:      orchestration.ModeParallel,
			wantAgents:    3,
		},
		{
			name:          "invalid - less than 2",
			parallelCount: "1",
			wantMode:      "",
			wantAgents:    0,
		},
		{
			name:          "invalid - not a number",
			parallelCount: "abc",
			wantMode:      "",
			wantAgents:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal conductor with initialized workspace
			c, err := New(WithWorkDir(t.TempDir()))
			if err != nil {
				t.Fatal(err)
			}

			config := c.createAdHocParallelConfig("implementing", tt.parallelCount)

			if tt.wantAgents == 0 {
				if config != nil {
					t.Errorf("expected nil config for invalid input, got %+v", config)
				}

				return
			}

			if config == nil {
				t.Fatal("expected non-nil config, got nil")
			}

			if config.Mode != tt.wantMode {
				t.Errorf("Mode = %v, want %v", config.Mode, tt.wantMode)
			}

			if len(config.Agents) != tt.wantAgents {
				t.Errorf("Agents count = %d, want %d", len(config.Agents), tt.wantAgents)
			}

			// Verify agent names for comma-separated case
			if strings.Contains(tt.parallelCount, ",") {
				expectedNames := strings.Split(tt.parallelCount, ",")
				for i, name := range expectedNames {
					expectedNames[i] = strings.TrimSpace(name)
				}
				for i, agent := range config.Agents {
					if agent.Agent != expectedNames[i] {
						t.Errorf("Agent %d = %q, want %q", i, agent.Agent, expectedNames[i])
					}
				}
			}
		})
	}
}
