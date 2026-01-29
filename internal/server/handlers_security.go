package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/valksor/go-mehrhof/internal/security"
	"github.com/valksor/go-mehrhof/internal/server/views"
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
	isHTMX := r.Header.Get("Hx-Request") == "true"

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
		targetDir = ws.Root()
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

	resp := scanResponse{
		Findings:      findings,
		TotalCount:    len(findings),
		BlockingCount: len(blocking),
		Passed:        passed,
	}

	if isHTMX {
		s.writeScanResultsHTML(w, resp)

		return
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// writeScanResultsHTML renders security scan results as HTML partial.
func (s *Server) writeScanResultsHTML(w http.ResponseWriter, resp scanResponse) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Determine overall status
	statusColor := "success"
	statusIcon := `<svg class="w-8 h-8" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M2.166 4.999A11.954 11.954 0 0010 1.944 11.954 11.954 0 0017.834 5c.11.65.166 1.32.166 2.001 0 5.225-3.34 9.67-8 11.317C5.34 16.67 2 12.225 2 7c0-.682.057-1.35.166-2.001zm11.541 3.708a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path></svg>`
	statusMessage := "No security issues found"

	if resp.TotalCount > 0 {
		if resp.BlockingCount > 0 {
			statusColor = "error"
			statusIcon = `<svg class="w-8 h-8" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path></svg>`
			statusMessage = fmt.Sprintf("Found %d issues (%d blocking)", resp.TotalCount, resp.BlockingCount)
		} else {
			statusColor = "warning"
			statusIcon = `<svg class="w-8 h-8" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path></svg>`
			statusMessage = fmt.Sprintf("Found %d issues (none blocking)", resp.TotalCount)
		}
	}

	html := fmt.Sprintf(`
		<div class="card">
			<div class="px-6 py-4 border-b border-surface-200 dark:border-surface-700 flex items-center justify-between">
				<div class="flex items-center gap-3">
					<div class="text-%s-600 dark:text-%s-400">
						%s
					</div>
					<span class="font-semibold text-surface-900 dark:text-surface-100">%s</span>
				</div>
				<span class="px-3 py-1 rounded-full text-xs font-semibold bg-%s-100 dark:%s-900/30 text-%s-700 dark:text-%s-300">
					%s
				</span>
			</div>
	`, statusColor, statusColor, statusIcon, statusMessage,
		statusColor, "bg-"+statusColor, statusColor, statusColor,
		strings.ToUpper(string([]rune(statusColor)[0]))+statusColor[1:])

	if resp.TotalCount == 0 {
		html += `
			<div class="p-6 text-center text-surface-600 dark:text-surface-400">
				<p>Your codebase passed all security checks.</p>
			</div>
		</div>
		`
		_, _ = w.Write([]byte(html))

		return
	}

	// Group findings by severity
	severityCounts := make(map[string]int)
	for _, f := range resp.Findings {
		severityCounts[f.Severity]++
	}

	html += `<div class="p-6 space-y-4">`

	// Summary badges
	html += `<div class="flex flex-wrap gap-2 mb-4">`
	var htmlSb229 strings.Builder
	for severity, count := range severityCounts {
		badgeColor := "surface"
		switch strings.ToLower(severity) {
		case "critical":
			badgeColor = "error"
		case "high":
			badgeColor = "error"
		case "medium":
			badgeColor = "warning"
		case "low":
			badgeColor = "info"
		}
		htmlSb229.WriteString(fmt.Sprintf(`<span class="px-3 py-1 rounded-full text-xs font-semibold bg-%s-100 dark:bg-%s-900/30 text-%s-700 dark:text-%s-300">%s: %d</span>`,
			badgeColor, badgeColor, badgeColor, badgeColor, severity, count))
	}
	html += htmlSb229.String()
	html += `</div>`

	// Findings list (collapsed by default, show first 5)
	html += `<details class="group"><summary class="cursor-pointer text-sm font-medium text-brand-600 dark:text-brand-400 hover:underline">View Details</summary><div class="mt-4 space-y-3 max-h-80 overflow-y-auto">`

	var htmlSb249 strings.Builder
	for i, f := range resp.Findings {
		if i >= 20 {
			htmlSb249.WriteString(fmt.Sprintf(`<p class="text-sm text-surface-500 dark:text-surface-400 text-center py-2">...and %d more findings</p>`, len(resp.Findings)-20))

			break
		}

		severityColor := "surface"
		switch strings.ToLower(f.Severity) {
		case "critical", "high":
			severityColor = "error"
		case "medium":
			severityColor = "warning"
		case "low":
			severityColor = "info"
		}

		location := f.File
		if f.Line > 0 {
			location = fmt.Sprintf("%s:%d", f.File, f.Line)
		}

		htmlSb249.WriteString(fmt.Sprintf(`
			<div class="p-3 rounded-lg border border-surface-200 dark:border-surface-700 bg-surface-50 dark:bg-surface-900">
				<div class="flex items-start justify-between gap-2 mb-2">
					<div class="flex items-center gap-2">
						<span class="px-2 py-0.5 rounded text-xs font-semibold bg-%s-100 dark:bg-%s-900/30 text-%s-700 dark:text-%s-300">%s</span>
						<span class="text-xs font-mono text-surface-500 dark:text-surface-400">%s</span>
					</div>
					<span class="text-xs text-surface-400 dark:text-surface-500">%s</span>
				</div>
				<p class="text-sm text-surface-700 dark:text-surface-300">%s</p>
				<p class="text-xs text-surface-500 dark:text-surface-400 font-mono mt-1">%s</p>
			</div>
		`, severityColor, severityColor, severityColor, severityColor, f.Severity, f.Scanner, f.RuleID, views.TruncateString(f.Message, 200), location))
	}
	html += htmlSb249.String()

	html += `</div></details></div></div>`

	_, _ = w.Write([]byte(html))
}

// handleScanPage renders the security scan page.
func (s *Server) handleScanPage(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil {
		s.writeError(w, http.StatusInternalServerError, "renderer not loaded")

		return
	}

	pageData := views.ComputePageData(
		s.modeString(),
		s.config.Mode == ModeGlobal,
		s.config.AuthStore != nil,
		s.canSwitchProject(),
		s.getCurrentUser(r),
	)

	// Detect project for scanner recommendations
	var projectInfo *views.ProjectInfoData
	if s.config.Conductor != nil {
		if ws := s.config.Conductor.GetWorkspace(); ws != nil {
			info := security.DetectProject(ws.Root())
			projectInfo = &views.ProjectInfoData{
				HasGoMod:       info.HasGoMod,
				HasPackageJSON: info.HasPackageJSON,
			}
		}
	}

	data := views.ScanData{
		PageData:    pageData,
		Enabled:     s.config.Conductor != nil,
		ProjectInfo: projectInfo,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderScan(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}
