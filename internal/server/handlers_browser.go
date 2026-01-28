package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/browser"
	"github.com/valksor/go-mehrhof/internal/server/views"
)

// handleBrowserUI renders the browser control panel page.
func (s *Server) handleBrowserUI(w http.ResponseWriter, r *http.Request) {
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

	data := views.BrowserData{
		PageData: pageData,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderBrowser(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// handleBrowserStatus returns the browser connection status.
func (s *Server) handleBrowserStatus(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ctrl := s.config.Conductor.GetBrowser(r.Context())
	if ctrl == nil {
		s.writeJSON(w, http.StatusOK, browserStatusResponse{
			Connected: false,
			Error:     "browser not configured",
		})

		return
	}

	tabs, err := ctrl.ListTabs(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusOK, browserStatusResponse{
			Connected: false,
			Error:     err.Error(),
		})

		return
	}

	var tabResponses []browserTabResponse
	for _, tab := range tabs {
		tabResponses = append(tabResponses, browserTabResponse{
			ID:    tab.ID,
			Title: tab.Title,
			URL:   tab.URL,
		})
	}

	s.writeJSON(w, http.StatusOK, browserStatusResponse{
		Connected: true,
		Host:      "localhost",
		Port:      ctrl.GetPort(),
		Tabs:      tabResponses,
	})
}

// handleBrowserTabs returns the list of browser tabs.
func (s *Server) handleBrowserTabs(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ctrl := s.config.Conductor.GetBrowser(r.Context())
	if ctrl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser not configured")

		return
	}

	tabs, err := ctrl.ListTabs(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list tabs: "+err.Error())

		return
	}

	var tabResponses []browserTabResponse
	for _, tab := range tabs {
		tabResponses = append(tabResponses, browserTabResponse{
			ID:    tab.ID,
			Title: tab.Title,
			URL:   tab.URL,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"tabs":  tabResponses,
		"count": len(tabResponses),
	})
}

// handleBrowserGoto opens a URL in a new tab.
func (s *Server) handleBrowserGoto(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ctrl := s.config.Conductor.GetBrowser(r.Context())
	if ctrl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser not configured")

		return
	}

	var req browserGotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.URL == "" {
		s.writeError(w, http.StatusBadRequest, "url is required")

		return
	}

	tab, err := ctrl.OpenTab(r.Context(), req.URL)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to open tab: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"tab": browserTabResponse{
			ID:    tab.ID,
			Title: tab.Title,
			URL:   tab.URL,
		},
	})
}

// handleBrowserNavigate navigates the current tab to a URL.
func (s *Server) handleBrowserNavigate(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ctrl := s.config.Conductor.GetBrowser(r.Context())
	if ctrl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser not configured")

		return
	}

	var req browserNavigateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.URL == "" {
		s.writeError(w, http.StatusBadRequest, "url is required")

		return
	}

	// Get tab ID - use first tab if not specified
	tabID := req.TabID
	if tabID == "" {
		tabs, err := ctrl.ListTabs(r.Context())
		if err != nil || len(tabs) == 0 {
			s.writeError(w, http.StatusBadRequest, "no tabs open")

			return
		}
		tabID = tabs[0].ID
	}

	if err := ctrl.Navigate(r.Context(), tabID, req.URL); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to navigate: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "navigated to " + req.URL,
	})
}

// handleBrowserScreenshot captures a screenshot.
func (s *Server) handleBrowserScreenshot(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ctrl := s.config.Conductor.GetBrowser(r.Context())
	if ctrl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser not configured")

		return
	}

	var req browserScreenshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use defaults if body is empty
		req = browserScreenshotRequest{Format: "png", Quality: 80}
	}

	// Set defaults
	if req.Format == "" {
		req.Format = "png"
	}
	if req.Quality == 0 {
		req.Quality = 80
	}

	// Get tab ID - use first tab if not specified
	tabID := req.TabID
	if tabID == "" {
		tabs, err := ctrl.ListTabs(r.Context())
		if err != nil || len(tabs) == 0 {
			s.writeError(w, http.StatusBadRequest, "no tabs open")

			return
		}
		tabID = tabs[0].ID
	}

	opts := browser.ScreenshotOptions{
		Format:   req.Format,
		Quality:  req.Quality,
		FullPage: req.FullPage,
	}

	data, err := ctrl.Screenshot(r.Context(), tabID, opts)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to capture screenshot: "+err.Error())

		return
	}

	// Return base64-encoded image
	s.writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"format":   req.Format,
		"data":     base64.StdEncoding.EncodeToString(data),
		"size":     len(data),
		"encoding": "base64",
	})
}

// handleBrowserClick clicks an element.
func (s *Server) handleBrowserClick(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ctrl := s.config.Conductor.GetBrowser(r.Context())
	if ctrl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser not configured")

		return
	}

	var req browserClickRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Selector == "" {
		s.writeError(w, http.StatusBadRequest, "selector is required")

		return
	}

	// Get tab ID - use first tab if not specified
	tabID := req.TabID
	if tabID == "" {
		tabs, err := ctrl.ListTabs(r.Context())
		if err != nil || len(tabs) == 0 {
			s.writeError(w, http.StatusBadRequest, "no tabs open")

			return
		}
		tabID = tabs[0].ID
	}

	if err := ctrl.Click(r.Context(), tabID, req.Selector); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to click: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"selector": req.Selector,
	})
}

