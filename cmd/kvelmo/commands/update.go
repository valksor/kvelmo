package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Re-fetch task from provider and generate delta specification if changed",
	Long: `Re-fetches the current task from its original provider and checks for changes.
If the task has changed, a delta specification file is generated describing what changed.`,
	RunE: runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	socketPath := socket.WorktreeSocketPath(cwd)

	if !socket.SocketExists(socketPath) {
		return errors.New("no worktree socket running\nRun '" + meta.Name + " start' first")
	}

	client, err := socket.NewClient(socketPath, socket.WithTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("connect to socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "update", nil)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	var result socket.UpdateResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	if !result.Changed {
		fmt.Println("Task unchanged")

		return nil
	}

	fmt.Println("Task updated")
	if result.NewSpecification != "" {
		fmt.Printf("Delta specification: %s\n", result.NewSpecification)
	}

	return nil
}
