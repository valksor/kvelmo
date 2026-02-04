package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/memory"
	"github.com/valksor/go-mehrhof/internal/server/views"
)

// handleMemoryUI renders the memory page.
func (s *Server) handleMemoryUI(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil {
		s.writeError(w, http.StatusInternalServerError, "renderer not loaded")

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

	// Check if memory system is available
	enabled := false
	if s.config.Conductor != nil {
		mem := s.config.Conductor.GetMemory()
		enabled = mem != nil
	}

	data := views.MemoryData{
		PageData: pageData,
		Enabled:  enabled,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderMemory(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// handleMemorySearch searches the memory system.
func (s *Server) handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	mem := s.config.Conductor.GetMemory()
	if mem == nil {
		s.writeJSON(w, http.StatusOK, memorySearchResponse{
			Results: []memoryResult{},
			Count:   0,
		})

		return
	}

	// Parse query parameters
	query := r.URL.Query().Get("q")
	if query == "" {
		s.writeError(w, http.StatusBadRequest, "q parameter is required")

		return
	}

	limit := 5
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Parse document types
	var docTypes []memory.DocumentType
	if typesStr := r.URL.Query().Get("types"); typesStr != "" {
		for _, t := range strings.Split(typesStr, ",") {
			switch strings.TrimSpace(strings.ToLower(t)) {
			case "code_change", "code":
				docTypes = append(docTypes, memory.TypeCodeChange)
			case "specification", "spec":
				docTypes = append(docTypes, memory.TypeSpecification)
			case "session":
				docTypes = append(docTypes, memory.TypeSession)
			case "solution":
				docTypes = append(docTypes, memory.TypeSolution)
			case "decision":
				docTypes = append(docTypes, memory.TypeDecision)
			case "error":
				docTypes = append(docTypes, memory.TypeError)
			}
		}
	}

	// Search memory
	results, err := mem.Search(r.Context(), query, memory.SearchOptions{
		Limit:         limit,
		MinScore:      0.65,
		DocumentTypes: docTypes,
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "search failed: "+err.Error())

		return
	}

	var memResults []memoryResult
	for _, result := range results {
		memResults = append(memResults, memoryResult{
			TaskID:   result.Document.TaskID,
			Type:     string(result.Document.Type),
			Score:    float64(result.Score),
			Content:  result.Document.Content,
			Metadata: result.Document.Metadata,
		})
	}

	// Check if this is an HTMX request - return HTML partial
	if r.Header.Get("Hx-Request") == "true" {
		s.writeMemoryResultsHTML(w, memResults)

		return
	}

	s.writeJSON(w, http.StatusOK, memorySearchResponse{
		Results: memResults,
		Count:   len(memResults),
	})
}

// writeMemoryResultsHTML renders memory search results as HTML partial.
func (s *Server) writeMemoryResultsHTML(w http.ResponseWriter, results []memoryResult) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if len(results) == 0 {
		_, _ = w.Write([]byte(`
			<div class="card">
				<div class="text-center py-12">
					<div class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-gradient-to-br from-surface-100 to-surface-50 dark:from-surface-800 dark:to-surface-900 flex items-center justify-center">
						<svg class="w-8 h-8 text-surface-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
						</svg>
					</div>
					<p class="text-surface-600 dark:text-surface-400">No results found.</p>
					<p class="text-sm text-surface-500 dark:text-surface-500 mt-1">Try a different query or broader search terms.</p>
				</div>
			</div>
		`))

		return
	}

	html := fmt.Sprintf(`
		<div class="card">
			<div class="px-6 py-4 border-b border-surface-200 dark:border-surface-700 flex items-center justify-between">
				<span class="text-sm font-medium text-surface-700 dark:text-surface-300">Found %d results</span>
				<span class="text-xs text-surface-500 dark:text-surface-400">Sorted by relevance</span>
			</div>
			<div class="divide-y divide-surface-100 dark:divide-surface-700">
	`, len(results))

	var htmlSb169 strings.Builder
	for _, result := range results {
		// Determine type badge color
		badgeColor := "bg-surface-100 text-surface-600 dark:bg-surface-800 dark:text-surface-400"
		switch result.Type {
		case "code_change":
			badgeColor = "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
		case "specification":
			badgeColor = "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300"
		case "session":
			badgeColor = "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300"
		case "solution":
			badgeColor = "bg-success-100 text-success-700 dark:bg-success-900/30 dark:text-success-300"
		}

		// Determine score color
		scoreColor := "text-surface-600 dark:text-surface-400"
		if result.Score > 0.8 {
			scoreColor = "text-success-600 dark:text-success-400"
		} else if result.Score > 0.7 {
			scoreColor = "text-warning-600 dark:text-warning-400"
		}

		// Truncate content
		content := result.Content
		if len(content) > 400 {
			content = content[:397] + "..."
		}

		// Short task ID
		taskID := result.TaskID
		if len(taskID) > 8 {
			taskID = taskID[:8]
		}

		htmlSb169.WriteString(fmt.Sprintf(`
			<div class="p-6 hover:bg-surface-50 dark:hover:bg-surface-800/50 transition-smooth">
				<div class="flex items-start justify-between gap-4 mb-3">
					<div class="flex items-center gap-3">
						<span class="font-mono text-sm text-surface-600 dark:text-surface-400 bg-surface-100 dark:bg-surface-800 px-2 py-0.5 rounded">%s</span>
						<span class="px-2.5 py-1 rounded-full text-xs font-semibold %s">%s</span>
					</div>
					<div class="flex items-center gap-2 text-sm">
						<span class="text-surface-500 dark:text-surface-400">Score:</span>
						<span class="font-semibold %s">%s</span>
					</div>
				</div>
				<p class="text-surface-700 dark:text-surface-300 text-sm leading-relaxed">%s</p>
			</div>
		`, taskID, badgeColor, result.Type, scoreColor, views.FormatPercent(result.Score*100), content))
	}
	html += htmlSb169.String()

	html += `
			</div>
		</div>
	`

	_, _ = w.Write([]byte(html))
}

// handleMemoryIndex indexes a task to memory.
func (s *Server) handleMemoryIndex(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	mem := s.config.Conductor.GetMemory()
	if mem == nil {
		s.writeError(w, http.StatusServiceUnavailable, "memory system not available")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Parse request - handle both form and JSON submissions
	var req memoryIndexRequest
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid form data: "+err.Error())

			return
		}
		req.TaskID = r.FormValue("task_id")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

			return
		}
	}

	if req.TaskID == "" {
		s.writeError(w, http.StatusBadRequest, "task_id is required")

		return
	}

	// Verify task exists
	_, err := ws.LoadWork(req.TaskID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "task not found: "+err.Error())

		return
	}

	// Create indexer and index the task
	indexer := memory.NewIndexer(mem, ws, nil)
	if err := indexer.IndexTask(r.Context(), req.TaskID); err != nil {
		if r.Header.Get("Hx-Request") == "true" {
			s.writeIndexResultHTML(w, req.TaskID, false, "Failed to index task: "+err.Error())

			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to index task: "+err.Error())

		return
	}

	if r.Header.Get("Hx-Request") == "true" {
		s.writeIndexResultHTML(w, req.TaskID, true, "Task indexed successfully")

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "task indexed successfully",
		"task_id": req.TaskID,
	})
}

