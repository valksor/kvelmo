package events

import (
	"errors"
	"testing"
	"time"

	"github.com/valksor/go-toolkit/eventbus"
)

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name  string
		event eventbus.Type
		want  string
	}{
		{"TypeStateChanged", TypeStateChanged, "state_changed"},
		{"TypeProgress", TypeProgress, "progress"},
		{"TypeError", TypeError, "error"},
		{"TypeFileChanged", TypeFileChanged, "file_changed"},
		{"TypeAgentMessage", TypeAgentMessage, "agent_message"},
		{"TypeCheckpoint", TypeCheckpoint, "checkpoint"},
		{"TypeBlueprintReady", TypeBlueprintReady, "blueprint_ready"},
		{"TypeBranchCreated", TypeBranchCreated, "branch_created"},
		{"TypePlanCompleted", TypePlanCompleted, "plan_completed"},
		{"TypeImplementDone", TypeImplementDone, "implement_done"},
		{"TypePRCreated", TypePRCreated, "pr_created"},
		{"TypeBrowserAction", TypeBrowserAction, "browser_action"},
		{"TypeBrowserTabOpened", TypeBrowserTabOpened, "browser_tab_opened"},
		{"TypeBrowserScreenshot", TypeBrowserScreenshot, "browser_screenshot"},
		{"TypeSandboxStatusChanged", TypeSandboxStatusChanged, "sandbox_status_changed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.event) != tt.want {
				t.Errorf("Event type = %q, want %q", tt.event, tt.want)
			}
		})
	}
}

func TestSandboxStatusChangedEvent(t *testing.T) {
	e := SandboxStatusChangedEvent{
		Enabled:  true,
		Active:   false,
		Platform: "linux",
	}

	result := e.ToEvent()

	if result.Type != TypeSandboxStatusChanged {
		t.Errorf("Type = %q, want %q", result.Type, TypeSandboxStatusChanged)
	}

	if result.Data["enabled"] != true {
		t.Error("enabled should be true")
	}

	if result.Data["active"] != false {
		t.Error("active should be false")
	}

	if result.Data["platform"] != "linux" {
		t.Error("platform should be linux")
	}
}

func TestStateChangedEvent(t *testing.T) {
	t.Run("with timestamp", func(t *testing.T) {
		now := time.Now()
		e := StateChangedEvent{
			From:      "idle",
			To:        "planning",
			Event:     "start",
			TaskID:    "task-123",
			Timestamp: now,
		}

		result := e.ToEvent()

		if result.Type != TypeStateChanged {
			t.Errorf("Type = %q, want %q", result.Type, TypeStateChanged)
		}

		if result.Data["from"] != "idle" {
			t.Error("from should be idle")
		}

		if result.Data["to"] != "planning" {
			t.Error("to should be planning")
		}

		if result.Data["event"] != "start" {
			t.Error("event should be start")
		}

		if result.Data["task_id"] != "task-123" {
			t.Error("task_id should be task-123")
		}

		if !result.Timestamp.Equal(now) {
			t.Error("timestamp should be preserved")
		}
	})

	t.Run("auto timestamp", func(t *testing.T) {
		before := time.Now()
		e := StateChangedEvent{
			From:   "idle",
			To:     "planning",
			TaskID: "task-456",
		}
		result := e.ToEvent()
		after := time.Now()

		if result.Timestamp.Before(before) || result.Timestamp.After(after) {
			t.Error("timestamp should be set to now")
		}
	})
}

func TestProgressEvent(t *testing.T) {
	t.Run("with timestamp", func(t *testing.T) {
		now := time.Now()
		e := ProgressEvent{
			Timestamp: now,
			TaskID:    "task-123",
			Phase:     "implementation",
			Message:   "Writing code...",
			Current:   5,
			Total:     10,
		}

		result := e.ToEvent()

		if result.Type != TypeProgress {
			t.Errorf("Type = %q, want %q", result.Type, TypeProgress)
		}

		if result.Data["task_id"] != "task-123" {
			t.Error("task_id mismatch")
		}

		if result.Data["phase"] != "implementation" {
			t.Error("phase mismatch")
		}

		if result.Data["message"] != "Writing code..." {
			t.Error("message mismatch")
		}

		if result.Data["current"] != 5 {
			t.Error("current mismatch")
		}

		if result.Data["total"] != 10 {
			t.Error("total mismatch")
		}
	})

	t.Run("auto timestamp", func(t *testing.T) {
		e := ProgressEvent{
			TaskID:  "task-789",
			Phase:   "planning",
			Message: "Creating specs...",
		}
		result := e.ToEvent()

		if result.Timestamp.IsZero() {
			t.Error("timestamp should be auto-set")
		}
	})
}

