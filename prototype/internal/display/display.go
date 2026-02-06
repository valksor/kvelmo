// Package display provides user-friendly formatting for internal state values.
// This separates display concerns from internal state representation, allowing
// user-facing text to evolve without breaking stored YAML data.
package display

import (
	"strings"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/display"
)

// StateDisplay maps internal workflow state values to user-friendly names.
var StateDisplay = map[workflow.State]string{
	workflow.StateIdle:          "Ready",
	workflow.StatePlanning:      "Planning",
	workflow.StateImplementing:  "Implementing",
	workflow.StateReviewing:     "Reviewing",
	workflow.StateDone:          "Completed",
	workflow.StateFailed:        "Failed",
	workflow.StateWaiting:       "Waiting",
	workflow.StateCheckpointing: "Checkpointing",
	workflow.StateReverting:     "Reverting",
	workflow.StateRestoring:     "Restoring",
}

// StateDescription provides additional context for each state.
var StateDescription = map[workflow.State]string{
	workflow.StateIdle:          "Ready to start",
	workflow.StatePlanning:      "AI is creating specifications",
	workflow.StateImplementing:  "AI is generating code",
	workflow.StateReviewing:     "Code review in progress",
	workflow.StateDone:          "Task completed successfully",
	workflow.StateFailed:        "Task failed with error",
	workflow.StateWaiting:       "Action required: Awaiting your response",
	workflow.StateCheckpointing: "Creating checkpoint",
	workflow.StateReverting:     "Reverting to previous state",
	workflow.StateRestoring:     "Restoring from checkpoint",
}

// StateAccessiblePrefix provides short text prefixes for accessibility.
// These help color-blind users distinguish states without relying on color alone.
var StateAccessiblePrefix = map[workflow.State]string{
	workflow.StateIdle:          "[*]", // Ready (neutral asterisk)
	workflow.StatePlanning:      "[P]", // Planning
	workflow.StateImplementing:  "[I]", // Implementing
	workflow.StateReviewing:     "[R]", // Reviewing (R is more intuitive than V)
	workflow.StateDone:          "[D]", // Done
	workflow.StateFailed:        "[F]", // Failed
	workflow.StateWaiting:       "[W]", // Waiting
	workflow.StateCheckpointing: "[C]", // Checkpointing
	workflow.StateReverting:     "[←]", // Reverting (arrow = going back)
	workflow.StateRestoring:     "[→]", // Restoring (arrow = going forward)
}

// ──────────────────────────────────────────────────────────────────────────────
// Progress Phase - context-aware display for idle state
// ──────────────────────────────────────────────────────────────────────────────

// ProgressPhase represents the task progress when in idle state.
// This allows showing meaningful state names like "Planned" or "Implemented"
// instead of generic "Ready".
type ProgressPhase string

const (
	PhaseStarted     ProgressPhase = "started"     // Task started, no planning done
	PhasePlanned     ProgressPhase = "planned"     // Specs exist, not implemented
	PhaseImplemented ProgressPhase = "implemented" // Files implemented, not reviewed
	PhaseReviewed    ProgressPhase = "reviewed"    // Has review feedback
)

// IdlePhaseDisplay maps progress phases to user-friendly names.
var IdlePhaseDisplay = map[ProgressPhase]string{
	PhaseStarted:     "Started",
	PhasePlanned:     "Planned",
	PhaseImplemented: "Implemented",
	PhaseReviewed:    "Reviewed",
}

// IdlePhaseDescription maps progress phases to actionable descriptions.
var IdlePhaseDescription = map[ProgressPhase]string{
	PhaseStarted:     "Run 'mehr plan' to create specifications",
	PhasePlanned:     "Run 'mehr implement' to generate code",
	PhaseImplemented: "Run 'mehr review' or 'mehr finish'",
	PhaseReviewed:    "Run 'mehr finish' to complete",
}

// ──────────────────────────────────────────────────────────────────────────────
// Optional Modifiers - additional workflow phases that can be run
// ──────────────────────────────────────────────────────────────────────────────

// OptionalModifiers represents optional workflow phases that have been run.
// These are displayed as suffixes to the main progress phase (e.g., "Implemented • Simplified").
type OptionalModifiers struct {
	Optimized  bool
	Simplified bool
}

// FormatOptionalModifiers returns a suffix string for optional modifiers.
// Returns "" if no modifiers, or " • Optimized", " • Simplified", " • Optimized • Simplified".
func FormatOptionalModifiers(mods OptionalModifiers) string {
	var parts []string
	if mods.Optimized {
		parts = append(parts, "Optimized")
	}
	if mods.Simplified {
		parts = append(parts, "Simplified")
	}
	if len(parts) == 0 {
		return ""
	}

	return " • " + strings.Join(parts, " • ")
}

// ──────────────────────────────────────────────────────────────────────────────
// Specification Status
// ──────────────────────────────────────────────────────────────────────────────

// SpecificationStatusDisplay maps internal specification status values to user-friendly names.
var SpecificationStatusDisplay = map[string]string{
	storage.SpecificationStatusDraft:        "Draft",
	storage.SpecificationStatusReady:        "Ready",
	storage.SpecificationStatusImplementing: "Implementing",
	storage.SpecificationStatusDone:         "Completed",
}

// SpecificationStatusIcon returns the visual icon for a specification status.
var SpecificationStatusIcon = map[string]string{
	storage.SpecificationStatusDraft:        "○", // empty circle
	storage.SpecificationStatusReady:        "◐", // half-filled
	storage.SpecificationStatusImplementing: "◑", // half-filled alternate
	storage.SpecificationStatusDone:         "●", // filled circle
}

