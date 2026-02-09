package commands

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/platform"
	"github.com/valksor/go-mehrhof/internal/server"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
)

// serveOptions holds options for building server configuration.
type serveOptions struct {
	port          int
	preferredPort int
	global        bool
	apiOnly       bool
}

// DefaultPreferredPort is the default port for the web server.
// Falls back to a random port if this port is already in use.
const DefaultPreferredPort = 6337

var (
	servePort    int
	serveGlobal  bool
	serveOpen    bool
	serveAPIOnly bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start web UI server",
	Long: `Start a local web server for task management.

By default, runs in project mode showing the current workspace.
Use --global to see all projects across the system.

The server listens on port 6337 by default.
If port 6337 is in use, a random available port is selected automatically.

Examples:
  mehr serve                        # Port 6337 (or random if taken)
  mehr serve --port 8080            # Specific port
  mehr serve --global               # Global mode (all projects)
  mehr serve --open                 # Open browser automatically
  mehr serve --api                  # API-only mode (no web UI, for IDE plugins)`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Main serve flags
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 0, "Server port (default: 6337, 0 = random)")
	serveCmd.Flags().BoolVar(&serveGlobal, "global", false, "Global mode (show all projects)")
	serveCmd.Flags().BoolVar(&serveOpen, "open", false, "Open browser automatically")
	serveCmd.Flags().BoolVar(&serveAPIOnly, "api", false, "API-only mode (no web UI)")
}

func runServe(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Build options from flags
	opts := serveOptions{
		port:          servePort,
		preferredPort: DefaultPreferredPort,
		global:        serveGlobal,
		apiOnly:       serveAPIOnly,
	}

	// Resolve port using testable logic (only when port not explicitly set)
	resolvedPort := opts.port
	if !cmd.Flags().Changed("port") {
		resolvedPort = resolveServePort(opts, portAvailable)
		if resolvedPort == 0 && opts.preferredPort > 0 {
			fmt.Printf("Port %d in use, using random available port\n", opts.preferredPort)
		}
	}

	// Create base server config using testable logic
	cfg := buildBaseServerConfig(opts, resolvedPort)

	if serveGlobal {
		fmt.Println("Starting Mehrhof Web UI in global mode...")
	} else {
		cfg.Mode = server.ModeProject

		// Resolve workspace root
		res, err := ResolveWorkspaceRoot(ctx)
		if err != nil {
			return fmt.Errorf("resolve workspace: %w", err)
		}
		cfg.WorkspaceRoot = res.Root

		// Initialize conductor
		opts := BuildConductorOptions(CommandOptions{
			Verbose: verbose,
		})
		cond, err := initializeConductor(ctx, opts...)
		if err != nil {
			return fmt.Errorf("initialize conductor: %w", err)
		}
		cfg.Conductor = cond
		cfg.EventBus = cond.GetEventBus()

		// Auto-register this project for the desktop app and global mode
		if err := registerProjectOnServe(ctx, res.Root); err != nil {
			// Non-fatal: log but continue serving
			fmt.Printf("Warning: failed to register project: %v\n", err)
		}

		fmt.Printf("Starting Mehrhof Web UI for project: %s\n", res.Root)
	}

	// Create server
	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down server...")
		cancel()
	}()

	// Start server in goroutine to get the port
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Wait for server to start and get port
	// Poll with small delay until port is assigned or error occurs
	for srv.Port() <= 0 {
		select {
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Small delay before next check
			<-time.After(10 * time.Millisecond)
		}
	}

	url := srv.URL()
	fmt.Printf("\nServer running at: %s\n", url)
	fmt.Println("\nPress Ctrl+C to stop")

	// Open browser if requested (skip in API-only mode)
	if serveOpen && !serveAPIOnly {
		if err := openBrowser(url); err != nil {
			fmt.Printf("Warning: could not open browser: %v\n", err)
		}
	}

	// Wait for server to finish
	return <-errCh
}

// registerProjectOnServe registers a project in the registry when serve starts.
// This enables the project to appear in the desktop app's project list.
func registerProjectOnServe(ctx context.Context, projectPath string) error {
	// Generate project ID
	projectID, err := storage.GenerateProjectID(ctx, projectPath)
	if err != nil {
		// Use directory name as fallback
		projectID = filepath.Base(projectPath)
	}

	projectName := filepath.Base(projectPath)

	// Try to get remote URL
	var remoteURL string
	if git, err := vcs.New(ctx, projectPath); err == nil {
		if remote, err := git.GetDefaultRemote(ctx); err == nil && remote != "" {
			remoteURL, _ = git.RemoteURL(ctx, remote)
		}
	}

	// Load registry and register project
	registry, err := storage.LoadRegistry()
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	if err := registry.Register(projectID, projectPath, remoteURL, projectName); err != nil {
		return fmt.Errorf("register project: %w", err)
	}

	return registry.Save()
}

// openBrowser opens the specified URL in the default browser.
// Note: We intentionally don't pass a context to exec.CommandContext here because:
// 1. Browser launch should not be cancelled when the server context is cancelled
// 2. The browser process is meant to outlive this command.
func openBrowser(url string) error {
	// WSL: use Windows-side browser via fallback chain
	if platform.IsWSL() {
		// Try wslview first (wslu utility, common on Ubuntu WSL)
		if path, err := exec.LookPath("wslview"); err == nil {
			if err := exec.Command(path, url).Start(); err == nil { //nolint:noctx // browser should outlive command
				return nil
			}
			// wslview found but failed, try next fallback
		}
		// Fallback: explorer.exe (always available in WSL)
		if err := exec.Command("explorer.exe", url).Start(); err == nil { //nolint:noctx // browser should outlive command
			return nil
		}
		// Both failed: return helpful error
		return fmt.Errorf("WSL detected but couldn't open browser. Visit %s manually", url)
	}

	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return exec.Command(cmd, args...).Start() //nolint:noctx // browser should outlive command
}

// portAvailable checks if a port is available for binding.
//

func portAvailable(host string, port int) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	lc := net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		return false
	}
	_ = ln.Close()

	return true
}

// ──────────────────────────────────────────────────────────────────────────────
// Testable logic functions
// ──────────────────────────────────────────────────────────────────────────────

// resolveServePort determines the actual port to use based on options.
// Returns the resolved port.
func resolveServePort(opts serveOptions, checkPort func(string, int) bool) int {
	// If explicit port specified, use it
	if opts.port != 0 {
		return opts.port
	}

	// Try preferred port if available
	if checkPort("localhost", opts.preferredPort) {
		return opts.preferredPort
	}

	// Fall back to random port (0)
	return 0
}

// buildBaseServerConfig creates a base server config from options.
// This does NOT include workspace-specific configuration.
func buildBaseServerConfig(opts serveOptions, port int) server.Config {
	cfg := server.Config{
		Port:    port,
		Host:    "localhost",
		APIOnly: opts.apiOnly,
	}

	if opts.global {
		cfg.Mode = server.ModeGlobal
	} else {
		cfg.Mode = server.ModeProject
	}

	return cfg
}
