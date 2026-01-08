//go:build no_browser
// +build no_browser

package browser

import (
	"context"

	"github.com/valksor/go-mehrhof/internal/browser"
	"github.com/valksor/go-mehrhof/internal/events"
)

// Tool stub when browser is disabled.
type Tool struct {
	Name        string
	Description string
	Execute     func(ctx context.Context, args map[string]any) (string, error)
}

// Adapter stub when browser is disabled.
type Adapter struct {
	controller browser.Controller
	eventBus   *events.Bus
}

// NewAdapter creates a stub adapter when browser is disabled.
func NewAdapter(controller browser.Controller, eventBus *events.Bus) (*Adapter, error) {
	return &Adapter{
		controller: controller,
		eventBus:   eventBus,
	}, nil
}

// Tools returns empty list when browser is disabled.
func (a *Adapter) Tools() []Tool {
	return []Tool{}
}
