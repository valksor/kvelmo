package server

import (
	"net/http"

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

	s.writeBudgetStatusJSON(w, cfg, state)
}

// writeBudgetStatusJSON renders the monthly budget status as JSON for API clients.
func (s *Server) writeBudgetStatusJSON(w http.ResponseWriter, cfg *storage.WorkspaceConfig, state *storage.MonthlyBudgetState) {
	enabled := false
	maxCost := float64(0)
	warningAt := float64(0.8)
	currency := "USD"

	if cfg != nil {
		enabled = cfg.Budget.Monthly.Enabled
		maxCost = cfg.Budget.Monthly.MaxCost
		if cfg.Budget.Monthly.WarningAt > 0 {
			warningAt = cfg.Budget.Monthly.WarningAt
		}
		if cfg.Budget.Monthly.Currency != "" {
			currency = cfg.Budget.Monthly.Currency
		}
	}

	response := map[string]any{
		"enabled": enabled,
	}

	if enabled {
		spent := float64(0)
		warningSent := false

		if state != nil {
			spent = state.Spent
			warningSent = state.WarningSent
		}

		response["max_cost"] = maxCost
		response["spent"] = spent
		response["remaining"] = maxCost - spent
		response["currency"] = currency
		response["warning_at"] = warningAt
		response["warned"] = warningSent
		response["limit_hit"] = spent >= maxCost
	}

	s.writeJSON(w, http.StatusOK, response)
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
