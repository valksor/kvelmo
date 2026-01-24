package storage

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAgentSettings_OptimizePrompts(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		want     bool
		wantStep map[string]bool
	}{
		{
			name: "global optimize enabled",
			yaml: `
agent:
  optimize_prompts: true
`,
			want:     true,
			wantStep: nil,
		},
		{
			name: "global optimize disabled",
			yaml: `
agent:
  optimize_prompts: false
`,
			want:     false,
			wantStep: nil,
		},
		{
			name: "optimize not set (default false)",
			yaml: `
agent:
  default: claude
`,
			want:     false,
			wantStep: nil,
		},
		{
			name: "step-specific optimize enabled",
			yaml: `
agent:
  steps:
    planning:
      optimize_prompts: true
`,
			want: false,
			wantStep: map[string]bool{
				"planning": true,
			},
		},
		{
			name: "step-specific optimize disabled",
			yaml: `
agent:
  steps:
    implementing:
      optimize_prompts: false
`,
			want: false,
			wantStep: map[string]bool{
				"implementing": false,
			},
		},
		{
			name: "global enabled with step overrides",
			yaml: `
agent:
  optimize_prompts: true
  steps:
    planning:
      optimize_prompts: true
    implementing:
      optimize_prompts: false
    reviewing:
      optimize_prompts: true
`,
			want: true,
			wantStep: map[string]bool{
				"planning":     true,
				"implementing": false,
				"reviewing":    true,
			},
		},
		{
			name: "all steps with optimization",
			yaml: `
agent:
  optimize_prompts: true
  steps:
    planning:
      optimize_prompts: true
    implementing:
      optimize_prompts: true
    reviewing:
      optimize_prompts: true
`,
			want: true,
			wantStep: map[string]bool{
				"planning":     true,
				"implementing": true,
				"reviewing":    true,
			},
		},
		{
			name: "step with other settings also has optimize",
			yaml: `
agent:
  steps:
    planning:
      name: claude
      optimize_prompts: true
      instructions: "Custom planning instructions"
`,
			want: false,
			wantStep: map[string]bool{
				"planning": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg WorkspaceConfig
			if err := yaml.Unmarshal([]byte(tt.yaml), &cfg); err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}

			if cfg.Agent.OptimizePrompts != tt.want {
				t.Errorf("Agent.OptimizePrompts = %v, want %v", cfg.Agent.OptimizePrompts, tt.want)
			}

			for step, wantOpt := range tt.wantStep {
				stepCfg, ok := cfg.Agent.Steps[step]
				if !ok {
					t.Errorf("Agent.Steps[%q] not found", step)

					continue
				}
				if stepCfg.OptimizePrompts != wantOpt {
					t.Errorf("Agent.Steps[%q].OptimizePrompts = %v, want %v", step, stepCfg.OptimizePrompts, wantOpt)
				}
			}
		})
	}
}

func TestStepAgentConfig_OptimizePrompts(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want map[string]bool
	}{
		{
			name: "planning step optimization",
			yaml: `
agent:
  steps:
    planning:
      optimize_prompts: true
`,
			want: map[string]bool{
				"planning": true,
			},
		},
		{
			name: "implementing step optimization",
			yaml: `
agent:
  steps:
    implementing:
      optimize_prompts: true
`,
			want: map[string]bool{
				"implementing": true,
			},
		},
		{
			name: "reviewing step optimization",
			yaml: `
agent:
  steps:
    reviewing:
      optimize_prompts: true
`,
			want: map[string]bool{
				"reviewing": true,
			},
		},
		{
			name: "optimizing step (dedicated optimizer agent)",
			yaml: `
agent:
  steps:
    optimizing:
      name: haiku
      optimize_prompts: false
`,
			want: map[string]bool{
				"optimizing": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg WorkspaceConfig
			if err := yaml.Unmarshal([]byte(tt.yaml), &cfg); err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}

			for step, wantOpt := range tt.want {
				stepCfg, ok := cfg.Agent.Steps[step]
				if !ok {
					t.Errorf("Agent.Steps[%q] not found", step)

					continue
				}
				if stepCfg.OptimizePrompts != wantOpt {
					t.Errorf("Agent.Steps[%q].OptimizePrompts = %v, want %v", step, stepCfg.OptimizePrompts, wantOpt)
				}
			}
		})
	}
}

func TestWorkspaceConfig_FullConfigWithOptimization(t *testing.T) {
	yamlStr := `
agent:
  default: claude
  timeout: 1800
  max_retries: 3
  instructions: "Global instructions"
  optimize_prompts: true
  steps:
    planning:
      name: claude
      optimize_prompts: true
    implementing:
      name: opus
      optimize_prompts: false
    reviewing:
      name: claude
      optimize_prompts: true

git:
  auto_commit: true

workflow:
  auto_init: true
`

	var cfg WorkspaceConfig
	if err := yaml.Unmarshal([]byte(yamlStr), &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	// Check global optimization
	if !cfg.Agent.OptimizePrompts {
		t.Error("Agent.OptimizePrompts should be true")
	}

	// Check step-specific optimization
	steps := map[string]bool{
		"planning":     true,
		"implementing": false,
		"reviewing":    true,
	}

	for step, want := range steps {
		stepCfg, ok := cfg.Agent.Steps[step]
		if !ok {
			t.Errorf("Agent.Steps[%q] not found", step)

			continue
		}
		if stepCfg.OptimizePrompts != want {
			t.Errorf("Agent.Steps[%q].OptimizePrompts = %v, want %v", step, stepCfg.OptimizePrompts, want)
		}
	}

	// Verify other config is still loaded correctly
	if cfg.Agent.Default != "claude" {
		t.Errorf("Agent.Default = %v, want claude", cfg.Agent.Default)
	}

	if cfg.Agent.Timeout != 1800 {
		t.Errorf("Agent.Timeout = %v, want 1800", cfg.Agent.Timeout)
	}
}

func TestWorkspaceConfig_OptimizePromptsRoundTrip(t *testing.T) {
	original := `
agent:
  optimize_prompts: true
  steps:
    planning:
      optimize_prompts: true
    implementing:
      optimize_prompts: false
`

	var cfg1, cfg2 WorkspaceConfig

	// Unmarshal
	if err := yaml.Unmarshal([]byte(original), &cfg1); err != nil {
		t.Fatalf("first Unmarshal error = %v", err)
	}

	// Marshal
	out, err := yaml.Marshal(cfg1)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	// Unmarshal again
	if err := yaml.Unmarshal(out, &cfg2); err != nil {
		t.Fatalf("second Unmarshal error = %v", err)
	}

	// Compare values
	if cfg1.Agent.OptimizePrompts != cfg2.Agent.OptimizePrompts {
		t.Errorf("OptimizePrompts round trip: %v != %v", cfg1.Agent.OptimizePrompts, cfg2.Agent.OptimizePrompts)
	}

	steps := []string{"planning", "implementing"}
	for _, step := range steps {
		s1, ok1 := cfg1.Agent.Steps[step]
		s2, ok2 := cfg2.Agent.Steps[step]
		if ok1 != ok2 {
			t.Errorf("Step %q presence mismatch: %v != %v", step, ok1, ok2)

			continue
		}
		if s1.OptimizePrompts != s2.OptimizePrompts {
			t.Errorf("Step %q OptimizePrompts round trip: %v != %v", step, s1.OptimizePrompts, s2.OptimizePrompts)
		}
	}
}
