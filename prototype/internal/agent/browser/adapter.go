// Package browser provides agent tool adapters for browser automation.
// It bridges the browser controller with the agent system, allowing
// AI agents to control Chrome for tasks like navigation, screenshots,
// DOM interaction, and monitoring.
package browser

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/valksor/go-mehrhof/internal/browser"
	"github.com/valksor/go-mehrhof/internal/events"
)

// Tool represents a browser operation available to agents.
type Tool struct {
	Name        string
	Description string
	Execute     func(ctx context.Context, args map[string]any) (string, error)
}

// Adapter provides browser tools to agents.
type Adapter struct {
	controller browser.Controller
	eventBus   *events.Bus
}

// NewAdapter creates a new browser adapter.
func NewAdapter(controller browser.Controller, eventBus *events.Bus) (*Adapter, error) {
	if controller == nil {
		return nil, errors.New("controller is required for browser adapter")
	}
	if eventBus == nil {
		slog.Warn("browser adapter created without event bus - browser events will not be published")
	}

	return &Adapter{
		controller: controller,
		eventBus:   eventBus,
	}, nil
}

// Tools returns all available browser tools.
func (a *Adapter) Tools() []Tool {
	return []Tool{
		a.openURLTool(),
		a.screenshotTool(),
		a.clickTool(),
		a.typeTool(),
		a.evaluateTool(),
		a.queryTool(),
		a.getConsoleLogsTool(),
		a.getNetworkRequestsTool(),
		a.detectAuthTool(),
		a.waitForLoginTool(),
	}
}

// openURLTool opens a URL in the browser.
func (a *Adapter) openURLTool() Tool {
	return Tool{
		Name:        "browser_open_url",
		Description: "Opens a URL in a new browser tab. Waits for the page to load.",
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			url, ok := args["url"].(string)
			if !ok {
				return "", errors.New("url parameter is required")
			}

			// Ensure browser is connected
			if err := a.controller.Connect(ctx); err != nil {
				return "", fmt.Errorf("connect to browser: %w", err)
			}

			// Open tab
			tab, err := a.controller.OpenTab(ctx, url)
			if err != nil {
				return "", fmt.Errorf("open tab: %w", err)
			}

			// Publish event
			a.publishEvent(events.BrowserTabOpenedEvent{
				TabID: tab.ID,
				URL:   url,
				Title: tab.Title,
			})

			return fmt.Sprintf("Opened %s in tab %s", url, tab.ID), nil
		},
	}
}

// screenshotTool takes a screenshot of the current page.
func (a *Adapter) screenshotTool() Tool {
	return Tool{
		Name:        "browser_screenshot",
		Description: "Takes a screenshot of the current page. Returns the path to the saved image.",
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			tabID, ok := args["tab_id"].(string)
			if !ok {
				return "", errors.New("tab_id parameter is required")
			}

			fullPage := true
			if fp, ok := args["full_page"].(bool); ok {
				fullPage = fp
			}

			format := "png"
			if f, ok := args["format"].(string); ok {
				format = f
			}

			quality := 80
			if q, ok := args["quality"].(int); ok {
				quality = q
			}

			opts := browser.ScreenshotOptions{
				Format:   format,
				Quality:  quality,
				FullPage: fullPage,
			}

			data, err := a.controller.Screenshot(ctx, tabID, opts)
			if err != nil {
				return "", fmt.Errorf("take screenshot: %w", err)
			}

			// Save screenshot to file so agent can use it
			screenshotDir := ".mehrhof/screenshots"
			if err := os.MkdirAll(screenshotDir, 0o755); err != nil {
				return "", fmt.Errorf("create screenshot directory: %w", err)
			}

			filename := fmt.Sprintf("screenshot_%d.%s", time.Now().UnixNano(), format)
			fullPath := filepath.Join(screenshotDir, filename)

			if err := os.WriteFile(fullPath, data, 0o644); err != nil {
				return "", fmt.Errorf("save screenshot: %w", err)
			}

			// Publish event
			a.publishEvent(events.BrowserScreenshotEvent{
				TabID:    tabID,
				Format:   format,
				FullPath: fullPath,
			})

			return fmt.Sprintf("Screenshot saved: %s (%d bytes)", fullPath, len(data)), nil
		},
	}
}

