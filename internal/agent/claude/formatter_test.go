package claude

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
)

func TestFormatterState_FormatEvent(t *testing.T) {
	state := &FormatterState{
		StartTime: time.Now(),
	}

	tests := []struct {
		name    string
		event   agent.Event
		wantLen int // Just check that output is non-empty
	}{
		{
			name: "text event",
			event: agent.Event{
				Type: agent.EventText,
				Text: "Hello world",
			},
			wantLen: 11,
		},
		{
			name: "tool use event",
			event: agent.Event{
				Type: agent.EventToolUse,
				ToolCall: &agent.ToolCall{
					Name: "Read",
					Input: map[string]any{
						"file_path": "test.go",
					},
				},
			},
			wantLen: 1, // Just check it produces output
		},
		{
			name: "error event",
			event: agent.Event{
				Type: agent.EventError,
				Data: map[string]any{
					"message": "test error",
				},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := state.FormatEvent(tt.event)
			if len(result) < tt.wantLen {
				t.Errorf("FormatEvent() produced output shorter than expected")
			}
		})
	}
}

func TestFormatterState_AddUsage(t *testing.T) {
	state := &FormatterState{}

	state.AddUsage(1000, 200, 500, 0.023)

	if state.TotalInput != 1000 {
		t.Errorf("Expected TotalInput 1000, got %d", state.TotalInput)
	}
	if state.TotalCached != 200 {
		t.Errorf("Expected TotalCached 200, got %d", state.TotalCached)
	}
	if state.TotalOutput != 500 {
		t.Errorf("Expected TotalOutput 500, got %d", state.TotalOutput)
	}
}

func TestFormatterState_SessionSummary(t *testing.T) {
	state := &FormatterState{
		StartTime:   time.Now().Add(-10 * time.Second),
		TotalCost:   0.123,
		TotalInput:  5000,
		TotalCached: 1000,
		TotalOutput: 2000,
		EditCount:   3,
	}

	summary := state.SessionSummary()

	if !strings.Contains(summary, "10s") {
		t.Error("Summary should contain duration")
	}
	if !strings.Contains(summary, "0.123") {
		t.Error("Summary should contain cost")
	}
	if !strings.Contains(summary, "Edits: 3") {
		t.Error("Summary should contain edit count")
	}
}

func TestFormatterState_Reset(t *testing.T) {
	state := &FormatterState{
		TotalCost:   0.5,
		TotalInput:  1000,
		TotalCached: 200,
		TotalOutput: 500,
		EditCount:   5,
		StartTime:   time.Now(),
	}

	state.Reset()

	if state.TotalCost != 0 {
		t.Error("Expected TotalCost to be 0 after reset")
	}
	if state.TotalInput != 0 {
		t.Error("Expected TotalInput to be 0 after reset")
	}
	if state.EditCount != 0 {
		t.Error("Expected EditCount to be 0 after reset")
	}
}

func TestParseUsageEvent(t *testing.T) {
	jsonData := []byte(`{
		"input_tokens": 1000,
		"cached_input_tokens": 200,
		"output_tokens": 500,
		"total_cost_usd": 0.023
	}`)

	input, cached, output, cost, err := ParseUsageEvent(jsonData)
	if err != nil {
		t.Fatalf("ParseUsageEvent() error = %v", err)
	}
	if input != 1000 {
		t.Errorf("Expected input 1000, got %d", input)
	}
	if cached != 200 {
		t.Errorf("Expected cached 200, got %d", cached)
	}
	if output != 500 {
		t.Errorf("Expected output 500, got %d", output)
	}
	if cost != 0.023 {
		t.Errorf("Expected cost 0.023, got %f", cost)
	}
}

func TestParseUsageEvent_InvalidJSON(t *testing.T) {
	inputTokens, cachedTokens, outputTokens, cost, err := ParseUsageEvent([]byte("invalid json"))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
	// Use variables to avoid dogsled warning
	_ = inputTokens
	_ = cachedTokens
	_ = outputTokens
	_ = cost
}

func TestFormatToolUse_Edit(t *testing.T) {
	state := &FormatterState{}

	event := agent.Event{
		Type: agent.EventToolUse,
		ToolCall: &agent.ToolCall{
			Name: "Write",
			Input: map[string]any{
				"file_path": "test.go",
			},
		},
	}

	result := state.FormatEvent(event)

	if !strings.Contains(result, "Write") {
		t.Error("Output should contain tool name")
	}

	// Check that edit count was incremented
	if state.EditCount != 1 {
		t.Errorf("Expected EditCount 1, got %d", state.EditCount)
	}
}

func TestPlural(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "s"},
		{1, ""},
		{2, "s"},
		{10, "s"},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.n), func(t *testing.T) {
			if got := plural(tt.n); got != tt.want {
				t.Errorf("plural(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestSimpleWriter_Write(t *testing.T) {
	var buf strings.Builder
	writer := NewSimpleWriter(&buf)

	event := agent.Event{
		Type: agent.EventText,
		Text: "test message",
	}

	err := writer.Write(event)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("Output should contain event data")
	}
	if !strings.HasSuffix(output, "\n") {
		t.Error("Output should end with newline")
	}
}
