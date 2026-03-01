package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

var (
	startDaemon  bool
	startVerbose bool
	startFrom    string
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start " + meta.Name + " sockets for the current directory",
	Long: `Start the global socket (if not running) and a worktree socket for the current directory.

By default, this command runs in the foreground and can be stopped with Ctrl+C.
Use --daemon to run in the background.

Use --from to load a task from a source:
  --from file:task.md           Load from local file
  --from github:owner/repo#123  Load GitHub issue/PR
  --from https://github.com/... Load from URL`,
	RunE: runStart,
}

func init() {
	StartCmd.Flags().BoolVarP(&startDaemon, "daemon", "d", false, "Run in background")
	StartCmd.Flags().BoolVarP(&startVerbose, "verbose", "v", false, "Show socket paths")
	StartCmd.Flags().StringVar(&startFrom, "from", "", "Task source (file:path, github:owner/repo#123, or URL)")
}

func runStart(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if err := socket.EnsureDir(); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	globalPath := socket.GlobalSocketPath()
	wtPath := socket.WorktreeSocketPath(cwd)

	if startVerbose {
		fmt.Printf("Global socket: %s\n", globalPath)
		fmt.Printf("Worktree socket: %s\n", wtPath)
	}

	if socket.SocketExists(wtPath) {
		fmt.Printf("Worktree socket already running for %s\n", cwd)

		return nil
	}

	if startDaemon {
		return errors.New("daemon mode not yet implemented")
	}

	return startForeground(cwd, globalPath, wtPath)
}

func startForeground(cwd, globalPath, wtPath string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	if !socket.SocketExists(globalPath) {
		release, err := socket.AcquireGlobalLock(socket.GlobalLockPath())
		if err != nil {
			fmt.Println("Waiting for global socket...")
			time.Sleep(500 * time.Millisecond)
		} else {
			fmt.Println("Starting global socket...")
			go func() {
				defer release()
				global := socket.NewGlobalSocket(globalPath)
				_ = global.Start(ctx)
			}()
			time.Sleep(100 * time.Millisecond)
		}
	} else {
		fmt.Println("Global socket already running")
	}

	fmt.Printf("Starting worktree socket for %s\n", cwd)
	wt, err := socket.NewWorktreeSocket(socket.WorktreeConfig{
		WorktreePath: cwd,
		SocketPath:   wtPath,
		GlobalPath:   globalPath,
		Pool:         nil, // No worker pool in standalone mode
	})
	if err != nil {
		return fmt.Errorf("create worktree socket: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- wt.Start(ctx)
	}()

	fmt.Println("Ready. Press Ctrl+C to stop.")

	select {
	case sig := <-sigCh:
		fmt.Printf("\nReceived %v, shutting down...\n", sig)
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("worktree socket: %w", err)
		}
	}

	return nil
}
