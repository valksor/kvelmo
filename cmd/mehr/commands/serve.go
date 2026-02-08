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
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/platform"
	"github.com/valksor/go-mehrhof/internal/server"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// DefaultPreferredPort is the default port for the web server.
// Falls back to a random port if this port is already in use.
const DefaultPreferredPort = 6337

var (
	servePort    int
	serveGlobal  bool
	serveOpen    bool
	serveAPIOnly bool

	// Register subcommand flags.
	serveRegisterList bool
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

var serveRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register project in global registry",
	Long: `Register the current project in the global registry.

Registered projects appear in global mode (mehr serve --global).
Use this to organize projects you want to manage from a single dashboard.

Examples:
  mehr serve register        # Register current project
  mehr serve register --list # List all registered projects`,
	RunE: runServeRegister,
}

var serveUnregisterCmd = &cobra.Command{
	Use:   "unregister [project-id]",
	Short: "Remove project from global registry",
	Long: `Remove a project from the global registry.

If no project ID is provided, removes the current project.
This removes the project from the global mode dashboard.

Examples:
  mehr serve unregister                      # Remove current project
  mehr serve unregister github.com-user-repo # Remove by project ID`,
	RunE: runServeUnregister,
	Args: cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Main serve flags
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 0, "Server port (default: 6337, 0 = random)")
	serveCmd.Flags().BoolVar(&serveGlobal, "global", false, "Global mode (show all projects)")
	serveCmd.Flags().BoolVar(&serveOpen, "open", false, "Open browser automatically")
	serveCmd.Flags().BoolVar(&serveAPIOnly, "api", false, "API-only mode (no web UI)")

	serveCmd.AddCommand(serveRegisterCmd)
	serveRegisterCmd.Flags().BoolVarP(&serveRegisterList, "list", "l", false, "List all registered projects")
	serveCmd.AddCommand(serveUnregisterCmd)
}

func runServe(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Handle default port with smart fallback
	if !cmd.Flags().Changed("port") {
		// Try preferred port 6337, fall back to random if taken
		if portAvailable("localhost", DefaultPreferredPort) {
			servePort = DefaultPreferredPort
		} else {
			fmt.Printf("Port %d in use, using random available port\n", DefaultPreferredPort)
			servePort = 0
		}
	}

	// Create server config
	cfg := server.Config{
		Port:    servePort,
		Host:    "localhost",
		APIOnly: serveAPIOnly,
	}

	if serveGlobal {
		cfg.Mode = server.ModeGlobal
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

// runServeRegister handles the "mehr serve register" command.
func runServeRegister(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Handle --list flag
	if serveRegisterList {
		return listRegisteredProjects()
	}

	// Resolve workspace root
	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}

	// Generate project ID
	projectID, err := storage.GenerateProjectID(ctx, res.Root)
	if err != nil {
		return fmt.Errorf("generate project ID: %w", err)
	}

	// Get remote URL if available
	var remoteURL string
	if res.Git != nil {
		remote, err := res.Git.GetDefaultRemote(ctx)
		if err == nil && remote != "" {
			remoteURL, _ = res.Git.RemoteURL(ctx, remote)
		}
	}
	remoteURL = storage.SanitizeRemoteURL(remoteURL)

	// Get project name from directory or remote
	name := filepath.Base(res.Root)
	if remoteURL != "" {
		// Extract repo name from remote URL
		name = extractRepoName(remoteURL)
	}

	// Load registry and register
	registry, err := storage.LoadRegistry()
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	if err := registry.Register(projectID, res.Root, remoteURL, name); err != nil {
		return fmt.Errorf("register project: %w", err)
	}

	if err := registry.Save(); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	fmt.Printf("Registered project: %s\n", projectID)
	fmt.Printf("  Path: %s\n", res.Root)
	if remoteURL != "" {
		fmt.Printf("  Remote: %s\n", remoteURL)
	}
	fmt.Printf("\nThis project can now be accessed in global mode.\n")

	return nil
}

// runServeUnregister handles the "mehr serve unregister" command.
func runServeUnregister(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var projectID string

	if len(args) > 0 {
		// Use provided project ID
		projectID = args[0]
	} else {
		// Use current project
		res, err := ResolveWorkspaceRoot(ctx)
		if err != nil {
			return err
		}

		projectID, err = storage.GenerateProjectID(ctx, res.Root)
		if err != nil {
			return fmt.Errorf("generate project ID: %w", err)
		}
	}

	// Load registry and unregister
	registry, err := storage.LoadRegistry()
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	if !registry.Unregister(projectID) {
		fmt.Printf("Project not found in registry: %s\n", projectID)

		return nil
	}

	if err := registry.Save(); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	fmt.Printf("Unregistered project: %s\n", projectID)

	return nil
}

// listRegisteredProjects lists all registered projects.
func listRegisteredProjects() error {
	registry, err := storage.LoadRegistry()
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	projects := registry.List()
	if len(projects) == 0 {
		fmt.Println("No projects registered.")
		fmt.Println("\nUse 'mehr serve register' in a project directory to register it.")

		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PROJECT ID\tNAME\tPATH\tREGISTERED")

	for _, p := range projects {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			p.ID,
			p.Name,
			truncatePath(p.Path, 40),
			p.RegisteredAt.Format("2006-01-02"),
		)
	}

	_ = w.Flush()

	fmt.Printf("\nTotal: %d project(s)\n", len(projects))

	return nil
}

// extractRepoName extracts the repository name from a git remote URL.
func extractRepoName(url string) string {
	// Handle various URL formats
	// https://github.com/user/repo.git -> repo
	// git@github.com:user/repo.git -> repo

	// Remove .git suffix
	url = trimSuffix(url, ".git")

	// Get the last path component
	parts := splitAny(url, "/:")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return url
}

// truncatePath truncates a path to fit within maxLen characters.
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}

	return "..." + path[len(path)-maxLen+3:]
}

// trimSuffix removes suffix from s if present.
func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}

	return s
}

// splitAny splits s by any character in sep.
func splitAny(s string, sep string) []string {
	var result []string
	start := 0

	for i, c := range s {
		for _, sc := range sep {
			if c == sc {
				if i > start {
					result = append(result, s[start:i])
				}
				start = i + 1

				break
			}
		}
	}

	if start < len(s) {
		result = append(result, s[start:])
	}

	return result
}
