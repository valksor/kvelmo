package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
	"github.com/valksor/kvelmo/pkg/storage"
)

var ListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all registered projects and their current task state",
	Long: `Queries the global socket for all registered worktrees with their state and task info.

Use --history to search archived task history in the current project:
  kvelmo list --history --search "auth" --state finished --since 2026-03-01`,
	RunE: runList,
}

func init() {
	ListCmd.Flags().Bool("history", false, "Search archived task history for the current project")
	ListCmd.Flags().String("search", "", "Filter history by keyword (matches title, branch, source)")
	ListCmd.Flags().String("tag", "", "Filter history by tag")
	ListCmd.Flags().String("since", "", "Show tasks completed after this date (RFC3339 or YYYY-MM-DD)")
	ListCmd.Flags().String("until", "", "Show tasks completed before this date (RFC3339 or YYYY-MM-DD)")
	ListCmd.Flags().String("state", "", "Filter by final state (e.g., finished, abandoned)")
	ListCmd.Flags().Int("limit", 0, "Maximum number of results (0 = unlimited)")
}

func runList(cmd *cobra.Command, args []string) error {
	history, _ := cmd.Flags().GetBool("history")
	if history {
		return runListHistory(cmd)
	}

	return runListProjects(cmd)
}

func runListProjects(_ *cobra.Command) error {
	globalPath := socket.GlobalSocketPath()

	if !socket.SocketExists(globalPath) {
		return errors.New("global socket not running\nRun '" + meta.Name + " serve' or '" + meta.Name + " start' first")
	}

	client, err := socket.NewClient(globalPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connect to global socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "tasks.list", nil)
	if err != nil {
		return fmt.Errorf("tasks.list: %w", err)
	}

	var result socket.TasksListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	if len(result.Tasks) == 0 {
		fmt.Println("No registered projects")

		return nil
	}

	fmt.Printf("%-16s  %-14s  %-30s  %s\n", "ID", "State", "Task", "Path")
	fmt.Printf("%-16s  %-14s  %-30s  %s\n", "----------------", "--------------", "------------------------------", "----")
	for _, t := range result.Tasks {
		taskDisplay := t.TaskTitle
		if taskDisplay == "" {
			taskDisplay = t.TaskID
		}
		if taskDisplay == "" {
			taskDisplay = "(no task)"
		}
		if len(taskDisplay) > 30 {
			taskDisplay = taskDisplay[:27] + "..."
		}

		fmt.Printf("%-16s  %-14s  %-30s  %s\n",
			t.ID,
			t.State,
			taskDisplay,
			t.Path,
		)
	}

	return nil
}

func runListHistory(cmd *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	wtPath := socket.WorktreeSocketPath(cwd)
	if !socket.SocketExists(wtPath) {
		return errors.New("worktree socket not running in current directory\nRun '" + meta.Name + " serve' first")
	}

	client, err := socket.NewClient(wtPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connect to worktree socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	search, _ := cmd.Flags().GetString("search")
	tag, _ := cmd.Flags().GetString("tag")
	sinceStr, _ := cmd.Flags().GetString("since")
	untilStr, _ := cmd.Flags().GetString("until")
	state, _ := cmd.Flags().GetString("state")
	limit, _ := cmd.Flags().GetInt("limit")

	params := map[string]any{}
	if search != "" {
		params["query"] = search
	}
	if tag != "" {
		params["tag"] = tag
	}
	if sinceStr != "" {
		t, err := parseFlexibleTime(sinceStr)
		if err != nil {
			return fmt.Errorf("parse --since: %w", err)
		}
		params["since"] = t.Format(time.RFC3339)
	}
	if untilStr != "" {
		t, err := parseFlexibleTime(untilStr)
		if err != nil {
			return fmt.Errorf("parse --until: %w", err)
		}
		params["until"] = t.Format(time.RFC3339)
	}
	if state != "" {
		params["state"] = state
	}
	if limit > 0 {
		params["limit"] = limit
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "task.search", params)
	if err != nil {
		return fmt.Errorf("task.search: %w", err)
	}

	var result struct {
		Tasks []storage.ArchivedTask `json:"tasks"`
		Count int                    `json:"count"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	if len(result.Tasks) == 0 {
		fmt.Println("No matching archived tasks")

		return nil
	}

	fmt.Printf("%-30s  %-12s  %-20s  %s\n", "Title", "State", "Source", "Completed")
	fmt.Printf("%-30s  %-12s  %-20s  %s\n",
		strings.Repeat("-", 30), strings.Repeat("-", 12),
		strings.Repeat("-", 20), strings.Repeat("-", 19))
	for _, t := range result.Tasks {
		title := t.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}

		source := t.Source
		if len(source) > 20 {
			source = source[:17] + "..."
		}

		completed := t.CompletedAt.Format("2006-01-02 15:04:05")

		fmt.Printf("%-30s  %-12s  %-20s  %s\n", title, t.FinalState, source, completed)
	}

	fmt.Printf("\n%d task(s) found\n", result.Count)

	return nil
}

// parseFlexibleTime parses a time string as either RFC3339 or YYYY-MM-DD.
func parseFlexibleTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("expected RFC3339 or YYYY-MM-DD format, got %q", s)
}
