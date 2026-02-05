package server

import (
	"encoding/json"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/security"
)

// handleSecurityScan runs security scans on the codebase.
func (s *Server) handleSecurityScan(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	var req scanRequest

	// Try to parse as JSON first
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			req = scanRequest{}
		}
	} else {
		// Parse form data for HTMX requests
		if err := r.ParseForm(); err == nil {
			scanners := r.Form["scanners"]
			// Map UI values to scanner names
			for _, scanner := range scanners {
				switch scanner {
				case "sast":
					req.Scanners = append(req.Scanners, "gosec")
				case "secrets":
					req.Scanners = append(req.Scanners, "gitleaks")
				case "vulns":
					req.Scanners = append(req.Scanners, "govulncheck")
				default:
					req.Scanners = append(req.Scanners, scanner)
				}
			}
		}
	}

	// Set defaults
	if req.FailLevel == "" {
		req.FailLevel = "critical"
	}
	if req.Format == "" {
		req.Format = "json"
	}

	// Validate fail level
	validLevels := map[string]bool{"critical": true, "high": true, "medium": true, "low": true, "any": true}
	if !validLevels[req.FailLevel] {
		s.writeError(w, http.StatusBadRequest, "invalid fail_level: must be one of critical, high, medium, low, any")

		return
	}

	// Validate scanners
	if len(req.Scanners) > 0 {
		validScanners := map[string]bool{
			"gosec":           true,
			"gitleaks":        true,
			"govulncheck":     true,
			"semgrep":         true,
			"npm-audit":       true,
			"eslint-security": true,
			"bandit":          true,
			"pip-audit":       true,
		}
		for _, scanner := range req.Scanners {
			if !validScanners[scanner] {
				s.writeError(w, http.StatusBadRequest, "invalid scanner '"+scanner+"': must be one of gosec, gitleaks, govulncheck, semgrep, npm-audit, eslint-security, bandit, pip-audit")

				return
			}
		}
	}

	// Determine scan directory
	targetDir := req.Dir
	if targetDir == "" {
		targetDir = ws.CodeRoot()
	}

	// Create scanner registry
	registry := security.NewScannerRegistry()

	// Register default scanners based on project detection
	projectInfo := security.DetectProject(targetDir)

	// Always register cross-language scanners
	registry.Register("gitleaks", security.NewGitleaksScanner(true, &security.GitleaksConfig{}))
	registry.Register("semgrep", security.NewSemgrepScanner(true, &security.SemgrepConfig{}))

	// Register Go scanners if Go project detected
	if projectInfo.HasGoMod {
		registry.Register("gosec", security.NewGosecScanner(true, &security.GosecConfig{}))
		registry.Register("govulncheck", security.NewGovulncheckScanner(true))
	}

	// Register JavaScript/TypeScript scanners if detected
	if projectInfo.HasPackageJSON {
		registry.Register("npm-audit", security.NewNpmAuditScanner(true, &security.NpmAuditConfig{}))
		registry.Register("eslint-security", security.NewESLintScanner(true, &security.ESLintConfig{}))
	}

	// Register Python scanners if detected
	if projectInfo.HasLanguage(security.LangPython) {
		registry.Register("bandit", security.NewBanditScanner(true, &security.BanditConfig{}))
		registry.Register("pip-audit", security.NewPipAuditScanner(true, &security.PipAuditConfig{}))
	}

	// Run scanners
	var results []*security.ScanResult
	var err error
	if len(req.Scanners) > 0 {
		results, err = registry.RunEnabled(r.Context(), targetDir, req.Scanners)
	} else {
		results, err = registry.RunAll(r.Context(), targetDir)
	}
	// Note: err is intentionally ignored here - some scanners may have failed
	// while others succeeded. Individual scanner errors are reflected in results.
	_ = err

	// Convert results to findings
	var findings []scanFinding
	for _, result := range results {
		for _, finding := range result.Findings {
			findings = append(findings, scanFinding{
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
	failLevel := security.ParseSeverity(req.FailLevel)
	blocking := security.GetBlockingFindings(results, failLevel)
	passed := len(blocking) == 0

	s.writeJSON(w, http.StatusOK, scanResponse{
		Findings:      findings,
		TotalCount:    len(findings),
		BlockingCount: len(blocking),
		Passed:        passed,
	})
}
