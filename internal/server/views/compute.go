package views

import (
	"sort"
	"time"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/security"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// ComputePageData creates the common page data from server configuration.
func ComputePageData(mode string, isGlobalMode, authEnabled, canSwitchProject bool, currentUser string) PageData {
	return PageData{
		Mode:             mode,
		IsGlobalMode:     isGlobalMode,
		IsProjectMode:    !isGlobalMode,
		AuthEnabled:      authEnabled,
		CanSwitchProject: canSwitchProject,
		CurrentUser:      currentUser,
		Events:           DefaultEventNames(),
	}
}

// ComputeDashboard creates the full dashboard data.
func ComputeDashboard(c *conductor.Conductor, ws *storage.Workspace, pageData PageData) DashboardData {
	data := DashboardData{
		PageData: pageData,
	}

	if pageData.IsGlobalMode {
		data.Projects = ComputeProjects()

		return data
	}

	// Project mode - compute each section independently
	data.Stats = ComputeStats(ws)
	data.ActiveWork = ComputeActiveWork(c, ws)
	data.Actions = ComputeActions(data.ActiveWork, ws)
	data.RecentTasks = ComputeRecentTasks(ws, 5)

	if data.ActiveWork != nil {
		data.Specifications = ComputeSpecifications(ws, data.ActiveWork.ID)
		data.Question = ComputeQuestion(ws, data.ActiveWork.ID)
		data.Costs = ComputeCosts(ws, data.ActiveWork.ID)
	}

	return data
}

// ComputeActiveWork detects the currently active work item.
// Currently only supports regular tasks - quick tasks and planning projects
// are not yet implemented in the conductor.
func ComputeActiveWork(c *conductor.Conductor, ws *storage.Workspace) *ActiveWorkData {
	if c == nil {
		return nil
	}

	// Check for regular task
	activeTask := c.GetActiveTask()
	if activeTask == nil {
		return nil
	}

	work := c.GetTaskWork()

	display := GetStateDisplay(activeTask.State)
	active := &ActiveWorkData{
		Type:       WorkTypeTask,
		ID:         activeTask.ID,
		Ref:        activeTask.Ref,
		State:      activeTask.State,
		Branch:     activeTask.Branch,
		Worktree:   activeTask.WorktreePath,
		StartedAt:  activeTask.Started,
		Started:    FormatTimeAgo(activeTask.Started),
		StateIcon:  display.Icon,
		StateBadge: display.Badge,
		StateColor: display.Color,
		BarColor:   display.BarColor,
	}

	if work != nil {
		active.Title = work.Metadata.Title
		// Description is not stored in WorkMetadata - could be derived from source content
		active.Labels = computeLabels(work.Metadata.Labels)
	}

	// Check for pending question
	if ws != nil && ws.HasPendingQuestion(activeTask.ID) {
		active.HasQuestion = true
	}

	// Check for specs
	if ws != nil {
		if specs, err := ws.ListSpecifications(activeTask.ID); err == nil && len(specs) > 0 {
			active.HasSpecs = true
		}
	}

	// Compute hierarchical context
	active.Hierarchy = ComputeHierarchyContext(c, ws, activeTask.ID)

	return active
}

// ComputeActions determines available actions based on current work state.
func ComputeActions(active *ActiveWorkData, ws *storage.Workspace) []ActionData {
	if active == nil {
		// No active work - show start options
		return []ActionData{
			{
				Command:     "start",
				Label:       "Start Task",
				Endpoint:    "/api/v1/workflow/start",
				Method:      "POST",
				ButtonClass: BtnPrimary,
			},
			{
				Command:     "quick",
				Label:       "Quick Task",
				Endpoint:    "/ui/quick",
				Method:      "GET",
				ButtonClass: BtnSecondary,
			},
		}
	}

	switch active.State {
	case StateIdle:
		if active.HasSpecs {
			return []ActionData{
				{
					Command:     "implement",
					Label:       "Implement",
					Endpoint:    "/api/v1/workflow/implement",
					Method:      "POST",
					ButtonClass: BtnPrimary,
					HasOptions:  true,
				},
				{
					Command:     "sync",
					Label:       "Sync",
					Endpoint:    "/api/v1/workflow/sync",
					Method:      "POST",
					ButtonClass: BtnSecondary,
					Tooltip:     "Sync task with external provider",
				},
				{
					Command:     "simplify",
					Label:       "Simplify",
					Endpoint:    "/api/v1/workflow/simplify",
					Method:      "POST",
					ButtonClass: BtnSecondary,
					Tooltip:     "Simplify specifications",
				},
				{
					Command:     "abandon",
					Label:       "Abandon",
					Endpoint:    "/api/v1/workflow/abandon",
					Method:      "POST",
					ButtonClass: BtnDanger,
					Dangerous:   true,
					Confirm:     "Abandon this task? This cannot be undone.",
				},
			}
		}

		return []ActionData{
			{
				Command:     "plan",
				Label:       "Plan",
				Endpoint:    "/api/v1/workflow/plan",
				Method:      "POST",
				ButtonClass: BtnPrimary,
			},
			{
				Command:     "abandon",
				Label:       "Abandon",
				Endpoint:    "/api/v1/workflow/abandon",
				Method:      "POST",
				ButtonClass: BtnDanger,
				Dangerous:   true,
				Confirm:     "Abandon this task? This cannot be undone.",
			},
		}

	case StatePlanning, StateImplementing, StateReviewing:
		return []ActionData{
			{
				Command:     "undo",
				Label:       "Undo",
				Endpoint:    "/api/v1/workflow/undo",
				Method:      "POST",
				ButtonClass: BtnSecondary,
			},
		}

	case StateDone:
		return []ActionData{
			{
				Command:     "finish",
				Label:       "Finish & Push",
				Endpoint:    "/api/v1/workflow/finish",
				Method:      "POST",
				ButtonClass: BtnSuccess,
			},
		}

	case StateWaiting:
		return []ActionData{
			{
				Command:     "continue",
				Label:       "Continue",
				Endpoint:    "/api/v1/workflow/continue",
				Method:      "POST",
				ButtonClass: BtnPrimary,
				Tooltip:     "Resume workflow with auto-execution",
			},
			{
				Command:     "undo",
				Label:       "Undo",
				Endpoint:    "/api/v1/workflow/undo",
				Method:      "POST",
				ButtonClass: BtnSecondary,
			},
		}

	case StatePaused:
		return []ActionData{
			{
				Command:     "budget",
				Label:       "Review Budget",
				Endpoint:    "/ui/settings#budget",
				Method:      "GET",
				ButtonClass: BtnWarning,
			},
			{
				Command:     "resume",
				Label:       "Resume",
				Endpoint:    "/api/v1/workflow/resume",
				Method:      "POST",
				ButtonClass: BtnPrimary,
			},
		}

	case StateFailed:
		return []ActionData{
			{
				Command:     "undo",
				Label:       "Undo & Retry",
				Endpoint:    "/api/v1/workflow/undo",
				Method:      "POST",
				ButtonClass: BtnSecondary,
			},
			{
				Command:     "abandon",
				Label:       "Abandon",
				Endpoint:    "/api/v1/workflow/abandon",
				Method:      "POST",
				ButtonClass: BtnDanger,
				Dangerous:   true,
				Confirm:     "Abandon this task? This cannot be undone.",
			},
		}
	}

	return nil
}

// ComputeStats aggregates workspace-level statistics.
func ComputeStats(ws *storage.Workspace) *StatsData {
	stats := &StatsData{
		StateLines: make([]StateLineData, 0),
	}

	if ws == nil {
		return stats
	}

	taskIDs, err := ws.ListWorks()
	if err != nil || len(taskIDs) == 0 {
		return stats
	}

	stats.TotalTasks = len(taskIDs)
	stateCounts := make(map[string]int)
	var totalCost float64
	var totalIn, totalOut, totalCached int

	for _, taskID := range taskIDs {
		work, err := ws.LoadWork(taskID)
		if err != nil {
			continue
		}

		state := work.Metadata.State
		if state == "" {
			state = StateIdle
		}
		stateCounts[state]++

		totalCost += work.Costs.TotalCostUSD
		totalIn += work.Costs.TotalInputTokens
		totalOut += work.Costs.TotalOutputTokens
		totalCached += work.Costs.TotalCachedTokens
	}

	totalTokens := totalIn + totalOut

	// Pre-format values
	stats.TotalCost = FormatCost(totalCost)
	stats.TotalTokens = FormatNumber(totalTokens)
	stats.CachedTokens = FormatNumber(totalCached)
	if totalTokens > 0 {
		stats.CachedPct = FormatPercent(float64(totalCached) / float64(totalTokens) * 100)
	}

	// Build sorted state lines
	for state, count := range stateCounts {
		display := GetStateDisplay(state)
		pct := float64(count) / float64(stats.TotalTasks) * 100
		stats.StateLines = append(stats.StateLines, StateLineData{
			State:    state,
			Icon:     display.Icon,
			Badge:    display.Badge,
			Count:    count,
			Percent:  FormatPercent(pct),
			Color:    display.Color,
			BarColor: display.BarColor,
		})
	}

	// Sort by count descending
	sort.Slice(stats.StateLines, func(i, j int) bool {
		return stats.StateLines[i].Count > stats.StateLines[j].Count
	})

	// Load monthly budget info
	cfg, _ := ws.LoadConfig()
	if cfg != nil && cfg.Budget.Monthly.MaxCost > 0 {
		if state, err := ws.LoadMonthlyBudgetState(); err == nil {
			stats.HasMonthly = true
			stats.MonthlySpent = FormatCost(state.Spent)
			stats.MonthlyMax = FormatCost(cfg.Budget.Monthly.MaxCost)
			stats.MonthlyPct = (state.Spent / cfg.Budget.Monthly.MaxCost) * 100
			stats.MonthlyColor = BudgetColor(stats.MonthlyPct)
			stats.MonthlyMonth = FormatMonth(time.Now())
		}
	}

	return stats
}

// ComputeSpecifications builds the specifications data for a task.
func ComputeSpecifications(ws *storage.Workspace, taskID string) *SpecificationsData {
	if ws == nil {
		return nil
	}

	specList, err := ws.ListSpecificationsWithStatus(taskID)
	if err != nil || len(specList) == 0 {
		return nil
	}

	specs := &SpecificationsData{
		Items: make([]SpecItemData, 0, len(specList)),
		Total: len(specList),
	}

	for _, spec := range specList {
		status := spec.Status
		if status == "" {
			status = SpecStatusPending
		}

		// Load full description
		description, _ := ws.LoadSpecification(taskID, spec.Number)

		statusDisplay := GetSpecStatusDisplay(status)

		item := SpecItemData{
			Number:      spec.Number,
			Name:        "specification-" + FormatNumber(spec.Number),
			Title:       spec.Title,
			Description: description,
			Component:   spec.Component,
			Status:      status,
			StatusIcon:  statusDisplay.Icon,
			StatusColor: statusDisplay.Color,
			IsCompleted: status == SpecStatusCompleted || status == storage.SpecificationStatusDone,
			IsActive:    status == SpecStatusActive || status == "implementing",
		}

		if !spec.CreatedAt.IsZero() {
			item.CreatedAt = FormatDateTime(spec.CreatedAt)
		}
		if !spec.CompletedAt.IsZero() {
			item.CompletedAt = FormatDateTime(spec.CompletedAt)
		}

		if item.IsCompleted {
			specs.Done++
		}

		specs.Items = append(specs.Items, item)
	}

	if specs.Total > 0 {
		specs.Progress = float64(specs.Done) / float64(specs.Total) * 100
	}

	return specs
}

// ComputeQuestion retrieves the pending question for a task.
func ComputeQuestion(ws *storage.Workspace, taskID string) *QuestionData {
	if ws == nil || !ws.HasPendingQuestion(taskID) {
		return nil
	}

	question, err := ws.LoadPendingQuestion(taskID)
	if err != nil {
		return nil
	}

	data := &QuestionData{
		Question: question.Question,
		TaskID:   taskID,
		Options:  make([]OptionData, 0, len(question.Options)),
	}

	for _, opt := range question.Options {
		data.Options = append(data.Options, OptionData{
			Label:       opt.Label,
			Value:       opt.Label, // QuestionOption has Label+Description, use Label as value
			Description: opt.Description,
		})
	}

	return data
}

// ComputeCosts builds the cost data for a task.
func ComputeCosts(ws *storage.Workspace, taskID string) *CostsData {
	if ws == nil {
		return nil
	}

	work, err := ws.LoadWork(taskID)
	if err != nil {
		return nil
	}

	costs := work.Costs
	total := costs.TotalInputTokens + costs.TotalOutputTokens

	if total == 0 {
		return nil
	}

	data := &CostsData{
		TotalCost:    FormatCost(costs.TotalCostUSD),
		TotalTokens:  FormatNumber(total),
		InputTokens:  FormatNumber(costs.TotalInputTokens),
		OutputTokens: FormatNumber(costs.TotalOutputTokens),
		CachedTokens: FormatNumber(costs.TotalCachedTokens),
	}

	if total > 0 {
		data.CachedPct = FormatPercent(float64(costs.TotalCachedTokens) / float64(total) * 100)
	}

	// Get budget config
	var budget storage.BudgetConfig
	if cfg, err := ws.LoadConfig(); err == nil {
		budget = cfg.Budget.PerTask
	}
	if work.Budget != nil {
		budget = *work.Budget
	}

	// Calculate budget usage
	if budget.MaxCost > 0 {
		data.HasBudget = true
		data.BudgetType = "cost"
		data.BudgetUsed = FormatCost(costs.TotalCostUSD)
		data.BudgetMax = FormatCost(budget.MaxCost)
		data.BudgetPct = (costs.TotalCostUSD / budget.MaxCost) * 100
		data.BudgetColor = BudgetColor(data.BudgetPct)
	} else if budget.MaxTokens > 0 {
		data.HasBudget = true
		data.BudgetType = "tokens"
		data.BudgetUsed = FormatNumber(total)
		data.BudgetMax = FormatNumber(budget.MaxTokens)
		data.BudgetPct = (float64(total) / float64(budget.MaxTokens)) * 100
		data.BudgetColor = BudgetColor(data.BudgetPct)
	}

	if work.BudgetStatus != nil {
		data.BudgetWarned = work.BudgetStatus.Warned
		data.BudgetLimitHit = work.BudgetStatus.LimitHit
	}

	// Add step breakdown if available
	for step, stepCosts := range costs.ByStep {
		stepTotal := stepCosts.InputTokens + stepCosts.OutputTokens
		data.Steps = append(data.Steps, StepCostData{
			Name:         step,
			InputTokens:  FormatNumber(stepCosts.InputTokens),
			OutputTokens: FormatNumber(stepCosts.OutputTokens),
			CachedTokens: FormatNumber(stepCosts.CachedTokens),
			TotalTokens:  FormatNumber(stepTotal),
			Cost:         FormatCost(stepCosts.CostUSD),
			Calls:        stepCosts.Calls,
		})
	}

	return data
}

// ComputeRecentTasks builds the recent tasks list.
func ComputeRecentTasks(ws *storage.Workspace, limit int) []RecentTaskData {
	if ws == nil {
		return nil
	}

	taskIDs, err := ws.ListWorks()
	if err != nil || len(taskIDs) == 0 {
		return nil
	}

	// Load tasks with their update times
	type taskWithTime struct {
		id   string
		work *storage.TaskWork
		time time.Time
	}

	tasks := make([]taskWithTime, 0, len(taskIDs))
	for _, id := range taskIDs {
		work, err := ws.LoadWork(id)
		if err != nil {
			continue
		}
		t := work.Metadata.UpdatedAt
		if t.IsZero() {
			t = work.Metadata.CreatedAt
		}
		tasks = append(tasks, taskWithTime{id: id, work: work, time: t})
	}

	// Sort by time descending
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].time.After(tasks[j].time)
	})

	// Take limit
	if len(tasks) > limit {
		tasks = tasks[:limit]
	}

	result := make([]RecentTaskData, 0, len(tasks))
	for _, t := range tasks {
		state := t.work.Metadata.State
		if state == "" {
			state = StateIdle
		}
		display := GetStateDisplay(state)

		title := t.work.Metadata.Title
		if title == "" {
			title = t.id
		}

		result = append(result, RecentTaskData{
			ID:         t.id,
			ShortID:    ShortID(t.id),
			Title:      title,
			State:      state,
			StateIcon:  display.Icon,
			StateColor: display.Color,
			TimeAgo:    FormatTimeAgo(t.time),
			Ref:        t.work.Source.Ref, // Ref is in SourceInfo, not WorkMetadata
		})
	}

	return result
}

