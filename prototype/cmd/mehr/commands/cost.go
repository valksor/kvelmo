package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/cost"
	"github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/storage"
	tkdisplay "github.com/valksor/go-toolkit/display"
)

var (
	costByStep   bool
	costAllTasks bool
	costSummary  bool
	costJSON     bool
	costChart    bool
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Show token usage and costs",
	Long: `Show token usage and API costs for tasks.

Displays input/output tokens, cached tokens, and estimated costs.
Costs are tracked per workflow step (planning, implementing, etc.).

Examples:
  mehr cost               # Show costs for active task
  mehr cost --breakdown   # Break down by workflow step
  mehr cost --all         # Show costs for all tasks
  mehr cost --summary     # Summary of all tasks
  mehr cost --chart       # Show ASCII charts
  mehr cost --json        # Output as JSON`,
	RunE: runCost,
}

func init() {
	rootCmd.AddCommand(costCmd)

	costCmd.Flags().BoolVar(&costByStep, "breakdown", false, "Show breakdown by workflow step")
	costCmd.Flags().BoolVar(&costAllTasks, "all", false, "Show costs for all tasks")
	costCmd.Flags().BoolVarP(&costSummary, "summary", "s", false, "Show summary of all tasks")
	costCmd.Flags().BoolVar(&costJSON, "json", false, "Output as JSON")
	costCmd.Flags().BoolVar(&costChart, "chart", false, "Show ASCII charts")
}

func runCost(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Resolve workspace root and git context
	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}

	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	if costAllTasks || costSummary {
		return showAllCosts(ws, costSummary)
	}

	// If in a worktree, auto-detect task from worktree path
	if res.IsWorktree {
		return showWorktreeCost(ws, res.Git)
	}

	return showActiveCost(ws)
}

// JSON output structures.
type jsonCostOutput struct {
	TaskID        string                  `json:"task_id"`
	Title         string                  `json:"title,omitempty"`
	TotalTokens   int                     `json:"total_tokens"`
	InputTokens   int                     `json:"input_tokens"`
	OutputTokens  int                     `json:"output_tokens"`
	CachedTokens  int                     `json:"cached_tokens"`
	CachedPercent float64                 `json:"cached_percent,omitempty"`
	TotalCostUSD  float64                 `json:"total_cost_usd"`
	ByStep        map[string]jsonStepCost `json:"by_step,omitempty"`
}

