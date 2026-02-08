//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/agent"
	routercommands "github.com/valksor/go-mehrhof/internal/conductor/commands"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
)

// captureStdStreams captures both stdout and stderr during function execution.
func captureStdStreams(fn func()) (string, string) {
	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// Capture stderr
	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	fn()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	_, _ = bufOut.ReadFrom(rOut)
	_, _ = bufErr.ReadFrom(rErr)

	return bufOut.String(), bufErr.String()
}

// ──────────────────────────────────────────────────────────────────────────────
// handleAgentEvent Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestInteractiveSession_HandleAgentEvent_Text(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	stdout, _ := captureStdStreams(func() {
		err := s.handleAgentEvent(agent.Event{
			Type: agent.EventText,
			Text: "Hello, world!",
		})
		if err != nil {
			t.Errorf("handleAgentEvent returned error: %v", err)
		}
	})

	if !strings.Contains(stdout, "Hello, world!") {
		t.Errorf("stdout = %q, want to contain 'Hello, world!'", stdout)
	}

	if !strings.Contains(s.transcript.String(), "Hello, world!") {
		t.Errorf("transcript = %q, want to contain 'Hello, world!'", s.transcript.String())
	}
}

func TestInteractiveSession_HandleAgentEvent_ToolUse(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	_, stderr := captureStdStreams(func() {
		err := s.handleAgentEvent(agent.Event{
			Type: agent.EventToolUse,
			ToolCall: &agent.ToolCall{
				Name: "read_file",
			},
		})
		if err != nil {
			t.Errorf("handleAgentEvent returned error: %v", err)
		}
	})

	if !strings.Contains(stderr, "read_file") {
		t.Errorf("stderr = %q, want to contain 'read_file'", stderr)
	}
}

func TestInteractiveSession_HandleAgentEvent_IgnoredTypes(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	ignoredTypes := []agent.EventType{
		agent.EventToolResult,
		agent.EventFile,
		agent.EventError,
		agent.EventUsage,
		agent.EventComplete,
	}

	for _, eventType := range ignoredTypes {
		t.Run(string(eventType), func(t *testing.T) {
			err := s.handleAgentEvent(agent.Event{
				Type: eventType,
			})
			if err != nil {
				t.Errorf("handleAgentEvent returned error for %s: %v", eventType, err)
			}
		})
	}
}

