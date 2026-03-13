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

var StopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the current operation",
	Long: `Stop the current operation (planning, implementing, etc.) and return to the previous stable state.

Unlike 'abort' which transitions to Failed state, 'stop' returns to a recoverable state:
  - Planning → Loaded (can re-plan)
  - Implementing → Planned (can re-implement)
  - Simplifying → Implemented (can re-simplify)
  - Optimizing → Implemented (can re-optimize)

This allows you to interrupt a long-running operation and continue from a known good state.`,
	RunE: runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	wtPath := socket.WorktreeSocketPath(cwd)

	if !socket.SocketExists(wtPath) {
		return errors.New("no worktree socket running\nRun '" + meta.Name + " start' first")
	}

	client, err := socket.NewClient(wtPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connect to worktree socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "stop", nil)
	if err != nil {
		return fmt.Errorf("stop call: %w", err)
	}

	var result struct {
		Status string `json:"status"`
		State  string `json:"state"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	fmt.Printf("Operation stopped (state: %s)\n", result.State)

	return nil
}
