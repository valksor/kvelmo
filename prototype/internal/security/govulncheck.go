package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

// maxOutputSize and limitedBuffer are defined in gosec.go

// GovulncheckScanner wraps the govulncheck vulnerability scanner.
type GovulncheckScanner struct {
	enabled bool
	tm      *ToolManager
}

// NewGovulncheckScanner creates a new govulncheck scanner.
func NewGovulncheckScanner(enabled bool, tm *ToolManager) *GovulncheckScanner {
	return &GovulncheckScanner{
		enabled: enabled,
		tm:      tm,
	}
}

// Name returns the name of the scanner.
func (g *GovulncheckScanner) Name() string {
	return "govulncheck"
}

// IsEnabled returns whether the scanner is enabled.
func (g *GovulncheckScanner) IsEnabled() bool {
	return g.enabled
}

// Scan runs the govulncheck scanner on the given directory.
func (g *GovulncheckScanner) Scan(ctx context.Context, dir string) (*ScanResult, error) {
	start := time.Now()

	// Build command args
	args := []string{
		"-json",
		"-mode", "binary",
		dir,
	}

	// Get govulncheck binary path
	binaryName := "govulncheck"
	if g.tm != nil {
		spec := ToolSpec{
			Name:       "govulncheck",
			Repository: "golang.org/x/vuln",
			BinaryName: "govulncheck",
		}
		binaryPath, toolErr := g.tm.EnsureTool(ctx, spec)
		// Tool not available - skip it
		if toolErr == nil {
			binaryName = binaryPath
		} else {
			return skippedResult(g.Name(), time.Since(start)), nil
		}
	}

	// Run govulncheck
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

	// Check if govulncheck is installed
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
			Error:    errors.New("govulncheck output exceeded maximum size limit"),
			Status:   ScanStatusError,
		}, nil
	}

	// Parse JSON output (one JSON object per line)
	lines := strings.Split(stdout.String(), "\n")
	findings := make([]Finding, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse each line as a JSON object
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			slog.Warn("Skipping malformed govulncheck line", "line", line, "error", err)

			continue
		}

		// We're interested in "Vuln" findings
		if result["Name"] == nil {
			continue
		}

		// Extract vuln info
		name, ok := result["Name"].(string)
		if !ok || name == "" {
			continue
		}

		// Look for OSV entries
		if osvEntries, ok := result["OSV"].([]interface{}); ok && len(osvEntries) > 0 {
			for _, osvEntry := range osvEntries {
				if osvMap, ok := osvEntry.(map[string]interface{}); ok {
					if id, ok := osvMap["id"].(string); ok {
						// Create a finding for this vulnerability
						finding := Finding{
							ID:          "govulncheck-" + id,
							Scanner:     "govulncheck",
							Severity:    deriveSeverityFromID(id),
							Title:       "Vulnerability in dependency: " + name,
							Description: fmt.Sprintf("Package %s has a known vulnerability: %s", name, id),
							CVE:         extractCVE(id),
							Fix: &FixSuggestion{
								Description: fmt.Sprintf("Update package %s to a fixed version", name),
								Command:     fmt.Sprintf("go get %s@latest", name),
							},
							Metadata: map[string]string{
								"package": name,
								"osv_id":  id,
								"alias":   strings.Join(extractAliases(osvMap), ", "),
							},
						}

						// Extract affected details
						if affected, ok := result["Affected"].([]interface{}); ok && len(affected) > 0 {
							if affMap, ok := affected[0].(map[string]interface{}); ok {
								if pkgPath, ok := affMap["Package"].(map[string]interface{}); ok {
									if path, ok := pkgPath["path"].(string); ok {
										finding.Location = Location{
											File: path,
										}
									}
								}
							}
						}

						findings = append(findings, finding)
					}
				}
			}
		}
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

// deriveSeverityFromID derives a severity level from an OSV ID.
func deriveSeverityFromID(id string) Severity {
	// If it has a CVE, it's likely high or critical
	if strings.HasPrefix(id, "CVE-") {
		return SeverityHigh
	}
	// GHSA IDs can vary, default to medium
	if strings.HasPrefix(id, "GHSA-") {
		return SeverityMedium
	}

	return SeverityMedium
}

// extractCVE extracts CVE ID from an OSV ID if present.
func extractCVE(id string) string {
	if strings.HasPrefix(id, "CVE-") {
		return id
	}

	return ""
}

// extractAliases extracts alias IDs from an OSV entry.
func extractAliases(osvMap map[string]interface{}) []string {
	if aliases, ok := osvMap["aliases"].([]interface{}); ok {
		result := make([]string, 0, len(aliases))
		for _, alias := range aliases {
			if aliasStr, ok := alias.(string); ok {
				result = append(result, aliasStr)
			}
		}

		return result
	}

	return []string{}
}
