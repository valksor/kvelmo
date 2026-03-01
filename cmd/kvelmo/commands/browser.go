package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/browser"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

var (
	browserTimeout time.Duration
	browserSession string
)

// BrowserCmd is the root command for browser automation.
var BrowserCmd = &cobra.Command{
	Use:   "browser",
	Short: "Browser automation via playwright-cli",
}

var browserInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install or update browser runtime",
	RunE:  runBrowserInstall,
}

var browserStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show browser runtime status",
	RunE:  runBrowserStatus,
}

var browserExecCmd = &cobra.Command{
	Use:   "exec [command] [args...]",
	Short: "Execute a playwright-cli command",
	Long: `Execute any playwright-cli command with automatic state management.

Common commands:
  open [url]      Open browser, optionally navigate to URL
  goto <url>      Navigate to URL
  click <ref>     Click element by reference
  type <text>     Type text into focused element
  screenshot      Take screenshot
  network         List network requests
  console [level] Show console messages
  close           Close browser

See playwright-cli documentation for full command list.`,
	Args:               cobra.MinimumNArgs(1),
	RunE:               runBrowserExec,
	DisableFlagParsing: true,
}

var browserConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or modify browser configuration",
	RunE:  runBrowserConfig,
}

var browserConfigSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a browser configuration value.

Keys:
  headless   true/false - Run browser without visible window (default: true)
  browser    chromium/firefox/webkit - Browser type (default: chromium)
  profile    name - Default auth profile name (default: default)
  timeout    seconds - Operation timeout (default: 30)`,
	Args: cobra.ExactArgs(2),
	RunE: runBrowserConfigSet,
}

// Agent-friendly browser commands (non-interactive, JSON output)

var browserNavigateCmd = &cobra.Command{
	Use:   "navigate <url>",
	Short: "Navigate to a URL",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserNavigate,
}

var browserSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Capture accessibility snapshot of current page",
	RunE:  runBrowserSnapshot,
}

var (
	screenshotOutput   string
	screenshotFullPage bool
)

var browserScreenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Take a screenshot of the current page",
	RunE:  runBrowserScreenshot,
}

var browserClickCmd = &cobra.Command{
	Use:   "click <selector>",
	Short: "Click an element",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserClick,
}

var browserTypeCmd = &cobra.Command{
	Use:   "type <selector> <text>",
	Short: "Type text into an element",
	Args:  cobra.ExactArgs(2),
	RunE:  runBrowserType,
}

var browserWaitTimeout int

var browserWaitCmd = &cobra.Command{
	Use:   "wait <selector>",
	Short: "Wait for an element to appear",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserWait,
}

var browserEvalCmd = &cobra.Command{
	Use:   "eval <javascript>",
	Short: "Evaluate JavaScript in the browser",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserEval,
}

var browserConsoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Show browser console messages",
	RunE:  runBrowserConsole,
}

var browserNetworkCmd = &cobra.Command{
	Use:   "network",
	Short: "Show network requests",
	RunE:  runBrowserNetwork,
}

var browserFillCmd = &cobra.Command{
	Use:   "fill <selector> <value>",
	Short: "Clear input and set value (more efficient than type)",
	Args:  cobra.ExactArgs(2),
	RunE:  runBrowserFill,
}

var browserSelectCmd = &cobra.Command{
	Use:   "select <selector> <value...>",
	Short: "Select option(s) from a dropdown",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runBrowserSelect,
}

var browserHoverCmd = &cobra.Command{
	Use:   "hover <selector>",
	Short: "Hover over an element",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserHover,
}

var browserFocusCmd = &cobra.Command{
	Use:   "focus <selector>",
	Short: "Focus an element",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserFocus,
}

var (
	scrollAmount   int
	scrollSelector string
)

var browserScrollCmd = &cobra.Command{
	Use:   "scroll <direction>",
	Short: "Scroll the page or element (up, down, left, right)",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserScroll,
}

var browserPressSelector string

var browserPressCmd = &cobra.Command{
	Use:   "press <key>",
	Short: "Press a key or key combination (e.g., Enter, Escape, Control+a)",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserPress,
}

var browserBackCmd = &cobra.Command{
	Use:   "back",
	Short: "Navigate back in browser history",
	RunE:  runBrowserBack,
}

var browserForwardCmd = &cobra.Command{
	Use:   "forward",
	Short: "Navigate forward in browser history",
	RunE:  runBrowserForward,
}

var browserReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload the current page",
	RunE:  runBrowserReload,
}

var browserDialogText string

var browserDialogCmd = &cobra.Command{
	Use:   "dialog <action>",
	Short: "Handle alert/confirm/prompt dialogs (accept or dismiss)",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowserDialog,
}

var browserUploadCmd = &cobra.Command{
	Use:   "upload <selector> <file...>",
	Short: "Upload files to a file input",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runBrowserUpload,
}

var (
	pdfOutput    string
	pdfFormat    string
	pdfLandscape bool
)

var browserPDFCmd = &cobra.Command{
	Use:   "pdf",
	Short: "Generate a PDF of the current page",
	RunE:  runBrowserPDF,
}

func init() {
	BrowserCmd.Long = fmt.Sprintf(`Browser automation powered by playwright-cli.

