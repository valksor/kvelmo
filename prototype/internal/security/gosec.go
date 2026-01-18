package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxOutputSize = 10 * 1024 * 1024 // 10MB maximum output size to prevent memory exhaustion

// limitedBuffer is a buffer with a size limit to prevent memory exhaustion.
type limitedBuffer struct {
	buf   []byte
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - len(b.buf)
	if remaining <= 0 {
		return len(p), nil // Silent discard when limit reached
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	b.buf = append(b.buf, p...)

	return len(p), nil
}

func (b *limitedBuffer) Bytes() []byte {
	return b.buf
}

func (b *limitedBuffer) String() string {
	return string(b.buf)
}

func (b *limitedBuffer) Len() int {
	return len(b.buf)
}

func (b *limitedBuffer) Reset() {
	b.buf = b.buf[:0]
}

// GosecScanner wraps the gosec security scanner.
type GosecScanner struct {
	enabled bool
	config  *GosecConfig
	tm      *ToolManager
}

// GosecConfig holds configuration for the gosec scanner.
type GosecConfig struct {
	// Severity controls the minimum severity level to report.
	Severity string `yaml:"severity"`
	// Confidence controls the minimum confidence level to report.
	Confidence string `yaml:"confidence"`
	// Exclude specifies a list of files to exclude.
	Exclude []string `yaml:"exclude"`
	// Include specifies a list of files to include.
	Include []string `yaml:"include"`
}

// GosecIssue represents a single gosec finding.
type GosecIssue struct {
	Severity   string `json:"severity"`
	Confidence string `json:"confidence"`
	RuleID     string `json:"rule_id"`
	What       string `json:"details"`
	File       string `json:"file"`
	Line       string `json:"line"`
	Col        string `json:"col"`
}

// GosecOutput represents the JSON output from gosec.
type GosecOutput struct {
	Issues []GosecIssue `json:"Issues"`
}

// NewGosecScanner creates a new gosec scanner.
func NewGosecScanner(enabled bool, config *GosecConfig, tm *ToolManager) *GosecScanner {
	if config == nil {
		config = &GosecConfig{}
	}

	return &GosecScanner{
		enabled: enabled,
		config:  config,
		tm:      tm,
	}
}

// Name returns the name of the scanner.
func (g *GosecScanner) Name() string {
	return "gosec"
}

// IsEnabled returns whether the scanner is enabled.
func (g *GosecScanner) IsEnabled() bool {
	return g.enabled
}

// Scan runs the gosec scanner on the given directory.
func (g *GosecScanner) Scan(ctx context.Context, dir string) (*ScanResult, error) {
	start := time.Now()

	// Validate config-derived arguments
	if err := validateGosecConfig(g.config); err != nil {
		return &ScanResult{
			Scanner:  g.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: time.Since(start),
			Error:    fmt.Errorf("invalid gosec config: %w", err),
			Status:   ScanStatusError,
		}, nil
	}

	// Build command args
	args := []string{
		"-fmt", "json",
		"-stdout",
		"-no-fail",
	}

	if g.config.Severity != "" {
		args = append(args, "-severity", g.config.Severity)
	}
	if g.config.Confidence != "" {
		args = append(args, "-confidence", g.config.Confidence)
	}
	if len(g.config.Exclude) > 0 {
		for _, excl := range g.config.Exclude {
			args = append(args, "-exclude", excl)
		}
	}

	args = append(args, dir)

	// Get gosec binary path
	binaryName := "gosec"
	if g.tm != nil {
		spec := ToolSpec{
			Name:       "gosec",
			Repository: "securego/gosec",
			BinaryName: "gosec",
		}
		binaryPath, toolErr := g.tm.EnsureTool(ctx, spec)
		// Tool not available - skip it
		if toolErr == nil {
			binaryName = binaryPath
		} else {
			return skippedResult(g.Name(), time.Since(start)), nil
		}
	}

	// Run gosec
	cmd := exec.CommandContext(ctx, binaryName, args...)
	cmd.Dir = dir // Explicitly set working directory to validated scan directory

	// Use limited buffers to prevent memory exhaustion from maliciously large outputs
	var stdout, stderr limitedBuffer
	stdout.limit = maxOutputSize
	stderr.limit = maxOutputSize
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	duration := time.Since(start)

	// Check if gosec is installed
	if runErr != nil && isCommandNotFound(runErr) {
		return skippedResult(g.Name(), duration), nil
	}

	// Check if output exceeded size limit
	if stdout.Len() >= maxOutputSize {
		return &ScanResult{
			Scanner:  g.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    errors.New("gosec output exceeded maximum size limit"),
			Status:   ScanStatusError,
		}, nil
	}

	// Parse JSON output
	var gosecResults GosecOutput
	if parseErr := json.Unmarshal(stdout.Bytes(), &gosecResults); parseErr != nil {
		return &ScanResult{
			Scanner:  g.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    fmt.Errorf("failed to parse gosec output: %w", parseErr),
			Status:   ScanStatusError,
		}, nil
	}

	// Convert to Findings
	findings := make([]Finding, 0, len(gosecResults.Issues))
	for i, issue := range gosecResults.Issues {
		// Parse line and column from strings with proper error handling
		line, err := strconv.Atoi(issue.Line)
		if err != nil {
			slog.Warn("Invalid line number in gosec finding", "finding_id", i, "line_value", issue.Line, "error", err)

			continue // Skip this finding as we can't properly locate it
		}
		col, err := strconv.Atoi(issue.Col)
		if err != nil {
			slog.Warn("Invalid column number in gosec finding", "finding_id", i, "col_value", issue.Col, "error", err)
			// Continue without column info - better than losing the entire finding
		}

		finding := Finding{
			ID:          fmt.Sprintf("gosec-%d", i),
			Scanner:     "gosec",
			Severity:    mapGosecSeverity(issue.Severity),
			Title:       issue.RuleID,
			Description: issue.What,
			Location: Location{
				File:   issue.File,
				Line:   line,
				Column: col,
			},
		}
		findings = append(findings, finding)
	}

	// Build summary
	summary := SummarizeFindings(findings)

	return &ScanResult{
		Scanner:  g.Name(),
		Findings: findings,
		Summary:  summary,
		Duration: duration,
		Status:   ScanStatusSuccess,
	}, nil
}

// mapGosecSeverity converts gosec severity to our Severity type.
func mapGosecSeverity(severity string) Severity {
	switch severity {
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

// isCommandNotFound checks if the error is due to command not found.
func isCommandNotFound(err error) bool {
	if err == nil {
		return false
	}

	// Check for exec.ExitError and use exit code
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// On Unix systems, exit status 127 typically means command not found
		if exitErr.ExitCode() == 127 {
			return true
		}
	}

	// Check for specific error messages
	errStr := strings.ToLower(err.Error())

	return strings.Contains(errStr, "executable file not found") ||
		strings.Contains(errStr, "command not found") ||
		strings.Contains(errStr, "no such file or directory") ||
		os.IsNotExist(err)
}

// validateGosecConfig validates the gosec configuration values.
func validateGosecConfig(config *GosecConfig) error {
	if config == nil {
		return nil
	}

	// Validate severity level
	if config.Severity != "" {
		validSeverities := map[string]bool{
			"low": true, "medium": true, "high": true,
		}
		if !validSeverities[strings.ToLower(config.Severity)] {
			return fmt.Errorf("invalid severity level: %s", config.Severity)
		}
	}

	// Validate confidence level
	if config.Confidence != "" {
		validConfidences := map[string]bool{
			"low": true, "medium": true, "high": true,
		}
		if !validConfidences[strings.ToLower(config.Confidence)] {
			return fmt.Errorf("invalid confidence level: %s", config.Confidence)
		}
	}

	// Validate exclude patterns for path traversal attempts
	for _, excl := range config.Exclude {
		// Check for obvious path traversal patterns
		if strings.Contains(excl, "..") {
			return fmt.Errorf("exclude pattern contains path traversal: %s", excl)
		}
		// Check for absolute paths (should be relative)
		if filepath.IsAbs(excl) {
			return fmt.Errorf("exclude pattern should be relative, not absolute: %s", excl)
		}
	}

	return nil
}
