package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/browser"
)

var (
	browserHost        string
	browserPort        int
	browserHeadless    bool
	browserStrictCerts bool // Enable strict certificate validation
	browserKeepAlive   bool // Keep browser running after command completes
	cookieProfile      string
	browserTabID       string             // Target a specific tab by ID (for MCP/agent use)
	browserMCPMode     bool               // Set by MCP server to enable session reuse between tool calls
	mcpCtrl            browser.Controller // Cached controller for MCP mode (reused across tool calls)
	mcpCtrlMu          sync.Mutex
)

// SetBrowserMCPMode enables MCP mode for browser commands.
// In MCP mode, a single browser controller is cached and reused across
// sequential MCP tool calls. This prevents WebSocket and goroutine leaks.
// Headless mode is also enabled since AI agents don't need a visible browser.
func SetBrowserMCPMode(enabled bool) {
	browserMCPMode = enabled
	if enabled {
		browserHeadless = true
	}
}

// CleanupBrowserMCP disconnects the cached MCP browser controller.
// Called once when the MCP server shuts down to release Chrome resources.
func CleanupBrowserMCP() {
	mcpCtrlMu.Lock()
	defer mcpCtrlMu.Unlock()

	if mcpCtrl != nil {
		_ = mcpCtrl.Disconnect()
		mcpCtrl = nil
	}
}

var browserCmd = &cobra.Command{
	Use:   "browser <command>",
	Short: "Control Chrome browser for automation",
	Long: `Browser automation using Chrome DevTools Protocol.

By default, launches an isolated Chrome instance on a random port.
This prevents hijacking your active browser session.

Tab Targeting (for MCP/agent use):
  1. List tabs:     mehr browser tabs
  2. Open new tab:  mehr browser goto https://example.com  (returns tab ID)
  3. Target tab:    mehr browser screenshot --tab-id <id>

If --tab-id is omitted, commands target the first open tab (backward compatible).

To keep the browser running for use by AI agents or further commands:
  mehr browser --keep-alive navigate https://example.com

The browser will continue running until you explicitly close it or stop the process.

To use existing Chrome:
  google-chrome --remote-debugging-port=9222
  mehr browser --port 9222 status

For headless automation:
  mehr browser --headless screenshot https://example.com`,
}

func init() {
	rootCmd.AddCommand(browserCmd)

	browserCmd.PersistentFlags().StringVar(&browserHost, "host", "localhost", "CDP host")
	browserCmd.PersistentFlags().IntVar(&browserPort, "port", 0, "CDP port (0 = random, 9222 = existing Chrome)")
	browserCmd.PersistentFlags().BoolVar(&browserHeadless, "headless", false, "Launch headless browser")
	browserCmd.PersistentFlags().BoolVar(&browserStrictCerts, "strict-certs", false, "Enable strict certificate validation (default: ignore)")
	browserCmd.PersistentFlags().BoolVar(&browserKeepAlive, "keep-alive", false, "Keep browser running after command completes")
	browserCmd.PersistentFlags().StringVar(&cookieProfile, "cookie-profile", "default", "Cookie profile to use")
	browserCmd.PersistentFlags().StringVar(&browserTabID, "tab-id", "", "Target a specific tab by ID (use 'browser tabs' to list IDs)")

	browserCmd.AddCommand(browserStatusCmd)
	browserCmd.AddCommand(browserTabsCmd)
	browserCmd.AddCommand(browserGotoCmd)
	browserCmd.AddCommand(browserNavigateCmd)
	browserCmd.AddCommand(browserCloseCmd)
	browserCmd.AddCommand(browserSwitchCmd)
	browserCmd.AddCommand(browserReloadCmd)
	browserCmd.AddCommand(browserScreenshotCmd)
	browserCmd.AddCommand(browserDOMCmd)
	browserCmd.AddCommand(browserClickCmd)
	browserCmd.AddCommand(browserTypeCmd)
	browserCmd.AddCommand(browserEvalCmd)
	browserCmd.AddCommand(browserConsoleCmd)
	browserCmd.AddCommand(browserNetworkCmd)
	browserCmd.AddCommand(browserCookiesCmd)
	browserCmd.AddCommand(browserSourceCmd)
	browserCmd.AddCommand(browserScriptsCmd)
	browserCmd.AddCommand(browserWebSocketCmd)
	browserCmd.AddCommand(browserCoverageCmd)
	browserCmd.AddCommand(browserStylesCmd)
}

// status command.
var browserStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check browser connection status",
	Long:  `Check if connected to a Chrome browser and show active tabs.`,
	RunE:  runBrowserStatus,
}

