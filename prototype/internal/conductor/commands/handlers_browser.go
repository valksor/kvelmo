package commands

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/valksor/go-mehrhof/internal/browser"
	"github.com/valksor/go-mehrhof/internal/conductor"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:        "browser",
			Description: "Browser automation commands",
			Category:    "tools",
			Subcommands: []string{
				"status", "tabs", "goto", "navigate", "screenshot",
				"click", "type", "eval", "dom", "reload", "close", "switch",
				"cookies-get", "cookies-set", "network", "console",
				"websocket", "source", "scripts", "styles", "coverage",
			},
		},
		Handler: handleBrowser,
	})
}

func handleBrowser(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	sub := GetString(inv.Options, "subcommand")
	if sub == "" && len(inv.Args) > 0 {
		sub = inv.Args[0]
	}

	switch sub {
	case "status":
		return handleBrowserStatus(ctx, cond, inv)
	case "tabs":
		return handleBrowserTabs(ctx, cond, inv)
	case "goto":
		return handleBrowserGoto(ctx, cond, inv)
	case "navigate":
		return handleBrowserNavigate(ctx, cond, inv)
	case "screenshot":
		return handleBrowserScreenshot(ctx, cond, inv)
	case "click":
		return handleBrowserClick(ctx, cond, inv)
	case "type":
		return handleBrowserType(ctx, cond, inv)
	case "eval":
		return handleBrowserEval(ctx, cond, inv)
	case "dom":
		return handleBrowserDOM(ctx, cond, inv)
	case "reload":
		return handleBrowserReload(ctx, cond, inv)
	case "close":
		return handleBrowserClose(ctx, cond, inv)
	case "switch":
		return handleBrowserSwitch(ctx, cond, inv)
	case "cookies-get":
		return handleBrowserCookiesGet(ctx, cond, inv)
	case "cookies-set":
		return handleBrowserCookiesSet(ctx, cond, inv)
	case "network":
		return handleBrowserNetwork(ctx, cond, inv)
	case "console":
		return handleBrowserConsole(ctx, cond, inv)
	case "websocket":
		return handleBrowserWebSocket(ctx, cond, inv)
	case "source":
		return handleBrowserSource(ctx, cond, inv)
	case "scripts":
		return handleBrowserScripts(ctx, cond, inv)
	case "styles":
		return handleBrowserStyles(ctx, cond, inv)
	case "coverage":
		return handleBrowserCoverage(ctx, cond, inv)
	default:
		return nil, errors.New("unknown browser subcommand: " + sub)
	}
}

// getBrowserCtrl extracts the browser controller from the conductor.
func getBrowserCtrl(ctx context.Context, cond *conductor.Conductor) (browser.Controller, error) {
	if cond == nil {
		return nil, errors.New("conductor not initialized")
	}

	ctrl := cond.GetBrowser(ctx)
	if ctrl == nil {
		return nil, errors.New("browser not configured")
	}

	return ctrl, nil
}

// resolveTabID returns the requested tab ID, or defaults to the first open tab.
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

func handleBrowserStatus(ctx context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	if cond == nil {
		return NewResult("Browser not connected").WithData(map[string]any{
			"connected": false,
			"error":     "conductor not initialized",
		}), nil
	}

	ctrl := cond.GetBrowser(ctx)
	if ctrl == nil {
		return NewResult("Browser not connected").WithData(map[string]any{
			"connected": false,
			"error":     "browser not configured",
		}), nil
	}

	tabs, listErr := ctrl.ListTabs(ctx)
	if listErr != nil {
		return NewResult("Browser not connected").WithData(map[string]any{ //nolint:nilerr // Status returns degraded result, not an error
			"connected": false,
			"error":     listErr.Error(),
		}), nil
	}

	tabData := make([]map[string]any, 0, len(tabs))
	for _, tab := range tabs {
		tabData = append(tabData, map[string]any{
			"id":    tab.ID,
			"title": tab.Title,
			"url":   tab.URL,
		})
	}

	return NewResult("Browser connected").WithData(map[string]any{
		"connected": true,
		"host":      "localhost",
		"port":      ctrl.GetPort(),
		"tabs":      tabData,
	}), nil
}

func handleBrowserTabs(ctx context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	tabs, err := ctrl.ListTabs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tabs: %w", err)
	}

	tabData := make([]map[string]any, 0, len(tabs))
	for _, tab := range tabs {
		tabData = append(tabData, map[string]any{
			"id":    tab.ID,
			"title": tab.Title,
			"url":   tab.URL,
		})
	}

	return NewResult(fmt.Sprintf("%d tab(s) open", len(tabs))).WithData(map[string]any{
		"tabs":  tabData,
		"count": len(tabData),
	}), nil
}

func handleBrowserGoto(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	url := GetString(inv.Options, "url")
	if url == "" && len(inv.Args) > 1 {
		url = inv.Args[1]
	}
	if url == "" {
		return nil, fmt.Errorf("%w: url is required", ErrBadRequest)
	}

	tab, err := ctrl.OpenTab(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("open tab: %w", err)
	}

	return NewResult("Opened " + url).WithData(map[string]any{
		"tab": map[string]any{
			"id":    tab.ID,
			"title": tab.Title,
			"url":   tab.URL,
		},
	}), nil
}

