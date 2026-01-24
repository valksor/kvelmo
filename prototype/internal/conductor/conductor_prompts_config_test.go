package conductor

import (
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestShouldOptimizePrompt(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *storage.WorkspaceConfig
		step        string
		want        bool
		description string
	}{
		{
			name:        "nil config returns false",
			cfg:         nil,
			step:        "planning",
			want:        false,
			description: "When config is nil, optimization should be disabled",
		},
		{
			name: "global disabled, step not configured",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					OptimizePrompts: false,
				},
			},
			step:        "planning",
			want:        false,
			description: "Global disabled, no step config should return false",
		},
		{
			name: "global enabled, step not configured",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					OptimizePrompts: true,
				},
			},
			step:        "planning",
			want:        true,
			description: "Global enabled should return true when step not configured",
		},
		{
			name: "global disabled, step enabled",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					OptimizePrompts: false,
					Steps: map[string]storage.StepAgentConfig{
						"planning": {
							OptimizePrompts: true,
						},
					},
				},
			},
			step:        "planning",
			want:        true,
			description: "Step-specific setting overrides global setting",
		},
		{
			name: "global enabled, step disabled",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					OptimizePrompts: true,
					Steps: map[string]storage.StepAgentConfig{
						"planning": {
							OptimizePrompts: false,
						},
					},
				},
			},
			step:        "planning",
			want:        false,
			description: "Step-specific false overrides global true",
		},
		{
			name: "global enabled, step configured with false explicitly",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					OptimizePrompts: true,
					Steps: map[string]storage.StepAgentConfig{
						"implementing": {
							OptimizePrompts: false,
						},
					},
				},
			},
			step:        "implementing",
			want:        false,
			description: "Step explicitly set to false should disable optimization for that step",
		},
		{
			name: "empty steps map",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					OptimizePrompts: true,
					Steps:           map[string]storage.StepAgentConfig{},
				},
			},
			step:        "reviewing",
			want:        true,
			description: "Empty steps map should fall back to global setting",
		},
		{
			name: "step not in steps map",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					OptimizePrompts: true,
					Steps: map[string]storage.StepAgentConfig{
						"planning": {},
					},
				},
			},
			step:        "implementing",
			want:        true,
			description: "Step not in map should fall back to global setting",
		},
		{
			name: "all steps with individual settings",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					OptimizePrompts: false, // Global disabled
					Steps: map[string]storage.StepAgentConfig{
						"planning":     {OptimizePrompts: true},
						"implementing": {OptimizePrompts: true},
						"reviewing":    {OptimizePrompts: false},
					},
				},
			},
			step:        "planning",
			want:        true,
			description: "Planning step should be enabled",
		},
		{
			name: "all steps with individual settings - implementing",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					OptimizePrompts: false,
					Steps: map[string]storage.StepAgentConfig{
						"planning":     {OptimizePrompts: true},
						"implementing": {OptimizePrompts: true},
						"reviewing":    {OptimizePrompts: false},
					},
				},
			},
			step:        "implementing",
			want:        true,
			description: "Implementing step should be enabled",
		},
		{
			name: "all steps with individual settings - reviewing",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					OptimizePrompts: false,
					Steps: map[string]storage.StepAgentConfig{
						"planning":     {OptimizePrompts: true},
						"implementing": {OptimizePrompts: true},
						"reviewing":    {OptimizePrompts: false},
					},
				},
			},
			step:        "reviewing",
			want:        false,
			description: "Reviewing step should be disabled",
		},
		{
			name: "step config exists but optimize_prompts not set (zero value)",
			cfg: &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					OptimizePrompts: true,
					Steps: map[string]storage.StepAgentConfig{
						"planning": {}, // OptimizePrompts is false by default
					},
				},
			},
			step:        "planning",
			want:        false,
			description: "Step config with unset OptimizePrompts should use false (zero value)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldOptimizePrompt(tt.cfg, tt.step)
			if got != tt.want {
				t.Errorf("%s: shouldOptimizePrompt() = %v, want %v", tt.description, got, tt.want)
			}
		})
	}
}

func TestShouldOptimizePrompt_AllWorkflowSteps(t *testing.T) {
	steps := []string{"planning", "implementing", "reviewing", "checkpointing", "simplifying", "optimizing"}

	for _, step := range steps {
		t.Run("step_"+step, func(t *testing.T) {
			cfg := &storage.WorkspaceConfig{
				Agent: storage.AgentSettings{
					OptimizePrompts: true,
				},
			}

			got := shouldOptimizePrompt(cfg, step)
			if !got {
				t.Errorf("shouldOptimizePrompt() for step %q should return true when global enabled", step)
			}
		})
	}
}

func TestShouldOptimizePrompt_Precedence(t *testing.T) {
	// Test that step-specific setting takes precedence over global
	cfg := &storage.WorkspaceConfig{
		Agent: storage.AgentSettings{
			OptimizePrompts: false, // Global disabled
			Steps: map[string]storage.StepAgentConfig{
				"planning": {
					OptimizePrompts: true, // Step enabled
				},
			},
		},
	}

	// Step-specific true should override global false
	if got := shouldOptimizePrompt(cfg, "planning"); !got {
		t.Error("Step-specific true should override global false")
	}

	// For other steps, global false should apply
	if got := shouldOptimizePrompt(cfg, "implementing"); got {
		t.Error("Global false should apply when step not configured")
	}
}