// tabs command.
var browserTabsCmd = &cobra.Command{
	Use:   "tabs",
	Short: "List browser tabs",
	Long:  `List all open tabs in the browser.`,
	RunE:  runBrowserTabs,
}

// goto command.
var browserGotoCmd = &cobra.Command{
	Use:   "goto <url>",
	Short: "Open URL in new tab",
	Long:  `Open a new browser tab and navigate to the specified URL.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserGoto,
}

// navigate command.
var browserNavigateCmd = &cobra.Command{
	Use:   "navigate <url>",
	Short: "Navigate current tab to URL",
	Long:  `Navigate the currently active tab to the specified URL.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserNavigate,
}

// close command.
var browserCloseCmd = &cobra.Command{
	Use:   "close <tab-id>",
	Short: "Close a tab",
	Long:  `Close a tab by its ID (use 'tabs' command to list IDs).`,
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserClose,
}

// switch command.
var browserSwitchCmd = &cobra.Command{
	Use:   "switch <tab-id>",
	Short: "Switch to a tab",
	Long:  `Switch to a tab by its ID (use 'tabs' command to list IDs).`,
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserSwitch,
}

// reload command.
var (
	reloadHard bool
)

var browserReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload current tab",
	Long:  `Reload the currently active tab. Use --hard to bypass cache.`,
	RunE:  runBrowserReload,
}

func init() {
	browserReloadCmd.Flags().BoolVar(&reloadHard, "hard", false, "Bypass cache")
}

// Command implementations

func runBrowserStatus(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctrl, cleanup, err := setupBrowserController(ctx, 10*time.Second)
	if err != nil {
		fmt.Printf("Not connected: %v\n", err)
		fmt.Println("\nTo launch Chrome with remote debugging:")
		fmt.Println("  google-chrome --remote-debugging-port=9222")
		fmt.Println("\nOr use headless mode:")
		fmt.Println("  mehr browser --headless status")

		return nil
	}
	defer cleanup()

	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}

	fmt.Printf("Connected to Chrome at %s:%d\n", browserHost, getActualPort(ctrl))
	fmt.Printf("Found %d tab(s)\n", len(tabs))
	for i, tab := range tabs {
		fmt.Printf("  %d. %s\n", i+1, truncateString(tab.Title, 60))
		fmt.Printf("     %s\n", tab.URL)
	}

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

