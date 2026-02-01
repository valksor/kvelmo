package security

import (
	"context"
	"time"
)

// Severity represents the severity level of a security finding.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Rank returns a numeric rank for severity (higher = more severe).
func (s Severity) Rank() int {
	switch s {
	case SeverityCritical:
		return 5
	case SeverityHigh:
		return 4
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

// Finding represents a single security finding.
type Finding struct {
	ID          string            `json:"id"`
	Scanner     string            `json:"scanner"`
	Severity    Severity          `json:"severity"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Location    Location          `json:"location"`
	Code        *CodeSnippet      `json:"code,omitempty"`
	CVE         string            `json:"cve,omitempty"`
	Fix         *FixSuggestion    `json:"fix,omitempty"`
	Metadata    map[string]string `json:"metadata"`
}

// Location represents the location of a finding in the codebase.
type Location struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	Length    int    `json:"length,omitempty"`     // Character length of the finding
	EndLine   int    `json:"end_line,omitempty"`   // End line (if multi-line)
	EndColumn int    `json:"end_column,omitempty"` // End column
}

// CodeSnippet represents a code snippet associated with a finding.
type CodeSnippet struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

// FixSuggestion represents a suggested fix for a finding.
type FixSuggestion struct {
	Description string `json:"description"`
	Patch       string `json:"patch,omitempty"`
	Command     string `json:"command,omitempty"`
}

// Summary represents a summary of scan results.
type Summary struct {
	Total      int              `json:"total"`
	BySeverity map[Severity]int `json:"by_severity"`
}

// ScanStatus represents the status of a scan.
type ScanStatus string

const (
	ScanStatusSuccess ScanStatus = "success"
	ScanStatusSkipped ScanStatus = "skipped" // Tool not installed
	ScanStatusError   ScanStatus = "error"
)

// ScanResult represents the result of running a scanner.
type ScanResult struct {
	Scanner  string        `json:"scanner"`
	Findings []Finding     `json:"findings"`
	Summary  Summary       `json:"summary"`
	Duration time.Duration `json:"duration"`
	Error    error         `json:"error,omitempty"`
	Status   ScanStatus    `json:"status"`
}

// Scanner interface defines the contract for security scanners.
type Scanner interface {
	// Name returns the name of the scanner.
	Name() string

	// Scan runs the security scanner on the given directory.
	Scan(ctx context.Context, dir string) (*ScanResult, error)

	// IsEnabled returns whether the scanner is enabled.
	IsEnabled() bool
}