The browser runtime (Node.js + playwright-cli) is automatically downloaded
on first use. No system dependencies required.

State is layered:
  - Global profiles (~/%s/browser-profiles/) for shared auth
  - Per-worktree state for project isolation`, meta.GlobalDir)
	browserInstallCmd.Long = fmt.Sprintf(`Download Node.js and playwright-cli to ~/%s/runtime/

This is run automatically on first use, but can be invoked manually to
force an update or pre-install the runtime.`, meta.GlobalDir)
	// Global flags
	BrowserCmd.PersistentFlags().DurationVarP(&browserTimeout, "timeout", "t", 30*time.Second, "Command timeout")
	BrowserCmd.PersistentFlags().StringVarP(&browserSession, "session", "s", "", "Session name for persistence")

	// Screenshot flags
	browserScreenshotCmd.Flags().StringVarP(&screenshotOutput, "output", "o", "", "Output file path")
	browserScreenshotCmd.Flags().BoolVar(&screenshotFullPage, "full-page", false, "Capture full scrollable page")

	// Wait flags
	browserWaitCmd.Flags().IntVar(&browserWaitTimeout, "timeout-ms", 30000, "Wait timeout in milliseconds")

	// Scroll flags
	browserScrollCmd.Flags().IntVarP(&scrollAmount, "amount", "a", 0, "Scroll amount in pixels")
	browserScrollCmd.Flags().StringVarP(&scrollSelector, "element", "e", "", "Element to scroll (scrolls page if not specified)")

	// Press flags
	browserPressCmd.Flags().StringVarP(&browserPressSelector, "element", "e", "", "Element to focus before pressing key")

	// Dialog flags
	browserDialogCmd.Flags().StringVarP(&browserDialogText, "text", "t", "", "Text to enter for prompt dialogs")

	// PDF flags
	browserPDFCmd.Flags().StringVarP(&pdfOutput, "output", "o", "", "Output file path")
	browserPDFCmd.Flags().StringVarP(&pdfFormat, "format", "f", "", "Paper format (A4, Letter, etc.)")
	browserPDFCmd.Flags().BoolVar(&pdfLandscape, "landscape", false, "Landscape orientation")

	// Subcommands
	BrowserCmd.AddCommand(browserInstallCmd)
	BrowserCmd.AddCommand(browserStatusCmd)
	BrowserCmd.AddCommand(browserExecCmd)
	BrowserCmd.AddCommand(browserConfigCmd)
	browserConfigCmd.AddCommand(browserConfigSetCmd)

	// Agent-friendly subcommands
	BrowserCmd.AddCommand(browserNavigateCmd)
	BrowserCmd.AddCommand(browserSnapshotCmd)
	BrowserCmd.AddCommand(browserScreenshotCmd)
	BrowserCmd.AddCommand(browserClickCmd)
	BrowserCmd.AddCommand(browserTypeCmd)
	BrowserCmd.AddCommand(browserWaitCmd)
	BrowserCmd.AddCommand(browserEvalCmd)
	BrowserCmd.AddCommand(browserConsoleCmd)
	BrowserCmd.AddCommand(browserNetworkCmd)
	BrowserCmd.AddCommand(browserFillCmd)
	BrowserCmd.AddCommand(browserSelectCmd)
	BrowserCmd.AddCommand(browserHoverCmd)
	BrowserCmd.AddCommand(browserFocusCmd)
	BrowserCmd.AddCommand(browserScrollCmd)
	BrowserCmd.AddCommand(browserPressCmd)
	BrowserCmd.AddCommand(browserBackCmd)
	BrowserCmd.AddCommand(browserForwardCmd)
	BrowserCmd.AddCommand(browserReloadCmd)
	BrowserCmd.AddCommand(browserDialogCmd)
	BrowserCmd.AddCommand(browserUploadCmd)
	BrowserCmd.AddCommand(browserPDFCmd)
}

func runBrowserInstall(cmd *cobra.Command, args []string) error {
	fmt.Println("Installing browser runtime...")

	client, ctx, cancel, err := globalBrowserClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	resp, err := client.Call(ctx, "browser.install", nil)
	if err != nil {
		return fmt.Errorf("browser.install: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	fmt.Println(result["message"])

	return nil
}

func runBrowserStatus(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	resp, err := client.Call(ctx, "browser.status", nil)
	if err != nil {
		return fmt.Errorf("browser.status: %w", err)
	}

	var result struct {
		Installed    bool            `json:"installed"`
		RuntimeDir   string          `json:"runtime_dir"`
		BinaryPath   string          `json:"binary_path"`
		Version      string          `json:"version,omitempty"`
		VersionError string          `json:"version_error,omitempty"`
		Config       *browser.Config `json:"config,omitempty"`
		ConfigError  string          `json:"config_error,omitempty"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	fmt.Printf("Runtime directory: %s\n", result.RuntimeDir)
	fmt.Printf("Installed: %v\n", result.Installed)

	if result.Installed {
		if result.VersionError != "" {
			fmt.Printf("Version: error - %s\n", result.VersionError)
		} else {
			fmt.Printf("Version: %s\n", result.Version)
		}
		fmt.Printf("Binary: %s\n", result.BinaryPath)
	}

	if result.ConfigError != "" {
		fmt.Printf("Config: error - %s\n", result.ConfigError)
	} else if result.Config != nil {
		fmt.Printf("\nConfiguration:\n")
		fmt.Printf("  Headless: %v\n", result.Config.Headless)
		fmt.Printf("  Browser: %s\n", result.Config.Browser)
		fmt.Printf("  Profile: %s\n", result.Config.Profile)
		fmt.Printf("  Timeout: %ds\n", result.Config.Timeout)
	}

	return nil
}