// FormatState returns the user-friendly display name for a workflow state.
// Falls back to the raw state string if not found in the mapping.
func FormatState(state workflow.State) string {
	if name, ok := StateDisplay[state]; ok {
		return name
	}

	return string(state)
}

// FormatStateString returns the user-friendly display name for a state string.
func FormatStateString(state string) string {
	return FormatState(workflow.State(state))
}

// GetStateDescription returns a brief description of what the state means.
func GetStateDescription(state workflow.State) string {
	if desc, ok := StateDescription[state]; ok {
		return desc
	}

	return ""
}

// ──────────────────────────────────────────────────────────────────────────────
// Progress-aware state formatting
// ──────────────────────────────────────────────────────────────────────────────

// DetectProgressPhase determines the progress phase from task context.
// Use this to get the appropriate phase for progress-aware display functions.
func DetectProgressPhase(hasSpecs, hasImplementedFiles, hasReviews bool) ProgressPhase {
	switch {
	case hasReviews:
		return PhaseReviewed
	case hasImplementedFiles:
		return PhaseImplemented
	case hasSpecs:
		return PhasePlanned
	default:
		return PhaseStarted
	}
}

// FormatIdleStateWithProgress returns the display name for idle state with progress context.
func FormatIdleStateWithProgress(phase ProgressPhase) string {
	if name, ok := IdlePhaseDisplay[phase]; ok {
		return name
	}

	return "Started"
}

// GetIdleStateDescription returns the description for idle state with progress context.
func GetIdleStateDescription(phase ProgressPhase) string {
	if desc, ok := IdlePhaseDescription[phase]; ok {
		return desc
	}

	return "Run 'mehr plan' to create specifications"
}

// FormatStateWithProgress returns state display name with progress awareness.
// For idle state, returns phase-specific name (Started/Planned/Implemented/Reviewed).
// For other states, returns the standard display name.
func FormatStateWithProgress(state workflow.State, phase ProgressPhase) string {
	if state == workflow.StateIdle {
		return FormatIdleStateWithProgress(phase)
	}

	return FormatState(state)
}

// GetStateDescriptionWithProgress returns description with progress awareness.
// For idle state, returns phase-specific actionable description.
// For other states, returns the standard description.
func GetStateDescriptionWithProgress(state workflow.State, phase ProgressPhase) string {
	if state == workflow.StateIdle {
		return GetIdleStateDescription(phase)
	}

	return GetStateDescription(state)
}

// FormatStateColoredWithProgress returns colored state with progress awareness.
// Format: "[*] Implemented" where the prefix is muted and the name is colored.
func FormatStateColoredWithProgress(state workflow.State, phase ProgressPhase) string {
	prefix := display.Muted(GetStateAccessiblePrefix(state))
	displayName := FormatStateWithProgress(state, phase)
	coloredName := ColorState(string(state), displayName)

	return prefix + " " + coloredName
}

// FormatStateWithProgressAndModifiers returns state display name with progress and optional modifiers.
// Example: "Implemented • Simplified" or "Planned • Optimized".
func FormatStateWithProgressAndModifiers(state workflow.State, phase ProgressPhase, mods OptionalModifiers) string {
	baseName := FormatStateWithProgress(state, phase)

	return baseName + FormatOptionalModifiers(mods)
}

// FormatStateColoredWithProgressAndModifiers returns colored state with progress and modifiers.
// Format: "[*] Implemented • Simplified" where the prefix is muted and the name is colored.
func FormatStateColoredWithProgressAndModifiers(state workflow.State, phase ProgressPhase, mods OptionalModifiers) string {
	prefix := display.Muted(GetStateAccessiblePrefix(state))
	displayName := FormatStateWithProgressAndModifiers(state, phase, mods)
	coloredName := ColorState(string(state), displayName)

	return prefix + " " + coloredName
}

// FormatSpecificationStatus returns the user-friendly display name for a specification status.
// Falls back to the raw status string if not found in the mapping.
func FormatSpecificationStatus(status string) string {
	if name, ok := SpecificationStatusDisplay[status]; ok {
		return name
	}

	return status
}

// GetSpecificationStatusIcon returns the icon for a specification status.
// Returns "?" for unknown statuses.
func GetSpecificationStatusIcon(status string) string {
	if icon, ok := SpecificationStatusIcon[status]; ok {
		return icon
	}

	return "?"
}

// FormatSpecificationStatusWithIcon returns "icon status" format for display.
func FormatSpecificationStatusWithIcon(status string) string {
	return GetSpecificationStatusIcon(status) + " " + FormatSpecificationStatus(status)
}

// Color-aware formatting functions

// GetStateAccessiblePrefix returns the accessibility prefix for a state.
func GetStateAccessiblePrefix(state workflow.State) string {
	if prefix, ok := StateAccessiblePrefix[state]; ok {
		return prefix
	}

	return "[?]"
}

// FormatStateColored returns a colored state display name with accessibility prefix.
// Format: "[P] Planning" where the prefix is muted and the name is colored.
func FormatStateColored(state workflow.State) string {
	prefix := display.Muted(GetStateAccessiblePrefix(state))
	displayName := FormatState(state)
	coloredName := ColorState(string(state), displayName)

	return prefix + " " + coloredName
}

// FormatStateStringColored returns a colored state display name from a string.
func FormatStateStringColored(state string) string {
	return FormatStateColored(workflow.State(state))
}

// FormatSpecificationStatusColored returns a colored specification status.
func FormatSpecificationStatusColored(status string) string {
	displayName := FormatSpecificationStatus(status)

	return ColorSpecStatus(status, displayName)
}

// FormatSpecificationStatusWithIconColored returns colored "icon status" format.
func FormatSpecificationStatusWithIconColored(status string) string {
	icon := GetSpecificationStatusIcon(status)
	name := FormatSpecificationStatusColored(status)

	return icon + " " + name
}
