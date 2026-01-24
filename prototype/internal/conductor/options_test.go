package conductor

import (
	"testing"
	"time"
)

func TestWithOptimizePrompts(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		want    bool
	}{
		{
			name:    "enable optimization",
			enabled: true,
			want:    true,
		},
		{
			name:    "disable optimization",
			enabled: false,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{}
			opt := WithOptimizePrompts(tt.enabled)
			opt(opts)

			if opts.OptimizePrompts != tt.want {
				t.Errorf("WithOptimizePrompts(%v) = %v, want %v", tt.enabled, opts.OptimizePrompts, tt.want)
			}
		})
	}
}

func TestDefaultOptions_HasOptimizePrompts(t *testing.T) {
	opts := DefaultOptions()

	// Default should be false
	if opts.OptimizePrompts != false {
		t.Errorf("DefaultOptions().OptimizePrompts = %v, want false", opts.OptimizePrompts)
	}
}

func TestOptions_Apply(t *testing.T) {
	tests := []struct {
		name    string
		base    Options
		opts    []Option
		wantOpt bool
	}{
		{
			name: "apply WithOptimizePrompts true",
			base: DefaultOptions(),
			opts: []Option{
				WithOptimizePrompts(true),
			},
			wantOpt: true,
		},
		{
			name: "apply WithOptimizePrompts false",
			base: Options{
				OptimizePrompts: true,
			},
			opts: []Option{
				WithOptimizePrompts(false),
			},
			wantOpt: false,
		},
		{
			name: "apply multiple options including optimization",
			base: DefaultOptions(),
			opts: []Option{
				WithVerbose(true),
				WithOptimizePrompts(true),
				WithTimeout(10 * time.Minute),
			},
			wantOpt: true,
		},
		{
			name: "apply no optimization options",
			base: DefaultOptions(),
			opts: []Option{
				WithVerbose(true),
				WithTimeout(5 * time.Minute),
			},
			wantOpt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Apply(tt.opts...)

			if tt.base.OptimizePrompts != tt.wantOpt {
				t.Errorf("after Apply(), OptimizePrompts = %v, want %v", tt.base.OptimizePrompts, tt.wantOpt)
			}
		})
	}
}

func TestOptions_OptimizePromptsWithOtherSettings(t *testing.T) {
	opts := DefaultOptions()
	opts.Apply(
		WithAgent("claude"),
		WithVerbose(true),
		WithOptimizePrompts(true),
		WithAutoMode(true),
	)

	// Verify OptimizePrompts is set
	if !opts.OptimizePrompts {
		t.Error("OptimizePrompts should be true")
	}

	// Verify other options are also set
	if opts.AgentName != "claude" {
		t.Errorf("AgentName = %v, want claude", opts.AgentName)
	}

	if !opts.Verbose {
		t.Error("Verbose should be true")
	}

	if !opts.AutoMode {
		t.Error("AutoMode should be true")
	}
}

func TestWithStepAgent_OptimizingStep(t *testing.T) {
	opts := DefaultOptions()
	opts.Apply(
		WithStepAgent("optimizing", "haiku"),
	)

	agent, ok := opts.StepAgents["optimizing"]
	if !ok {
		t.Fatal("StepAgents should contain 'optimizing' step")
	}

	if agent != "haiku" {
		t.Errorf("StepAgents['optimizing'] = %v, want haiku", agent)
	}
}
