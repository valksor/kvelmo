package server

import (
	"net/http"
	"strings"

	"github.com/valksor/go-mehrhof/internal/library"
)

// getLibrary returns the library manager, preferring conductor's library
// but falling back to shared-only library in global mode when no project is selected.
func (s *Server) getLibrary() *library.Manager {
	if s.config.Conductor != nil {
		if lib := s.config.Conductor.GetLibrary(); lib != nil {
			return lib
		}
	}

	return s.sharedLibrary // May be nil if not in global mode
}

// handleLibraryList returns library collections.
func (s *Server) handleLibraryList(w http.ResponseWriter, r *http.Request) {
	lib := s.getLibrary()
	if lib == nil {
		s.writeJSON(w, http.StatusOK, libraryListResponse{
			Enabled:     false,
			Collections: []libraryCollectionResponse{},
			Count:       0,
		})

		return
	}

	// Parse filter options
	opts := &library.ListOptions{}
	if r.URL.Query().Get("shared") == "true" {
		opts.SharedOnly = true
	}
	if r.URL.Query().Get("project") == "true" {
		opts.ProjectOnly = true
	}
	if tag := r.URL.Query().Get("tag"); tag != "" {
		opts.Tag = tag
	}
	// In global mode with no explicit filter, default to shared-only
	if s.config.Mode == ModeGlobal && !opts.SharedOnly && !opts.ProjectOnly {
		opts.SharedOnly = true
	}

	collections, err := lib.List(r.Context(), opts)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list collections: "+err.Error())

		return
	}

	// Convert to response format
	var response []libraryCollectionResponse
	for _, c := range collections {
		response = append(response, libraryCollectionResponse{
			ID:          c.ID,
			Name:        c.Name,
			Source:      c.Source,
			SourceType:  string(c.SourceType),
			IncludeMode: string(c.IncludeMode),
			PageCount:   c.PageCount,
			TotalSize:   c.TotalSize,
			Location:    c.Location,
			PulledAt:    c.PulledAt,
			Tags:        c.Tags,
			Paths:       c.Paths,
		})
	}

	s.writeJSON(w, http.StatusOK, libraryListResponse{
		Enabled:     true,
		Collections: response,
		Count:       len(response),
	})
}

// handleLibraryShow returns details for a specific collection.
func (s *Server) handleLibraryShow(w http.ResponseWriter, r *http.Request) {
	lib := s.getLibrary()
	if lib == nil {
		s.writeError(w, http.StatusServiceUnavailable, "library system not available")

		return
	}

	// Get collection name/ID from path
	nameOrID := strings.TrimPrefix(r.URL.Path, "/api/v1/library/")
	if nameOrID == "" {
		s.writeError(w, http.StatusBadRequest, "collection name or ID required")

		return
	}

	collection, err := lib.Show(r.Context(), nameOrID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "collection not found: "+err.Error())

		return
	}

	// Get page list
	pages, _ := lib.ListPages(r.Context(), collection.ID)

	s.writeJSON(w, http.StatusOK, libraryShowResponse{
		Collection: libraryCollectionResponse{
			ID:          collection.ID,
			Name:        collection.Name,
			Source:      collection.Source,
			SourceType:  string(collection.SourceType),
			IncludeMode: string(collection.IncludeMode),
			PageCount:   collection.PageCount,
			TotalSize:   collection.TotalSize,
			Location:    collection.Location,
			PulledAt:    collection.PulledAt,
			Tags:        collection.Tags,
			Paths:       collection.Paths,
		},
		Pages: pages,
	})
}

