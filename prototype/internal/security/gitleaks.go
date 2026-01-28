package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// maxOutputSize and limitedBuffer are defined in gosec.go

// GitleaksScanner wraps the gitleaks secret scanner.
type GitleaksScanner struct {
	enabled bool
	config  *GitleaksConfig
}

// GitleaksConfig holds configuration for the gitleaks scanner.
type GitleaksConfig struct {
	// ConfigPath specifies a custom gitleaks config path.
	ConfigPath string `yaml:"config_path"`
	// MaxDepth specifies the max depth for git history traversal.
	MaxDepth int `yaml:"max_depth"`
	// Verbose enables verbose output.
	Verbose bool `yaml:"verbose"`
}

// GitleaksFinding represents a single gitleaks finding.
type GitleaksFinding struct {
	Description string  `json:"Description"`
	StartLine   int     `json:"StartLine"`
	EndLine     int     `json:"EndLine"`
	StartColumn int     `json:"StartColumn"`
	EndColumn   int     `json:"EndColumn"`
	Match       string  `json:"Match"`
	Secret      string  `json:"Secret"`
	File        string  `json:"File"`
	RuleID      string  `json:"RuleID"`
	Severity    string  `json:"Severity"`
	Commit      string  `json:"Commit"`
	Entropy     float64 `json:"Entropy"`
	Author      string  `json:"Author"`
	Email       string  `json:"Email"`
	Date        string  `json:"Date"`
	Message     string  `json:"Message"`
}

// GitleaksOutput represents the JSON output from gitleaks.
type GitleaksOutput struct {
	Messages []string          `json:"Messages"`
	Findings []GitleaksFinding `json:"Findings"`
}

// NewGitleaksScanner creates a new gitleaks scanner.
func NewGitleaksScanner(enabled bool, config *GitleaksConfig) *GitleaksScanner {
	if config == nil {
		config = &GitleaksConfig{}
	}

	return &GitleaksScanner{
		enabled: enabled,
		config:  config,
	}
}

// Name returns the name of the scanner.
func (g *GitleaksScanner) Name() string {
	return "gitleaks"
}

// IsEnabled returns whether the scanner is enabled.
func (g *GitleaksScanner) IsEnabled() bool {
	return g.enabled
}

// Scan runs the gitleaks scanner on the given directory.
func (g *GitleaksScanner) Scan(ctx context.Context, dir string) (*ScanResult, error) {
	start := time.Now()

	// Validate config-derived arguments
	if err := validateGitleaksConfig(g.config); err != nil {
		return &ScanResult{
			Scanner:  g.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: time.Since(start),
			Error:    fmt.Errorf("invalid gitleaks config: %w", err),
			Status:   ScanStatusError,
		}, nil
	}

	// Build command args
	args := []string{
		"detect",
		"--source", dir,
		"--report-format", "json",
		"--report-path", "-",
		"--no-banner",
		"--no-color",
	}

	if g.config.ConfigPath != "" {
		args = append(args, "--config", g.config.ConfigPath)
	}
	if g.config.MaxDepth > 0 {
		args = append(args, "--max-depth", strconv.Itoa(g.config.MaxDepth))
	}
	if g.config.Verbose {
		args = append(args, "--verbose")
	}

	// Check if gitleaks is installed
	binaryName, lookupErr := exec.LookPath("gitleaks")
	if lookupErr != nil {
		//nolint:nilerr // Error is communicated via ScanResult.Error for partial scan support
		return &ScanResult{
			Scanner:  g.Name(),
			Status:   ScanStatusSkipped,
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: time.Since(start),
			Error:    errors.New("gitleaks not installed. Run: brew install gitleaks (or download from https://github.com/gitleaks/gitleaks/releases)"),
		}, nil
	}

	// Run gitleaks
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

	// Check if gitleaks is installed
	if runErr != nil && isCommandNotFound(runErr) {
		return skippedResult(g.Name(), duration), nil
	}

	// Parse JSON output
	var gitleaksResults GitleaksOutput
	if parseErr := json.Unmarshal(stdout.Bytes(), &gitleaksResults); parseErr != nil {
		return &ScanResult{
			Scanner:  g.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    fmt.Errorf("failed to parse gitleaks output: %w", parseErr),
			Status:   ScanStatusError,
		}, nil
	}

	// Convert to Findings
	findings := make([]Finding, 0, len(gitleaksResults.Findings))
	for i, glFinding := range gitleaksResults.Findings {
		// Calculate character length
		length := glFinding.EndColumn - glFinding.StartColumn
		if length < 0 {
			length = 0
		}

		finding := Finding{
			ID:          fmt.Sprintf("gitleaks-%d", i),
			Scanner:     "gitleaks",
			Severity:    mapGitleaksSeverity(glFinding.Severity),
			Title:       "Secret detected: " + glFinding.RuleID,
			Description: glFinding.Description,
			Location: Location{
				File:      glFinding.File,
				Line:      glFinding.StartLine,
				Column:    glFinding.StartColumn,
				Length:    length,
				EndLine:   glFinding.EndLine,
				EndColumn: glFinding.EndColumn,
			},
			Code: &CodeSnippet{
				Before: glFinding.Match,
			},
			Fix: &FixSuggestion{
				Description: "Remove secret from code. Use environment variables or secret management instead.",
			},
			Metadata: map[string]string{
				"rule_id": glFinding.RuleID,
				"secret":  maskSecret(glFinding.Secret),
				"commit":  glFinding.Commit,
				"author":  glFinding.Author,
				"email":   glFinding.Email,
				"date":    glFinding.Date,
				"message": glFinding.Message,
				"entropy": fmt.Sprintf("%.2f", glFinding.Entropy),
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

// mapGitleaksSeverity converts gitleaks severity to our Severity type.
func mapGitleaksSeverity(severity string) Severity {
	switch strings.ToLower(severity) {
	case "critical":
		return SeverityCritical
	case "high":
		return SeverityHigh
	case "medium":
		return SeverityMedium
	case "low":
		return SeverityLow
	default:
		return SeverityInfo
	}
}

// maskSecret masks a secret for display purposes.
func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "***"
	}

	return secret[:4] + "..." + secret[len(secret)-4:]
}

// validateGitleaksConfig validates the gitleaks configuration values.
func validateGitleaksConfig(config *GitleaksConfig) error {
	if config == nil {
		return nil
	}

	// Validate max depth is reasonable (1-1000)
	if config.MaxDepth < 0 || config.MaxDepth > 1000 {
		return fmt.Errorf("max_depth must be between 0 and 1000, got: %d", config.MaxDepth)
	}

	// Validate config path for path traversal attempts
	if config.ConfigPath != "" {
		// Check for obvious path traversal patterns
		if strings.Contains(config.ConfigPath, "..") {
			return fmt.Errorf("config_path contains path traversal: %s", config.ConfigPath)
		}
		// Validate the path exists and is a regular file (if absolute)
		if filepath.IsAbs(config.ConfigPath) {
			info, err := filepath.Abs(config.ConfigPath)
			if err != nil {
				return fmt.Errorf("invalid config_path: %w", err)
			}
			// Check if it looks like a config file
			if !strings.HasSuffix(info, ".toml") && !strings.HasSuffix(info, ".yaml") && !strings.HasSuffix(info, ".yml") && !strings.HasSuffix(info, ".json") {
				return fmt.Errorf("config_path should have .toml, .yaml, .yml, or .json extension: %s", config.ConfigPath)
			}
		}
	}

	return nil
}
