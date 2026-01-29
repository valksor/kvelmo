package views

import (
	"embed"
	"html/template"
	"io"
)

// Renderer handles template loading and rendering.
type Renderer struct {
	fs        embed.FS
	templates map[string]*template.Template
	funcMap   template.FuncMap
}

// NewRenderer creates a new template renderer with the given embedded filesystem.
// The filesystem should contain templates at templates/*.html and templates/partials/*.html.
func NewRenderer(fs embed.FS) (*Renderer, error) {
	r := &Renderer{
		fs:        fs,
		templates: make(map[string]*template.Template),
		funcMap:   defaultFuncMap(),
	}

	if err := r.loadTemplates(); err != nil {
		return nil, err
	}

	return r, nil
}

// defaultFuncMap returns the function map for templates.
func defaultFuncMap() template.FuncMap {
	return template.FuncMap{
		// State display
		"stateIcon":  func(state string) string { return GetStateDisplay(state).Icon },
		"stateColor": func(state string) string { return GetStateDisplay(state).Color },
		"stateBadge": func(state string) string { return GetStateDisplay(state).Badge },
		"stateBar":   func(state string) string { return GetStateDisplay(state).BarColor },

		// Spec status display
		"specIcon":  func(status string) string { return GetSpecStatusDisplay(status).Icon },
		"specColor": func(status string) string { return GetSpecStatusDisplay(status).Color },

		// Formatting
		"formatCost":     FormatCost,
		"formatNumber":   FormatNumber,
		"formatPercent":  FormatPercent,
		"formatPct":      FormatPercent, // Alias for backward compatibility
		"timeAgo":        FormatTimeAgo,
		"formatDate":     FormatDate,
		"formatDuration": FormatDuration,
		"shortID":        ShortID,
		"truncate":       TruncateString,
		"pluralize":      Pluralize,

		// Labels
		"labelColor": LabelColor,

		// Utilities
		"mul": func(a, b float64) float64 { return a * b },
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}

			return a / b
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },

		// Conditionals
		"eq": func(a, b interface{}) bool { return a == b },
		"ne": func(a, b interface{}) bool { return a != b },
		"gt": func(a, b float64) bool { return a > b },
		"lt": func(a, b float64) bool { return a < b },
		"ge": func(a, b float64) bool { return a >= b },
		"le": func(a, b float64) bool { return a <= b },

		// String utilities
		"safe": func(s string) template.HTML { return template.HTML(s) }, //nolint:gosec // For pre-sanitized content

		// Constants - SSE event names for HTMX triggers
		"eventWorkflowStateChanged": func() string { return EventWorkflowStateChanged },
		"eventSpecUpdated":          func() string { return EventSpecUpdated },
		"eventQuestionAsked":        func() string { return EventQuestionAsked },
		"eventCostsUpdated":         func() string { return EventCostsUpdated },
		"eventQuickTasksUpdated":    func() string { return EventQuickTasksUpdated },
		"eventHierarchyUpdated":     func() string { return EventHierarchyUpdated },
	}
}

// loadTemplates loads all HTML templates.
func (r *Renderer) loadTemplates() error {
	// Page templates that extend base
	pages := []string{
		"dashboard",
		"interactive",
		"login",
		"settings",
		"project",
		"browser",
		"history",
		"license",
		"quick",
		"memory",
		"find",
		"links",
	}

	for _, page := range pages {
		tmpl, err := template.New(page+".html").Funcs(r.funcMap).ParseFS(
			r.fs,
			"templates/base.html",
			"templates/pages/"+page+".html",
			"templates/partials/*.html",
		)
		if err != nil {
			// Try legacy path (templates/*.html)
			tmpl, err = template.New(page+".html").Funcs(r.funcMap).ParseFS(
				r.fs,
				"templates/base.html",
				"templates/"+page+".html",
				"templates/partials/*.html",
			)
			if err != nil {
				return err
			}
		}
		r.templates[page] = tmpl
	}

	// Load standalone partials
	partials := []string{
		"task_card",
		"actions",
		"specification",
		"specifications",
		"question",
		"costs",
		"stats",
		"active_work",
		"recent_tasks",
		"labels",
		"modal",
		"memory_results",
		"memory_stats",
	}

	for _, name := range partials {
		// Try new location first, fall back to legacy
		tmpl, err := template.New(name+".html").Funcs(r.funcMap).ParseFS(
			r.fs,
			"templates/partials/"+name+".html",
		)
		if err != nil {
			// Partial may not exist yet, skip
			continue
		}
		r.templates["partial:"+name] = tmpl
	}

	// Load empty state partials
	emptyStates := []string{
		"no_task",
		"no_stats",
		"no_project",
		"no_recent_tasks",
	}

	for _, name := range emptyStates {
		tmpl, err := template.New(name+".html").Funcs(r.funcMap).ParseFS(
			r.fs,
			"templates/partials/empty_states/"+name+".html",
		)
		if err != nil {
			continue
		}
		r.templates["empty:"+name] = tmpl
	}

	return nil
}

