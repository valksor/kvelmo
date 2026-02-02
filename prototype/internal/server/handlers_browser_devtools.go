package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/valksor/go-mehrhof/internal/browser"
)

var errNoTabs = errors.New("no tabs open")

// resolveTabID returns the requested tab ID, or defaults to the first open tab.
func (s *Server) resolveTabID(r *http.Request, ctrl browser.Controller, tabID string) (string, error) {
	if tabID != "" {
		return tabID, nil
	}

	tabs, err := ctrl.ListTabs(r.Context())
	if err != nil || len(tabs) == 0 {
		return "", errNoTabs
	}

	return tabs[0].ID, nil
}

// getBrowserController extracts the browser controller from the conductor,
// writing an error response and returning nil if unavailable.
func (s *Server) getBrowserController(w http.ResponseWriter, r *http.Request) browser.Controller {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return nil
	}

	ctrl := s.config.Conductor.GetBrowser(r.Context())
	if ctrl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser not configured")

		return nil
	}

	return ctrl
}

// defaultDuration returns a time.Duration from seconds, defaulting to 5s.
func defaultDuration(seconds int) time.Duration {
	if seconds <= 0 {
		return 5 * time.Second
	}

	return time.Duration(seconds) * time.Second
}

// handleBrowserNetwork monitors network requests for a duration.
func (s *Server) handleBrowserNetwork(w http.ResponseWriter, r *http.Request) {
	ctrl := s.getBrowserController(w, r)
	if ctrl == nil {
		return
	}

	var req browserNetworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = browserNetworkRequest{}
	}

	tabID, err := s.resolveTabID(r, ctrl, req.TabID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "no tabs open")

		return
	}

	// Configure body capture if requested
	if req.CaptureBody {
		opts := browser.NetworkMonitorOptions{
			CaptureBody: true,
			MaxBodySize: req.MaxBodySize,
		}
		if opts.MaxBodySize <= 0 {
			opts.MaxBodySize = 1024 * 1024 // 1MB default
		}
		ctrl.SetNetworkMonitorOptions(opts)
	}

	requests, err := ctrl.GetNetworkRequests(r.Context(), tabID, defaultDuration(req.Duration))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to monitor network: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"requests": requests,
		"count":    len(requests),
	})
}

// handleBrowserConsole monitors console logs for a duration.
func (s *Server) handleBrowserConsole(w http.ResponseWriter, r *http.Request) {
	ctrl := s.getBrowserController(w, r)
	if ctrl == nil {
		return
	}

	var req browserConsoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = browserConsoleRequest{}
	}

	tabID, err := s.resolveTabID(r, ctrl, req.TabID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "no tabs open")

		return
	}

	messages, err := ctrl.GetConsoleLogs(r.Context(), tabID, defaultDuration(req.Duration))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to monitor console: "+err.Error())

		return
	}

	// Filter by level if specified
	if req.Level != "" {
		filtered := make([]browser.ConsoleMessage, 0)
		for _, msg := range messages {
			if msg.Level == req.Level {
				filtered = append(filtered, msg)
			}
		}
		messages = filtered
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"messages": messages,
		"count":    len(messages),
	})
}

// handleBrowserWebSocket monitors WebSocket frames for a duration.
func (s *Server) handleBrowserWebSocket(w http.ResponseWriter, r *http.Request) {
	ctrl := s.getBrowserController(w, r)
	if ctrl == nil {
		return
	}

	var req browserWebSocketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = browserWebSocketRequest{}
	}

	tabID, err := s.resolveTabID(r, ctrl, req.TabID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "no tabs open")

		return
	}

	frames, err := ctrl.GetWebSocketFrames(r.Context(), tabID, defaultDuration(req.Duration))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to monitor websocket: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"frames":  frames,
		"count":   len(frames),
	})
}

// handleBrowserSource returns the full HTML source of the current page.
func (s *Server) handleBrowserSource(w http.ResponseWriter, r *http.Request) {
	ctrl := s.getBrowserController(w, r)
	if ctrl == nil {
		return
	}

	var req browserSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = browserSourceRequest{}
	}

	tabID, err := s.resolveTabID(r, ctrl, req.TabID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "no tabs open")

		return
	}

	source, err := ctrl.GetPageSource(r.Context(), tabID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to get page source: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"source":  source,
		"length":  len(source),
	})
}

// handleBrowserScripts returns all JavaScript sources loaded in the page.
func (s *Server) handleBrowserScripts(w http.ResponseWriter, r *http.Request) {
	ctrl := s.getBrowserController(w, r)
	if ctrl == nil {
		return
	}

	var req browserScriptsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = browserScriptsRequest{}
	}

	tabID, err := s.resolveTabID(r, ctrl, req.TabID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "no tabs open")

		return
	}

	scripts, err := ctrl.GetScriptSources(r.Context(), tabID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to get scripts: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"scripts": scripts,
		"count":   len(scripts),
	})
}

// handleBrowserStyles inspects CSS styles for an element.
func (s *Server) handleBrowserStyles(w http.ResponseWriter, r *http.Request) {
	ctrl := s.getBrowserController(w, r)
	if ctrl == nil {
		return
	}

	var req browserStylesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Selector == "" {
		s.writeError(w, http.StatusBadRequest, "selector is required")

		return
	}

	tabID, err := s.resolveTabID(r, ctrl, req.TabID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "no tabs open")

		return
	}

	// Default to computed styles if neither specified
	if !req.Computed && !req.Matched {
		req.Computed = true
	}

	result := map[string]any{
		"success":  true,
		"selector": req.Selector,
	}

	if req.Computed {
		computed, err := ctrl.GetComputedStyles(r.Context(), tabID, req.Selector)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to get computed styles: "+err.Error())

			return
		}
		result["computed"] = computed
	}

	if req.Matched {
		matched, err := ctrl.GetMatchedStyles(r.Context(), tabID, req.Selector)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to get matched styles: "+err.Error())

			return
		}
		result["matched"] = matched
	}

	s.writeJSON(w, http.StatusOK, result)
}

// handleBrowserCoverage measures JS and CSS code coverage.
func (s *Server) handleBrowserCoverage(w http.ResponseWriter, r *http.Request) {
	ctrl := s.getBrowserController(w, r)
	if ctrl == nil {
		return
	}

	var req browserCoverageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = browserCoverageRequest{}
	}

	tabID, err := s.resolveTabID(r, ctrl, req.TabID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "no tabs open")

		return
	}

	// Default to tracking both JS and CSS
	trackJS := req.TrackJS
	trackCSS := req.TrackCSS
	if !trackJS && !trackCSS {
		trackJS = true
		trackCSS = true
	}

	summary, jsEntries, cssEntries, err := ctrl.GetCoverage(
		r.Context(), tabID, defaultDuration(req.Duration), trackJS, trackCSS,
	)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to get coverage: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"summary":     summary,
		"js_entries":  jsEntries,
		"css_entries": cssEntries,
	})
}