func runBrowserTabs(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctrl, cleanup, err := setupBrowserController(ctx, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}

	if len(tabs) == 0 {
		fmt.Println("No tabs open.")

		return nil
	}

	for i, tab := range tabs {
		fmt.Printf("%d. [%s] %s\n", i+1, tab.ID, tab.Title)
		fmt.Printf("   %s\n", tab.URL)
	}

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

func runBrowserGoto(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	url := args[0]

	ctrl, cleanup, err := setupBrowserController(ctx, 30*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	tab, err := ctrl.OpenTab(ctx, url)
	if err != nil {
		return fmt.Errorf("open tab: %w", err)
	}

	fmt.Printf("Opened new tab: %s\n", tab.Title)
	fmt.Printf("URL: %s\n", tab.URL)
	fmt.Printf("Tab ID: %s\n", tab.ID)

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

func runBrowserNavigate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	url := args[0]

	ctrl, cleanup, err := setupBrowserController(ctx, 30*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	tabID, err := resolveTabID(ctx, ctrl, browserTabID)
	if err != nil {
		return err
	}

	if err := ctrl.Navigate(ctx, tabID, url); err != nil {
		return fmt.Errorf("navigate: %w", err)
	}

	fmt.Printf("Navigated to: %s\n", url)

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

func runBrowserClose(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	tabID := args[0]

	ctrl, cleanup, err := setupBrowserController(ctx, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	if err := ctrl.CloseTab(ctx, tabID); err != nil {
		return fmt.Errorf("close tab: %w", err)
	}

	fmt.Printf("Closed tab: %s\n", tabID)

	return nil
}

func runBrowserSwitch(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	tabID := args[0]

	ctrl, cleanup, err := setupBrowserController(ctx, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	tab, err := ctrl.SwitchTab(ctx, tabID)
	if err != nil {
		return fmt.Errorf("switch tab: %w", err)
	}

	fmt.Printf("Switched to: %s\n", tab.Title)
	fmt.Printf("URL: %s\n", tab.URL)

	return nil
}

func runBrowserReload(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctrl, cleanup, err := setupBrowserController(ctx, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	tabID, err := resolveTabID(ctx, ctrl, browserTabID)
	if err != nil {
		return err
	}

	if err := ctrl.Reload(ctx, tabID, reloadHard); err != nil {
		return fmt.Errorf("reload: %w", err)
	}

	fmt.Println("Page reloaded")

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

// Helper functions

// setupBrowserController creates and connects a browser controller with the current configuration.
// It returns the controller and a cleanup function that should be called when done.
// The cleanup function will handle disconnect or no-op depending on the keepAlive setting.
//
// In MCP mode, a single controller is cached and reused across calls. This prevents
// WebSocket/goroutine leaks and preserves tab state between sequential tool invocations.
// Cleanup is handled globally by CleanupBrowserMCP() when the MCP server shuts down.
func setupBrowserController(ctx context.Context, timeout time.Duration) (browser.Controller, func(), error) {
	if browserMCPMode {
		return getOrCreateMCPController(ctx, timeout)
	}

	cfg := browser.Config{
		Host:             browserHost,
		Port:             browserPort,
		Headless:         browserHeadless,
		IgnoreCertErrors: !browserStrictCerts,
		Timeout:          timeout,
	}

	ctrl := browser.NewController(cfg)
	if err := ctrl.Connect(ctx); err != nil {
		return nil, nil, fmt.Errorf("connect: %w", err)
	}

	var cleanup func()
	if !browserKeepAlive {
		cleanup = func() { disconnectWrapper(ctrl) }
	} else {
		setupKeepAliveSignalHandler(ctx, ctrl)
		cleanup = func() {}
	}

	return ctrl, cleanup, nil
}

// getOrCreateMCPController returns the cached controller or creates one on first use.
// Uses context.Background() for Connect because Chrome must outlive individual tool calls.
// exec.CommandContext(ctx) in the session launcher kills Chrome when ctx is canceled,
// and each tool call's context is canceled after the call returns.
func getOrCreateMCPController(_ context.Context, timeout time.Duration) (browser.Controller, func(), error) {
	mcpCtrlMu.Lock()
	defer mcpCtrlMu.Unlock()

	if mcpCtrl != nil {
		return mcpCtrl, func() {}, nil
	}

	cfg := browser.Config{
		Host:             browserHost,
		Port:             browserPort,
		Headless:         browserHeadless,
		IgnoreCertErrors: !browserStrictCerts,
		Timeout:          timeout,
	}

	ctrl := browser.NewController(cfg)
	if err := ctrl.Connect(context.Background()); err != nil { //nolint:contextcheck // Chrome must outlive individual tool call contexts
		return nil, nil, fmt.Errorf("connect: %w", err)
	}

	mcpCtrl = ctrl

	return ctrl, func() {}, nil
}

// disconnectWrapper wraps disconnect with error logging.
func disconnectWrapper(ctrl browser.Controller) {
	if err := ctrl.Disconnect(); err != nil {
		fmt.Printf("Warning: disconnect failed: %v\n", err)
	}
}

// getActualPort returns the actual port being used (for random ports).
func getActualPort(ctrl browser.Controller) int {
	return ctrl.GetPort()
}

// truncateString truncates a string to max length.
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}

	return s[:maxLength-3] + "..."
}

// printKeepAliveMessage prints the keep-alive status message.
func printKeepAliveMessage(ctrl browser.Controller) {
	port := getActualPort(ctrl)
	fmt.Printf("\nBrowser is running in background (port %d)\n", port)
	fmt.Printf("Session saved to: .mehrhof/browser.json\n")
	fmt.Printf("Use Ctrl+C to stop this process, or close the browser window to exit.\n")
}

// setupKeepAliveSignalHandler sets up signal handling for graceful shutdown when using keep-alive.
func setupKeepAliveSignalHandler(ctx context.Context, ctrl browser.Controller) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			fmt.Println("\nShutting down browser...")
			_ = ctrl.Disconnect()
		case <-ctx.Done():
		}
	}()
}

// resolveTabID returns the requested tab ID, or defaults to the first open tab.
// This enables explicit tab targeting via --tab-id flag while maintaining
// backward compatibility (omitting the flag uses the first tab).
func resolveTabID(ctx context.Context, ctrl browser.Controller, tabID string) (string, error) {
	if tabID != "" {
		return tabID, nil
	}

	tabs, err := ctrl.ListTabs(ctx)
	if err != nil || len(tabs) == 0 {
		return "", errors.New("no tabs open")
	}

	return tabs[0].ID, nil
}
