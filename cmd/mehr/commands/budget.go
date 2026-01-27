package commands

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/display"
)

var (
	budgetTaskID string

	taskMaxCost     float64
	taskMaxTokens   int
	taskOnLimit     string
	taskWarningAt   float64
	taskCurrency    string
	monthlyMaxCost  float64
	monthlyWarning  float64
	monthlyCurrency string

	resumeConfirm bool
	resetMonth    bool
)

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Manage cost and token budgets",
	Long: `Manage task and workspace budgets to control token usage and costs.

Examples:
  mehr budget status                 # Show budgets for active task
  mehr budget set --monthly-max-cost 100 --monthly-warning-at 0.8
  mehr budget task set --max-cost 5 --on-limit pause
  mehr budget resume --confirm       # Resume a paused task
  mehr budget reset --month          # Reset monthly budget tracking`,
}

var budgetStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show budget status",
	RunE:  runBudgetStatus,
}

var budgetSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set workspace budget defaults",
	RunE:  runBudgetSet,
}

var budgetTaskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage task budgets",
}

var budgetTaskSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set budget for a task",
	RunE:  runBudgetTaskSet,
}

var budgetResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a task paused due to budget limits",
	RunE:  runBudgetResume,
}

var budgetResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset monthly budget tracking",
	RunE:  runBudgetReset,
}

func init() {
	rootCmd.AddCommand(budgetCmd)

	budgetCmd.AddCommand(budgetStatusCmd)
	budgetCmd.AddCommand(budgetSetCmd)
	budgetCmd.AddCommand(budgetTaskCmd)
	budgetCmd.AddCommand(budgetResumeCmd)
	budgetCmd.AddCommand(budgetResetCmd)

	budgetStatusCmd.Flags().StringVar(&budgetTaskID, "task", "", "Task ID (defaults to active task)")

	budgetSetCmd.Flags().Float64Var(&taskMaxCost, "task-max-cost", 0, "Default max cost per task (USD)")
	budgetSetCmd.Flags().IntVar(&taskMaxTokens, "task-max-tokens", 0, "Default max tokens per task")
	budgetSetCmd.Flags().StringVar(&taskOnLimit, "task-on-limit", "", "Default task limit behavior (warn|pause|stop)")
	budgetSetCmd.Flags().Float64Var(&taskWarningAt, "task-warning-at", 0, "Default task warning threshold (0-1)")
	budgetSetCmd.Flags().StringVar(&taskCurrency, "task-currency", "", "Default task currency (e.g., USD)")
	budgetSetCmd.Flags().Float64Var(&monthlyMaxCost, "monthly-max-cost", 0, "Monthly max cost (USD)")
	budgetSetCmd.Flags().Float64Var(&monthlyWarning, "monthly-warning-at", 0, "Monthly warning threshold (0-1)")
	budgetSetCmd.Flags().StringVar(&monthlyCurrency, "monthly-currency", "", "Monthly currency (e.g., USD)")

	budgetTaskCmd.AddCommand(budgetTaskSetCmd)
	budgetTaskSetCmd.Flags().StringVar(&budgetTaskID, "task", "", "Task ID (defaults to active task)")
	budgetTaskSetCmd.Flags().Float64Var(&taskMaxCost, "max-cost", 0, "Max cost for the task (USD)")
	budgetTaskSetCmd.Flags().IntVar(&taskMaxTokens, "max-tokens", 0, "Max tokens for the task")
	budgetTaskSetCmd.Flags().StringVar(&taskOnLimit, "on-limit", "", "Limit behavior (warn|pause|stop)")
	budgetTaskSetCmd.Flags().Float64Var(&taskWarningAt, "warning-at", 0, "Warning threshold (0-1)")
	budgetTaskSetCmd.Flags().StringVar(&taskCurrency, "currency", "", "Currency (e.g., USD)")

	budgetResumeCmd.Flags().BoolVar(&resumeConfirm, "confirm", false, "Confirm resume after budget pause")
	budgetResetCmd.Flags().BoolVar(&resetMonth, "month", false, "Reset monthly budget tracking for the current month")
}

