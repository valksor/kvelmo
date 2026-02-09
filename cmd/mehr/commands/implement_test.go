//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/helper_test"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// Note: TestImplementCommand_Aliases is in common_test.go

func TestImplementCommand_Properties(t *testing.T) {
	if implementCmd.Use != "implement [review <number>]" {
		t.Errorf("Use = %q, want %q", implementCmd.Use, "implement [review <number>]")
	}

	if implementCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if implementCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if implementCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestImplementCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "dry-run flag",
			flagName:     "dry-run",
			shorthand:    "n",
			defaultValue: "false",
		},
		{
			name:         "agent-implement flag",
			flagName:     "agent-implement",
			shorthand:    "",
			defaultValue: "",
		},
		{
			name:         "force flag",
			flagName:     "force",
			shorthand:    "",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := implementCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			if tt.shorthand != "" {
				shorthand := implementCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestImplementCommand_ShortDescription(t *testing.T) {
	expected := "Implement the specifications for the active task"
	if implementCmd.Short != expected {
		t.Errorf("Short = %q, want %q", implementCmd.Short, expected)
	}
}

func TestImplementCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"implementation phase",
		"generate code",
		"specifications",
		"mehr plan",
	}

	for _, substr := range contains {
		if !containsString(implementCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestImplementCommand_NoAliases(t *testing.T) {
	// Aliases removed in favor of prefix matching
	if len(implementCmd.Aliases) > 0 {
		t.Errorf("implement command should have no aliases, got %v", implementCmd.Aliases)
	}
}

func TestImplementCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "implement [review <number>]" {
			found = true

			break
		}
	}
	if !found {
		t.Error("implement command not registered in root command")
	}
}

func TestImplementCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr implement",
		"--dry-run",
		"--verbose",
	}

	for _, example := range examples {
		if !containsString(implementCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestImplementCommand_DryRunHasShorthand(t *testing.T) {
	flag := implementCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("dry-run flag not found")

		return
	}
	if flag.Shorthand != "n" {
		t.Errorf("dry-run flag shorthand = %q, want 'n'", flag.Shorthand)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// runImplementLogic behavioral tests
// ──────────────────────────────────────────────────────────────────────────────

func TestRunImplementLogic_NoActiveTask(t *testing.T) {
	mock := helper_test.NewMockConductor()
	// No active task set

	opts := implementOptions{}
	err := runImplementLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error for no active task")
	}
	if err != nil && err.Error() != "no active task" {
		t.Errorf("error = %q, want %q", err.Error(), "no active task")
	}
}

func TestRunImplementLogic_CallsImplement(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "planning"})

	opts := implementOptions{}
	err := runImplementLogic(context.Background(), mock, opts, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ImplementCalls != 1 {
		t.Errorf("Implement called %d times, want 1", mock.ImplementCalls)
	}
	if mock.RunImplementationCalls != 1 {
		t.Errorf("RunImplementation called %d times, want 1", mock.RunImplementationCalls)
	}
}

func TestRunImplementLogic_ForceResetsState(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"})

	var stdout bytes.Buffer
	opts := implementOptions{force: true}
	err := runImplementLogic(context.Background(), mock, opts, &stdout)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ResetStateCalls != 1 {
		t.Errorf("ResetState called %d times, want 1", mock.ResetStateCalls)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("reset")) {
		t.Error("expected output to contain 'reset'")
	}
}

func TestRunImplementLogic_PropagatesImplementError(t *testing.T) {
	implementErr := errors.New("failed to enter implement state")
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "planning"}).
		WithImplementError(implementErr)

	opts := implementOptions{}
	err := runImplementLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, implementErr) {
		t.Errorf("error = %v, want wrapped %v", err, implementErr)
	}
}

func TestRunImplementLogic_PropagatesRunImplementationError(t *testing.T) {
	runErr := errors.New("agent failed")
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "planning"}).
		WithRunImplementationError(runErr)

	opts := implementOptions{}
	err := runImplementLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, runErr) {
		t.Errorf("error = %v, want wrapped %v", err, runErr)
	}
}

