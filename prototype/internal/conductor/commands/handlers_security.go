package commands

import (
	"context"
	"errors"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/security"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "security-scan",
			Aliases:      []string{"scan"},
			Description:  "Run security scans on the codebase",
			Category:     "control",
			RequiresTask: false,
			MutatesState: false,
		},
		Handler: handleSecurityScan,
	})
}

// scanFindingData represents a security finding in scan results.
type scanFindingData struct {
	Scanner  string `json:"scanner"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	RuleID   string `json:"rule_id,omitempty"`
}

func handleSecurityScan(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	// Extract options
	targetDir := GetString(inv.Options, "dir")
	if targetDir == "" {
		targetDir = ws.CodeRoot()
	}

	failLevel := GetString(inv.Options, "fail_level")
	if failLevel == "" {
		failLevel = "critical"
	}

	// Validate fail level
	validLevels := map[string]bool{"critical": true, "high": true, "medium": true, "low": true, "any": true}
	if !validLevels[failLevel] {
		return nil, errors.New("invalid fail_level: must be one of critical, high, medium, low, any")
	}

	// Get requested scanners
	var scanners []string
	if raw, ok := inv.Options["scanners"]; ok {
		if s, ok := raw.([]string); ok {
			scanners = s
		}
		// Handle []any from JSON decoding
		if arr, ok := raw.([]any); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					scanners = append(scanners, s)
				}
			}
		}
	}

	// Validate scanners
	if len(scanners) > 0 {
		validScanners := map[string]bool{
			"gosec": true, "gitleaks": true, "govulncheck": true, "semgrep": true,
			"npm-audit": true, "eslint-security": true, "bandit": true, "pip-audit": true,
		}
		for _, scanner := range scanners {
			if !validScanners[scanner] {
				return nil, errors.New("invalid scanner '" + scanner + "'")
			}
		}
	}

	// Create scanner registry and register scanners based on project detection
	registry := security.NewScannerRegistry()
	projectInfo := security.DetectProject(targetDir)

	registry.Register("gitleaks", security.NewGitleaksScanner(true, &security.GitleaksConfig{}))
	registry.Register("semgrep", security.NewSemgrepScanner(true, &security.SemgrepConfig{}))

	if projectInfo.HasGoMod {
		registry.Register("gosec", security.NewGosecScanner(true, &security.GosecConfig{}))
		registry.Register("govulncheck", security.NewGovulncheckScanner(true))
	}

	if projectInfo.HasPackageJSON {
		registry.Register("npm-audit", security.NewNpmAuditScanner(true, &security.NpmAuditConfig{}))
		registry.Register("eslint-security", security.NewESLintScanner(true, &security.ESLintConfig{}))
	}

	if projectInfo.HasLanguage(security.LangPython) {
		registry.Register("bandit", security.NewBanditScanner(true, &security.BanditConfig{}))
		registry.Register("pip-audit", security.NewPipAuditScanner(true, &security.PipAuditConfig{}))
	}

	// Run scanners
	var results []*security.ScanResult
	if len(scanners) > 0 {
		results, _ = registry.RunEnabled(ctx, targetDir, scanners)
	} else {
		results, _ = registry.RunAll(ctx, targetDir)
	}

	// Convert results to findings
	var findings []scanFindingData
	for _, result := range results {
		for _, finding := range result.Findings {
			findings = append(findings, scanFindingData{
				Scanner:  result.Scanner,
				Severity: string(finding.Severity),
				Message:  finding.Title + ": " + finding.Description,
				File:     finding.Location.File,
				Line:     finding.Location.Line,
				Column:   finding.Location.Column,
				RuleID:   finding.ID,
			})
		}
	}

	// Check for blocking findings
	sevLevel := security.ParseSeverity(failLevel)
	blocking := security.GetBlockingFindings(results, sevLevel)
	passed := len(blocking) == 0

	return NewResult("Security scan complete").WithData(map[string]any{
		"findings":       findings,
		"total_count":    len(findings),
		"blocking_count": len(blocking),
		"passed":         passed,
	}), nil
}
