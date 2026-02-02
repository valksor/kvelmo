package automation

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/eventbus"
)

var (
	// ErrProviderNotSupported is returned for unsupported providers.
	ErrProviderNotSupported = errors.New("provider not supported")

	// ErrAutomationDisabled is returned when automation is disabled.
	ErrAutomationDisabled = errors.New("automation is disabled")

	// ErrInvalidSignature is returned when webhook signature validation fails.
	ErrInvalidSignature = errors.New("invalid webhook signature")

	// ErrAccessDenied is returned when access control denies the request.
	ErrAccessDenied = errors.New("access denied")
)

// WebhookParser parses provider-specific webhook payloads into normalized events.
type WebhookParser interface {
	// Parse parses a webhook request into a WebhookEvent.
	Parse(r *http.Request, body []byte) (*WebhookEvent, error)

	// ValidateSignature validates the webhook signature.
	ValidateSignature(r *http.Request, body []byte, secret string) error
}

// Automation coordinates webhook processing and job execution.
//
//nolint:containedctx // Long-running service requires stored context for graceful shutdown
type Automation struct {
	mu sync.RWMutex

	config   *storage.AutomationSettings
	queue    *JobQueue
	filter   *AccessFilter
	parsers  map[string]WebhookParser
	handler  JobHandler
	eventBus *eventbus.Bus

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Config holds configuration for the Automation coordinator.
type Config struct {
	Settings *storage.AutomationSettings
	EventBus *eventbus.Bus
	Handler  JobHandler // Job execution handler
}

// New creates a new Automation coordinator.
func New(cfg Config) *Automation {
	if cfg.Settings == nil {
		cfg.Settings = &storage.AutomationSettings{}
	}

	// Create queue configuration.
	queueCfg := QueueConfig{
		MaxWorkers: 1,
		JobTimeout: 30 * time.Minute,
		EventBus:   cfg.EventBus,
	}

	if cfg.Settings.Queue.MaxConcurrent > 0 {
		queueCfg.MaxWorkers = cfg.Settings.Queue.MaxConcurrent
	}

	if cfg.Settings.Queue.JobTimeout != "" {
		if d, err := time.ParseDuration(cfg.Settings.Queue.JobTimeout); err == nil {
			queueCfg.JobTimeout = d
		}
	}

	return &Automation{
		config:   cfg.Settings,
		queue:    NewJobQueue(queueCfg),
		filter:   NewAccessFilter(&cfg.Settings.AccessControl),
		parsers:  make(map[string]WebhookParser),
		handler:  cfg.Handler,
		eventBus: cfg.EventBus,
	}
}

// RegisterParser registers a webhook parser for a provider.
func (a *Automation) RegisterParser(provider string, parser WebhookParser) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.parsers[provider] = parser
}

// Start begins the automation coordinator.
func (a *Automation) Start(ctx context.Context) {
	a.mu.Lock()
	a.ctx, a.cancel = context.WithCancel(ctx)
	a.mu.Unlock()

	if a.handler == nil {
		slog.Warn("automation started without job handler")

		return
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.queue.Start(a.ctx, a.handler)
	}()

	slog.Info("automation started",
		"workers", a.queue.maxWorkers,
		"enabled", a.config.Enabled,
	)
}

// Stop gracefully shuts down the automation coordinator.
func (a *Automation) Stop(timeout time.Duration) error {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	a.mu.Unlock()

	// Wait for queue to stop.
	if err := a.queue.Stop(timeout); err != nil {
		return err
	}

	// Wait for coordinator goroutine.
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return errors.New("timeout waiting for automation to stop")
	}
}

// HandleWebhook processes an incoming webhook request.
func (a *Automation) HandleWebhook(w http.ResponseWriter, r *http.Request, provider string) {
	if !a.config.Enabled {
		a.writeError(w, http.StatusServiceUnavailable, ErrAutomationDisabled)

		return
	}

	// Check provider is enabled.
	providerCfg, ok := a.config.Providers[provider]
	if !ok || !providerCfg.Enabled {
		a.writeError(w, http.StatusNotFound, fmt.Errorf("provider %s not enabled", provider))

		return
	}

	// Get parser for provider.
	a.mu.RLock()
	parser, ok := a.parsers[provider]
	a.mu.RUnlock()
	if !ok {
		a.writeError(w, http.StatusNotImplemented, ErrProviderNotSupported)

		return
	}

	// Limit request body to 10MB to prevent OOM from oversized payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	// Read body for signature validation and parsing.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, fmt.Errorf("failed to read body: %w", err))

		return
	}

	// Validate signature.
	if providerCfg.WebhookSecret != "" {
		if err := parser.ValidateSignature(r, body, providerCfg.WebhookSecret); err != nil {
			slog.Warn("webhook signature validation failed",
				"provider", provider,
				"error", err,
			)
			a.writeError(w, http.StatusUnauthorized, ErrInvalidSignature)

			return
		}
	}

	// Parse webhook event.
	event, err := parser.Parse(r, body)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, fmt.Errorf("failed to parse webhook: %w", err))

		return
	}

	// Check access control.
	allowed, reason := a.filter.IsAllowed(&event.Sender, &event.Repository)
	if !allowed {
		slog.Info("webhook access denied",
			"provider", provider,
			"user", event.Sender.Login,
			"repo", event.Repository.FullName,
			"reason", reason,
		)
		a.writeError(w, http.StatusForbidden, fmt.Errorf("%w: %s", ErrAccessDenied, reason))

		return
	}

	// Check if event should be processed based on config.
	workflowType, shouldProcess := a.shouldProcess(event, &providerCfg)
	if !shouldProcess {
		// Acknowledge but don't process.
		slog.Debug("webhook event skipped",
			"provider", provider,
			"type", event.Type,
			"action", event.Action,
		)
		a.writeJSON(w, http.StatusOK, map[string]string{
			"status": "skipped",
			"reason": "event not configured for processing",
		})

		return
	}

	// Create job.
	job := &WebhookJob{
		Event:        event,
		WorkflowType: workflowType,
		MaxAttempts:  1,
	}

	if a.config.Queue.RetryAttempts > 0 {
		job.MaxAttempts = a.config.Queue.RetryAttempts + 1
	}

	// Check for priority labels.
	if len(a.config.Queue.PriorityLabels) > 0 {
		var labels []string
		if event.Issue != nil {
			labels = event.Issue.Labels
		} else if event.PullRequest != nil {
			labels = event.PullRequest.Labels
		}
		for _, label := range labels {
			for _, priorityLabel := range a.config.Queue.PriorityLabels {
				if strings.EqualFold(label, priorityLabel) {
					job.Priority = 10

					break
				}
			}
		}
	}

	// Enqueue job.
	if err := a.queue.Enqueue(job); err != nil {
		a.writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to enqueue job: %w", err))

		return
	}

	slog.Info("webhook processed",
		"provider", provider,
		"type", event.Type,
		"job_id", job.ID,
		"workflow", workflowType,
	)

	a.writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "accepted",
		"job_id": job.ID,
	})
}

