package commands

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/workflow"
)

func TestHandleStartValidation(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		inv    Invocation
		errSub string
	}{
		{
			name:   "empty reference",
			inv:    Invocation{},
			errSub: "start requires a reference",
		},
		{
			name:   "whitespace only reference",
			inv:    Invocation{Options: map[string]any{"ref": "   "}},
			errSub: "start requires a reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleStart(context.Background(), cond, tt.inv)
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestStartRefExtraction(t *testing.T) {
	// Test that reference is extracted from options or args
	tests := []struct {
		name    string
		inv     Invocation
		wantRef string
	}{
		{
			name:    "ref from options",
			inv:     Invocation{Options: map[string]any{"ref": "github:123"}},
			wantRef: "github:123",
		},
		{
			name:    "ref from args",
			inv:     Invocation{Args: []string{"file:task.md"}},
			wantRef: "file:task.md",
		},
		{
			name:    "ref from multiple args joined",
			inv:     Invocation{Args: []string{"some", "task", "ref"}},
			wantRef: "some task ref",
		},
		{
			name:    "options ref takes precedence",
			inv:     Invocation{Options: map[string]any{"ref": "github:1"}, Args: []string{"file:x"}},
			wantRef: "github:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := DecodeOptions[startOptions](tt.inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}

			ref := strings.TrimSpace(opts.Ref)
			if ref == "" && len(tt.inv.Args) > 0 {
				ref = strings.Join(tt.inv.Args, " ")
			}

			if ref != tt.wantRef {
				t.Errorf("ref = %q, want %q", ref, tt.wantRef)
			}
		})
	}
}

func TestHandleImplementSubcommandValidation(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		args   []string
		errSub string
	}{
		{
			name:   "review without number",
			args:   []string{"review"},
			errSub: "usage: implement review <number>",
		},
		{
			name:   "review with non-integer",
			args:   []string{"review", "abc"},
			errSub: "review number must be an integer",
		},
		{
			name:   "review with zero",
			args:   []string{"review", "0"},
			errSub: "review number must be positive",
		},
		{
			name:   "review with negative",
			args:   []string{"review", "-1"},
			errSub: "review number must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleImplement(context.Background(), cond, Invocation{Args: tt.args})
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestHandleReviewSubcommandValidation(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		args   []string
		errSub string
	}{
		{
			name:   "view with non-number",
			args:   []string{"view", "abc"},
			errSub: "review view requires a number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleReview(context.Background(), cond, Invocation{Args: tt.args})
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestHandleReviewViewNoWorkspace(t *testing.T) {
	cond := mustNewConductor(t)

	result, err := handleReviewView(context.Background(), cond, 1)
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "no workspace available") {
		t.Fatalf("expected 'no workspace available' error, got %v", err)
	}
}

func TestHandleContinueNoTask(t *testing.T) {
	cond := mustNewConductor(t)

	result, err := handleContinue(context.Background(), cond, Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if !errors.Is(err, ErrNoActiveTask) {
		t.Fatalf("expected ErrNoActiveTask, got %v", err)
	}
}

func TestContinueNextActionsForState(t *testing.T) {
	tests := []struct {
		name           string
		state          workflow.State
		specifications int
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:           "idle with no specs",
			state:          workflow.StateIdle,
			specifications: 0,
			wantContains:   []string{"plan", "notes"},
			wantNotContain: []string{"implement"},
		},
		{
			name:           "idle with specs",
			state:          workflow.StateIdle,
			specifications: 3,
			wantContains:   []string{"implement", "plan", "notes"},
		},
		{
			name:           "planning state",
			state:          workflow.StatePlanning,
			specifications: 0,
			wantContains:   []string{"implement", "question", "notes"},
		},
		{
			name:           "implementing state",
			state:          workflow.StateImplementing,
			specifications: 0,
			wantContains:   []string{"implement", "undo", "finish", "notes"},
		},
		{
			name:           "reviewing state",
			state:          workflow.StateReviewing,
			specifications: 0,
			wantContains:   []string{"finish", "implement", "question"},
		},
		{
			name:           "failed state",
			state:          workflow.StateFailed,
			specifications: 0,
			wantContains:   []string{"task", "implement", "notes"},
		},
		{
			name:           "waiting state",
			state:          workflow.StateWaiting,
			specifications: 0,
			wantContains:   []string{"answer"},
		},
		{
			name:           "paused state",
			state:          workflow.StatePaused,
			specifications: 0,
			wantContains:   []string{"resume", "costs"},
		},
		{
			name:           "done state",
			state:          workflow.StateDone,
			specifications: 0,
			wantContains:   []string{"start"},
		},
		{
			name:           "checkpointing state",
			state:          workflow.StateCheckpointing,
			specifications: 0,
			wantContains:   []string{"task"},
		},
		{
			name:           "reverting state",
			state:          workflow.StateReverting,
			specifications: 0,
			wantContains:   []string{"task"},
		},
		{
			name:           "restoring state",
			state:          workflow.StateRestoring,
			specifications: 0,
			wantContains:   []string{"task"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions := continueNextActionsForState(tt.state, tt.specifications)

			actionsStr := strings.Join(actions, " ")

			for _, want := range tt.wantContains {
				if !strings.Contains(actionsStr, want) {
					t.Errorf("expected actions to contain %q, got %v", want, actions)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(actionsStr, notWant) {
					t.Errorf("expected actions to NOT contain %q, got %v", notWant, actions)
				}
			}
		})
	}
}

func TestContinueNextActionsUnknownState(t *testing.T) {
	// Test unknown state returns default actions
	actions := continueNextActionsForState(workflow.State("unknown"), 0)

	if len(actions) == 0 {
		t.Fatal("expected default actions for unknown state")
	}

	actionsStr := strings.Join(actions, " ")
	if !strings.Contains(actionsStr, "task") || !strings.Contains(actionsStr, "notes") {
		t.Errorf("expected default actions to include task and notes, got %v", actions)
	}
}

func TestWorkflowHandlersNotInitialized(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		call   func() (*Result, error)
		errSub string
	}{
		{
			name: "plan without initialization",
			call: func() (*Result, error) {
				return handlePlan(context.Background(), cond, Invocation{})
			},
			errSub: "planning failed",
		},
		{
			name: "implement without initialization",
			call: func() (*Result, error) {
				return handleImplement(context.Background(), cond, Invocation{})
			},
			errSub: "implementation failed",
		},
		{
			name: "review without initialization",
			call: func() (*Result, error) {
				return handleReview(context.Background(), cond, Invocation{})
			},
			errSub: "review failed",
		},
		{
			name: "finish without initialization",
			call: func() (*Result, error) {
				return handleFinish(context.Background(), cond, Invocation{})
			},
			errSub: "finish failed",
		},
		{
			name: "abandon without initialization",
			call: func() (*Result, error) {
				return handleAbandon(context.Background(), cond, Invocation{})
			},
			errSub: "abandon failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.call()
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestStartOptionsDecoding(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]any
		wantRef string
	}{
		{
			name:    "ref from options",
			options: map[string]any{"ref": "github:123"},
			wantRef: "github:123",
		},
		{
			name:    "template option",
			options: map[string]any{"ref": "file:test.md", "template": "feature"},
			wantRef: "file:test.md",
		},
		{
			name:    "no_branch option",
			options: map[string]any{"ref": "file:test.md", "no_branch": true},
			wantRef: "file:test.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := Invocation{Options: tt.options}
			opts, err := DecodeOptions[startOptions](inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}
			if opts.Ref != tt.wantRef {
				t.Errorf("Ref = %q, want %q", opts.Ref, tt.wantRef)
			}
		})
	}
}

func TestImplementOptionsDecoding(t *testing.T) {
	tests := []struct {
		name          string
		options       map[string]any
		wantComponent string
		wantParallel  string
	}{
		{
			name:          "no options",
			options:       map[string]any{},
			wantComponent: "",
			wantParallel:  "",
		},
		{
			name:          "component option",
			options:       map[string]any{"component": "auth"},
			wantComponent: "auth",
			wantParallel:  "",
		},
		{
			name:          "parallel option",
			options:       map[string]any{"parallel": "2"},
			wantComponent: "",
			wantParallel:  "2",
		},
		{
			name:          "both options",
			options:       map[string]any{"component": "api", "parallel": "3"},
			wantComponent: "api",
			wantParallel:  "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := Invocation{Options: tt.options}
			opts, err := DecodeOptions[implementOptions](inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}
			if opts.Component != tt.wantComponent {
				t.Errorf("Component = %q, want %q", opts.Component, tt.wantComponent)
			}
			if opts.Parallel != tt.wantParallel {
				t.Errorf("Parallel = %q, want %q", opts.Parallel, tt.wantParallel)
			}
		})
	}
}

func TestContinueOptionsDecoding(t *testing.T) {
	tests := []struct {
		name     string
		options  map[string]any
		wantAuto bool
	}{
		{
			name:     "no auto",
			options:  map[string]any{},
			wantAuto: false,
		},
		{
			name:     "auto true",
			options:  map[string]any{"auto": true},
			wantAuto: true,
		},
		{
			name:     "auto false",
			options:  map[string]any{"auto": false},
			wantAuto: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := Invocation{Options: tt.options}
			opts, err := DecodeOptions[continueOptions](inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}
			if opts.Auto != tt.wantAuto {
				t.Errorf("Auto = %v, want %v", opts.Auto, tt.wantAuto)
			}
		})
	}
}

func TestHandleReviewNumericArg(t *testing.T) {
	cond := mustNewConductor(t)

	// When passed a numeric first arg, should try to view that review
	result, err := handleReview(context.Background(), cond, Invocation{Args: []string{"1"}})

	// Without workspace, should fail with workspace error
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "workspace") {
		t.Fatalf("expected workspace-related error, got %v", err)
	}
}

func TestHandleImplementReviewValidNumber(t *testing.T) {
	cond := mustNewConductor(t)

	// Valid review number should proceed to implement review
	result, err := handleImplement(context.Background(), cond, Invocation{Args: []string{"review", "1"}})

	// Without workspace, should fail with implement review error
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "implement review failed") {
		t.Fatalf("expected 'implement review failed' error, got %v", err)
	}
}