// clickTool clicks an element on the page.
func (a *Adapter) clickTool() Tool {
	return Tool{
		Name:        "browser_click",
		Description: "Clicks an element matching a CSS selector. Example selector: 'button.submit', '#login-btn', '.cta-button'.",
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			tabID, ok := args["tab_id"].(string)
			if !ok {
				return "", errors.New("tab_id parameter is required")
			}

			selector, ok := args["selector"].(string)
			if !ok {
				return "", errors.New("selector parameter is required")
			}

			if err := a.controller.Click(ctx, tabID, selector); err != nil {
				return "", fmt.Errorf("click element: %w", err)
			}

			// Publish event
			a.publishEvent(events.BrowserActionEvent{
				Action:   "click",
				Selector: selector,
				Success:  true,
			})

			return "Clicked element: " + selector, nil
		},
	}
}

// typeTool types text into an element.
func (a *Adapter) typeTool() Tool {
	return Tool{
		Name:        "browser_type",
		Description: "Types text into an input element matching a CSS selector. Set clear=true to clear existing text first.",
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			tabID, ok := args["tab_id"].(string)
			if !ok {
				return "", errors.New("tab_id parameter is required")
			}

			selector, ok := args["selector"].(string)
			if !ok {
				return "", errors.New("selector parameter is required")
			}

			text, ok := args["text"].(string)
			if !ok {
				return "", errors.New("text parameter is required")
			}

			clearField := false
			if c, ok := args["clear"].(bool); ok {
				clearField = c
			}

			if err := a.controller.Type(ctx, tabID, selector, text, clearField); err != nil {
				return "", fmt.Errorf("type text: %w", err)
			}

			// Publish event
			a.publishEvent(events.BrowserActionEvent{
				Action:   "type",
				Selector: selector,
				Success:  true,
			})

			return "Typed text into " + selector, nil
		},
	}
}

// evaluateTool executes JavaScript in the page.
func (a *Adapter) evaluateTool() Tool {
	return Tool{
		Name:        "browser_evaluate",
		Description: "Evaluates JavaScript code in the page context and returns the result.",
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			tabID, ok := args["tab_id"].(string)
			if !ok {
				return "", errors.New("tab_id parameter is required")
			}

			expression, ok := args["expression"].(string)
			if !ok {
				return "", errors.New("expression parameter is required")
			}

			result, err := a.controller.Eval(ctx, tabID, expression)
			if err != nil {
				return "", fmt.Errorf("evaluate javascript: %w", err)
			}

			// Publish event
			a.publishEvent(events.BrowserActionEvent{
				Action:  "evaluate",
				Success: true,
			})

			return fmt.Sprintf("Result: %v", result), nil
		},
	}
}

// queryTool queries the DOM for elements.
func (a *Adapter) queryTool() Tool {
	return Tool{
		Name:        "browser_query",
		Description: "Queries the DOM for elements matching a CSS selector. Returns element details including text content and visibility.",
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			tabID, ok := args["tab_id"].(string)
			if !ok {
				return "", errors.New("tab_id parameter is required")
			}

			selector, ok := args["selector"].(string)
			if !ok {
				return "", errors.New("selector parameter is required")
			}

			all := false
			if a, ok := args["all"].(bool); ok {
				all = a
			}

			var result string
			if all {
				elements, err := a.controller.QuerySelectorAll(ctx, tabID, selector)
				if err != nil {
					return "", fmt.Errorf("query elements: %w", err)
				}
				result = fmt.Sprintf("Found %d elements matching %s", len(elements), selector)
			} else {
				element, err := a.controller.QuerySelector(ctx, tabID, selector)
				if err != nil {
					return "", fmt.Errorf("query element: %w", err)
				}
				result = fmt.Sprintf("Element: %s, Text: %s, Visible: %v",
					element.TagName, element.TextContent, element.Visible)
			}

			return result, nil
		},
	}
}

