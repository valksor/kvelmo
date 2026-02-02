package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/browser"
)

// --- View Source commands ---

var sourceOutput string

var browserSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Get page HTML source",
	Long: `Retrieve the full HTML source of the current page.

Examples:
  mehr browser source
  mehr browser source --output page.html`,
	RunE: runBrowserSource,
}

func init() {
	browserSourceCmd.Flags().StringVarP(&sourceOutput, "output", "o", "", "Write source to file instead of stdout")
}

var scriptsURLFilter string

var browserScriptsCmd = &cobra.Command{
	Use:   "scripts",
	Short: "List loaded JavaScript sources",
	Long: `List all JavaScript sources loaded in the current page.

Shows script URLs and sizes. Use --url to filter by URL pattern.

Examples:
  mehr browser scripts
  mehr browser scripts --url "main"`,
	RunE: runBrowserScripts,
}

func init() {
	browserScriptsCmd.Flags().StringVar(&scriptsURLFilter, "url", "", "Filter scripts by URL pattern")
}

func runBrowserSource(cmd *cobra.Command, _ []string) error {
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
		return errors.New("no tabs open")
	}

	source, err := ctrl.GetPageSource(ctx, tabs[0].ID)
	if err != nil {
		return fmt.Errorf("get page source: %w", err)
	}

	if sourceOutput != "" {
		if err := os.WriteFile(sourceOutput, []byte(source), 0o644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
		fmt.Printf("Page source saved to: %s (%d bytes)\n", sourceOutput, len(source))
	} else {
		fmt.Println(source)
	}

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

func runBrowserScripts(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	ctrl, cleanup, err := setupBrowserController(ctx, 15*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}
	if len(tabs) == 0 {
		return errors.New("no tabs open")
	}

	scripts, err := ctrl.GetScriptSources(ctx, tabs[0].ID)
	if err != nil {
		return fmt.Errorf("get scripts: %w", err)
	}

	if len(scripts) == 0 {
		fmt.Println("No scripts found")

		return nil
	}

	for i, script := range scripts {
		if scriptsURLFilter != "" && !containsIgnoreCase(script.URL, scriptsURLFilter) {
			continue
		}
		fmt.Printf("%d. %s (%d bytes)\n", i+1, script.URL, script.Length)
	}

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

// --- WebSocket command ---

var (
	wsDuration float64
	wsURL      string
)

var browserWebSocketCmd = &cobra.Command{
	Use:   "websocket",
	Short: "Monitor WebSocket connections",
	Long: `Monitor WebSocket connections and frames for a duration.

Captures sent and received WebSocket messages.

Examples:
  mehr browser websocket --duration 5
  mehr browser websocket --url "socket.io"`,
	RunE: runBrowserWebSocket,
}

func init() {
	browserWebSocketCmd.Flags().Float64VarP(&wsDuration, "duration", "d", 5.0, "Capture duration in seconds")
	browserWebSocketCmd.Flags().StringVar(&wsURL, "url", "", "Filter by WebSocket URL pattern")
}

func runBrowserWebSocket(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	timeout := time.Duration(wsDuration*float64(time.Second)) + 5*time.Second
	ctrl, cleanup, err := setupBrowserController(ctx, timeout)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}
	if len(tabs) == 0 {
		return errors.New("no tabs open")
	}

	duration := time.Duration(wsDuration * float64(time.Second))
	frames, err := ctrl.GetWebSocketFrames(ctx, tabs[0].ID, duration)
	if err != nil {
		return fmt.Errorf("get websocket frames: %w", err)
	}

	if len(frames) == 0 {
		fmt.Println("No WebSocket frames captured during the monitoring period")

		return nil
	}

	for _, frame := range frames {
		arrow := "→ sent"
		switch frame.Direction {
		case "received":
			arrow = "← recv"
		case "error":
			arrow = "✗ error"
		}

		data := frame.Data
		if frame.Error != "" {
			data = frame.Error
		}
		if len(data) > 200 {
			data = data[:200] + "..."
		}

		fmt.Printf("[%s] [%s] %s\n", frame.Timestamp.Format("15:04:05.000"), arrow, data)
	}

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

// --- Coverage command ---

var (
	coverageDuration float64
	coverageJS       bool
	coverageCSS      bool
	coverageDetail   bool
)

var browserCoverageCmd = &cobra.Command{
	Use:   "coverage",
	Short: "Track CSS/JS code coverage",
	Long: `Measure JavaScript and CSS code coverage for the current page.

Starts coverage tracking, waits for the specified duration, then reports
which code was actually used vs unused.

Examples:
  mehr browser coverage --duration 5
  mehr browser coverage --detail
  mehr browser coverage --css --no-js`,
	RunE: runBrowserCoverage,
}

func init() {
	browserCoverageCmd.Flags().Float64VarP(&coverageDuration, "duration", "d", 5.0, "Tracking duration in seconds")
	browserCoverageCmd.Flags().BoolVar(&coverageJS, "js", true, "Track JavaScript coverage")
	browserCoverageCmd.Flags().BoolVar(&coverageCSS, "css", true, "Track CSS coverage")
	browserCoverageCmd.Flags().BoolVar(&coverageDetail, "detail", false, "Show per-file breakdown")
}

func runBrowserCoverage(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	timeout := time.Duration(coverageDuration*float64(time.Second)) + 10*time.Second
	ctrl, cleanup, err := setupBrowserController(ctx, timeout)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cleanup()

	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}
	if len(tabs) == 0 {
		return errors.New("no tabs open")
	}

	duration := time.Duration(coverageDuration * float64(time.Second))
	summary, jsEntries, cssEntries, err := ctrl.GetCoverage(ctx, tabs[0].ID, duration, coverageJS, coverageCSS)
	if err != nil {
		return fmt.Errorf("get coverage: %w", err)
	}

	fmt.Println("Coverage Summary:")
	if coverageJS {
		fmt.Printf("  JS:    %.1f%% used (%s / %s)\n",
			summary.JSPercentage,
			formatBytes(summary.JSUsedBytes),
			formatBytes(summary.JSTotalBytes))
	}
	if coverageCSS {
		fmt.Printf("  CSS:   %.1f%% used (%s / %s)\n",
			summary.CSSPercentage,
			formatBytes(summary.CSSUsedBytes),
			formatBytes(summary.CSSTotalBytes))
	}
	fmt.Printf("  Total: %.1f%% used (%s / %s)\n",
		summary.TotalPercentage,
		formatBytes(summary.TotalUsed),
		formatBytes(summary.TotalSize))

	if coverageDetail {
		if len(jsEntries) > 0 {
			fmt.Println("\nJS Coverage:")
			for _, entry := range jsEntries {
				pct := float64(0)
				if entry.TotalBytes > 0 {
					pct = float64(entry.UsedBytes) / float64(entry.TotalBytes) * 100
				}
				fmt.Printf("  %-60s %.1f%% (%s / %s)\n",
					truncateString(entry.URL, 60), pct,
					formatBytes(entry.UsedBytes), formatBytes(entry.TotalBytes))
			}
		}
		if len(cssEntries) > 0 {
			fmt.Println("\nCSS Coverage:")
			for _, entry := range cssEntries {
				pct := float64(0)
				if entry.TotalBytes > 0 {
					pct = float64(entry.UsedBytes) / float64(entry.TotalBytes) * 100
				}
				url := entry.StyleSheetID
				if entry.URL != "" {
					url = entry.URL
				}
				fmt.Printf("  %-60s %.1f%% (%s / %s)\n",
					truncateString(url, 60), pct,
					formatBytes(entry.UsedBytes), formatBytes(entry.TotalBytes))
			}
		}
	}

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

// --- Styles command ---

var (
	stylesSelector  string
	stylesComputed  bool
	stylesMatched   bool
	stylesInherited bool
	stylesFilter    string
)

var browserStylesCmd = &cobra.Command{
	Use:   "styles --selector <css>",
	Short: "Inspect CSS styles on an element",
	Long: `Inspect computed CSS styles, matched rules, and inherited styles for a DOM element.

Examples:
  mehr browser styles --selector "h1" --computed
  mehr browser styles --selector ".my-class" --matched
  mehr browser styles --selector "#main" --computed --filter "font"`,
	RunE: runBrowserStyles,
}

func init() {
	browserStylesCmd.Flags().StringVar(&stylesSelector, "selector", "", "CSS selector for target element")
	browserStylesCmd.Flags().BoolVar(&stylesComputed, "computed", true, "Show computed styles")
	browserStylesCmd.Flags().BoolVar(&stylesMatched, "matched", false, "Show matched CSS rules")
	browserStylesCmd.Flags().BoolVar(&stylesInherited, "inherited", false, "Show inherited styles")
	browserStylesCmd.Flags().StringVar(&stylesFilter, "filter", "", "Filter properties by name pattern")
	_ = browserStylesCmd.MarkFlagRequired("selector")
}

func runBrowserStyles(cmd *cobra.Command, _ []string) error {
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
		return errors.New("no tabs open")
	}

	// Show computed styles
	if stylesComputed {
		styles, err := ctrl.GetComputedStyles(ctx, tabs[0].ID, stylesSelector)
		if err != nil {
			return fmt.Errorf("get computed styles: %w", err)
		}

		fmt.Printf("Computed styles for %q:\n", stylesSelector)
		count := 0
		for _, s := range styles {
			if stylesFilter != "" && !containsIgnoreCase(s.Name, stylesFilter) {
				continue
			}
			// Skip empty values for cleaner output
			if s.Value == "" {
				continue
			}
			fmt.Printf("  %s: %s\n", s.Name, s.Value)
			count++
		}
		fmt.Printf("  (%d properties shown)\n", count)
	}

	// Show matched rules
	if stylesMatched || stylesInherited {
		matched, err := ctrl.GetMatchedStyles(ctx, tabs[0].ID, stylesSelector)
		if err != nil {
			return fmt.Errorf("get matched styles: %w", err)
		}

		if stylesMatched {
			// Inline styles
			if len(matched.InlineStyles) > 0 {
				fmt.Printf("\nInline styles:\n")
				printCSSProperties(matched.InlineStyles, stylesFilter)
			}

			// Matched rules
			if len(matched.MatchedRules) > 0 {
				fmt.Printf("\nMatched CSS rules for %q:\n", stylesSelector)
				for i, rule := range matched.MatchedRules {
					source := rule.SourceURL
					if source == "" {
						source = rule.Origin
					}
					fmt.Printf("  %d. %s (%s)\n", i+1, rule.Selector, source)
					printCSSProperties(rule.Properties, stylesFilter)
				}
			}

			// Pseudo-element styles
			for _, pe := range matched.PseudoElements {
				fmt.Printf("\nPseudo-element ::%s:\n", pe.PseudoType)
				for _, rule := range pe.MatchedRules {
					fmt.Printf("  %s\n", rule.Selector)
					printCSSProperties(rule.Properties, stylesFilter)
				}
			}
		}

		if stylesInherited && len(matched.InheritedStyles) > 0 {
			fmt.Printf("\nInherited styles:\n")
			for i, inherited := range matched.InheritedStyles {
				if len(inherited.InlineStyles) == 0 && len(inherited.MatchedRules) == 0 {
					continue
				}
				fmt.Printf("  Ancestor %d:\n", i+1)
				if len(inherited.InlineStyles) > 0 {
					fmt.Printf("    Inline:\n")
					for _, prop := range inherited.InlineStyles {
						if stylesFilter != "" && !containsIgnoreCase(prop.Name, stylesFilter) {
							continue
						}
						fmt.Printf("      %s: %s\n", prop.Name, prop.Value)
					}
				}
				for _, rule := range inherited.MatchedRules {
					fmt.Printf("    %s:\n", rule.Selector)
					printCSSProperties(rule.Properties, stylesFilter)
				}
			}
		}
	}

	if browserKeepAlive {
		printKeepAliveMessage(ctrl)
	}

	return nil
}

// printCSSProperties prints CSS properties with optional filtering.
func printCSSProperties(props []browser.CSSProperty, filter string) {
	for _, prop := range props {
		if filter != "" && !containsIgnoreCase(prop.Name, filter) {
			continue
		}
		important := ""
		if prop.Important {
			important = " !important"
		}
		fmt.Printf("     %s: %s%s\n", prop.Name, prop.Value, important)
	}
}

// formatBytes formats byte count as human-readable string.
func formatBytes(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}

	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}

// containsIgnoreCase checks if a string contains another (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
