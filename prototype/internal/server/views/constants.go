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

// Button classes - Tailwind CSS classes for action buttons.
const (
	BtnPrimary   = "inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-offset-gray-800"
	BtnSecondary = "inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 text-sm font-medium rounded-md text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-offset-gray-800"
	BtnDanger    = "inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 dark:focus:ring-offset-gray-800"
	BtnSuccess   = "inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 dark:focus:ring-offset-gray-800"
	BtnWarning   = "inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-yellow-600 hover:bg-yellow-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-yellow-500 dark:focus:ring-offset-gray-800"
	BtnGhost     = "inline-flex items-center px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-offset-gray-800"
)

// Progress bar colors - for budget and completion indicators.
const (
	ProgressGreen  = "bg-green-500"
	ProgressYellow = "bg-yellow-500"
	ProgressRed    = "bg-red-500"
	ProgressBlue   = "bg-blue-500"
	ProgressPurple = "bg-purple-500"
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
var StateDisplay = map[string]StateDisplayInfo{
	StateIdle: {
		Icon:     "○",
		Badge:    "Ready",
		Color:    "text-gray-500 dark:text-gray-400",
		BarColor: "bg-gray-500",
	},
	StatePlanning: {
		Icon:     "◐",
		Badge:    "Planning...",
		Color:    "text-blue-500 dark:text-blue-400",
		BarColor: "bg-blue-500",
	},
	StateImplementing: {
		Icon:     "◑",
		Badge:    "Implementing...",
		Color:    "text-purple-500 dark:text-purple-400",
		BarColor: "bg-purple-500",
	},
	StateReviewing: {
		Icon:     "◉",
		Badge:    "Reviewing...",
		Color:    "text-orange-500 dark:text-orange-400",
		BarColor: "bg-orange-500",
	},
	StateDone: {
		Icon:     "●",
		Badge:    "Done",
		Color:    "text-green-500 dark:text-green-400",
		BarColor: "bg-green-500",
	},
	StateFailed: {
		Icon:     "✗",
		Badge:    "Failed",
		Color:    "text-red-500 dark:text-red-400",
		BarColor: "bg-red-500",
	},
	StateWaiting: {
		Icon:     "?",
		Badge:    "Waiting...",
		Color:    "text-yellow-500 dark:text-yellow-400",
		BarColor: "bg-yellow-500",
	},
	StatePaused: {
		Icon:     "⏸",
		Badge:    "Paused",
		Color:    "text-yellow-600 dark:text-yellow-500",
		BarColor: "bg-yellow-600",
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
		Color:    "text-gray-500 dark:text-gray-400",
		BarColor: "bg-gray-500",
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
var SpecStatusDisplay = map[string]SpecStatusDisplayInfo{
	SpecStatusPending: {
		Icon:  "○",
		Color: "text-gray-400 dark:text-gray-500",
	},
	SpecStatusActive: {
		Icon:  "◐",
		Color: "text-blue-500 dark:text-blue-400",
	},
	SpecStatusCompleted: {
		Icon:  "●",
		Color: "text-green-500 dark:text-green-400",
	},
	SpecStatusSkipped: {
		Icon:  "⊘",
		Color: "text-gray-400 dark:text-gray-500",
	},
}

// GetSpecStatusDisplay returns display info for a spec status.
func GetSpecStatusDisplay(status string) SpecStatusDisplayInfo {
	if info, ok := SpecStatusDisplay[status]; ok {
		return info
	}

	return SpecStatusDisplayInfo{
		Icon:  "?",
		Color: "text-gray-400 dark:text-gray-500",
	}
}

// Label color palettes - deterministic colors based on label hash.
var labelColors = []string{
	"bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
	"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
	"bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
	"bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
	"bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200",
	"bg-pink-100 text-pink-800 dark:bg-pink-900 dark:text-pink-200",
	"bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200",
	"bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200",
	"bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200",
	"bg-teal-100 text-teal-800 dark:bg-teal-900 dark:text-teal-200",
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
