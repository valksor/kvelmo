package commands

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/mcp"
	"github.com/valksor/go-mehrhof/internal/registration"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/cli"
	"github.com/valksor/go-toolkit/log"
)

var mcpCmd = &cobra.Command{
	Use:    "mcp",
	Short:  "Start MCP server (for AI agents)",
	Long:   `Starts the Model Context Protocol (MCP) server over stdio, allowing AI agents to call Mehrhof commands.`,
	RunE:   runMCPServer,
	Hidden: false, // Visible in help
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMCPServer(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Disable logging to stderr - MCP protocol uses stdout for JSON-RPC
	// and stderr logging can interfere with client parsing.
	// Configure both go-toolkit logger and default slog to discard output.
	log.Configure(log.Options{
		Output: io.Discard,
	})
	slog.SetDefault(slog.New(slog.DiscardHandler))

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Use WaitGroup to ensure signal handler goroutine completes before function returns
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sigCh
		cancel()
		// Note: cancel() is non-blocking (just closes a channel), so this goroutine
		// will not hang even if the context cancellation takes time to propagate.
	}()
	defer wg.Wait() // Wait for signal handler to complete before returning

	// Create tool registry and register commands
	toolRegistry := mcp.NewToolRegistry(rootCmd)

	// Register safe commands for agent use
	registerSafeCommands(toolRegistry)

	// Register workspace data access tools
	mcp.RegisterWorkspaceTools(toolRegistry)

	// Initialize conductor for registry tools
	// Note: We need a minimal conductor setup just to get registries
	// Don't call Initialize() to avoid loading workspace state
	cond, err := conductor.New()
	if err != nil {
		// Conductor initialization is required for registry tools
		// We fail-fast here to avoid partial functionality
		return fmt.Errorf("conductor initialization failed: %w", err)
	}

	// Register standard providers and agents
	registration.RegisterStandardProviders(cond)
	if err := registration.RegisterStandardAgents(cond); err != nil {
		return fmt.Errorf("agent registration failed: %w", err)
	}

	// Register registry tools
	mcp.RegisterRegistryTools(toolRegistry, cond)

	// Load workspace config for MCP settings (rate limiting, etc.)
	var serverOpts []mcp.ServerOption
	ws, err := storage.OpenWorkspace(ctx, ".", nil)
	if err != nil {
		log.Warn("Failed to open workspace, using default config", "error", err)
	}
	if ws != nil {
		cfg, err := ws.LoadConfig()
		if err != nil {
			log.Warn("Failed to load workspace config, using defaults", "error", err)
			cfg = storage.NewDefaultWorkspaceConfig()
		}
		if cfg.MCP != nil && cfg.MCP.RateLimit != nil {
			rate := cfg.MCP.RateLimit.Rate
			burst := cfg.MCP.RateLimit.Burst
			if rate > 0 && burst > 0 {
				serverOpts = append(serverOpts, mcp.WithRateLimit(rate, burst))
				log.Info("MCP server: using custom rate limit from config",
					"rate", rate, "burst", burst)
			}
		}
	}

	// Create MCP server
	server := mcp.NewServer(toolRegistry, serverOpts...)

	// Start server (blocks until context canceled)
	if err := server.Serve(ctx); err != nil {
		return fmt.Errorf("MCP server error: %w", err)
	}

	log.Info("MCP server stopped")

	return nil
}

// registerSafeCommands registers commands that are safe for AI agents to use.
func registerSafeCommands(registry *mcp.ToolRegistry) {
	// Get commands from root command
	var commandsToRegister []*cobra.Command

	// Status and info commands
	commandsToRegister = append(commandsToRegister, statusCmd)
	commandsToRegister = append(commandsToRegister, listCmd)
	commandsToRegister = append(commandsToRegister, guideCmd)
	commandsToRegister = append(commandsToRegister, cli.NewVersionCommand("mehr"))

	// Browser commands
	commandsToRegister = append(commandsToRegister, browserCmd)
	commandsToRegister = append(commandsToRegister, browserStatusCmd)
	commandsToRegister = append(commandsToRegister, browserTabsCmd)
	commandsToRegister = append(commandsToRegister, browserGotoCmd)
	commandsToRegister = append(commandsToRegister, browserNavigateCmd)
	commandsToRegister = append(commandsToRegister, browserCloseCmd)
	commandsToRegister = append(commandsToRegister, browserSwitchCmd)
	commandsToRegister = append(commandsToRegister, browserReloadCmd)
	commandsToRegister = append(commandsToRegister, browserScreenshotCmd)
	commandsToRegister = append(commandsToRegister, browserDOMCmd)
	commandsToRegister = append(commandsToRegister, browserClickCmd)
	commandsToRegister = append(commandsToRegister, browserTypeCmd)
	commandsToRegister = append(commandsToRegister, browserEvalCmd)
	commandsToRegister = append(commandsToRegister, browserConsoleCmd)
	commandsToRegister = append(commandsToRegister, browserNetworkCmd)

	// Config commands (read-only)
	commandsToRegister = append(commandsToRegister, configCmd)

	// Info and discovery commands
	commandsToRegister = append(commandsToRegister, providersCmd)
	commandsToRegister = append(commandsToRegister, templatesCmd)
	commandsToRegister = append(commandsToRegister, agentsCmd)
	commandsToRegister = append(commandsToRegister, costCmd)

	// Register all commands with default arg mapper
	registry.RegisterCommands(commandsToRegister, mcp.DefaultArgMapper)
}
