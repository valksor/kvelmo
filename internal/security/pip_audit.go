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

// PipAuditScanner wraps the pip-audit dependency vulnerability scanner.
type PipAuditScanner struct {
	enabled bool
	config  *PipAuditConfig
}

// PipAuditConfig holds configuration for the pip-audit scanner.
type PipAuditConfig struct {
	// RequirementsFile specifies the requirements file to audit (default: requirements.txt)
	RequirementsFile string `yaml:"requirements_file"`
	// Strict enables strict mode (fail on any vulnerability)
	Strict bool `yaml:"strict"`
	// IgnoreVulns specifies vulnerability IDs to ignore
	IgnoreVulns []string `yaml:"ignore_vulns"`
}

// PipAuditOutput represents the JSON output from pip-audit.
type PipAuditOutput struct {
	Dependencies []PipAuditDependency `json:"dependencies"`
}

// PipAuditDependency represents a dependency in pip-audit output.
type PipAuditDependency struct {
	Name    string         `json:"name"`
	Version string         `json:"version"`
	Vulns   []PipAuditVuln `json:"vulns"`
}

// PipAuditVuln represents a vulnerability in pip-audit output.
type PipAuditVuln struct {
	ID          string   `json:"id"`
	FixVersions []string `json:"fix_versions"`
	Aliases     []string `json:"aliases"`
	Description string   `json:"description"`
}

// NewPipAuditScanner creates a new pip-audit scanner.
func NewPipAuditScanner(enabled bool, config *PipAuditConfig) *PipAuditScanner {
	if config == nil {
		config = &PipAuditConfig{}
	}

	return &PipAuditScanner{
		enabled: enabled,
		config:  config,
	}
}

// Name returns the name of the scanner.
func (p *PipAuditScanner) Name() string {
	return "pip-audit"
}

// IsEnabled returns whether the scanner is enabled.
func (p *PipAuditScanner) IsEnabled() bool {
	return p.enabled
}

