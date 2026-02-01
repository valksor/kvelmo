//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

// Note: TestStatusCommand_Aliases is in common_test.go

func TestStatusCommand_Properties(t *testing.T) {
	if statusCmd.Use != "status" {
		t.Errorf("Use = %q, want %q", statusCmd.Use, "status")
	}

	if statusCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if statusCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if statusCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestStatusCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "all flag",
			flagName:     "all",
			shorthand:    "a",
			defaultValue: "false",
		},
		{
			name:         "json flag",
			flagName:     "json",
			shorthand:    "",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := statusCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			if tt.shorthand != "" {
				shorthand := statusCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestStatusCommand_ShortDescription(t *testing.T) {
	expected := "Show full task details"
	if statusCmd.Short != expected {
		t.Errorf("Short = %q, want %q", statusCmd.Short, expected)
	}
}

func TestStatusCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"Comprehensive view",
		"Task metadata",
		"Specifications",
		"Git checkpoints",
		"Session history",
	}

	for _, substr := range contains {
		if !containsString(statusCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestStatusCommand_WhenToUseSection(t *testing.T) {
	// Should document when to use status vs. a guide vs continue
	expected := []string{
		"RELATED COMMANDS",
		"guide",
		"status",
		"continue",
	}

	for _, s := range expected {
		if !containsString(statusCmd.Long, s) {
			t.Errorf("Long description does not contain %q", s)
		}
	}
}

func TestStatusCommand_OutputFormats(t *testing.T) {
	// Should document output formats
	if !containsString(statusCmd.Long, "OUTPUT FORMATS") {
		t.Error("Long description does not document OUTPUT FORMATS section")
	}

	if !containsString(statusCmd.Long, "--json") {
		t.Error("Long description does not mention --json flag")
	}
}

func TestStatusCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr status",
		"--all",
		"--json",
	}

	for _, example := range examples {
		if !containsString(statusCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestStatusCommand_SeeAlsoSection(t *testing.T) {
	// Should reference related commands (in CHOOSING THE RIGHT COMMAND section)
	related := []string{
		"guide",
		"continue",
	}

	for _, cmd := range related {
		if !containsString(statusCmd.Long, cmd) {
			t.Errorf("Long description does not reference %q", cmd)
		}
	}
}

func TestStatusCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "status" {
			found = true

			break
		}
	}
	if !found {
		t.Error("status command not registered in root command")
	}
}

func TestStatusCommand_NoAliases(t *testing.T) {
	// Aliases removed in favor of prefix matching
	if len(statusCmd.Aliases) > 0 {
		t.Errorf("status command should have no aliases, got %v", statusCmd.Aliases)
	}
}

func TestStatusCommand_JSONFlagNoShorthand(t *testing.T) {
	// JSON flag should not have shorthand to avoid conflicts
	flag := statusCmd.Flags().Lookup("json")
	if flag == nil {
		t.Fatal("json flag not found")

		return
	}
	if flag.Shorthand != "" {
		t.Errorf("json flag has shorthand %q, expected none", flag.Shorthand)
	}
}

// --- Phase 4: Behavioral tests ---

func TestShowActiveTask_NoActiveTask(t *testing.T) {
	tc := NewTestContext(t)

	oldJSON := statusJSON
	defer func() { statusJSON = oldJSON }()

	statusJSON = false

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showActiveTask(context.Background(), tc.Workspace, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showActiveTask returned error: %v", err)
	}

	if !strings.Contains(output, "No active task") {
		t.Errorf("expected output to contain %q, got:\n%s", "No active task", output)
	}
}

func TestShowActiveTask_WithTask(t *testing.T) {
	tc := NewTestContext(t)

	tc.CreateActiveTask("test-task-1", "file:test.md")
	tc.CreateTaskWork("test-task-1", "My Test Task")

	oldJSON := statusJSON
	oldDiagram := statusDiagram

	defer func() {
		statusJSON = oldJSON
		statusDiagram = oldDiagram
	}()

	statusJSON = false
	statusDiagram = false

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showActiveTask(context.Background(), tc.Workspace, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showActiveTask returned error: %v", err)
	}

	expected := []string{
		"Active Task:",
		"test-task-1",
		"My Test Task",
		"No specifications yet",
		"mehr plan",
		"mehr note",
		"mehr finish",
	}

	for _, s := range expected {
		if !strings.Contains(output, s) {
			t.Errorf("expected output to contain %q, got:\n%s", s, output)
		}
	}
}

func TestShowActiveTask_JSONOutput(t *testing.T) {
	tc := NewTestContext(t)

	tc.CreateActiveTask("test-task-1", "file:test.md")
	tc.CreateTaskWork("test-task-1", "My Test Task")

	oldJSON := statusJSON
	defer func() { statusJSON = oldJSON }()

	statusJSON = true

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showActiveTask(context.Background(), tc.Workspace, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showActiveTask returned error: %v", err)
	}

	expected := []string{
		`"task_id"`,
		`"test-task-1"`,
		`"My Test Task"`,
	}

	for _, s := range expected {
		if !strings.Contains(output, s) {
			t.Errorf("expected JSON output to contain %q, got:\n%s", s, output)
		}
	}
}

func TestShowAllTasks_Empty(t *testing.T) {
	tc := NewTestContext(t)

	oldJSON := statusJSON
	defer func() { statusJSON = oldJSON }()

	statusJSON = false

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showAllTasks(tc.Workspace)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showAllTasks returned error: %v", err)
	}

	if !strings.Contains(output, "No tasks found") {
		t.Errorf("expected output to contain %q, got:\n%s", "No tasks found", output)
	}
}

func TestShowAllTasks_WithTasks(t *testing.T) {
	tc := NewTestContext(t)

	tc.CreateActiveTask("task-1", "file:task1.md")
	tc.CreateTaskWork("task-1", "First Task")
	tc.CreateTaskWork("task-2", "Second Task")

	oldJSON := statusJSON
	defer func() { statusJSON = oldJSON }()

	statusJSON = false

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showAllTasks(tc.Workspace)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showAllTasks returned error: %v", err)
	}

	expected := []string{
		"TASK ID",
		"First Task",
		"task-2",
		"Second Task",
	}

	for _, s := range expected {
		if !strings.Contains(output, s) {
			t.Errorf("expected output to contain %q, got:\n%s", s, output)
		}
	}
}

