package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BanditScanner wraps the Bandit Python security linter.
type BanditScanner struct {
	enabled bool
	config  *BanditConfig
}

// BanditConfig holds configuration for the Bandit scanner.
type BanditConfig struct {
	// Severity specifies the minimum severity level (low, medium, high)
	Severity string `yaml:"severity"`
	// Confidence specifies the minimum confidence level (low, medium, high)
	Confidence string `yaml:"confidence"`
	// Exclude specifies paths to exclude from scanning
	Exclude []string `yaml:"exclude"`
	// Skip specifies test IDs to skip (e.g., B101, B102)
	Skip []string `yaml:"skip"`
}

// BanditOutput represents the JSON output from Bandit.
type BanditOutput struct {
	Errors  []BanditError  `json:"errors"`
	Results []BanditResult `json:"results"`
	Metrics struct {
		TotalIssues int `json:"_totals"`
	} `json:"metrics"`
}

// BanditError represents an error from Bandit.
type BanditError struct {
	Filename string `json:"filename"`
	Reason   string `json:"reason"`
}

// BanditResult represents a single Bandit finding.
type BanditResult struct {
	Code            string `json:"code"`
	Col             int    `json:"col_offset"`
	EndCol          int    `json:"end_col_offset"`
	Filename        string `json:"filename"`
	IssueConfidence string `json:"issue_confidence"`
	IssueCWE        struct {
		ID   int    `json:"id"`
		Link string `json:"link"`
	} `json:"issue_cwe"`
	IssueSeverity string `json:"issue_severity"`
	IssueText     string `json:"issue_text"`
	LineNumber    int    `json:"line_number"`
	EndLineNumber int    `json:"end_line_number"`
	MoreInfo      string `json:"more_info"`
	TestID        string `json:"test_id"`
	TestName      string `json:"test_name"`
}

// NewBanditScanner creates a new Bandit scanner.
func NewBanditScanner(enabled bool, config *BanditConfig) *BanditScanner {
	if config == nil {
		config = &BanditConfig{}
	}

	return &BanditScanner{
		enabled: enabled,
		config:  config,
	}
}

// Name returns the name of the scanner.
func (b *BanditScanner) Name() string {
	return "bandit"
}

// IsEnabled returns whether the scanner is enabled.
func (b *BanditScanner) IsEnabled() bool {
	return b.enabled
}

// Scan runs Bandit on the given directory.
func (b *BanditScanner) Scan(ctx context.Context, dir string) (*ScanResult, error) {
	start := time.Now()

	// Check if bandit is installed
	if _, lookupErr := exec.LookPath("bandit"); lookupErr != nil {
		//nolint:nilerr // Error is communicated via ScanResult.Error for partial scan support
		return &ScanResult{
			Scanner:  b.Name(),
			Status:   ScanStatusSkipped,
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: time.Since(start),
			Error:    errors.New("bandit not installed. Run: pip install bandit"),
		}, nil
	}

	// Build command args
	args := []string{
		"-r",         // Recursive
		"-f", "json", // JSON format
		"-q", // Quiet (don't print banner)
	}

	// Add severity filter
	if b.config.Severity != "" {
		args = append(args, "-l")
		switch strings.ToLower(b.config.Severity) {
		case "high":
			args = append(args, "-ll") // Only high severity
		case "medium":
			args = append(args, "-l") // Medium and above
			// low is default, no additional flag needed
		}
	}

	// Add confidence filter
	if b.config.Confidence != "" {
		args = append(args, "-i")
		switch strings.ToLower(b.config.Confidence) {
		case "high":
			args = append(args, "-ii") // Only high confidence
		case "medium":
			args = append(args, "-i") // Medium and above
			// low is default, no additional flag needed
		}
	}

	// Add exclude patterns
	if len(b.config.Exclude) > 0 {
		args = append(args, "--exclude", strings.Join(b.config.Exclude, ","))
	}

	// Add skip test IDs
	if len(b.config.Skip) > 0 {
		args = append(args, "--skip", strings.Join(b.config.Skip, ","))
	}

	// Add target directory
	args = append(args, dir)

	// Run bandit
	cmd := exec.CommandContext(ctx, "bandit", args...)
	cmd.Dir = dir

	stdout := newLimitedBuffer(maxOutputSize)
	stderr := newLimitedBuffer(maxOutputSize)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Bandit returns non-zero exit code when issues are found
	_ = cmd.Run()
	duration := time.Since(start)

	// Check if output exceeded size limit
	if stdout.Len() >= maxOutputSize {
		return &ScanResult{
			Scanner:  b.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    errors.New("bandit output exceeded maximum size limit"),
			Status:   ScanStatusError,
		}, nil
	}

	// Handle empty output (no Python files found)
	if stdout.Len() == 0 {
		return &ScanResult{
			Scanner:  b.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Status:   ScanStatusSuccess,
		}, nil
	}

	// Parse JSON output
	var banditOutput BanditOutput
	if parseErr := json.Unmarshal(stdout.Bytes(), &banditOutput); parseErr != nil {
		return &ScanResult{
			Scanner:  b.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    fmt.Errorf("failed to parse bandit output: %w", parseErr),
			Status:   ScanStatusError,
		}, nil
	}

	// Convert to Findings
	findings := make([]Finding, 0, len(banditOutput.Results))
	for i, result := range banditOutput.Results {
		finding := Finding{
			ID:          fmt.Sprintf("bandit-%d", i),
			Scanner:     "bandit",
			Severity:    mapBanditSeverity(result.IssueSeverity),
			Title:       fmt.Sprintf("%s: %s", result.TestID, result.TestName),
			Description: result.IssueText,
			Location: Location{
				File:      result.Filename,
				Line:      result.LineNumber,
				Column:    result.Col,
				EndLine:   result.EndLineNumber,
				EndColumn: result.EndCol,
			},
			Metadata: make(map[string]string),
		}

		// Add code snippet
		if result.Code != "" {
			finding.Code = &CodeSnippet{
				Before: result.Code,
			}
		}

		// Add CWE information
		if result.IssueCWE.ID > 0 {
			finding.Metadata["cwe_id"] = fmt.Sprintf("CWE-%d", result.IssueCWE.ID)
			finding.Metadata["cwe_link"] = result.IssueCWE.Link
		}

		// Add metadata
		finding.Metadata["test_id"] = result.TestID
		finding.Metadata["test_name"] = result.TestName
		finding.Metadata["confidence"] = result.IssueConfidence
		if result.MoreInfo != "" {
			finding.Metadata["more_info"] = result.MoreInfo
		}

		// Add fix suggestion with link to more info
		if result.MoreInfo != "" {
			finding.Fix = &FixSuggestion{
				Description: fmt.Sprintf("See Bandit documentation for %s: %s", result.TestID, result.MoreInfo),
			}
		}

		findings = append(findings, finding)
	}

	// Build summary
	summary := SummarizeFindings(findings)

	return &ScanResult{
		Scanner:  b.Name(),
		Findings: findings,
		Summary:  summary,
		Duration: duration,
		Status:   ScanStatusSuccess,
	}, nil
}

// mapBanditSeverity converts Bandit severity to our Severity type.
func mapBanditSeverity(severity string) Severity {
	switch strings.ToUpper(severity) {
	case "HIGH":
		return SeverityHigh
	case "MEDIUM":
		return SeverityMedium
	case "LOW":
		return SeverityLow
	default:
		return SeverityInfo
	}
}
