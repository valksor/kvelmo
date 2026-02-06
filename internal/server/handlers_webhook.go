package server

import (
	"net/http"

	"github.com/valksor/go-mehrhof/internal/automation"
)

// DISABLED: automation temporarily unavailable (requires remote serve)
// Blank references prevent "unused" lint errors for disabled code.
var (
	_ = (*Server).handleWebhook
	_ = (*Server).handleAutomationStatus
	_ = (*Server).handleAutomationJobs
	_ = (*Server).handleAutomationJob
	_ = (*Server).handleAutomationJobCancel
	_ = (*Server).handleAutomationJobRetry
	_ = (*Server).handleAutomationConfig
	_ = (*Server).getAutomationConfig
	_ = convertJobToMap
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
//
//nolint:unparam // Required by http.HandlerFunc interface
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

	if err := s.automation.RetryJob(jobID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to retry job: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "retry_queued",
		"job_id": jobID,
	})
}

// handleAutomationConfig returns the automation configuration.
// GET /api/v1/automation/config.
//
//nolint:unparam // Required by http.HandlerFunc interface
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
