package server

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/valksor/go-mehrhof/internal/links"
	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// handleLinksUI renders the links page.
func (s *Server) handleLinksUI(w http.ResponseWriter, r *http.Request) {
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

	// Check if links system is available
	enabled := false
	if s.config.Conductor != nil {
		ws := s.config.Conductor.GetWorkspace()
		if ws != nil {
			cfg, err := ws.LoadConfig()
			if err == nil && cfg.Links != nil {
				enabled = cfg.Links.Enabled
			}
		}
	}

	data := views.LinksData{
		PageData: pageData,
		Enabled:  enabled,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderLinks(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// handleListLinks returns all links in the system.
func (s *Server) handleListLinks(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	linkMgr := storage.GetLinkManager(r.Context(), ws)
	if linkMgr == nil {
		s.writeJSON(w, http.StatusOK, linksListResponse{
			Links: []linkData{},
			Count: 0,
		})

		return
	}

	// Get all entities with outgoing links
	allLinks := linkMgr.GetIndex()
	var allLinksData []linkData
	for _, forwardLinks := range allLinks.Forward {
		for _, link := range forwardLinks {
			allLinksData = append(allLinksData, linkDataFromLinks(link))
		}
	}

	s.writeJSON(w, http.StatusOK, linksListResponse{
		Links: allLinksData,
		Count: len(allLinksData),
	})
}

// handleGetEntityLinks returns links for a specific entity.
func (s *Server) handleGetEntityLinks(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Extract entity ID from URL path
	entityID := strings.TrimPrefix(r.URL.Path, "/api/v1/links/")
	if entityID == "" {
		s.writeError(w, http.StatusBadRequest, "entity ID is required")

		return
	}

	linkMgr := storage.GetLinkManager(r.Context(), ws)
	if linkMgr == nil {
		s.writeJSON(w, http.StatusOK, entityLinksResponse{
			EntityID: entityID,
			Outgoing: []linkData{},
			Incoming: []linkData{},
		})

		return
	}

	outgoing := linkMgr.GetOutgoing(entityID)
	incoming := linkMgr.GetIncoming(entityID)

	var outgoingData, incomingData []linkData
	for _, link := range outgoing {
		outgoingData = append(outgoingData, linkDataFromLinks(link))
	}
	for _, link := range incoming {
		incomingData = append(incomingData, linkDataFromLinks(link))
	}

	s.writeJSON(w, http.StatusOK, entityLinksResponse{
		EntityID: entityID,
		Outgoing: outgoingData,
		Incoming: incomingData,
	})
}

// handleSearchLinks searches for entities by name.
func (s *Server) handleSearchLinks(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		s.writeError(w, http.StatusBadRequest, "q parameter is required")

		return
	}

	linkMgr := storage.GetLinkManager(r.Context(), ws)
	if linkMgr == nil {
		s.writeJSON(w, http.StatusOK, linksSearchResponse{
			Query:   query,
			Results: []entityResult{},
			Count:   0,
		})

		return
	}

	// Search the name registry for matches
	names := linkMgr.GetNames()
	var results []entityResult

	// Search in all registries
	searchRegistry(names.Specs, query, "spec", &results)
	searchRegistry(names.Sessions, query, "session", &results)
	searchRegistry(names.Decisions, query, "decision", &results)
	searchRegistry(names.Tasks, query, "task", &results)
	searchRegistry(names.Notes, query, "note", &results)

	s.writeJSON(w, http.StatusOK, linksSearchResponse{
		Query:   query,
		Results: results,
		Count:   len(results),
	})
}

// searchRegistry searches a registry map for matching entities.
func searchRegistry(registry map[string]string, query, entityType string, results *[]entityResult) {
	queryLower := strings.ToLower(query)
	for name, entityID := range registry {
		if strings.Contains(strings.ToLower(name), queryLower) {
			// Parse entity ID to get task and ID
			typ, taskID, id := links.ParseEntityID(entityID)

			*results = append(*results, entityResult{
				EntityID: entityID,
				Type:     entityType,
				Name:     name,
				TaskID:   taskID,
				ID:       id,
				FullType: string(typ),
			})
		}
	}
}

// handleLinksStats returns statistics about the link graph.
func (s *Server) handleLinksStats(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		if r.Header.Get("Hx-Request") == "true" {
			s.writeLinksStatsHTML(w, nil)

			return
		}
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		if r.Header.Get("Hx-Request") == "true" {
			s.writeLinksStatsHTML(w, nil)

			return
		}
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	linkMgr := storage.GetLinkManager(r.Context(), ws)
	if linkMgr == nil {
		resp := &linksStatsResponse{
			TotalLinks:     0,
			TotalSources:   0,
			TotalTargets:   0,
			OrphanEntities: 0,
			MostLinked:     []entityResult{},
			Enabled:        false,
		}
		if r.Header.Get("Hx-Request") == "true" {
			s.writeLinksStatsHTML(w, resp)

			return
		}
		s.writeJSON(w, http.StatusOK, resp)

		return
	}

	stats := linkMgr.GetStats()
	if stats == nil {
		if r.Header.Get("Hx-Request") == "true" {
			s.writeLinksStatsHTML(w, nil)

			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to get stats")

		return
	}

	// Get most linked entities
	linkIndex := linkMgr.GetIndex()
	var mostLinked []entityResult
	for source, forwardLinks := range linkIndex.Forward {
		typ, taskID, id := links.ParseEntityID(source)
		totalLinks := len(forwardLinks) + len(linkIndex.Backward[source])
		mostLinked = append(mostLinked, entityResult{
			EntityID:   source,
			Type:       string(typ),
			TaskID:     taskID,
			ID:         id,
			TotalLinks: totalLinks,
		})
	}

	// Sort by total links (descending)
	slices.SortFunc(mostLinked, func(a, b entityResult) int {
		if a.TotalLinks > b.TotalLinks {
			return -1
		}
		if a.TotalLinks < b.TotalLinks {
			return 1
		}

		return 0
	})

	// Keep top 10
	if len(mostLinked) > 10 {
		mostLinked = mostLinked[:10]
	}

	resp := &linksStatsResponse{
		TotalLinks:     stats.TotalLinks,
		TotalSources:   stats.TotalSources,
		TotalTargets:   stats.TotalTargets,
		OrphanEntities: stats.OrphanEntities,
		MostLinked:     mostLinked,
		Enabled:        true,
	}

	if r.Header.Get("Hx-Request") == "true" {
		s.writeLinksStatsHTML(w, resp)

		return
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// handleRebuildLinks rebuilds the link index from workspace content.
func (s *Server) handleRebuildLinks(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	linkMgr := storage.GetLinkManager(r.Context(), ws)
	if linkMgr == nil {
		s.writeError(w, http.StatusServiceUnavailable, "links not available")

		return
	}

	if err := linkMgr.Rebuild(); err != nil {
		if r.Header.Get("Hx-Request") == "true" {
			s.writeLinksRebuildResultHTML(w, false, "Failed to rebuild index: "+err.Error())

			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to rebuild index: "+err.Error())

		return
	}

	stats := linkMgr.GetStats()
	if r.Header.Get("Hx-Request") == "true" {
		s.writeLinksRebuildResultHTML(w, true, fmt.Sprintf("Index rebuilt successfully: %d links, %d entities", stats.TotalLinks, stats.TotalSources))

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"message":       "index rebuilt successfully",
		"total_links":   stats.TotalLinks,
		"total_sources": stats.TotalSources,
		"total_targets": stats.TotalTargets,
	})
}

// writeLinksStatsHTML renders links stats as HTML partial.
func (s *Server) writeLinksStatsHTML(w http.ResponseWriter, stats *linksStatsResponse) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if stats == nil || !stats.Enabled {
		_, _ = w.Write([]byte(`
			<div class="text-center py-4">
				<p class="text-surface-500 dark:text-surface-400 text-sm">Links system not available.</p>
				<p class="text-surface-400 dark:text-surface-500 text-xs mt-1">Enable links in workspace settings.</p>
			</div>
		`))

		return
	}

	html := fmt.Sprintf(`
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<span class="text-sm text-surface-600 dark:text-surface-400">Total Links</span>
				<span class="text-2xl font-bold text-surface-900 dark:text-surface-100">%s</span>
			</div>
			<div class="flex items-center justify-between">
				<span class="text-sm text-surface-600 dark:text-surface-400">Total Sources</span>
				<span class="text-lg font-semibold text-surface-900 dark:text-surface-100">%s</span>
			</div>
			<div class="flex items-center justify-between">
				<span class="text-sm text-surface-600 dark:text-surface-400">Total Targets</span>
				<span class="text-lg font-semibold text-surface-900 dark:text-surface-100">%s</span>
			</div>
	`, views.FormatNumber(stats.TotalLinks), views.FormatNumber(stats.TotalSources), views.FormatNumber(stats.TotalTargets))

	if len(stats.MostLinked) > 0 {
		html += `
			<div class="border-t border-surface-200 dark:border-surface-700 pt-4 space-y-3">
				<span class="text-sm font-medium text-surface-700 dark:text-surface-300">Most Linked Entities</span>
				<div class="space-y-2">
		`

		var htmlBuilder strings.Builder
		for i, entity := range stats.MostLinked {
			shortID := entity.ID
			if len(shortID) > 20 {
				shortID = shortID[:20] + "..."
			}

			htmlBuilder.WriteString(fmt.Sprintf(`
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2">
						<span class="text-xs text-surface-500">%d.</span>
						<span class="text-sm text-surface-700 dark:text-surface-300 font-mono">%s</span>
					</div>
					<span class="text-sm font-medium text-surface-900 dark:text-surface-100">%d</span>
				</div>
		`, i+1, shortID, entity.TotalLinks))
		}
		html += htmlBuilder.String()

		html += `
				</div>
			</div>
		`
	}

	html += `</div>`

	_, _ = w.Write([]byte(html))
}

// writeLinksRebuildResultHTML renders rebuild result as HTML feedback.
func (s *Server) writeLinksRebuildResultHTML(w http.ResponseWriter, success bool, message string) {
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

// Response types for API.

type linksListResponse struct {
	Links []linkData `json:"links"`
	Count int        `json:"count"`
}

type entityLinksResponse struct {
	EntityID string     `json:"entity_id"`
	Outgoing []linkData `json:"outgoing"`
	Incoming []linkData `json:"incoming"`
}

type linksSearchResponse struct {
	Query   string         `json:"query"`
	Results []entityResult `json:"results"`
	Count   int            `json:"count"`
}

type linksStatsResponse struct {
	TotalLinks     int            `json:"total_links"`
	TotalSources   int            `json:"total_sources"`
	TotalTargets   int            `json:"total_targets"`
	OrphanEntities int            `json:"orphan_entities"`
	MostLinked     []entityResult `json:"most_linked"`
	Enabled        bool           `json:"enabled"`
}

type linkData struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Context   string `json:"context"`
	CreatedAt string `json:"created_at"`
}

type entityResult struct {
	EntityID   string `json:"entity_id"`
	Type       string `json:"type"`
	Name       string `json:"name,omitempty"`
	TaskID     string `json:"task_id,omitempty"`
	ID         string `json:"id,omitempty"`
	FullType   string `json:"full_type,omitempty"`
	TotalLinks int    `json:"total_links,omitempty"`
}

// linkDataFromLinks converts a links.Link to linkData.
func linkDataFromLinks(link links.Link) linkData {
	return linkData{
		Source:    link.Source,
		Target:    link.Target,
		Context:   link.Context,
		CreatedAt: link.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