// Scan runs pip-audit on the given directory.
func (p *PipAuditScanner) Scan(ctx context.Context, dir string) (*ScanResult, error) {
	start := time.Now()

	// Check if pip-audit is installed
	if _, lookupErr := exec.LookPath("pip-audit"); lookupErr != nil {
		//nolint:nilerr // Error is communicated via ScanResult.Error for partial scan support
		return &ScanResult{
			Scanner:  p.Name(),
			Status:   ScanStatusSkipped,
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: time.Since(start),
			Error:    errors.New("pip-audit not installed. Run: pip install pip-audit"),
		}, nil
	}

	// Determine which requirements file to use
	requirementsFile := p.config.RequirementsFile
	if requirementsFile == "" {
		// Try to find a requirements file
		candidates := []string{
			"requirements.txt",
			"requirements-dev.txt",
			"requirements/base.txt",
			"requirements/prod.txt",
		}

		for _, candidate := range candidates {
			if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
				requirementsFile = candidate

				break
			}
		}

		// Check for pyproject.toml
		if requirementsFile == "" {
			if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
				// pip-audit can use pyproject.toml directly
				requirementsFile = "pyproject.toml"
			}
		}

		if requirementsFile == "" {
			return &ScanResult{
				Scanner:  p.Name(),
				Status:   ScanStatusSkipped,
				Findings: []Finding{},
				Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
				Duration: time.Since(start),
				Error:    errors.New("no requirements.txt or pyproject.toml found"),
			}, nil
		}
	}

	// Build command args
	args := []string{
		"--format", "json",
		"--progress-spinner=off",
	}

	// Determine input source
	if requirementsFile == "pyproject.toml" {
		// Let pip-audit scan the local project
		args = append(args, "--local")
	} else {
		// Use specific requirements file
		args = append(args, "-r", filepath.Join(dir, requirementsFile))
	}

	if p.config.Strict {
		args = append(args, "--strict")
	}

	// Add ignored vulnerabilities
	for _, vuln := range p.config.IgnoreVulns {
		args = append(args, "--ignore-vuln", vuln)
	}

	// Run pip-audit
	cmd := exec.CommandContext(ctx, "pip-audit", args...)
	cmd.Dir = dir

	var stdout, stderr limitedBuffer
	stdout.limit = maxOutputSize
	stderr.limit = maxOutputSize
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// pip-audit returns non-zero exit code when vulnerabilities are found
	_ = cmd.Run()
	duration := time.Since(start)

	// Check if output exceeded size limit
	if stdout.Len() >= maxOutputSize {
		return &ScanResult{
			Scanner:  p.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    errors.New("pip-audit output exceeded maximum size limit"),
			Status:   ScanStatusError,
		}, nil
	}

	// Handle empty output
	if stdout.Len() == 0 {
		return &ScanResult{
			Scanner:  p.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Status:   ScanStatusSuccess,
		}, nil
	}

	// Parse JSON output - pip-audit outputs an array directly
	var dependencies []PipAuditDependency
	if parseErr := json.Unmarshal(stdout.Bytes(), &dependencies); parseErr != nil {
		return &ScanResult{
			Scanner:  p.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    fmt.Errorf("failed to parse pip-audit output: %w (stderr: %s)", parseErr, stderr.String()),
			Status:   ScanStatusError,
		}, nil
	}

	// Convert to Findings
	findings := make([]Finding, 0)
	findingIndex := 0

	for _, dep := range dependencies {
		for _, vuln := range dep.Vulns {
			// Determine severity based on vulnerability ID prefix
			severity := mapPipAuditSeverity(vuln.ID)

			// Build description
			description := vuln.Description
			if description == "" {
				description = fmt.Sprintf("Package %s %s has a known vulnerability: %s", dep.Name, dep.Version, vuln.ID)
			}

			// Build fix suggestion
			var fix *FixSuggestion
			if len(vuln.FixVersions) > 0 {
				fixVersion := vuln.FixVersions[len(vuln.FixVersions)-1] // Latest fix version
				fix = &FixSuggestion{
					Description: fmt.Sprintf("Update %s to version %s or later", dep.Name, fixVersion),
					Command:     fmt.Sprintf("pip install %s>=%s", dep.Name, fixVersion),
				}
			}

			finding := Finding{
				ID:          fmt.Sprintf("pip-audit-%d", findingIndex),
				Scanner:     "pip-audit",
				Severity:    severity,
				Title:       fmt.Sprintf("%s in %s %s", vuln.ID, dep.Name, dep.Version),
				Description: description,
				Location: Location{
					File: requirementsFile,
				},
				Fix:      fix,
				Metadata: make(map[string]string),
			}

			finding.Metadata["package"] = dep.Name
			finding.Metadata["version"] = dep.Version
			finding.Metadata["vuln_id"] = vuln.ID

			// Extract CVE from aliases or ID
			cve := extractCVEFromID(vuln.ID)
			if cve == "" {
				for _, alias := range vuln.Aliases {
					if strings.HasPrefix(alias, "CVE-") {
						cve = alias

						break
					}
				}
			}
			if cve != "" {
				finding.CVE = cve
			}

			if len(vuln.Aliases) > 0 {
				finding.Metadata["aliases"] = strings.Join(vuln.Aliases, ", ")
			}

			if len(vuln.FixVersions) > 0 {
				finding.Metadata["fix_versions"] = strings.Join(vuln.FixVersions, ", ")
			}

			findings = append(findings, finding)
			findingIndex++
		}
	}

	// Build summary
	summary := SummarizeFindings(findings)

	return &ScanResult{
		Scanner:  p.Name(),
		Findings: findings,
		Summary:  summary,
		Duration: duration,
		Status:   ScanStatusSuccess,
	}, nil
}

// mapPipAuditSeverity estimates severity based on vulnerability ID patterns.
// pip-audit doesn't provide severity directly, so we infer from the ID.
func mapPipAuditSeverity(vulnID string) Severity {
	// PYSEC IDs don't include severity, but CVEs might be mapped
	// For now, default to High for any known vulnerability
	if strings.HasPrefix(vulnID, "CVE-") || strings.HasPrefix(vulnID, "PYSEC-") || strings.HasPrefix(vulnID, "GHSA-") {
		return SeverityHigh
	}

	return SeverityMedium
}

// extractCVEFromID extracts CVE ID if the vulnerability ID is a CVE.
func extractCVEFromID(id string) string {
	if strings.HasPrefix(id, "CVE-") {
		return id
	}

	return ""
}
