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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use defaults if body is empty
		req = scanRequest{}
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
		validScanners := map[string]bool{"gosec": true, "gitleaks": true, "govulncheck": true}
		for _, scanner := range req.Scanners {
			if !validScanners[scanner] {
				s.writeError(w, http.StatusBadRequest, "invalid scanner '"+scanner+"': must be one of gosec, gitleaks, govulncheck")

				return
			}
		}
	}

	// Determine scan directory
	targetDir := req.Dir
	if targetDir == "" {
		targetDir = ws.Root()
	}

	// Load workspace config for scanner settings
	cfg, err := ws.LoadConfig()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to load config: "+err.Error())

		return
	}

	// Create scanner registry
	registry := security.NewScannerRegistry()

	// Initialize tool manager
	var toolMgr *security.ToolManager
	if cfg.Security != nil && cfg.Security.Tools != nil {
		toolMgr, err = security.NewToolManager(cfg.Security.Tools.CacheDir, cfg.Security.Tools.AutoDownload)
	} else {
		toolMgr, err = security.NewToolManager("", true) // Default: auto-download enabled
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to initialize tool manager: "+err.Error())

		return
	}
	registry.SetToolManager(toolMgr)

	// Register default scanners
	tm := registry.GetToolManager()
	registry.Register("gosec", security.NewGosecScanner(true, &security.GosecConfig{}, tm))
	registry.Register("gitleaks", security.NewGitleaksScanner(true, &security.GitleaksConfig{}, tm))
	registry.Register("govulncheck", security.NewGovulncheckScanner(true, tm))

	// Run scanners
	var results []*security.ScanResult
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
