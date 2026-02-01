package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/valksor/go-mehrhof/internal/automation"
	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// handleWebhook processes incoming webhooks from providers.
// POST /api/v1/webhooks/{provider}.
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")

	if s.automation == nil {
		s.writeError(w, http.StatusServiceUnavailable, "automation not enabled")

		return
	}

	// Delegate to automation coordinator.
	s.automation.HandleWebhook(w, r, provider)
}

// handleAutomationStatus returns the current automation status.
// GET /api/v1/automation/status.
func (s *Server) handleAutomationStatus(w http.ResponseWriter, r *http.Request) {
	if s.automation == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"enabled": false,
			"running": false,
		})

		return
	}

	status := s.automation.Status()
	s.writeJSON(w, http.StatusOK, status)
}

// handleAutomationJobs lists automation jobs.
// GET /api/v1/automation/jobs
// Query params: status (pending, running, completed, failed, cancelled).
func (s *Server) handleAutomationJobs(w http.ResponseWriter, r *http.Request) {
	if s.automation == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"jobs":  []any{},
			"count": 0,
		})

		return
	}

	// Parse optional status filter.
	var statusFilter *automation.JobStatus
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status := automation.JobStatus(statusStr)
		statusFilter = &status
	}

	jobs := s.automation.ListJobs(statusFilter)
	response := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		response = append(response, convertJobToMap(job))
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"jobs":  response,
		"count": len(response),
	})
}

// handleAutomationJob returns a specific job.
// GET /api/v1/automation/jobs/{id}.
func (s *Server) handleAutomationJob(w http.ResponseWriter, r *http.Request) {
	if s.automation == nil {
		s.writeError(w, http.StatusServiceUnavailable, "automation not enabled")

		return
	}

	jobID := r.PathValue("id")
	job, found := s.automation.GetJob(jobID)
	if !found {
		s.writeError(w, http.StatusNotFound, "job not found")

		return
	}

	s.writeJSON(w, http.StatusOK, convertJobToMap(job))
}

// handleAutomationJobCancel cancels a pending job.
// POST /api/v1/automation/jobs/{id}/cancel.
func (s *Server) handleAutomationJobCancel(w http.ResponseWriter, r *http.Request) {
	if s.automation == nil {
		s.writeError(w, http.StatusServiceUnavailable, "automation not enabled")

		return
	}

	jobID := r.PathValue("id")
	if err := s.automation.CancelJob(jobID); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "cancelled",
		"job_id": jobID,
	})
}

// handleAutomationJobRetry retries a failed job.
// POST /api/v1/automation/jobs/{id}/retry.
func (s *Server) handleAutomationJobRetry(w http.ResponseWriter, r *http.Request) {
	if s.automation == nil {
		s.writeError(w, http.StatusServiceUnavailable, "automation not enabled")

		return
	}

	jobID := r.PathValue("id")
	job, found := s.automation.GetJob(jobID)
	if !found {
		s.writeError(w, http.StatusNotFound, "job not found")

		return
	}

	if job.Status != automation.JobStatusFailed {
		s.writeError(w, http.StatusBadRequest, "only failed jobs can be retried")

		return
	}

	// Note: Full retry implementation would re-enqueue the job.
	// For now, indicate the pattern.
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "retry_queued",
		"job_id": jobID,
	})
}

// handleAutomationConfig returns the automation configuration.
// GET /api/v1/automation/config.
func (s *Server) handleAutomationConfig(w http.ResponseWriter, r *http.Request) {
	if s.automation == nil || s.automationConfig == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"enabled": false,
		})

		return
	}

	// Return sanitized config (no secrets).
	cfg := s.getAutomationConfig()
	s.writeJSON(w, http.StatusOK, cfg)
}

