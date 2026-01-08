//go:build !(linux && arm64) && !no_browser

package browser

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/browser"
	"github.com/valksor/go-mehrhof/internal/events"
)

// MockController is a mock implementation of browser.Controller for testing.
type MockController struct {
	connectErr               error
	isConnected              bool
	openTabErr               error
	openTabResult            *browser.Tab
	screenshotErr            error
	screenshotResult         []byte
	clickErr                 error
	typeErr                  error
	evalErr                  error
	evalResult               any
	querySelectorErr         error
	querySelectorResult      *browser.DOMElement
	querySelectorAllErr      error
	querySelectorAllResult   []browser.DOMElement
	getConsoleLogsErr        error
	getConsoleLogsResult     []browser.ConsoleMessage
	getNetworkRequestsErr    error
	getNetworkRequestsResult []browser.NetworkRequest
	detectAuthErr            error
	detectAuthResult         *browser.AuthRequirement
	waitForLoginErr          error
}

func (m *MockController) Connect(ctx context.Context) error {
	if m.connectErr != nil {
		return m.connectErr
	}
	m.isConnected = true

	return nil
}

func (m *MockController) Disconnect() error {
	m.isConnected = false

	return nil
}

func (m *MockController) IsConnected() bool {
	return m.isConnected
}

func (m *MockController) ListTabs(ctx context.Context) ([]browser.Tab, error) {
	return nil, errors.New("not implemented")
}

func (m *MockController) OpenTab(ctx context.Context, url string) (*browser.Tab, error) {
	if m.openTabErr != nil {
		return nil, m.openTabErr
	}

	return m.openTabResult, nil
}

func (m *MockController) CloseTab(ctx context.Context, tabID string) error {
	return nil
}

func (m *MockController) SwitchTab(ctx context.Context, tabID string) (*browser.Tab, error) {
	return nil, errors.New("not implemented")
}

func (m *MockController) Navigate(ctx context.Context, tabID, url string) error {
	return nil
}

func (m *MockController) Reload(ctx context.Context, tabID string, hard bool) error {
	return nil
}

func (m *MockController) Screenshot(ctx context.Context, tabID string, opts browser.ScreenshotOptions) ([]byte, error) {
	if m.screenshotErr != nil {
		return nil, m.screenshotErr
	}

	return m.screenshotResult, nil
}

func (m *MockController) QuerySelector(ctx context.Context, tabID, selector string) (*browser.DOMElement, error) {
	if m.querySelectorErr != nil {
		return nil, m.querySelectorErr
	}

	return m.querySelectorResult, nil
}

func (m *MockController) QuerySelectorAll(ctx context.Context, tabID, selector string) ([]browser.DOMElement, error) {
	if m.querySelectorAllErr != nil {
		return nil, m.querySelectorAllErr
	}

	return m.querySelectorAllResult, nil
}

func (m *MockController) Click(ctx context.Context, tabID, selector string) error {
	if m.clickErr != nil {
		return m.clickErr
	}

	return nil
}

func (m *MockController) Type(ctx context.Context, tabID, selector, text string, clearField bool) error {
	if m.typeErr != nil {
		return m.typeErr
	}

	return nil
}

func (m *MockController) Eval(ctx context.Context, tabID, expression string) (any, error) {
	if m.evalErr != nil {
		return nil, m.evalErr
	}

	return m.evalResult, nil
}

func (m *MockController) GetConsoleLogs(ctx context.Context, tabID string, duration time.Duration) ([]browser.ConsoleMessage, error) {
	if m.getConsoleLogsErr != nil {
		return nil, m.getConsoleLogsErr
	}

	return m.getConsoleLogsResult, nil
}

func (m *MockController) GetNetworkRequests(ctx context.Context, tabID string, duration time.Duration) ([]browser.NetworkRequest, error) {
	if m.getNetworkRequestsErr != nil {
		return nil, m.getNetworkRequestsErr
	}

	return m.getNetworkRequestsResult, nil
}

func (m *MockController) DetectAuth(ctx context.Context, tabID string) (*browser.AuthRequirement, error) {
	if m.detectAuthErr != nil {
		return nil, m.detectAuthErr
	}

	return m.detectAuthResult, nil
}

