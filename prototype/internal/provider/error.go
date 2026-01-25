package provider

import (
	"fmt"
	"strings"
)

// DiagnosticError provides detailed error information with recovery hints.
type DiagnosticError struct {
	Provider    string   // Provider name (e.g., "github", "jira")
	Cause       string   // Root cause description
	Message     string   // User-friendly error message
	Suggestions []string // Recovery steps
	DocsURL     string   // Optional documentation URL
	Err         error    // Underlying error for error chain
}

func (e *DiagnosticError) Error() string {
	msg := fmt.Sprintf("%s error: %s", e.Provider, e.Message)
	if e.Cause != "" {
		msg += "\nCause: " + e.Cause
	}
	if len(e.Suggestions) > 0 {
		msg += "\n\nRecovery steps:"
		var msgSb24 strings.Builder
		for i, step := range e.Suggestions {
			msgSb24.WriteString(fmt.Sprintf("\n  %d. %s", i+1, step))
		}
		msg += msgSb24.String()
	}
	if e.DocsURL != "" {
		msg += "\n\nDocumentation: " + e.DocsURL
	}

	return msg
}

// Unwrap returns the underlying error for error chain support.
func (e *DiagnosticError) Unwrap() error {
	return e.Err
}

// DiagnosticHints provides error-specific recovery suggestions.
type DiagnosticHints interface {
	// DiagnosticHints returns recovery suggestions for an error.
	// Returns nil if no hints available.
	DiagnosticHints(err error) []string
}

// WrapDiagnosticError wraps an error with diagnostic information.
func WrapDiagnosticError(provider, message, cause string, suggestions []string, docsURL string) error {
	return &DiagnosticError{
		Provider:    provider,
		Cause:       cause,
		Message:     message,
		Suggestions: suggestions,
		DocsURL:     docsURL,
	}
}

// CommonHints provides common diagnostic hints for frequent errors.
var CommonHints = struct {
	Unauthorized []string
	RateLimited  []string
	NotFound     []string
	Network      []string
}{
	Unauthorized: []string{
		"Check that your API token is correctly set",
		"Verify the token has not expired",
		"Ensure the token has required permissions/scopes",
	},
	RateLimited: []string{
		"Wait for the rate limit to reset",
		"Consider reducing request frequency",
		"Check if a higher tier plan is available",
	},
	NotFound: []string{
		"Verify the reference ID is correct",
		"Check that the resource exists",
		"Ensure you have access to the resource",
	},
	Network: []string{
		"Check your internet connection",
		"Verify the provider API is accessible",
		"Check if a proxy or firewall is blocking requests",
	},
}
