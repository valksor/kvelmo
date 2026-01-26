package server

import (
	"embed"
	"html/template"
	"io"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
)

//go:embed templates/*.html templates/partials/*.html
var templateFS embed.FS

// Templates holds parsed templates for the server.
type Templates struct {
	dashboard *template.Template
	login     *template.Template
	settings  *template.Template
	project   *template.Template
	browser   *template.Template
	history   *template.Template
	license   *template.Template
	partials  map[string]*template.Template
}

// templateFuncs returns the function map for templates.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"stateIcon":  stateIcon,
		"stateColor": stateColor,
		"stateBadge": stateBadge,
		"specIcon":   specIcon,
		"specColor":  specColor,
		"timeAgo":    timeAgo,
		"formatCost": formatCost,
	}
}

// LoadTemplates loads all HTML templates.
func LoadTemplates() (*Templates, error) {
	t := &Templates{
		partials: make(map[string]*template.Template),
	}

	// Load dashboard template
	dashboard, err := template.New("dashboard.html").Funcs(templateFuncs()).ParseFS(
		templateFS,
		"templates/base.html",
		"templates/dashboard.html",
		"templates/partials/specification.html",
	)
	if err != nil {
		return nil, err
	}
	t.dashboard = dashboard

	// Load login template
	login, err := template.New("login.html").Funcs(templateFuncs()).ParseFS(
		templateFS,
		"templates/base.html",
		"templates/login.html",
	)
	if err != nil {
		return nil, err
	}
	t.login = login

	// Load settings template
	settings, err := template.New("settings.html").Funcs(templateFuncs()).ParseFS(
		templateFS,
		"templates/base.html",
		"templates/settings.html",
	)
	if err != nil {
		return nil, err
	}
	t.settings = settings

	// Load project template
	project, err := template.New("project.html").Funcs(templateFuncs()).ParseFS(
		templateFS,
		"templates/base.html",
		"templates/project.html",
	)
	if err != nil {
		return nil, err
	}
	t.project = project

	// Load browser template
	browser, err := template.New("browser.html").Funcs(templateFuncs()).ParseFS(
		templateFS,
		"templates/base.html",
		"templates/browser.html",
	)
	if err != nil {
		return nil, err
	}
	t.browser = browser

	// Load history template
	history, err := template.New("history.html").Funcs(templateFuncs()).ParseFS(
		templateFS,
		"templates/base.html",
		"templates/history.html",
	)
	if err != nil {
		return nil, err
	}
	t.history = history

	// Load license template
	license, err := template.New("license.html").Funcs(templateFuncs()).ParseFS(
		templateFS,
		"templates/base.html",
		"templates/license.html",
	)
	if err != nil {
		return nil, err
	}
	t.license = license

	// Load partials
	partialNames := []string{"task_card", "actions", "specification", "question", "costs", "card", "modal", "input", "skeleton"}
	for _, name := range partialNames {
		partial, err := template.New(name+".html").Funcs(templateFuncs()).ParseFS(
			templateFS,
			"templates/partials/"+name+".html",
		)
		if err != nil {
			return nil, err
		}
		t.partials[name] = partial
	}

	return t, nil
}

// RenderDashboard renders the dashboard page.
func (t *Templates) RenderDashboard(w io.Writer, data DashboardData) error {
	return t.dashboard.ExecuteTemplate(w, "base", data)
}

// RenderLogin renders the login page.
func (t *Templates) RenderLogin(w io.Writer, data LoginData) error {
	return t.login.ExecuteTemplate(w, "base", data)
}

// RenderPartial renders a partial template.
func (t *Templates) RenderPartial(w io.Writer, name string, data any) error {
	partial, ok := t.partials[name]
	if !ok {
		return nil
	}

	return partial.Execute(w, data)
}

// RenderSettings renders the settings page.
func (t *Templates) RenderSettings(w io.Writer, data SettingsData) error {
	return t.settings.ExecuteTemplate(w, "base", data)
}

// RenderProject renders the project planning page.
func (t *Templates) RenderProject(w io.Writer, data ProjectData) error {
	return t.project.ExecuteTemplate(w, "base", data)
}

// RenderBrowser renders the browser control panel page.
func (t *Templates) RenderBrowser(w io.Writer, data BrowserData) error {
	return t.browser.ExecuteTemplate(w, "base", data)
}

// RenderHistory renders the task history page.
func (t *Templates) RenderHistory(w io.Writer, data HistoryData) error {
	return t.history.ExecuteTemplate(w, "base", data)
}

// RenderLicense renders the license page.
func (t *Templates) RenderLicense(w io.Writer, data LicenseData) error {
	return t.license.ExecuteTemplate(w, "base", data)
}

// DashboardData holds data for the dashboard template.
type DashboardData struct {
	Mode             string
	AuthEnabled      bool
	IsGlobalMode     bool
	CanSwitchProject bool                      // True if server started in global mode
	Projects         []storage.ProjectMetadata // For global mode project picker
	HasTask          bool
	Task             *TaskData
	Guide            *GuideData
	Specifications   SpecificationsData
	PendingQuestion  *QuestionData
	Costs            *CostsData
}

// TaskData holds task information for display.
type TaskData struct {
	ID       string
	Title    string
	State    string
	Branch   string
	Worktree string
	Started  time.Time
	Ref      string
}

