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

var TagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage task tags",
	Long:  "Add, remove, or list tags on the current task for categorization and filtering.",
}

var tagAddCmd = &cobra.Command{
	Use:   "add <tag> [tag...]",
	Short: "Add tags to the current task",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runTagAdd,
}

var tagRemoveCmd = &cobra.Command{
	Use:   "remove <tag>",
	Short: "Remove a tag from the current task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTagRemove,
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tags on the current task",
	RunE:  runTagList,
}

func init() {
	TagCmd.AddCommand(tagAddCmd)
	TagCmd.AddCommand(tagRemoveCmd)
	TagCmd.AddCommand(tagListCmd)
}

func runTagAdd(_ *cobra.Command, args []string) error {
	client, cleanup, err := connectWorktree()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "task.tag", map[string]any{
		"action": "add",
		"tags":   args,
	})
	if err != nil {
		return fmt.Errorf("task.tag: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("task.tag: %s", resp.Error.Message)
	}

	fmt.Printf("Added tags: %s\n", strings.Join(args, ", "))

	return nil
}

func runTagRemove(_ *cobra.Command, args []string) error {
	client, cleanup, err := connectWorktree()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "task.tag", map[string]any{
		"action": "remove",
		"tags":   []string{args[0]},
	})
	if err != nil {
		return fmt.Errorf("task.tag: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("task.tag: %s", resp.Error.Message)
	}

	fmt.Printf("Removed tag: %s\n", args[0])

	return nil
}

func runTagList(_ *cobra.Command, _ []string) error {
	client, cleanup, err := connectWorktree()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "task.tag", map[string]any{
		"action": "list",
	})
	if err != nil {
		return fmt.Errorf("task.tag: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("task.tag: %s", resp.Error.Message)
	}

	var result struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	if len(result.Tags) == 0 {
		fmt.Println("No tags.")
	} else {
		for _, tag := range result.Tags {
			fmt.Printf("  %s\n", tag)
		}
	}

	return nil
}

func connectWorktree() (*socket.Client, func(), error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, fmt.Errorf("get working directory: %w", err)
	}

	wtPath := socket.WorktreeSocketPath(cwd)
	if !socket.SocketExists(wtPath) {
		return nil, nil, errors.New("no worktree socket running\nRun '" + meta.Name + " start' first")
	}

	client, err := socket.NewClient(wtPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return nil, nil, fmt.Errorf("connect to worktree socket: %w", err)
	}

	return client, func() { _ = client.Close() }, nil
}
