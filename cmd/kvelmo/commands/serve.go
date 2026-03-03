package commands

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/agent"
	"github.com/valksor/kvelmo/pkg/agent/claude"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
	"github.com/valksor/kvelmo/pkg/web"
	"github.com/valksor/kvelmo/pkg/worker"
)

// DefaultPreferredPort is the default port for the web server.
// Falls back to a random port if this port is already in use.
const DefaultPreferredPort = 6337

var (
	servePort    int
	serveStatic  string
	serveVerbose bool
	serveOpen    bool
)

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start " + meta.Name + " web UI",
	RunE:  runServe,
}

func init() {
	ServeCmd.Long = fmt.Sprintf(`Start the %[1]s web UI server.

The web UI provides a project picker to manage and connect to projects.
Projects are added/removed via the web interface.

The server listens on port 6337 by default.
If port 6337 is in use, a random available port is selected automatically.

Examples:
  %[1]s serve              # Port 6337 (or random if taken)
  %[1]s serve --port 8080  # Specific port
  %[1]s serve --open       # Open browser automatically`, meta.Name)
	ServeCmd.Flags().IntVarP(&servePort, "port", "p", 0, "Server port (default: 6337, 0 = random)")
	ServeCmd.Flags().StringVar(&serveStatic, "static", "", "Static files directory")
	ServeCmd.Flags().BoolVarP(&serveVerbose, "verbose", "v", false, "Verbose output")
	ServeCmd.Flags().BoolVar(&serveOpen, "open", false, "Open browser automatically")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Pre-flight check: verify at least one AI agent CLI is available
	if !hasAgentCLI() {
		fmt.Println()
		fmt.Println("  Warning: No AI agent CLI found in PATH (claude or codex)")
		fmt.Println()
		fmt.Println("  kvelmo uses AI agents to plan and implement tasks.")
		fmt.Println("  Install Claude CLI: https://docs.anthropic.com/en/docs/claude-code/getting-started")
		fmt.Println("  Or install Codex CLI: https://help.openai.com/en/articles/11096431-openai-codex-cli-getting-started")
		fmt.Println()
		// Continue anyway - user might configure an agent later
	}

	// Resolve port (6337 preferred, fallback to random)
	port := resolvePort(cmd, servePort)

	// Find static directory
	staticDir := findStaticDir(serveStatic)

	if serveVerbose && staticDir != "" {
		fmt.Printf("Static files: %s\n", staticDir)
	}

	// Ensure socket directories exist
	if err := socket.EnsureDir(); err != nil {
		return fmt.Errorf("create socket directories: %w", err)
	}

	// Create context for coordinated shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start GlobalSocket with worker pool
	globalPath := socket.GlobalSocketPath()
	lockPath := socket.GlobalLockPath()
	var globalSocket *socket.GlobalSocket

	// Try to acquire lock - if we get it, we're the primary instance
	release, err := socket.AcquireGlobalLock(lockPath)
	if err != nil {
		// Another instance has the lock
		if serveVerbose {
			fmt.Println("Global socket already running (another instance)")
		}
	} else {
		// We got the lock - clean up any stale socket and start fresh
		if socket.SocketExists(globalPath) {
			_ = os.Remove(globalPath)
		}

		// Create worker pool with Claude agent registered
		registry := agent.NewRegistry()
		if err := claude.Register(registry); err != nil {
			return fmt.Errorf("register claude agent: %w", err)
		}

		poolCfg := worker.DefaultPoolConfig()
		poolCfg.Agents = registry
		pool := worker.NewPool(poolCfg)
		if err := pool.Start(); err != nil {
			return fmt.Errorf("start worker pool: %w", err)
		}

		// Ensure pool cleanup on early return
		var poolCleaned bool
		defer func() {
			if !poolCleaned && globalSocket == nil {
				_ = pool.Stop()
			}
		}()

		// Add a default worker with auto-detected agent (actually connects to Claude/Codex CLI)
		// The default worker cannot be removed
		// Empty string for agent name triggers auto-detection (claude > codex > error)
		_, err := pool.AddAgentWorker(ctx, "", true)
		if err != nil {
			fmt.Printf("Warning: Failed to add agent worker: %v\n", err)
			fmt.Println("Jobs will run in simulation mode until a worker is connected.")
			// Fall back to default worker (simulation mode)
			_ = pool.AddDefaultWorker("")
		}

		// Create global socket with pool
		globalSocket = socket.NewGlobalSocketWithPool(globalPath, pool)
		poolCleaned = true // Pool is now managed by globalSocket

		go func() {
			defer release()
			if err := globalSocket.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
				fmt.Printf("Global socket error: %v\n", err)
			}
		}()
		time.Sleep(100 * time.Millisecond)

		// Pre-warm memory adapter so Cybertron model download completes before
		// the first memory.search call rather than causing a cold-start delay.
		socket.PrewarmMemory(ctx)
	}

	// Create web server with worktree creator (if we own the global socket)
	var webOpts []web.ServerOption
	if globalSocket != nil {
		webOpts = append(webOpts, web.WithWorktreeCreator(globalSocket))
	}
	webServer, err := web.NewServer(staticDir, port, webOpts...)
	if err != nil {
		return fmt.Errorf("create web server: %w", err)
	}

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start web server
	go func() {
		fmt.Printf("\n  %s running at %s\n\n", meta.Name, webServer.URL())
		if err := webServer.Start(); err != nil {
			fmt.Printf("Web server error: %v\n", err)
		}
	}()

	// Open browser if requested
	if serveOpen {
		openBrowser(webServer.URL())
	}

	// Wait for signal
	<-sigCh
	fmt.Println("\nShutting down...")

	// Cancel context to stop all components
	cancel()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	_ = webServer.Shutdown(shutdownCtx)

	if globalSocket != nil {
		_ = globalSocket.Stop()
	}

	fmt.Println("Goodbye.")

	return nil
}

