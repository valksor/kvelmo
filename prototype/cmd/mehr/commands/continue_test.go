//go:build !testbinary
// +build !testbinary

package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/helper_test"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

func TestContinueCommand_Properties(t *testing.T) {
	if continueCmd.Use != "continue" {
		t.Errorf("Use = %q, want %q", continueCmd.Use, "continue")
	}

	if continueCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if continueCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if continueCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestContinueCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "auto flag",
			flagName:     "auto",
			shorthand:    "",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := continueCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			if tt.shorthand != "" {
				shorthand := continueCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestContinueCommand_ShortDescription(t *testing.T) {
	expected := "Resume work on task"
	if continueCmd.Short != expected {
		t.Errorf("Short = %q, want %q", continueCmd.Short, expected)
	}
}

func TestContinueCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"auto-pilot",
		"RELATED COMMANDS",
		"AUTO-EXECUTION LOGIC",
		"specifications",
	}

	for _, substr := range contains {
		if !containsString(continueCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestContinueCommand_DocumentsWhenToUse(t *testing.T) {
	// Should document when to use a guide, status, continue
	comparisons := []string{
		"guide",
		"status",
		"continue",
	}

	for _, comp := range comparisons {
		if !containsString(continueCmd.Long, comp) {
			t.Errorf("Long description does not document comparison with %q", comp)
		}
	}
}

func TestContinueCommand_NoAliases(t *testing.T) {
	// Aliases removed in favor of prefix matching
	if len(continueCmd.Aliases) > 0 {
		t.Errorf("continue command should have no aliases, got %v", continueCmd.Aliases)
	}
}

func TestContinueCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr continue",
		"--auto",
	}

	for _, example := range examples {
		if !containsString(continueCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestContinueCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "continue" {
			found = true

			break
		}
	}
	if !found {
		t.Error("continue command not registered in root command")
	}
}

func TestContinueCommand_DocumentsSeeAlso(t *testing.T) {
	// Should reference related commands in CHOOSING THE RIGHT COMMAND section
	if !containsString(continueCmd.Long, "guide") {
		t.Error("Long description does not reference 'guide'")
	}

	if !containsString(continueCmd.Long, "status") {
		t.Error("Long description does not reference 'status'")
	}
}

// --- Behavioral tests ---

func TestExecuteNextStep_StateHandling(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		specs   int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "failed state returns error",
			state:   string(workflow.StateFailed),
			specs:   0,
			wantErr: true,
			errMsg:  "failed state",
		},
		{
			name:    "waiting state returns error",
			state:   string(workflow.StateWaiting),
			specs:   0,
			wantErr: true,
			errMsg:  "waiting for user input",
		},
		{
			name:    "paused state returns error",
			state:   string(workflow.StatePaused),
			specs:   0,
			wantErr: true,
			errMsg:  "paused due to budget",
		},
		{
			name:    "done state no error",
			state:   string(workflow.StateDone),
			specs:   1,
			wantErr: false,
		},
		{
			name:    "implementing state no error",
			state:   string(workflow.StateImplementing),
			specs:   1,
			wantErr: false,
		},
		{
			name:    "reviewing state no error",
			state:   string(workflow.StateReviewing),
			specs:   1,
			wantErr: false,
		},
		{
			name:    "checkpointing state no error",
			state:   string(workflow.StateCheckpointing),
			specs:   1,
			wantErr: false,
		},
		{
			name:    "reverting state no error",
			state:   string(workflow.StateReverting),
			specs:   1,
			wantErr: false,
		},
		{
			name:    "restoring state no error",
			state:   string(workflow.StateRestoring),
			specs:   1,
			wantErr: false,
		},
		{
			name:    "unknown state no error",
			state:   "custom-state",
			specs:   1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &conductor.TaskStatus{
				TaskID:         "test-task",
				State:          tt.state,
				Specifications: tt.specs,
			}

			// Pass nil conductor - these states don't call Plan/Implement
			err := executeNextStep(t.Context(), nil, status)

			if tt.wantErr {
				if err == nil {
					t.Errorf("executeNextStep() expected error, got nil")
				} else if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("executeNextStep() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("executeNextStep() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestExecuteNextStep_ErrorTypes(t *testing.T) {
	// Test that error messages are informative
	tests := []struct {
		name     string
		state    string
		contains string
	}{
		{
			name:     "failed error is descriptive",
			state:    string(workflow.StateFailed),
			contains: "failed",
		},
		{
			name:     "waiting error mentions input",
			state:    string(workflow.StateWaiting),
			contains: "input",
		},
		{
			name:     "paused error mentions budget",
			state:    string(workflow.StatePaused),
			contains: "budget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &conductor.TaskStatus{
				TaskID: "test-task",
				State:  tt.state,
			}

			err := executeNextStep(t.Context(), nil, status)

			if err == nil {
				t.Fatal("expected error")
			}

			if !containsString(err.Error(), tt.contains) {
				t.Errorf("error %q should contain %q", err.Error(), tt.contains)
			}
		})
	}
}

// --- Mock conductor tests ---

func TestExecuteNextStep_CallsPlan(t *testing.T) {
	// When idle with no specs, executeNextStep should call Plan
	mock := helper_test.NewMockConductor()

	status := &conductor.TaskStatus{
		TaskID:         "test-task",
		State:          string(workflow.StateIdle),
		Specifications: 0, // No specs
	}

	err := executeNextStep(context.Background(), mock, status)
	if err != nil {
		t.Errorf("executeNextStep() unexpected error: %v", err)
	}

	if mock.PlanCalls != 1 {
		t.Errorf("Plan was called %d times, want 1", mock.PlanCalls)
	}

	if mock.ImplementCalls != 0 {
		t.Errorf("Implement was called %d times, want 0", mock.ImplementCalls)
	}
}

func TestExecuteNextStep_CallsImplement(t *testing.T) {
	// When idle with specs, executeNextStep should call Implement
	mock := helper_test.NewMockConductor()

	status := &conductor.TaskStatus{
		TaskID:         "test-task",
		State:          string(workflow.StateIdle),
		Specifications: 3, // Has specs
	}

	err := executeNextStep(context.Background(), mock, status)
	if err != nil {
		t.Errorf("executeNextStep() unexpected error: %v", err)
	}

	if mock.ImplementCalls != 1 {
		t.Errorf("Implement was called %d times, want 1", mock.ImplementCalls)
	}

	if mock.PlanCalls != 0 {
		t.Errorf("Plan was called %d times, want 0", mock.PlanCalls)
	}
}

func TestExecuteNextStep_PlanningStateCallsImplement(t *testing.T) {
	// When in planning state, executeNextStep should call Implement
	mock := helper_test.NewMockConductor()

	status := &conductor.TaskStatus{
		TaskID:         "test-task",
		State:          string(workflow.StatePlanning),
		Specifications: 1,
	}

	err := executeNextStep(context.Background(), mock, status)
	if err != nil {
		t.Errorf("executeNextStep() unexpected error: %v", err)
	}

	if mock.ImplementCalls != 1 {
		t.Errorf("Implement was called %d times, want 1", mock.ImplementCalls)
	}
}

func TestExecuteNextStep_PropagatesPlanError(t *testing.T) {
	// When Plan returns an error, executeNextStep should propagate it
	mock := helper_test.NewMockConductor().
		WithPlanError(context.DeadlineExceeded)

	status := &conductor.TaskStatus{
		TaskID:         "test-task",
		State:          string(workflow.StateIdle),
		Specifications: 0,
	}

	err := executeNextStep(context.Background(), mock, status)
	if err == nil {
		t.Error("expected error from Plan")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestExecuteNextStep_PropagatesImplementError(t *testing.T) {
	// When Implement returns an error, executeNextStep should propagate it
	mock := helper_test.NewMockConductor().
		WithImplementError(context.Canceled)

	status := &conductor.TaskStatus{
		TaskID:         "test-task",
		State:          string(workflow.StateIdle),
		Specifications: 2, // Has specs, will call Implement
	}

	err := executeNextStep(context.Background(), mock, status)
	if err == nil {
		t.Error("expected error from Implement")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want %v", err, context.Canceled)
	}
}

// --- Factory injection tests ---

func TestSetConductorFactory_ReturnsRestoreFunction(t *testing.T) {
	// Verify SetConductorFactory returns a working restore function
	originalFactory := conductorFactory

	callCount := 0
	restore := SetConductorFactory(func(ctx context.Context, opts ...conductor.Option) (ConductorAPI, error) {
		callCount++

		return helper_test.NewMockConductor(), nil
	})

	// Verify our factory is now active
	_, _ = CreateConductor(context.Background())
	if callCount != 1 {
		t.Errorf("custom factory was called %d times, want 1", callCount)
	}

	// Restore original
	restore()

	// Verify factory was restored
	if conductorFactory == nil {
		t.Error("conductorFactory is nil after restore")
	}

	// The factory should be back to the original
	// (we can't easily verify this since defaultConductorFactory requires workspace)
	_ = originalFactory // Just for the linter
}

func TestMockConductor_ImplementsInterface(t *testing.T) {
	// Verify MockConductor satisfies ConductorAPI at compile time
	var _ ConductorAPI = (*helper_test.MockConductor)(nil)
}

func TestMockConductor_FluentBuilders(t *testing.T) {
	// Verify fluent builder pattern works correctly
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{
			ID:    "test-123",
			State: "idle",
		}).
		WithStatus(&conductor.TaskStatus{
			TaskID:         "test-123",
			State:          "idle",
			Specifications: 5,
		}).
		WithTaskID("test-123")

	if mock.GetActiveTask() == nil {
		t.Error("GetActiveTask() should not return nil")
	}

	if mock.GetActiveTask().ID != "test-123" {
		t.Errorf("ActiveTask.ID = %q, want %q", mock.GetActiveTask().ID, "test-123")
	}

	status, err := mock.Status(context.Background())
	if err != nil {
		t.Errorf("Status() unexpected error: %v", err)
	}

	if status.Specifications != 5 {
		t.Errorf("Status().Specifications = %d, want 5", status.Specifications)
	}

	if mock.GetTaskID() != "test-123" {
		t.Errorf("GetTaskID() = %q, want %q", mock.GetTaskID(), "test-123")
	}
}
