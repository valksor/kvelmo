//go:build no_browser
// +build no_browser

package browser

import (
	"context"
	"sync"
	"time"
)

// ConsoleMonitor stub when browser is disabled.
type ConsoleMonitor struct {
	messages []ConsoleMessage
	mu       sync.RWMutex
	filter   ConsoleFilter
}

// NewConsoleMonitor returns a stub console monitor.
func NewConsoleMonitor(filter ConsoleFilter) *ConsoleMonitor {
	return &ConsoleMonitor{
		messages: []ConsoleMessage{},
		filter:   filter,
	}
}

// NewConsoleMonitorAll returns a stub console monitor.
func NewConsoleMonitorAll() *ConsoleMonitor {
	return &ConsoleMonitor{
		messages: []ConsoleMessage{},
		filter:   ConsoleFilter{},
	}
}

// Start returns an error - browser is disabled.
func (m *ConsoleMonitor) Start(ctx context.Context, page interface{}) error {
	return ErrDisabled
}

// Stop is a no-op.
func (m *ConsoleMonitor) Stop() error {
	return nil
}

// GetMessages returns empty list - no console activity when disabled.
func (m *ConsoleMonitor) GetMessages() []ConsoleMessage {
	return []ConsoleMessage{}
}

// GetErrors returns empty list.
func (m *ConsoleMonitor) GetErrors() []ConsoleMessage {
	return []ConsoleMessage{}
}

// GetWarnings returns empty list.
func (m *ConsoleMonitor) GetWarnings() []ConsoleMessage {
	return []ConsoleMessage{}
}

// GetMessagesByLevel returns empty list.
func (m *ConsoleMonitor) GetMessagesByLevel(level string) []ConsoleMessage {
	return []ConsoleMessage{}
}

// GetMessagesByPattern returns empty list.
func (m *ConsoleMonitor) GetMessagesByPattern(pattern string) []ConsoleMessage {
	return []ConsoleMessage{}
}

// GetMessagesSince returns empty list.
func (m *ConsoleMonitor) GetMessagesSince(since time.Time) []ConsoleMessage {
	return []ConsoleMessage{}
}

// GetMessagesForURL returns empty list.
func (m *ConsoleMonitor) GetMessagesForURL(url string) []ConsoleMessage {
	return []ConsoleMessage{}
}

// Clear is a no-op.
func (m *ConsoleMonitor) Clear() {}

// Count returns 0 - no messages when disabled.
func (m *ConsoleMonitor) Count() int {
	return 0
}