func runBrowserExec(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), browserTimeout)
	defer cancel()

	opts := &browser.ExecOptions{}

	cwd, err := os.Getwd()
	if err == nil {
		opts.WorktreePath = cwd
	}

	if browserSession != "" {
		opts.SessionName = browserSession
	}

	// Interactive execution requires terminal attachment — stays as a direct call.
	return browser.ExecInteractive(ctx, opts, args...)
}

func runBrowserConfig(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	resp, err := client.Call(ctx, "browser.config.get", nil)
	if err != nil {
		return fmt.Errorf("browser.config.get: %w", err)
	}

	var cfg browser.Config
	if err := json.Unmarshal(resp.Result, &cfg); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	fmt.Printf("headless = %v\n", cfg.Headless)
	fmt.Printf("browser = %s\n", cfg.Browser)
	fmt.Printf("profile = %s\n", cfg.Profile)
	fmt.Printf("timeout = %d\n", cfg.Timeout)

	return nil
}

func runBrowserConfigSet(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	if len(args) != 2 {
		return fmt.Errorf("expected key and value, got %d argument(s)", len(args))
	}

	key, value := args[0], args[1]

	params := map[string]string{
		"key":   key,
		"value": value,
	}

	resp, err := client.Call(ctx, "browser.config.set", params)
	if err != nil {
		return fmt.Errorf("browser.config.set: %w", err)
	}

	var cfg browser.Config
	if err := json.Unmarshal(resp.Result, &cfg); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	fmt.Printf("%s = %s\n", key, value)

	return nil
}

