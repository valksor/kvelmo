package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/valksor/go-mehrhof/internal/library"
	"github.com/valksor/go-mehrhof/internal/server/views"
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
		if r.Header.Get("Hx-Request") == "true" {
			s.writeLibraryEmptyHTML(w)

			return
		}
		s.writeJSON(w, http.StatusOK, libraryListResponse{
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

	// Check if this is an HTMX request - return HTML partial
	if r.Header.Get("Hx-Request") == "true" {
		s.writeLibraryCollectionsHTML(w, collections)

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

	// Check if this is an HTMX request - return HTML partial
	if r.Header.Get("Hx-Request") == "true" {
		s.writeLibraryDetailHTML(w, collection, pages)

		return
	}

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

	if r.Header.Get("Hx-Request") == "true" {
		s.writeLibraryFeedbackHTML(w, true, fmt.Sprintf("Collection '%s' removed successfully", nameOrID))

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
		stats := &libraryStatsResponse{Enabled: false}
		if r.Header.Get("Hx-Request") == "true" {
			s.writeLibraryStatsHTML(w, stats)

			return
		}
		s.writeJSON(w, http.StatusOK, stats)

		return
	}

	// Collect stats from collections - in global mode, only show shared stats
	listOpts := &library.ListOptions{}
	if s.config.Mode == ModeGlobal {
		listOpts.SharedOnly = true
	}
	collections, err := lib.List(r.Context(), listOpts)
	if err != nil {
		if r.Header.Get("Hx-Request") == "true" {
			s.writeLibraryStatsHTML(w, nil)

			return
		}
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

	stats := &libraryStatsResponse{
		TotalCollections: len(collections),
		TotalPages:       totalPages,
		TotalSize:        totalSize,
		ProjectCount:     projectCount,
		SharedCount:      sharedCount,
		ByMode:           byMode,
		Enabled:          true,
	}

	if r.Header.Get("Hx-Request") == "true" {
		s.writeLibraryStatsHTML(w, stats)

		return
	}

	s.writeJSON(w, http.StatusOK, stats)
}

// writeLibraryEmptyHTML renders the empty state for library collections.
func (s *Server) writeLibraryEmptyHTML(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	_, _ = w.Write([]byte(`
		<div class="card">
			<div class="text-center py-12">
				<div class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-base-200 flex items-center justify-center">
					<svg class="w-8 h-8 text-base-content/40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"></path>
					</svg>
				</div>
				<p class="text-base-content/60">No documentation collections yet.</p>
				<p class="text-sm text-base-content/60 mt-1">Pull documentation from a URL, file, or git repository to get started.</p>
			</div>
		</div>
	`))
}

// writeLibraryCollectionsHTML renders library collections as HTML partial.
func (s *Server) writeLibraryCollectionsHTML(w http.ResponseWriter, collections []*library.Collection) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if len(collections) == 0 {
		s.writeLibraryEmptyHTML(w)

		return
	}

	html := fmt.Sprintf(`
		<div class="card">
			<div class="px-6 py-4 border-b border-base-300 flex items-center justify-between">
				<span class="text-sm font-medium text-base-content/80">%d %s</span>
				<div class="flex gap-2">
					<button hx-get="/api/v1/library" hx-target="#library-collections" hx-swap="innerHTML" class="btn btn-sm btn-ghost">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
						</svg>
						Refresh
					</button>
				</div>
			</div>
			<div class="divide-y divide-base-100">
	`, len(collections), pluralize(len(collections), "collection", "collections"))

	var htmlSb strings.Builder
	for _, c := range collections {
		// Source type icon
		sourceIcon := "M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"
		switch c.SourceType {
		case library.SourceURL:
			// Default URL icon (no change needed)
		case library.SourceFile:
			sourceIcon = "M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
		case library.SourceGit:
			sourceIcon = "M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
		}

		// Mode badge
		modeBadge := "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
		modeText := "auto"
		switch c.IncludeMode {
		case library.IncludeModeAuto:
			// Default auto badge (no change needed)
		case library.IncludeModeExplicit:
			modeBadge = "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300"
			modeText = "explicit"
		case library.IncludeModeAlways:
			modeBadge = "bg-success-100 text-success-700 dark:bg-success-900/30 dark:text-success-300"
			modeText = "always"
		}

		// Location badge
		locationBadge := "bg-base-200 text-base-content/60"
		locationText := "project"
		if c.Location == "shared" {
			locationBadge = "bg-warning-100 text-warning-700 dark:bg-warning-900/30 dark:text-warning-300"
			locationText = "shared"
		}

		// Tags
		tagsHTML := ""
		if len(c.Tags) > 0 {
			var tagsHTMLSb385 strings.Builder
			for _, tag := range c.Tags[:min(3, len(c.Tags))] {
				tagsHTMLSb385.WriteString(fmt.Sprintf(`<span class="px-2 py-0.5 text-xs rounded bg-base-200 text-base-content/60">%s</span>`, tag))
			}
			tagsHTML += tagsHTMLSb385.String()
		}

		htmlSb.WriteString(fmt.Sprintf(`
			<div class="p-6 hover:bg-base-50 dark:hover:bg-base-800/50 transition-smooth">
				<div class="flex items-start justify-between gap-4">
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-3 mb-2">
							<svg class="w-5 h-5 text-base-content/60 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="%s"></path>
							</svg>
							<h3 class="font-semibold text-base-content truncate">%s</h3>
							<span class="px-2 py-0.5 rounded text-xs font-medium %s">%s</span>
							<span class="px-2 py-0.5 rounded text-xs font-medium %s">%s</span>
						</div>
						<p class="text-sm text-base-content/60 truncate mb-2">%s</p>
						<div class="flex items-center gap-4 text-xs text-base-content/50">
							<span>%d pages</span>
							<span>%s</span>
							<span>%s</span>
						</div>
						%s
					</div>
					<div class="flex items-center gap-2 flex-shrink-0">
						<button hx-get="/api/v1/library/%s" hx-target="#library-detail-modal" hx-swap="innerHTML" class="btn btn-sm btn-ghost">
							View
						</button>
						<button hx-delete="/api/v1/library/%s" hx-target="#library-collections" hx-swap="innerHTML" hx-confirm="Remove collection '%s'?" class="btn btn-sm btn-ghost text-error-600 hover:bg-error-50">
							Remove
						</button>
					</div>
				</div>
			</div>
		`, sourceIcon, c.Name, modeBadge, modeText, locationBadge, locationText,
			c.Source, c.PageCount, views.FormatBytes(c.TotalSize), views.FormatTimeAgo(c.PulledAt),
			tagsHTML, c.ID, c.ID, c.Name))
	}
	html += htmlSb.String()

	html += `
			</div>
		</div>
		<div id="library-detail-modal"></div>
	`

	_, _ = w.Write([]byte(html))
}

// writeLibraryDetailHTML renders collection detail as HTML partial.
func (s *Server) writeLibraryDetailHTML(w http.ResponseWriter, c *library.Collection, pages []string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := fmt.Sprintf(`
		<div class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4" onclick="if(event.target===this)this.remove()">
			<div class="bg-base-100 rounded-2xl shadow-2xl max-w-2xl w-full max-h-[80vh] overflow-hidden">
				<div class="px-6 py-4 border-b border-base-300 flex items-center justify-between">
					<h3 class="text-lg font-bold text-base-content">%s</h3>
					<button onclick="this.closest('.fixed').remove()" class="btn btn-sm btn-ghost">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
						</svg>
					</button>
				</div>
				<div class="p-6 overflow-y-auto max-h-[60vh]">
					<dl class="grid grid-cols-2 gap-4 text-sm mb-6">
						<div>
							<dt class="text-base-content/60">Source</dt>
							<dd class="font-medium text-base-content truncate">%s</dd>
						</div>
						<div>
							<dt class="text-base-content/60">Type</dt>
							<dd class="font-medium text-base-content">%s</dd>
						</div>
						<div>
							<dt class="text-base-content/60">Include Mode</dt>
							<dd class="font-medium text-base-content">%s</dd>
						</div>
						<div>
							<dt class="text-base-content/60">Location</dt>
							<dd class="font-medium text-base-content">%s</dd>
						</div>
						<div>
							<dt class="text-base-content/60">Pages</dt>
							<dd class="font-medium text-base-content">%d</dd>
						</div>
						<div>
							<dt class="text-base-content/60">Total Size</dt>
							<dd class="font-medium text-base-content">%s</dd>
						</div>
					</dl>
	`, c.Name, c.Source, c.SourceType, c.IncludeMode, c.Location, c.PageCount, views.FormatBytes(c.TotalSize))

	if len(c.Paths) > 0 {
		html += `
					<div class="mb-4">
						<h4 class="text-sm font-medium text-base-content/80 mb-2">Path Patterns</h4>
						<div class="flex flex-wrap gap-2">
		`
		var htmlSb485 strings.Builder
		for _, p := range c.Paths {
			htmlSb485.WriteString(fmt.Sprintf(`<code class="text-xs bg-base-200 px-2 py-1 rounded">%s</code>`, p))
		}
		html += htmlSb485.String()
		html += `
						</div>
					</div>
		`
	}

	if len(pages) > 0 {
		html += `
					<div>
						<h4 class="text-sm font-medium text-base-content/80 mb-2">Pages</h4>
						<div class="bg-base-200 rounded-lg p-3 max-h-48 overflow-y-auto">
							<ul class="text-xs font-mono space-y-1">
		`
		var htmlSb501 strings.Builder
		for _, p := range pages[:min(50, len(pages))] {
			htmlSb501.WriteString(fmt.Sprintf(`<li class="text-base-content/70">%s</li>`, p))
		}
		html += htmlSb501.String()
		if len(pages) > 50 {
			html += fmt.Sprintf(`<li class="text-base-content/50 italic">... and %d more</li>`, len(pages)-50)
		}
		html += `
							</ul>
						</div>
					</div>
		`
	}

	html += `
				</div>
			</div>
		</div>
	`

	_, _ = w.Write([]byte(html))
}

// writeLibraryStatsHTML renders library statistics as HTML partial.
func (s *Server) writeLibraryStatsHTML(w http.ResponseWriter, stats *libraryStatsResponse) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if stats == nil || !stats.Enabled {
		_, _ = w.Write([]byte(`
			<div class="text-center py-4">
				<p class="text-base-content/50 text-sm">Library system not available.</p>
				<p class="text-base-content/40 text-xs mt-1">Configure in workspace settings.</p>
			</div>
		`))

		return
	}

	html := fmt.Sprintf(`
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<span class="text-sm text-base-content/60">Collections</span>
				<span class="text-2xl font-bold text-base-content">%s</span>
			</div>
			<div class="flex items-center justify-between">
				<span class="text-sm text-base-content/60">Total Pages</span>
				<span class="text-lg font-semibold text-base-content">%s</span>
			</div>
			<div class="flex items-center justify-between">
				<span class="text-sm text-base-content/60">Total Size</span>
				<span class="text-lg font-semibold text-base-content">%s</span>
			</div>
	`, views.FormatNumber(stats.TotalCollections), views.FormatNumber(stats.TotalPages), views.FormatBytes(stats.TotalSize))

	if stats.ProjectCount > 0 || stats.SharedCount > 0 {
		html += `
			<div class="border-t border-base-300 pt-4 space-y-2">
				<span class="text-sm font-medium text-base-content/80">By Location</span>
		`
		if stats.ProjectCount > 0 {
			html += fmt.Sprintf(`
				<div class="flex items-center justify-between">
					<span class="text-sm text-base-content/60">Project</span>
					<span class="text-sm font-medium text-base-content">%d</span>
				</div>
			`, stats.ProjectCount)
		}
		if stats.SharedCount > 0 {
			html += fmt.Sprintf(`
				<div class="flex items-center justify-between">
					<span class="text-sm text-base-content/60">Shared</span>
					<span class="text-sm font-medium text-base-content">%d</span>
				</div>
			`, stats.SharedCount)
		}
		html += `
			</div>
		`
	}

	html += `</div>`

	_, _ = w.Write([]byte(html))
}

// writeLibraryFeedbackHTML renders feedback message as HTML.
func (s *Server) writeLibraryFeedbackHTML(w http.ResponseWriter, success bool, message string) {
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

// Helper function for pluralization.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}

// Response types for library handlers.

type libraryListResponse struct {
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

type libraryStatsResponse struct {
	TotalCollections int            `json:"total_collections"`
	TotalPages       int            `json:"total_pages"`
	TotalSize        int64          `json:"total_size"`
	ProjectCount     int            `json:"project_count"`
	SharedCount      int            `json:"shared_count"`
	ByMode           map[string]int `json:"by_mode"`
	Enabled          bool           `json:"enabled"`
}
