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
)

func TestResetCommand_Properties(t *testing.T) {
	if resetCmd.Use != "reset" {
		t.Errorf("Use = %q, want %q", resetCmd.Use, "reset")
	}

	if resetCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if resetCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if resetCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestResetCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "yes flag",
			flagName:     "yes",
			shorthand:    "y",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := resetCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			// Check default value
			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			// Check shorthand if specified
			if tt.shorthand != "" {
				shorthand := resetCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestResetCommand_ShortDescription(t *testing.T) {
	expected := "Reset workflow state to idle without losing work"
	if resetCmd.Short != expected {
		t.Errorf("Short = %q, want %q", resetCmd.Short, expected)
	}
}

func TestResetCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"workflow state",
		"idle",
		"preserves",
		"specifications",
		"notes",
		"code changes",
	}

	for _, substr := range contains {
		if !containsString(resetCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestResetCommand_DocumentsUseCases(t *testing.T) {
	useCases := []string{
		"Agent hangs",
		"stuck in planning",
		"retry a step",
	}

	for _, useCase := range useCases {
		if !containsString(resetCmd.Long, useCase) {
			t.Errorf("Long description does not document use case %q", useCase)
		}
	}
}

func TestResetCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr reset",
		"--yes",
	}

	for _, example := range examples {
		if !containsString(resetCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestResetCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "reset" {
			found = true

			break
		}
	}
	if !found {
		t.Error("reset command not registered in root command")
	}
}

func TestResetCommand_NoAliases(t *testing.T) {
	if len(resetCmd.Aliases) > 0 {
		t.Errorf("reset command should have no aliases, got %v", resetCmd.Aliases)
	}
}

// --- Behavioral tests using MockConductor ---

func TestRunResetLogic_NoActiveTask(t *testing.T) {
	// When no active task exists, runResetLogic should return an error
	mock := helper_test.NewMockConductor()
	// ActiveTask is nil by default

	err := runResetLogic(context.Background(), mock, true)

	if err == nil {
		t.Error("runResetLogic() expected error for no active task, got nil")
	}

	if !containsString(err.Error(), "no active task") {
		t.Errorf("error = %q, want to contain 'no active task'", err.Error())
	}
}

func TestRunResetLogic_AlreadyIdle(t *testing.T) {
	// When state is already idle, runResetLogic should return nil (no-op)
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{
			ID:    "test-task",
			State: "idle",
		}).
		WithStatus(&conductor.TaskStatus{
			TaskID: "test-task",
			State:  "idle",
		})

	err := runResetLogic(context.Background(), mock, true)
	if err != nil {
		t.Errorf("runResetLogic() unexpected error: %v", err)
	}

	// ResetState should NOT be called when already idle
	if mock.ResetStateCalls != 0 {
		t.Errorf("ResetState was called %d times, want 0", mock.ResetStateCalls)
	}
}

func TestRunResetLogic_CallsResetState(t *testing.T) {
	// When in non-idle state, runResetLogic should call ResetState
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{
			ID:    "test-task",
			State: "implementing",
		}).
		WithStatus(&conductor.TaskStatus{
			TaskID: "test-task",
			State:  "implementing",
		})

	err := runResetLogic(context.Background(), mock, true) // skipConfirm=true
	if err != nil {
		t.Errorf("runResetLogic() unexpected error: %v", err)
	}

	if mock.ResetStateCalls != 1 {
		t.Errorf("ResetState was called %d times, want 1", mock.ResetStateCalls)
	}
}

func TestRunResetLogic_PropagatesStatusError(t *testing.T) {
	// When Status() returns an error, runResetLogic should propagate it
	statusErr := errors.New("database connection failed")
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{
			ID:    "test-task",
			State: "implementing",
		}).
		WithStatusError(statusErr)

	err := runResetLogic(context.Background(), mock, true)

	if err == nil {
		t.Error("runResetLogic() expected error from Status, got nil")
	}

	if !errors.Is(err, statusErr) {
		t.Errorf("error = %v, want wrapped %v", err, statusErr)
	}
}

func TestRunResetLogic_PropagatesResetError(t *testing.T) {
	// When ResetState() returns an error, runResetLogic should propagate it
	resetErr := errors.New("reset operation failed")
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{
			ID:    "test-task",
			State: "implementing",
		}).
		WithStatus(&conductor.TaskStatus{
			TaskID: "test-task",
			State:  "implementing",
		}).
		WithResetStateError(resetErr)

	err := runResetLogic(context.Background(), mock, true)

	if err == nil {
		t.Error("runResetLogic() expected error from ResetState, got nil")
	}

	if !errors.Is(err, resetErr) {
		t.Errorf("error = %v, want wrapped %v", err, resetErr)
	}
}
