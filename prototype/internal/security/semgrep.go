package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SemgrepScanner wraps the Semgrep cross-language security scanner.
type SemgrepScanner struct {
	enabled bool
	config  *SemgrepConfig
}

// SemgrepConfig holds configuration for the Semgrep scanner.
type SemgrepConfig struct {
	// Config specifies the ruleset to use (default: "auto")
	Config string `yaml:"config"`
	// Exclude specifies patterns to exclude from scanning
	Exclude []string `yaml:"exclude"`
	// Severity specifies the minimum severity level to report (error, warning, info)
	Severity string `yaml:"severity"`
	// Timeout specifies the timeout for the scan in seconds
	Timeout int `yaml:"timeout"`
}

// SemgrepOutput represents the JSON output from semgrep.
type SemgrepOutput struct {
	Results []SemgrepResult `json:"results"`
	Errors  []SemgrepError  `json:"errors"`
	Version string          `json:"version"`
}

// SemgrepResult represents a single semgrep finding.
type SemgrepResult struct {
	CheckID string `json:"check_id"`
	Path    string `json:"path"`
	Start   struct {
		Line   int `json:"line"`
		Col    int `json:"col"`
		Offset int `json:"offset"`
	} `json:"start"`
	End struct {
		Line   int `json:"line"`
		Col    int `json:"col"`
		Offset int `json:"offset"`
	} `json:"end"`
	Extra struct {
		Message   string         `json:"message"`
		Metadata  map[string]any `json:"metadata"`
		Severity  string         `json:"severity"`
		Lines     string         `json:"lines"`
		Fix       string         `json:"fix"`
		Metavars  map[string]any `json:"metavars"`
		IsIgnored bool           `json:"is_ignored"`
	} `json:"extra"`
}

// SemgrepError represents an error from semgrep.
type SemgrepError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Level   string `json:"level"`
	Path    string `json:"path"`
}

// NewSemgrepScanner creates a new Semgrep scanner.
func NewSemgrepScanner(enabled bool, config *SemgrepConfig) *SemgrepScanner {
	if config == nil {
		config = &SemgrepConfig{
			Config: "auto",
		}
	}
	if config.Config == "" {
		config.Config = "auto"
	}

	return &SemgrepScanner{
		enabled: enabled,
		config:  config,
	}
}

// Name returns the name of the scanner.
func (s *SemgrepScanner) Name() string {
	return "semgrep"
}

// IsEnabled returns whether the scanner is enabled.
func (s *SemgrepScanner) IsEnabled() bool {
	return s.enabled
}

