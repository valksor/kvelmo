package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-toolkit/eventbus"
)

// handleStandaloneReview handles POST /api/v1/workflow/review/standalone.
// Performs a standalone code review without requiring an active task.
func (s *Server) handleStandaloneReview(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req standaloneReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	// Default to uncommitted mode if not specified
	if req.Mode == "" {
		req.Mode = "uncommitted"
	}

	// Map request to conductor options
	diffOpts := conductor.StandaloneDiffOptions{
		Mode:       mapDiffMode(req.Mode),
		BaseBranch: req.BaseBranch,
		Range:      req.Range,
		Files:      req.Files,
		Context:    req.Context,
	}

	// Default checkpoint to true if not explicitly set and applying fixes
	createCheckpoint := req.CreateCheckpoint
	if req.ApplyFixes && !req.CreateCheckpoint {
		// If apply_fixes is true but create_checkpoint is false (default value),
		// check if the field was explicitly set. For safety, default to true.
		// The client should explicitly pass create_checkpoint: false to disable.
		createCheckpoint = true
	}

	reviewOpts := conductor.StandaloneReviewOptions{
		StandaloneDiffOptions: diffOpts,
		Agent:                 req.Agent,
		ApplyFixes:            req.ApplyFixes,
		CreateCheckpoint:      createCheckpoint,
	}

	// Check if SSE streaming is requested
	acceptHeader := r.Header.Get("Accept")
	if acceptHeader == "text/event-stream" {
		s.handleStandaloneReviewSSE(w, r, reviewOpts)

		return
	}

	// Synchronous review
	result, err := s.config.Conductor.ReviewStandalone(r.Context(), reviewOpts)
	if err != nil {
		s.writeJSON(w, http.StatusOK, standaloneReviewResponse{
			Success: false,
			Error:   err.Error(),
		})

		return
	}

	// Build response
	resp := standaloneReviewResponse{
		Success: true,
		Verdict: result.Verdict,
		Summary: result.Summary,
	}

	// Map issues
	for _, issue := range result.Issues {
		resp.Issues = append(resp.Issues, standaloneReviewIssue{
			Severity:    issue.Severity,
			Category:    issue.Category,
			File:        issue.File,
			Line:        issue.Line,
			Description: issue.Message, // ReviewIssue uses Message field
		})
	}

	// Map changes (if fixes were applied)
	for _, change := range result.Changes {
		resp.Changes = append(resp.Changes, standaloneFileChange{
			Path:      change.Path,
			Operation: string(change.Operation),
		})
	}

	// Map usage
	if result.Usage != nil {
		resp.Usage = &standaloneUsageInfo{
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
			CachedTokens: result.Usage.CachedTokens,
			CostUSD:      result.Usage.CostUSD,
		}
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// handleStandaloneReviewSSE handles streaming review via Server-Sent Events.
func (s *Server) handleStandaloneReviewSSE(w http.ResponseWriter, r *http.Request, opts conductor.StandaloneReviewOptions) {
	if _, ok := w.(http.Flusher); !ok {
		s.writeError(w, http.StatusBadRequest, "streaming not supported")

		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Subscribe to agent message events
	eventBus := s.config.Conductor.GetEventBus()
	if eventBus == nil {
		s.writeErrorSSE(w, "event bus not available")

		return
	}

	eventCh := make(chan eventbus.Event, 100)
	unsubscribeID := eventBus.SubscribeAll(func(e eventbus.Event) {
		if e.Type == events.TypeAgentMessage || e.Type == events.TypeProgress {
			select {
			case eventCh <- e:
			default:
				// Channel full, drop event
			}
		}
	})
	defer eventBus.Unsubscribe(unsubscribeID)
	defer close(eventCh)

	// Start review in background
	ctx := r.Context()
	resultCh := make(chan *conductor.StandaloneReviewResult, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := s.config.Conductor.ReviewStandalone(ctx, opts)
		if err != nil {
			errCh <- err
		} else {
			resultCh <- result
		}
	}()

	// Stream events to client
	for {
		select {
		case <-ctx.Done():
			return

		case e := <-eventCh:
			s.streamEvent(w, e)

		case err := <-errCh:
			sendSSE(w, "", fmt.Sprintf(`{"event":"error","error":"%s"}`, escapeJSON(err.Error())))
			sendSSE(w, "", `{"event":"done"}`)

			return

		case result := <-resultCh:
			// Send final result
			resp := standaloneReviewResponse{
				Success: true,
				Verdict: result.Verdict,
				Summary: result.Summary,
			}
			for _, issue := range result.Issues {
				resp.Issues = append(resp.Issues, standaloneReviewIssue{
					Severity:    issue.Severity,
					File:        issue.File,
					Line:        issue.Line,
					Description: issue.Message, // ReviewIssue uses Message field
				})
			}
			for _, change := range result.Changes {
				resp.Changes = append(resp.Changes, standaloneFileChange{
					Path:      change.Path,
					Operation: string(change.Operation),
				})
			}
			if result.Usage != nil {
				resp.Usage = &standaloneUsageInfo{
					InputTokens:  result.Usage.InputTokens,
					OutputTokens: result.Usage.OutputTokens,
					CostUSD:      result.Usage.CostUSD,
				}
			}
			jsonData, err := json.Marshal(resp)
			if err != nil {
				sendSSE(w, "", fmt.Sprintf(`{"event":"error","error":"%s"}`, escapeJSON(err.Error())))
				sendSSE(w, "", `{"event":"done"}`)

				return
			}
			sendSSE(w, "", fmt.Sprintf(`{"event":"result","data":%s}`, string(jsonData)))
			sendSSE(w, "", `{"event":"done"}`)

			return
		}
	}
}

// handleStandaloneSimplify handles POST /api/v1/workflow/simplify/standalone.
// Performs standalone code simplification without requiring an active task.
func (s *Server) handleStandaloneSimplify(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req standaloneSimplifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	// Default to uncommitted mode if not specified
	if req.Mode == "" {
		req.Mode = "uncommitted"
	}

	// Map request to conductor options
	diffOpts := conductor.StandaloneDiffOptions{
		Mode:       mapDiffMode(req.Mode),
		BaseBranch: req.BaseBranch,
		Range:      req.Range,
		Files:      req.Files,
		Context:    req.Context,
	}

	simplifyOpts := conductor.StandaloneSimplifyOptions{
		StandaloneDiffOptions: diffOpts,
		Agent:                 req.Agent,
		CreateCheckpoint:      req.CreateCheckpoint,
	}

	// Check if SSE streaming is requested
	acceptHeader := r.Header.Get("Accept")
	if acceptHeader == "text/event-stream" {
		s.handleStandaloneSimplifySSE(w, r, simplifyOpts)

		return
	}

	// Synchronous simplify
	result, err := s.config.Conductor.SimplifyStandalone(r.Context(), simplifyOpts)
	if err != nil {
		s.writeJSON(w, http.StatusOK, standaloneSimplifyResponse{
			Success: false,
			Error:   err.Error(),
		})

		return
	}

	// Build response
	resp := standaloneSimplifyResponse{
		Success: true,
		Summary: result.Summary,
	}

	// Map changes
	for _, change := range result.Changes {
		resp.Changes = append(resp.Changes, standaloneFileChange{
			Path:      change.Path,
			Operation: string(change.Operation),
		})
	}

	// Map usage
	if result.Usage != nil {
		resp.Usage = &standaloneUsageInfo{
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
			CachedTokens: result.Usage.CachedTokens,
			CostUSD:      result.Usage.CostUSD,
		}
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// handleStandaloneSimplifySSE handles streaming simplify via Server-Sent Events.
func (s *Server) handleStandaloneSimplifySSE(w http.ResponseWriter, r *http.Request, opts conductor.StandaloneSimplifyOptions) {
	if _, ok := w.(http.Flusher); !ok {
		s.writeError(w, http.StatusBadRequest, "streaming not supported")

		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Subscribe to agent message events
	eventBus := s.config.Conductor.GetEventBus()
	if eventBus == nil {
		s.writeErrorSSE(w, "event bus not available")

		return
	}

	eventCh := make(chan eventbus.Event, 100)
	unsubscribeID := eventBus.SubscribeAll(func(e eventbus.Event) {
		if e.Type == events.TypeAgentMessage || e.Type == events.TypeProgress {
			select {
			case eventCh <- e:
			default:
			}
		}
	})
	defer eventBus.Unsubscribe(unsubscribeID)
	defer close(eventCh)

	// Start simplify in background
	ctx := r.Context()
	resultCh := make(chan *conductor.StandaloneSimplifyResult, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := s.config.Conductor.SimplifyStandalone(ctx, opts)
		if err != nil {
			errCh <- err
		} else {
			resultCh <- result
		}
	}()

	// Stream events to client
	for {
		select {
		case <-ctx.Done():
			return

		case e := <-eventCh:
			s.streamEvent(w, e)

		case err := <-errCh:
			sendSSE(w, "", fmt.Sprintf(`{"event":"error","error":"%s"}`, escapeJSON(err.Error())))
			sendSSE(w, "", `{"event":"done"}`)

			return

		case result := <-resultCh:
			// Send final result
			resp := standaloneSimplifyResponse{
				Success: true,
				Summary: result.Summary,
			}
			for _, change := range result.Changes {
				resp.Changes = append(resp.Changes, standaloneFileChange{
					Path:      change.Path,
					Operation: string(change.Operation),
				})
			}
			if result.Usage != nil {
				resp.Usage = &standaloneUsageInfo{
					InputTokens:  result.Usage.InputTokens,
					OutputTokens: result.Usage.OutputTokens,
					CostUSD:      result.Usage.CostUSD,
				}
			}
			jsonData, err := json.Marshal(resp)
			if err != nil {
				sendSSE(w, "", fmt.Sprintf(`{"event":"error","error":"%s"}`, escapeJSON(err.Error())))
				sendSSE(w, "", `{"event":"done"}`)

				return
			}
			sendSSE(w, "", fmt.Sprintf(`{"event":"result","data":%s}`, string(jsonData)))
			sendSSE(w, "", `{"event":"done"}`)

			return
		}
	}
}

// streamEvent streams an event to the SSE client.
func (s *Server) streamEvent(w http.ResponseWriter, e eventbus.Event) {
	if eventData, ok := e.Data["event"].(map[string]any); ok {
		if eventType, ok := eventData["type"].(string); ok {
			switch eventType {
			case "content":
				if text, ok := eventData["text"].(string); ok {
					sendSSE(w, "", fmt.Sprintf(`{"event":"content","text":"%s"}`, escapeJSON(text)))
				}
			case "progress":
				if msg, ok := eventData["message"].(string); ok {
					sendSSE(w, "", fmt.Sprintf(`{"event":"progress","message":"%s"}`, escapeJSON(msg)))
				}
			}
		}
	}
}

// mapDiffMode maps string mode to conductor.StandaloneDiffMode.
func mapDiffMode(mode string) conductor.StandaloneDiffMode {
	switch mode {
	case "uncommitted":
		return conductor.DiffModeUncommitted
	case "branch":
		return conductor.DiffModeBranch
	case "range":
		return conductor.DiffModeRange
	case "files":
		return conductor.DiffModeFiles
	default:
		return conductor.DiffModeUncommitted
	}
}
