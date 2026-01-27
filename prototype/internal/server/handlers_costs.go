package server

import (
	"net/http"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// handleGetTaskCosts returns the costs for a specific task.
func (s *Server) handleGetTaskCosts(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	// Get task ID from path
	taskID := r.PathValue("id")
	if taskID == "" {
		s.writeError(w, http.StatusBadRequest, "task ID is required")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	work, err := ws.LoadWork(taskID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "task not found: "+err.Error())

		return
	}

	costs := work.Costs
	totalTokens := costs.TotalInputTokens + costs.TotalOutputTokens

	cachedPercent := 0.0
	if totalTokens > 0 && costs.TotalCachedTokens > 0 {
		cachedPercent = float64(costs.TotalCachedTokens) / float64(totalTokens) * 100
	}

	resp := taskCostResponse{
		TaskID:        taskID,
		Title:         work.Metadata.Title,
		TotalTokens:   totalTokens,
		InputTokens:   costs.TotalInputTokens,
		OutputTokens:  costs.TotalOutputTokens,
		CachedTokens:  costs.TotalCachedTokens,
		CachedPercent: cachedPercent,
		TotalCostUSD:  costs.TotalCostUSD,
	}
	if cfg, err := ws.LoadConfig(); err == nil {
		resp.Budget = buildBudgetInfo(work, cfg)
	}

	// Add by-step breakdown if available
	if len(costs.ByStep) > 0 {
		resp.ByStep = make(map[string]stepCost)
		for step, stats := range costs.ByStep {
			resp.ByStep[step] = stepCost{
				InputTokens:  stats.InputTokens,
				OutputTokens: stats.OutputTokens,
				CachedTokens: stats.CachedTokens,
				TotalTokens:  stats.InputTokens + stats.OutputTokens,
				CostUSD:      stats.CostUSD,
				Calls:        stats.Calls,
			}
		}
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// handleGetAllCosts returns costs for all tasks with optional summary.
func (s *Server) handleGetAllCosts(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	taskIDs, err := ws.ListWorks()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list tasks: "+err.Error())

		return
	}

	var tasks []taskCostResponse
	var grandTotalInput, grandTotalOutput, grandTotalCached int
	var grandTotalCost float64

	cfg, _ := ws.LoadConfig()
	for _, taskID := range taskIDs {
		work, err := ws.LoadWork(taskID)
		if err != nil {
			continue
		}

		costs := work.Costs
		totalTokens := costs.TotalInputTokens + costs.TotalOutputTokens

		cachedPercent := 0.0
		if totalTokens > 0 && costs.TotalCachedTokens > 0 {
			cachedPercent = float64(costs.TotalCachedTokens) / float64(totalTokens) * 100
		}

		title := work.Metadata.Title
		if title == "" {
			title = "(untitled)"
		}

		taskCost := taskCostResponse{
			TaskID:        taskID,
			Title:         title,
			TotalTokens:   totalTokens,
			InputTokens:   costs.TotalInputTokens,
			OutputTokens:  costs.TotalOutputTokens,
			CachedTokens:  costs.TotalCachedTokens,
			CachedPercent: cachedPercent,
			TotalCostUSD:  costs.TotalCostUSD,
		}
		if cfg != nil {
			taskCost.Budget = buildBudgetInfo(work, cfg)
		}

		// Add by-step breakdown
		if len(costs.ByStep) > 0 {
			taskCost.ByStep = make(map[string]stepCost)
			for step, stats := range costs.ByStep {
				taskCost.ByStep[step] = stepCost{
					InputTokens:  stats.InputTokens,
					OutputTokens: stats.OutputTokens,
					CachedTokens: stats.CachedTokens,
					TotalTokens:  stats.InputTokens + stats.OutputTokens,
					CostUSD:      stats.CostUSD,
					Calls:        stats.Calls,
				}
			}
		}

		tasks = append(tasks, taskCost)

		grandTotalInput += costs.TotalInputTokens
		grandTotalOutput += costs.TotalOutputTokens
		grandTotalCached += costs.TotalCachedTokens
		grandTotalCost += costs.TotalCostUSD
	}

	var monthly *monthlyBudgetInfo
	if cfg != nil && cfg.Budget.Monthly.MaxCost > 0 {
		if state, err := ws.LoadMonthlyBudgetState(); err == nil {
			monthly = &monthlyBudgetInfo{
				Month:       state.Month,
				Spent:       state.Spent,
				MaxCost:     cfg.Budget.Monthly.MaxCost,
				WarningAt:   cfg.Budget.Monthly.WarningAt,
				WarningSent: state.WarningSent,
			}
		}
	}

	s.writeJSON(w, http.StatusOK, allCostsResponse{
		Tasks: tasks,
		GrandTotal: grandTotal{
			InputTokens:  grandTotalInput,
			OutputTokens: grandTotalOutput,
			TotalTokens:  grandTotalInput + grandTotalOutput,
			CachedTokens: grandTotalCached,
			CostUSD:      grandTotalCost,
		},
		Monthly: monthly,
	})
}

func buildBudgetInfo(work *storage.TaskWork, cfg *storage.WorkspaceConfig) *budgetInfo {
	if work == nil || cfg == nil {
		return nil
	}

	budget := cfg.Budget.PerTask
	if work.Budget != nil {
		budget = *work.Budget
	}

	info := &budgetInfo{
		MaxTokens: budget.MaxTokens,
		MaxCost:   budget.MaxCost,
		Currency:  budget.Currency,
		OnLimit:   budget.OnLimit,
		WarningAt: budget.WarningAt,
	}
	if work.BudgetStatus != nil {
		info.Warned = work.BudgetStatus.Warned
		info.LimitHit = work.BudgetStatus.LimitHit
	}

	if info.MaxTokens == 0 && info.MaxCost == 0 && info.OnLimit == "" && info.WarningAt == 0 {
		return nil
	}

	return info
}
