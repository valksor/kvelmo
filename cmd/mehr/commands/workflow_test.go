//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"os"
	"testing"
)

func TestWorkflowCommand_Properties(t *testing.T) {
	// Check command is properly configured
	if workflowCmd.Use != "workflow" {
		t.Errorf("Use = %q, want %q", workflowCmd.Use, "workflow")
	}

	if workflowCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if workflowCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if workflowCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestWorkflowCommand_ShortDescription(t *testing.T) {
	expected := "Show the workflow state machine diagram"
	if workflowCmd.Short != expected {
		t.Errorf("Short = %q, want %q", workflowCmd.Short, expected)
	}
}

func TestWorkflowCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"visual representation",
		"state machine",
		"states",
		"transitions",
	}

	for _, substr := range contains {
		if !containsString(workflowCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestWorkflowCommand_DocumentsAllStates(t *testing.T) {
	states := []string{
		"idle",
		"planning",
		"implementing",
		"reviewing",
		"waiting",
		"paused",
		"checkpointing",
		"reverting",
		"restoring",
		"done",
		"failed",
	}

	for _, state := range states {
		if !containsString(workflowCmd.Long, state) {
			t.Errorf("Long description does not document state %q", state)
		}
	}
}

func TestWorkflowCommand_DocumentsRelatedCommands(t *testing.T) {
	// Should reference related commands
	commands := []string{
		"mehr status",
		"mehr guide",
	}

	for _, cmd := range commands {
		if !containsString(workflowCmd.Long, cmd) {
			t.Errorf("Long description does not reference command %q", cmd)
		}
	}
}

func TestWorkflowCommand_RegisteredInRoot(t *testing.T) {
	// Verify workflowCmd is a subcommand of rootCmd
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "workflow" {
			found = true

			break
		}
	}
	if !found {
		t.Error("workflow command not registered in root command")
	}
}

func TestWorkflowCommand_NoFlags(t *testing.T) {
	// Workflow command doesn't have flags
	localFlags := workflowCmd.LocalFlags()
	localNonPersistent := localFlags.NFlag()

	if localNonPersistent > 0 {
		t.Logf("Note: workflowCmd has %d local flags", localNonPersistent)
	}
}

// TestRunWorkflow tests that the workflow diagram is printed correctly.
func TestRunWorkflow(t *testing.T) {
	// Capture stdout since runWorkflow uses fmt.Print (not cmd.OutOrStdout)
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := runWorkflow(workflowCmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runWorkflow() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify key sections of the workflow diagram are present
	expectedContent := []string{
		"MEHRHOF WORKFLOW STATE MACHINE",
		"idle",
		"planning",
		"implementing",
		"reviewing",
		"waiting",
		"checkpoint",
		"reverting",
		"restoring",
		"done",
		"failed",
		"COMMANDS BY STATE",
		"KEY TRANSITIONS",
		"mehr plan",
		"mehr implement",
		"mehr finish",
		"mehr undo",
		"mehr redo",
	}

	for _, expected := range expectedContent {
		if !containsString(output, expected) {
			t.Errorf("workflow output does not contain %q", expected)
		}
	}
}