// handleBrowserType types text into an element.
func (s *Server) handleBrowserType(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ctrl := s.config.Conductor.GetBrowser(r.Context())
	if ctrl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser not configured")

		return
	}

	var req browserTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Selector == "" {
		s.writeError(w, http.StatusBadRequest, "selector is required")

		return
	}

	// Get tab ID - use first tab if not specified
	tabID := req.TabID
	if tabID == "" {
		tabs, err := ctrl.ListTabs(r.Context())
		if err != nil || len(tabs) == 0 {
			s.writeError(w, http.StatusBadRequest, "no tabs open")

			return
		}
		tabID = tabs[0].ID
	}

	if err := ctrl.Type(r.Context(), tabID, req.Selector, req.Text, req.Clear); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to type: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"selector": req.Selector,
	})
}

// handleBrowserEval evaluates JavaScript.
func (s *Server) handleBrowserEval(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ctrl := s.config.Conductor.GetBrowser(r.Context())
	if ctrl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser not configured")

		return
	}

	var req browserEvalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Expression == "" {
		s.writeError(w, http.StatusBadRequest, "expression is required")

		return
	}

	// Get tab ID - use first tab if not specified
	tabID := req.TabID
	if tabID == "" {
		tabs, err := ctrl.ListTabs(r.Context())
		if err != nil || len(tabs) == 0 {
			s.writeError(w, http.StatusBadRequest, "no tabs open")

			return
		}
		tabID = tabs[0].ID
	}

	result, err := ctrl.Eval(r.Context(), tabID, req.Expression)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to evaluate: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"result":  result,
	})
}

// handleBrowserDOM queries DOM elements.
func (s *Server) handleBrowserDOM(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ctrl := s.config.Conductor.GetBrowser(r.Context())
	if ctrl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser not configured")

		return
	}

	var req browserDOMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.Selector == "" {
		s.writeError(w, http.StatusBadRequest, "selector is required")

		return
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 20
	}

	// Get tab ID - use first tab if not specified
	tabID := req.TabID
	if tabID == "" {
		tabs, err := ctrl.ListTabs(r.Context())
		if err != nil || len(tabs) == 0 {
			s.writeError(w, http.StatusBadRequest, "no tabs open")

			return
		}
		tabID = tabs[0].ID
	}

	if req.All {
		elems, err := ctrl.QuerySelectorAll(r.Context(), tabID, req.Selector)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to query: "+err.Error())

			return
		}

		var elements []browserDOMElement
		limit := req.Limit
		if limit > len(elems) {
			limit = len(elems)
		}
		for i := range limit {
			elem := browserDOMElement{
				TagName:     elems[i].TagName,
				TextContent: elems[i].TextContent,
				Visible:     elems[i].Visible,
			}
			if req.HTML {
				elem.OuterHTML = elems[i].OuterHTML
			}
			elements = append(elements, elem)
		}

		s.writeJSON(w, http.StatusOK, map[string]any{
			"success":  true,
			"elements": elements,
			"count":    len(elems),
			"showing":  len(elements),
		})

		return
	}

	elem, err := ctrl.QuerySelector(r.Context(), tabID, req.Selector)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query: "+err.Error())

		return
	}

	if elem == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"element": nil,
		})

		return
	}

	element := browserDOMElement{
		TagName:     elem.TagName,
		TextContent: elem.TextContent,
		Visible:     elem.Visible,
	}
	if req.HTML {
		element.OuterHTML = elem.OuterHTML
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"element": element,
	})
}

// handleBrowserReload reloads the current page.
func (s *Server) handleBrowserReload(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ctrl := s.config.Conductor.GetBrowser(r.Context())
	if ctrl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser not configured")

		return
	}

	var req browserReloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use defaults if body is empty
		req = browserReloadRequest{}
	}

	// Get tab ID - use first tab if not specified
	tabID := req.TabID
	if tabID == "" {
		tabs, err := ctrl.ListTabs(r.Context())
		if err != nil || len(tabs) == 0 {
			s.writeError(w, http.StatusBadRequest, "no tabs open")

			return
		}
		tabID = tabs[0].ID
	}

	if err := ctrl.Reload(r.Context(), tabID, req.Hard); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to reload: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "page reloaded",
	})
}

// handleBrowserClose closes a tab.
func (s *Server) handleBrowserClose(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ctrl := s.config.Conductor.GetBrowser(r.Context())
	if ctrl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser not configured")

		return
	}

	var req browserCloseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.TabID == "" {
		s.writeError(w, http.StatusBadRequest, "tab_id is required")

		return
	}

	if err := ctrl.CloseTab(r.Context(), req.TabID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to close tab: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "tab closed",
	})
}
