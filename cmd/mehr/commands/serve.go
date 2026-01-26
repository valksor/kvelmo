package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/server"
	"github.com/valksor/go-mehrhof/internal/storage"
)

var (
	servePort       int
	serveHost       string
	serveGlobal     bool
	serveOpen       bool
	serveTunnelInfo bool

	// Register subcommand flags.
	serveRegisterList bool
)

// Auth subcommands.
var serveAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage web UI authentication",
	Long: `Manage user credentials for web UI authentication.

Authentication is required when exposing the server to the network
using --host 0.0.0.0 or a specific IP address.

Examples:
  mehr serve auth add admin mypassword  # Add a user
  mehr serve auth list                  # List all users
  mehr serve auth remove admin          # Remove a user
  mehr serve auth passwd admin newpass  # Change password`,
}

var serveAuthAddCmd = &cobra.Command{
	Use:   "add <username> <password>",
	Short: "Add a user",
	Args:  cobra.ExactArgs(2),
	RunE:  runServeAuthAdd,
}

var serveAuthListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	RunE:  runServeAuthList,
}

var serveAuthRemoveCmd = &cobra.Command{
	Use:   "remove <username>",
	Short: "Remove a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runServeAuthRemove,
}

var serveAuthPasswdCmd = &cobra.Command{
	Use:   "passwd <username> <newpassword>",
	Short: "Change a user's password",
	Args:  cobra.ExactArgs(2),
	RunE:  runServeAuthPasswd,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start web UI server",
	Long: `Start a local web server for task management.

By default, runs in project mode showing the current workspace.
Use --global to see all projects across the system.

The server uses a random available port by default to avoid conflicts.
Use --port to specify a specific port if needed.

Remote Access:
  Use --host 0.0.0.0 to bind to all network interfaces.
  Authentication is required when exposing to the network.
  Use 'mehr serve auth add' to configure users first.

Examples:
  mehr serve                        # Project mode on random port
  mehr serve --port 3000            # Project mode on port 3000
  mehr serve --global               # Global mode (all projects)
  mehr serve --open                 # Open browser automatically
  mehr serve --host 0.0.0.0         # Network accessible (requires auth)
  mehr serve --tunnel-info          # Show SSH tunnel instructions

Subcommands:
  mehr serve register     # Register project for remote access
  mehr serve unregister   # Remove project from registry
  mehr serve auth         # Manage authentication`,
	RunE: runServe,
}

var serveRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register project for remote access",
	Long: `Register the current project in the global registry.

This enables the project to be accessed in remote mode,
where the web UI can be hosted centrally and control projects remotely.

For local global mode (mehr serve --global), all projects are shown
automatically - registration is only needed for remote access.

Examples:
  mehr serve register        # Register current project
  mehr serve register --list # List all registered projects`,
	RunE: runServeRegister,
}

var serveUnregisterCmd = &cobra.Command{
	Use:   "unregister [project-id]",
	Short: "Remove project from remote registry",
	Long: `Remove a project from the global registry.

If no project ID is provided, removes the current project.
This prevents the project from being accessed in remote mode.

Examples:
  mehr serve unregister                      # Remove current project
  mehr serve unregister github.com-user-repo # Remove by project ID`,
	RunE: runServeUnregister,
	Args: cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Main serve flags
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 0, "Server port (0 = random available port)")
	serveCmd.Flags().StringVar(&serveHost, "host", "localhost", "Host to bind to (use 0.0.0.0 for all interfaces)")
	serveCmd.Flags().BoolVar(&serveGlobal, "global", false, "Global mode (show all projects)")
	serveCmd.Flags().BoolVar(&serveOpen, "open", false, "Open browser automatically")
	serveCmd.Flags().BoolVar(&serveTunnelInfo, "tunnel-info", false, "Show SSH tunnel instructions")

	// Register subcommand
	serveCmd.AddCommand(serveRegisterCmd)
	serveRegisterCmd.Flags().BoolVarP(&serveRegisterList, "list", "l", false, "List all registered projects")

	// Unregister subcommand
	serveCmd.AddCommand(serveUnregisterCmd)

	// Auth subcommands
	serveCmd.AddCommand(serveAuthCmd)
	serveAuthCmd.AddCommand(serveAuthAddCmd)
	serveAuthCmd.AddCommand(serveAuthListCmd)
	serveAuthCmd.AddCommand(serveAuthRemoveCmd)
	serveAuthCmd.AddCommand(serveAuthPasswdCmd)
}

