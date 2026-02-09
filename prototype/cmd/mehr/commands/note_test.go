//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/helper_test"
	"github.com/valksor/go-mehrhof/internal/storage"
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

// --- Behavioral tests ---

func TestFindNoteByNumber(t *testing.T) {
	notes := []storage.Note{
		{Number: 1, Content: "First note", State: "idle"},
		{Number: 2, Content: "Second note", State: "planning"},
		{Number: 3, Content: "Third note", State: "implementing"},
	}

	tests := []struct {
		name     string
		number   int
		wantNil  bool
		wantNote string
	}{
		{
			name:     "find first note",
			number:   1,
			wantNil:  false,
			wantNote: "First note",
		},
		{
			name:     "find middle note",
			number:   2,
			wantNil:  false,
			wantNote: "Second note",
		},
		{
			name:     "find last note",
			number:   3,
			wantNil:  false,
			wantNote: "Third note",
		},
		{
			name:    "note not found - zero",
			number:  0,
			wantNil: true,
		},
		{
			name:    "note not found - too high",
			number:  99,
			wantNil: true,
		},
		{
			name:    "note not found - negative",
			number:  -1,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findNoteByNumber(notes, tt.number)

			if tt.wantNil {
				if result != nil {
					t.Errorf("findNoteByNumber(%d) = %v, want nil", tt.number, result)
				}
			} else {
				if result == nil {
					t.Errorf("findNoteByNumber(%d) = nil, want note", tt.number)
				} else if result.Content != tt.wantNote {
					t.Errorf("findNoteByNumber(%d).Content = %q, want %q", tt.number, result.Content, tt.wantNote)
				}
			}
		})
	}
}

func TestFindNoteByNumber_EmptyList(t *testing.T) {
	result := findNoteByNumber([]storage.Note{}, 1)
	if result != nil {
		t.Errorf("findNoteByNumber on empty list = %v, want nil", result)
	}
}

func TestDisplayNote(t *testing.T) {
	// Capture stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	note := &storage.Note{
		Number:    1,
		Content:   "Test note content",
		State:     "planning",
		Timestamp: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	displayNote(note)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	expected := []string{
		"#1",
		"2026-01-15",
		"10:30",
		"planning",
		"Test note content",
	}

	for _, s := range expected {
		if !strings.Contains(output, s) {
			t.Errorf("displayNote output should contain %q, got:\n%s", s, output)
		}
	}
}

func TestDisplayNote_NoState(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	note := &storage.Note{
		Number:    5,
		Content:   "Note without state",
		State:     "", // No state
		Timestamp: time.Date(2026, 2, 1, 14, 0, 0, 0, time.UTC),
	}

	displayNote(note)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain number and content
	if !strings.Contains(output, "#5") {
		t.Error("output should contain note number #5")
	}
	if !strings.Contains(output, "Note without state") {
		t.Error("output should contain note content")
	}

	// Should NOT contain brackets for empty state (i.e., no "[]")
	if strings.Contains(output, "[]") {
		t.Error("output should not contain empty state brackets")
	}
}

func TestNoteListCommand_Properties(t *testing.T) {
	if noteListCmd.Use != "list" {
		t.Errorf("Use = %q, want %q", noteListCmd.Use, "list")
	}

	if noteListCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if noteListCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestNoteViewCommand_Properties(t *testing.T) {
	if noteViewCmd.Use != "view [number]" {
		t.Errorf("Use = %q, want %q", noteViewCmd.Use, "view [number]")
	}

	if noteViewCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if noteViewCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestNoteViewCommand_AcceptsMaxOneArg(t *testing.T) {
	// Args should be MaximumNArgs(1)
	// Test that the command is configured correctly
	if noteViewCmd.Args == nil {
		t.Error("Args validator should be set")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// handleNoteMessage behavioral tests
// ──────────────────────────────────────────────────────────────────────────────

func TestHandleNoteMessage_PendingQuestion(t *testing.T) {
	mock := helper_test.NewMockConductor()

	nc := noteContext{hasPendingQuestion: true, isWaitingState: false}
	err := handleNoteMessage(context.Background(), mock, "my answer", nc)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should call AnswerQuestion, not AddNote
	if len(mock.AnswerQuestionCalls) != 1 {
		t.Errorf("AnswerQuestion called %d times, want 1", len(mock.AnswerQuestionCalls))
	}
	if mock.AnswerQuestionCalls[0] != "my answer" {
		t.Errorf("AnswerQuestion arg = %q, want %q", mock.AnswerQuestionCalls[0], "my answer")
	}
	if len(mock.AddNoteCalls) != 0 {
		t.Errorf("AddNote called %d times, want 0 (should use AnswerQuestion)", len(mock.AddNoteCalls))
	}
}

func TestHandleNoteMessage_WaitingState(t *testing.T) {
	mock := helper_test.NewMockConductor()

	nc := noteContext{hasPendingQuestion: false, isWaitingState: true}
	err := handleNoteMessage(context.Background(), mock, "my note", nc)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should call AddNote AND ResetState
	if len(mock.AddNoteCalls) != 1 {
		t.Errorf("AddNote called %d times, want 1", len(mock.AddNoteCalls))
	}
	if mock.AddNoteCalls[0] != "my note" {
		t.Errorf("AddNote arg = %q, want %q", mock.AddNoteCalls[0], "my note")
	}
	if mock.ResetStateCalls != 1 {
		t.Errorf("ResetState called %d times, want 1", mock.ResetStateCalls)
	}
}

func TestHandleNoteMessage_RegularNote(t *testing.T) {
	mock := helper_test.NewMockConductor()

	nc := noteContext{hasPendingQuestion: false, isWaitingState: false}
	err := handleNoteMessage(context.Background(), mock, "regular note", nc)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should call AddNote only
	if len(mock.AddNoteCalls) != 1 {
		t.Errorf("AddNote called %d times, want 1", len(mock.AddNoteCalls))
	}
	if mock.AddNoteCalls[0] != "regular note" {
		t.Errorf("AddNote arg = %q, want %q", mock.AddNoteCalls[0], "regular note")
	}
	// Should NOT call ResetState
	if mock.ResetStateCalls != 0 {
		t.Errorf("ResetState called %d times, want 0", mock.ResetStateCalls)
	}
}

func TestHandleNoteMessage_PropagatesAnswerError(t *testing.T) {
	answerErr := errors.New("answer failed")
	mock := helper_test.NewMockConductor().WithAnswerError(answerErr)

	nc := noteContext{hasPendingQuestion: true, isWaitingState: false}
	err := handleNoteMessage(context.Background(), mock, "my answer", nc)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, answerErr) {
		t.Errorf("error = %v, want wrapped %v", err, answerErr)
	}
}

func TestHandleNoteMessage_PropagatesResetError(t *testing.T) {
	resetErr := errors.New("reset failed")
	mock := helper_test.NewMockConductor().WithResetStateError(resetErr)

	nc := noteContext{hasPendingQuestion: false, isWaitingState: true}
	err := handleNoteMessage(context.Background(), mock, "my note", nc)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, resetErr) {
		t.Errorf("error = %v, want wrapped %v", err, resetErr)
	}
}

func TestHandleNoteMessage_PropagatesAddNoteError(t *testing.T) {
	noteErr := errors.New("save failed")
	mock := helper_test.NewMockConductor().WithAddNoteError(noteErr)

	nc := noteContext{hasPendingQuestion: false, isWaitingState: false}
	err := handleNoteMessage(context.Background(), mock, "my note", nc)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, noteErr) {
		t.Errorf("error = %v, want wrapped %v", err, noteErr)
	}
}
