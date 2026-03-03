package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

var (
	statusTimeout time.Duration
	statusVerbose bool
	statusJSON    bool
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current task state",
	Long:  "Connect to the worktree socket and display the current task state.",
	RunE:  runStatus,
}

func init() {
	StatusCmd.Flags().DurationVarP(&statusTimeout, "timeout", "t", 5*time.Second, "Connection timeout")
	StatusCmd.Flags().BoolVarP(&statusVerbose, "verbose", "v", false, "Show socket paths")
	StatusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output raw JSON response")
}

func runStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	wtPath := socket.WorktreeSocketPath(cwd)

	if statusVerbose {
		fmt.Printf("Socket: %s\n", wtPath)
	}

	if !socket.SocketExists(wtPath) {
		return fmt.Errorf("no worktree socket running for %s\nRun '"+meta.Name+" start' first", cwd)
	}

	client, err := socket.NewClient(wtPath, socket.WithTimeout(statusTimeout))
	if err != nil {
		return fmt.Errorf("connect to worktree socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), statusTimeout)
	defer cancel()

	resp, err := client.Call(ctx, "status", nil)
	if err != nil {
		return fmt.Errorf("status call: %w", err)
	}

	// --json: output raw JSON
	if statusJSON {
		var pretty interface{}
		if jsonErr := json.Unmarshal(resp.Result, &pretty); jsonErr != nil {
			fmt.Println(string(resp.Result))

			return nil
		}
		out, jsonErr := json.MarshalIndent(pretty, "", "  ")
		if jsonErr != nil {
			fmt.Println(string(resp.Result))

			return nil
		}
		fmt.Println(string(out))

		return nil
	}

	var result socket.StatusResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse status: %w", err)
	}

	fmt.Printf("Path:  %s\n", result.Path)
	fmt.Printf("State: Task: %s\n", capitalize(string(result.State)))

	if result.Task != nil {
		fmt.Printf("Task:  %s - %s\n", result.Task.ID, result.Task.Title)
		fmt.Printf("Source: %s\n", result.Task.Source)
	}

	if result.PendingPromptID != "" {
		fmt.Printf("\n! Quality gate waiting for your input.\n")
		fmt.Printf("  Run: kvelmo quality respond --prompt-id %s [--yes|--no]\n", result.PendingPromptID)
	}

	return nil
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}

	return s
}
