package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/library"
	"github.com/valksor/go-mehrhof/internal/server/views"
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
		if r.Header.Get("Hx-Request") == "true" {
			s.writeLibraryFeedbackHTML(w, false, "Source is required")

			return
		}
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
			if r.Header.Get("Hx-Request") == "true" {
				s.writeIncompleteCrawlHTML(w, incompleteErr)

				return
			}
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

		if r.Header.Get("Hx-Request") == "true" {
			s.writeLibraryFeedbackHTML(w, false, "Failed to pull: "+err.Error())

			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to pull: "+err.Error())

		return
	}

	if r.Header.Get("Hx-Request") == "true" {
		s.writeLibraryPullResultHTML(w, result)

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
		if r.Header.Get("Hx-Request") == "true" {
			s.writeLibraryFeedbackHTML(w, false, "Source is required")

			return
		}
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
		if r.Header.Get("Hx-Request") == "true" {
			s.writeLibraryFeedbackHTML(w, false, "Preview failed: "+err.Error())

			return
		}
		s.writeError(w, http.StatusInternalServerError, "preview failed: "+err.Error())

		return
	}

	if r.Header.Get("Hx-Request") == "true" {
		s.writeLibraryPreviewHTML(w, result)

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

// writeLibraryPullResultHTML renders the pull result as HTML.
func (s *Server) writeLibraryPullResultHTML(w http.ResponseWriter, result *library.PullResult) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := fmt.Sprintf(`
		<div class="p-4 rounded-lg bg-success-50 dark:bg-success-900/20 border border-success-200 dark:border-success-800">
			<div class="flex items-start gap-3">
				<svg class="w-5 h-5 text-success-600 dark:text-success-400 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
					<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path>
				</svg>
				<div>
					<h4 class="font-semibold text-success-800 dark:text-success-200">Documentation pulled successfully</h4>
					<dl class="mt-2 text-sm text-success-700 dark:text-success-300 space-y-1">
						<div class="flex gap-2">
							<dt class="font-medium">Collection:</dt>
							<dd>%s</dd>
						</div>
						<div class="flex gap-2">
							<dt class="font-medium">Pages:</dt>
							<dd>%d</dd>
						</div>
						<div class="flex gap-2">
							<dt class="font-medium">Size:</dt>
							<dd>%s</dd>
						</div>
					</dl>
				</div>
			</div>
		</div>
		<script>
			// Refresh collections list
			htmx.trigger('#library-collections', 'load');
			htmx.trigger('#library-stats', 'load');
		</script>
	`, result.Collection.Name, result.PagesWritten, views.FormatBytes(result.Collection.TotalSize))

	_, _ = w.Write([]byte(html))
}

// writeLibraryPreviewHTML renders the preview result as HTML.
func (s *Server) writeLibraryPreviewHTML(w http.ResponseWriter, result *library.PullResult) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if len(result.DryRunURLs) == 0 {
		_, _ = w.Write([]byte(`
			<div class="p-4 rounded-lg bg-warning-50 dark:bg-warning-900/20 border border-warning-200 dark:border-warning-800">
				<div class="flex items-start gap-3">
					<svg class="w-5 h-5 text-warning-600 dark:text-warning-400 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
						<path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path>
					</svg>
					<div>
						<h4 class="font-semibold text-warning-800 dark:text-warning-200">No pages found</h4>
						<p class="text-sm text-warning-700 dark:text-warning-300 mt-1">The preview found no pages to crawl. Check the source URL.</p>
					</div>
				</div>
			</div>
		`))

		return
	}

	html := fmt.Sprintf(`
		<div class="p-4 rounded-lg bg-info-50 dark:bg-info-900/20 border border-info-200 dark:border-info-800">
			<div class="flex items-start gap-3">
				<svg class="w-5 h-5 text-info-600 dark:text-info-400 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
					<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"></path>
				</svg>
				<div class="flex-1">
					<h4 class="font-semibold text-info-800 dark:text-info-200">Preview: %d pages would be crawled</h4>
					<div class="mt-2 max-h-48 overflow-y-auto bg-base-100 dark:bg-base-900 rounded-lg p-3">
						<ul class="text-xs font-mono space-y-1 text-info-700 dark:text-info-300">
	`, len(result.DryRunURLs))

	var htmlSb276 strings.Builder
	for i, url := range result.DryRunURLs[:min(20, len(result.DryRunURLs))] {
		htmlSb276.WriteString(fmt.Sprintf(`<li>%d. %s</li>`, i+1, url))
	}
	html += htmlSb276.String()
	if len(result.DryRunURLs) > 20 {
		html += fmt.Sprintf(`<li class="italic text-info-500">... and %d more</li>`, len(result.DryRunURLs)-20)
	}

	html += `
						</ul>
					</div>
					<p class="text-sm text-info-600 dark:text-info-400 mt-2">Click "Pull" to fetch these pages.</p>
				</div>
			</div>
		</div>
	`

	_, _ = w.Write([]byte(html))
}

// writeIncompleteCrawlHTML renders the incomplete crawl dialog with resume/restart options.
func (s *Server) writeIncompleteCrawlHTML(w http.ResponseWriter, err *library.IncompleteCrawlError) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := fmt.Sprintf(`
		<div class="p-4 rounded-lg bg-warning-50 dark:bg-warning-900/20 border border-warning-200 dark:border-warning-800">
			<div class="flex items-start gap-3">
				<svg class="w-5 h-5 text-warning-600 dark:text-warning-400 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
					<path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path>
				</svg>
				<div class="flex-1">
					<h4 class="font-semibold text-warning-800 dark:text-warning-200">Incomplete Crawl Found</h4>
					<dl class="mt-2 text-sm text-warning-700 dark:text-warning-300 space-y-1">
						<div class="flex gap-2">
							<dt class="font-medium">Collection:</dt>
							<dd>%s</dd>
						</div>
						<div class="flex gap-2">
							<dt class="font-medium">Started:</dt>
							<dd>%s</dd>
						</div>
						<div class="flex gap-2">
							<dt class="font-medium">Progress:</dt>
							<dd>%d/%d pages (%d failed, %d pending)</dd>
						</div>
					</dl>
					<div class="mt-4 flex gap-2">
						<button
							hx-post="/api/v1/library/pull"
							hx-include="#library-pull-form"
							hx-vals='{"continue": "true"}'
							hx-target="#pull-feedback"
							hx-swap="innerHTML"
							class="btn btn-sm btn-primary">
							Resume Crawl
						</button>
						<button
							hx-post="/api/v1/library/pull"
							hx-include="#library-pull-form"
							hx-vals='{"restart": "true"}'
							hx-target="#pull-feedback"
							hx-swap="innerHTML"
							class="btn btn-sm btn-ghost">
							Start Fresh
						</button>
					</div>
				</div>
			</div>
		</div>
	`, err.CollectionID, err.StartedAt.Format("2006-01-02 15:04"), err.Success, err.Total, err.Failed, err.Pending)

	_, _ = w.Write([]byte(html))
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
