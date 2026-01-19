package display

import (
	"fmt"
	"strings"

	"github.com/valksor/go-toolkit/display"
)

// Suggestion represents a suggested action for error recovery.
type Suggestion struct {
	Command     string
	Description string
}

// ErrorWithSuggestions formats an error message with actionable suggestions.
func ErrorWithSuggestions(message string, suggestions []Suggestion) string {
	var sb strings.Builder

	// Error header
	sb.WriteString(display.ErrorMsg("%s", message))
	sb.WriteString("\n")

	// Add suggestions if any
	if len(suggestions) > 0 {
		sb.WriteString("\n")
		sb.WriteString(display.Muted("Suggested actions:"))
		sb.WriteString("\n")
		for _, s := range suggestions {
			sb.WriteString(fmt.Sprintf("  %s %s - %s\n",
				display.Muted("•"),
				display.Cyan(s.Command),
				s.Description,
			))
		}
	}

	return sb.String()
}

// ErrorWithContext formats an error with contextual information and suggestions.
func ErrorWithContext(err error, context string, suggestions []Suggestion) string {
	var sb strings.Builder

	// Context header
	sb.WriteString(display.ErrorMsg("Error: %s", context))
	sb.WriteString("\n")

	// Add the underlying error if present
	if err != nil {
		sb.WriteString(display.Muted(fmt.Sprintf("  Cause: %v", err)))
		sb.WriteString("\n")
	}

	// Add suggestions if any
	if len(suggestions) > 0 {
		sb.WriteString("\n")
		sb.WriteString(display.Muted("Suggested actions:"))
		sb.WriteString("\n")
		for _, s := range suggestions {
			sb.WriteString(fmt.Sprintf("  %s %s - %s\n",
				display.Muted("•"),
				display.Cyan(s.Command),
				s.Description,
			))
		}
	}

	return sb.String()
}

// ValidationError formats a validation error with field-specific suggestions.
func ValidationError(field string, message string, suggestions []Suggestion) string {
	var sb strings.Builder

	sb.WriteString(display.ErrorMsg("Validation Error: %s", field))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n", message))

	if len(suggestions) > 0 {
		sb.WriteString("\n")
		sb.WriteString(display.Muted("Fix:"))
		sb.WriteString("\n")
		for _, s := range suggestions {
			sb.WriteString(fmt.Sprintf("  %s %s\n",
				display.Muted("•"),
				s.Description,
			))
		}
	}

	return sb.String()
}

// ProviderError formats a provider-specific error with contextual help.
func ProviderError(provider string, err error, suggestions []Suggestion) string {
	return ErrorWithContext(
		err,
		fmt.Sprintf("Failed to load from %s provider", provider),
		suggestions,
	)
}

// Common error messages with suggestions

// NoActiveTaskError returns a formatted "no active task" error.
func NoActiveTaskError() string {
	return ErrorWithSuggestions(
		"No active task",
		[]Suggestion{
			{Command: "mehr start <reference>", Description: "Start a new task"},
			{Command: "mehr list", Description: "View all tasks in workspace"},
		},
	)
}

// TaskFailedError returns a formatted task failure error with next steps.
func TaskFailedError(step string, err error) string {
	return ErrorWithContext(
		err,
		"Task failed during "+step,
		[]Suggestion{
			{Command: "mehr status", Description: "View detailed error information"},
			{Command: "mehr note", Description: "Add notes about the error"},
			{Command: "mehr undo", Description: "Revert last changes if applicable"},
		},
	)
}

// ConfigError returns a formatted configuration error.
func ConfigError(err error, configPath string) string {
	return ErrorWithContext(
		err,
		"Configuration error in "+configPath,
		[]Suggestion{
			{Command: "mehr config validate", Description: "Validate workspace configuration"},
			{Command: "cat .mehrhof/config.yaml", Description: "View configuration file"},
		},
	)
}

// AgentError returns a formatted agent-related error.
func AgentError(agent string, err error) string {
	return ErrorWithContext(
		err,
		"Agent error: "+agent,
		[]Suggestion{
			{Command: "mehr agents list", Description: "List available agents"},
			{Command: "mehr --agent=<name> <ref>", Description: "Try with a different agent"},
		},
	)
}

// GitError returns a formatted git-related error.
func GitError(operation string, err error) string {
	return ErrorWithContext(
		err,
		fmt.Sprintf("Git %s failed", operation),
		[]Suggestion{
			{Command: "git status", Description: "Check git repository status"},
			{Command: "mehr start --no-branch <ref>", Description: "Skip branch creation"},
		},
	)
}

// ConductorError returns a formatted error for conductor initialization failures.
// It wraps internal "conductor" terminology into user-friendly messaging.
func ConductorError(stage string, err error) string {
	var suggestions []Suggestion

	// Provide context-aware suggestions based on the error
	switch stage {
	case "initialize":
		suggestions = []Suggestion{
			{Command: "mehr init", Description: "Initialize workspace"},
			{Command: "mehr status", Description: "Check workspace status"},
		}
	case "register":
		suggestions = []Suggestion{
			{Command: "mehr agents list", Description: "List available agents"},
			{Command: "mehr --agent=<name> <ref>", Description: "Specify an agent explicitly"},
		}
	default:
		suggestions = []Suggestion{
			{Command: "mehr status", Description: "Check current status"},
			{Command: "mehr guide", Description: "Get next-step guidance"},
		}
	}

	return ErrorWithContext(
		err,
		fmt.Sprintf("Failed to %s workspace", stage),
		suggestions,
	)
}

// WorkspaceError returns a formatted error for workspace-related failures.
// It suggests initializing the workspace if that's likely the issue.
func WorkspaceError(operation string, err error) string {
	suggestions := []Suggestion{
		{Command: "mehr init", Description: "Initialize workspace in current directory"},
	}

	// Add fallback suggestion
	if operation != "open" {
		suggestions = append(suggestions, Suggestion{
			Command:     "mehr status",
			Description: "Check workspace status",
		})
	}

	return ErrorWithContext(
		err,
		fmt.Sprintf("Workspace %s failed", operation),
		suggestions,
	)
}