// resolvePort determines the actual port to use.
// If port is explicitly set, use it. Otherwise try preferred port, fallback to random.
func resolvePort(cmd *cobra.Command, explicit int) int {
	// If explicit port specified via flag, use it
	if cmd.Flags().Changed("port") {
		return explicit
	}

	// Try preferred port
	if portAvailable("localhost", DefaultPreferredPort) {
		return DefaultPreferredPort
	}

	// Fallback to random
	fmt.Printf("Port %d in use, using random available port\n", DefaultPreferredPort)

	return 0
}

// portAvailable checks if a port is available for binding.
func portAvailable(host string, port int) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	ln, err := net.Listen("tcp", addr) //nolint:noctx // Quick port check, no context needed
	if err != nil {
		return false
	}
	_ = ln.Close()

	return true
}

// findStaticDir locates the static files directory.
func findStaticDir(explicit string) string {
	if explicit != "" {
		return explicit
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	candidates := []string{
		filepath.Join(cwd, "web", "dist"),
		"web/dist",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}

	return ""
}

// openBrowser opens the specified URL in the default browser.
//
//nolint:noctx // Fire-and-forget command, no context needed
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch {
	case fileExists("/usr/bin/open"): // macOS
		cmd = exec.Command("/usr/bin/open", url)
	case fileExists("/usr/bin/xdg-open"): // Linux
		cmd = exec.Command("/usr/bin/xdg-open", url)
	default:
		// Fallback: try "open" from PATH (macOS) or "xdg-open" (Linux)
		if path, err := exec.LookPath("open"); err == nil {
			cmd = exec.Command(path, url)
		} else if path, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.Command(path, url)
		} else {
			return
		}
	}

	_ = cmd.Start()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// hasAgentCLI checks if at least one AI agent CLI is available in PATH.
func hasAgentCLI() bool {
	// Check for Claude CLI
	if _, err := exec.LookPath("claude"); err == nil {
		return true
	}

	// Check for Codex CLI
	if _, err := exec.LookPath("codex"); err == nil {
		return true
	}

	return false
}
