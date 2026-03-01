package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

var DeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete the current task",
	Long: `Clear the current task. Only allowed when the task is in a terminal state
(submitted, failed, or none).`,
	RunE: runDelete,
}

func init() {
	DeleteCmd.Flags().Bool("delete-branch", false, "Also delete the git branch")
}

func runDelete(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	socketPath := socket.WorktreeSocketPath(cwd)

	if !socket.SocketExists(socketPath) {
		return errors.New("no worktree socket running\nRun '" + meta.Name + " start' first")
	}

	client, err := socket.NewClient(socketPath, socket.WithTimeout(10*time.Second))
	if err != nil {
		return fmt.Errorf("connect to socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	deleteBranch, _ := cmd.Flags().GetBool("delete-branch")

	params := map[string]any{
		"delete_branch": deleteBranch,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.Call(ctx, "delete", params)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	fmt.Println("Task deleted")

	return nil
}
