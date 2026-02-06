package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/browser"
)

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
