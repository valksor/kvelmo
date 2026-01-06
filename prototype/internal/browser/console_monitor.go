package browser

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ConsoleMonitor monitors console messages for a browser tab.
type ConsoleMonitor struct {
	messages []ConsoleMessage
	mu       sync.RWMutex
	filter   ConsoleFilter
	cancel   context.CancelFunc
	done     chan struct{}
	wg       sync.WaitGroup // Track goroutine
}

// ConsoleFilter defines which console messages to capture.
type ConsoleFilter struct {
	Levels    []string // Capture only these levels (empty = all)
	Pattern   string   // Only capture messages matching this pattern
	SourceURL string   // Only capture messages from this URL
}

// NewConsoleMonitor creates a new console monitor.
func NewConsoleMonitor(filter ConsoleFilter) *ConsoleMonitor {
	return &ConsoleMonitor{
		messages: []ConsoleMessage{},
		filter:   filter,
	}
}

// NewConsoleMonitorAll creates a monitor that captures all console messages.
func NewConsoleMonitorAll() *ConsoleMonitor {
	return &ConsoleMonitor{
		messages: []ConsoleMessage{},
		filter:   ConsoleFilter{},
	}
}

// Start begins monitoring console messages for a page.
func (m *ConsoleMonitor) Start(ctx context.Context, page *rod.Page) error {
	// Enable runtime events (for console messages)
	_ = proto.RuntimeEnable{}.Call(page)

	// Create cancelable context for this monitor
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.done = make(chan struct{})

	// Start event listener in background
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.listenForEvents(ctx, page)
	}()

	return nil
}

// Stop stops the console monitor and cleans up resources.
// Returns error if timeout occurs waiting for goroutine to exit.
func (m *ConsoleMonitor) Stop() error {
	if m.cancel != nil {
		m.cancel()
	}

	// Wait for goroutine to exit with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		// Log warning about goroutine leak
		slog.Warn("console monitor goroutine leak detected",
			"timeout", "5s")
		// Return error but at least we logged it
		return errors.New("console monitor stop timeout after 5s")
	}
}

// listenForEvents listens for console events.
func (m *ConsoleMonitor) listenForEvents(ctx context.Context, page *rod.Page) {
	defer close(m.done)

	// Get page info once (URL won't change during event handling)
	info, _ := page.Info()

	// Use EachEvent to listen for console events
	page.Context(ctx).EachEvent(
		func(e *proto.RuntimeConsoleAPICalled) {
			// Convert console type to our level format
			level := strings.ToLower(string(e.Type))

			// Extract message text from args
			var text strings.Builder
			for i, arg := range e.Args {
				if i > 0 {
					text.WriteString(" ")
				}
				// Use Description which has the string representation
				text.WriteString(arg.Description)
			}

			// Convert timestamp from milliseconds to time.Time
			timestamp := time.Unix(0, int64(e.Timestamp)*int64(time.Millisecond))

			m.AddMessage(ConsoleMessage{
				Level:     level,
				Text:      text.String(),
				URL:       info.URL,
				Timestamp: timestamp,
			})
		},
		func(e *proto.RuntimeExceptionThrown) {
			// Convert timestamp from milliseconds to time.Time
			timestamp := time.Unix(0, int64(e.Timestamp)*int64(time.Millisecond))

			m.AddMessage(ConsoleMessage{
				Level:     "error",
				Text:      e.ExceptionDetails.Exception.Description,
				URL:       info.URL,
				Timestamp: timestamp,
			})
		},
	)()

	// Wait for context cancellation
	<-ctx.Done()
}

// GetMessages returns all captured console messages.
func (m *ConsoleMonitor) GetMessages() []ConsoleMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ConsoleMessage, len(m.messages))
	copy(result, m.messages)

	return result
}

// GetErrors returns only error-level console messages.
func (m *ConsoleMonitor) GetErrors() []ConsoleMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []ConsoleMessage{}
	for _, msg := range m.messages {
		if msg.Level == "error" {
			result = append(result, msg)
		}
	}

	return result
}

// GetWarnings returns only warning-level console messages.
func (m *ConsoleMonitor) GetWarnings() []ConsoleMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []ConsoleMessage{}
	for _, msg := range m.messages {
		if msg.Level == "warning" {
			result = append(result, msg)
		}
	}

	return result
}

// GetMessagesByLevel returns messages filtered by level.
func (m *ConsoleMonitor) GetMessagesByLevel(level string) []ConsoleMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []ConsoleMessage{}
	for _, msg := range m.messages {
		if msg.Level == level {
			result = append(result, msg)
		}
	}

	return result
}

// GetMessagesByPattern returns messages matching a pattern.
func (m *ConsoleMonitor) GetMessagesByPattern(pattern string) []ConsoleMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []ConsoleMessage{}
	for _, msg := range m.messages {
		// Simple contains check
		if contains(msg.Text, pattern) {
			result = append(result, msg)
		}
	}

	return result
}

// GetMessagesSince returns messages after a certain timestamp.
func (m *ConsoleMonitor) GetMessagesSince(since time.Time) []ConsoleMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []ConsoleMessage{}
	for _, msg := range m.messages {
		if msg.Timestamp.After(since) {
			result = append(result, msg)
		}
	}

	return result
}

// GetMessagesForURL returns messages from a specific URL.
func (m *ConsoleMonitor) GetMessagesForURL(url string) []ConsoleMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []ConsoleMessage{}
	for _, msg := range m.messages {
		if msg.URL == url {
			result = append(result, msg)
		}
	}

	return result
}

// Clear clears all captured messages.
func (m *ConsoleMonitor) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = []ConsoleMessage{}
}

// Count returns the number of captured messages.
func (m *ConsoleMonitor) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.messages)
}

// shouldCapture determines if a message should be captured based on filter settings.
func (m *ConsoleMonitor) shouldCapture(msg ConsoleMessage) bool {
	// Check level filter
	if len(m.filter.Levels) > 0 {
		levelMatch := false
		for _, level := range m.filter.Levels {
			if msg.Level == level {
				levelMatch = true

				break
			}
		}
		if !levelMatch {
			return false
		}
	}

	// Check pattern filter
	if m.filter.Pattern != "" && !contains(msg.Text, m.filter.Pattern) {
		return false
	}

	// Check URL filter
	if m.filter.SourceURL != "" && msg.URL != m.filter.SourceURL {
		return false
	}

	return true
}

// AddMessage adds a console message if it passes the filter.
func (m *ConsoleMonitor) AddMessage(msg ConsoleMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldCapture(msg) {
		m.messages = append(m.messages, msg)
	}
}