// Scan runs the Semgrep scanner on the given directory.
func (s *SemgrepScanner) Scan(ctx context.Context, dir string) (*ScanResult, error) {
	start := time.Now()

	// Check if semgrep is installed
	if _, lookupErr := exec.LookPath("semgrep"); lookupErr != nil {
		//nolint:nilerr // Error is communicated via ScanResult.Error for partial scan support
		return &ScanResult{
			Scanner:  s.Name(),
			Status:   ScanStatusSkipped,
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: time.Since(start),
			Error:    errors.New("semgrep not installed. Run: pip install semgrep"),
		}, nil
	}

	// Build command args
	args := []string{
		"scan",
		"--json",
		"--config", s.config.Config,
	}

	// Add severity filter if specified
	if s.config.Severity != "" {
		args = append(args, "--severity", s.config.Severity)
	}

	// Add exclude patterns
	for _, pattern := range s.config.Exclude {
		args = append(args, "--exclude", pattern)
	}

	// Add timeout if specified
	if s.config.Timeout > 0 {
		args = append(args, "--timeout", strconv.Itoa(s.config.Timeout))
	}

	// Add target directory
	args = append(args, dir)

	// Run semgrep
	cmd := exec.CommandContext(ctx, "semgrep", args...)
	cmd.Dir = dir

	var stdout, stderr limitedBuffer
	stdout.limit = maxOutputSize
	stderr.limit = maxOutputSize
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	duration := time.Since(start)

	// Check if semgrep is installed (command not found)
	if runErr != nil && isCommandNotFound(runErr) {
		return &ScanResult{
			Scanner:  s.Name(),
			Status:   ScanStatusSkipped,
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    errors.New("semgrep not installed. Run: pip install semgrep"),
		}, nil
	}

	// Check if output exceeded size limit
	if stdout.Len() >= maxOutputSize {
		return &ScanResult{
			Scanner:  s.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    errors.New("semgrep output exceeded maximum size limit"),
			Status:   ScanStatusError,
		}, nil
	}

	// Parse JSON output
	var semgrepOutput SemgrepOutput
	if parseErr := json.Unmarshal(stdout.Bytes(), &semgrepOutput); parseErr != nil {
		// Semgrep may have failed with a non-JSON error
		if runErr != nil {
			return &ScanResult{
				Scanner:  s.Name(),
				Findings: []Finding{},
				Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
				Duration: duration,
				Error:    fmt.Errorf("semgrep failed: %w (stderr: %s)", runErr, stderr.String()),
				Status:   ScanStatusError,
			}, nil
		}

		return &ScanResult{
			Scanner:  s.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    fmt.Errorf("failed to parse semgrep output: %w", parseErr),
			Status:   ScanStatusError,
		}, nil
	}

	// Convert to Findings
	findings := make([]Finding, 0, len(semgrepOutput.Results))
	for i, result := range semgrepOutput.Results {
		if result.Extra.IsIgnored {
			continue
		}

		finding := Finding{
			ID:          fmt.Sprintf("semgrep-%d", i),
			Scanner:     "semgrep",
			Severity:    mapSemgrepSeverity(result.Extra.Severity),
			Title:       result.CheckID,
			Description: result.Extra.Message,
			Location: Location{
				File:      result.Path,
				Line:      result.Start.Line,
				Column:    result.Start.Col,
				EndLine:   result.End.Line,
				EndColumn: result.End.Col,
			},
			Metadata: make(map[string]string),
		}

		// Add code snippet if available
		if result.Extra.Lines != "" {
			finding.Code = &CodeSnippet{
				Before: result.Extra.Lines,
			}
		}

		// Add fix suggestion if available
		if result.Extra.Fix != "" {
			finding.Fix = &FixSuggestion{
				Description: "Apply suggested fix",
				Patch:       result.Extra.Fix,
			}
		}

		// Extract CWE/OWASP from metadata if present
		if result.Extra.Metadata != nil {
			if cwe, ok := result.Extra.Metadata["cwe"].(string); ok {
				finding.Metadata["cwe"] = cwe
			}
			if cwes, ok := result.Extra.Metadata["cwe"].([]any); ok && len(cwes) > 0 {
				if cwe, ok := cwes[0].(string); ok {
					finding.Metadata["cwe"] = cwe
				}
			}
			if owasp, ok := result.Extra.Metadata["owasp"].(string); ok {
				finding.Metadata["owasp"] = owasp
			}
			if owasps, ok := result.Extra.Metadata["owasp"].([]any); ok && len(owasps) > 0 {
				if owasp, ok := owasps[0].(string); ok {
					finding.Metadata["owasp"] = owasp
				}
			}
			if category, ok := result.Extra.Metadata["category"].(string); ok {
				finding.Metadata["category"] = category
			}
		}

		finding.Metadata["check_id"] = result.CheckID

		findings = append(findings, finding)
	}

	// Build summary
	summary := SummarizeFindings(findings)

	return &ScanResult{
		Scanner:  s.Name(),
		Findings: findings,
		Summary:  summary,
		Duration: duration,
		Status:   ScanStatusSuccess,
	}, nil
}

// mapSemgrepSeverity converts semgrep severity to our Severity type.
func mapSemgrepSeverity(severity string) Severity {
	switch strings.ToLower(severity) {
	case "error":
		return SeverityHigh
	case "warning":
		return SeverityMedium
	case "info":
		return SeverityLow
	default:
		return SeverityInfo
	}
}