func TestErrorEvent(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		testErr := errors.New("something went wrong")
		e := ErrorEvent{
			Error:  testErr,
			TaskID: "task-123",
			Fatal:  true,
		}

		result := e.ToEvent()

		if result.Type != TypeError {
			t.Errorf("Type = %q, want %q", result.Type, TypeError)
		}

		if result.Data["error"] != "something went wrong" {
			t.Error("error message mismatch")
		}

		if result.Data["task_id"] != "task-123" {
			t.Error("task_id mismatch")
		}

		if result.Data["fatal"] != true {
			t.Error("fatal should be true")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		e := ErrorEvent{
			TaskID: "task-456",
			Fatal:  false,
		}

		result := e.ToEvent()

		if result.Data["error"] != "" {
			t.Error("error should be empty string for nil error")
		}

		if result.Data["fatal"] != false {
			t.Error("fatal should be false")
		}
	})
}

func TestFileChangedEvent(t *testing.T) {
	e := FileChangedEvent{
		TaskID:    "task-123",
		Path:      "/path/to/file.go",
		Operation: "create",
	}

	result := e.ToEvent()

	if result.Type != TypeFileChanged {
		t.Errorf("Type = %q, want %q", result.Type, TypeFileChanged)
	}

	if result.Data["task_id"] != "task-123" {
		t.Error("task_id mismatch")
	}

	if result.Data["path"] != "/path/to/file.go" {
		t.Error("path mismatch")
	}

	if result.Data["operation"] != "create" {
		t.Error("operation mismatch")
	}

	if result.Timestamp.IsZero() {
		t.Error("timestamp should be auto-set")
	}
}

func TestCheckpointEvent(t *testing.T) {
	e := CheckpointEvent{
		TaskID:  "task-123",
		Commit:  "abc123def",
		Message: "Checkpoint: Initial implementation",
	}

	result := e.ToEvent()

	if result.Type != TypeCheckpoint {
		t.Errorf("Type = %q, want %q", result.Type, TypeCheckpoint)
	}

	if result.Data["task_id"] != "task-123" {
		t.Error("task_id mismatch")
	}

	if result.Data["commit"] != "abc123def" {
		t.Error("commit mismatch")
	}

	if result.Data["message"] != "Checkpoint: Initial implementation" {
		t.Error("message mismatch")
	}
}

func TestAgentMessageEvent(t *testing.T) {
	e := AgentMessageEvent{
		TaskID:  "task-123",
		Content: "I'm implementing the feature",
		Role:    "assistant",
	}

	result := e.ToEvent()

	if result.Type != TypeAgentMessage {
		t.Errorf("Type = %q, want %q", result.Type, TypeAgentMessage)
	}

	if result.Data["task_id"] != "task-123" {
		t.Error("task_id mismatch")
	}

	if result.Data["content"] != "I'm implementing the feature" {
		t.Error("content mismatch")
	}

	if result.Data["role"] != "assistant" {
		t.Error("role mismatch")
	}
}

func TestBlueprintReadyEvent(t *testing.T) {
	e := BlueprintReadyEvent{
		TaskID:      "task-123",
		BlueprintID: "bp-456",
	}

	result := e.ToEvent()

	if result.Type != TypeBlueprintReady {
		t.Errorf("Type = %q, want %q", result.Type, TypeBlueprintReady)
	}

	if result.Data["task_id"] != "task-123" {
		t.Error("task_id mismatch")
	}

	if result.Data["blueprint_id"] != "bp-456" {
		t.Error("blueprint_id mismatch")
	}
}

func TestBranchCreatedEvent(t *testing.T) {
	e := BranchCreatedEvent{
		TaskID: "task-123",
		Branch: "feature/new-feature",
	}

	result := e.ToEvent()

	if result.Type != TypeBranchCreated {
		t.Errorf("Type = %q, want %q", result.Type, TypeBranchCreated)
	}

	if result.Data["task_id"] != "task-123" {
		t.Error("task_id mismatch")
	}

	if result.Data["branch"] != "feature/new-feature" {
		t.Error("branch mismatch")
	}
}

func TestPlanCompletedEvent(t *testing.T) {
	e := PlanCompletedEvent{
		TaskID:          "task-123",
		SpecificationID: 42,
	}

	result := e.ToEvent()

	if result.Type != TypePlanCompleted {
		t.Errorf("Type = %q, want %q", result.Type, TypePlanCompleted)
	}

	if result.Data["task_id"] != "task-123" {
		t.Error("task_id mismatch")
	}

	if result.Data["specification_id"] != 42 {
		t.Error("specification_id mismatch")
	}
}

func TestImplementDoneEvent(t *testing.T) {
	e := ImplementDoneEvent{
		TaskID:   "task-123",
		DiffStat: "10 files changed, 100 insertions(+), 50 deletions(-)",
	}

	result := e.ToEvent()

	if result.Type != TypeImplementDone {
		t.Errorf("Type = %q, want %q", result.Type, TypeImplementDone)
	}

	if result.Data["task_id"] != "task-123" {
		t.Error("task_id mismatch")
	}

	if result.Data["diff_stat"] != "10 files changed, 100 insertions(+), 50 deletions(-)" {
		t.Error("diff_stat mismatch")
	}
}

