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
)

var LogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show agent activity for the current task",
	Long:  `Display timestamped agent activity log for the active task. This shows the same chat history as 'chat history' but formatted as an activity log with prominent timestamps and truncated messages.`,
	RunE:  runLogs,
}

func init() {
	LogsCmd.Flags().IntP("limit", "n", 50, "Number of messages to show")
	LogsCmd.Flags().Bool("full", false, "Show full message content without truncation")
	LogsCmd.Flags().Bool("json", false, "Output raw JSON")
}

func runLogs(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	globalPath := socket.GlobalSocketPath()

	if !socket.SocketExists(globalPath) {
		return errors.New("no global socket running\nRun '" + meta.Name + " serve' first")
	}

	client, err := socket.NewClient(globalPath, socket.WithTimeout(10*time.Second))
	if err != nil {
		return fmt.Errorf("connect to global socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	wtPath := socket.WorktreeSocketPath(cwd)
	worktreeID := wtPath

	params := map[string]any{
		"worktree_id": worktreeID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "chat.history", params)
	if err != nil {
		return fmt.Errorf("chat.history call: %w", err)
	}

	outputJSON, _ := cmd.Flags().GetBool("json")
	if outputJSON {
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

	var result struct {
		Messages []struct {
			ID        string   `json:"id"`
			Role      string   `json:"role"`
			Content   string   `json:"content"`
			Mentions  []string `json:"mentions,omitempty"`
			Timestamp string   `json:"timestamp,omitempty"`
			JobID     string   `json:"job_id,omitempty"`
		} `json:"messages"`
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	if len(result.Messages) == 0 {
		fmt.Println("No activity logs for this task")

		return nil
	}

	limit, _ := cmd.Flags().GetInt("limit")
	showFull, _ := cmd.Flags().GetBool("full")

	messages := result.Messages
	if limit > 0 && limit < len(messages) {
		messages = messages[len(messages)-limit:]
	}

	fmt.Printf("Activity log (%d messages, showing %d)\n", len(result.Messages), len(messages))
	fmt.Println(strings.Repeat("=", 60))

	for _, msg := range messages {
		roleLabel := "USER"
		switch msg.Role {
		case "assistant":
			roleLabel = "AGENT"
		case "system":
			roleLabel = "SYSTEM"
		}

		timestamp := "          "
		if msg.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, msg.Timestamp); err == nil {
				timestamp = t.Format("15:04:05")
			}
		}

		fmt.Printf("%s  %-6s  ", timestamp, roleLabel)

		content := strings.TrimSpace(msg.Content)
		if !showFull && len(content) > 200 {
			content = content[:200] + " [truncated, use --full to see all]"
		}

		// Replace newlines with indented continuation for compact display
		content = strings.ReplaceAll(content, "\n", "\n                  ")
		fmt.Printf("%s\n", content)
	}

	return nil
}