// handleLibraryItems returns page items (metadata + content) for a collection.
func (s *Server) handleLibraryItems(w http.ResponseWriter, r *http.Request) {
	lib := s.getLibrary()
	if lib == nil {
		s.writeError(w, http.StatusServiceUnavailable, "library system not available")

		return
	}

	collectionID := r.PathValue("id")
	if collectionID == "" {
		s.writeError(w, http.StatusBadRequest, "collection id is required")

		return
	}

	collection, err := lib.Show(r.Context(), collectionID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "collection not found: "+err.Error())

		return
	}

	pagePaths, err := lib.ListPages(r.Context(), collection.ID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list pages: "+err.Error())

		return
	}

	items := make([]libraryItemResponse, 0, len(pagePaths))
	for _, pagePath := range pagePaths {
		page, content, showErr := lib.ShowPage(r.Context(), collection.ID, pagePath)
		if showErr != nil {
			continue
		}

		title := pagePath
		if page != nil && page.Title != "" {
			title = page.Title
		}

		items = append(items, libraryItemResponse{
			ID:         pagePath,
			Title:      title,
			Content:    content,
			Collection: collection.ID,
		})
	}

	s.writeJSON(w, http.StatusOK, libraryItemsResponse{
		Collection: collection.ID,
		Items:      items,
		Count:      len(items),
	})
}

// handleLibraryRemove deletes a collection.
func (s *Server) handleLibraryRemove(w http.ResponseWriter, r *http.Request) {
	lib := s.getLibrary()
	if lib == nil {
		s.writeError(w, http.StatusServiceUnavailable, "library system not available")

		return
	}

	// Get collection name/ID from path
	nameOrID := strings.TrimPrefix(r.URL.Path, "/api/v1/library/")
	if nameOrID == "" {
		s.writeError(w, http.StatusBadRequest, "collection name or ID required")

		return
	}

	if err := lib.Remove(r.Context(), nameOrID, false); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to remove collection: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "collection removed successfully",
	})
}

// handleLibraryStats returns library statistics.
func (s *Server) handleLibraryStats(w http.ResponseWriter, r *http.Request) {
	lib := s.getLibrary()
	if lib == nil {
		s.writeJSON(w, http.StatusOK, &libraryStatsResponse{Enabled: false})

		return
	}

	// Collect stats from collections - in global mode, only show shared stats
	listOpts := &library.ListOptions{}
	if s.config.Mode == ModeGlobal {
		listOpts.SharedOnly = true
	}
	collections, err := lib.List(r.Context(), listOpts)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to get stats: "+err.Error())

		return
	}

	var totalPages int
	var totalSize int64
	projectCount := 0
	sharedCount := 0
	byMode := make(map[string]int)

	for _, c := range collections {
		totalPages += c.PageCount
		totalSize += c.TotalSize
		byMode[string(c.IncludeMode)]++
		if c.Location == "shared" {
			sharedCount++
		} else {
			projectCount++
		}
	}

	s.writeJSON(w, http.StatusOK, &libraryStatsResponse{
		TotalCollections: len(collections),
		TotalPages:       totalPages,
		TotalSize:        totalSize,
		ProjectCount:     projectCount,
		SharedCount:      sharedCount,
		ByMode:           byMode,
		Enabled:          true,
	})
}

// Response types for library handlers.

type libraryListResponse struct {
	Enabled     bool                        `json:"enabled"`
	Collections []libraryCollectionResponse `json:"collections"`
	Count       int                         `json:"count"`
}

type libraryCollectionResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	SourceType  string   `json:"source_type"`
	IncludeMode string   `json:"include_mode"`
	PageCount   int      `json:"page_count"`
	TotalSize   int64    `json:"total_size"`
	Location    string   `json:"location"`
	PulledAt    any      `json:"pulled_at"`
	Tags        []string `json:"tags,omitempty"`
	Paths       []string `json:"paths,omitempty"`
}

type libraryShowResponse struct {
	Collection libraryCollectionResponse `json:"collection"`
	Pages      []string                  `json:"pages"`
}

type libraryItemResponse struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Collection string   `json:"collection"`
	Tags       []string `json:"tags,omitempty"`
}

type libraryItemsResponse struct {
	Collection string                `json:"collection"`
	Items      []libraryItemResponse `json:"items"`
	Count      int                   `json:"count"`
}

type libraryStatsResponse struct {
	TotalCollections int            `json:"total_collections"`
	TotalPages       int            `json:"total_pages"`
	TotalSize        int64          `json:"total_size"`
	ProjectCount     int            `json:"project_count"`
	SharedCount      int            `json:"shared_count"`
	ByMode           map[string]int `json:"by_mode"`
	Enabled          bool           `json:"enabled"`
}
