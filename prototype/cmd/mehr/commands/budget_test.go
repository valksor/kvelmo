//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/helper_test"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/paths"
)

func TestBudgetCommand_Structure(t *testing.T) {
	if budgetCmd.Use != "budget" {
		t.Errorf("Use = %q, want %q", budgetCmd.Use, "budget")
	}
	if budgetCmd.Short == "" {
		t.Error("Short description is empty")
	}
	if budgetCmd.Long == "" {
		t.Error("Long description is empty")
	}
}

func TestBudgetCommand_RegisteredInRoot(t *testing.T) {
	if !hasCommand(rootCmd, "budget") {
		t.Error("budget command not registered with rootCmd")
	}
}

func TestBudgetCommand_Subcommands(t *testing.T) {
	expected := []string{"status", "set", "task", "resume", "reset"}
	for _, name := range expected {
		if !hasCommand(budgetCmd, name) {
			t.Errorf("budget command missing subcommand %q", name)
		}
	}
}

func TestFormatLimitAction(t *testing.T) {
	tests := []struct {
		name   string
		action string
		want   string
	}{
		{"empty defaults to warn", "", "warn"},
		{"stop", "stop", "stop"},
		{"pause", "pause", "pause"},
		{"warn", "warn", "warn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLimitAction(tt.action)
			if got != tt.want {
				t.Errorf("formatLimitAction(%q) = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}

func TestFormatWarning(t *testing.T) {
	tests := []struct {
		name    string
		warning float64
		want    string
	}{
		{"zero is disabled", 0, "disabled"},
		{"negative is disabled", -0.1, "disabled"},
		{"80 percent", 0.8, "80%"},
		{"50 percent", 0.5, "50%"},
		{"100 percent", 1.0, "100%"},
		{"25 percent", 0.25, "25%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatWarning(tt.warning)
			if got != tt.want {
				t.Errorf("formatWarning(%v) = %q, want %q", tt.warning, got, tt.want)
			}
		})
	}
}

func TestFormatLimit(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  string
	}{
		{"zero is unlimited", 0, "unlimited"},
		{"negative is unlimited", -1, "unlimited"},
		{"thousand", 1000, "1,000"},
		{"million", 1000000, "1,000,000"},
		{"small number", 42, "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLimit(tt.value)
			if got != tt.want {
				t.Errorf("formatLimit(%d) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestCurrentBudgetMonth(t *testing.T) {
	got := currentBudgetMonth()
	expected := time.Now().Format("2006-01")

	if got != expected {
		t.Errorf("currentBudgetMonth() = %q, want %q", got, expected)
	}

	// Verify the format matches YYYY-MM pattern
	if len(got) != 7 || got[4] != '-' {
		t.Errorf("currentBudgetMonth() = %q, does not match YYYY-MM format", got)
	}
}

// setupBudgetWorkspace creates a workspace for budget tests.
func setupBudgetWorkspace(t *testing.T) *storage.Workspace {
	t.Helper()

	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	restoreHome := paths.SetHomeDirForTesting(homeDir)
	t.Cleanup(restoreHome)

	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir

	ws, err := storage.OpenWorkspace(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("OpenWorkspace: %v", err)
	}

	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	t.Chdir(tmpDir)

	return ws
}

// saveBudgetFlags saves and defers restore of all budget-related flags.
func saveBudgetFlags(t *testing.T) {
	t.Helper()

	origTaskID := budgetTaskID
	origTaskMaxCost := taskMaxCost
	origTaskMaxTokens := taskMaxTokens
	origTaskOnLimit := taskOnLimit
	origTaskWarningAt := taskWarningAt
	origTaskCurrency := taskCurrency
	origMonthlyMaxCost := monthlyMaxCost
	origMonthlyWarning := monthlyWarning
	origMonthlyCurrency := monthlyCurrency
	origResumeConfirm := resumeConfirm
	origResetMonth := resetMonth

	t.Cleanup(func() {
		budgetTaskID = origTaskID
		taskMaxCost = origTaskMaxCost
		taskMaxTokens = origTaskMaxTokens
		taskOnLimit = origTaskOnLimit
		taskWarningAt = origTaskWarningAt
		taskCurrency = origTaskCurrency
		monthlyMaxCost = origMonthlyMaxCost
		monthlyWarning = origMonthlyWarning
		monthlyCurrency = origMonthlyCurrency
		resumeConfirm = origResumeConfirm
		resetMonth = origResetMonth
	})

	budgetTaskID = ""
	taskMaxCost = 0
	taskMaxTokens = 0
	taskOnLimit = ""
	taskWarningAt = 0
	taskCurrency = ""
	monthlyMaxCost = 0
	monthlyWarning = 0
	monthlyCurrency = ""
	resumeConfirm = false
	resetMonth = false
}

func TestRunBudgetStatus_NoTask(t *testing.T) {
	_ = setupBudgetWorkspace(t)
	saveBudgetFlags(t)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runBudgetStatus(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runBudgetStatus() error = %v", err)
	}

	if !strings.Contains(output, "No active task found") {
		t.Errorf("output missing 'No active task found'\nGot:\n%s", output)
	}
}

func TestRunBudgetStatus_WithTask(t *testing.T) {
	ws := setupBudgetWorkspace(t)
	saveBudgetFlags(t)

	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))

	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: helper_test.SampleTaskContent("Budget Task"),
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	work.Metadata.Title = "Budget Task"

	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runBudgetStatus(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if runErr != nil {
		t.Fatalf("runBudgetStatus() error = %v", runErr)
	}

	for _, substr := range []string{"Task Budget", "task-1"} {
		if !strings.Contains(output, substr) {
			t.Errorf("output missing %q\nGot:\n%s", substr, output)
		}
	}
}

func TestRunBudgetSet_UpdatesConfig(t *testing.T) {
	ws := setupBudgetWorkspace(t)
	saveBudgetFlags(t)

	monthlyMaxCost = 100.0

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Simulate a flag being "changed"
	cmd.Flags().Float64Var(&monthlyMaxCost, "monthly-max-cost", 0, "")
	_ = cmd.Flags().Set("monthly-max-cost", "100")

	runErr := runBudgetSet(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if runErr != nil {
		t.Fatalf("runBudgetSet() error = %v", runErr)
	}

	if !strings.Contains(output, "Budget settings updated") {
		t.Errorf("output missing 'Budget settings updated'\nGot:\n%s", output)
	}

	// Verify config was saved
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Budget.Monthly.MaxCost != 100.0 {
		t.Errorf("saved monthly max cost = %v, want 100.0", cfg.Budget.Monthly.MaxCost)
	}
}

func TestRunBudgetTaskSet_NoTask(t *testing.T) {
	_ = setupBudgetWorkspace(t)
	saveBudgetFlags(t)

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runBudgetTaskSet(cmd, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "no task specified") {
		t.Errorf("error = %q, want it to contain 'no task specified'", err.Error())
	}
}

func TestRunBudgetTaskSet_WithTask(t *testing.T) {
	ws := setupBudgetWorkspace(t)
	saveBudgetFlags(t)

	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))

	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: helper_test.SampleTaskContent("Budget Task"),
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	work.Metadata.Title = "Budget Task"

	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	taskMaxCost = 5.0

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Float64Var(&taskMaxCost, "max-cost", 0, "")
	_ = cmd.Flags().Set("max-cost", "5")

	runErr := runBudgetTaskSet(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if runErr != nil {
		t.Fatalf("runBudgetTaskSet() error = %v", runErr)
	}

	if !strings.Contains(output, "Task budget updated") {
		t.Errorf("output missing 'Task budget updated'\nGot:\n%s", output)
	}
}

func TestRunBudgetReset_NoFlag(t *testing.T) {
	saveBudgetFlags(t)
	resetMonth = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runBudgetReset(cmd, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "reset requires --month") {
		t.Errorf("error = %q, want it to contain 'reset requires --month'", err.Error())
	}
}

func TestRunBudgetReset_WithFlag(t *testing.T) {
	_ = setupBudgetWorkspace(t)
	saveBudgetFlags(t)
	resetMonth = true

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runBudgetReset(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runBudgetReset() error = %v", err)
	}

	if !strings.Contains(output, "Monthly budget tracking reset") {
		t.Errorf("output missing 'Monthly budget tracking reset'\nGot:\n%s", output)
	}
}

func TestRunBudgetResume_NoConfirm(t *testing.T) {
	saveBudgetFlags(t)
	resumeConfirm = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runBudgetResume(cmd, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "resume requires --confirm") {
		t.Errorf("error = %q, want it to contain 'resume requires --confirm'", err.Error())
	}
}