// ComputeProjects loads the project list for global mode.
func ComputeProjects() []ProjectData {
	registry, err := storage.LoadRegistry()
	if err != nil {
		return nil
	}

	projects := registry.List()
	result := make([]ProjectData, 0, len(projects))

	for _, p := range projects {
		result = append(result, ProjectData{
			ID:         p.ID,
			Name:       p.Name,
			Path:       p.Path,
			RemoteURL:  p.RemoteURL,
			LastAccess: FormatTimeAgo(p.LastAccess),
		})
	}

	return result
}

// ComputeGuide builds the guide data for CLI-style help.
func ComputeGuide(c *conductor.Conductor, ws *storage.Workspace) *GuideData {
	guide := &GuideData{
		NextActions: make([]GuideActionData, 0),
	}

	if c == nil {
		return guide
	}

	activeTask := c.GetActiveTask()
	if activeTask == nil {
		guide.HasTask = false

		return guide
	}

	guide.HasTask = true
	guide.TaskID = activeTask.ID
	guide.State = activeTask.State

	// Get title
	if work := c.GetTaskWork(); work != nil {
		guide.Title = work.Metadata.Title
	}

	// Count specs
	if ws != nil {
		if specs, err := ws.ListSpecifications(activeTask.ID); err == nil {
			guide.Specifications = len(specs)
		}
	}

	// Check for pending question
	if ws != nil {
		guide.PendingQuestion = ComputeQuestion(ws, activeTask.ID)
	}

	// Build actions
	active := ComputeActiveWork(c, ws)
	actions := ComputeActions(active, ws)

	for i, action := range actions {
		guide.NextActions = append(guide.NextActions, GuideActionData{
			Command:     action.Command,
			Description: action.Label,
			Endpoint:    action.Endpoint,
			Primary:     i == 0, // First action is primary
		})
	}

	return guide
}