func (m *MockController) WaitForLogin(ctx context.Context, tabID string, auth *browser.AuthRequirement) error {
	if m.waitForLoginErr != nil {
		return m.waitForLoginErr
	}

	return nil
}

func (m *MockController) GetCookies(ctx context.Context) ([]browser.Cookie, error) {
	return []browser.Cookie{}, nil
}

func (m *MockController) SetCookies(ctx context.Context, cookies []browser.Cookie) error {
	return nil
}

func (m *MockController) GetPort() int {
	return 9222
}

// TestNewAdapter verifies adapter creation.
func TestNewAdapter(t *testing.T) {
	tests := []struct {
		name       string
		controller browser.Controller
		eventBus   *events.Bus
		expectErr  bool
	}{
		{
			name:       "valid adapter",
			controller: &MockController{},
			eventBus:   events.NewBus(),
			expectErr:  false,
		},
		{
			name:       "nil controller",
			controller: nil,
			eventBus:   events.NewBus(),
			expectErr:  true,
		},
		{
			name:       "nil event bus (allowed)",
			controller: &MockController{},
			eventBus:   nil,
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := NewAdapter(tt.controller, tt.eventBus)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if adapter == nil {
				t.Fatal("expected adapter, got nil")
			}

			if adapter.controller != tt.controller {
				t.Error("controller not set correctly")
			}

			if adapter.eventBus != tt.eventBus {
				t.Error("eventBus not set correctly")
			}
		})
	}
}

// TestToolsReturnsAllTools verifies Tools() returns all expected tools.
func TestToolsReturnsAllTools(t *testing.T) {
	controller := &MockController{}
	eventBus := events.NewBus()
	adapter, err := NewAdapter(controller, eventBus)
	if err != nil {
		t.Fatalf("NewAdapter() failed: %v", err)
	}

	tools := adapter.Tools()

	expectedToolCount := 14
	if len(tools) != expectedToolCount {
		t.Errorf("Tools() returned %d tools, want %d", len(tools), expectedToolCount)
	}

	// Verify each tool has required fields
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("tool has empty Name")
		}
		if tool.Description == "" {
			t.Errorf("tool %s has empty Description", tool.Name)
		}
		if tool.Execute == nil {
			t.Errorf("tool %s has nil Execute", tool.Name)
		}
	}

	// Verify expected tool names
	expectedTools := map[string]bool{
		"browser_open_url":             false,
		"browser_screenshot":           false,
		"browser_click":                false,
		"browser_type":                 false,
		"browser_evaluate":             false,
		"browser_query":                false,
		"browser_get_console_logs":     false,
		"browser_get_network_requests": false,
		"browser_detect_auth":          false,
		"browser_wait_for_login":       false,
		"browser_get_cookies":          false,
		"browser_set_cookies":          false,
		"browser_export_cookies":       false,
		"browser_import_cookies":       false,
	}

	for _, tool := range tools {
		if _, exists := expectedTools[tool.Name]; !exists {
			t.Errorf("unexpected tool: %s", tool.Name)
		}
		expectedTools[tool.Name] = true
	}

	for name, found := range expectedTools {
		if !found {
			t.Errorf("missing tool: %s", name)
		}
	}
}

