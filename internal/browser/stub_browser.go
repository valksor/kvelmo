//go:build no_browser
// +build no_browser

package browser

import (
	"context"
	"errors"
	"time"
)

// ErrDisabled is returned when browser operations are attempted in a build without browser support.
var ErrDisabled = errors.New("browser support is disabled in this build (built with 'no_browser' tag)")

// stubController implements Controller interface when browser is disabled.
type stubController struct{}

// NewController creates a stub controller when browser is disabled.
func NewController(config Config) Controller {
	return &stubController{}
}

func (c *stubController) Connect(ctx context.Context) error {
	return errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) Disconnect() error {
	return nil // No-op
}

func (c *stubController) IsConnected() bool {
	return false
}

func (c *stubController) GetPort() int {
	return 0
}

func (c *stubController) ListTabs(ctx context.Context) ([]Tab, error) {
	return nil, errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) OpenTab(ctx context.Context, url string) (*Tab, error) {
	return nil, errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) CloseTab(ctx context.Context, tabID string) error {
	return errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) SwitchTab(ctx context.Context, tabID string) (*Tab, error) {
	return nil, errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) Navigate(ctx context.Context, tabID, url string) error {
	return errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) Reload(ctx context.Context, tabID string, hard bool) error {
	return errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) Screenshot(ctx context.Context, tabID string, opts ScreenshotOptions) ([]byte, error) {
	return nil, errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) QuerySelector(ctx context.Context, tabID, selector string) (*DOMElement, error) {
	return nil, errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) QuerySelectorAll(ctx context.Context, tabID, selector string) ([]DOMElement, error) {
	return nil, errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) Click(ctx context.Context, tabID, selector string) error {
	return errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) Type(ctx context.Context, tabID, selector, text string, clearField bool) error {
	return errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) Eval(ctx context.Context, tabID, expression string) (any, error) {
	return nil, errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) GetConsoleLogs(ctx context.Context, tabID string, duration time.Duration) ([]ConsoleMessage, error) {
	return nil, errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

func (c *stubController) GetNetworkRequests(ctx context.Context, tabID string, duration time.Duration) ([]NetworkRequest, error) {
	return nil, errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}
