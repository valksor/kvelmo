package commands

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestControlHandlersSimple(t *testing.T) {
	if result, err := handleExit(context.Background(), nil, Invocation{}); err != nil || result != ExitResult {
		t.Fatalf("handleExit result=%#v err=%v", result, err)
	}

	result, err := handleClear(context.Background(), nil, Invocation{})
	if err != nil {
		t.Fatalf("handleClear returned error: %v", err)
	}
	if result == nil || result.Message != "clear" {
		t.Fatalf("unexpected clear result: %#v", result)
	}

	result, err = handleHelp(context.Background(), nil, Invocation{})
	if err != nil {
		t.Fatalf("handleHelp returned error: %v", err)
	}
	if result == nil || result.Type != ResultHelp {
		t.Fatalf("unexpected help result: %#v", result)
	}
}

func TestArgumentValidationPaths(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		call   func() (*Result, error)
		errSub string
	}{
		{
			name: "start requires ref",
			call: func() (*Result, error) {
				return handleStart(context.Background(), cond, Invocation{})
			},
			errSub: "start requires a reference",
		},
		{
			name: "answer requires response",
			call: func() (*Result, error) {
				return handleAnswer(context.Background(), cond, Invocation{})
			},
			errSub: "answer requires a response",
		},
		{
			name: "note requires message",
			call: func() (*Result, error) {
				return handleNote(context.Background(), cond, Invocation{})
			},
			errSub: "note requires a message",
		},
		{
			name: "quick requires description",
			call: func() (*Result, error) {
				return handleQuick(context.Background(), cond, Invocation{})
			},
			errSub: "quick requires a description",
		},
		{
			name: "implement review missing number",
			call: func() (*Result, error) {
				return handleImplement(context.Background(), cond, Invocation{Args: []string{"review"}})
			},
			errSub: "usage: implement review <number>",
		},
		{
			name: "implement review number not int",
			call: func() (*Result, error) {
				return handleImplement(context.Background(), cond, Invocation{Args: []string{"review", "x"}})
			},
			errSub: "review number must be an integer",
		},
		{
			name: "implement review number not positive",
			call: func() (*Result, error) {
				return handleImplement(context.Background(), cond, Invocation{Args: []string{"review", "0"}})
			},
			errSub: "review number must be positive",
		},
		{
			name: "review view requires number",
			call: func() (*Result, error) {
				return handleReview(context.Background(), cond, Invocation{Args: []string{"view", "x"}})
			},
			errSub: "review view requires a number",
		},
		{
			name: "spec number must be integer",
			call: func() (*Result, error) {
				return handleSpecification(context.Background(), cond, Invocation{Args: []string{"x"}})
			},
			errSub: "no workspace available",
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

func TestHandlersWithoutWorkspaceOrTask(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name string
		call func() (*Result, error)
	}{
		{
			name: "handleCost no task",
			call: func() (*Result, error) {
				return handleCost(context.Background(), cond, Invocation{})
			},
		},
		{
			name: "handleContinue no task",
			call: func() (*Result, error) {
				return handleContinue(context.Background(), cond, Invocation{})
			},
		},
		{
			name: "handleAuto no task",
			call: func() (*Result, error) {
				return handleAuto(context.Background(), cond, Invocation{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.call()
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if !errors.Is(err, ErrNoActiveTask) {
				t.Fatalf("expected ErrNoActiveTask, got %v", err)
			}
		})
	}

	for _, tt := range []struct {
		name   string
		call   func() (*Result, error)
		errSub string
	}{
		{
			name: "handleBudget no workspace",
			call: func() (*Result, error) {
				return handleBudget(context.Background(), cond, Invocation{})
			},
			errSub: "no workspace available",
		},
		{
			name: "handleList no workspace",
			call: func() (*Result, error) {
				return handleList(context.Background(), cond, Invocation{})
			},
			errSub: "no workspace available",
		},
		{
			name: "handleSpecification no workspace",
			call: func() (*Result, error) {
				return handleSpecification(context.Background(), cond, Invocation{})
			},
			errSub: "no workspace available",
		},
		{
			name: "handleReviewView no workspace",
			call: func() (*Result, error) {
				return handleReviewView(context.Background(), cond, 1)
			},
			errSub: "no workspace available",
		},
		{
			name: "handleStatus not initialized",
			call: func() (*Result, error) {
				return handleStatus(context.Background(), cond, Invocation{})
			},
			errSub: "failed to get status",
		},
		{
			name: "handleUndo not initialized",
			call: func() (*Result, error) {
				return handleUndo(context.Background(), cond, Invocation{})
			},
			errSub: "undo failed",
		},
		{
			name: "handleRedo not initialized",
			call: func() (*Result, error) {
				return handleRedo(context.Background(), cond, Invocation{})
			},
			errSub: "redo failed",
		},
		{
			name: "handlePlan not initialized",
			call: func() (*Result, error) {
				return handlePlan(context.Background(), cond, Invocation{})
			},
			errSub: "planning failed",
		},
		{
			name: "handleFinish not initialized",
			call: func() (*Result, error) {
				return handleFinish(context.Background(), cond, Invocation{})
			},
			errSub: "finish failed",
		},
		{
			name: "handleAbandon not initialized",
			call: func() (*Result, error) {
				return handleAbandon(context.Background(), cond, Invocation{})
			},
			errSub: "abandon failed",
		},
	} {
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