// getConsoleLogsTool retrieves console logs.
func (a *Adapter) getConsoleLogsTool() Tool {
	return Tool{
		Name:        "browser_get_console_logs",
		Description: "Retrieves console logs from the browser tab. Specify duration in seconds (default: 5).",
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			tabID, ok := args["tab_id"].(string)
			if !ok {
				return "", errors.New("tab_id parameter is required")
			}

			duration := 5
			if d, ok := args["duration_seconds"].(int); ok {
				duration = d
			}

			logs, err := a.controller.GetConsoleLogs(ctx, tabID, time.Duration(duration)*time.Second)
			if err != nil {
				return "", fmt.Errorf("get console logs: %w", err)
			}

			return fmt.Sprintf("Retrieved %d console logs", len(logs)), nil
		},
	}
}

// getNetworkRequestsTool retrieves network requests.
func (a *Adapter) getNetworkRequestsTool() Tool {
	return Tool{
		Name:        "browser_get_network_requests",
		Description: "Retrieves network requests made by the browser tab. Specify duration in seconds (default: 5).",
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			tabID, ok := args["tab_id"].(string)
			if !ok {
				return "", errors.New("tab_id parameter is required")
			}

			duration := 5
			if d, ok := args["duration_seconds"].(int); ok {
				duration = d
			}

			requests, err := a.controller.GetNetworkRequests(ctx, tabID, time.Duration(duration)*time.Second)
			if err != nil {
				return "", fmt.Errorf("get network requests: %w", err)
			}

			return fmt.Sprintf("Retrieved %d network requests", len(requests)), nil
		},
	}
}

// detectAuthTool detects if the current page requires authentication.
func (a *Adapter) detectAuthTool() Tool {
	return Tool{
		Name:        "browser_detect_auth",
		Description: "Detects if the current page requires authentication (login form, auth wall, etc.). Returns auth requirement details or null if no auth needed.",
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			tabID, ok := args["tab_id"].(string)
			if !ok {
				return "", errors.New("tab_id parameter is required")
			}

			auth, err := a.controller.DetectAuth(ctx, tabID)
			if err != nil {
				return "", fmt.Errorf("detect auth: %w", err)
			}

			if auth == nil {
				return "No authentication required", nil
			}

			return fmt.Sprintf("Authentication required: %s (type: %s)", auth.Hint, auth.Type), nil
		},
	}
}

// waitForLoginTool waits for the user to manually login in the browser.
func (a *Adapter) waitForLoginTool() Tool {
	return Tool{
		Name:        "browser_wait_for_login",
		Description: "Pauses the workflow and prompts the user to login manually in the browser window. Waits for user to press Enter after login is complete.",
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			tabID, ok := args["tab_id"].(string)
			if !ok {
				return "", errors.New("tab_id parameter is required")
			}

			// First detect auth requirements
			auth, err := a.controller.DetectAuth(ctx, tabID)
			if err != nil {
				return "", fmt.Errorf("detect auth: %w", err)
			}

			if auth == nil {
				return "No authentication required, continuing workflow", nil
			}

			// Wait for user to login
			if err := a.controller.WaitForLogin(ctx, tabID, auth); err != nil {
				return "", fmt.Errorf("wait for login: %w", err)
			}

			return "User completed login, continuing workflow", nil
		},
	}
}

// publishEvent publishes a browser event to the event bus.
func (a *Adapter) publishEvent(event events.Eventer) {
	if a.eventBus == nil {
		slog.Warn("browser adapter: no event bus configured, event dropped",
			"event_type", event.ToEvent().Type)

		return
	}
	a.eventBus.Publish(event)
}

// GetController returns the underlying browser controller.
func (a *Adapter) GetController() browser.Controller {
	return a.controller
}
