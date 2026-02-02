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

// screenshot command.
var (
	screenshotOutput   string
	screenshotFormat   string
	screenshotQuality  int
	screenshotFullPage bool
)

var browserScreenshotCmd = &cobra.Command{
	Use:   "screenshot [url]",
	Short: "Capture screenshot",
	Long: `Capture a screenshot of the current tab or navigate to URL first.

If URL is provided, opens a new tab, navigates to it, then captures screenshot.
Otherwise, captures the currently active tab.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBrowserScreenshot,
}

func init() {
	browserScreenshotCmd.Flags().StringVarP(&screenshotOutput, "output", "o", "", "Output file path")
	browserScreenshotCmd.Flags().StringVarP(&screenshotFormat, "format", "f", "png", "Format (png, jpeg)")
	browserScreenshotCmd.Flags().IntVar(&screenshotQuality, "quality", 80, "JPEG quality (1-100)")
	browserScreenshotCmd.Flags().BoolVarP(&screenshotFullPage, "full-page", "F", false, "Capture full scrollable page")
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

// dom command.
var (
	domSelector string
	domAll      bool
	domHTML     bool
	domLimit    int
	domComputed bool
)

var browserDOMCmd = &cobra.Command{
	Use:   "dom --selector <css>",
	Short: "Query DOM elements",
	Long: `Query DOM elements using CSS selector.

Examples:
  mehr browser dom --selector "h1"
  mehr browser dom --selector ".class" --all
  mehr browser dom --selector "#id" --html`,
	RunE: runBrowserDOM,
}

func init() {
	browserDOMCmd.Flags().StringVar(&domSelector, "selector", "", "CSS selector")
	browserDOMCmd.Flags().BoolVar(&domAll, "all", false, "Return all matching elements")
	browserDOMCmd.Flags().BoolVar(&domHTML, "html", false, "Include outer HTML")
	browserDOMCmd.Flags().IntVar(&domLimit, "limit", 20, "Max elements to return")
	browserDOMCmd.Flags().BoolVar(&domComputed, "computed", false, "Show key computed CSS styles")
	_ = browserDOMCmd.MarkFlagRequired("selector")
}

// click command.
var (
	clickSelector string
)

var browserClickCmd = &cobra.Command{
	Use:   "click --selector <css>",
	Short: "Click an element",
	Long: `Click an element using CSS selector.

Example:
  mehr browser click --selector "button[type='submit']"`,
	RunE: runBrowserClick,
}

func init() {
	browserClickCmd.Flags().StringVar(&clickSelector, "selector", "", "CSS selector")
	_ = browserClickCmd.MarkFlagRequired("selector")
}

// type command.
var (
	typeSelector string
	typeClear    bool
)

var browserTypeCmd = &cobra.Command{
	Use:   "type --selector <css> <text>",
	Short: "Type text into an element",
	Long: `Type text into an input field using CSS selector.

Examples:
  mehr browser type --selector "#input" "Hello World"
  mehr browser type --selector "#input" --clear "New text"`,
	Args: cobra.ExactArgs(1),
	RunE: runBrowserType,
}

func init() {
	browserTypeCmd.Flags().StringVar(&typeSelector, "selector", "", "CSS selector")
	browserTypeCmd.Flags().BoolVar(&typeClear, "clear", false, "Clear field before typing")
	_ = browserTypeCmd.MarkFlagRequired("selector")
}

// eval command.
var browserEvalCmd = &cobra.Command{
	Use:   "eval <expression>",
	Short: "Evaluate JavaScript",
	Long: `Evaluate JavaScript expression in the current tab.

Example:
  mehr browser eval "document.title"`,
	Args: cobra.ExactArgs(1),
	RunE: runBrowserEval,
}

// console command.
var (
	consoleDuration float64
	consoleLevel    string
)

var browserConsoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Capture console logs",
	Long: `Capture console logs from the current tab for a duration.

Note: Full console monitoring requires complex event handling.
This implementation currently returns empty results.`,
	RunE: runBrowserConsole,
}

func init() {
	browserConsoleCmd.Flags().Float64VarP(&consoleDuration, "duration", "d", 1.0, "Capture duration in seconds")
	browserConsoleCmd.Flags().StringVar(&consoleLevel, "level", "", "Filter by log level")
}

// network command.
var (
	networkDuration    float64
	networkType        string
	networkCaptureBody bool
	networkMaxBodySize int
)

var browserNetworkCmd = &cobra.Command{
	Use:   "network",
	Short: "Capture network requests",
	Long: `Capture network requests from the current tab for a duration.

Use --body to also capture request and response bodies.
Bodies are truncated at --max-body-size (default 1MB).`,
	RunE: runBrowserNetwork,
}

func init() {
	browserNetworkCmd.Flags().Float64VarP(&networkDuration, "duration", "d", 3.0, "Capture duration in seconds")
	browserNetworkCmd.Flags().StringVar(&networkType, "type", "", "Filter by resource type")
	browserNetworkCmd.Flags().BoolVar(&networkCaptureBody, "body", false, "Capture request and response bodies")
	browserNetworkCmd.Flags().IntVar(&networkMaxBodySize, "max-body-size", 1024*1024, "Max body size in bytes (default 1MB)")
}

// cookies command group.
var browserCookiesCmd = &cobra.Command{
	Use:   "cookies <subcommand>",
	Short: "Cookie management commands",
	Long:  `Manage browser cookies for session persistence.`,
	RunE:  func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

// cookies export command.
var (
	cookieOutputPath    string
	cookieProfileExport string
)

var browserCookiesExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export cookies to file",
	Long: `Export browser cookies to a JSON file.

If no output path is specified, uses the default cookie storage location
(~/.mehrhof/cookies-{profile}.json). This is useful for backing up cookies
or transferring them between machines.`,
	RunE: runBrowserCookiesExport,
}

func init() {
	browserCookiesExportCmd.Flags().StringVarP(&cookieOutputPath, "output", "o", "", "Output file path (default: ~/.mehrhof/cookies-{profile}.json)")
	browserCookiesExportCmd.Flags().StringVar(&cookieProfileExport, "profile", "", "Cookie profile to export (default: uses --cookie-profile flag)")
}

// cookies import command.
var (
	cookieInputPath     string
	cookieProfileImport string
)

var browserCookiesImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import cookies from file",
	Long: `Import browser cookies from a JSON file.

If no input path is specified, uses the default cookie storage location
(~/.mehrhof/cookies-{profile}.json). This is useful for restoring cookies
or transferring them between machines.`,
	RunE: runBrowserCookiesImport,
}

func init() {
	browserCookiesImportCmd.Flags().StringVarP(&cookieInputPath, "file", "f", "", "Input file path (default: ~/.mehrhof/cookies-{profile}.json)")
	browserCookiesImportCmd.Flags().StringVar(&cookieProfileImport, "profile", "", "Cookie profile to import (default: uses --cookie-profile flag)")
}

func init() {
	browserCookiesCmd.AddCommand(browserCookiesExportCmd)
	browserCookiesCmd.AddCommand(browserCookiesImportCmd)
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

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

func runBrowserScreenshot(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctrl, cleanup, err := setupBrowserController(ctx, 30*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	var tabID string
	if len(args) > 0 {
		// Open a new tab with URL
		tab, err := ctrl.OpenTab(ctx, args[0])
		if err != nil {
			return fmt.Errorf("open tab: %w", err)
		}
		tabID = tab.ID
		defer func() {
			if err := ctrl.CloseTab(ctx, tabID); err != nil {
				fmt.Printf("Warning: close tab failed: %v\n", err)
			}
		}()
	} else {
		// Use the first available tab
		tabs, err := ctrl.ListTabs(ctx)
		if err != nil {
			return fmt.Errorf("list tabs: %w", err)
		}
		if len(tabs) == 0 {
			return errors.New("no tabs open")
		}
		tabID = tabs[0].ID
	}

	opts := browser.ScreenshotOptions{
		Format:   screenshotFormat,
		Quality:  screenshotQuality,
		FullPage: screenshotFullPage,
	}

	data, err := ctrl.Screenshot(ctx, tabID, opts)
	if err != nil {
		return fmt.Errorf("screenshot: %w", err)
	}

	outputPath := screenshotOutput
	if outputPath == "" {
		outputPath = "screenshot." + screenshotFormat
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Printf("Screenshot saved: %s\n", outputPath)

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

	// Use the first available tab
	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}
	if len(tabs) == 0 {
		return errors.New("no tabs open")
	}

	if err := ctrl.Navigate(ctx, tabs[0].ID, url); err != nil {
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

	// Use the first available tab
	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}
	if len(tabs) == 0 {
		return errors.New("no tabs open")
	}

	if err := ctrl.Reload(ctx, tabs[0].ID, reloadHard); err != nil {
		return fmt.Errorf("reload: %w", err)
	}

	fmt.Println("Page reloaded")

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

func runBrowserDOM(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctrl, cleanup, err := setupBrowserController(ctx, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	// Use the first available tab
	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}
	if len(tabs) == 0 {
		return errors.New("no tabs open")
	}

	if domAll {
		elems, err := ctrl.QuerySelectorAll(ctx, tabs[0].ID, domSelector)
		if err != nil {
			return fmt.Errorf("query selector: %w", err)
		}

		if len(elems) == 0 {
			fmt.Printf("No elements found for: %s\n", domSelector)

			return nil
		}

		limit := domLimit
		if limit > len(elems) {
			limit = len(elems)
		}

		for i := range limit {
			elem := elems[i]
			fmt.Printf("%d. <%s>\n", i+1, elem.TagName)
			if elem.TextContent != "" {
				text := elem.TextContent
				if len(text) > 50 {
					text = text[:50] + "..."
				}
				fmt.Printf("   Text: %s\n", text)
			}
			if domHTML && elem.OuterHTML != "" {
				html := elem.OuterHTML
				if len(html) > 100 {
					html = html[:100] + "..."
				}
				fmt.Printf("   HTML: %s\n", html)
			}
		}

		if len(elems) > limit {
			fmt.Printf("\n... and %d more (use --limit to show more)\n", len(elems)-limit)
		}
	} else {
		elem, err := ctrl.QuerySelector(ctx, tabs[0].ID, domSelector)
		if err != nil {
			return fmt.Errorf("query selector: %w", err)
		}

		if elem == nil {
			fmt.Printf("No element found for: %s\n", domSelector)

			return nil
		}

		fmt.Printf("<%s>\n", elem.TagName)
		if elem.TextContent != "" {
			fmt.Printf("Text: %s\n", elem.TextContent)
		}
		if domHTML && elem.OuterHTML != "" {
			fmt.Printf("HTML: %s\n", elem.OuterHTML)
		}
		fmt.Printf("Visible: %v\n", elem.Visible)

		if domComputed {
			printKeyComputedStyles(ctx, ctrl, tabs[0].ID, domSelector)
		}
	}

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

// printKeyComputedStyles shows a subset of important computed CSS properties.
func printKeyComputedStyles(ctx context.Context, ctrl browser.Controller, tabID, selector string) {
	styles, err := ctrl.GetComputedStyles(ctx, tabID, selector)
	if err != nil {
		fmt.Printf("Computed styles: error: %v\n", err)

		return
	}

	// Key properties most useful for quick inspection
	keyProps := map[string]bool{
		"display": true, "position": true, "color": true, "background-color": true,
		"font-size": true, "font-family": true, "font-weight": true,
		"margin-top": true, "margin-right": true, "margin-bottom": true, "margin-left": true,
		"padding-top": true, "padding-right": true, "padding-bottom": true, "padding-left": true,
		"width": true, "height": true, "overflow": true, "z-index": true,
		"opacity": true, "visibility": true, "box-sizing": true,
	}

	fmt.Println("Computed styles:")
	for _, s := range styles {
		if keyProps[s.Name] && s.Value != "" {
			fmt.Printf("  %s: %s\n", s.Name, s.Value)
		}
	}
}

func runBrowserClick(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctrl, cleanup, err := setupBrowserController(ctx, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	// Use the first available tab
	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}
	if len(tabs) == 0 {
		return errors.New("no tabs open")
	}

	if err := ctrl.Click(ctx, tabs[0].ID, clickSelector); err != nil {
		return fmt.Errorf("click: %w", err)
	}

	fmt.Printf("Clicked: %s\n", clickSelector)

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

func runBrowserType(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	text := args[0]

	ctrl, cleanup, err := setupBrowserController(ctx, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	// Use the first available tab
	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}
	if len(tabs) == 0 {
		return errors.New("no tabs open")
	}

	if err := ctrl.Type(ctx, tabs[0].ID, typeSelector, text, typeClear); err != nil {
		return fmt.Errorf("type: %w", err)
	}

	fmt.Printf("Typed into: %s\n", typeSelector)

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

func runBrowserEval(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	expression := args[0]

	ctrl, cleanup, err := setupBrowserController(ctx, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	// Use the first available tab
	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}
	if len(tabs) == 0 {
		return errors.New("no tabs open")
	}

	result, err := ctrl.Eval(ctx, tabs[0].ID, expression)
	if err != nil {
		return fmt.Errorf("eval: %w", err)
	}

	fmt.Printf("Result: %v\n", result)

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

func runBrowserConsole(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	timeout := time.Duration(consoleDuration*float64(time.Second)) + 5*time.Second
	ctrl, cleanup, err := setupBrowserController(ctx, timeout)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	// Use the first available tab
	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}
	if len(tabs) == 0 {
		return errors.New("no tabs open")
	}

	duration := time.Duration(consoleDuration * float64(time.Second))
	messages, err := ctrl.GetConsoleLogs(ctx, tabs[0].ID, duration)
	if err != nil {
		return fmt.Errorf("get console logs: %w", err)
	}

	if len(messages) == 0 {
		fmt.Println("No console messages captured during the monitoring period")

		return nil
	}

	for _, msg := range messages {
		fmt.Printf("[%s] %s\n", msg.Level, msg.Text)
	}

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

func runBrowserNetwork(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	timeout := time.Duration(networkDuration*float64(time.Second)) + 5*time.Second
	ctrl, cleanup, err := setupBrowserController(ctx, timeout)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	// Configure body capture if requested
	if networkCaptureBody {
		ctrl.SetNetworkMonitorOptions(browser.NetworkMonitorOptions{
			CaptureBody: true,
			MaxBodySize: networkMaxBodySize,
		})
	}

	// Use the first available tab
	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}
	if len(tabs) == 0 {
		return errors.New("no tabs open")
	}

	duration := time.Duration(networkDuration * float64(time.Second))
	requests, err := ctrl.GetNetworkRequests(ctx, tabs[0].ID, duration)
	if err != nil {
		return fmt.Errorf("get network requests: %w", err)
	}

	if len(requests) == 0 {
		fmt.Println("No network requests captured during the monitoring period")

		return nil
	}

	for _, req := range requests {
		status := ""
		if req.Status > 0 {
			status = fmt.Sprintf(" -> %d %s", req.Status, req.StatusText)
		}
		fmt.Printf("[%s] %s %s%s\n", req.Timestamp.Format("15:04:05.000"), req.Method, req.URL, status)

		if networkCaptureBody {
			if req.RequestBody != "" {
				fmt.Printf("  Request Body: %s\n", truncateString(req.RequestBody, 200))
			}
			if req.ResponseBody != "" {
				fmt.Printf("  Response Body: %s\n", truncateString(req.ResponseBody, 200))
			}
		}
	}

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

func runBrowserCookiesExport(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Determine profile
	profile := cookieProfileExport
	if profile == "" {
		profile = cookieProfile
	}
	if profile == "" {
		profile = "default"
	}

	ctrl, cleanup, err := setupBrowserController(ctx, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	// Get cookies from the browser
	cookies, err := ctrl.GetCookies(ctx)
	if err != nil {
		return fmt.Errorf("get cookies: %w", err)
	}

	// Determine the output path
	outputPath := cookieOutputPath
	if outputPath == "" {
		cookiePath, err := browser.CookiePath(profile)
		if err != nil {
			return fmt.Errorf("get default cookie path: %w", err)
		}
		outputPath = cookiePath
	}

	// Save cookies
	storage := browser.NewCookieStorage("")
	if err := storage.Save(profile, cookies); err != nil {
		return fmt.Errorf("save cookies: %w", err)
	}

	fmt.Printf("Exported %d cookies from profile '%s' to: %s\n", len(cookies), profile, outputPath)

	return nil
}

func runBrowserCookiesImport(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Determine profile
	profile := cookieProfileImport
	if profile == "" {
		profile = cookieProfile
	}
	if profile == "" {
		profile = "default"
	}

	// Determine the input path
	inputPath := cookieInputPath
	if inputPath == "" {
		cookiePath, err := browser.CookiePath(profile)
		if err != nil {
			return fmt.Errorf("get default cookie path: %w", err)
		}
		inputPath = cookiePath
	}

	// Load cookies from a file
	storage := browser.NewCookieStorage("")
	cookies, err := storage.Load(profile)
	if err != nil {
		return fmt.Errorf("load cookies: %w", err)
	}

	ctrl, cleanup, err := setupBrowserController(ctx, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	// Set cookies in a browser
	if err := ctrl.SetCookies(ctx, cookies); err != nil {
		return fmt.Errorf("set cookies: %w", err)
	}

	fmt.Printf("Imported %d cookies from '%s' to profile '%s'\n", len(cookies), inputPath, profile)

	return nil
}
