package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/stack"
)

func TestHandleDeleteQueueTaskValidation(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		inv    Invocation
		errSub string
	}{
		{
			name:   "no args",
			inv:    Invocation{},
			errSub: "delete requires a task reference",
		},
		{
			name:   "invalid reference format",
			inv:    Invocation{Args: []string{"invalid-ref"}},
			errSub: "invalid queue task reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleDeleteQueueTask(context.Background(), cond, tt.inv)
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestHandleDeleteQueueTaskNoWorkspace(t *testing.T) {
	cond := mustNewConductor(t)

	// Valid format but no workspace
	result, err := handleDeleteQueueTask(context.Background(), cond, Invocation{Args: []string{"queue/task-1"}})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "workspace not initialized") {
		t.Fatalf("expected 'workspace not initialized' error, got %v", err)
	}
}

func TestHandleExportQueueTaskValidation(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		inv    Invocation
		errSub string
	}{
		{
			name:   "no args",
			inv:    Invocation{},
			errSub: "export requires a task reference",
		},
		{
			name:   "invalid reference format",
			inv:    Invocation{Args: []string{"no-slash"}},
			errSub: "invalid queue task reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleExportQueueTask(context.Background(), cond, tt.inv)
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestHandleOptimizeQueueTaskValidation(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		inv    Invocation
		errSub string
	}{
		{
			name:   "no args",
			inv:    Invocation{},
			errSub: "optimize requires a task reference",
		},
		{
			name:   "invalid reference format",
			inv:    Invocation{Args: []string{"bad-format"}},
			errSub: "invalid queue task reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleOptimizeQueueTask(context.Background(), cond, tt.inv)
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestHandleSubmitQueueTaskValidation(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		inv    Invocation
		errSub string
	}{
		{
			name:   "no args",
			inv:    Invocation{},
			errSub: "submit requires",
		},
		{
			name:   "invalid reference format",
			inv:    Invocation{Args: []string{"no-slash"}},
			errSub: "invalid queue task reference",
		},
		{
			name:   "missing provider",
			inv:    Invocation{Args: []string{"queue/task-1"}},
			errSub: "submit requires",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleSubmitQueueTask(context.Background(), cond, tt.inv)
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestHandleSubmitSourceTaskValidation(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		inv    Invocation
		errSub string
	}{
		{
			name:   "no source",
			inv:    Invocation{},
			errSub: "source is required",
		},
		{
			name:   "source but no provider",
			inv:    Invocation{Args: []string{"github:123"}},
			errSub: "provider is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleSubmitSourceTask(context.Background(), cond, tt.inv)
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestOptimizeCommandOptionsDecoding(t *testing.T) {
	tests := []struct {
		name      string
		options   map[string]any
		wantAgent string
	}{
		{
			name:      "empty options",
			options:   map[string]any{},
			wantAgent: "",
		},
		{
			name:      "agent option",
			options:   map[string]any{"agent": "claude-sonnet"},
			wantAgent: "claude-sonnet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := Invocation{Options: tt.options}
			opts, err := DecodeOptions[optimizeCommandOptions](inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}
			if opts.Agent != tt.wantAgent {
				t.Errorf("Agent = %q, want %q", opts.Agent, tt.wantAgent)
			}
		})
	}
}

func TestSubmitCommandOptionsDecoding(t *testing.T) {
	tests := []struct {
		name         string
		options      map[string]any
		wantProvider string
		wantLabels   []string
		wantDryRun   bool
	}{
		{
			name:    "empty options",
			options: map[string]any{},
		},
		{
			name:         "provider option",
			options:      map[string]any{"provider": "github"},
			wantProvider: "github",
		},
		{
			name:       "labels option",
			options:    map[string]any{"labels": []string{"bug", "p0"}},
			wantLabels: []string{"bug", "p0"},
		},
		{
			name:       "dry run flag",
			options:    map[string]any{"dry_run": true},
			wantDryRun: true,
		},
		{
			name: "all options",
			options: map[string]any{
				"provider": "jira",
				"labels":   []string{"feature"},
				"dry_run":  true,
			},
			wantProvider: "jira",
			wantLabels:   []string{"feature"},
			wantDryRun:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := Invocation{Options: tt.options}
			opts, err := DecodeOptions[submitCommandOptions](inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}
			if opts.Provider != tt.wantProvider {
				t.Errorf("Provider = %q, want %q", opts.Provider, tt.wantProvider)
			}
			if opts.DryRun != tt.wantDryRun {
				t.Errorf("DryRun = %v, want %v", opts.DryRun, tt.wantDryRun)
			}
			if len(opts.Labels) != len(tt.wantLabels) {
				t.Errorf("Labels = %v, want %v", opts.Labels, tt.wantLabels)
			}
		})
	}
}

func TestSubmitSourceCommandOptionsDecoding(t *testing.T) {
	tests := []struct {
		name             string
		options          map[string]any
		wantSource       string
		wantProvider     string
		wantTitle        string
		wantInstructions string
		wantQueueID      string
		wantOptimize     bool
		wantDryRun       bool
		wantPriority     int
	}{
		{
			name:    "empty options",
			options: map[string]any{},
		},
		{
			name:       "source option",
			options:    map[string]any{"source": "github:123"},
			wantSource: "github:123",
		},
		{
			name:         "provider option",
			options:      map[string]any{"provider": "linear"},
			wantProvider: "linear",
		},
		{
			name:      "title option",
			options:   map[string]any{"title": "New Feature"},
			wantTitle: "New Feature",
		},
		{
			name:             "instructions option",
			options:          map[string]any{"instructions": "Follow TDD"},
			wantInstructions: "Follow TDD",
		},
		{
			name:        "queue_id option",
			options:     map[string]any{"queue_id": "backlog"},
			wantQueueID: "backlog",
		},
		{
			name:         "optimize flag",
			options:      map[string]any{"optimize": true},
			wantOptimize: true,
		},
		{
			name:       "dry_run flag",
			options:    map[string]any{"dry_run": true},
			wantDryRun: true,
		},
		{
			name:         "priority option",
			options:      map[string]any{"priority": 2},
			wantPriority: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := Invocation{Options: tt.options}
			opts, err := DecodeOptions[submitSourceCommandOptions](inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}
			if opts.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", opts.Source, tt.wantSource)
			}
			if opts.Provider != tt.wantProvider {
				t.Errorf("Provider = %q, want %q", opts.Provider, tt.wantProvider)
			}
			if opts.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", opts.Title, tt.wantTitle)
			}
			if opts.Instructions != tt.wantInstructions {
				t.Errorf("Instructions = %q, want %q", opts.Instructions, tt.wantInstructions)
			}
			if opts.QueueID != tt.wantQueueID {
				t.Errorf("QueueID = %q, want %q", opts.QueueID, tt.wantQueueID)
			}
			if opts.Optimize != tt.wantOptimize {
				t.Errorf("Optimize = %v, want %v", opts.Optimize, tt.wantOptimize)
			}
			if opts.DryRun != tt.wantDryRun {
				t.Errorf("DryRun = %v, want %v", opts.DryRun, tt.wantDryRun)
			}
			if opts.Priority != tt.wantPriority {
				t.Errorf("Priority = %d, want %d", opts.Priority, tt.wantPriority)
			}
		})
	}
}

// Stack handler tests

func TestHandleStackSubcommandRouting(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name       string
		subcommand string
		wantErr    bool
		errSub     string
	}{
		{
			name:       "unknown subcommand",
			subcommand: "",
			wantErr:    true,
			errSub:     "unknown stack subcommand",
		},
		{
			name:       "list needs workspace",
			subcommand: "list",
			wantErr:    true,
			errSub:     "workspace not initialized",
		},
		{
			name:       "sync needs workspace",
			subcommand: "sync",
			wantErr:    true,
			errSub:     "workspace not initialized",
		},
		{
			name:       "rebase needs workspace",
			subcommand: "rebase",
			wantErr:    true,
			errSub:     "workspace not initialized",
		},
		{
			name:       "rebase-preview needs workspace",
			subcommand: "rebase-preview",
			wantErr:    true,
			errSub:     "workspace not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := Invocation{Options: map[string]any{"subcommand": tt.subcommand}}
			result, err := handleStack(context.Background(), cond, inv)

			if tt.wantErr {
				if result != nil {
					t.Fatalf("expected nil result, got %#v", result)
				}
				if err == nil || !strings.Contains(err.Error(), tt.errSub) {
					t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestHandleStackRebaseValidation(t *testing.T) {
	cond := mustNewConductor(t)

	// Set up invocation with subcommand but no workspace
	inv := Invocation{Options: map[string]any{"subcommand": "rebase"}}
	result, err := handleStack(context.Background(), cond, inv)

	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "workspace not initialized") {
		t.Fatalf("expected workspace error, got %v", err)
	}
}

func TestGetStackStateIcon(t *testing.T) {
	tests := []struct {
		state stack.StackState
		want  string
	}{
		{stack.StateMerged, "check"},
		{stack.StateNeedsRebase, "refresh"},
		{stack.StateConflict, "x-circle"},
		{stack.StatePendingReview, "clock"},
		{stack.StateApproved, "check-circle"},
		{stack.StateAbandoned, "slash"},
		{stack.StateActive, "play"},
		{stack.StackState("unknown"), "circle"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := getStackStateIcon(tt.state)
			if got != tt.want {
				t.Errorf("getStackStateIcon(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestBoolToInt(t *testing.T) {
	tests := []struct {
		input bool
		want  int
	}{
		{true, 1},
		{false, 0},
	}

	for _, tt := range tests {
		got := boolToInt(tt.input)
		if got != tt.want {
			t.Errorf("boolToInt(%v) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestSubmitSourceExtraction(t *testing.T) {
	// Test that source is extracted from options or args
	tests := []struct {
		name       string
		inv        Invocation
		wantSource string
	}{
		{
			name:       "source from options",
			inv:        Invocation{Options: map[string]any{"source": "github:123"}},
			wantSource: "github:123",
		},
		{
			name:       "source from args",
			inv:        Invocation{Args: []string{"file:task.md"}},
			wantSource: "file:task.md",
		},
		{
			name:       "options source takes precedence",
			inv:        Invocation{Options: map[string]any{"source": "github:1"}, Args: []string{"file:x"}},
			wantSource: "github:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := DecodeOptions[submitSourceCommandOptions](tt.inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}

			source := strings.TrimSpace(opts.Source)
			if source == "" && len(tt.inv.Args) > 0 {
				source = strings.TrimSpace(tt.inv.Args[0])
			}

			if source != tt.wantSource {
				t.Errorf("source = %q, want %q", source, tt.wantSource)
			}
		})
	}
}
