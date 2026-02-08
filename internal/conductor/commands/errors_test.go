package commands

import (
	"context"
	"testing"

	"github.com/valksor/go-mehrhof/internal/conductor"
)

func TestClassifyError(t *testing.T) {
	base := &Result{State: "planning", TaskID: "task-1"}

	cases := []struct {
		name string
		err  error
		want ResultType
	}{
		{name: "pending question", err: conductor.ErrPendingQuestion, want: ResultWaiting},
		{name: "budget paused", err: conductor.ErrBudgetPaused, want: ResultPaused},
		{name: "budget stopped", err: conductor.ErrBudgetStopped, want: ResultStopped},
		{name: "cancelled", err: context.Canceled, want: ResultMessage},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyError(base, tc.err)
			if got == nil {
				t.Fatalf("classifyError returned nil")
			}
			if got.Type != tc.want {
				t.Fatalf("classifyError type = %q, want %q", got.Type, tc.want)
			}
			if got.State != base.State || got.TaskID != base.TaskID {
				t.Fatalf("state/task mismatch: %#v", got)
			}
		})
	}
}

func TestEnrichWaitingResultNoOp(t *testing.T) {
	EnrichWaitingResult(nil, nil)
	EnrichWaitingResult(&Result{Type: ResultMessage}, nil)
}
