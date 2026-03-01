package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered projects and their current task state",
	Long:  "Queries the global socket for all registered worktrees with their state and task info.",
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
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
