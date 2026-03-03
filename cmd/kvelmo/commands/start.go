package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/agent"
	"github.com/valksor/kvelmo/pkg/agent/claude"
	"github.com/valksor/kvelmo/pkg/agent/codex"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
	"github.com/valksor/kvelmo/pkg/worker"
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

	// Check if this is a git repository
	if !isGitRepository(cwd) {
		fmt.Println()
		fmt.Println("  Warning: Not a git repository")
		fmt.Println()
		fmt.Println("  Some features will be limited:")
		fmt.Println("    • No branch creation for tasks")
		fmt.Println("    • No checkpoint/undo functionality")
		fmt.Println("    • No PR submission")
		fmt.Println()
		fmt.Println("  Run 'git init' to enable full functionality.")
		fmt.Println()
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

	// Track global socket errors for later reporting
	var globalErrCh chan error
	if !socket.SocketExists(globalPath) {
		release, err := socket.AcquireGlobalLock(socket.GlobalLockPath())
		if err != nil {
			fmt.Println("Waiting for global socket...")
			time.Sleep(500 * time.Millisecond)
		} else {
			fmt.Println("Starting global socket...")
			globalErrCh = make(chan error, 1)
			go func() {
				defer release()
				global := socket.NewGlobalSocket(globalPath)
				globalErrCh <- global.Start(ctx)
			}()
			// Wait briefly to catch immediate startup failures
			select {
			case err := <-globalErrCh:
				if err != nil && !errors.Is(err, context.Canceled) {
					return fmt.Errorf("global socket: %w", err)
				}
			case <-time.After(100 * time.Millisecond):
				// Socket is starting, proceed
			}
		}
	} else {
		fmt.Println("Global socket already running")
	}

	// Create worker pool for standalone mode (same as serve)
	registry := agent.NewRegistry()
	if err := claude.Register(registry); err != nil {
		return fmt.Errorf("register claude agent: %w", err)
	}
	if err := codex.Register(registry); err != nil {
		return fmt.Errorf("register codex agent: %w", err)
	}

	poolCfg := worker.DefaultPoolConfig()
	poolCfg.Agents = registry
	pool := worker.NewPool(poolCfg)
	if err := pool.Start(); err != nil {
		return fmt.Errorf("start worker pool: %w", err)
	}
	defer func() { _ = pool.Stop() }()

	fmt.Printf("Starting worktree socket for %s\n", cwd)
	wt, err := socket.NewWorktreeSocket(socket.WorktreeConfig{
		WorktreePath: cwd,
		SocketPath:   wtPath,
		GlobalPath:   globalPath,
		Pool:         pool,
	})
	if err != nil {
		return fmt.Errorf("create worktree socket: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- wt.Start(ctx)
	}()

	// Wait for socket to be ready before loading task
	if !waitForSocket(wtPath, 5*time.Second) {
		return errors.New("worktree socket failed to start")
	}

	// Add agent worker in background (connecting to Claude/Codex CLI can be slow)
	go func() {
		if _, err := pool.AddAgentWorker(ctx, "", true); err != nil {
			fmt.Printf("Warning: Failed to add agent worker: %v\n", err)
			fmt.Println("Jobs will run in simulation mode.")
			_ = pool.AddDefaultWorker("")
		}
	}()

	// If --from was specified, load the task via RPC
	if startFrom != "" {
		if err := loadTaskViaRPC(wtPath, startFrom); err != nil {
			fmt.Printf("Warning: failed to load task: %v\n", err)
		} else {
			fmt.Printf("Task loaded from %s\n", startFrom)
		}
	}

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
	case err := <-globalErrCh:
		// globalErrCh is nil if we didn't start the global socket, so this case won't fire
		if err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("global socket: %w", err)
		}
	}

	return nil
}

// isGitRepository checks if the given path is inside a git repository.
func isGitRepository(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir") //nolint:noctx // Quick one-shot check

	return cmd.Run() == nil
}

// waitForSocket waits for a socket to become available.
func waitForSocket(socketPath string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if socket.SocketExists(socketPath) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}

	return false
}

// loadTaskViaRPC connects to the worktree socket and loads a task.
func loadTaskViaRPC(socketPath, source string) error {
	client, err := socket.NewClient(socketPath, socket.WithTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("connect to socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	params := map[string]any{
		"source": source,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = client.Call(ctx, "start", params)
	if err != nil {
		return fmt.Errorf("call start: %w", err)
	}

	return nil
}