// GuideData holds guidance information.
type GuideData struct {
	NextActions []ActionData
}

// ActionData holds action button information.
type ActionData struct {
	Command     string
	Description string
	Endpoint    string
	Method      string
	Dangerous   bool
}

// SpecificationData holds specification information.
type SpecificationData struct {
	Number      int
	Name        string
	Title       string
	Status      string
	Description string
	Component   string
	CreatedAt   string
	CompletedAt string
}

// SpecificationsData holds specifications list with progress information.
type SpecificationsData struct {
	Specifications []SpecificationData
	Total          int
	Done           int
	Progress       float64
}

// QuestionData holds pending question information.
type QuestionData struct {
	Question string
	Options  []string
}

// CostsData holds cost information.
type CostsData struct {
	TotalCostUSD  float64
	TotalTokens   int
	InputTokens   int
	OutputTokens  int
	CachedTokens  int
	CachedPercent float64
}

// LoginData holds data for the login template.
type LoginData struct {
	Mode             string
	AuthEnabled      bool
	CanSwitchProject bool
	IsGlobalMode     bool // True when in global mode
	Error            string
	Redirect         string
}

// SettingsData holds data for the settings template.
type SettingsData struct {
	Mode             string
	AuthEnabled      bool
	CanSwitchProject bool
	IsGlobalMode     bool                      // True when in global mode
	ShowSensitive    bool                      // true for Project mode, false for Global mode
	Config           *storage.WorkspaceConfig  // Reuse existing config struct
	Agents           []string                  // Available agents for dropdown
	Success          string                    // Success message after save
	Error            string                    // Error message
	Projects         []storage.ProjectMetadata // Available projects for picker (global mode)
	SelectedProject  string                    // Currently selected project ID (global mode)
}

// ProjectData holds data for the project planning template.
type ProjectData struct {
	Mode             string
	AuthEnabled      bool
	CanSwitchProject bool
	IsGlobalMode     bool                      // True when in global mode
	Projects         []storage.ProjectMetadata // Available projects for picker
}

// BrowserData holds data for the browser control panel template.
type BrowserData struct {
	Mode             string
	AuthEnabled      bool
	CanSwitchProject bool
	IsGlobalMode     bool                      // True when in global mode
	Projects         []storage.ProjectMetadata // Available projects for picker
}

// HistoryData holds data for the task history template.
type HistoryData struct {
	Mode             string
	AuthEnabled      bool
	CanSwitchProject bool
	IsGlobalMode     bool                      // True when in global mode
	Projects         []storage.ProjectMetadata // Available projects for picker
}

// LicenseData holds data for the license template.
type LicenseData struct {
	Mode             string
	AuthEnabled      bool
	CanSwitchProject bool
	IsGlobalMode     bool // True when in global mode
	ProjectLicense   string
}

// Template helper functions.

func stateIcon(state string) string {
	switch state {
	case "idle":
		return "○"
	case "planning":
		return "◐"
	case "implementing":
		return "◑"
	case "reviewing":
		return "◉"
	case "done":
		return "●"
	case "failed":
		return "✗"
	case "waiting":
		return "?"
	default:
		return "○"
	}
}

func stateColor(state string) string {
	switch state {
	case "idle":
		return "text-gray-500"
	case "planning":
		return "text-blue-500"
	case "implementing":
		return "text-yellow-500"
	case "reviewing":
		return "text-purple-500"
	case "done":
		return "text-green-500"
	case "failed":
		return "text-red-500"
	case "waiting":
		return "text-orange-500"
	default:
		return "text-gray-500"
	}
}

func stateBadge(state string) string {
	switch state {
	case "idle":
		return "Ready"
	case "planning":
		return "Planning..."
	case "implementing":
		return "Implementing..."
	case "reviewing":
		return "Reviewing..."
	case "done":
		return "Complete"
	case "failed":
		return "Failed"
	case "waiting":
		return "Waiting for input"
	default:
		return state
	}
}

func specIcon(status string) string {
	switch status {
	case "draft":
		return "○"
	case "ready":
		return "◐"
	case "implementing":
		return "◑"
	case "done":
		return "●"
	default:
		return "○"
	}
}

func specColor(status string) string {
	switch status {
	case "draft":
		return "text-gray-400"
	case "ready":
		return "text-blue-500"
	case "implementing":
		return "text-yellow-500"
	case "done":
		return "text-green-500"
	default:
		return "text-gray-400"
	}
}

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}

		return string(rune('0'+mins/10)) + string(rune('0'+mins%10)) + " minutes ago"
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}

		return string(rune('0'+hours/10)) + string(rune('0'+hours%10)) + " hours ago"
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}

		return string(rune('0'+days/10)) + string(rune('0'+days%10)) + " days ago"
	}
}

func formatCost(cost float64) string {
	if cost < 0.01 {
		return "<$0.01"
	}
	// Simple formatting without fmt.Sprintf
	cents := int(cost * 100)
	dollars := cents / 100
	remainder := cents % 100
	result := "$"
	if dollars > 0 {
		result += string(rune('0' + dollars))
	} else {
		result += "0"
	}
	result += "."
	result += string(rune('0' + remainder/10))
	result += string(rune('0' + remainder%10))

	return result
}
