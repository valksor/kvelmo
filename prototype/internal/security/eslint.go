package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ESLintScanner wraps the ESLint security plugin scanner.
type ESLintScanner struct {
	enabled bool
	config  *ESLintConfig
}

// ESLintConfig holds configuration for the ESLint scanner.
type ESLintConfig struct {
	// ConfigPath specifies a custom ESLint config path
	ConfigPath string `yaml:"config_path"`
	// Extensions specifies file extensions to scan (default: .js,.jsx,.ts,.tsx)
	Extensions []string `yaml:"extensions"`
	// IgnorePatterns specifies patterns to ignore
	IgnorePatterns []string `yaml:"ignore_patterns"`
}

// ESLintOutput represents the JSON output from ESLint.
type ESLintOutput []ESLintFileResult

// ESLintFileResult represents the ESLint results for a single file.
type ESLintFileResult struct {
	FilePath            string          `json:"filePath"`
	Messages            []ESLintMessage `json:"messages"`
	ErrorCount          int             `json:"errorCount"`
	FatalErrorCount     int             `json:"fatalErrorCount"`
	WarningCount        int             `json:"warningCount"`
	FixableErrorCount   int             `json:"fixableErrorCount"`
	FixableWarningCount int             `json:"fixableWarningCount"`
	Source              string          `json:"source"`
}

// ESLintMessage represents a single ESLint finding.
type ESLintMessage struct {
	RuleID    string `json:"ruleId"`
	Severity  int    `json:"severity"` // 0=off, 1=warning, 2=error
	Message   string `json:"message"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine"`
	EndColumn int    `json:"endColumn"`
	NodeType  string `json:"nodeType"`
	Fix       *struct {
		Range []int  `json:"range"`
		Text  string `json:"text"`
	} `json:"fix,omitempty"`
}

// NewESLintScanner creates a new ESLint scanner.
func NewESLintScanner(enabled bool, config *ESLintConfig) *ESLintScanner {
	if config == nil {
		config = &ESLintConfig{}
	}
	if len(config.Extensions) == 0 {
		config.Extensions = []string{".js", ".jsx", ".ts", ".tsx"}
	}

	return &ESLintScanner{
		enabled: enabled,
		config:  config,
	}
}

// Name returns the name of the scanner.
func (e *ESLintScanner) Name() string {
	return "eslint-security"
}

// IsEnabled returns whether the scanner is enabled.
func (e *ESLintScanner) IsEnabled() bool {
	return e.enabled
}