type jsonStepCost struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CachedTokens int     `json:"cached_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	Calls        int     `json:"calls"`
}

type jsonAllCostsOutput struct {
	Tasks      []jsonCostOutput `json:"tasks"`
	GrandTotal jsonGrandTotal   `json:"grand_total"`
}

type jsonGrandTotal struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	CachedTokens int     `json:"cached_tokens"`
	CostUSD      float64 `json:"cost_usd"`
}

type jsonSummaryOutput struct {
	TaskCount  int                     `json:"task_count"`
	GrandTotal jsonGrandTotal          `json:"grand_total"`
	ByStep     map[string]jsonStepCost `json:"by_step"`
}

func showWorktreeCost(ws *storage.Workspace, git interface{}) error {
	if git == nil {
		return errors.New("not in a worktree")
	}
	active, err := ws.FindTaskByWorktreePath(ws.Root())
	if err != nil {
		return fmt.Errorf("find task by worktree: %w", err)
	}

	if active == nil {
		fmt.Print(display.ErrorWithSuggestions(
			"No task associated with this worktree",
			[]display.Suggestion{
				{Command: "mehr start <reference>", Description: "Start a new task in this worktree"},
				{Command: "mehr list --all", Description: "View all tasks in workspace"},
			},
		))

		return nil
	}

	return showTaskCost(ws, active.ID, active.ID)
}

func showActiveCost(ws *storage.Workspace) error {
	if !ws.HasActiveTask() {
		fmt.Print(display.NoActiveTaskError())

		return nil
	}

	active, err := ws.LoadActiveTask()
	if err != nil {
		return fmt.Errorf("load active task: %w", err)
	}

	return showTaskCost(ws, active.ID, active.ID)
}

func showTaskCost(ws *storage.Workspace, taskID, label string) error {
	work, err := ws.LoadWork(taskID)
	if err != nil {
		return fmt.Errorf("load work: %w", err)
	}

	costs := work.Costs

	// Check if any costs have been recorded
	if costs.TotalInputTokens == 0 && costs.TotalOutputTokens == 0 {
		if costJSON {
			return outputJSON(jsonCostOutput{
				TaskID:       taskID,
				Title:        work.Metadata.Title,
				InputTokens:  0,
				OutputTokens: 0,
				TotalTokens:  0,
				CachedTokens: 0,
				TotalCostUSD: 0,
			})
		}
		fmt.Printf("No cost data available for task: %s\n", tkdisplay.Bold(label))
		fmt.Printf("\nRun 'mehr plan' or 'mehr implement' to generate costs.\n")

		return nil
	}

	// Calculate totals
	totalTokens := costs.TotalInputTokens + costs.TotalOutputTokens
	cachedPercent := 0.0
	if totalTokens > 0 && costs.TotalCachedTokens > 0 {
		cachedPercent = float64(costs.TotalCachedTokens) / float64(totalTokens) * 100
	}

	// JSON output
	if costJSON {
		output := jsonCostOutput{
			TaskID:       taskID,
			Title:        work.Metadata.Title,
			InputTokens:  costs.TotalInputTokens,
			OutputTokens: costs.TotalOutputTokens,
			TotalTokens:  totalTokens,
			CachedTokens: costs.TotalCachedTokens,
			TotalCostUSD: costs.TotalCostUSD,
		}
		if cachedPercent > 0 {
			output.CachedPercent = cachedPercent
		}

		// Add by-step breakdown if requested or if there are multiple steps
		if costByStep || len(costs.ByStep) > 1 {
			output.ByStep = make(map[string]jsonStepCost)
			for step, stats := range costs.ByStep {
				output.ByStep[step] = jsonStepCost{
					InputTokens:  stats.InputTokens,
					OutputTokens: stats.OutputTokens,
					CachedTokens: stats.CachedTokens,
					TotalTokens:  stats.InputTokens + stats.OutputTokens,
					CostUSD:      stats.CostUSD,
					Calls:        stats.Calls,
				}
			}
		}

		return outputJSON(output)
	}

	// Regular text output
	fmt.Printf("Costs for task: %s\n", tkdisplay.Bold(label))
	if work.Metadata.Title != "" {
		fmt.Printf("  Title: %s\n", work.Metadata.Title)
	}

	fmt.Printf("\n%s\n", tkdisplay.Bold("Total Usage:"))
	fmt.Printf("  Input tokens:  %s\n", formatNumber(costs.TotalInputTokens))
	fmt.Printf("  Output tokens: %s\n", formatNumber(costs.TotalOutputTokens))
	fmt.Printf("  Cached tokens: %s", formatNumber(costs.TotalCachedTokens))
	if costs.TotalCachedTokens > 0 {
		fmt.Printf(" (%.1f%% of total)", cachedPercent)
	}
	fmt.Println()
	fmt.Printf("  Total tokens:  %s\n", formatNumber(totalTokens))
	fmt.Printf("  Total cost:    %s\n", tkdisplay.Bold(formatCost(costs.TotalCostUSD)))
	fmt.Println(tkdisplay.Muted("  (Based on Claude API pricing)"))

	// Show by-step breakdown if requested or if there are multiple steps
	if costByStep || len(costs.ByStep) > 1 {
		if len(costs.ByStep) > 0 {
			fmt.Printf("\n%s\n", tkdisplay.Bold("By Step:"))

			// Sort steps by name for consistent output
			steps := make([]string, 0, len(costs.ByStep))
			for step := range costs.ByStep {
				steps = append(steps, step)
			}
			slices.Sort(steps)

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "  STEP\t\tINPUT\tOUTPUT\tCACHED\tCOST\tCALLS")
			for _, step := range steps {
				stats := costs.ByStep[step]
				_, _ = fmt.Fprintf(w, "  %s\t\t%s\t%s\t%s\t%s\t%d\n",
					formatStepName(step),
					formatNumber(stats.InputTokens),
					formatNumber(stats.OutputTokens),
					formatNumber(stats.CachedTokens),
					formatCost(stats.CostUSD),
					stats.Calls,
				)
			}
			_ = w.Flush()
		}
	}

	// Show chart if requested
	if costChart {
		fmt.Printf("\n%s\n", tkdisplay.Bold("Cost Visualization:"))
		renderStepCostChart(costs.ByStep)
	}

	return nil
}

func outputJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	return enc.Encode(v)
}

func showAllCosts(ws *storage.Workspace, summaryMode bool) error {
	taskIDs, err := ws.ListWorks()
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}

	if len(taskIDs) == 0 {
		if costJSON {
			return outputJSON(jsonAllCostsOutput{
				Tasks:      []jsonCostOutput{},
				GrandTotal: jsonGrandTotal{},
			})
		}
		fmt.Println("No tasks found in workspace.")

		return nil
	}

	// Check which task is active
	var activeID string
	if ws.HasActiveTask() {
		active, _ := ws.LoadActiveTask()
		if active != nil {
			activeID = active.ID
		}
	}

	if summaryMode {
		return showCostSummary(ws, taskIDs)
	}

	// JSON output
	if costJSON {
		var tasks []jsonCostOutput
		var grandTotalInput, grandTotalOutput int
		var grandTotalCost float64

		for _, taskID := range taskIDs {
			work, err := ws.LoadWork(taskID)
			if err != nil {
				continue
			}

			costs := work.Costs
			totalTokens := costs.TotalInputTokens + costs.TotalOutputTokens

			title := work.Metadata.Title
			if title == "" {
				title = "(untitled)"
			}

			taskJSON := jsonCostOutput{
				TaskID:       taskID,
				Title:        title,
				InputTokens:  costs.TotalInputTokens,
				OutputTokens: costs.TotalOutputTokens,
				TotalTokens:  totalTokens,
				CachedTokens: costs.TotalCachedTokens,
				TotalCostUSD: costs.TotalCostUSD,
			}
			tasks = append(tasks, taskJSON)

			grandTotalInput += costs.TotalInputTokens
			grandTotalOutput += costs.TotalOutputTokens
			grandTotalCost += costs.TotalCostUSD
		}

		return outputJSON(jsonAllCostsOutput{
			Tasks: tasks,
			GrandTotal: jsonGrandTotal{
				InputTokens:  grandTotalInput,
				OutputTokens: grandTotalOutput,
				TotalTokens:  grandTotalInput + grandTotalOutput,
				CostUSD:      grandTotalCost,
			},
		})
	}

	// Regular text output
	// Show all tasks with costs
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TASK ID\tTITLE\tINPUT\tOUTPUT\tTOTAL\tCOST")

	for _, taskID := range taskIDs {
		work, err := ws.LoadWork(taskID)
		if err != nil {
			continue
		}

		costs := work.Costs
		totalTokens := costs.TotalInputTokens + costs.TotalOutputTokens

		title := work.Metadata.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}
		if title == "" {
			title = "(untitled)"
		}

		// Mark active task
		if taskID == activeID {
			title = "* " + title
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			taskID,
			title,
			formatNumber(costs.TotalInputTokens),
			formatNumber(costs.TotalOutputTokens),
			formatNumber(totalTokens),
			formatCost(costs.TotalCostUSD),
		)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush table: %w", err)
	}

	// Calculate grand total
	var grandTotalInput, grandTotalOutput, grandTotalCost int64
	for _, taskID := range taskIDs {
		work, err := ws.LoadWork(taskID)
		if err != nil {
			continue
		}
		grandTotalInput += int64(work.Costs.TotalInputTokens)
		grandTotalOutput += int64(work.Costs.TotalOutputTokens)
		grandTotalCost += int64(work.Costs.TotalCostUSD * 10000) // Convert to fixed-point
	}

	fmt.Println()
	fmt.Printf("Total: %s input, %s output, %s\n",
		formatNumber(int(grandTotalInput)),
		formatNumber(int(grandTotalOutput)),
		formatCost(float64(grandTotalCost)/10000),
	)

	// Show chart if requested
	if costChart {
		fmt.Printf("\n%s\n", tkdisplay.Bold("Cost Visualization:"))
		renderAllTasksChart(ws, taskIDs)
	}

	return nil
}

func showCostSummary(ws *storage.Workspace, taskIDs []string) error {
	var grandTotalInput, grandTotalOutput, grandTotalCached int
	var grandTotalCost float64
	var taskCount int

	// Per-step totals
	stepTotals := make(map[string]*storage.StepCostStats)

	for _, taskID := range taskIDs {
		work, err := ws.LoadWork(taskID)
		if err != nil {
			continue
		}

		taskCount++
		costs := work.Costs

		grandTotalInput += costs.TotalInputTokens
		grandTotalOutput += costs.TotalOutputTokens
		grandTotalCached += costs.TotalCachedTokens
		grandTotalCost += costs.TotalCostUSD

		// Aggregate step totals
		for step, stats := range costs.ByStep {
			if stepTotals[step] == nil {
				stepTotals[step] = &storage.StepCostStats{}
			}
			s := stepTotals[step]
			s.InputTokens += stats.InputTokens
			s.OutputTokens += stats.OutputTokens
			s.CachedTokens += stats.CachedTokens
			s.CostUSD += stats.CostUSD
			s.Calls += stats.Calls
		}
	}

	if taskCount == 0 {
		if costJSON {
			return outputJSON(jsonSummaryOutput{
				TaskCount:  0,
				GrandTotal: jsonGrandTotal{},
				ByStep:     make(map[string]jsonStepCost),
			})
		}
		fmt.Println("No tasks found.")

		return nil
	}

	totalTokens := grandTotalInput + grandTotalOutput

	// JSON output
	if costJSON {
		byStep := make(map[string]jsonStepCost)
		for step, stats := range stepTotals {
			byStep[step] = jsonStepCost{
				InputTokens:  stats.InputTokens,
				OutputTokens: stats.OutputTokens,
				CachedTokens: stats.CachedTokens,
				TotalTokens:  stats.InputTokens + stats.OutputTokens,
				CostUSD:      stats.CostUSD,
				Calls:        stats.Calls,
			}
		}

		return outputJSON(jsonSummaryOutput{
			TaskCount: taskCount,
			GrandTotal: jsonGrandTotal{
				InputTokens:  grandTotalInput,
				OutputTokens: grandTotalOutput,
				TotalTokens:  totalTokens,
				CachedTokens: grandTotalCached,
				CostUSD:      grandTotalCost,
			},
			ByStep: byStep,
		})
	}

	// Regular text output
	fmt.Printf("Cost Summary for %d task(s)\n", taskCount)
	fmt.Printf("%s\n", strings.Repeat("─", 40))

	cachedPercent := 0.0
	if totalTokens > 0 && grandTotalCached > 0 {
		cachedPercent = float64(grandTotalCached) / float64(totalTokens) * 100
	}

	fmt.Printf("\n%s\n", tkdisplay.Bold("Grand Totals:"))
	fmt.Printf("  Input tokens:  %s\n", formatNumber(grandTotalInput))
	fmt.Printf("  Output tokens: %s\n", formatNumber(grandTotalOutput))
	fmt.Printf("  Cached tokens: %s", formatNumber(grandTotalCached))
	if grandTotalCached > 0 {
		fmt.Printf(" (%.1f%% of total)", cachedPercent)
	}
	fmt.Println()
	fmt.Printf("  Total tokens:  %s\n", formatNumber(totalTokens))
	fmt.Printf("  Total cost:    %s\n", tkdisplay.Bold(formatCost(grandTotalCost)))

	if len(stepTotals) > 0 {
		fmt.Printf("\n%s\n", tkdisplay.Bold("By Step:"))

		// Sort steps by name
		steps := make([]string, 0, len(stepTotals))
		for step := range stepTotals {
			steps = append(steps, step)
		}
		slices.Sort(steps)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "  STEP\t\tINPUT\tOUTPUT\tCACHED\tCOST\tCALLS")
		for _, step := range steps {
			stats := stepTotals[step]
			_, _ = fmt.Fprintf(w, "  %s\t\t%s\t%s\t%s\t%s\t%d\n",
				formatStepName(step),
				formatNumber(stats.InputTokens),
				formatNumber(stats.OutputTokens),
				formatNumber(stats.CachedTokens),
				formatCost(stats.CostUSD),
				stats.Calls,
			)
		}
		_ = w.Flush()
	}

	// Show chart if requested
	if costChart {
		fmt.Printf("\n%s\n", tkdisplay.Bold("Cost Visualization:"))
		renderSummaryChart(stepTotals)
	}

	return nil
}

// formatNumber formats a number with thousands separator.
func formatNumber(n int) string {
	if n == 0 {
		return "0"
	}

	s := strconv.FormatInt(int64(n), 10)
	numCommas := (len(s) - 1) / 3
	result := make([]byte, 0, len(s)+numCommas)

	// First segment (may be 1-3 digits)
	firstLen := len(s) % 3
	if firstLen == 0 {
		firstLen = 3
	}
	result = append(result, s[:firstLen]...)

	// Remaining segments with commas
	for i := firstLen; i < len(s); i += 3 {
		result = append(result, ',')
		result = append(result, s[i:i+3]...)
	}

	return string(result)
}

// formatCost formats a cost in USD with appropriate precision
// Shows 2 decimals for amounts >= $0.01, 4 decimals for smaller amounts.
func formatCost(cost float64) string {
	if cost == 0 {
		return "$0.00"
	}
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}

	return fmt.Sprintf("$%.2f", cost)
}

// formatStepName formats a step name for display.
func formatStepName(step string) string {
	// Capitalize first letter
	if len(step) == 0 {
		return step
	}

	return strings.ToUpper(step[:1]) + step[1:]
}

// renderStepCostChart renders a bar chart of costs by workflow step.
func renderStepCostChart(byStep map[string]storage.StepCostStats) {
	if len(byStep) == 0 {
		fmt.Println("  No step data available for chart.")

		return
	}

	// Prepare bars for the chart
	var bars []cost.Bar
	maxVal := 0

	for _, stats := range byStep {
		totalTokens := stats.InputTokens + stats.OutputTokens
		if totalTokens > maxVal {
			maxVal = totalTokens
		}
	}

	// Sort steps by name
	steps := make([]string, 0, len(byStep))
	for step := range byStep {
		steps = append(steps, step)
	}
	slices.Sort(steps)

	// Create bars
	for _, step := range steps {
		stats := byStep[step]
		totalTokens := stats.InputTokens + stats.OutputTokens
		bars = append(bars, cost.Bar{
			Label: formatStepName(step),
			Value: totalTokens,
		})
	}

	// Generate horizontal bar chart
	opts := cost.ChartOptions{
		Title:      "Token Usage by Step",
		Width:      50,
		ShowValues: true,
		ScaleLabel: "tokens",
	}
	chart := cost.ASCIIBarChart(bars, opts)
	fmt.Print(chart)
}

// renderAllTasksChart renders a bar chart comparing costs across all tasks.
func renderAllTasksChart(ws *storage.Workspace, taskIDs []string) {
	type taskCostData struct {
		ID    string
		Title string
		Cost  int
	}

	var tasks []taskCostData
	maxCost := 0

	for _, taskID := range taskIDs {
		work, err := ws.LoadWork(taskID)
		if err != nil {
			continue
		}

		totalCost := work.Costs.TotalInputTokens + work.Costs.TotalOutputTokens
		if totalCost > maxCost {
			maxCost = totalCost
		}

		title := work.Metadata.Title
		if title == "" {
			title = "(untitled)"
		}
		if len(title) > 20 {
			title = title[:17] + "..."
		}

		tasks = append(tasks, taskCostData{
			ID:    taskID,
			Title: title,
			Cost:  totalCost,
		})
	}

	if len(tasks) == 0 {
		fmt.Println("  No task data available for chart.")

		return
	}

	// Sort by cost (highest first)
	for i := range len(tasks) - 1 {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[i].Cost < tasks[j].Cost {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}

	// Limit to top 10 tasks for readability
	if len(tasks) > 10 {
		tasks = tasks[:10]
	}

	// Create bars
	var bars []cost.Bar
	for _, task := range tasks {
		label := task.Title
		if len(label) > 15 {
			label = label[:12] + "..."
		}
		bars = append(bars, cost.Bar{
			Label: label,
			Value: task.Cost,
		})
	}

	// Generate horizontal bar chart
	opts := cost.ChartOptions{
		Title:      "Token Usage by Task (Top 10)",
		Width:      50,
		ShowValues: true,
		ScaleLabel: "tokens",
	}
	chart := cost.ASCIIBarChart(bars, opts)
	fmt.Print(chart)
}

// renderSummaryChart renders charts for the summary view.
func renderSummaryChart(stepTotals map[string]*storage.StepCostStats) {
	if len(stepTotals) == 0 {
		fmt.Println("  No step data available for chart.")

		return
	}

	// First, render bar chart
	var bars []cost.Bar
	maxVal := 0

	// Sort steps by name
	steps := make([]string, 0, len(stepTotals))
	for step := range stepTotals {
		steps = append(steps, step)
	}
	slices.Sort(steps)

	for _, stepName := range steps {
		stats := stepTotals[stepName]
		totalTokens := stats.InputTokens + stats.OutputTokens
		if totalTokens > maxVal {
			maxVal = totalTokens
		}
	}

	for _, stepName := range steps {
		stats := stepTotals[stepName]
		totalTokens := stats.InputTokens + stats.OutputTokens
		bars = append(bars, cost.Bar{
			Label: formatStepName(stepName),
			Value: totalTokens,
		})
	}

	// Generate horizontal bar chart
	barOpts := cost.ChartOptions{
		Title:      "Total Token Usage by Step",
		Width:      50,
		ShowValues: true,
		ScaleLabel: "tokens",
	}
	chart := cost.ASCIIBarChart(bars, barOpts)
	fmt.Print(chart)

	// Second, render pie chart
	var slices []struct {
		Label   string
		Value   int
		Percent float64
	}

	totalTokens := 0
	for _, stats := range stepTotals {
		totalTokens += stats.InputTokens + stats.OutputTokens
	}

	for _, step := range steps {
		stats := stepTotals[step]
		stepTotal := stats.InputTokens + stats.OutputTokens
		percent := 0.0
		if totalTokens > 0 {
			percent = float64(stepTotal) / float64(totalTokens) * 100
		}
		slices = append(slices, struct {
			Label   string
			Value   int
			Percent float64
		}{
			Label:   formatStepName(step),
			Value:   stepTotal,
			Percent: percent,
		})
	}

	pieOpts := cost.ChartOptions{
		Title: "\nToken Distribution by Step",
	}
	pieChart := cost.ASCIIPieChart(slices, pieOpts)
	fmt.Print(pieChart)
}
