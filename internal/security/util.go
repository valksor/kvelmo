package security

import (
	"fmt"
	"strings"
	"time"
)

// ParseSeverity converts string to Severity.
func ParseSeverity(s string) Severity {
	switch strings.ToLower(s) {
	case "critical":
		return SeverityCritical
	case "high":
		return SeverityHigh
	case "medium":
		return SeverityMedium
	case "low":
		return SeverityLow
	case "any":
		return SeverityInfo
	default:
		return SeverityCritical
	}
}

// IsToolEnabled checks if a specific tool is enabled in the tools configuration.
// Returns true if the tool is explicitly enabled or if no tools are configured (default enabled).
func IsToolEnabled(tools []map[string]interface{}, toolName string) bool {
	// No tools configured - enable by default
	if len(tools) == 0 {
		return true
	}

	// Check if the tool is explicitly enabled
	for _, tool := range tools {
		name, nameOk := tool["name"].(string)
		if !nameOk || name != toolName {
			continue
		}

		enabled, ok := tool["enabled"].(bool)

		return ok && enabled
	}

	// Tool not found in config - disable
	return false
}

// FormatFindings formats findings for display.
func FormatFindings(results []*ScanResult) string {
	var sb strings.Builder

	// Separate results by status
	var skipped []string
	var success []*ScanResult
	var errors []*ScanResult

	totalFindings := 0
	totalBySeverity := make(map[Severity]int)

	// Collect totals and categorize by status
	for _, result := range results {
		switch result.Status {
		case ScanStatusSkipped:
			skipped = append(skipped, result.Scanner)
		case ScanStatusSuccess:
			totalFindings += result.Summary.Total
			for severity, count := range result.Summary.BySeverity {
				totalBySeverity[severity] += count
			}
			success = append(success, result)
		case ScanStatusError:
			errors = append(errors, result)
		}
	}

	// Display skipped tools at top
	if len(skipped) > 0 {
		sb.WriteString("## Skipped Tools\n\n")
		sb.WriteString(fmt.Sprintf("The following tools were not available and were skipped: %s\n\n", strings.Join(skipped, ", ")))
		sb.WriteString("Install manually or enable auto-download in config.\n\n")
	}

	// Display errors
	for _, result := range errors {
		sb.WriteString(fmt.Sprintf("## %s: Error\n\n", result.Scanner))
		if result.Error != nil {
			sb.WriteString(fmt.Sprintf("%s\n\n", result.Error))
		}
	}

	// Display successful scans
	for _, result := range success {
		sb.WriteString(fmt.Sprintf("## %s (%s)\n\n", result.Scanner, result.Duration))
		sb.WriteString(fmt.Sprintf("Found %d issue(s)\n\n", result.Summary.Total))

		for _, finding := range result.Findings {
			sb.WriteString(fmt.Sprintf("### %s\n\n", finding.Title))
			sb.WriteString(fmt.Sprintf("**Severity**: %s\n", finding.Severity))
			sb.WriteString(fmt.Sprintf("**Location**: %s:%d\n", finding.Location.File, finding.Location.Line))
			sb.WriteString(fmt.Sprintf("**Description**: %s\n", finding.Description))

			if finding.CVE != "" {
				sb.WriteString(fmt.Sprintf("**CVE**: %s\n", finding.CVE))
			}

			if finding.Fix != nil {
				sb.WriteString(fmt.Sprintf("**Fix**: %s\n", finding.Fix.Description))
				if finding.Fix.Command != "" {
					sb.WriteString(fmt.Sprintf("**Command**: `%s`\n", finding.Fix.Command))
				}
			}

			sb.WriteString("\n")
		}
	}

	// Add overall summary
	sb.WriteString("---\n\n")
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("**Total Findings**: %d (from %d scanner(s))\n\n", totalFindings, len(success)))

	if len(totalBySeverity) > 0 {
		sb.WriteString("**By Severity**:\n")
		// Order by severity (high to low)
		for _, severity := range []Severity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo} {
			if count, ok := totalBySeverity[severity]; ok && count > 0 {
				sb.WriteString(fmt.Sprintf("- %s: %d\n", severity, count))
			}
		}
	}

	return sb.String()
}

// ShouldBlock determines if findings should block based on configuration.
func ShouldBlock(results []*ScanResult, blockLevel Severity) bool {
	for _, result := range results {
		// Skip errors and skipped tools
		if result.Status != ScanStatusSuccess {
			continue
		}

		for _, finding := range result.Findings {
			if finding.Severity.Rank() >= blockLevel.Rank() {
				return true
			}
		}
	}

	return false
}

// GetBlockingFindings returns all findings that match or exceed the block level.
func GetBlockingFindings(results []*ScanResult, blockLevel Severity) []Finding {
	var blocking []Finding

	for _, result := range results {
		// Skip errors and skipped tools
		if result.Status != ScanStatusSuccess {
			continue
		}

		for _, finding := range result.Findings {
			if finding.Severity.Rank() >= blockLevel.Rank() {
				blocking = append(blocking, finding)
			}
		}
	}

	return blocking
}

// skippedResult returns a ScanResult for a skipped scanner.
func skippedResult(scannerName string, duration time.Duration) *ScanResult {
	return &ScanResult{
		Scanner:  scannerName,
		Status:   ScanStatusSkipped,
		Findings: []Finding{},
		Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
		Duration: duration,
	}
}

// FormatSARIFReport formats a SARIF report as a JSON string.
// Returns the JSON string or an error message string if formatting fails.
func FormatSARIFReport(report *SARIFReport) string {
	data, err := report.MarshalJSON()
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return string(data)
}

// FormatJSONResults formats scan results as a JSON string.
// Returns the JSON string or an error message string if formatting fails.
func FormatJSONResults(results []*ScanResult) string {
	data, err := MarshalJSONResults(results)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return string(data)
}
