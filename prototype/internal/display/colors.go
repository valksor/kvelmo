// Package display provides domain-specific display formatting for mehrhof.
// For common display utilities, use github.com/valksor/go-toolkit/display.
package display

import "github.com/valksor/go-toolkit/display"

// ColorState returns a colored state string based on the state value.
func ColorState(state, displayName string) string {
	switch state {
	case "idle":
		return display.Muted(displayName)
	case "planning", "implementing", "reviewing", "checkpointing":
		return display.Info(displayName)
	case "done":
		return display.Success(displayName)
	case "failed":
		return display.Error(displayName)
	case "waiting":
		return display.Warning(displayName)
	default:
		return displayName
	}
}

// ColorSpecStatus returns a colored specification status.
func ColorSpecStatus(status, displayName string) string {
	switch status {
	case "draft":
		return display.Muted(displayName)
	case "ready":
		return display.Warning(displayName)
	case "implementing":
		return display.Info(displayName)
	case "done":
		return display.Success(displayName)
	default:
		return displayName
	}
}

// WorktreeIndicator returns a visual indicator showing the current context.
// Use this to help users understand if they're in a worktree or main repo.
func WorktreeIndicator(isWorktree bool) string {
	if isWorktree {
		return display.Muted("[worktree]")
	}

	return ""
}
