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

var AbandonCmd = &cobra.Command{
	Use:     "abandon",
	Aliases: []string{"abn"},
	Short:   "Abandon the current task",
	Long: `Stop any running jobs, delete the git branch (unless --keep-branch), and reset state.
If worktree isolation was used, the worktree is also removed.`,
	RunE: runAbandon,
}

func init() {
	AbandonCmd.Flags().Bool("keep-branch", false, "Keep the git branch after abandoning")
}

func runAbandon(cmd *cobra.Command, args []string) error {
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

	keepBranch, _ := cmd.Flags().GetBool("keep-branch")

	params := map[string]any{
		"keep_branch": keepBranch,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.Call(ctx, "abandon", params)
	if err != nil {
		return fmt.Errorf("abandon: %w", err)
	}

	fmt.Println("Task abandoned")
	if !keepBranch {
		fmt.Println("Branch deleted")
	}

	return nil
}
