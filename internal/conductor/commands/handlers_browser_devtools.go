package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/valksor/go-mehrhof/internal/browser"
	"github.com/valksor/go-mehrhof/internal/conductor"
)

// defaultMonitorDuration returns a time.Duration from seconds, defaulting to 5s.
func defaultMonitorDuration(seconds int) time.Duration {
	if seconds <= 0 {
		return 5 * time.Second
	}

	return time.Duration(seconds) * time.Second
}

func handleBrowserCookiesGet(ctx context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	cookies, err := ctrl.GetCookies(ctx)
	if err != nil {
		return nil, fmt.Errorf("get cookies: %w", err)
	}

	return NewResult(fmt.Sprintf("%d cookie(s)", len(cookies))).WithData(map[string]any{
		"cookies": cookies,
		"count":   len(cookies),
	}), nil
}

func handleBrowserCookiesSet(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	// Extract cookies from options - they may arrive as a JSON-decoded slice.
	rawCookies, ok := inv.Options["cookies"]
	if !ok || rawCookies == nil {
		return nil, fmt.Errorf("%w: cookies is required", ErrBadRequest)
	}

	cookies, err := decodeCookies(rawCookies)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cookies: %s", ErrBadRequest, err.Error())
	}

	if err := ctrl.SetCookies(ctx, cookies); err != nil {
		return nil, fmt.Errorf("set cookies: %w", err)
	}

	return NewResult(fmt.Sprintf("%d cookie(s) set", len(cookies))).WithData(map[string]any{
		"count": len(cookies),
	}), nil
}

// decodeCookies converts a raw option value into a typed cookie slice.
func decodeCookies(raw any) ([]browser.Cookie, error) {
	// If already the right type (unlikely but handle it).
	if c, ok := raw.([]browser.Cookie); ok {
		return c, nil
	}

	// Re-marshal then unmarshal to handle map[string]any representations.
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, errors.New("marshal cookies: " + err.Error())
	}

	var cookies []browser.Cookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil, errors.New("unmarshal cookies: " + err.Error())
	}

	return cookies, nil
}

func handleBrowserNetwork(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	captureBody := GetBool(inv.Options, "capture_body")
	if captureBody {
		maxBodySize := GetInt(inv.Options, "max_body_size")
		if maxBodySize <= 0 {
			maxBodySize = 1024 * 1024 // 1MB default
		}
		ctrl.SetNetworkMonitorOptions(browser.NetworkMonitorOptions{
			CaptureBody: true,
			MaxBodySize: maxBodySize,
		})
	}

	duration := defaultMonitorDuration(GetInt(inv.Options, "duration"))

	requests, err := ctrl.GetNetworkRequests(ctx, tabID, duration)
	if err != nil {
		return nil, fmt.Errorf("monitor network: %w", err)
	}

	return NewResult(fmt.Sprintf("%d network request(s)", len(requests))).WithData(map[string]any{
		"requests": requests,
		"count":    len(requests),
	}), nil
}

func handleBrowserConsole(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	duration := defaultMonitorDuration(GetInt(inv.Options, "duration"))

	messages, err := ctrl.GetConsoleLogs(ctx, tabID, duration)
	if err != nil {
		return nil, fmt.Errorf("monitor console: %w", err)
	}

	// Filter by level if specified.
	level := GetString(inv.Options, "level")
	if level != "" {
		filtered := make([]browser.ConsoleMessage, 0, len(messages))
		for _, msg := range messages {
			if msg.Level == level {
				filtered = append(filtered, msg)
			}
		}
		messages = filtered
	}

	return NewResult(fmt.Sprintf("%d console message(s)", len(messages))).WithData(map[string]any{
		"messages": messages,
		"count":    len(messages),
	}), nil
}

func handleBrowserWebSocket(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	duration := defaultMonitorDuration(GetInt(inv.Options, "duration"))

	frames, err := ctrl.GetWebSocketFrames(ctx, tabID, duration)
	if err != nil {
		return nil, fmt.Errorf("monitor websocket: %w", err)
	}

	return NewResult(fmt.Sprintf("%d websocket frame(s)", len(frames))).WithData(map[string]any{
		"frames": frames,
		"count":  len(frames),
	}), nil
}

func handleBrowserSource(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	source, err := ctrl.GetPageSource(ctx, tabID)
	if err != nil {
		return nil, fmt.Errorf("get page source: %w", err)
	}

	return NewResult("Page source retrieved").WithData(map[string]any{
		"source": source,
		"length": len(source),
	}), nil
}

func handleBrowserScripts(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	scripts, err := ctrl.GetScriptSources(ctx, tabID)
	if err != nil {
		return nil, fmt.Errorf("get scripts: %w", err)
	}

	return NewResult(fmt.Sprintf("%d script(s)", len(scripts))).WithData(map[string]any{
		"scripts": scripts,
		"count":   len(scripts),
	}), nil
}

func handleBrowserStyles(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
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

	computed := GetBool(inv.Options, "computed")
	matched := GetBool(inv.Options, "matched")

	// Default to computed styles when neither is specified.
	if !computed && !matched {
		computed = true
	}

	result := map[string]any{
		"selector": selector,
	}

	if computed {
		styles, err := ctrl.GetComputedStyles(ctx, tabID, selector)
		if err != nil {
			return nil, fmt.Errorf("get computed styles: %w", err)
		}
		result["computed"] = styles
	}

	if matched {
		styles, err := ctrl.GetMatchedStyles(ctx, tabID, selector)
		if err != nil {
			return nil, fmt.Errorf("get matched styles: %w", err)
		}
		result["matched"] = styles
	}

	return NewResult("Styles retrieved").WithData(result), nil
}

func handleBrowserCoverage(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ctrl, err := getBrowserCtrl(ctx, cond)
	if err != nil {
		return nil, err
	}

	tabID, err := resolveTabID(ctx, ctrl, GetString(inv.Options, "tab_id"))
	if err != nil {
		return nil, err
	}

	duration := defaultMonitorDuration(GetInt(inv.Options, "duration"))
	trackJS := GetBool(inv.Options, "track_js")
	trackCSS := GetBool(inv.Options, "track_css")

	// Default to tracking both when neither is specified.
	if !trackJS && !trackCSS {
		trackJS = true
		trackCSS = true
	}

	summary, jsEntries, cssEntries, err := ctrl.GetCoverage(ctx, tabID, duration, trackJS, trackCSS)
	if err != nil {
		return nil, fmt.Errorf("get coverage: %w", err)
	}

	return NewResult("Coverage collected").WithData(map[string]any{
		"summary":     summary,
		"js_entries":  jsEntries,
		"css_entries": cssEntries,
	}), nil
}
