package display

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
	toolkitdisplay "github.com/valksor/go-toolkit/display"
)

func TestFormatState(t *testing.T) {
	tests := []struct {
		name  string
		state workflow.State
		want  string
	}{
		{"idle", workflow.StateIdle, "Ready"},
		{"planning", workflow.StatePlanning, "Planning"},
		{"implementing", workflow.StateImplementing, "Implementing"},
		{"reviewing", workflow.StateReviewing, "Reviewing"},
		{"done", workflow.StateDone, "Completed"},
		{"failed", workflow.StateFailed, "Failed"},
		{"waiting", workflow.StateWaiting, "Waiting"},
		{"checkpointing", workflow.StateCheckpointing, "Checkpointing"},
		{"reverting", workflow.StateReverting, "Reverting"},
		{"restoring", workflow.StateRestoring, "Restoring"},
		{"unknown state", workflow.State("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatState(tt.state); got != tt.want {
				t.Errorf("FormatState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatStateString(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  string
	}{
		{"idle", "idle", "Ready"},
		{"planning", "planning", "Planning"},
		{"unknown", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatStateString(tt.state); got != tt.want {
				t.Errorf("FormatStateString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStateDescription(t *testing.T) {
	tests := []struct {
		name  string
		state workflow.State
		want  string
	}{
		{"idle", workflow.StateIdle, "Ready to start"},
		{"planning", workflow.StatePlanning, "AI is creating specifications"},
		{"implementing", workflow.StateImplementing, "AI is generating code"},
		{"reviewing", workflow.StateReviewing, "Code review in progress"},
		{"done", workflow.StateDone, "Task completed successfully"},
		{"failed", workflow.StateFailed, "Task failed with error"},
		{"waiting", workflow.StateWaiting, "Action required: Awaiting your response"},
		{"checkpointing", workflow.StateCheckpointing, "Creating checkpoint"},
		{"reverting", workflow.StateReverting, "Reverting to previous state"},
		{"restoring", workflow.StateRestoring, "Restoring from checkpoint"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetStateDescription(tt.state); got != tt.want {
				t.Errorf("GetStateDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatSpecificationStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{"draft", storage.SpecificationStatusDraft, "Draft"},
		{"ready", storage.SpecificationStatusReady, "Ready"},
		{"implementing", storage.SpecificationStatusImplementing, "Implementing"},
		{"done", storage.SpecificationStatusDone, "Completed"},
		{"unknown", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatSpecificationStatus(tt.status); got != tt.want {
				t.Errorf("FormatSpecificationStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSpecificationStatusIcon(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{"draft", storage.SpecificationStatusDraft, "○"},
		{"ready", storage.SpecificationStatusReady, "◐"},
		{"implementing", storage.SpecificationStatusImplementing, "◑"},
		{"done", storage.SpecificationStatusDone, "●"},
		{"unknown", "unknown", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSpecificationStatusIcon(tt.status); got != tt.want {
				t.Errorf("GetSpecificationStatusIcon() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatSpecificationStatusWithIcon(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{"draft", storage.SpecificationStatusDraft, "○ Draft"},
		{"ready", storage.SpecificationStatusReady, "◐ Ready"},
		{"implementing", storage.SpecificationStatusImplementing, "◑ Implementing"},
		{"done", storage.SpecificationStatusDone, "● Completed"},
		{"unknown", "unknown", "? unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatSpecificationStatusWithIcon(tt.status); got != tt.want {
				t.Errorf("FormatSpecificationStatusWithIcon() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStateDisplayMapCompleteness(t *testing.T) {
	// Verify all workflow states have display names
	states := []workflow.State{
		workflow.StateIdle,
		workflow.StatePlanning,
		workflow.StateImplementing,
		workflow.StateReviewing,
		workflow.StateDone,
		workflow.StateFailed,
		workflow.StateWaiting,
		workflow.StateCheckpointing,
		workflow.StateReverting,
		workflow.StateRestoring,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			if got := FormatState(state); got == "" {
				t.Errorf("StateDisplay missing entry for %v", state)
			}
		})
	}
}

func TestSpecificationStatusMapsCompleteness(t *testing.T) {
	// Verify all specification statuses have display names and icons
	statuses := []string{
		storage.SpecificationStatusDraft,
		storage.SpecificationStatusReady,
		storage.SpecificationStatusImplementing,
		storage.SpecificationStatusDone,
	}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			if got := FormatSpecificationStatus(status); got == "" {
				t.Errorf("SpecificationStatusDisplay missing entry for %v", status)
			}
			if got := GetSpecificationStatusIcon(status); got == "" {
				t.Errorf("SpecificationStatusIcon missing entry for %v", status)
			}
		})
	}
}

func TestFormatStateColored(t *testing.T) {
	// Just verify it doesn't crash and returns something
	state := workflow.StatePlanning
	got := FormatStateColored(state)
	if got == "" {
		t.Error("FormatStateColored() returned empty string")
	}
	// Should contain the state name
	expected := FormatState(state)
	if got != expected {
		// With colors, it should have the prefix at least
		if got == "" {
			t.Errorf("FormatStateColored() = %v, want something containing %v", got, expected)
		}
	}
}

func TestColorState(t *testing.T) {
	tests := []struct {
		name        string
		state       string
		displayName string
		wantMuted   bool
		wantInfo    bool
		wantSuccess bool
		wantError   bool
		wantWarning bool
	}{
		{"idle", "idle", "Ready", true, false, false, false, false},
		{"planning", "planning", "Planning", false, true, false, false, false},
		{"implementing", "implementing", "Implementing", false, true, false, false, false},
		{"done", "done", "Completed", false, false, true, false, false},
		{"failed", "failed", "Failed", false, false, false, true, false},
		{"waiting", "waiting", "Waiting", false, false, false, false, true},
	}

	toolkitdisplay.SetColorsEnabled(true)
	defer toolkitdisplay.SetColorsEnabled(false)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ColorState(tt.state, tt.displayName)
			if got == "" {
				t.Errorf("ColorState() returned empty string")
			}
		})
	}
}

func TestColorSpecStatus(t *testing.T) {
	tests := []struct {
		name        string
		status      string
		displayName string
	}{
		{"draft", "draft", "Draft"},
		{"ready", "ready", "Ready"},
		{"implementing", "implementing", "Implementing"},
		{"done", "done", "Completed"},
	}

	toolkitdisplay.SetColorsEnabled(true)
	defer toolkitdisplay.SetColorsEnabled(false)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ColorSpecStatus(tt.status, tt.displayName)
			if got == "" {
				t.Errorf("ColorSpecStatus() returned empty string")
			}
		})
	}
}

func TestWorktreeIndicator(t *testing.T) {
	tests := []struct {
		name       string
		isWorktree bool
		wantEmpty  bool
	}{
		{"worktree", true, false},
		{"not worktree", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WorktreeIndicator(tt.isWorktree)
			if (got == "") != tt.wantEmpty {
				t.Errorf("WorktreeIndicator() = %v, want empty %v", got, tt.wantEmpty)
			}
		})
	}
}

func TestErrorWithSuggestions(t *testing.T) {
	suggestions := []Suggestion{
		{Command: "test", Description: "Test command"},
	}

	got := ErrorWithSuggestions("test error", suggestions)
	if got == "" {
		t.Error("ErrorWithSuggestions() returned empty string")
	}
	if !contains(got, "test error") {
		t.Errorf("ErrorWithSuggestions() = %v, want containing 'test error'", got)
	}
	if !contains(got, "test") {
		t.Errorf("ErrorWithSuggestions() = %v, want containing 'test' command", got)
	}
}

func TestErrorWithContext(t *testing.T) {
	err := errors.New("test error")
	suggestions := []Suggestion{
		{Command: "fix", Description: "Fix it"},
	}

	got := ErrorWithContext(err, "context", suggestions)
	if got == "" {
		t.Error("ErrorWithContext() returned empty string")
	}
	if !contains(got, "context") {
		t.Errorf("ErrorWithContext() = %v, want containing 'context'", got)
	}
	if !contains(got, "test error") {
		t.Errorf("ErrorWithContext() = %v, want containing 'test error'", got)
	}
}

func TestValidationError(t *testing.T) {
	suggestions := []Suggestion{
		{Command: "fix", Description: "Fix field"},
	}

	got := ValidationError("field", "is required", suggestions)
	if got == "" {
		t.Error("ValidationError() returned empty string")
	}
	if !contains(got, "field") {
		t.Errorf("ValidationError() = %v, want containing 'field'", got)
	}
	if !contains(got, "is required") {
		t.Errorf("ValidationError() = %v, want containing 'is required'", got)
	}
}

func TestProviderError(t *testing.T) {
	err := errors.New("provider failed")
	got := ProviderError("test", err, nil)

	if got == "" {
		t.Error("ProviderError() returned empty string")
	}
	if !contains(got, "test provider") {
		t.Errorf("ProviderError() = %v, want containing 'test provider'", got)
	}
}

func TestNoActiveTaskError(t *testing.T) {
	got := NoActiveTaskError()
	if got == "" {
		t.Error("NoActiveTaskError() returned empty string")
	}
	if !contains(got, "No active task") {
		t.Errorf("NoActiveTaskError() = %v, want containing 'No active task'", got)
	}
}

func TestTaskFailedError(t *testing.T) {
	err := errors.New("step failed")
	got := TaskFailedError("implementing", err)

	if got == "" {
		t.Error("TaskFailedError() returned empty string")
	}
	if !contains(got, "implementing") {
		t.Errorf("TaskFailedError() = %v, want containing 'implementing'", got)
	}
}

func TestConfigError(t *testing.T) {
	err := errors.New("config invalid")
	got := ConfigError(err, "/path/to/config")

	if got == "" {
		t.Error("ConfigError() returned empty string")
	}
	if !contains(got, "/path/to/config") {
		t.Errorf("ConfigError() = %v, want containing config path", got)
	}
}

func TestAgentError(t *testing.T) {
	err := errors.New("agent failed")
	got := AgentError("test-agent", err)

	if got == "" {
		t.Error("AgentError() returned empty string")
	}
	if !contains(got, "test-agent") {
		t.Errorf("AgentError() = %v, want containing 'test-agent'", got)
	}
}

func TestGitError(t *testing.T) {
	err := errors.New("git failed")
	got := GitError("branch", err)

	if got == "" {
		t.Error("GitError() returned empty string")
	}
	if !contains(got, "Git branch") {
		t.Errorf("GitError() = %v, want containing 'Git branch'", got)
	}
}

func TestConductorError(t *testing.T) {
	err := errors.New("init failed")
	got := ConductorError("initialize", err)

	if got == "" {
		t.Error("ConductorError() returned empty string")
	}
	if !contains(got, "initialize workspace") {
		t.Errorf("ConductorError() = %v, want containing 'initialize workspace'", got)
	}
}

func TestWorkspaceError(t *testing.T) {
	err := errors.New("open failed")
	got := WorkspaceError("open", err)

	if got == "" {
		t.Error("WorkspaceError() returned empty string")
	}
	if !contains(got, "Workspace open") {
		t.Errorf("WorkspaceError() = %v, want containing 'Workspace open'", got)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestFormatTaskInfo(t *testing.T) {
	info := TaskInfo{
		TaskID:      "task-123",
		Title:       "Test Task",
		ExternalKey: "KEY-123",
		State:       "planning",
		Source:      "github",
		Branch:      "feature/test",
		Worktree:    "/path/to/worktree",
		Started:     time.Now().Format("2006-01-02 15:04:05"),
	}

	opts := DefaultTaskInfoOptions()
	got := FormatTaskInfo("Task", info, opts)

	if got == "" {
		t.Error("FormatTaskInfo() returned empty string")
	}
	if !contains(got, "task-123") {
		t.Errorf("FormatTaskInfo() = %v, want containing 'task-123'", got)
	}
	if !contains(got, "Test Task") {
		t.Errorf("FormatTaskInfo() = %v, want containing 'Test Task'", got)
	}
}

func TestFormatNextSteps(t *testing.T) {
	steps := []NextStep{
		{Command: "step1", Description: "First step"},
		{Command: "step2", Description: "Second step"},
	}

	got := FormatNextSteps(steps)

	if got == "" {
		t.Error("FormatNextSteps() returned empty string")
	}
	if !contains(got, "step1") {
		t.Errorf("FormatNextSteps() = %v, want containing 'step1'", got)
	}
	if !contains(got, "First step") {
		t.Errorf("FormatNextSteps() = %v, want containing 'First step'", got)
	}
}

func TestFormatConfirmation(t *testing.T) {
	details := []string{"Detail 1", "Detail 2"}
	warning := "This is a warning"

	got := FormatConfirmation("Test action", details, warning)

	if got == "" {
		t.Error("FormatConfirmation() returned empty string")
	}
	if !contains(got, "Test action") {
		t.Errorf("FormatConfirmation() = %v, want containing 'Test action'", got)
	}
	if !contains(got, "This is a warning") {
		t.Errorf("FormatConfirmation() = %v, want containing warning", got)
	}
}