// TestOpenURLTool tests the browser_open_url tool.
func TestOpenURLTool(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]any
		setupMock    func(*MockController)
		expectErr    bool
		expectResult string
	}{
		{
			name:      "missing url parameter",
			args:      map[string]any{},
			setupMock: func(m *MockController) {},
			expectErr: true,
		},
		{
			name:      "wrong url type",
			args:      map[string]any{"url": 123},
			setupMock: func(m *MockController) {},
			expectErr: true,
		},
		{
			name: "connect error",
			args: map[string]any{"url": "https://example.com"},
			setupMock: func(m *MockController) {
				m.connectErr = errors.New("connection failed")
			},
			expectErr: true,
		},
		{
			name: "open tab error",
			args: map[string]any{"url": "https://example.com"},
			setupMock: func(m *MockController) {
				m.openTabErr = errors.New("failed to open tab")
			},
			expectErr: true,
		},
		{
			name: "success",
			args: map[string]any{"url": "https://example.com"},
			setupMock: func(m *MockController) {
				m.openTabResult = &browser.Tab{
					ID:    "tab123",
					URL:   "https://example.com",
					Title: "Example Domain",
				}
			},
			expectErr:    false,
			expectResult: "Opened https://example.com in tab tab123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &MockController{}
			tt.setupMock(controller)

			adapter, err := NewAdapter(controller, events.NewBus())
			if err != nil {
				t.Fatalf("NewAdapter() failed: %v", err)
			}

			tools := adapter.Tools()
			var tool *Tool
			for _, t := range tools {
				if t.Name == "browser_open_url" {
					tool = &t

					break
				}
			}

			if tool == nil {
				t.Fatal("browser_open_url tool not found")
			}

			result, err := tool.Execute(context.Background(), tt.args)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")

				return
			}

			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if tt.expectResult != "" && result != tt.expectResult {
				t.Errorf("result = %s, want %s", result, tt.expectResult)
			}
		})
	}
}

