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

var CheckpointsCmd = &cobra.Command{
	Use:   "checkpoints",
	Short: "List checkpoint history",
	Long: `List all checkpoints for the current task.
Checkpoints are created after each agent operation (plan, implement, etc.)
and can be navigated with 'undo' and 'redo' commands.`,
	RunE: runCheckpoints,
}

func runCheckpoints(cmd *cobra.Command, args []string) error {
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

	resp, err := client.Call(ctx, "checkpoints", nil)
	if err != nil {
		return fmt.Errorf("checkpoints call: %w", err)
	}

	// CheckpointInfo matches the socket response structure
	type CheckpointInfo struct {
		SHA       string `json:"sha"`
		Message   string `json:"message"`
		Author    string `json:"author"`
		Timestamp string `json:"timestamp"`
	}

	var result struct {
		Checkpoints []CheckpointInfo `json:"checkpoints"`
		RedoStack   []CheckpointInfo `json:"redo_stack"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	if len(result.Checkpoints) == 0 {
		fmt.Println("No checkpoints")

		return nil
	}

	fmt.Println("Checkpoints (oldest to newest):")
	for i, cp := range result.Checkpoints {
		marker := "  "
		if i == len(result.Checkpoints)-1 {
			marker = "* " // Current position
		}
		shortSHA := cp.SHA
		if len(shortSHA) > 8 {
			shortSHA = shortSHA[:8]
		}
		if cp.Message != "" {
			fmt.Printf("%s%d. %s - %s\n", marker, i+1, shortSHA, cp.Message)
		} else {
			fmt.Printf("%s%d. %s\n", marker, i+1, shortSHA)
		}
	}

	if len(result.RedoStack) > 0 {
		fmt.Printf("\nRedo stack: %d checkpoint(s) available\n", len(result.RedoStack))
	}

	return nil
}

var checkpointsGotoCmd = &cobra.Command{
	Use:   "goto <sha>",
	Short: "Jump to a specific checkpoint SHA",
	Args:  cobra.ExactArgs(1),
	RunE:  runCheckpointsGoto,
}

func init() {
	CheckpointsCmd.AddCommand(checkpointsGotoCmd)
}

func runCheckpointsGoto(cmd *cobra.Command, args []string) error {
	sha := args[0]

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

	resp, err := client.Call(ctx, "checkpoint.goto", map[string]any{"sha": sha})
	if err != nil {
		return fmt.Errorf("checkpoint.goto call: %w", err)
	}

	var result struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	fmt.Printf("Moved to checkpoint %s\n", result.SHA[:8])

	return nil
}
