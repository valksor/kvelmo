// Package validation provides configuration validation for mehrhof.
//
// This package re-exports the generic validation framework from go-toolkit
// for backward compatibility and convenience.
package validation

import "github.com/valksor/go-toolkit/validate"

// Re-export types from go-toolkit for backward compatibility.
type (
	// Severity indicates the importance of a validation finding.
	Severity = validate.Severity
	// Finding represents a single validation issue.
	Finding = validate.Finding
	// Result holds all validation findings.
	Result = validate.Result
)

// Severity constants.
const (
	SeverityError   = validate.SeverityError
	SeverityWarning = validate.SeverityWarning
	SeverityInfo    = validate.SeverityInfo
)

// Re-export functions from go-toolkit for backward compatibility.
var (
	// NewResult creates an empty validation result.
	NewResult = validate.NewResult
)

// AddError adds an error finding.
// This is a convenience function to avoid type assertions.
func AddError(r *Result, code, message, path, file string) {
	r.AddError(code, message, path, file)
}

// AddErrorWithSuggestion adds an error finding with a fix suggestion.
func AddErrorWithSuggestion(r *Result, code, message, path, file, suggestion string) {
	r.AddErrorWithSuggestion(code, message, path, file, suggestion)
}

// AddWarning adds a warning finding.
func AddWarning(r *Result, code, message, path, file string) {
	r.AddWarning(code, message, path, file)
}

// AddWarningWithSuggestion adds a warning finding with a fix suggestion.
func AddWarningWithSuggestion(r *Result, code, message, path, file, suggestion string) {
	r.AddWarningWithSuggestion(code, message, path, file, suggestion)
}

// AddInfo adds an informational finding.
func AddInfo(r *Result, code, message, path, file string) {
	r.AddInfo(code, message, path, file)
}

// Merge combines another result into this one.
func Merge(r, other *Result) {
	r.Merge(other)
}

// Format returns the result in the specified format.
func Format(r *Result, format string) string {
	return r.Format(format)
}