// TestScreenshotTool tests the browser_screenshot tool.
func TestScreenshotTool(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*MockController)
		expectErr bool
	}{
		{
			name:      "missing tab_id",
			args:      map[string]any{},
			setupMock: func(m *MockController) {},
			expectErr: true,
		},
		{
			name: "valid screenshot",
			args: map[string]any{
				"tab_id":    "tab123",
				"full_page": true,
				"format":    "png",
				"quality":   90,
			},
			setupMock: func(m *MockController) {
				m.screenshotResult = []byte("fake-screenshot-data")
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &MockController{}
			tt.setupMock(controller)

			adapter, _ := NewAdapter(controller, events.NewBus())
			tools := adapter.Tools()

			var tool *Tool
			for _, t := range tools {
				if t.Name == "browser_screenshot" {
					tool = &t

					break
				}
			}

			result, err := tool.Execute(context.Background(), tt.args)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectErr && result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

// TestClickTool tests the browser_click tool.
func TestClickTool(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*MockController)
		expectErr bool
	}{
		{
			name:      "missing tab_id",
			args:      map[string]any{"selector": "#button"},
			setupMock: func(m *MockController) {},
			expectErr: true,
		},
		{
			name:      "missing selector",
			args:      map[string]any{"tab_id": "tab123"},
			setupMock: func(m *MockController) {},
			expectErr: true,
		},
		{
			name: "click error",
			args: map[string]any{
				"tab_id":   "tab123",
				"selector": "#button",
			},
			setupMock: func(m *MockController) {
				m.clickErr = errors.New("element not found")
			},
			expectErr: true,
		},
		{
			name: "success",
			args: map[string]any{
				"tab_id":   "tab123",
				"selector": "#submit",
			},
			setupMock: func(m *MockController) {},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &MockController{}
			tt.setupMock(controller)

			adapter, _ := NewAdapter(controller, events.NewBus())
			tools := adapter.Tools()

			var tool *Tool
			for _, t := range tools {
				if t.Name == "browser_click" {
					tool = &t

					break
				}
			}

			result, err := tool.Execute(context.Background(), tt.args)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectErr && result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

// TestTypeTool tests the browser_type tool.
func TestTypeTool(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*MockController)
		expectErr bool
	}{
		{
			name: "missing required parameter",
			args: map[string]any{
				"tab_id":   "tab123",
				"selector": "#input",
			},
			setupMock: func(m *MockController) {},
			expectErr: true,
		},
		{
			name: "success with clear",
			args: map[string]any{
				"tab_id":   "tab123",
				"selector": "#input",
				"text":     "hello",
				"clear":    true,
			},
			setupMock: func(m *MockController) {},
			expectErr: false,
		},
		{
			name: "success without clear",
			args: map[string]any{
				"tab_id":   "tab123",
				"selector": "#input",
				"text":     "world",
			},
			setupMock: func(m *MockController) {},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &MockController{}
			tt.setupMock(controller)

			adapter, _ := NewAdapter(controller, events.NewBus())
			tools := adapter.Tools()

			var tool *Tool
			for _, t := range tools {
				if t.Name == "browser_type" {
					tool = &t

					break
				}
			}

			_, err := tool.Execute(context.Background(), tt.args)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestEvaluateTool tests the browser_evaluate tool.
func TestEvaluateTool(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]any
		setupMock    func(*MockController)
		expectErr    bool
		expectResult string
	}{
		{
			name:      "missing expression",
			args:      map[string]any{"tab_id": "tab123"},
			setupMock: func(m *MockController) {},
			expectErr: true,
		},
		{
			name: "eval error",
			args: map[string]any{
				"tab_id":     "tab123",
				"expression": "invalid syntax",
			},
			setupMock: func(m *MockController) {
				m.evalErr = errors.New("syntax error")
			},
			expectErr: true,
		},
		{
			name: "success",
			args: map[string]any{
				"tab_id":     "tab123",
				"expression": "1 + 1",
			},
			setupMock: func(m *MockController) {
				m.evalResult = 2
			},
			expectErr:    false,
			expectResult: "Result: 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &MockController{}
			tt.setupMock(controller)

			adapter, _ := NewAdapter(controller, events.NewBus())
			tools := adapter.Tools()

			var tool *Tool
			for _, t := range tools {
				if t.Name == "browser_evaluate" {
					tool = &t

					break
				}
			}

			result, err := tool.Execute(context.Background(), tt.args)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectResult != "" && result != tt.expectResult {
				t.Errorf("result = %s, want %s", result, tt.expectResult)
			}
		})
	}
}

// TestQueryTool tests the browser_query tool.
func TestQueryTool(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*MockController)
		expectErr bool
	}{
		{
			name:      "missing selector",
			args:      map[string]any{"tab_id": "tab123"},
			setupMock: func(m *MockController) {},
			expectErr: true,
		},
		{
			name: "query single element",
			args: map[string]any{
				"tab_id":   "tab123",
				"selector": "h1",
				"all":      false,
			},
			setupMock: func(m *MockController) {
				m.querySelectorResult = &browser.DOMElement{
					TagName:     "H1",
					TextContent: "Welcome",
					Visible:     true,
				}
			},
			expectErr: false,
		},
		{
			name: "query all elements",
			args: map[string]any{
				"tab_id":   "tab123",
				"selector": "p",
				"all":      true,
			},
			setupMock: func(m *MockController) {
				m.querySelectorAllResult = []browser.DOMElement{
					{TagName: "P", TextContent: "Para 1"},
					{TagName: "P", TextContent: "Para 2"},
				}
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &MockController{}
			tt.setupMock(controller)

			adapter, _ := NewAdapter(controller, events.NewBus())
			tools := adapter.Tools()

			var tool *Tool
			for _, t := range tools {
				if t.Name == "browser_query" {
					tool = &t

					break
				}
			}

			result, err := tool.Execute(context.Background(), tt.args)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectErr && result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

// TestGetConsoleLogsTool tests the browser_get_console_logs tool.
func TestGetConsoleLogsTool(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*MockController)
		expectErr bool
	}{
		{
			name:      "missing tab_id",
			args:      map[string]any{},
			setupMock: func(m *MockController) {},
			expectErr: true,
		},
		{
			name: "success with default duration",
			args: map[string]any{"tab_id": "tab123"},
			setupMock: func(m *MockController) {
				m.getConsoleLogsResult = []browser.ConsoleMessage{
					{Level: "info", Text: "log message"},
				}
			},
			expectErr: false,
		},
		{
			name: "success with custom duration",
			args: map[string]any{
				"tab_id":           "tab123",
				"duration_seconds": 10,
			},
			setupMock: func(m *MockController) {
				m.getConsoleLogsResult = []browser.ConsoleMessage{}
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &MockController{}
			tt.setupMock(controller)

			adapter, _ := NewAdapter(controller, events.NewBus())
			tools := adapter.Tools()

			var tool *Tool
			for _, t := range tools {
				if t.Name == "browser_get_console_logs" {
					tool = &t

					break
				}
			}

			result, err := tool.Execute(context.Background(), tt.args)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectErr && result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

// TestDetectAuthTool tests the browser_detect_auth tool.
func TestDetectAuthTool(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]any
		setupMock    func(*MockController)
		expectErr    bool
		expectResult string
	}{
		{
			name:      "missing tab_id",
			args:      map[string]any{},
			setupMock: func(m *MockController) {},
			expectErr: true,
		},
		{
			name: "no auth required",
			args: map[string]any{"tab_id": "tab123"},
			setupMock: func(m *MockController) {
				m.detectAuthResult = nil
			},
			expectErr:    false,
			expectResult: "No authentication required",
		},
		{
			name: "auth required",
			args: map[string]any{"tab_id": "tab123"},
			setupMock: func(m *MockController) {
				m.detectAuthResult = &browser.AuthRequirement{
					Type: "login_form",
					Hint: "Please login",
				}
			},
			expectErr:    false,
			expectResult: "Authentication required: Please login (type: login_form)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &MockController{}
			tt.setupMock(controller)

			adapter, _ := NewAdapter(controller, events.NewBus())
			tools := adapter.Tools()

			var tool *Tool
			for _, t := range tools {
				if t.Name == "browser_detect_auth" {
					tool = &t

					break
				}
			}

			result, err := tool.Execute(context.Background(), tt.args)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectResult != "" && result != tt.expectResult {
				t.Errorf("result = %s, want %s", result, tt.expectResult)
			}
		})
	}
}

// TestWaitForLoginTool tests the browser_wait_for_login tool.
func TestWaitForLoginTool(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]any
		setupMock    func(*MockController)
		expectErr    bool
		expectResult string
	}{
		{
			name: "no auth required",
			args: map[string]any{"tab_id": "tab123"},
			setupMock: func(m *MockController) {
				m.detectAuthResult = nil
			},
			expectErr:    false,
			expectResult: "No authentication required, continuing workflow",
		},
		{
			name: "wait for login success",
			args: map[string]any{"tab_id": "tab123"},
			setupMock: func(m *MockController) {
				m.detectAuthResult = &browser.AuthRequirement{
					Type: "login_form",
				}
			},
			expectErr:    false,
			expectResult: "User completed login, continuing workflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &MockController{}
			tt.setupMock(controller)

			adapter, _ := NewAdapter(controller, events.NewBus())
			tools := adapter.Tools()

			var tool *Tool
			for _, t := range tools {
				if t.Name == "browser_wait_for_login" {
					tool = &t

					break
				}
			}

			result, err := tool.Execute(context.Background(), tt.args)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectResult != "" && result != tt.expectResult {
				t.Errorf("result = %s, want %s", result, tt.expectResult)
			}
		})
	}
}