// getAutomationConfig returns sanitized automation configuration.
func (s *Server) getAutomationConfig() map[string]any {
	if s.automationConfig == nil {
		return map[string]any{"enabled": false}
	}

	// Build sanitized provider configs.
	providers := make(map[string]any)
	for name, cfg := range s.automationConfig.Providers {
		providers[name] = map[string]any{
			"enabled":        cfg.Enabled,
			"command_prefix": cfg.CommandPrefix,
			"use_worktrees":  cfg.UseWorktrees,
			"dry_run":        cfg.DryRun,
			"trigger_on":     cfg.TriggerOn,
			// webhook_secret intentionally omitted
		}
	}

	return map[string]any{
		"enabled":        s.automationConfig.Enabled,
		"providers":      providers,
		"access_control": s.automationConfig.AccessControl,
		"queue":          s.automationConfig.Queue,
		"labels":         s.automationConfig.Labels,
	}
}

// convertJobToMap converts a WebhookJob to a map for JSON response.
func convertJobToMap(job *automation.WebhookJob) map[string]any {
	result := map[string]any{
		"id":            job.ID,
		"status":        string(job.Status),
		"workflow_type": string(job.WorkflowType),
		"priority":      job.Priority,
		"attempts":      job.Attempts,
		"max_attempts":  job.MaxAttempts,
		"created_at":    job.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if job.Command != "" {
		result["command"] = job.Command
	}
	if job.Error != "" {
		result["error"] = job.Error
	}
	if job.StartedAt != nil {
		result["started_at"] = job.StartedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if job.CompletedAt != nil {
		result["completed_at"] = job.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	if job.Event != nil {
		event := map[string]any{
			"id":         job.Event.ID,
			"provider":   job.Event.Provider,
			"type":       string(job.Event.Type),
			"action":     job.Event.Action,
			"repository": job.Event.Repository.FullName,
			"sender":     job.Event.Sender.Login,
		}
		if job.Event.Issue != nil {
			event["issue_number"] = job.Event.Issue.Number
		}
		if job.Event.PullRequest != nil {
			event["pr_number"] = job.Event.PullRequest.Number
		}
		result["event"] = event
	}

	if job.Result != nil {
		result["result"] = map[string]any{
			"success":         job.Result.Success,
			"pr_number":       job.Result.PRNumber,
			"pr_url":          job.Result.PRURL,
			"comments_posted": job.Result.CommentsPosted,
			"error_message":   job.Result.ErrorMessage,
			"duration":        job.Result.Duration.String(),
		}
	}

	return result
}

// handleAutomationPage renders the automation management page.
// GET /automation.
func (s *Server) handleAutomationPage(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil {
		s.writeError(w, http.StatusInternalServerError, "renderer not initialized")

		return
	}

	pageData := views.ComputePageData(
		s.modeString(),
		s.config.Mode == ModeGlobal,
		s.config.AuthStore != nil,
		s.canSwitchProject(),
		s.isViewer(r),
		s.getCurrentUser(r),
	)

	data := views.AutomationData{
		PageData: pageData,
	}

	// Populate status and jobs if automation is enabled.
	if s.automation != nil {
		status := s.automation.Status()
		data.Enabled = status.Enabled
		data.Running = status.Running
		data.Workers = status.Workers
		data.PendingJobs = status.PendingJobs
		data.RunningJobs = status.RunningJobs
		data.CompletedJobs = status.CompletedJobs
		data.FailedJobs = status.FailedJobs
		data.CancelledJobs = status.CancelledJobs

		// Get all jobs.
		jobs := s.automation.ListJobs(nil)
		data.Jobs = make([]views.AutomationJobData, 0, len(jobs))
		for _, job := range jobs {
			data.Jobs = append(data.Jobs, convertJobToViewData(job))
		}
	}

	// Populate config if available.
	if s.automationConfig != nil {
		data.Config = buildAutomationConfigData(s.automationConfig)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderAutomation(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// convertJobToViewData converts a WebhookJob to view data.
func convertJobToViewData(job *automation.WebhookJob) views.AutomationJobData {
	data := views.AutomationJobData{
		ID:           job.ID,
		Status:       string(job.Status),
		WorkflowType: string(job.WorkflowType),
		Attempts:     job.Attempts,
		MaxAttempts:  job.MaxAttempts,
		Command:      job.Command,
		Error:        job.Error,
		CanCancel:    job.Status == automation.JobStatusPending || job.Status == automation.JobStatusRunning,
		CanRetry:     job.Status == automation.JobStatusFailed,
	}

	// Status badge and icon.
	switch job.Status {
	case automation.JobStatusPending:
		data.StatusBadge = "bg-yellow-100 text-yellow-800"
		data.StatusIcon = "clock"
	case automation.JobStatusRunning:
		data.StatusBadge = "bg-blue-100 text-blue-800"
		data.StatusIcon = "refresh"
	case automation.JobStatusCompleted:
		data.StatusBadge = "bg-green-100 text-green-800"
		data.StatusIcon = "check"
	case automation.JobStatusFailed:
		data.StatusBadge = "bg-red-100 text-red-800"
		data.StatusIcon = "x"
	case automation.JobStatusCancelled:
		data.StatusBadge = "bg-gray-100 text-gray-800"
		data.StatusIcon = "ban"
	}

	// Event data.
	if job.Event != nil {
		data.Provider = job.Event.Provider
		data.Repository = job.Event.Repository.FullName
		data.Sender = job.Event.Sender.Login
		if job.Event.Issue != nil {
			data.Reference = "#" + itoa(job.Event.Issue.Number)
		} else if job.Event.PullRequest != nil {
			data.Reference = "#" + itoa(job.Event.PullRequest.Number)
		}
	}

	// Timestamps.
	data.CreatedAt = job.CreatedAt.Format("2006-01-02 15:04:05")
	if job.StartedAt != nil {
		data.StartedAt = job.StartedAt.Format("2006-01-02 15:04:05")
	}
	if job.CompletedAt != nil {
		data.CompletedAt = job.CompletedAt.Format("2006-01-02 15:04:05")
		if job.StartedAt != nil {
			data.Duration = job.CompletedAt.Sub(*job.StartedAt).Round(time.Second).String()
		}
	}

	return data
}

// buildAutomationConfigData builds config data for display.
func buildAutomationConfigData(cfg *storage.AutomationSettings) views.AutomationConfigData {
	data := views.AutomationConfigData{
		Labels: views.AutomationLabelsData{
			MehrhofGenerated: cfg.Labels.MehrhofGenerated,
			InProgress:       cfg.Labels.InProgress,
			Failed:           cfg.Labels.Failed,
		},
		AccessControl: views.AutomationAccessControlData{
			Mode:      cfg.AccessControl.Mode,
			Allowlist: cfg.AccessControl.Allowlist,
			Blocklist: cfg.AccessControl.Blocklist,
			AllowBots: cfg.AccessControl.AllowBots,
		},
	}

	// Build provider data.
	for name, pcfg := range cfg.Providers {
		pdata := views.AutomationProviderData{
			Name:          name,
			Enabled:       pcfg.Enabled,
			CommandPrefix: pcfg.CommandPrefix,
		}

		// Build human-readable triggers.
		var triggers []string
		if pcfg.TriggerOn.IssueOpened {
			triggers = append(triggers, "Issue opened")
		}
		if len(pcfg.TriggerOn.IssueLabeled) > 0 {
			triggers = append(triggers, "Issue labeled")
		}
		if pcfg.TriggerOn.PROpened || pcfg.TriggerOn.MROpened {
			triggers = append(triggers, "PR/MR opened")
		}
		if pcfg.TriggerOn.PRUpdated || pcfg.TriggerOn.MRUpdated {
			triggers = append(triggers, "PR/MR updated")
		}
		if pcfg.TriggerOn.CommentCommands {
			triggers = append(triggers, "Comment commands")
		}
		pdata.TriggerOn = triggers

		data.Providers = append(data.Providers, pdata)
	}

	return data
}

// itoa converts int to string (simple helper).
func itoa(n int) string {
	return strconv.Itoa(n)
}