// shouldProcess determines if an event should be processed based on configuration.
func (a *Automation) shouldProcess(event *WebhookEvent, cfg *storage.ProviderAutoConfig) (WorkflowType, bool) {
	trigger := cfg.TriggerOn

	switch event.Type {
	case EventTypeIssueOpened:
		if trigger.IssueOpened {
			return WorkflowTypeIssueFix, true
		}
	case EventTypeIssueLabeled:
		if event.Issue != nil && len(trigger.IssueLabeled) > 0 {
			for _, label := range event.Issue.Labels {
				for _, triggerLabel := range trigger.IssueLabeled {
					if strings.EqualFold(label, triggerLabel) {
						return WorkflowTypeIssueFix, true
					}
				}
			}
		}
	case EventTypePROpened:
		if trigger.PROpened || trigger.MROpened {
			// Skip if it's a mehrhof-generated PR.
			if a.isMehrhofGenerated(event) {
				return "", false
			}

			return WorkflowTypePRReview, true
		}
	case EventTypePRUpdated:
		if trigger.PRUpdated || trigger.MRUpdated {
			if a.isMehrhofGenerated(event) {
				return "", false
			}

			return WorkflowTypePRReview, true
		}
	case EventTypeIssueComment, EventTypePRComment:
		if trigger.CommentCommands {
			if event.Comment != nil && strings.Contains(event.Comment.Body, cfg.CommandPrefix) {
				return WorkflowTypeCommand, true
			}
		}
	case EventTypeIssueClosed, EventTypeIssueEdited, EventTypePRClosed, EventTypePRMerged, EventTypeUnknown:
		// These event types are not processed for automation.
	}

	return "", false
}

// isMehrhofGenerated checks if a PR/MR was generated by mehrhof.
func (a *Automation) isMehrhofGenerated(event *WebhookEvent) bool {
	if event.PullRequest == nil {
		return false
	}

	label := a.config.Labels.MehrhofGenerated
	if label == "" {
		label = "mehrhof-generated"
	}

	for _, l := range event.PullRequest.Labels {
		if strings.EqualFold(l, label) {
			return true
		}
	}

	return false
}

// Status returns the current automation status.
func (a *Automation) Status() QueueStatus {
	status := a.queue.Status()
	status.Enabled = a.config.Enabled

	return status
}

// GetJob returns a job by ID.
func (a *Automation) GetJob(id string) (*WebhookJob, bool) {
	return a.queue.GetJob(id)
}

// ListJobs returns all jobs, optionally filtered by status.
func (a *Automation) ListJobs(status *JobStatus) []*WebhookJob {
	return a.queue.ListJobs(status)
}

// CancelJob cancels a pending job.
func (a *Automation) CancelJob(id string) error {
	return a.queue.CancelJob(id)
}

// RetryJob resets a failed job and re-enqueues it for processing.
func (a *Automation) RetryJob(id string) error {
	return a.queue.RetryJob(id)
}

// writeJSON writes a JSON response.
func (a *Automation) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// writeError writes an error response.
func (a *Automation) writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if encErr := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); encErr != nil {
		slog.Error("failed to encode error response", "error", encErr)
	}
}

// ValidateGitHubSignature validates a GitHub webhook signature.
func ValidateGitHubSignature(r *http.Request, body []byte, secret string) error {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		return errors.New("missing X-Hub-Signature-256 header")
	}

	// Remove "sha256=" prefix.
	signature = strings.TrimPrefix(signature, "sha256=")

	// Compute expected signature.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return errors.New("signature mismatch")
	}

	return nil
}

// ValidateGitLabSignature validates a GitLab webhook signature.
func ValidateGitLabSignature(r *http.Request, secret string) error {
	token := r.Header.Get("X-Gitlab-Token")
	if token == "" {
		return errors.New("missing X-Gitlab-Token header")
	}

	if token != secret {
		return errors.New("token mismatch")
	}

	return nil
}