// TestAdapterIntegration tests adapter with real browser (if available).
func TestAdapterIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping adapter integration test in short mode")
	}

	if os.Getenv("TEST_BROWSER_VISIBLE") != "true" {
		t.Skip("set TEST_BROWSER_VISIBLE=true to run integration test")
	}

	ctx := context.Background()

	cfg := browser.Config{
		Host:     "localhost",
		Port:     0,
		Headless: true,
		Timeout:  30 * time.Second,
	}

	controller := browser.NewController(cfg)
	if err := controller.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer func() { _ = controller.Disconnect() }()

	adapter, err := NewAdapter(controller, events.NewBus())
	if err != nil {
		t.Fatalf("NewAdapter() failed: %v", err)
	}

	tools := adapter.Tools()
	if len(tools) == 0 {
		t.Fatal("Tools() returned empty list")
	}

	// Test browser_open_url tool
	var openURLTool *Tool
	for _, t := range tools {
		if t.Name == "browser_open_url" {
			openURLTool = &t

			break
		}
	}

	if openURLTool == nil {
		t.Fatal("browser_open_url tool not found")
	}

	result, err := openURLTool.Execute(ctx, map[string]any{
		"url": "https://example.com",
	})
	if err != nil {
		t.Fatalf("browser_open_url failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}
