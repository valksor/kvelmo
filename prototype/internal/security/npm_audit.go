package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// NpmAuditScanner wraps the npm audit dependency vulnerability scanner.
type NpmAuditScanner struct {
	enabled bool
	config  *NpmAuditConfig
}

// NpmAuditConfig holds configuration for the npm audit scanner.
type NpmAuditConfig struct {
	// Level specifies the minimum severity level to report (critical, high, moderate, low)
	Level string `yaml:"level"`
	// Production only audit production dependencies
	Production bool `yaml:"production"`
	// IgnoreAdvisories specifies advisory IDs to ignore
	IgnoreAdvisories []string `yaml:"ignore_advisories"`
}

// NpmAuditOutput represents the JSON output from npm audit (npm v7+).
type NpmAuditOutput struct {
	AuditReportVersion int                         `json:"auditReportVersion"`
	Vulnerabilities    map[string]NpmVulnerability `json:"vulnerabilities"`
	Metadata           NpmAuditMetadata            `json:"metadata"`
}

// NpmVulnerability represents a vulnerability in npm audit output.
type NpmVulnerability struct {
	Name         string   `json:"name"`
	Severity     string   `json:"severity"`
	IsDirect     bool     `json:"isDirect"`
	Via          []any    `json:"via"` // Can be string or object
	Effects      []string `json:"effects"`
	Range        string   `json:"range"`
	Nodes        []string `json:"nodes"`
	FixAvailable any      `json:"fixAvailable"` // Can be bool or object
}

// NpmAuditVia represents advisory information when via is an object.
type NpmAuditVia struct {
	Source     int      `json:"source"`
	Name       string   `json:"name"`
	Dependency string   `json:"dependency"`
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	Severity   string   `json:"severity"`
	CWE        []string `json:"cwe"`
	CVSS       struct {
		Score        float64 `json:"score"`
		VectorString string  `json:"vectorString"`
	} `json:"cvss"`
	Range string `json:"range"`
}

// NpmAuditMetadata contains metadata about the audit.
type NpmAuditMetadata struct {
	Vulnerabilities struct {
		Info     int `json:"info"`
		Low      int `json:"low"`
		Moderate int `json:"moderate"`
		High     int `json:"high"`
		Critical int `json:"critical"`
		Total    int `json:"total"`
	} `json:"vulnerabilities"`
	Dependencies struct {
		Prod         int `json:"prod"`
		Dev          int `json:"dev"`
		Optional     int `json:"optional"`
		Peer         int `json:"peer"`
		PeerOptional int `json:"peerOptional"`
		Total        int `json:"total"`
	} `json:"dependencies"`
}

// NewNpmAuditScanner creates a new npm audit scanner.
func NewNpmAuditScanner(enabled bool, config *NpmAuditConfig) *NpmAuditScanner {
	if config == nil {
		config = &NpmAuditConfig{}
	}

	return &NpmAuditScanner{
		enabled: enabled,
		config:  config,
	}
}

// Name returns the name of the scanner.
func (n *NpmAuditScanner) Name() string {
	return "npm-audit"
}

// IsEnabled returns whether the scanner is enabled.
func (n *NpmAuditScanner) IsEnabled() bool {
	return n.enabled
}

