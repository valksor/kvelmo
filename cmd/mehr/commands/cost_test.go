//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// Note: TestCostCommand_BreakdownFlag is in common_test.go

func TestCostCommand_Properties(t *testing.T) {
	if costCmd.Use != "cost" {
		t.Errorf("Use = %q, want %q", costCmd.Use, "cost")
	}

	if costCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if costCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if costCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestCostCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "breakdown flag",
			flagName:     "breakdown",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "all flag",
			flagName:     "all",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "summary flag",
			flagName:     "summary",
			shorthand:    "s",
			defaultValue: "false",
		},
		{
			name:         "json flag",
			flagName:     "json",
			shorthand:    "",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := costCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			if tt.shorthand != "" {
				shorthand := costCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestCostCommand_ShortDescription(t *testing.T) {
	expected := "Show token usage and costs"
	if costCmd.Short != expected {
		t.Errorf("Short = %q, want %q", costCmd.Short, expected)
	}
}

func TestCostCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"token usage",
		"API costs",
		"input/output tokens",
		"cached tokens",
		"estimated costs",
	}

	for _, substr := range contains {
		if !containsString(costCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestCostCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr cost",
		"--breakdown",
		"--all",
		"--summary",
		"--json",
	}

	for _, example := range examples {
		if !containsString(costCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestCostCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "cost" {
			found = true

			break
		}
	}
	if !found {
		t.Error("cost command not registered in root command")
	}
}

func TestCostCommand_SummaryFlagHasShorthand(t *testing.T) {
	flag := costCmd.Flags().Lookup("summary")
	if flag == nil {
		t.Fatal("summary flag not found")

		return
	}
	if flag.Shorthand != "s" {
		t.Errorf("summary flag shorthand = %q, want 's'", flag.Shorthand)
	}
}

func TestCostCommand_NoAliases(t *testing.T) {
	// Cost command should not have aliases to avoid confusion
	if len(costCmd.Aliases) > 0 {
		t.Errorf("cost command has unexpected aliases: %v", costCmd.Aliases)
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{123, "123"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatNumber(tt.input)
			if result != tt.expected {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0, "$0.00"},
		{0.001, "$0.0010"},
		{0.009, "$0.0090"},
		{0.01, "$0.01"},
		{0.10, "$0.10"},
		{1.00, "$1.00"},
		{1.23, "$1.23"},
		{12.34, "$12.34"},
		{123.45, "$123.45"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatCost(tt.input)
			if result != tt.expected {
				t.Errorf("formatCost(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRenderStepCostChart_Empty(t *testing.T) {
	// Capture stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	renderStepCostChart(map[string]storage.StepCostStats{})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "No step data") {
		t.Errorf("renderStepCostChart(empty) output = %q, want it to contain 'No step data'", output)
	}
}

func TestFormatStepName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"planning", "Planning"},
		{"implementing", "Implementing"},
		{"reviewing", "Reviewing"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatStepName(tt.input)
			if result != tt.expected {
				t.Errorf("formatStepName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRenderSummaryChart_Empty(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	renderSummaryChart(map[string]*storage.StepCostStats{})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "No step data") {
		t.Errorf("renderSummaryChart(empty) output = %q, want it to contain 'No step data'", output)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Phase 4: Behavioral tests with workspace data
// ─────────────────────────────────────────────────────────────────────────────

func TestShowActiveCost_NoActiveTask(t *testing.T) {
	tc := NewTestContext(t)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showActiveCost(tc.Workspace)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showActiveCost() error = %v, want nil", err)
	}

	if !strings.Contains(output, "No active task") {
		t.Errorf("output = %q, want it to contain 'No active task'", output)
	}
}

func TestShowTaskCost_NoCostData(t *testing.T) {
	tc := NewTestContext(t)
	tc.CreateActiveTask("task-1", "file:test.md")
	tc.CreateTaskWork("task-1", "Zero Cost Task")

	// Save and restore package-level flags
	oldJSON := costJSON
	oldByStep := costByStep
	oldChart := costChart
	defer func() {
		costJSON = oldJSON
		costByStep = oldByStep
		costChart = oldChart
	}()
	costJSON = false
	costByStep = false
	costChart = false

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showTaskCost(tc.Workspace, "task-1", "task-1")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showTaskCost() error = %v, want nil", err)
	}

	if !strings.Contains(output, "No cost data available") {
		t.Errorf("output = %q, want it to contain 'No cost data available'", output)
	}
}

func TestShowTaskCost_WithCostData(t *testing.T) {
	tc := NewTestContext(t)
	tc.CreateActiveTask("task-1", "file:test.md")
	work := tc.CreateTaskWork("task-1", "Cost Test Task")
	work.Costs.TotalInputTokens = 1000
	work.Costs.TotalOutputTokens = 500
	work.Costs.TotalCachedTokens = 200
	work.Costs.TotalCostUSD = 0.05
	work.Costs.ByStep = map[string]storage.StepCostStats{
		"planning":     {InputTokens: 600, OutputTokens: 200, CostUSD: 0.03, Calls: 1},
		"implementing": {InputTokens: 400, OutputTokens: 300, CostUSD: 0.02, Calls: 1},
	}
	if err := tc.Workspace.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	// Save and restore package-level flags
	oldJSON := costJSON
	oldByStep := costByStep
	oldChart := costChart
	defer func() {
		costJSON = oldJSON
		costByStep = oldByStep
		costChart = oldChart
	}()
	costJSON = false
	costByStep = true
	costChart = false

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showTaskCost(tc.Workspace, "task-1", "task-1")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showTaskCost() error = %v, want nil", err)
	}

	expectedSubstrings := []string{
		"Cost Test Task",
		"1,000",
		"500",
		"200",
		"$0.05",
		"By Step:",
		"Planning",
		"Implementing",
	}
	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}

func TestShowTaskCost_JSONOutput(t *testing.T) {
	tc := NewTestContext(t)
	tc.CreateActiveTask("task-1", "file:test.md")
	work := tc.CreateTaskWork("task-1", "Cost Test Task")
	work.Costs.TotalInputTokens = 1000
	work.Costs.TotalOutputTokens = 500
	work.Costs.TotalCachedTokens = 200
	work.Costs.TotalCostUSD = 0.05
	work.Costs.ByStep = map[string]storage.StepCostStats{
		"planning":     {InputTokens: 600, OutputTokens: 200, CostUSD: 0.03, Calls: 1},
		"implementing": {InputTokens: 400, OutputTokens: 300, CostUSD: 0.02, Calls: 1},
	}
	if err := tc.Workspace.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	// Save and restore package-level flags
	oldJSON := costJSON
	defer func() { costJSON = oldJSON }()
	costJSON = true

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showTaskCost(tc.Workspace, "task-1", "task-1")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showTaskCost() error = %v, want nil", err)
	}

	expectedSubstrings := []string{
		`"task_id"`,
		`"total_cost_usd"`,
		`"input_tokens"`,
	}
	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}

func TestShowAllCosts_Empty(t *testing.T) {
	tc := NewTestContext(t)

	// Save and restore package-level flags
	oldJSON := costJSON
	oldChart := costChart
	defer func() {
		costJSON = oldJSON
		costChart = oldChart
	}()
	costJSON = false
	costChart = false

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showAllCosts(tc.Workspace, false)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showAllCosts() error = %v, want nil", err)
	}

	if !strings.Contains(output, "No tasks found") {
		t.Errorf("output = %q, want it to contain 'No tasks found'", output)
	}
}

func TestRenderStepCostChart_WithData(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	byStep := map[string]storage.StepCostStats{
		"planning":     {InputTokens: 600, OutputTokens: 200},
		"implementing": {InputTokens: 400, OutputTokens: 300},
	}
	renderStepCostChart(byStep)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	expectedSubstrings := []string{
		"Token Usage by Step",
		"Planning",
		"Implementing",
	}
	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Phase 5: Deep behavioral tests for showAllCosts, showCostSummary, charts
// ─────────────────────────────────────────────────────────────────────────────

func TestShowAllCosts_WithTasks(t *testing.T) {
	tc := NewTestContext(t)
	tc.CreateActiveTask("task-1", "file:task1.md")
	work1 := tc.CreateTaskWork("task-1", "First Task")
	work1.Costs.TotalInputTokens = 1000
	work1.Costs.TotalOutputTokens = 500
	work1.Costs.TotalCostUSD = 0.05
	if err := tc.Workspace.SaveWork(work1); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	work2 := tc.CreateTaskWork("task-2", "Second Task")
	work2.Costs.TotalInputTokens = 2000
	work2.Costs.TotalOutputTokens = 1000
	work2.Costs.TotalCostUSD = 0.10
	if err := tc.Workspace.SaveWork(work2); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	oldJSON := costJSON
	oldChart := costChart
	defer func() {
		costJSON = oldJSON
		costChart = oldChart
	}()
	costJSON = false
	costChart = false

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showAllCosts(tc.Workspace, false)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showAllCosts() error = %v, want nil", err)
	}

	expectedSubstrings := []string{
		"TASK ID",
		"First Task",
		"Second Task",
		"Total:",
	}
	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}

func TestShowAllCosts_JSONOutput(t *testing.T) {
	tc := NewTestContext(t)
	tc.CreateActiveTask("task-1", "file:task1.md")
	work1 := tc.CreateTaskWork("task-1", "First Task")
	work1.Costs.TotalInputTokens = 1000
	work1.Costs.TotalOutputTokens = 500
	work1.Costs.TotalCostUSD = 0.05
	if err := tc.Workspace.SaveWork(work1); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	work2 := tc.CreateTaskWork("task-2", "Second Task")
	work2.Costs.TotalInputTokens = 2000
	work2.Costs.TotalOutputTokens = 1000
	work2.Costs.TotalCostUSD = 0.10
	if err := tc.Workspace.SaveWork(work2); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	oldJSON := costJSON
	oldChart := costChart
	defer func() {
		costJSON = oldJSON
		costChart = oldChart
	}()
	costJSON = true
	costChart = false

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showAllCosts(tc.Workspace, false)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showAllCosts() error = %v, want nil", err)
	}

	expectedSubstrings := []string{
		`"tasks"`,
		`"grand_total"`,
		`"task-1"`,
		`"task-2"`,
	}
	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}

func TestShowCostSummary_WithTasks(t *testing.T) {
	tc := NewTestContext(t)
	tc.CreateActiveTask("task-1", "file:task1.md")
	work1 := tc.CreateTaskWork("task-1", "First Task")
	work1.Costs.TotalInputTokens = 1000
	work1.Costs.TotalOutputTokens = 500
	work1.Costs.TotalCostUSD = 0.05
	work1.Costs.ByStep = map[string]storage.StepCostStats{
		"planning": {InputTokens: 600, OutputTokens: 200, CostUSD: 0.03, Calls: 1},
	}
	if err := tc.Workspace.SaveWork(work1); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	work2 := tc.CreateTaskWork("task-2", "Second Task")
	work2.Costs.TotalInputTokens = 2000
	work2.Costs.TotalOutputTokens = 1000
	work2.Costs.TotalCostUSD = 0.10
	work2.Costs.ByStep = map[string]storage.StepCostStats{
		"planning":     {InputTokens: 1000, OutputTokens: 500, CostUSD: 0.05, Calls: 1},
		"implementing": {InputTokens: 1000, OutputTokens: 500, CostUSD: 0.05, Calls: 2},
	}
	if err := tc.Workspace.SaveWork(work2); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	oldJSON := costJSON
	oldChart := costChart
	defer func() {
		costJSON = oldJSON
		costChart = oldChart
	}()
	costJSON = false
	costChart = false

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showAllCosts(tc.Workspace, true)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showAllCosts(summaryMode) error = %v, want nil", err)
	}

	expectedSubstrings := []string{
		"Cost Summary",
		"Grand Totals:",
		"By Step:",
		"Planning",
		"Implementing",
	}
	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}

func TestShowAllCosts_WithChart(t *testing.T) {
	tc := NewTestContext(t)
	tc.CreateActiveTask("task-1", "file:task1.md")
	work1 := tc.CreateTaskWork("task-1", "First Task")
	work1.Costs.TotalInputTokens = 1000
	work1.Costs.TotalOutputTokens = 500
	work1.Costs.TotalCostUSD = 0.05
	if err := tc.Workspace.SaveWork(work1); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	work2 := tc.CreateTaskWork("task-2", "Second Task")
	work2.Costs.TotalInputTokens = 2000
	work2.Costs.TotalOutputTokens = 1000
	work2.Costs.TotalCostUSD = 0.10
	if err := tc.Workspace.SaveWork(work2); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	oldJSON := costJSON
	oldChart := costChart
	defer func() {
		costJSON = oldJSON
		costChart = oldChart
	}()
	costJSON = false
	costChart = true

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := showAllCosts(tc.Workspace, false)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("showAllCosts(chart) error = %v, want nil", err)
	}

	expectedSubstrings := []string{
		"Cost Visualization:",
		"Token Usage",
	}
	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}

func TestRenderSummaryChart_WithData(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	stepTotals := map[string]*storage.StepCostStats{
		"planning":     {InputTokens: 600, OutputTokens: 200},
		"implementing": {InputTokens: 400, OutputTokens: 300},
	}
	renderSummaryChart(stepTotals)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	expectedSubstrings := []string{
		"Total Token Usage by Step",
		"Planning",
		"Implementing",
		"Token Distribution",
	}
	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}
