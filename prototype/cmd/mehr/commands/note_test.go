//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

func TestNoteCommand_Properties(t *testing.T) {
	// Check command is properly configured
	if noteCmd.Use != "note [message]" {
		t.Errorf("Use = %q, want %q", noteCmd.Use, "note [message]")
	}

	if noteCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if noteCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if noteCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestNoteCommand_HasAnswerAlias(t *testing.T) {
	// The "answer" alias is kept for semantic distinction:
	// - "note" = add context/requirements
	// - "answer" = respond to agent questions
	if len(noteCmd.Aliases) != 1 || noteCmd.Aliases[0] != "answer" {
		t.Errorf("note command should have 'answer' alias, got %v", noteCmd.Aliases)
	}
}

func TestNoteCommand_ShortDescription(t *testing.T) {
	expected := "Add notes to the task or answer agent questions"
	if noteCmd.Short != expected {
		t.Errorf("Short = %q, want %q", noteCmd.Short, expected)
	}
}

func TestNoteCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"Add notes",
		"notes.md",
		"work directory",
		"agent runs",
		"pending",
	}

	for _, substr := range contains {
		if !containsString(noteCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestNoteCommand_ExamplesContains(t *testing.T) {
	examples := []string{
		"mehr note",
		`"Use PostgreSQL"`,
		`"Add error handling"`,
	}

	for _, example := range examples {
		if !containsString(noteCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestNoteCommand_RegisteredInRoot(t *testing.T) {
	// Verify noteCmd is a subcommand of rootCmd
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "note [message]" {
			found = true

			break
		}
	}
	if !found {
		t.Error("note command not registered in root command")
	}
}

func TestNoteCommand_InteractiveModeDocumented(t *testing.T) {
	// Interactive mode should be documented
	if !containsString(noteCmd.Long, "interactive mode") {
		t.Error("Long description does not mention interactive mode")
	}
}

func TestNoteCommand_NoFlags(t *testing.T) {
	// Note command doesn't have flags in the current implementation
	// Verify no unexpected flags were added
	flags := noteCmd.Flags()

	// Only inherited flags should be present (like --help)
	localFlags := noteCmd.LocalFlags()
	localNonPersistent := localFlags.NFlag()

	if localNonPersistent > 0 {
		// If flags are added in the future, this test documents them
		t.Logf("Note: noteCmd has %d local flags", localNonPersistent)
	}

	// Check that common flags like --verbose are not local to this command
	if flags.Lookup("verbose") != nil && localFlags.Lookup("verbose") != nil {
		t.Error("verbose should be a persistent flag from root, not local")
	}
}

func TestNoteCommand_UsesWorkDirectory(t *testing.T) {
	// The command should document that it saves it to the work directory
	if !containsString(noteCmd.Long, "work directory") {
		t.Error("Long description does not mention work directory")
	}
}
