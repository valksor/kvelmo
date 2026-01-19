package browser

import (
	"testing"
	"time"
)

// TestConsoleMonitor tests console message monitoring functionality.
func TestConsoleMonitor(t *testing.T) {
	t.Run("NewConsoleMonitor", func(t *testing.T) {
		filter := ConsoleFilter{
			Levels:  []string{"error", "warn"},
			Pattern: "test",
		}

		mon := NewConsoleMonitor(filter)
		if mon.filter.Levels == nil {
			t.Error("filter levels not set")
		}
		if mon.filter.Pattern != "test" {
			t.Errorf("filter.Pattern = %s, want 'test'", mon.filter.Pattern)
		}
		if len(mon.GetMessages()) != 0 {
			t.Error("new monitor should have no messages")
		}
	})

	t.Run("NewConsoleMonitorAll", func(t *testing.T) {
		mon := NewConsoleMonitorAll()
		if mon.filter.Levels != nil {
			t.Error("filter levels should be nil for 'all' monitor")
		}
		if mon.filter.Pattern != "" {
			t.Error("filter pattern should be empty for 'all' monitor")
		}
	})

	t.Run("AddMessage", func(t *testing.T) {
		mon := NewConsoleMonitorAll()

		msg := ConsoleMessage{
			Level:     "info",
			Text:      "Test message",
			URL:       "https://example.com",
			Timestamp: time.Now(),
		}

		mon.AddMessage(msg)

		messages := mon.GetMessages()
		if len(messages) != 1 {
			t.Errorf("got %d messages, want 1", len(messages))
		}

		if messages[0].Text != "Test message" {
			t.Errorf("Text = %s, want 'Test message'", messages[0].Text)
		}
	})

	t.Run("AddMessageWithFilter", func(t *testing.T) {
		filter := ConsoleFilter{
			Levels: []string{"error"},
		}
		mon := NewConsoleMonitor(filter)

		// Add error message (should pass filter)
		errorMsg := ConsoleMessage{
			Level:     "error",
			Text:      "Error occurred",
			Timestamp: time.Now(),
		}
		mon.AddMessage(errorMsg)

		// Add info message (should be filtered out)
		infoMsg := ConsoleMessage{
			Level:     "info",
			Text:      "Info message",
			Timestamp: time.Now(),
		}
		mon.AddMessage(infoMsg)

		messages := mon.GetMessages()
		if len(messages) != 1 {
			t.Errorf("got %d messages, want 1 (info should be filtered)", len(messages))
		}

		if messages[0].Level != "error" {
			t.Errorf("Level = %s, want 'error'", messages[0].Level)
		}
	})

	t.Run("GetMessages", func(t *testing.T) {
		mon := NewConsoleMonitorAll()

		// Add multiple messages
		msg1 := ConsoleMessage{
			Level:     "info",
			Text:      "Message 1",
			Timestamp: time.Now(),
		}
		msg2 := ConsoleMessage{
			Level:     "warn",
			Text:      "Message 2",
			Timestamp: time.Now(),
		}
		msg3 := ConsoleMessage{
			Level:     "error",
			Text:      "Message 3",
			Timestamp: time.Now(),
		}

		mon.AddMessage(msg1)
		mon.AddMessage(msg2)
		mon.AddMessage(msg3)

		messages := mon.GetMessages()
		if len(messages) != 3 {
			t.Errorf("got %d messages, want 3", len(messages))
		}

		// Verify order is preserved
		if messages[0].Text != "Message 1" {
			t.Errorf("first message = %s, want 'Message 1'", messages[0].Text)
		}
		if messages[1].Text != "Message 2" {
			t.Errorf("second message = %s, want 'Message 2'", messages[1].Text)
		}
		if messages[2].Text != "Message 3" {
			t.Errorf("third message = %s, want 'Message 3'", messages[2].Text)
		}
	})

	t.Run("GetErrors", func(t *testing.T) {
		mon := NewConsoleMonitorAll()

		// Add messages of different levels
		msg1 := ConsoleMessage{Level: "info", Text: "Info", Timestamp: time.Now()}
		msg2 := ConsoleMessage{Level: "warn", Text: "Warning", Timestamp: time.Now()}
		msg3 := ConsoleMessage{Level: "error", Text: "Error", Timestamp: time.Now()}
		msg4 := ConsoleMessage{Level: "error", Text: "Another error", Timestamp: time.Now()}

		mon.AddMessage(msg1)
		mon.AddMessage(msg2)
		mon.AddMessage(msg3)
		mon.AddMessage(msg4)

		errors := mon.GetErrors()
		if len(errors) != 2 {
			t.Errorf("got %d errors, want 2", len(errors))
		}

		for _, err := range errors {
			if err.Level != "error" {
				t.Errorf("got level %s, want 'error'", err.Level)
			}
		}
	})

	t.Run("GetWarnings", func(t *testing.T) {
		mon := NewConsoleMonitorAll()

		// Add messages
		mon.AddMessage(ConsoleMessage{Level: "info", Text: "Info", Timestamp: time.Now()})
		mon.AddMessage(ConsoleMessage{Level: "warning", Text: "Warning 1", Timestamp: time.Now()})
		mon.AddMessage(ConsoleMessage{Level: "error", Text: "Error", Timestamp: time.Now()})
		mon.AddMessage(ConsoleMessage{Level: "warning", Text: "Warning 2", Timestamp: time.Now()})

		warnings := mon.GetWarnings()
		if len(warnings) != 2 {
			t.Errorf("got %d warnings, want 2", len(warnings))
		}

		for _, warn := range warnings {
			if warn.Level != "warning" {
				t.Errorf("got level %s, want 'warning'", warn.Level)
			}
		}
	})

	t.Run("GetMessagesByLevel", func(t *testing.T) {
		mon := NewConsoleMonitorAll()

		// Add messages with different levels
		levels := []string{"info", "warn", "error", "debug", "info"}
		for i, level := range levels {
			mon.AddMessage(ConsoleMessage{
				Level:     level,
				Text:      "Message " + string(rune('A'+i)),
				Timestamp: time.Now(),
			})
		}

		// Test each level
		infoMsgs := mon.GetMessagesByLevel("info")
		if len(infoMsgs) != 2 {
			t.Errorf("got %d info messages, want 2", len(infoMsgs))
		}

		warnMsgs := mon.GetMessagesByLevel("warn")
		if len(warnMsgs) != 1 {
			t.Errorf("got %d warn messages, want 1", len(warnMsgs))
		}

		errorMsgs := mon.GetMessagesByLevel("error")
		if len(errorMsgs) != 1 {
			t.Errorf("got %d error messages, want 1", len(errorMsgs))
		}

		// Non-existent level
		fooMsgs := mon.GetMessagesByLevel("foo")
		if len(fooMsgs) != 0 {
			t.Errorf("got %d foo messages, want 0", len(fooMsgs))
		}
	})

	t.Run("GetMessagesByPattern", func(t *testing.T) {
		mon := NewConsoleMonitorAll()

		// Add messages with different text
		msgs := []string{
			"Error: API request failed",
			"Warning: High memory usage",
			"Error: Database connection lost",
			"Info: Application started",
		}
		for _, text := range msgs {
			mon.AddMessage(ConsoleMessage{
				Level:     "info",
				Text:      text,
				Timestamp: time.Now(),
			})
		}

		// Get messages containing "Error"
		errorMsgs := mon.GetMessagesByPattern("Error")
		if len(errorMsgs) != 2 {
			t.Errorf("got %d messages with 'Error', want 2", len(errorMsgs))
		}

		// Get messages containing "API"
		apiMsgs := mon.GetMessagesByPattern("API")
		if len(apiMsgs) != 1 {
			t.Errorf("got %d messages with 'API', want 1", len(apiMsgs))
		}

		// Non-existent pattern
		fooMsgs := mon.GetMessagesByPattern("foobar")
		if len(fooMsgs) != 0 {
			t.Errorf("got %d messages with 'foobar', want 0", len(fooMsgs))
		}
	})

	t.Run("GetMessagesSince", func(t *testing.T) {
		mon := NewConsoleMonitorAll()

		now := time.Now()
		oldTime := now.Add(-1 * time.Hour)

		// Add old message
		mon.AddMessage(ConsoleMessage{
			Level:     "info",
			Text:      "Old message",
			Timestamp: oldTime,
		})

		// Add new messages
		newMsg1 := ConsoleMessage{
			Level:     "info",
			Text:      "New message 1",
			Timestamp: now,
		}
		newMsg2 := ConsoleMessage{
			Level:     "warn",
			Text:      "New message 2",
			Timestamp: now.Add(1 * time.Second),
		}

		mon.AddMessage(newMsg1)
		mon.AddMessage(newMsg2)

		// Get messages since 30 minutes ago
		since := now.Add(-30 * time.Minute)
		recentMsgs := mon.GetMessagesSince(since)
		if len(recentMsgs) != 2 {
			t.Errorf("got %d recent messages, want 2", len(recentMsgs))
		}

		// All messages should be after the threshold
		for _, msg := range recentMsgs {
			if !msg.Timestamp.After(since) {
				t.Errorf("message %s is not after threshold", msg.Text)
			}
		}
	})

	t.Run("GetMessagesForURL", func(t *testing.T) {
		mon := NewConsoleMonitorAll()

		// Add messages from different URLs
		msgs := []struct {
			url  string
			text string
		}{
			{"https://example.com/page1", "Message 1"},
			{"https://example.com/page2", "Message 2"},
			{"https://example.com/page1", "Message 3"},
			{"https://other.com/page", "Message 4"},
		}

		for _, msg := range msgs {
			mon.AddMessage(ConsoleMessage{
				Level:     "info",
				Text:      msg.text,
				URL:       msg.url,
				Timestamp: time.Now(),
			})
		}

		// Get messages for page1
		page1Msgs := mon.GetMessagesForURL("https://example.com/page1")
		if len(page1Msgs) != 2 {
			t.Errorf("got %d messages for page1, want 2", len(page1Msgs))
		}

		// Get messages for page2
		page2Msgs := mon.GetMessagesForURL("https://example.com/page2")
		if len(page2Msgs) != 1 {
			t.Errorf("got %d messages for page2, want 1", len(page2Msgs))
		}

		// Non-existent URL
		fooMsgs := mon.GetMessagesForURL("https://example.com/foo")
		if len(fooMsgs) != 0 {
			t.Errorf("got %d messages for non-existent URL, want 0", len(fooMsgs))
		}
	})

	t.Run("Clear", func(t *testing.T) {
		mon := NewConsoleMonitorAll()

		// Add messages
		for range 5 {
			mon.AddMessage(ConsoleMessage{
				Level:     "info",
				Text:      "Message",
				Timestamp: time.Now(),
			})
		}

		if len(mon.GetMessages()) != 5 {
			t.Fatalf("got %d messages, want 5 before clear", len(mon.GetMessages()))
		}

		mon.Clear()

		if len(mon.GetMessages()) != 0 {
			t.Errorf("got %d messages after clear, want 0", len(mon.GetMessages()))
		}
	})

	t.Run("Count", func(t *testing.T) {
		mon := NewConsoleMonitorAll()

		if mon.Count() != 0 {
			t.Errorf("Count() = %d, want 0 initially", mon.Count())
		}

		// Add messages
		for i := 1; i <= 5; i++ {
			mon.AddMessage(ConsoleMessage{
				Level:     "info",
				Text:      "Message",
				Timestamp: time.Now(),
			})

			if mon.Count() != i {
				t.Errorf("Count() = %d, want %d", mon.Count(), i)
			}
		}
	})

	t.Run("FilterWithPattern", func(t *testing.T) {
		filter := ConsoleFilter{
			Pattern: "error",
		}
		mon := NewConsoleMonitor(filter)

		// Add messages with and without pattern
		mon.AddMessage(ConsoleMessage{
			Level:     "info",
			Text:      "This is an error message",
			Timestamp: time.Now(),
		})
		mon.AddMessage(ConsoleMessage{
			Level:     "info",
			Text:      "This is a warning message",
			Timestamp: time.Now(),
		})
		mon.AddMessage(ConsoleMessage{
			Level:     "error",
			Text:      "Critical error occurred",
			Timestamp: time.Now(),
		})

		messages := mon.GetMessages()
		if len(messages) != 2 {
			t.Errorf("got %d messages, want 2 (only those with 'error')", len(messages))
		}

		// Verify pattern matches
		for _, msg := range messages {
			if !contains(msg.Text, "error") {
				t.Errorf("message '%s' does not contain 'error'", msg.Text)
			}
		}
	})

	t.Run("FilterWithURL", func(t *testing.T) {
		filter := ConsoleFilter{
			SourceURL: "https://example.com/page1", // Exact match
		}
		mon := NewConsoleMonitor(filter)

		// Add messages from different URLs
		mon.AddMessage(ConsoleMessage{
			Level:     "info",
			Text:      "Message 1",
			URL:       "https://example.com/page1",
			Timestamp: time.Now(),
		})
		mon.AddMessage(ConsoleMessage{
			Level:     "info",
			Text:      "Message 2",
			URL:       "https://other.com/page",
			Timestamp: time.Now(),
		})

		messages := mon.GetMessages()
		if len(messages) != 1 {
			t.Errorf("got %d messages, want 1 (only from page1)", len(messages))
		}

		if len(messages) > 0 && messages[0].URL != "https://example.com/page1" {
			t.Errorf("URL = %s, want 'https://example.com/page1'", messages[0].URL)
		}
	})

	t.Run("ThreadSafety", func(t *testing.T) {
		mon := NewConsoleMonitorAll()

		// Add messages concurrently
		done := make(chan bool)
		for i := range 10 {
			go func(_ int) {
				msg := ConsoleMessage{
					Level:     "info",
					Text:      "Message",
					Timestamp: time.Now(),
				}
				mon.AddMessage(msg)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for range 10 {
			<-done
		}

		// Verify all messages were added
		if mon.Count() != 10 {
			t.Errorf("Count() = %d, want 10", mon.Count())
		}
	})
}

// TestConsoleMessageSerialization tests console message structure.
func TestConsoleMessageSerialization(t *testing.T) {
	msg := ConsoleMessage{
		Level:     "error",
		Text:      "Test error message",
		URL:       "https://example.com/app.js",
		Line:      42,
		Column:    10,
		Timestamp: time.Now(),
	}

	// Verify all fields are set
	if msg.Level == "" {
		t.Error("Level is empty")
	}
	if msg.Text == "" {
		t.Error("Text is empty")
	}
	if msg.URL == "" {
		t.Error("URL is empty")
	}
	if msg.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
}