func TestInteractiveSession_HandleAgentEvent_ToolUseNilToolCall(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	_, stderr := captureStdStreams(func() {
		err := s.handleAgentEvent(agent.Event{
			Type:     agent.EventToolUse,
			ToolCall: nil, // nil ToolCall should be handled gracefully
		})
		if err != nil {
			t.Errorf("handleAgentEvent returned error: %v", err)
		}
	})

	// Should not print tool indicator for nil ToolCall
	if strings.Contains(stderr, "→") {
		t.Errorf("stderr should not contain tool indicator for nil ToolCall: %q", stderr)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// renderResult Tests for Missing Result Types
// ──────────────────────────────────────────────────────────────────────────────

func TestRenderResult_Waiting(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	stdout, _ := captureStdStreams(func() {
		s.renderResult(&routercommands.Result{
			Type:    routercommands.ResultWaiting,
			Message: "Choose an option",
			Data: routercommands.WaitingData{
				Question: "Which approach?",
				Options: []routercommands.QuestionOption{
					{Label: "Option A", Description: "First option"},
					{Label: "Option B", Description: "Second option"},
				},
			},
		})
	})

	if !strings.Contains(stdout, "Which approach?") {
		t.Errorf("stdout should contain question: %q", stdout)
	}
}

func TestRenderResult_WaitingWithMessage(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	stdout, _ := captureStdStreams(func() {
		s.renderResult(&routercommands.Result{
			Type:    routercommands.ResultWaiting,
			Message: "Please provide input",
			Data:    nil, // No WaitingData, should fall back to message
		})
	})

	if !strings.Contains(stdout, "Please provide input") {
		t.Errorf("stdout should contain message: %q", stdout)
	}
}

func TestRenderResult_Paused(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	stdout, _ := captureStdStreams(func() {
		s.renderResult(&routercommands.Result{
			Type:    routercommands.ResultPaused,
			Message: "Operation paused by user",
		})
	})

	if !strings.Contains(stdout, "Operation paused by user") {
		t.Errorf("stdout should contain message: %q", stdout)
	}
}

func TestRenderResult_Stopped(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	_, stderr := captureStdStreams(func() {
		s.renderResult(&routercommands.Result{
			Type:    routercommands.ResultStopped,
			Message: "Agent stopped unexpectedly",
		})
	})

	if !strings.Contains(stderr, "Agent stopped unexpectedly") {
		t.Errorf("stderr should contain message: %q", stderr)
	}
}

func TestRenderResult_Conflict(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	_, stderr := captureStdStreams(func() {
		s.renderResult(&routercommands.Result{
			Type:    routercommands.ResultConflict,
			Message: "Merge conflict detected",
		})
	})

	if !strings.Contains(stderr, "Merge conflict detected") {
		t.Errorf("stderr should contain message: %q", stderr)
	}
}

func TestRenderResult_Exit(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	stdout, stderr := captureStdStreams(func() {
		// ResultExit should not produce any output
		s.renderResult(&routercommands.Result{
			Type:    routercommands.ResultExit,
			Message: "Goodbye",
		})
	})

	// Exit should be silent
	if stdout != "" || stderr != "" {
		t.Errorf("ResultExit should produce no output, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestRenderResult_ListWithSpecifications(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	stdout, _ := captureStdStreams(func() {
		s.renderResult(&routercommands.Result{
			Type:    routercommands.ResultList,
			Message: "Specifications",
			Data: []routercommands.SpecificationItem{
				{Number: 1, Title: "First spec", Status: "open"},
				{Number: 2, Title: "Second spec", Status: "done"},
			},
		})
	})

	if !strings.Contains(stdout, "spec-1") {
		t.Errorf("stdout should contain 'spec-1': %q", stdout)
	}
}

func TestRenderResult_ListWithCurrentTask(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	stdout, _ := captureStdStreams(func() {
		s.renderResult(&routercommands.Result{
			Type:    routercommands.ResultList,
			TaskID:  "task-2",
			Message: "Tasks",
			Data: []routercommands.TaskListItem{
				{ID: "task-1", Title: "First task", State: "idle"},
				{ID: "task-2", Title: "Second task", State: "planning"},
			},
		})
	})

	// Current task should have * prefix (with possible ANSI codes in between)
	if !strings.Contains(stdout, "*") || !strings.Contains(stdout, "task-2") {
		t.Errorf("stdout should mark current task with '*' and contain 'task-2': %q", stdout)
	}
}

func TestRenderResult_HelpWithAliases(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	stdout, _ := captureStdStreams(func() {
		s.renderResult(&routercommands.Result{
			Type:    routercommands.ResultHelp,
			Message: "help",
			Data: []routercommands.CommandInfo{
				{Name: "status", Category: "info", Description: "Show status", Aliases: []string{"st", "s"}},
			},
		})
	})

	if !strings.Contains(stdout, "st") || !strings.Contains(stdout, "s") {
		t.Errorf("stdout should contain aliases: %q", stdout)
	}
}

func TestRenderResult_StatusWithBranch(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	stdout, _ := captureStdStreams(func() {
		s.renderResult(&routercommands.Result{
			Type:    routercommands.ResultStatus,
			Message: "status",
			Data: routercommands.StatusData{
				TaskID: "task-1",
				Title:  "Test Task",
				State:  "planning",
				Branch: "feature/test-123--test-task",
			},
		})
	})

	if !strings.Contains(stdout, "feature/test-123--test-task") {
		t.Errorf("stdout should contain branch name: %q", stdout)
	}
}

func TestRenderResult_BudgetWarned(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	stdout, _ := captureStdStreams(func() {
		s.renderResult(&routercommands.Result{
			Type:    routercommands.ResultBudget,
			Message: "budget",
			Data: routercommands.BudgetData{
				Type:       "cost",
				Used:       "$8.50",
				Max:        "$10.00",
				Percentage: 85,
				Warned:     true,
			},
		})
	})

	// Should contain warning indicator
	if !strings.Contains(stdout, "Warning") && !strings.Contains(stdout, "warning") {
		t.Errorf("stdout should contain warning: %q", stdout)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// handleEvent Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestInteractiveSession_HandleEvent_StateChanged(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	stdout, _ := captureStdStreams(func() {
		s.handleEvent(eventbus.Event{
			Type: events.TypeStateChanged,
			Data: map[string]any{
				"from": "idle",
				"to":   "planning",
			},
		})
	})

	if !strings.Contains(stdout, "planning") {
		t.Errorf("stdout should contain new state: %q", stdout)
	}

	if s.state != workflow.StatePlanning {
		t.Errorf("session state = %q, want planning", s.state)
	}
}

func TestInteractiveSession_HandleEvent_Progress(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	_, stderr := captureStdStreams(func() {
		s.handleEvent(eventbus.Event{
			Type: events.TypeProgress,
			Data: map[string]any{
				"message": "Processing file 3/10",
			},
		})
	})

	if !strings.Contains(stderr, "Processing file 3/10") {
		t.Errorf("stderr should contain progress message: %q", stderr)
	}
}

func TestInteractiveSession_HandleEvent_FileChanged(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	_, stderr := captureStdStreams(func() {
		s.handleEvent(eventbus.Event{
			Type: events.TypeFileChanged,
			Data: map[string]any{
				"path":      "internal/server/handlers.go",
				"operation": "modified",
			},
		})
	})

	if !strings.Contains(stderr, "internal/server/handlers.go") {
		t.Errorf("stderr should contain file path: %q", stderr)
	}

	if !strings.Contains(stderr, "modified") {
		t.Errorf("stderr should contain operation: %q", stderr)
	}
}

func TestInteractiveSession_HandleEvent_Error(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	_, stderr := captureStdStreams(func() {
		s.handleEvent(eventbus.Event{
			Type: events.TypeError,
			Data: map[string]any{
				"error": "connection timeout",
			},
		})
	})

	if !strings.Contains(stderr, "connection timeout") {
		t.Errorf("stderr should contain error message: %q", stderr)
	}
}

func TestInteractiveSession_HandleEvent_UnknownType(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))

	stdout, stderr := captureStdStreams(func() {
		s.handleEvent(eventbus.Event{
			Type: "unknown_event_type",
			Data: map[string]any{},
		})
	})

	// Unknown events should not produce output
	if stdout != "" || stderr != "" {
		t.Errorf("unknown event should produce no output, got stdout=%q stderr=%q", stdout, stderr)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// capitalizeFirst Tests (from interactive_features.go)
// ──────────────────────────────────────────────────────────────────────────────

func TestCapitalizeFirst_SingleChar(t *testing.T) {
	if got := capitalizeFirst("a"); got != "A" {
		t.Errorf("capitalizeFirst('a') = %q, want 'A'", got)
	}
}

func TestCapitalizeFirst_AlreadyCapitalized(t *testing.T) {
	if got := capitalizeFirst("Hello"); got != "Hello" {
		t.Errorf("capitalizeFirst('Hello') = %q, want 'Hello'", got)
	}
}

func TestCapitalizeFirst_AllCaps(t *testing.T) {
	if got := capitalizeFirst("HELLO"); got != "HELLO" {
		t.Errorf("capitalizeFirst('HELLO') = %q, want 'HELLO'", got)
	}
}

func TestCapitalizeFirst_Number(t *testing.T) {
	if got := capitalizeFirst("123abc"); got != "123abc" {
		t.Errorf("capitalizeFirst('123abc') = %q, want '123abc'", got)
	}
}
