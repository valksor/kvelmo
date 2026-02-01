//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/stack"
)

func TestStackCommand_Properties(t *testing.T) {
	if stackCmd.Use != "stack" {
		t.Errorf("Use = %q, want %q", stackCmd.Use, "stack")
	}

	if stackCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if stackCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if stackCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestStackCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{
			name:         "graph flag",
			flagName:     "graph",
			defaultValue: "false",
		},
		{
			name:         "mermaid flag",
			flagName:     "mermaid",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := stackCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestStackCommand_Subcommands(t *testing.T) {
	subcommands := stackCmd.Commands()

	expectedNames := []string{"rebase", "sync"}
	for _, name := range expectedNames {
		found := false
		for _, cmd := range subcommands {
			if cmd.Name() == name {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("stack command missing %q subcommand", name)
		}
	}
}

func TestStackCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "stack" {
			found = true

			break
		}
	}
	if !found {
		t.Error("stack command not registered in root command")
	}
}

func TestGetStateIcon(t *testing.T) {
	tests := []struct {
		name     string
		state    stack.StackState
		expected string
	}{
		{"merged", stack.StateMerged, "✓"},
		{"needs rebase", stack.StateNeedsRebase, "⟳"},
		{"conflict", stack.StateConflict, "✗"},
		{"pending review", stack.StatePendingReview, "◐"},
		{"approved", stack.StateApproved, "◉"},
		{"abandoned", stack.StateAbandoned, "⊘"},
		{"active", stack.StateActive, "●"},
		{"unknown state", stack.StackState("unknown"), "●"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStateIcon(tt.state)
			if got != tt.expected {
				t.Errorf("getStateIcon(%q) = %q, want %q", tt.state, got, tt.expected)
			}
		})
	}
}

func TestStateToGraphStatus(t *testing.T) {
	tests := []struct {
		name     string
		state    stack.StackState
		expected string
	}{
		{"merged", stack.StateMerged, "done"},
		{"active", stack.StateActive, "in_progress"},
		{"needs rebase", stack.StateNeedsRebase, "blocked"},
		{"conflict", stack.StateConflict, "blocked"},
		{"pending review", stack.StatePendingReview, "pending"},
		{"approved", stack.StateApproved, "pending"},
		{"abandoned", stack.StateAbandoned, "pending"},
		{"unknown", stack.StackState("unknown"), "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stateToGraphStatus(tt.state)
			if got != tt.expected {
				t.Errorf("stateToGraphStatus(%q) = %q, want %q", tt.state, got, tt.expected)
			}
		})
	}
}

func TestHandleRebaseResult_Nil(t *testing.T) {
	testErr := errors.New("test error")
	err := handleRebaseResult(nil, testErr)
	if !errors.Is(err, testErr) {
		t.Errorf("handleRebaseResult(nil, err) = %v, want %v", err, testErr)
	}
}

func TestHandleRebaseResult_Success(t *testing.T) {
	// Capture stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	result := &stack.RebaseResult{
		RebasedTasks: []stack.RebaseTaskResult{
			{TaskID: "task-1", NewBase: "main"},
			{TaskID: "task-2", NewBase: "main"},
		},
	}

	err := handleRebaseResult(result, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("handleRebaseResult() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !containsString(output, "task-1") {
		t.Errorf("output should contain 'task-1', got %q", output)
	}

	if !containsString(output, "task-2") {
		t.Errorf("output should contain 'task-2', got %q", output)
	}
}

func TestHandleRebaseResult_WithSkipped(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	result := &stack.RebaseResult{
		SkippedTasks: []stack.SkippedTask{
			{TaskID: "task-3", Reason: "already up to date"},
		},
	}

	_ = handleRebaseResult(result, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !containsString(output, "task-3") {
		t.Errorf("output should contain 'task-3', got %q", output)
	}

	if !containsString(output, "skipped") {
		t.Errorf("output should contain 'skipped', got %q", output)
	}
}

func TestHandleRebaseResult_Conflict(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	result := &stack.RebaseResult{
		FailedTask: &stack.FailedRebase{
			TaskID:       "task-4",
			Branch:       "feature/task-4",
			OntoBase:     "main",
			IsConflict:   true,
			ConflictHint: "Resolve conflicts in file.go",
		},
	}

	err := handleRebaseResult(result, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Fatal("handleRebaseResult() should return error for failed rebase")
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !containsString(output, "task-4") {
		t.Errorf("output should contain 'task-4', got %q", output)
	}

	if !containsString(output, "conflict") {
		t.Errorf("output should contain 'conflict', got %q", output)
	}

	if !containsString(output, "Resolve conflicts") {
		t.Errorf("output should contain hint, got %q", output)
	}
}

func TestHandleRebaseResult_ErrorWithoutConflict(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	result := &stack.RebaseResult{
		FailedTask: &stack.FailedRebase{
			TaskID: "task-5",
			Branch: "feature/task-5",
			Error:  errors.New("git error"),
		},
	}

	err := handleRebaseResult(result, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Fatal("handleRebaseResult() should return error for failed rebase")
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !containsString(output, "task-5") {
		t.Errorf("output should contain 'task-5', got %q", output)
	}
}

func TestRunStack_NoStacks(t *testing.T) {
	_ = NewTestContext(t)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runStack(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runStack() error = %v", err)
	}

	if !containsString(output, "No stacked features found") {
		t.Errorf("output should contain 'No stacked features found', got %q", output)
	}
}
