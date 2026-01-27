package server

import (
	"embed"
	"html/template"
	"io"
	"time"

	"github.com/valksor/go-mehrhof/internal/sandbox"
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
	quick     *template.Template
	partials  map[string]*template.Template
}

// templateFuncs returns the function map for templates.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"stateIcon":    stateIcon,
		"stateColor":   stateColor,
		"stateBadge":   stateBadge,
		"specIcon":     specIcon,
		"specColor":    specColor,
		"timeAgo":      timeAgo,
		"formatCost":   formatCost,
		"formatNumber": formatNumber,
		"formatPct":    formatPct,
		"labelColor":   labelColor,
		"mul": func(a, b float64) float64 {
			return a * b
		},
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
		"templates/partials/labels.html",
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

	// Load quick template
	quick, err := template.New("quick.html").Funcs(templateFuncs()).ParseFS(
		templateFS,
		"templates/base.html",
		"templates/quick.html",
	)
	if err != nil {
		return nil, err
	}
	t.quick = quick

	// Load partials
	partialNames := []string{"task_card", "actions", "specification", "question", "costs", "workspace_stats", "card", "modal", "input", "skeleton", "labels"}
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

// RenderQuick renders the quick tasks page.
func (t *Templates) RenderQuick(w io.Writer, data QuickData) error {
	return t.quick.ExecuteTemplate(w, "base", data)
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
	WorkspaceStats   *WorkspaceStatsData // Workspace-level stats when no active task
}

// TaskData holds task information for display.
type TaskData struct {
	ID            string
	Title         string
	State         string
	Branch        string
	Worktree      string
	Started       time.Time
	Ref           string
	SandboxActive bool
	Labels        []string
}

// GuideData holds guidance information.
type GuideData struct {
	NextActions    []ActionData
	SandboxEnabled bool
	SandboxActive  bool
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
	TotalCostUSD    float64
	TotalTokens     int
	InputTokens     int
	OutputTokens    int
	CachedTokens    int
	CachedPercent   float64
	BudgetMaxCost   float64
	BudgetMaxTokens int
	BudgetOnLimit   string
	BudgetWarningAt float64
	BudgetPercent   float64
	BudgetWarned    bool
	BudgetLimitHit  bool
}

// WorkspaceStatsData holds workspace-level statistics for the dashboard.
type WorkspaceStatsData struct {
	TotalTasks   int                // Total number of tasks
	ByState      map[string]int     // Tasks grouped by state
	ByStatePct   map[string]float64 // Percentage of tasks per state
	TotalCostUSD float64            // Total cost across all tasks
	TotalTokens  int                // Total tokens across all tasks
	InputTokens  int                // Total input tokens
	OutputTokens int                // Total output tokens
	CachedTokens int                // Total cached tokens
	MonthlySpent float64            // Monthly budget spent
	MonthlyMax   float64            // Monthly budget max
	MonthlyPct   float64            // Monthly budget percentage
	HasMonthly   bool               // Whether monthly budget is configured
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
	SandboxStatus    sandbox.Status            // Sandbox status information
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

// QuickData holds data for the quick tasks template.
type QuickData struct {
	Mode             string
	AuthEnabled      bool
	IsGlobalMode     bool
	CanSwitchProject bool
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
	case "paused":
		return "||"
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
	case "paused":
		return "text-amber-600"
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
	case "paused":
		return "Paused (budget limit)"
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

// formatNumber formats a large number with K/M/B suffixes.
func formatNumber(n int) string {
	if n < 1000 {
		return itoa(n)
	}
	if n < 1_000_000 {
		// Show as XK (e.g., 1.2K)
		k := float64(n) / 1000

		return formatFloat(k) + "K"
	}
	if n < 1_000_000_000 {
		// Show as XM (e.g., 1.2M)
		m := float64(n) / 1_000_000

		return formatFloat(m) + "M"
	}
	// Show as XB
	b := float64(n) / 1_000_000_000

	return formatFloat(b) + "B"
}

// formatFloat formats a float with 1 decimal place, without fmt.Sprintf.
func formatFloat(f float64) string {
	whole := int(f)
	tenths := int((f - float64(whole)) * 10)
	if tenths < 0 {
		tenths = -tenths
	}
	result := itoa(whole) + "."
	result += string(rune('0' + tenths))

	return result
}

// formatPct formats a percentage with optional decimal.
func formatPct(pct float64) string {
	if pct < 1 {
		// Show as "<1%" for small values
		return "<1%"
	}
	whole := int(pct)
	if pct == float64(whole) {
		return itoa(whole) + "%"
	}
	// One decimal place
	tenths := int((pct - float64(whole)) * 10)

	return itoa(whole) + "." + string(rune('0'+tenths)) + "%"
}

// labelColor returns a CSS color class for a label based on its hash.
// This ensures consistent colors for the same label across renders.
func labelColor(label string) string {
	colors := []string{
		"bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
		"bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
		"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
		"bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
		"bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200",
		"bg-pink-100 text-pink-800 dark:bg-pink-900 dark:text-pink-200",
		"bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200",
		"bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200",
		"bg-teal-100 text-teal-800 dark:bg-teal-900 dark:text-teal-200",
		"bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200",
	}

	// Simple hash of label string
	hash := 0
	for _, c := range label {
		hash += int(c)
	}

	return colors[hash%len(colors)]
}
