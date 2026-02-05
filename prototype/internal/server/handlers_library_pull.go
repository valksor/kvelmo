package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/library"
)

// handleLibraryPull handles pulling documentation from a source.
func (s *Server) handleLibraryPull(w http.ResponseWriter, r *http.Request) {
	lib := s.getLibrary()
	if lib == nil {
		s.writeError(w, http.StatusServiceUnavailable, "library system not available")

		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid form data: "+err.Error())

		return
	}

	source := strings.TrimSpace(r.FormValue("source"))
	if source == "" {
		s.writeError(w, http.StatusBadRequest, "source is required")

		return
	}

	// Build pull options
	opts := &library.PullOptions{
		Name:        strings.TrimSpace(r.FormValue("name")),
		IncludeMode: parseIncludeMode(r.FormValue("mode")),
		Shared:      r.FormValue("shared") == "on" || r.FormValue("shared") == "true",
	}

	// Parse paths
	if paths := strings.TrimSpace(r.FormValue("paths")); paths != "" {
		for _, p := range strings.Split(paths, ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				opts.Paths = append(opts.Paths, trimmed)
			}
		}
	}

	// Parse tags
	if tags := strings.TrimSpace(r.FormValue("tags")); tags != "" {
		for _, t := range strings.Split(tags, ",") {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				opts.Tags = append(opts.Tags, trimmed)
			}
		}
	}

	// Parse crawl options
	if maxDepth := r.FormValue("max_depth"); maxDepth != "" {
		if d, err := strconv.Atoi(maxDepth); err == nil && d > 0 {
			opts.MaxDepth = d
		}
	}
	if maxPages := r.FormValue("max_pages"); maxPages != "" {
		if p, err := strconv.Atoi(maxPages); err == nil && p > 0 {
			opts.MaxPages = p
		}
	}

	// Parse resume options
	opts.Continue = r.FormValue("continue") == "true" || r.FormValue("continue") == "on"
	opts.ForceRestart = r.FormValue("restart") == "true" || r.FormValue("restart") == "on"

	// Parse crawl filtering options
	if domainScope := r.FormValue("domain_scope"); domainScope != "" {
		opts.DomainScope = domainScope
	}
	opts.VersionFilter = r.FormValue("version_filter") == "on" || r.FormValue("version_filter") == "true"
	if version := strings.TrimSpace(r.FormValue("version")); version != "" {
		opts.VersionPath = version
	}

	// Pull documentation
	result, err := lib.Pull(r.Context(), source, opts)
	if err != nil {
		// Handle incomplete crawl error specially
		var incompleteErr *library.IncompleteCrawlError
		if errors.As(err, &incompleteErr) {
			s.writeJSON(w, http.StatusConflict, map[string]any{
				"error":         "incomplete_crawl",
				"message":       incompleteErr.Error(),
				"collection_id": incompleteErr.CollectionID,
				"total":         incompleteErr.Total,
				"success":       incompleteErr.Success,
				"failed":        incompleteErr.Failed,
				"pending":       incompleteErr.Pending,
				"started_at":    incompleteErr.StartedAt,
			})

			return
		}

		s.writeError(w, http.StatusInternalServerError, "failed to pull: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, libraryPullResponse{
		Success:      true,
		CollectionID: result.Collection.ID,
		Name:         result.Collection.Name,
		PagesWritten: result.PagesWritten,
		Source:       source,
	})
}

// handleLibraryPullPreview previews what would be pulled without actually pulling.
func (s *Server) handleLibraryPullPreview(w http.ResponseWriter, r *http.Request) {
	lib := s.getLibrary()
	if lib == nil {
		s.writeError(w, http.StatusServiceUnavailable, "library system not available")

		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid form data: "+err.Error())

		return
	}

	source := strings.TrimSpace(r.FormValue("source"))
	if source == "" {
		s.writeError(w, http.StatusBadRequest, "source is required")

		return
	}

	// Build pull options with dry run
	opts := &library.PullOptions{
		Name:        strings.TrimSpace(r.FormValue("name")),
		IncludeMode: parseIncludeMode(r.FormValue("mode")),
		Shared:      r.FormValue("shared") == "on" || r.FormValue("shared") == "true",
		DryRun:      true, // Enable dry run
	}

	// Parse crawl options
	if maxDepth := r.FormValue("max_depth"); maxDepth != "" {
		if d, err := strconv.Atoi(maxDepth); err == nil && d > 0 {
			opts.MaxDepth = d
		}
	}
	if maxPages := r.FormValue("max_pages"); maxPages != "" {
		if p, err := strconv.Atoi(maxPages); err == nil && p > 0 {
			opts.MaxPages = p
		}
	}

	// Parse crawl filtering options
	if domainScope := r.FormValue("domain_scope"); domainScope != "" {
		opts.DomainScope = domainScope
	}
	opts.VersionFilter = r.FormValue("version_filter") == "on" || r.FormValue("version_filter") == "true"
	if version := strings.TrimSpace(r.FormValue("version")); version != "" {
		opts.VersionPath = version
	}

	// Preview pull
	result, err := lib.Pull(r.Context(), source, opts)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "preview failed: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, libraryPreviewResponse{
		URLs:  result.DryRunURLs,
		Count: len(result.DryRunURLs),
	})
}

// parseIncludeMode converts form value to library.IncludeMode.
func parseIncludeMode(value string) library.IncludeMode {
	switch strings.ToLower(value) {
	case "explicit":
		return library.IncludeModeExplicit
	case "always":
		return library.IncludeModeAlways
	default:
		return library.IncludeModeAuto
	}
}

// Response types for library pull handlers.

type libraryPullResponse struct {
	Success      bool   `json:"success"`
	CollectionID string `json:"collection_id"`
	Name         string `json:"name"`
	PagesWritten int    `json:"pages_written"`
	Source       string `json:"source"`
}

type libraryPreviewResponse struct {
	URLs  []string `json:"urls"`
	Count int      `json:"count"`
}
