// Package display provides domain-specific display formatting for mehrhof.
// For common display utilities, use github.com/valksor/go-toolkit/display.
package display

import toolkitdisplay "github.com/valksor/go-toolkit/display"

// ColorState returns a colored state string based on the state value.
func ColorState(state, displayName string) string {
	switch state {
	case "idle":
		return toolkitdisplay.Muted(displayName)
	case "planning", "implementing", "reviewing", "checkpointing":
		return toolkitdisplay.Info(displayName)
	case "done":
		return toolkitdisplay.Success(displayName)
	case "failed":
		return toolkitdisplay.Error(displayName)
	case "waiting":
		return toolkitdisplay.Warning(displayName)
	default:
		return displayName
	}
}

// ColorSpecStatus returns a colored specification status.
func ColorSpecStatus(status, displayName string) string {
	switch status {
	case "draft":
		return toolkitdisplay.Muted(displayName)
	case "ready":
		return toolkitdisplay.Warning(displayName)
	case "implementing":
		return toolkitdisplay.Info(displayName)
	case "done":
		return toolkitdisplay.Success(displayName)
	default:
		return displayName
	}
}

// WorktreeIndicator returns a visual indicator showing the current context.
// Use this to help users understand if they're in a worktree or main repo.
func WorktreeIndicator(isWorktree bool) string {
	if isWorktree {
		return toolkitdisplay.Muted("[worktree]")
	}

	return ""
}