func handleBrowserNavigate(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	url := GetString(inv.Options, "url")
	if url == "" {
		return nil, fmt.Errorf("%w: url is required", ErrBadRequest)
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	if err := ctrl.Navigate(ctx, tabID, url); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	return NewResult("Navigated to " + url).WithData(map[string]any{
		"tab_id": tabID,
		"url":    url,
	}), nil
}

func handleBrowserScreenshot(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	format := GetString(inv.Options, "format")
	if format == "" {
		format = "png"
	}

	quality := GetInt(inv.Options, "quality")
	if quality == 0 {
		quality = 80
	}

	opts := browser.ScreenshotOptions{
		Format:   format,
		Quality:  quality,
		FullPage: GetBool(inv.Options, "full_page"),
	}

	data, err := ctrl.Screenshot(ctx, tabID, opts)
	if err != nil {
		return nil, fmt.Errorf("screenshot: %w", err)
	}

	return NewResult("Screenshot captured").WithData(map[string]any{
		"format":   format,
		"data":     base64.StdEncoding.EncodeToString(data),
		"size":     len(data),
		"encoding": "base64",
	}), nil
}

func handleBrowserClick(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	selector := GetString(inv.Options, "selector")
	if selector == "" {
		return nil, fmt.Errorf("%w: selector is required", ErrBadRequest)
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	if err := ctrl.Click(ctx, tabID, selector); err != nil {
		return nil, fmt.Errorf("click: %w", err)
	}

	return NewResult("Clicked " + selector).WithData(map[string]any{
		"selector": selector,
	}), nil
}

func handleBrowserType(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	selector := GetString(inv.Options, "selector")
	if selector == "" {
		return nil, fmt.Errorf("%w: selector is required", ErrBadRequest)
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	text := GetString(inv.Options, "text")
	clearField := GetBool(inv.Options, "clear")

	if err := ctrl.Type(ctx, tabID, selector, text, clearField); err != nil {
		return nil, fmt.Errorf("type: %w", err)
	}

	return NewResult("Typed into " + selector).WithData(map[string]any{
		"selector": selector,
	}), nil
}

func handleBrowserEval(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	expression := GetString(inv.Options, "expression")
	if expression == "" {
		return nil, fmt.Errorf("%w: expression is required", ErrBadRequest)
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	result, err := ctrl.Eval(ctx, tabID, expression)
	if err != nil {
		return nil, fmt.Errorf("eval: %w", err)
	}

	return NewResult("Expression evaluated").WithData(map[string]any{
		"result": result,
	}), nil
}

func handleBrowserDOM(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	selector := GetString(inv.Options, "selector")
	if selector == "" {
		return nil, fmt.Errorf("%w: selector is required", ErrBadRequest)
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	all := GetBool(inv.Options, "all")
	html := GetBool(inv.Options, "html")

	if all {
		limit := GetInt(inv.Options, "limit")
		if limit == 0 {
			limit = 20
		}

		elems, err := ctrl.QuerySelectorAll(ctx, tabID, selector)
		if err != nil {
			return nil, fmt.Errorf("query: %w", err)
		}

		showing := limit
		if showing > len(elems) {
			showing = len(elems)
		}

		elements := make([]map[string]any, 0, showing)
		for i := range showing {
			elem := map[string]any{
				"tag_name":     elems[i].TagName,
				"text_content": elems[i].TextContent,
				"visible":      elems[i].Visible,
			}
			if html {
				elem["outer_html"] = elems[i].OuterHTML
			}
			elements = append(elements, elem)
		}

		return NewResult(fmt.Sprintf("Found %d element(s)", len(elems))).WithData(map[string]any{
			"elements": elements,
			"count":    len(elems),
			"showing":  len(elements),
		}), nil
	}

	elem, err := ctrl.QuerySelector(ctx, tabID, selector)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	if elem == nil {
		return NewResult("No element found").WithData(map[string]any{
			"element": nil,
		}), nil
	}

	element := map[string]any{
		"tag_name":     elem.TagName,
		"text_content": elem.TextContent,
		"visible":      elem.Visible,
	}
	if html {
		element["outer_html"] = elem.OuterHTML
	}

	return NewResult("Element found").WithData(map[string]any{
		"element": element,
	}), nil
}

func handleBrowserReload(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	hard := GetBool(inv.Options, "hard")

	if err := ctrl.Reload(ctx, tabID, hard); err != nil {
		return nil, fmt.Errorf("reload: %w", err)
	}

	return NewResult("Page reloaded").WithData(map[string]any{
		"tab_id": tabID,
		"hard":   hard,
	}), nil
}

func handleBrowserClose(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	tabID := GetString(inv.Options, "tab_id")
	if tabID == "" {
		return nil, fmt.Errorf("%w: tab_id is required", ErrBadRequest)
	}

	if err := ctrl.CloseTab(ctx, tabID); err != nil {
		return nil, fmt.Errorf("close tab: %w", err)
	}

	return NewResult("Tab closed").WithData(map[string]any{
		"tab_id": tabID,
	}), nil
}

func handleBrowserSwitch(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	tabID := GetString(inv.Options, "tab_id")
	if tabID == "" {
		return nil, fmt.Errorf("%w: tab_id is required", ErrBadRequest)
	}

	tab, err := ctrl.SwitchTab(ctx, tabID)
	if err != nil {
		return nil, fmt.Errorf("switch tab: %w", err)
	}

	return NewResult("Switched to tab").WithData(map[string]any{
		"tab": map[string]any{
			"id":    tab.ID,
			"title": tab.Title,
			"url":   tab.URL,
		},
	}), nil
}