// Scan runs ESLint with security plugin on the given directory.
func (e *ESLintScanner) Scan(ctx context.Context, dir string) (*ScanResult, error) {
	start := time.Now()

	// Check if package.json exists
	pkgJSON := filepath.Join(dir, "package.json")
	if _, err := os.Stat(pkgJSON); os.IsNotExist(err) {
		return &ScanResult{
			Scanner:  e.Name(),
			Status:   ScanStatusSkipped,
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: time.Since(start),
			Error:    errors.New("package.json not found - not a JavaScript/TypeScript project"),
		}, nil
	}

	// Try to use npx for ESLint (most reliable across different setups)
	eslintCmd := "npx"
	eslintArgs := []string{"eslint"}

	// Check if npx is available
	_, npxErr := exec.LookPath("npx")
	if npxErr != nil {
		// Fall back to direct eslint command
		_, eslintErr := exec.LookPath("eslint")
		if eslintErr != nil {
			//nolint:nilerr // Error is communicated via ScanResult.Error for partial scan support
			return &ScanResult{
				Scanner:  e.Name(),
				Status:   ScanStatusSkipped,
				Findings: []Finding{},
				Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
				Duration: time.Since(start),
				Error:    errors.New("eslint not installed. Run: npm install eslint eslint-plugin-security"),
			}, nil
		}
		eslintCmd = "eslint"
		eslintArgs = []string{}
	}

	// Build command args
	args := append(eslintArgs,
		"--format", "json",
		"--no-error-on-unmatched-pattern",
	)

	// Use custom config or default security config
	if e.config.ConfigPath != "" {
		args = append(args, "--config", e.config.ConfigPath)
	} else {
		// Use inline config for security rules
		args = append(args, "--plugin", "security")
		// Enable all security rules as warnings
		args = append(args,
			"--rule", "security/detect-buffer-noassert: warn",
			"--rule", "security/detect-child-process: warn",
			"--rule", "security/detect-disable-mustache-escape: warn",
			"--rule", "security/detect-eval-with-expression: warn",
			"--rule", "security/detect-new-buffer: warn",
			"--rule", "security/detect-no-csrf-before-method-override: warn",
			"--rule", "security/detect-non-literal-fs-filename: warn",
			"--rule", "security/detect-non-literal-regexp: warn",
			"--rule", "security/detect-non-literal-require: warn",
			"--rule", "security/detect-object-injection: warn",
			"--rule", "security/detect-possible-timing-attacks: warn",
			"--rule", "security/detect-pseudoRandomBytes: warn",
			"--rule", "security/detect-unsafe-regex: warn",
		)
	}

	// Add extensions
	if len(e.config.Extensions) > 0 {
		args = append(args, "--ext", strings.Join(e.config.Extensions, ","))
	}

	// Add ignore patterns
	for _, pattern := range e.config.IgnorePatterns {
		args = append(args, "--ignore-pattern", pattern)
	}

	// Add target directory
	args = append(args, dir)

	// Run ESLint
	cmd := exec.CommandContext(ctx, eslintCmd, args...)
	cmd.Dir = dir

	var stdout, stderr limitedBuffer
	stdout.limit = maxOutputSize
	stderr.limit = maxOutputSize
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// ESLint returns non-zero exit code when issues are found
	runErr := cmd.Run()
	duration := time.Since(start)

	// Check if command not found
	if runErr != nil && isCommandNotFound(runErr) {
		return &ScanResult{
			Scanner:  e.Name(),
			Status:   ScanStatusSkipped,
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    errors.New("eslint not installed. Run: npm install eslint eslint-plugin-security"),
		}, nil
	}

	// Check if output exceeded size limit
	if stdout.Len() >= maxOutputSize {
		return &ScanResult{
			Scanner:  e.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    errors.New("eslint output exceeded maximum size limit"),
			Status:   ScanStatusError,
		}, nil
	}

	// Handle empty output (no files to scan)
	if stdout.Len() == 0 {
		return &ScanResult{
			Scanner:  e.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Status:   ScanStatusSuccess,
		}, nil
	}

	// Parse JSON output
	var eslintOutput ESLintOutput
	if parseErr := json.Unmarshal(stdout.Bytes(), &eslintOutput); parseErr != nil {
		// Check if security plugin is not installed
		if strings.Contains(stderr.String(), "eslint-plugin-security") ||
			strings.Contains(stderr.String(), "plugin:security") {
			return &ScanResult{
				Scanner:  e.Name(),
				Status:   ScanStatusSkipped,
				Findings: []Finding{},
				Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
				Duration: duration,
				Error:    errors.New("eslint-plugin-security not installed. Run: npm install eslint-plugin-security"),
			}, nil
		}

		return &ScanResult{
			Scanner:  e.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    fmt.Errorf("failed to parse eslint output: %w (stderr: %s)", parseErr, stderr.String()),
			Status:   ScanStatusError,
		}, nil
	}

	// Convert to Findings
	findings := make([]Finding, 0)
	findingIndex := 0

	for _, fileResult := range eslintOutput {
		for _, msg := range fileResult.Messages {
			// Only include security-related rules
			if !isSecurityRule(msg.RuleID) {
				continue
			}

			finding := Finding{
				ID:          fmt.Sprintf("eslint-security-%d", findingIndex),
				Scanner:     "eslint-security",
				Severity:    mapESLintSeverity(msg.Severity),
				Title:       msg.RuleID,
				Description: msg.Message,
				Location: Location{
					File:      fileResult.FilePath,
					Line:      msg.Line,
					Column:    msg.Column,
					EndLine:   msg.EndLine,
					EndColumn: msg.EndColumn,
				},
				Metadata: make(map[string]string),
			}

			finding.Metadata["rule_id"] = msg.RuleID
			if msg.NodeType != "" {
				finding.Metadata["node_type"] = msg.NodeType
			}

			// Add fix suggestion if available
			if msg.Fix != nil {
				finding.Fix = &FixSuggestion{
					Description: "ESLint can automatically fix this issue",
					Command:     "npx eslint --fix " + fileResult.FilePath,
				}
			}

			findings = append(findings, finding)
			findingIndex++
		}
	}

	// Build summary
	summary := SummarizeFindings(findings)

	return &ScanResult{
		Scanner:  e.Name(),
		Findings: findings,
		Summary:  summary,
		Duration: duration,
		Status:   ScanStatusSuccess,
	}, nil
}

// isSecurityRule checks if a rule ID is a security-related rule.
func isSecurityRule(ruleID string) bool {
	// Include security plugin rules
	if strings.HasPrefix(ruleID, "security/") {
		return true
	}

	// Include known security-related core ESLint rules
	securityRules := map[string]bool{
		"no-eval":             true,
		"no-implied-eval":     true,
		"no-new-func":         true,
		"no-script-url":       true,
		"no-unsafe-innerhtml": true,
		"no-unsafe-negation":  true,
	}

	return securityRules[ruleID]
}

// mapESLintSeverity converts ESLint severity to our Severity type.
func mapESLintSeverity(severity int) Severity {
	switch severity {
	case 2:
		return SeverityHigh
	case 1:
		return SeverityMedium
	default:
		return SeverityInfo
	}
}
