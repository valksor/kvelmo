// Package views provides pre-computed view data structures and computation functions
// for the web UI, following the principle "templates render, handlers decide."
package views

// Workflow states - canonical state names used throughout the UI.
const (
	StateIdle         = "idle"
	StatePlanning     = "planning"
	StateImplementing = "implementing"
	StateReviewing    = "reviewing"
	StateDone         = "done"
	StateFailed       = "failed"
	StateWaiting      = "waiting"
	StatePaused       = "paused"
)

// SSE event names - used for server-sent events and HTMX triggers.
const (
	EventWorkflowStateChanged = "workflow_state_changed"
	EventSpecUpdated          = "spec_updated"
	EventQuestionAsked        = "question_asked"
	EventCostsUpdated         = "costs_updated"
	EventTaskCreated          = "task_created"
	EventTaskCompleted        = "task_completed"
	EventBudgetWarning        = "budget_warning"
	EventBudgetLimit          = "budget_limit"
	EventQuickTasksUpdated    = "quick_tasks_updated"
	EventHierarchyUpdated     = "hierarchy_updated"
)

// Work types - different types of active work items.
const (
	WorkTypeTask    = "task"
	WorkTypeQuick   = "quick"
	WorkTypeProject = "project"
)

// Button classes - DaisyUI button classes.
const (
	BtnPrimary   = "btn btn-primary"
	BtnSecondary = "btn btn-secondary"
	BtnDanger    = "btn btn-error"
	BtnSuccess   = "btn btn-success"
	BtnWarning   = "btn btn-warning"
	BtnGhost     = "btn btn-ghost"
)

// Progress bar colors - DaisyUI semantic colors for budget and completion indicators.
const (
	ProgressGreen  = "bg-success"
	ProgressYellow = "bg-warning"
	ProgressRed    = "bg-error"
	ProgressBlue   = "bg-info"
	ProgressPurple = "bg-primary"
)

// StateDisplayInfo contains all display properties for a workflow state.
type StateDisplayInfo struct {
	Icon     string // Unicode icon character
	Badge    string // Human-readable badge text
	Color    string // Text color class
	BarColor string // Progress bar color class
}

// StateDisplay maps workflow states to their display properties.
// This is the single source of truth for state visualization.
// Uses DaisyUI semantic color classes.
var StateDisplay = map[string]StateDisplayInfo{
	StateIdle: {
		Icon:     "○",
		Badge:    "Ready",
		Color:    "text-base-content/60",
		BarColor: "bg-base-300",
	},
	StatePlanning: {
		Icon:     "◐",
		Badge:    "Planning...",
		Color:    "text-info",
		BarColor: "bg-info",
	},
	StateImplementing: {
		Icon:     "◑",
		Badge:    "Implementing...",
		Color:    "text-warning",
		BarColor: "bg-warning",
	},
	StateReviewing: {
		Icon:     "◉",
		Badge:    "Reviewing...",
		Color:    "text-primary",
		BarColor: "bg-primary",
	},
	StateDone: {
		Icon:     "●",
		Badge:    "Done",
		Color:    "text-success",
		BarColor: "bg-success",
	},
	StateFailed: {
		Icon:     "✗",
		Badge:    "Failed",
		Color:    "text-error",
		BarColor: "bg-error",
	},
	StateWaiting: {
		Icon:     "?",
		Badge:    "Waiting...",
		Color:    "text-warning",
		BarColor: "bg-warning",
	},
	StatePaused: {
		Icon:     "⏸",
		Badge:    "Paused",
		Color:    "text-neutral",
		BarColor: "bg-neutral",
	},
}

// GetStateDisplay returns display info for a state, with a fallback for unknown states.
func GetStateDisplay(state string) StateDisplayInfo {
	if info, ok := StateDisplay[state]; ok {
		return info
	}
	// Fallback for unknown states
	return StateDisplayInfo{
		Icon:     "?",
		Badge:    state,
		Color:    "text-base-content/60",
		BarColor: "bg-base-300",
	}
}

// SpecStatus constants for specification states.
const (
	SpecStatusPending   = "pending"
	SpecStatusActive    = "active"
	SpecStatusCompleted = "completed"
	SpecStatusSkipped   = "skipped"
)

// SpecStatusDisplayInfo contains display properties for spec statuses.
type SpecStatusDisplayInfo struct {
	Icon  string
	Color string
}

// SpecStatusDisplay maps spec statuses to their display properties.
// Uses DaisyUI semantic color classes.
var SpecStatusDisplay = map[string]SpecStatusDisplayInfo{
	SpecStatusPending: {
		Icon:  "○",
		Color: "text-base-content/40",
	},
	SpecStatusActive: {
		Icon:  "◐",
		Color: "text-info",
	},
	SpecStatusCompleted: {
		Icon:  "●",
		Color: "text-success",
	},
	SpecStatusSkipped: {
		Icon:  "⊘",
		Color: "text-base-content/40",
	},
}

// GetSpecStatusDisplay returns display info for a spec status.
func GetSpecStatusDisplay(status string) SpecStatusDisplayInfo {
	if info, ok := SpecStatusDisplay[status]; ok {
		return info
	}

	return SpecStatusDisplayInfo{
		Icon:  "?",
		Color: "text-base-content/40",
	}
}

// Label color palettes - DaisyUI badge variants for deterministic colors based on label hash.
var labelColors = []string{
	"badge badge-info badge-outline",
	"badge badge-success badge-outline",
	"badge badge-warning badge-outline",
	"badge badge-error badge-outline",
	"badge badge-primary badge-outline",
	"badge badge-secondary badge-outline",
	"badge badge-accent badge-outline",
	"badge badge-neutral badge-outline",
	"badge badge-info badge-soft",
	"badge badge-success badge-soft",
}

// LabelColor returns a deterministic color class for a label based on its hash.
func LabelColor(label string) string {
	hash := 0
	for _, c := range label {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}

	return labelColors[hash%len(labelColors)]
}