func runBudgetStatus(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}
	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	taskID := budgetTaskID
	if taskID == "" && ws.HasActiveTask() {
		active, err := ws.LoadActiveTask()
		if err == nil {
			taskID = active.ID
		}
	}

	if taskID != "" {
		work, err := ws.LoadWork(taskID)
		if err != nil {
			return fmt.Errorf("load task work: %w", err)
		}

		taskBudget := cfg.Budget.PerTask
		if work.Budget != nil {
			taskBudget = *work.Budget
		}

		fmt.Println(display.Bold("Task Budget"))
		fmt.Printf("  Task ID:   %s\n", taskID)
		fmt.Printf("  Tokens:    %s / %s\n", formatNumber(work.Costs.TotalInputTokens+work.Costs.TotalOutputTokens), formatLimit(taskBudget.MaxTokens))
		fmt.Printf("  Cost:      %s / %s\n", formatCost(work.Costs.TotalCostUSD), formatCost(taskBudget.MaxCost))
		fmt.Printf("  On limit:  %s\n", formatLimitAction(taskBudget.OnLimit))
		fmt.Printf("  Warning:   %s\n", formatWarning(taskBudget.WarningAt))
		if work.BudgetStatus != nil && work.BudgetStatus.Warned {
			fmt.Printf("  Status:    %s (warned)\n", display.Warning("warning issued"))
		}
		if work.BudgetStatus != nil && work.BudgetStatus.LimitHit {
			fmt.Printf("  Status:    %s (limit hit)\n", display.Error("limit reached"))
		}
		fmt.Println()
	} else {
		fmt.Println(display.WarningMsg("No active task found."))
	}

	monthly, err := ws.LoadMonthlyBudgetState()
	if err != nil {
		return fmt.Errorf("load monthly budget: %w", err)
	}

	if cfg.Budget.Monthly.MaxCost > 0 {
		fmt.Println(display.Bold("Monthly Budget"))
		fmt.Printf("  Month:     %s\n", monthly.Month)
		fmt.Printf("  Spent:     %s / %s\n", formatCost(monthly.Spent), formatCost(cfg.Budget.Monthly.MaxCost))
		fmt.Printf("  Warning:   %s\n", formatWarning(cfg.Budget.Monthly.WarningAt))
		if monthly.WarningSent {
			fmt.Printf("  Status:    %s (warned)\n", display.Warning("warning issued"))
		}
	} else {
		fmt.Println(display.Muted("Monthly budget not configured."))
	}

	return nil
}

func runBudgetSet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}
	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cmd.Flags().Changed("task-max-cost") {
		cfg.Budget.PerTask.MaxCost = taskMaxCost
	}
	if cmd.Flags().Changed("task-max-tokens") {
		cfg.Budget.PerTask.MaxTokens = taskMaxTokens
	}
	if cmd.Flags().Changed("task-on-limit") {
		cfg.Budget.PerTask.OnLimit = taskOnLimit
	}
	if cmd.Flags().Changed("task-warning-at") {
		cfg.Budget.PerTask.WarningAt = taskWarningAt
	}
	if cmd.Flags().Changed("task-currency") {
		cfg.Budget.PerTask.Currency = taskCurrency
	}
	if cmd.Flags().Changed("monthly-max-cost") {
		cfg.Budget.Monthly.MaxCost = monthlyMaxCost
	}
	if cmd.Flags().Changed("monthly-warning-at") {
		cfg.Budget.Monthly.WarningAt = monthlyWarning
	}
	if cmd.Flags().Changed("monthly-currency") {
		cfg.Budget.Monthly.Currency = monthlyCurrency
	}

	if err := ws.SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println(display.SuccessMsg("Budget settings updated."))

	return nil
}

func runBudgetTaskSet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}
	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	taskID := budgetTaskID
	if taskID == "" && ws.HasActiveTask() {
		active, err := ws.LoadActiveTask()
		if err == nil {
			taskID = active.ID
		}
	}
	if taskID == "" {
		return errors.New("no task specified and no active task")
	}

	work, err := ws.LoadWork(taskID)
	if err != nil {
		return fmt.Errorf("load work: %w", err)
	}

	if work.Budget == nil {
		work.Budget = &storage.BudgetConfig{}
	}
	if cmd.Flags().Changed("max-cost") {
		work.Budget.MaxCost = taskMaxCost
	}
	if cmd.Flags().Changed("max-tokens") {
		work.Budget.MaxTokens = taskMaxTokens
	}
	if cmd.Flags().Changed("on-limit") {
		work.Budget.OnLimit = taskOnLimit
	}
	if cmd.Flags().Changed("warning-at") {
		work.Budget.WarningAt = taskWarningAt
	}
	if cmd.Flags().Changed("currency") {
		work.Budget.Currency = taskCurrency
	}

	if err := ws.SaveWork(work); err != nil {
		return fmt.Errorf("save work: %w", err)
	}

	fmt.Println(display.SuccessMsg("Task budget updated."))

	return nil
}

func runBudgetResume(cmd *cobra.Command, args []string) error {
	if !resumeConfirm {
		return errors.New("resume requires --confirm")
	}

	ctx := cmd.Context()
	cond, err := initializeConductor(ctx, conductor.WithVerbose(verbose))
	if err != nil {
		return err
	}

	if err := cond.ResumePaused(ctx); err != nil {
		return err
	}

	fmt.Println(display.SuccessMsg("Task resumed."))

	return nil
}

func runBudgetReset(cmd *cobra.Command, args []string) error {
	if !resetMonth {
		return errors.New("reset requires --month")
	}

	ctx := cmd.Context()
	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}
	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	state := &storage.MonthlyBudgetState{Month: currentBudgetMonth()}
	if err := ws.SaveMonthlyBudgetState(state); err != nil {
		return fmt.Errorf("save monthly budget state: %w", err)
	}

	fmt.Println(display.SuccessMsg("Monthly budget tracking reset."))

	return nil
}

func formatLimitAction(action string) string {
	if action == "" {
		return "warn"
	}

	return action
}

func formatWarning(warning float64) string {
	if warning <= 0 {
		return "disabled"
	}

	return strconv.FormatFloat(warning*100, 'f', 0, 64) + "%"
}

func formatLimit(value int) string {
	if value <= 0 {
		return "unlimited"
	}

	return formatNumber(value)
}

func currentBudgetMonth() string {
	return time.Now().Format("2006-01")
}