// globalBrowserClient connects to the global socket with a longer timeout for install operations.
func globalBrowserClient() (*socket.Client, context.Context, context.CancelFunc, error) {
	gPath := socket.GlobalSocketPath()
	if !socket.SocketExists(gPath) {
		return nil, nil, nil, errors.New(meta.Name + " server not running\nRun '" + meta.Name + " serve' first")
	}
	client, err := socket.NewClient(gPath, socket.WithTimeout(10*time.Minute))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("connect to server: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	return client, ctx, cancel, nil
}

// Agent-friendly browser command implementations

func runBrowserNavigate(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]string{"url": args[0]}
	resp, err := client.Call(ctx, "browser.navigate", params)
	if err != nil {
		return fmt.Errorf("browser.navigate: %w", err)
	}

	// Output JSON for agent parsing
	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserSnapshot(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	resp, err := client.Call(ctx, "browser.snapshot", nil)
	if err != nil {
		return fmt.Errorf("browser.snapshot: %w", err)
	}

	var result struct {
		Snapshot string `json:"snapshot"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	// Output plain text accessibility tree for agent parsing
	fmt.Println(result.Snapshot)

	return nil
}

func runBrowserScreenshot(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]any{}
	if screenshotOutput != "" {
		params["path"] = screenshotOutput
	}
	if screenshotFullPage {
		params["full_page"] = true
	}

	// If running inside a project, pass worktree_id so the socket stores the screenshot directly.
	cwd, _ := os.Getwd()
	wtPath := socket.WorktreeSocketPath(cwd)
	if socket.SocketExists(wtPath) {
		params["worktree_id"] = socket.WorktreeIDFromPath(cwd)
	}

	resp, err := client.Call(ctx, "browser.screenshot", params)
	if err != nil {
		return fmt.Errorf("browser.screenshot: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserClick(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]string{"selector": args[0]}
	resp, err := client.Call(ctx, "browser.click", params)
	if err != nil {
		return fmt.Errorf("browser.click: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserType(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]string{
		"selector": args[0],
		"text":     args[1],
	}
	resp, err := client.Call(ctx, "browser.type", params)
	if err != nil {
		return fmt.Errorf("browser.type: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserWait(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]any{
		"selector":   args[0],
		"timeout_ms": browserWaitTimeout,
	}
	resp, err := client.Call(ctx, "browser.wait", params)
	if err != nil {
		return fmt.Errorf("browser.wait: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserEval(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]string{"js": args[0]}
	resp, err := client.Call(ctx, "browser.eval", params)
	if err != nil {
		return fmt.Errorf("browser.eval: %w", err)
	}

	var result struct {
		Result string `json:"result"`
		Error  string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	if result.Error != "" {
		return fmt.Errorf("eval error: %s", result.Error)
	}
	fmt.Println(result.Result)

	return nil
}

func runBrowserConsole(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	resp, err := client.Call(ctx, "browser.console", nil)
	if err != nil {
		return fmt.Errorf("browser.console: %w", err)
	}

	var result struct {
		Messages []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	// Output as readable text
	for _, msg := range result.Messages {
		fmt.Printf("[%s] %s\n", msg.Type, msg.Text)
	}

	return nil
}

func runBrowserNetwork(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	resp, err := client.Call(ctx, "browser.network", nil)
	if err != nil {
		return fmt.Errorf("browser.network: %w", err)
	}

	// Output JSON for agent parsing
	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserFill(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]string{
		"selector": args[0],
		"value":    args[1],
	}
	resp, err := client.Call(ctx, "browser.fill", params)
	if err != nil {
		return fmt.Errorf("browser.fill: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserSelect(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]any{
		"selector": args[0],
		"values":   args[1:],
	}
	resp, err := client.Call(ctx, "browser.select", params)
	if err != nil {
		return fmt.Errorf("browser.select: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserHover(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]string{"selector": args[0]}
	resp, err := client.Call(ctx, "browser.hover", params)
	if err != nil {
		return fmt.Errorf("browser.hover: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserFocus(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]string{"selector": args[0]}
	resp, err := client.Call(ctx, "browser.focus", params)
	if err != nil {
		return fmt.Errorf("browser.focus: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserScroll(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]any{
		"direction": args[0],
	}
	if scrollAmount > 0 {
		params["amount"] = scrollAmount
	}
	if scrollSelector != "" {
		params["selector"] = scrollSelector
	}

	resp, err := client.Call(ctx, "browser.scroll", params)
	if err != nil {
		return fmt.Errorf("browser.scroll: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserPress(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]any{
		"key": args[0],
	}
	if browserPressSelector != "" {
		params["selector"] = browserPressSelector
	}

	resp, err := client.Call(ctx, "browser.press", params)
	if err != nil {
		return fmt.Errorf("browser.press: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserBack(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	resp, err := client.Call(ctx, "browser.back", nil)
	if err != nil {
		return fmt.Errorf("browser.back: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserForward(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	resp, err := client.Call(ctx, "browser.forward", nil)
	if err != nil {
		return fmt.Errorf("browser.forward: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserReload(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	resp, err := client.Call(ctx, "browser.reload", nil)
	if err != nil {
		return fmt.Errorf("browser.reload: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserDialog(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]any{
		"action": args[0],
	}
	if browserDialogText != "" {
		params["text"] = browserDialogText
	}

	resp, err := client.Call(ctx, "browser.dialog", params)
	if err != nil {
		return fmt.Errorf("browser.dialog: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserUpload(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]any{
		"selector": args[0],
		"files":    args[1:],
	}

	resp, err := client.Call(ctx, "browser.upload", params)
	if err != nil {
		return fmt.Errorf("browser.upload: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}

func runBrowserPDF(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := globalSocketClient()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer cancel()

	params := map[string]any{}
	if pdfOutput != "" {
		params["path"] = pdfOutput
	}
	if pdfFormat != "" {
		params["format"] = pdfFormat
	}
	if pdfLandscape {
		params["landscape"] = true
	}

	resp, err := client.Call(ctx, "browser.pdf", params)
	if err != nil {
		return fmt.Errorf("browser.pdf: %w", err)
	}

	fmt.Println(string(resp.Result))

	return nil
}
