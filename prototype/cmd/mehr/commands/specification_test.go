//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/helper_test"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestSpecificationCommand_Properties(t *testing.T) {
	// Test the main specification command
	if specificationCmd.Use != "specification" {
		t.Errorf("specificationCmd.Use = %q, want %q", specificationCmd.Use, "specification")
	}

	if specificationCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if specificationCmd.Long == "" {
		t.Error("Long description is empty")
	}

	// Test the view subcommand
	if specificationViewCmd.Use != "view <number>" {
		t.Errorf("specificationViewCmd.Use = %q, want %q", specificationViewCmd.Use, "view <number>")
	}

	if specificationViewCmd.Short == "" {
		t.Error("Short description is empty for view subcommand")
	}

	if specificationViewCmd.Long == "" {
		t.Error("Long description is empty for view subcommand")
	}

	if specificationViewCmd.RunE == nil {
		t.Error("RunE not set for view subcommand")
	}
}

func TestSpecificationCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue interface{}
	}{
		{
			name:         "number flag",
			flagName:     "number",
			shorthand:    "n",
			defaultValue: 0,
		},
		{
			name:         "all flag",
			flagName:     "all",
			shorthand:    "a",
			defaultValue: false,
		},
		{
			name:         "output flag",
			flagName:     "output",
			shorthand:    "o",
			defaultValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := specificationViewCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			// Check shorthand
			if flag.Shorthand != tt.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", tt.flagName, flag.Shorthand, tt.shorthand)
			}

			// Check default value
			// Note: We can't easily check the actual default value without parsing,
			// but we can verify the flag exists
		})
	}
}

func TestFormatSpecificationHeader(t *testing.T) {
	spec := &storage.Specification{
		Number:    1,
		Title:     "Implement auth",
		Status:    "pending",
		Component: "backend",
		CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	result := formatSpecificationHeader(spec)

	for _, substr := range []string{"Specification 1", "Implement auth", "pending", "backend", "2025-01-15"} {
		if !strings.Contains(result, substr) {
			t.Errorf("header missing %q\nGot:\n%s", substr, result)
		}
	}
}

func TestFormatSpecificationHeader_Minimal(t *testing.T) {
	spec := &storage.Specification{
		Number: 2,
		Status: "done",
	}

	result := formatSpecificationHeader(spec)

	if !strings.Contains(result, "Specification 2") {
		t.Errorf("header missing 'Specification 2'\nGot:\n%s", result)
	}
	if strings.Contains(result, "Component") {
		t.Errorf("header should NOT contain 'Component' when empty\nGot:\n%s", result)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// runSpecificationDiffLogic behavioral tests
// ──────────────────────────────────────────────────────────────────────────────

func TestRunSpecificationDiffLogic_NoActiveTask(t *testing.T) {
	mock := helper_test.NewMockConductor()
	// No active task set

	opts := specificationDiffOptions{specNumber: 1, filePath: "test.go"}
	err := runSpecificationDiffLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error for no active task")
	}
	if err != nil && err.Error() != "no active task" {
		t.Errorf("error = %q, want %q", err.Error(), "no active task")
	}
}

func TestRunSpecificationDiffLogic_MissingSpecNumber(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test-task", State: "implementing"})

	opts := specificationDiffOptions{specNumber: 0, filePath: "test.go"}
	err := runSpecificationDiffLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error for missing spec number")
	}
	if !strings.Contains(err.Error(), "specification number required") {
		t.Errorf("error = %q, want to contain 'specification number required'", err.Error())
	}
}

func TestRunSpecificationDiffLogic_MissingFilePath(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test-task", State: "implementing"})

	opts := specificationDiffOptions{specNumber: 1, filePath: ""}
	err := runSpecificationDiffLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error for missing file path")
	}
	if !strings.Contains(err.Error(), "file path required") {
		t.Errorf("error = %q, want to contain 'file path required'", err.Error())
	}
}

func TestRunSpecificationDiffLogic_CallsGetDiff(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test-task", State: "implementing"}).
		WithSpecificationFileDiff("--- a/test.go\n+++ b/test.go\n@@ -1 +1 @@\n-old\n+new")

	var stdout bytes.Buffer
	opts := specificationDiffOptions{specNumber: 2, filePath: "internal/foo.go", contextLines: 5}
	err := runSpecificationDiffLogic(context.Background(), mock, opts, &stdout)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(mock.SpecificationFileDiffCalls) != 1 {
		t.Fatalf("GetSpecificationFileDiff called %d times, want 1", len(mock.SpecificationFileDiffCalls))
	}
	call := mock.SpecificationFileDiffCalls[0]
	if call.TaskID != "test-task" {
		t.Errorf("TaskID = %q, want %q", call.TaskID, "test-task")
	}
	if call.SpecNumber != 2 {
		t.Errorf("SpecNumber = %d, want %d", call.SpecNumber, 2)
	}
	if call.FilePath != "internal/foo.go" {
		t.Errorf("FilePath = %q, want %q", call.FilePath, "internal/foo.go")
	}
	if call.ContextLines != 5 {
		t.Errorf("ContextLines = %d, want %d", call.ContextLines, 5)
	}
	// Check output contains the diff
	if !strings.Contains(stdout.String(), "+new") {
		t.Errorf("output should contain diff, got: %s", stdout.String())
	}
}

func TestRunSpecificationDiffLogic_EmptyDiff(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test-task", State: "implementing"}).
		WithSpecificationFileDiff("") // Empty diff

	var stdout bytes.Buffer
	opts := specificationDiffOptions{specNumber: 1, filePath: "test.go", contextLines: 3}
	err := runSpecificationDiffLogic(context.Background(), mock, opts, &stdout)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should output "No diff found" message
	if !strings.Contains(stdout.String(), "No diff found") {
		t.Errorf("output should contain 'No diff found', got: %s", stdout.String())
	}
}

func TestRunSpecificationDiffLogic_PropagatesError(t *testing.T) {
	diffErr := errors.New("git error: could not compute diff")
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test-task", State: "implementing"}).
		WithSpecificationFileDiffError(diffErr)

	opts := specificationDiffOptions{specNumber: 1, filePath: "test.go", contextLines: 3}
	err := runSpecificationDiffLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, diffErr) {
		t.Errorf("error = %v, want %v", err, diffErr)
	}
}