// Scan runs npm audit on the given directory.
func (n *NpmAuditScanner) Scan(ctx context.Context, dir string) (*ScanResult, error) {
	start := time.Now()

	// Check if npm is installed
	if _, lookupErr := exec.LookPath("npm"); lookupErr != nil {
		//nolint:nilerr // Error is communicated via ScanResult.Error for partial scan support
		return &ScanResult{
			Scanner:  n.Name(),
			Status:   ScanStatusSkipped,
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: time.Since(start),
			Error:    errors.New("npm not installed"),
		}, nil
	}

	// Check if package-lock.json exists (required for npm audit)
	lockFile := filepath.Join(dir, "package-lock.json")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		return &ScanResult{
			Scanner:  n.Name(),
			Status:   ScanStatusSkipped,
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: time.Since(start),
			Error:    errors.New("package-lock.json not found. Run 'npm install' first"),
		}, nil
	}

	// Build command args
	args := []string{
		"audit",
		"--json",
	}

	if n.config.Level != "" {
		args = append(args, "--audit-level", n.config.Level)
	}

	if n.config.Production {
		args = append(args, "--omit=dev")
	}

	// Run npm audit
	cmd := exec.CommandContext(ctx, "npm", args...)
	cmd.Dir = dir

	var stdout, stderr limitedBuffer
	stdout.limit = maxOutputSize
	stderr.limit = maxOutputSize
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// npm audit returns non-zero exit code when vulnerabilities are found
	// This is expected behavior, not an error
	_ = cmd.Run()
	duration := time.Since(start)

	// Check if output exceeded size limit
	if stdout.Len() >= maxOutputSize {
		return &ScanResult{
			Scanner:  n.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    errors.New("npm audit output exceeded maximum size limit"),
			Status:   ScanStatusError,
		}, nil
	}

	// Parse JSON output
	var auditOutput NpmAuditOutput
	if parseErr := json.Unmarshal(stdout.Bytes(), &auditOutput); parseErr != nil {
		return &ScanResult{
			Scanner:  n.Name(),
			Findings: []Finding{},
			Summary:  Summary{Total: 0, BySeverity: make(map[Severity]int)},
			Duration: duration,
			Error:    fmt.Errorf("failed to parse npm audit output: %w", parseErr),
			Status:   ScanStatusError,
		}, nil
	}

	// Convert to Findings
	findings := make([]Finding, 0)
	findingIndex := 0

	for pkgName, vuln := range auditOutput.Vulnerabilities {
		// Skip if in ignore list
		if n.shouldIgnore(pkgName) {
			continue
		}

		// Parse via field to get advisory details
		var title, url string
		var cwes []string
		var advisoryID int

		for _, v := range vuln.Via {
			switch via := v.(type) {
			case map[string]any:
				if t, ok := via["title"].(string); ok {
					title = t
				}
				if u, ok := via["url"].(string); ok {
					url = u
				}
				if source, ok := via["source"].(float64); ok {
					advisoryID = int(source)
				}
				if cweList, ok := via["cwe"].([]any); ok {
					for _, c := range cweList {
						if cweStr, ok := c.(string); ok {
							cwes = append(cwes, cweStr)
						}
					}
				}
			case string:
				// via is just a package name reference
				if title == "" {
					title = "Vulnerable dependency: " + via
				}
			}
		}

		if title == "" {
			title = "Vulnerability in " + pkgName
		}

		// Build fix suggestion
		var fix *FixSuggestion
		switch fixVal := vuln.FixAvailable.(type) {
		case bool:
			if fixVal {
				fix = &FixSuggestion{
					Description: "A fix is available",
					Command:     "npm audit fix",
				}
			}
		case map[string]any:
			if name, ok := fixVal["name"].(string); ok {
				if version, ok := fixVal["version"].(string); ok {
					fix = &FixSuggestion{
						Description: fmt.Sprintf("Update %s to %s", name, version),
						Command:     fmt.Sprintf("npm install %s@%s", name, version),
					}
				}
			}
		}

		finding := Finding{
			ID:          fmt.Sprintf("npm-audit-%d", findingIndex),
			Scanner:     "npm-audit",
			Severity:    mapNpmSeverity(vuln.Severity),
			Title:       title,
			Description: fmt.Sprintf("Package %s has a %s severity vulnerability. Affected range: %s", pkgName, vuln.Severity, vuln.Range),
			Location: Location{
				File: "package-lock.json",
			},
			Fix:      fix,
			Metadata: make(map[string]string),
		}

		finding.Metadata["package"] = pkgName
		finding.Metadata["range"] = vuln.Range
		if url != "" {
			finding.Metadata["url"] = url
		}
		if advisoryID > 0 {
			finding.Metadata["advisory_id"] = strconv.Itoa(advisoryID)
		}
		if len(cwes) > 0 {
			finding.Metadata["cwe"] = strings.Join(cwes, ", ")
		}
		if vuln.IsDirect {
			finding.Metadata["is_direct"] = "true"
		} else {
			finding.Metadata["is_direct"] = "false"
		}

		findings = append(findings, finding)
		findingIndex++
	}

	// Build summary
	summary := SummarizeFindings(findings)

	return &ScanResult{
		Scanner:  n.Name(),
		Findings: findings,
		Summary:  summary,
		Duration: duration,
		Status:   ScanStatusSuccess,
	}, nil
}

// shouldIgnore checks if a package should be ignored based on config.
func (n *NpmAuditScanner) shouldIgnore(pkgName string) bool {
	for _, ignored := range n.config.IgnoreAdvisories {
		if ignored == pkgName {
			return true
		}
	}

	return false
}

// mapNpmSeverity converts npm severity to our Severity type.
func mapNpmSeverity(severity string) Severity {
	switch strings.ToLower(severity) {
	case "critical":
		return SeverityCritical
	case "high":
		return SeverityHigh
	case "moderate":
		return SeverityMedium
	case "low":
		return SeverityLow
	case "info":
		return SeverityInfo
	default:
		return SeverityInfo
	}
}