func TestPrintSpecLegend(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printSpecLegend()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	expected := []string{
		"Draft",
		"Ready",
		"Implementing",
		"Completed",
	}

	for _, s := range expected {
		if !strings.Contains(output, s) {
			t.Errorf("expected legend output to contain %q, got:\n%s", s, output)
		}
	}
}

// --- Phase 5: Deep behavioral tests ---

func TestShowAllTasks_JSONOutput(t *testing.T) {
	tc := NewTestContext(t)

	tc.CreateActiveTask("task-1", "file:task1.md")
	tc.CreateTaskWork("task-1", "First Task")
	tc.CreateTaskWork("task-2", "Second Task")

	oldJSON := statusJSON
	defer func() { statusJSON = oldJSON }()

	statusJSON = true

	// Capture stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showAllTasks(tc.Workspace)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showAllTasks returned error: %v", err)
	}

	expected := []string{
		`"tasks"`,
		`"task_id"`,
		`"First Task"`,
	}

	for _, s := range expected {
		if !strings.Contains(output, s) {
			t.Errorf("expected JSON output to contain %q, got:\n%s", s, output)
		}
	}
}

func TestHasImplementedSpecifications_NoSpecs(t *testing.T) {
	tc := NewTestContext(t)

	tc.CreateActiveTask("task-1", "file:task1.md")
	tc.CreateTaskWork("task-1", "Task Without Specs")

	result := hasImplementedSpecifications(tc.Workspace, "task-1")
	if result {
		t.Error("hasImplementedSpecifications should return false when task has no specifications")
	}
}

func TestBuildJSONStatusTask(t *testing.T) {
	tc := NewTestContext(t)

	tc.CreateActiveTask("task-1", "file:task1.md")
	tc.CreateTaskWork("task-1", "JSON Status Task")

	active, err := tc.Workspace.LoadActiveTask()
	if err != nil {
		t.Fatalf("LoadActiveTask: %v", err)
	}

	work, err := tc.Workspace.LoadWork(active.ID)
	if err != nil {
		t.Fatalf("LoadWork: %v", err)
	}

	result := buildJSONStatusTask(context.Background(), tc.Workspace, nil, active, work, "")

	if result.TaskID != "task-1" {
		t.Errorf("TaskID = %q, want %q", result.TaskID, "task-1")
	}
	if result.Title != "JSON Status Task" {
		t.Errorf("Title = %q, want %q", result.Title, "JSON Status Task")
	}
	if result.State != active.State {
		t.Errorf("State = %q, want %q", result.State, active.State)
	}
	if !result.IsActive {
		t.Error("IsActive should be true")
	}
}