func runServe(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Handle --tunnel-info flag: show info and exit without starting server
	if serveTunnelInfo {
		port := servePort
		if port == 0 {
			port = 3000 // Default port for documentation
		}
		fmt.Printf("SSH Tunnel Instructions:\n")
		fmt.Printf("  Access remote serve from your local machine (-L flag):\n")
		fmt.Printf("    ssh -L 8080:localhost:%d user@remote-server\n", port)
		fmt.Printf("    Then open: http://localhost:8080 on YOUR local machine\n")
		fmt.Printf("\n  Access local serve from remote server (-R flag):\n")
		fmt.Printf("    ssh -R 8080:localhost:%d user@remote-server\n", port)
		fmt.Printf("    Then open: http://localhost:8080 on THE REMOTE server\n")

		return nil
	}

	// Create server config
	cfg := server.Config{
		Port: servePort,
		Host: serveHost,
	}

	// Check if auth is required (non-localhost binding)
	requiresAuth := serveHost != "" && serveHost != "localhost" && serveHost != "127.0.0.1"
	if requiresAuth {
		authStore, err := storage.LoadAuthStore()
		if err != nil {
			return fmt.Errorf("load auth store: %w", err)
		}

		if authStore.Count() == 0 {
			return errors.New("authentication required for network access\n\n" +
				"Add a user first:\n" +
				"  mehr serve auth add <username> <password>")
		}

		cfg.AuthStore = authStore
		fmt.Println("Authentication enabled for network access")
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

	// Show security notice for network binding
	if requiresAuth {
		fmt.Println("\nNetwork access enabled - authentication required")
	}

	fmt.Println("\nPress Ctrl+C to stop")

	// Open browser if requested
	if serveOpen {
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
	fmt.Printf("\nThis project can now be accessed in remote mode.\n")

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

// runServeAuthAdd handles "mehr serve auth add".
func runServeAuthAdd(_ *cobra.Command, args []string) error {
	username := args[0]
	password := args[1]

	authStore, err := storage.LoadAuthStore()
	if err != nil {
		return fmt.Errorf("load auth store: %w", err)
	}

	if err := authStore.AddUser(username, password); err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			return fmt.Errorf("user %q already exists", username)
		}

		return fmt.Errorf("add user: %w", err)
	}

	if err := authStore.Save(); err != nil {
		return fmt.Errorf("save auth store: %w", err)
	}

	fmt.Printf("User %q added successfully.\n", username)

	return nil
}

// runServeAuthList handles "mehr serve auth list".
func runServeAuthList(_ *cobra.Command, _ []string) error {
	authStore, err := storage.LoadAuthStore()
	if err != nil {
		return fmt.Errorf("load auth store: %w", err)
	}

	users := authStore.ListUsers()
	if len(users) == 0 {
		fmt.Println("No users configured.")
		fmt.Println("\nUse 'mehr serve auth add <username> <password>' to add a user.")

		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "USERNAME\tCREATED")

	for _, username := range users {
		user, exists := authStore.GetUser(username)
		if !exists {
			continue
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\n",
			user.Username,
			user.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	_ = w.Flush()

	fmt.Printf("\nTotal: %d user(s)\n", len(users))

	return nil
}

// runServeAuthRemove handles "mehr serve auth remove".
func runServeAuthRemove(_ *cobra.Command, args []string) error {
	username := args[0]

	authStore, err := storage.LoadAuthStore()
	if err != nil {
		return fmt.Errorf("load auth store: %w", err)
	}

	if !authStore.RemoveUser(username) {
		fmt.Printf("User %q not found.\n", username)

		return nil
	}

	if err := authStore.Save(); err != nil {
		return fmt.Errorf("save auth store: %w", err)
	}

	fmt.Printf("User %q removed.\n", username)

	return nil
}

// runServeAuthPasswd handles "mehr serve auth passwd".
func runServeAuthPasswd(_ *cobra.Command, args []string) error {
	username := args[0]
	newPassword := args[1]

	authStore, err := storage.LoadAuthStore()
	if err != nil {
		return fmt.Errorf("load auth store: %w", err)
	}

	if err := authStore.UpdatePassword(username, newPassword); err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return fmt.Errorf("user %q not found", username)
		}

		return fmt.Errorf("update password: %w", err)
	}

	if err := authStore.Save(); err != nil {
		return fmt.Errorf("save auth store: %w", err)
	}

	fmt.Printf("Password updated for user %q.\n", username)

	return nil
}