func TestPRCreatedEvent(t *testing.T) {
	e := PRCreatedEvent{
		TaskID:   "task-123",
		PRURL:    "https://github.com/user/repo/pull/42",
		PRNumber: 42,
	}

	result := e.ToEvent()

	if result.Type != TypePRCreated {
		t.Errorf("Type = %q, want %q", result.Type, TypePRCreated)
	}

	if result.Data["task_id"] != "task-123" {
		t.Error("task_id mismatch")
	}

	if result.Data["pr_number"] != 42 {
		t.Error("pr_number mismatch")
	}

	if result.Data["pr_url"] != "https://github.com/user/repo/pull/42" {
		t.Error("pr_url mismatch")
	}
}

func TestBrowserActionEvent(t *testing.T) {
	e := BrowserActionEvent{
		Action:   "click",
		URL:      "https://example.com",
		Selector: "#submit-btn",
		Success:  true,
	}

	result := e.ToEvent()

	if result.Type != TypeBrowserAction {
		t.Errorf("Type = %q, want %q", result.Type, TypeBrowserAction)
	}

	if result.Data["action"] != "click" {
		t.Error("action mismatch")
	}

	if result.Data["url"] != "https://example.com" {
		t.Error("url mismatch")
	}

	if result.Data["selector"] != "#submit-btn" {
		t.Error("selector mismatch")
	}

	if result.Data["success"] != true {
		t.Error("success should be true")
	}
}

func TestBrowserActionEventError(t *testing.T) {
	e := BrowserActionEvent{
		Action:  "click",
		Success: false,
		Error:   "element not found",
	}

	result := e.ToEvent()

	if result.Data["success"] != false {
		t.Error("success should be false")
	}

	if result.Data["error"] != "element not found" {
		t.Error("error mismatch")
	}
}

func TestBrowserTabOpenedEvent(t *testing.T) {
	e := BrowserTabOpenedEvent{
		TabID: "tab-123",
		URL:   "https://example.com",
		Title: "Example Page",
	}

	result := e.ToEvent()

	if result.Type != TypeBrowserTabOpened {
		t.Errorf("Type = %q, want %q", result.Type, TypeBrowserTabOpened)
	}

	if result.Data["tab_id"] != "tab-123" {
		t.Error("tab_id mismatch")
	}

	if result.Data["url"] != "https://example.com" {
		t.Error("url mismatch")
	}

	if result.Data["title"] != "Example Page" {
		t.Error("title mismatch")
	}
}

func TestBrowserScreenshotEvent(t *testing.T) {
	e := BrowserScreenshotEvent{
		TabID:    "tab-123",
		Format:   "png",
		FullPath: "/screenshots/screenshot.png",
	}

	result := e.ToEvent()

	if result.Type != TypeBrowserScreenshot {
		t.Errorf("Type = %q, want %q", result.Type, TypeBrowserScreenshot)
	}

	if result.Data["tab_id"] != "tab-123" {
		t.Error("tab_id mismatch")
	}

	if result.Data["format"] != "png" {
		t.Error("format mismatch")
	}

	if result.Data["full_path"] != "/screenshots/screenshot.png" {
		t.Error("full_path mismatch")
	}
}

func TestEventTimestamps(t *testing.T) {
	// Test all event types set timestamp when zero
	now := time.Now()
	events := []struct {
		name  string
		event interface{ ToEvent() eventbus.Event }
	}{
		{"StateChanged", &StateChangedEvent{From: "a", To: "b"}},
		{"Progress", &ProgressEvent{TaskID: "t1"}},
		{"Error", &ErrorEvent{TaskID: "t2"}},
		{"FileChanged", &FileChangedEvent{TaskID: "t3"}},
		{"Checkpoint", &CheckpointEvent{TaskID: "t4"}},
		{"AgentMessage", &AgentMessageEvent{TaskID: "t5"}},
		{"BlueprintReady", &BlueprintReadyEvent{TaskID: "t6"}},
		{"BranchCreated", &BranchCreatedEvent{TaskID: "t7"}},
		{"PlanCompleted", &PlanCompletedEvent{TaskID: "t8"}},
		{"ImplementDone", &ImplementDoneEvent{TaskID: "t9"}},
		{"PRCreated", &PRCreatedEvent{TaskID: "t10"}},
		{"BrowserAction", &BrowserActionEvent{Action: "test"}},
		{"BrowserTabOpened", &BrowserTabOpenedEvent{TabID: "t11"}},
		{"BrowserScreenshot", &BrowserScreenshotEvent{TabID: "t12"}},
	}

	for _, tt := range events {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.ToEvent()
			if result.Timestamp.IsZero() {
				t.Errorf("Timestamp should be auto-set for %s", tt.name)
			}
			if result.Timestamp.Before(now.Add(-1 * time.Second)) {
				t.Errorf("Timestamp should be recent for %s", tt.name)
			}
		})
	}
}