// Helper functions

func computeLabels(labels []string) []LabelData {
	result := make([]LabelData, 0, len(labels))
	for _, label := range labels {
		result = append(result, LabelData{
			Text:  label,
			Color: LabelColor(label),
		})
	}

	return result
}

// ComputeSettingsProjects converts storage.ProjectMetadata slice to ProjectData slice.
func ComputeSettingsProjects(projects []storage.ProjectMetadata) []ProjectData {
	result := make([]ProjectData, 0, len(projects))
	for _, p := range projects {
		result = append(result, ProjectData{
			ID:         p.ID,
			Name:       p.Name,
			Path:       p.Path,
			RemoteURL:  p.RemoteURL,
			LastAccess: FormatTimeAgo(p.LastAccess),
		})
	}

	return result
}

// ComputeProjectInfo converts security.ProjectInfo to ProjectInfoData for the UI.
func ComputeProjectInfo(dir string) *ProjectInfoData {
	info := security.DetectProject(dir)

	// Convert languages to strings
	languages := make([]string, 0, len(info.Languages))
	for _, lang := range info.Languages {
		languages = append(languages, lang.DisplayName())
	}

	// Get applicable scanners
	applicableScanners := security.GetApplicableScanners(info)
	scanners := make([]ScannerInfoData, 0, len(applicableScanners))
	for _, scanner := range applicableScanners {
		langStrings := make([]string, 0, len(scanner.Languages))
		for _, lang := range scanner.Languages {
			langStrings = append(langStrings, lang.DisplayName())
		}
		scanners = append(scanners, ScannerInfoData{
			Name:           scanner.Name,
			DisplayName:    scanner.DisplayName,
			Description:    scanner.Description,
			Type:           scanner.Type,
			Languages:      langStrings,
			InstallCommand: scanner.InstallCommand,
			Requires:       scanner.Requires,
			AlwaysShow:     scanner.AlwaysShow,
		})
	}

	return &ProjectInfoData{
		Languages:          languages,
		HasGoMod:           info.HasGoMod,
		HasPackageJSON:     info.HasPackageJSON,
		HasPackageLockJSON: info.HasPackageLockJSON,
		HasYarnLock:        info.HasYarnLock,
		HasTSConfig:        info.HasTSConfig,
		HasPyProjectTOML:   info.HasPyProjectTOML,
		HasRequirementsTXT: info.HasRequirementsTXT,
		HasSetupPy:         info.HasSetupPy,
		HasPipfile:         info.HasPipfile,
		HasComposerJSON:    info.HasComposerJSON,
		HasGemfile:         info.HasGemfile,
		HasCargoTOML:       info.HasCargoTOML,
		ApplicableScanners: scanners,
	}
}

// ComputeHierarchyContext computes hierarchical task context for display.
func ComputeHierarchyContext(c *conductor.Conductor, ws *storage.Workspace, taskID string) *HierarchyData {
	if c == nil || ws == nil {
		return nil
	}

	// Load task work to check for stored hierarchy info
	work, err := ws.LoadWork(taskID)
	if err != nil || work.Hierarchy == nil {
		return nil
	}

	hierarchy := &HierarchyData{}

	// If we have stored hierarchy info, use it (for now we'll return empty data
	// since we'd need to make API calls to get full parent/sibling details)
	// This could be enhanced in the future to make provider calls
	if work.Hierarchy.ParentID != "" {
		hierarchy.Parent = &ParentTaskData{
			ID:    work.Hierarchy.ParentID,
			Title: work.Hierarchy.ParentTitle,
		}
	}

	if len(work.Hierarchy.SiblingIDs) > 0 {
		hierarchy.Siblings = make([]*SiblingTaskData, 0, len(work.Hierarchy.SiblingIDs))
		for _, siblingID := range work.Hierarchy.SiblingIDs {
			hierarchy.Siblings = append(hierarchy.Siblings, &SiblingTaskData{
				ID: siblingID,
			})
		}
	}

	return hierarchy
}
