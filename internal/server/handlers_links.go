package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/valksor/go-mehrhof/internal/links"
	"github.com/valksor/go-mehrhof/internal/storage"
)

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
		s.writeJSON(w, http.StatusOK, &linksStatsResponse{
			TotalLinks:     0,
			TotalSources:   0,
			TotalTargets:   0,
			OrphanEntities: 0,
			MostLinked:     []entityResult{},
			Enabled:        false,
		})

		return
	}

	stats := linkMgr.GetStats()
	if stats == nil {
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

	s.writeJSON(w, http.StatusOK, &linksStatsResponse{
		TotalLinks:     stats.TotalLinks,
		TotalSources:   stats.TotalSources,
		TotalTargets:   stats.TotalTargets,
		OrphanEntities: stats.OrphanEntities,
		MostLinked:     mostLinked,
		Enabled:        true,
	})
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
		s.writeError(w, http.StatusInternalServerError, "failed to rebuild index: "+err.Error())

		return
	}

	stats := linkMgr.GetStats()
	s.writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"message":       "index rebuilt successfully",
		"total_links":   stats.TotalLinks,
		"total_sources": stats.TotalSources,
		"total_targets": stats.TotalTargets,
	})
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
