package events

import (
	"github.com/valksor/go-toolkit/eventbus"
)

// Re-export Bus type and constructor from go-toolkit for backward compatibility.
type (
	// Handler processes events.
	Handler = eventbus.Handler
	// Subscription tracks a handler registration.
	Subscription = eventbus.Subscription
	// Bus manages event pub/sub.
	Bus = eventbus.Bus
)

// NewBus creates a new event bus.
func NewBus() *Bus {
	return eventbus.NewBus()
}
