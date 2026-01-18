package conductor

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/security"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// RunSecurityScan runs security scanners on the work directory and stores results.
func (c *Conductor) RunSecurityScan(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	// Load config to check if security is enabled
	cfg, err := c.workspace.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.Security == nil || !cfg.Security.Enabled {
		// Security not enabled, skip
		return nil
	}

	// Check if we should run on current phase
	taskState := c.activeTask.State
	shouldRun := false

	if cfg.Security.RunOn.Implementing && taskState == "implementing" {
		shouldRun = true
	} else if cfg.Security.RunOn.Reviewing && taskState == "reviewing" {
		shouldRun = true
	}

	if !shouldRun {
		return nil
	}

	// Get work directory
	workDir := c.workspace.WorkPath(c.activeTask.ID)

	// Create scanner registry
	registry := security.NewScannerRegistry()
	registerScannersFromConfig(registry, cfg.Security)

	// Run scanners
	results, err := registry.RunAll(ctx, workDir)
	if err != nil {
		c.logError(fmt.Errorf("security scan: %w", err))
		// Don't fail the workflow on scan errors
		return nil
	}

	// Store results in work unit
	wu := c.machine.WorkUnit()
	if wu != nil {
		wu.SecurityResults = convertToWorkflowSecurityResults(results)

		// Check for blocking findings
		if cfg.Security.FailOn.BlockFinish {
			blockLevel := security.ParseSeverity(cfg.Security.FailOn.Level)
			if security.ShouldBlock(results, blockLevel) {
				blocking := security.GetBlockingFindings(results, blockLevel)
				wu.SecurityResults.HasBlocking = true
				wu.SecurityResults.BlockingCount = len(blocking)

				// Log blocking findings
				c.logError(fmt.Errorf("security scan blocked: %d finding(s) at or above '%s' severity",
					len(blocking), blockLevel))
			}
		}
	}

	// Generate SARIF report if configured
	if cfg.Security.Output.Format == "sarif" {
		reportPath := cfg.Security.Output.File
		if reportPath == "" {
			reportPath = filepath.Join(workDir, "security-report.json")
		}
		if err := security.GenerateAndWriteSARIF(results, reportPath); err != nil {
			c.logError(fmt.Errorf("generate SARIF report: %w", err))
		}
	}

	return nil
}

// GetSecurityFindingsForReview returns formatted security findings for the review phase.
func (c *Conductor) GetSecurityFindingsForReview() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	wu := c.machine.WorkUnit()
	if wu == nil || wu.SecurityResults == nil {
		return ""
	}

	var findings []string

	for _, result := range wu.SecurityResults.Results {
		if len(result.Findings) == 0 {
			continue
		}

		findings = append(findings, fmt.Sprintf("## %s Scanner (%s)\n", result.Scanner, result.Duration))
		findings = append(findings, fmt.Sprintf("Found %d issue(s)\n", len(result.Findings)))

		for _, finding := range result.Findings {
			findings = append(findings, fmt.Sprintf("\n### %s\n", finding.Title))
			findings = append(findings, fmt.Sprintf("**Severity**: %s\n", finding.Severity))
			if finding.Location != nil {
				findings = append(findings, fmt.Sprintf("**Location**: %s:%d\n", finding.Location.File, finding.Location.Line))
			}
			findings = append(findings, fmt.Sprintf("**Description**: %s\n", finding.Description))
			if finding.CVE != "" {
				findings = append(findings, fmt.Sprintf("**CVE**: %s\n", finding.CVE))
			}
		}

		findings = append(findings, "")
	}

	if len(findings) == 0 {
		return ""
	}

	return strings.Join(findings, "\n")
}

// HasBlockingSecurityFindings checks if there are blocking security findings.
func (c *Conductor) HasBlockingSecurityFindings() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	wu := c.machine.WorkUnit()
	if wu == nil || wu.SecurityResults == nil {
		return false
	}

	return wu.SecurityResults.HasBlocking
}

// convertToWorkflowSecurityResults converts security.ScanResult to workflow.SecurityScanResults.
func convertToWorkflowSecurityResults(results []*security.ScanResult) *workflow.SecurityScanResults {
	summary := workflow.SecurityScanSummary{
		BySeverity: make(map[string]int),
	}

	convertedResults := make([]*workflow.SecurityScanResult, len(results))
	totalFindings := 0

	for i, result := range results {
		convertedFindings := make([]*workflow.SecurityFinding, len(result.Findings))

		for j, finding := range result.Findings {
			convertedFindings[j] = &workflow.SecurityFinding{
				ID:          finding.ID,
				Scanner:     finding.Scanner,
				Severity:    string(finding.Severity),
				Title:       finding.Title,
				Description: finding.Description,
			}

			if finding.Location.File != "" {
				convertedFindings[j].Location = &workflow.SecurityLocation{
					File:   finding.Location.File,
					Line:   finding.Location.Line,
					Column: finding.Location.Column,
				}
			}

			if finding.CVE != "" {
				convertedFindings[j].CVE = finding.CVE
			}
		}

		convertedResults[i] = &workflow.SecurityScanResult{
			Scanner:  result.Scanner,
			Findings: convertedFindings,
			Duration: result.Duration,
			Error:    result.Error,
		}

		// Update summary
		for severity, count := range result.Summary.BySeverity {
			summary.BySeverity[string(severity)] += count
		}

		totalFindings += result.Summary.Total
	}

	summary.Total = totalFindings

	return &workflow.SecurityScanResults{
		ScannedAt:   time.Now(),
		Results:     convertedResults,
		Summary:     summary,
		HasBlocking: false,
	}
}

// registerScannersFromConfig registers scanners based on workspace security config.
func registerScannersFromConfig(registry *security.ScannerRegistry, settings *storage.SecuritySettings) {
	tm := registry.GetToolManager()

	// Register SAST scanners (e.g., gosec)
	if settings.Scanners.SAST != nil && settings.Scanners.SAST.Enabled {
		if security.IsToolEnabled(settings.Scanners.SAST.Tools, "gosec") {
			registry.Register("gosec", security.NewGosecScanner(true, &security.GosecConfig{}, tm))
		}
	}

	// Register secret scanners (e.g., gitleaks)
	if settings.Scanners.Secrets != nil && settings.Scanners.Secrets.Enabled {
		if security.IsToolEnabled(settings.Scanners.Secrets.Tools, "gitleaks") {
			registry.Register("gitleaks", security.NewGitleaksScanner(true, &security.GitleaksConfig{}, tm))
		}
	}

	// Register dependency scanners (e.g., govulncheck)
	if settings.Scanners.Dependencies != nil && settings.Scanners.Dependencies.Enabled {
		if security.IsToolEnabled(settings.Scanners.Dependencies.Tools, "govulncheck") {
			registry.Register("govulncheck", security.NewGovulncheckScanner(true, tm))
		}
	}
}