// Render renders a page template with the given data.
func (r *Renderer) Render(w io.Writer, name string, data interface{}) error {
	tmpl, ok := r.templates[name]
	if !ok {
		return &TemplateNotFoundError{Name: name}
	}

	return tmpl.ExecuteTemplate(w, "base", data)
}

// RenderPartial renders a partial template with the given data.
func (r *Renderer) RenderPartial(w io.Writer, name string, data interface{}) error {
	tmpl, ok := r.templates["partial:"+name]
	if !ok {
		return &TemplateNotFoundError{Name: name}
	}

	return tmpl.Execute(w, data)
}

// RenderEmptyState renders an empty state partial.
func (r *Renderer) RenderEmptyState(w io.Writer, name string, data interface{}) error {
	tmpl, ok := r.templates["empty:"+name]
	if !ok {
		return &TemplateNotFoundError{Name: name}
	}

	return tmpl.ExecuteTemplate(w, name, data)
}

// Page-specific render methods for type safety

// RenderDashboard renders the dashboard page.
func (r *Renderer) RenderDashboard(w io.Writer, data DashboardData) error {
	return r.Render(w, "dashboard", data)
}

// RenderLogin renders the login page.
func (r *Renderer) RenderLogin(w io.Writer, data LoginData) error {
	return r.Render(w, "login", data)
}

// RenderSettings renders the settings page.
func (r *Renderer) RenderSettings(w io.Writer, data SettingsData) error {
	return r.Render(w, "settings", data)
}

// RenderProject renders the project planning page.
func (r *Renderer) RenderProject(w io.Writer, data ProjectPlanningData) error {
	return r.Render(w, "project", data)
}

// RenderBrowser renders the browser control panel page.
func (r *Renderer) RenderBrowser(w io.Writer, data BrowserData) error {
	return r.Render(w, "browser", data)
}

// RenderHistory renders the task history page.
func (r *Renderer) RenderHistory(w io.Writer, data HistoryData) error {
	return r.Render(w, "history", data)
}

// RenderLicense renders the license page.
func (r *Renderer) RenderLicense(w io.Writer, data LicenseData) error {
	return r.Render(w, "license", data)
}

// RenderQuick renders the quick tasks page.
func (r *Renderer) RenderQuick(w io.Writer, data QuickTasksData) error {
	return r.Render(w, "quick", data)
}

// RenderMemory renders the memory page.
func (r *Renderer) RenderMemory(w io.Writer, data MemoryData) error {
	return r.Render(w, "memory", data)
}

// RenderLinks renders the links page.
func (r *Renderer) RenderLinks(w io.Writer, data LinksData) error {
	return r.Render(w, "links", data)
}

// RenderFind renders the find page.
func (r *Renderer) RenderFind(w io.Writer, data interface{}) error {
	return r.Render(w, "find", data)
}

// TemplateNotFoundError is returned when a template is not found.
type TemplateNotFoundError struct {
	Name string
}

func (e *TemplateNotFoundError) Error() string {
	return "template not found: " + e.Name
}
