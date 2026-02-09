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

func TestAbandonCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		description  string
		defaultValue bool
	}{
		{
			name:         "yes flag",
			flagName:     "yes",
			shorthand:    "y",
			defaultValue: false,
			description:  "Skip confirmation prompt",
		},
		{
			name:         "keep-branch flag",
			flagName:     "keep-branch",
			defaultValue: false,
			description:  "Keep the git branch",
		},
		{
			name:         "keep-work flag",
			flagName:     "keep-work",
			defaultValue: false,
			description:  "Keep the work directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := abandonCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			// Check default value
			if flag.DefValue != "false" {
				t.Errorf("flag %q default value = %q, want false", tt.flagName, flag.DefValue)
			}

			// Check shorthand if specified
			if tt.shorthand != "" {
				shorthand := abandonCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestAbandonCommand_Properties(t *testing.T) {
	// Check command is properly configured
	if abandonCmd.Use != "abandon" {
		t.Errorf("Use = %q, want %q", abandonCmd.Use, "abandon")
	}

	if abandonCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if abandonCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if abandonCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

// --- Behavioral tests using MockConductor ---

func TestRunAbandonLogic_NoActiveTask(t *testing.T) {
	// When no active task exists, runAbandonLogic should return an error
	mock := helper_test.NewMockConductor()
	// ActiveTask is nil by default

	opts := abandonOptions{skipConfirm: true}
	err := runAbandonLogic(context.Background(), mock, opts)

	if err == nil {
		t.Error("runAbandonLogic() expected error for no active task, got nil")
	}

	if !containsString(err.Error(), "no active task") {
		t.Errorf("error = %q, want to contain 'no active task'", err.Error())
	}
}

func TestRunAbandonLogic_CallsDelete(t *testing.T) {
	// When active task exists, runAbandonLogic should call Delete
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{
			ID:    "test-task",
			State: "implementing",
		}).
		WithStatus(&conductor.TaskStatus{
			TaskID: "test-task",
			State:  "implementing",
		})

	opts := abandonOptions{skipConfirm: true}
	err := runAbandonLogic(context.Background(), mock, opts)
	if err != nil {
		t.Errorf("runAbandonLogic() unexpected error: %v", err)
	}

	if len(mock.DeleteCalls) != 1 {
		t.Errorf("Delete was called %d times, want 1", len(mock.DeleteCalls))
	}
}

func TestRunAbandonLogic_PassesDeleteOptions(t *testing.T) {
	// Verify options are correctly passed to Delete
	tests := []struct {
		name           string
		opts           abandonOptions
		wantForce      bool
		wantKeepBranch bool
		wantDeleteWork *bool
	}{
		{
			name:           "default options",
			opts:           abandonOptions{skipConfirm: true},
			wantForce:      true,
			wantKeepBranch: false,
			wantDeleteWork: nil, // nil means defer to config
		},
		{
			name:           "keep branch",
			opts:           abandonOptions{skipConfirm: true, keepBranch: true},
			wantForce:      true,
			wantKeepBranch: true,
			wantDeleteWork: nil,
		},
		{
			name:           "keep work explicitly set",
			opts:           abandonOptions{skipConfirm: true, keepWorkChanged: true, keepWork: true},
			wantForce:      true,
			wantKeepBranch: false,
			wantDeleteWork: conductor.BoolPtr(false), // keepWork=true means DeleteWork=false
		},
		{
			name:           "delete work explicitly set",
			opts:           abandonOptions{skipConfirm: true, keepWorkChanged: true, keepWork: false},
			wantForce:      true,
			wantKeepBranch: false,
			wantDeleteWork: conductor.BoolPtr(true), // keepWork=false means DeleteWork=true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := helper_test.NewMockConductor().
				WithActiveTask(&storage.ActiveTask{ID: "test-task", State: "idle"}).
				WithStatus(&conductor.TaskStatus{TaskID: "test-task", State: "idle"})

			err := runAbandonLogic(context.Background(), mock, tt.opts)
			if err != nil {
				t.Fatalf("runAbandonLogic() unexpected error: %v", err)
			}

			if len(mock.DeleteCalls) != 1 {
				t.Fatalf("Delete called %d times, want 1", len(mock.DeleteCalls))
			}

			got := mock.DeleteCalls[0]

			if got.Force != tt.wantForce {
				t.Errorf("DeleteOptions.Force = %v, want %v", got.Force, tt.wantForce)
			}

			if got.KeepBranch != tt.wantKeepBranch {
				t.Errorf("DeleteOptions.KeepBranch = %v, want %v", got.KeepBranch, tt.wantKeepBranch)
			}

			if tt.wantDeleteWork == nil {
				if got.DeleteWork != nil {
					t.Errorf("DeleteOptions.DeleteWork = %v, want nil", *got.DeleteWork)
				}
			} else {
				if got.DeleteWork == nil {
					t.Errorf("DeleteOptions.DeleteWork = nil, want %v", *tt.wantDeleteWork)
				} else if *got.DeleteWork != *tt.wantDeleteWork {
					t.Errorf("DeleteOptions.DeleteWork = %v, want %v", *got.DeleteWork, *tt.wantDeleteWork)
				}
			}
		})
	}
}

func TestRunAbandonLogic_PropagatesDeleteError(t *testing.T) {
	// When Delete() returns an error, runAbandonLogic should propagate it
	deleteErr := errors.New("delete operation failed")
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test-task", State: "idle"}).
		WithStatus(&conductor.TaskStatus{TaskID: "test-task", State: "idle"}).
		WithDeleteError(deleteErr)

	opts := abandonOptions{skipConfirm: true}
	err := runAbandonLogic(context.Background(), mock, opts)

	if err == nil {
		t.Error("runAbandonLogic() expected error from Delete, got nil")
	}

	if !errors.Is(err, deleteErr) {
		t.Errorf("error = %v, want wrapped %v", err, deleteErr)
	}
}

func TestRunAbandonLogic_PropagatesStatusError(t *testing.T) {
	// When Status() returns an error, runAbandonLogic should propagate it
	statusErr := errors.New("status check failed")
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test-task", State: "idle"}).
		WithStatusError(statusErr)

	opts := abandonOptions{skipConfirm: true}
	err := runAbandonLogic(context.Background(), mock, opts)

	if err == nil {
		t.Error("runAbandonLogic() expected error from Status, got nil")
	}

	if !errors.Is(err, statusErr) {
		t.Errorf("error = %v, want wrapped %v", err, statusErr)
	}
}
