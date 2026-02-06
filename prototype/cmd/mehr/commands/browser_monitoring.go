package commands

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/browser"
)

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