func TestRunImplementLogic_HandlesBudgetPaused(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "planning"}).
		WithRunImplementationError(conductor.ErrBudgetPaused)

	opts := implementOptions{}
	err := runImplementLogic(context.Background(), mock, opts, nil)
	// Budget pause returns the sentinel error for caller to handle
	if !errors.Is(err, conductor.ErrBudgetPaused) {
		t.Errorf("expected ErrBudgetPaused, got: %v", err)
	}
}

func TestRunImplementLogic_HandlesBudgetStopped(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "planning"}).
		WithRunImplementationError(conductor.ErrBudgetStopped)

	opts := implementOptions{}
	err := runImplementLogic(context.Background(), mock, opts, nil)
	// Budget stop returns the sentinel error for caller to handle
	if !errors.Is(err, conductor.ErrBudgetStopped) {
		t.Errorf("expected ErrBudgetStopped, got: %v", err)
	}
}

func TestRunImplementLogic_PropagatesResetStateError(t *testing.T) {
	resetErr := errors.New("reset failed")
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"}).
		WithResetStateError(resetErr)

	opts := implementOptions{force: true}
	err := runImplementLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, resetErr) {
		t.Errorf("error = %v, want wrapped %v", err, resetErr)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// runImplementReviewLogic behavioral tests
// ──────────────────────────────────────────────────────────────────────────────

func TestRunImplementReviewLogic_InvalidNumber(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"})

	tests := []struct {
		name   string
		number int
	}{
		{"zero", 0},
		{"negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := implementReviewOptions{reviewNumber: tt.number}
			err := runImplementReviewLogic(context.Background(), mock, opts, nil)
			if err == nil {
				t.Error("expected error for invalid review number")
			}
		})
	}
}

func TestRunImplementReviewLogic_NoActiveTask(t *testing.T) {
	mock := helper_test.NewMockConductor()
	// No active task set

	opts := implementReviewOptions{reviewNumber: 1}
	err := runImplementReviewLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error for no active task")
	}
	if err != nil && err.Error() != "no active task" {
		t.Errorf("error = %q, want %q", err.Error(), "no active task")
	}
}

func TestRunImplementReviewLogic_CallsImplementReview(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"})

	opts := implementReviewOptions{reviewNumber: 2}
	err := runImplementReviewLogic(context.Background(), mock, opts, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(mock.ImplementReviewCalls) != 1 {
		t.Errorf("ImplementReview called %d times, want 1", len(mock.ImplementReviewCalls))
	}
	if mock.ImplementReviewCalls[0] != 2 {
		t.Errorf("ImplementReview called with %d, want 2", mock.ImplementReviewCalls[0])
	}
	if len(mock.RunReviewImplementationCalls) != 1 {
		t.Errorf("RunReviewImplementation called %d times, want 1", len(mock.RunReviewImplementationCalls))
	}
	if mock.RunReviewImplementationCalls[0] != 2 {
		t.Errorf("RunReviewImplementation called with %d, want 2", mock.RunReviewImplementationCalls[0])
	}
}

func TestRunImplementReviewLogic_ForceResetsState(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"})

	var stdout bytes.Buffer
	opts := implementReviewOptions{reviewNumber: 1, force: true}
	err := runImplementReviewLogic(context.Background(), mock, opts, &stdout)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.ResetStateCalls != 1 {
		t.Errorf("ResetState called %d times, want 1", mock.ResetStateCalls)
	}
}

func TestRunImplementReviewLogic_PropagatesImplementReviewError(t *testing.T) {
	reviewErr := errors.New("failed to enter review implementation state")
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"}).
		WithImplementReviewError(reviewErr)

	opts := implementReviewOptions{reviewNumber: 1}
	err := runImplementReviewLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, reviewErr) {
		t.Errorf("error = %v, want wrapped %v", err, reviewErr)
	}
}

func TestRunImplementReviewLogic_PropagatesRunError(t *testing.T) {
	runErr := errors.New("review implementation agent failed")
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"}).
		WithRunReviewImplementationError(runErr)

	opts := implementReviewOptions{reviewNumber: 1}
	err := runImplementReviewLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, runErr) {
		t.Errorf("error = %v, want wrapped %v", err, runErr)
	}
}
