package events

import (
	"time"

	"github.com/valksor/go-toolkit/eventbus"
)

// Type identifies event categories.
// This is an alias for go-toolkit's eventbus.Type for domain-specific event type constants.
type Type = eventbus.Type

// Event is the base event structure from go-toolkit.
// This is an alias for convenience in defining domain-specific events.
type Event = eventbus.Event

// Domain-specific event type constants for mehrhof.
const (
	TypeStateChanged   Type = "state_changed"
	TypeProgress       Type = "progress"
	TypeError          Type = "error"
	TypeFileChanged    Type = "file_changed"
	TypeAgentMessage   Type = "agent_message"
	TypeCheckpoint     Type = "checkpoint"
	TypeBlueprintReady Type = "blueprint_ready"

	// GitHub-related events.
	TypeBranchCreated Type = "branch_created"
	TypePlanCompleted Type = "plan_completed"
	TypeImplementDone Type = "implement_done"
	TypePRCreated     Type = "pr_created"

	// Browser-related events.
	TypeBrowserAction     Type = "browser_action"
	TypeBrowserTabOpened  Type = "browser_tab_opened"
	TypeBrowserScreenshot Type = "browser_screenshot"

	// Sandbox-related events.
	TypeSandboxStatusChanged Type = "sandbox_status_changed"
)

// SandboxStatusChangedEvent when sandbox status changes.
type SandboxStatusChangedEvent struct {
	Enabled  bool
	Active   bool
	Platform string
}

func (e SandboxStatusChangedEvent) ToEvent() eventbus.Event {
	return eventbus.Event{
		Type: TypeSandboxStatusChanged,
		Data: map[string]any{
			"enabled":  e.Enabled,
			"active":   e.Active,
			"platform": e.Platform,
		},
	}
}

// StateChangedEvent when workflow state changes.
type StateChangedEvent struct {
	From      string
	To        string
	Event     string // Triggering event
	TaskID    string
	Timestamp time.Time
}

func (e StateChangedEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypeStateChanged,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"from":    e.From,
			"to":      e.To,
			"event":   e.Event,
			"task_id": e.TaskID,
		},
	}
}

// ProgressEvent for progress updates.
type ProgressEvent struct {
	Timestamp time.Time
	TaskID    string
	Phase     string
	Message   string
	Current   int
	Total     int
}

func (e ProgressEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypeProgress,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"task_id": e.TaskID,
			"phase":   e.Phase,
			"message": e.Message,
			"current": e.Current,
			"total":   e.Total,
		},
	}
}

// ErrorEvent for errors.
type ErrorEvent struct {
	Timestamp time.Time
	Error     error
	TaskID    string
	Fatal     bool
}

func (e ErrorEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	errMsg := ""
	if e.Error != nil {
		errMsg = e.Error.Error()
	}

	return eventbus.Event{
		Type:      TypeError,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"task_id": e.TaskID,
			"error":   errMsg,
			"fatal":   e.Fatal,
		},
	}
}

// FileChangedEvent when files are modified.
type FileChangedEvent struct {
	TaskID    string
	Path      string
	Operation string // create, update, delete
	Timestamp time.Time
}

func (e FileChangedEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypeFileChanged,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"task_id":   e.TaskID,
			"path":      e.Path,
			"operation": e.Operation,
		},
	}
}

// CheckpointEvent when a checkpoint is created.
type CheckpointEvent struct {
	Timestamp time.Time
	TaskID    string
	Commit    string
	Message   string
}

func (e CheckpointEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypeCheckpoint,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"task_id": e.TaskID,
			"commit":  e.Commit,
			"message": e.Message,
		},
	}
}

// AgentMessageEvent for agent output.
type AgentMessageEvent struct {
	TaskID    string
	Content   string
	Role      string // assistant, tool, system
	Timestamp time.Time
}

func (e AgentMessageEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypeAgentMessage,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"task_id": e.TaskID,
			"content": e.Content,
			"role":    e.Role,
		},
	}
}

// BlueprintReadyEvent when a blueprint is ready.
type BlueprintReadyEvent struct {
	Timestamp   time.Time
	TaskID      string
	BlueprintID string
}

func (e BlueprintReadyEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypeBlueprintReady,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"task_id":      e.TaskID,
			"blueprint_id": e.BlueprintID,
		},
	}
}

// BranchCreatedEvent when a task branch is created.
type BranchCreatedEvent struct {
	Timestamp time.Time
	TaskID    string
	Branch    string
}

func (e BranchCreatedEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypeBranchCreated,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"task_id": e.TaskID,
			"branch":  e.Branch,
		},
	}
}

// PlanCompletedEvent when planning phase completes.
type PlanCompletedEvent struct {
	Timestamp       time.Time
	TaskID          string
	SpecificationID int
}

func (e PlanCompletedEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypePlanCompleted,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"task_id":          e.TaskID,
			"specification_id": e.SpecificationID,
		},
	}
}

// ImplementDoneEvent when implementation phase completes.
type ImplementDoneEvent struct {
	Timestamp time.Time
	TaskID    string
	DiffStat  string
}

func (e ImplementDoneEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypeImplementDone,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"task_id":   e.TaskID,
			"diff_stat": e.DiffStat,
		},
	}
}

// PRCreatedEvent when a pull request is created.
type PRCreatedEvent struct {
	Timestamp time.Time
	TaskID    string
	PRURL     string
	PRNumber  int
}

func (e PRCreatedEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypePRCreated,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"task_id":   e.TaskID,
			"pr_number": e.PRNumber,
			"pr_url":    e.PRURL,
		},
	}
}

// BrowserActionEvent for browser automation actions.
type BrowserActionEvent struct {
	Timestamp time.Time
	Action    string
	URL       string
	Selector  string
	Success   bool
	Error     string
}

func (e BrowserActionEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypeBrowserAction,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"action":   e.Action,
			"url":      e.URL,
			"selector": e.Selector,
			"success":  e.Success,
			"error":    e.Error,
		},
	}
}

// BrowserTabOpenedEvent when a browser tab is opened.
type BrowserTabOpenedEvent struct {
	Timestamp time.Time
	TabID     string
	URL       string
	Title     string
}

func (e BrowserTabOpenedEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypeBrowserTabOpened,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"tab_id": e.TabID,
			"url":    e.URL,
			"title":  e.Title,
		},
	}
}

// BrowserScreenshotEvent when a screenshot is captured.
type BrowserScreenshotEvent struct {
	Timestamp time.Time
	TabID     string
	Format    string
	FullPath  string
}

func (e BrowserScreenshotEvent) ToEvent() eventbus.Event {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	return eventbus.Event{
		Type:      TypeBrowserScreenshot,
		Timestamp: e.Timestamp,
		Data: map[string]any{
			"tab_id":    e.TabID,
			"format":    e.Format,
			"full_path": e.FullPath,
		},
	}
}
