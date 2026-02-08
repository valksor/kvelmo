package server

import (
	"net/http"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// handleBudgetMonthlyStatus returns the monthly budget status as HTML partial.
func (s *Server) handleBudgetMonthlyStatus(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor != nil {
		s.handleViaRouter(CommandRoute{
			Command: "budget-monthly",
		})(w, r)

		return
	}

	var cfg *storage.WorkspaceConfig
	var state *storage.MonthlyBudgetState

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
		enabled = cfg.Budget.Enabled
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