// writeIndexResultHTML renders index result as HTML feedback.
func (s *Server) writeIndexResultHTML(w http.ResponseWriter, _ string, success bool, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	colorClass := "text-success-600 dark:text-success-400 bg-success-50 dark:bg-success-900/20 border-success-200 dark:border-success-800"
	icon := `<svg class="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path></svg>`

	if !success {
		colorClass = "text-error-600 dark:text-error-400 bg-error-50 dark:bg-error-900/20 border-error-200 dark:border-error-800"
		icon = `<svg class="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path></svg>`
	}

	html := fmt.Sprintf(`
		<div class="p-3 rounded-lg border %s flex items-start gap-2 text-sm">
			%s
			<span>%s</span>
		</div>
	`, colorClass, icon, message)

	_, _ = w.Write([]byte(html))
}

// handleMemoryStats returns memory system statistics.
func (s *Server) handleMemoryStats(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		if r.Header.Get("Hx-Request") == "true" {
			s.writeMemoryStatsHTML(w, nil)

			return
		}
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	mem := s.config.Conductor.GetMemory()
	if mem == nil {
		resp := &memoryStatsResponse{
			TotalDocuments: 0,
			ByType:         map[string]int{},
			Enabled:        false,
		}
		if r.Header.Get("Hx-Request") == "true" {
			s.writeMemoryStatsHTML(w, resp)

			return
		}
		s.writeJSON(w, http.StatusOK, resp)

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		if r.Header.Get("Hx-Request") == "true" {
			s.writeMemoryStatsHTML(w, nil)

			return
		}
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Create indexer to get stats
	indexer := memory.NewIndexer(mem, ws, nil)
	stats, err := indexer.GetStats(r.Context())
	if err != nil {
		if r.Header.Get("Hx-Request") == "true" {
			s.writeMemoryStatsHTML(w, nil)

			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to get stats: "+err.Error())

		return
	}

	resp := &memoryStatsResponse{
		TotalDocuments: stats.TotalDocuments,
		ByType:         stats.ByType,
		Enabled:        true,
	}

	if r.Header.Get("Hx-Request") == "true" {
		s.writeMemoryStatsHTML(w, resp)

		return
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// writeMemoryStatsHTML renders memory stats as HTML partial.
func (s *Server) writeMemoryStatsHTML(w http.ResponseWriter, stats *memoryStatsResponse) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if stats == nil || !stats.Enabled {
		_, _ = w.Write([]byte(`
			<div class="text-center py-4">
				<p class="text-surface-500 dark:text-surface-400 text-sm">Memory system not available.</p>
				<p class="text-surface-400 dark:text-surface-500 text-xs mt-1">Configure memory in workspace settings.</p>
			</div>
		`))

		return
	}

	html := fmt.Sprintf(`
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<span class="text-sm text-surface-600 dark:text-surface-400">Total Documents</span>
				<span class="text-2xl font-bold text-surface-900 dark:text-surface-100">%s</span>
			</div>
	`, views.FormatNumber(stats.TotalDocuments))

	if len(stats.ByType) > 0 {
		html += `
			<div class="border-t border-surface-200 dark:border-surface-700 pt-4 space-y-3">
				<span class="text-sm font-medium text-surface-700 dark:text-surface-300">By Type</span>
				<div class="space-y-2">
		`

		var htmlSb420 strings.Builder
		for docType, count := range stats.ByType {
			htmlSb420.WriteString(fmt.Sprintf(`
				<div class="flex items-center justify-between">
					<span class="text-sm text-surface-600 dark:text-surface-400 capitalize">%s</span>
					<span class="text-sm font-medium text-surface-900 dark:text-surface-100">%s</span>
				</div>
			`, docType, views.FormatNumber(count)))
		}
		html += htmlSb420.String()

		html += `
				</div>
			</div>
		`
	}

	html += `</div>`

	_, _ = w.Write([]byte(html))
}
