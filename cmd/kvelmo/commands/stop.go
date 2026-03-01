package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/socket"
)

var (
	stopTimeout time.Duration
	stopForce   bool
)

var StopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the worktree socket",
	Long: `Stop the worktree socket for the current directory.

By default, sends a graceful shutdown request and waits for the socket to exit. Use --force to skip the graceful wait and unregister immediately.`,
	RunE: runStop,
}

func init() {
	StopCmd.Flags().DurationVarP(&stopTimeout, "timeout", "t", 2*time.Second, "Graceful shutdown timeout")
	StopCmd.Flags().BoolVarP(&stopForce, "force", "f", false, "Skip graceful shutdown and unregister immediately")
}

func runStop(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	wtPath := socket.WorktreeSocketPath(cwd)
	if !socket.SocketExists(wtPath) {
		fmt.Println("No worktree socket running")

		return nil
	}

	if !stopForce {
		client, err := socket.NewClient(wtPath, socket.WithTimeout(stopTimeout))
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
			_, _ = client.Call(ctx, "shutdown", nil)
			cancel()
			_ = client.Close()

			deadline := time.Now().Add(stopTimeout)
			for time.Now().Before(deadline) {
				if !socket.SocketExists(wtPath) {
					fmt.Printf("Stopped worktree: %s\n", cwd)

					return nil
				}
				time.Sleep(100 * time.Millisecond)
			}
			fmt.Println("Graceful shutdown timed out, forcing...")
		}
	}

	fmt.Printf("Stopped worktree: %s\n", cwd)

	globalPath := socket.GlobalSocketPath()
	if socket.SocketExists(globalPath) {
		client, err := socket.NewClient(globalPath, socket.WithTimeout(time.Second))
		if err == nil {
			defer func() { _ = client.Close() }()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			id := socket.WorktreeIDFromPath(cwd)
			_, _ = client.Call(ctx, "projects.unregister", socket.UnregisterParams{ID: id})
		}
	}

	return nil
}
