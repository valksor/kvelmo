package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
	"github.com/valksor/kvelmo/pkg/storage"
)

var (
	statsJSON bool
	statsAll  bool
)

var StatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show task analytics for the current project",
	Long:  "Display completion stats, success rate, and recent task history from the archive.",
	RunE:  runStats,
}

func init() {
	StatsCmd.Flags().BoolVar(&statsJSON, "json", false, "Output as JSON")
	StatsCmd.Flags().BoolVar(&statsAll, "all", false, "Show stats across all registered projects")
}

type statsOutput struct {
	Total       int            `json:"total"`
	ByState     map[string]int `json:"by_state"`
	SuccessRate float64        `json:"success_rate"`
	AvgDuration string         `json:"avg_duration,omitempty"`
	Recent      []recentTask   `json:"recent,omitempty"`
}

type recentTask struct {
	Title       string `json:"title"`
	FinalState  string `json:"final_state"`
	CompletedAt string `json:"completed_at"`
	Duration    string `json:"duration"`
}

func runStats(cmd *cobra.Command, args []string) error {
	if statsAll {
		return runStatsAll()
	}

	return runStatsProject()
}

func runStatsProject() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	wtPath := socket.WorktreeSocketPath(cwd)
	if !socket.SocketExists(wtPath) {
		return fmt.Errorf("no worktree socket running for %s\nRun '%s start' first", cwd, meta.Name)
	}

	client, err := socket.NewClient(wtPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connect to worktree socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "task.history", nil)
	if err != nil {
		return fmt.Errorf("task.history: %w", err)
	}

	var result struct {
		Tasks []storage.ArchivedTask `json:"tasks"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse history: %w", err)
	}

	output := computeStats(result.Tasks)

	return printStats(output)
}

func runStatsAll() error {
	globalPath := socket.GlobalSocketPath()
	if !socket.SocketExists(globalPath) {
		return fmt.Errorf("global socket not running\nRun '%s serve' or '%s start' first", meta.Name, meta.Name)
	}

	client, err := socket.NewClient(globalPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connect to global socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "tasks.list", nil)
	if err != nil {
		return fmt.Errorf("tasks.list: %w", err)
	}

	var listResult socket.TasksListResult
	if err := json.Unmarshal(resp.Result, &listResult); err != nil {
		return fmt.Errorf("parse project list: %w", err)
	}

	var allTasks []storage.ArchivedTask

	for _, t := range listResult.Tasks {
		wtPath := socket.WorktreeSocketPath(t.Path)
		if !socket.SocketExists(wtPath) {
			continue
		}

		wtClient, err := socket.NewClient(wtPath, socket.WithTimeout(3*time.Second))
		if err != nil {
			continue
		}

		wtCtx, wtCancel := context.WithTimeout(context.Background(), 3*time.Second)
		resp, err := wtClient.Call(wtCtx, "task.history", nil)
		wtCancel()
		_ = wtClient.Close()

		if err != nil {
			continue
		}

		var result struct {
			Tasks []storage.ArchivedTask `json:"tasks"`
		}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			continue
		}

		allTasks = append(allTasks, result.Tasks...)
	}

	output := computeStats(allTasks)

	return printStats(output)
}

func computeStats(tasks []storage.ArchivedTask) statsOutput {
	out := statsOutput{
		Total:   len(tasks),
		ByState: make(map[string]int),
	}

	if len(tasks) == 0 {
		return out
	}

	var totalDuration time.Duration
	var durationCount int

	for _, t := range tasks {
		out.ByState[t.FinalState]++

		dur := t.CompletedAt.Sub(t.StartedAt)
		if dur > 0 {
			totalDuration += dur
			durationCount++
		}
	}

	finished := out.ByState["finished"] + out.ByState["submitted"]
	if out.Total > 0 {
		out.SuccessRate = math.Round(float64(finished)/float64(out.Total)*1000) / 10
	}

	if durationCount > 0 {
		avg := totalDuration / time.Duration(durationCount)
		out.AvgDuration = formatDuration(avg)
	}

	limit := 5
	if len(tasks) < limit {
		limit = len(tasks)
	}
	for _, t := range tasks[:limit] {
		dur := t.CompletedAt.Sub(t.StartedAt)
		title := t.Title
		if title == "" {
			title = t.ID
		}
		out.Recent = append(out.Recent, recentTask{
			Title:       title,
			FinalState:  t.FinalState,
			CompletedAt: t.CompletedAt.Format("2006-01-02 15:04"),
			Duration:    formatDuration(dur),
		})
	}

	return out
}

func printStats(out statsOutput) error {
	if statsJSON {
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal json: %w", err)
		}
		fmt.Println(string(data))

		return nil
	}

	if out.Total == 0 {
		fmt.Println("No completed tasks yet.")

		return nil
	}

	fmt.Printf("Tasks: %d total", out.Total)
	for state, count := range out.ByState {
		fmt.Printf(", %d %s", count, state)
	}
	fmt.Println()

	fmt.Printf("Success rate: %.1f%%\n", out.SuccessRate)

	if out.AvgDuration != "" {
		fmt.Printf("Avg duration: %s\n", out.AvgDuration)
	}

	if len(out.Recent) > 0 {
		fmt.Printf("\nRecent tasks:\n")
		for _, t := range out.Recent {
			fmt.Printf("  %-30s  %-10s  %s  (%s)\n", truncate(t.Title, 30), t.FinalState, t.CompletedAt, t.Duration)
		}
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}

	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	return s[:limit-3] + "..."
}
