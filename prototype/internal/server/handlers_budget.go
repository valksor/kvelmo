package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// handleBudgetMonthlyStatus returns the monthly budget status as HTML partial.
func (s *Server) handleBudgetMonthlyStatus(w http.ResponseWriter, r *http.Request) {
	var cfg *storage.WorkspaceConfig
	var state *storage.MonthlyBudgetState

	if s.config.Conductor != nil {
		ws := s.config.Conductor.GetWorkspace()
		if ws != nil {
			var err error
			cfg, err = ws.LoadConfig()
			if err != nil {
				// Fall back to defaults on error
				cfg = storage.NewDefaultWorkspaceConfig()
			}

			state, err = ws.LoadMonthlyBudgetState()
			if err != nil {
				// No state file yet - that's ok, state will be nil
				state = nil
			}
		}
	}

	// Use defaults if no config loaded (no workspace case)
	if cfg == nil {
		cfg = storage.NewDefaultWorkspaceConfig()
	}

	// Show budget status with config (defaults if no workspace, 0 spent if no state)
	s.writeBudgetStatusHTML(w, cfg, state, "")
}

// writeBudgetStatusHTML renders the monthly budget status as HTML.
func (s *Server) writeBudgetStatusHTML(w http.ResponseWriter, cfg *storage.WorkspaceConfig, state *storage.MonthlyBudgetState, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if errMsg != "" {
		_, _ = w.Write([]byte(`<div class="text-error-600 dark:text-error-400 text-sm">` + errMsg + `</div>`))

		return
	}

	// Check if the budget is configured
	maxCost := float64(0)
	warningAt := float64(0.8)

	if cfg != nil {
		maxCost = cfg.Budget.Monthly.MaxCost
		if cfg.Budget.Monthly.WarningAt > 0 {
			warningAt = cfg.Budget.Monthly.WarningAt
		}
	}

	if maxCost <= 0 {
		_, _ = w.Write([]byte(`
			<div class="text-center py-4">
				<p class="text-surface-500 dark:text-surface-400 text-sm">No monthly budget configured.</p>
				<p class="text-surface-400 dark:text-surface-500 text-xs mt-1">Set a monthly max cost above to enable tracking.</p>
			</div>
		`))

		return
	}

	// Get spending data
	spent := float64(0)
	warningSent := false
	monthLabel := time.Now().Format("January 2006")

	if state != nil {
		spent = state.Spent
		warningSent = state.WarningSent
		// Parse month if available
		if t, err := time.Parse("2006-01", state.Month); err == nil {
			monthLabel = t.Format("January 2006")
		}
	}

	// Calculate percentage
	pct := float64(0)
	if maxCost > 0 {
		pct = (spent / maxCost) * 100
	}

	// Determine status color
	statusColor := "success"
	if pct >= 100 {
		statusColor = "error"
	} else if pct >= warningAt*100 {
		statusColor = "warning"
	}

	html := fmt.Sprintf(`
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<span class="text-sm font-medium text-surface-700 dark:text-surface-300">%s</span>
				<span class="text-sm text-surface-500 dark:text-surface-400">%s / %s</span>
			</div>
			<div class="w-full bg-surface-200 dark:bg-surface-700 rounded-full h-3">
				<div class="bg-%s-500 h-3 rounded-full transition-all duration-300" style="width: %s"></div>
			</div>
			<div class="flex items-center justify-between text-xs text-surface-500 dark:text-surface-400">
				<span>%s used</span>`,
		monthLabel,
		views.FormatCost(spent),
		views.FormatCost(maxCost),
		statusColor,
		views.FormatPercent(pct),
		views.FormatPercent(pct),
	)

	if warningSent {
		html += `<span class="text-warning-600 dark:text-warning-400">Warning sent</span>`
	}

	html += `
			</div>
		</div>
	`

	_, _ = w.Write([]byte(html))
}

// handleBudgetMonthlyReset resets the monthly budget tracking.
func (s *Server) handleBudgetMonthlyReset(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Reset monthly budget state
	if err := ws.ResetMonthlyBudget(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to reset budget: "+err.Error())

		return
	}

	// Return updated status HTML
	s.handleBudgetMonthlyStatus(w, r)
}
