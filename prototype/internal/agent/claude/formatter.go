// Package claude provides Claude-specific agent functionality including stream formatting.
package claude

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
)

// FormatterState tracks session metrics for display.
type FormatterState struct {
	TotalCost   float64
	TotalInput  int
	TotalCached int
	TotalOutput int
	StartTime   time.Time
	EditCount   int
}

// FormatEvent formats a single event for display.
func (f *FormatterState) FormatEvent(event agent.Event) string {
	switch event.Type {
	case agent.EventText:
		return f.formatText(event)
	case agent.EventToolUse:
		return f.formatToolUse(event)
	case agent.EventToolResult:
		return f.formatToolResult(event)
	case agent.EventError:
		return f.formatError(event)
	case agent.EventComplete:
		return f.formatComplete(event)
	case agent.EventUsage:
		return f.formatUsage(event)
	case agent.EventFile:
		return "" // File events don't need special formatting
	default:
		return ""
	}
}

// formatText formats streaming text content.
func (f *FormatterState) formatText(event agent.Event) string {
	// Use the Text field which is already extracted
	return event.Text
}

// formatToolUse formats a tool use event.
func (f *FormatterState) formatToolUse(event agent.Event) string {
	// Use the ToolCall field which is already parsed
	toolUse := event.ToolCall
	if toolUse == nil {
		return ""
	}

	var icon string
	var color string

	switch toolUse.Name {
	case "Read":
		icon, color = "▸", "\033[36m" // Cyan
	case "Edit":
		icon, color = "▸", "\033[33m" // Yellow
	case "Write":
		icon, color = "✓", "\033[92m" // Light green
		f.EditCount++
	case "Bash":
		icon, color = "▸", "\033[90m" // Gray
	case "Grep":
		icon, color = "▸", "\033[34m" // Blue
	case "Glob":
		icon, color = "▸", "\033[34m" // Blue
	case "Task":
		icon, color = "▸", "\033[35m" // Magenta
	case "TodoWrite":
		icon, color = "✓", "\033[92m" // Light green
	default:
		icon, color = "▸", "\033[90m" // Gray
	}

	reset := "\033[0m"

	// Build summary line
	var parts []string
	parts = append(parts, fmt.Sprintf("%s%s%s %s", color, icon, reset, toolUse.Name))

	// Add file path if available
	if fp, ok := toolUse.Input["file_path"].(string); ok {
		parts = append(parts, fp)
	}

	return strings.Join(parts, " ")
}

// formatToolResult formats a tool result event.
func (f *FormatterState) formatToolResult(event agent.Event) string {
	// For simplicity, just show an icon indicating completion
	// Could add status indicators, error messages, etc.
	return "  └─ " + "\033[92m" + "done" + "\033[0m"
}

// formatError formats an error event.
func (f *FormatterState) formatError(event agent.Event) string {
	// Extract error message from Data
	if msg, ok := event.Data["message"].(string); ok {
		return "\033[91m" + "✗ " + msg + "\033[0m" // Red
	}

	return "\033[91m" + "✗ Unknown error" + "\033[0m"
}

// formatComplete formats a completion event.
func (f *FormatterState) formatComplete(agent.Event) string {
	duration := time.Since(f.StartTime)
	cachedPct := 0.0
	if f.TotalInput > 0 {
		cachedPct = float64(f.TotalCached) / float64(f.TotalInput) * 100
	}

	summary := "────────────────────────────────────────────\n"
	summary += fmt.Sprintf("\033[92m✓\033[0m Completed in %ds | $%.4f | %.0f%% cached | %d edit%s\n",
		int(duration.Seconds()),
		f.TotalCost,
		cachedPct,
		f.EditCount,
		plural(f.EditCount))

	return summary
}

// formatUsage formats a usage event and updates totals.
func (f *FormatterState) formatUsage(event agent.Event) string {
	// Extract usage from Data map
	if usageMap, ok := event.Data["usage"].(map[string]any); ok {
		if inputTokens, ok := usageMap["input_tokens"].(float64); ok {
			f.TotalInput += int(inputTokens)
		}
		if cachedTokens, ok := usageMap["cached_input_tokens"].(float64); ok {
			f.TotalCached += int(cachedTokens)
		}
		if outputTokens, ok := usageMap["output_tokens"].(float64); ok {
			f.TotalOutput += int(outputTokens)
		}
		if cost, ok := usageMap["total_cost_usd"].(float64); ok {
			f.TotalCost += cost
		}
	}

	return ""
}

// AddUsage adds usage data from a parsed event.
func (f *FormatterState) AddUsage(inputTokens, cachedTokens, outputTokens int, cost float64) {
	f.TotalInput += inputTokens
	f.TotalCached += cachedTokens
	f.TotalOutput += outputTokens
	f.TotalCost += cost
}

// Reset clears all state.
func (f *FormatterState) Reset() {
	f.TotalCost = 0
	f.TotalInput = 0
	f.TotalCached = 0
	f.TotalOutput = 0
	f.EditCount = 0
	f.StartTime = time.Now()
}

// SessionSummary returns a formatted session summary.
func (f *FormatterState) SessionSummary() string {
	duration := time.Since(f.StartTime)
	cachedPct := 0.0
	if f.TotalInput > 0 {
		cachedPct = float64(f.TotalCached) / float64(f.TotalInput) * 100
	}

	return fmt.Sprintf("Duration: %ds, Cost: $%.4f, Tokens: %d input (%.0f%% cached), %d output, Edits: %d",
		int(duration.Seconds()),
		f.TotalCost,
		f.TotalInput,
		cachedPct,
		f.TotalOutput,
		f.EditCount,
	)
}

// plural returns "s" if n != 1, else "".
func plural(n int) string {
	if n == 1 {
		return ""
	}

	return "s"
}

// ParseUsageEvent parses a JSON usage event from the Claude CLI.
func ParseUsageEvent(data []byte) (int, int, int, float64, error) {
	var event map[string]interface{}
	if err := json.Unmarshal(data, &event); err != nil {
		return 0, 0, 0, 0, err
	}

	var inputTokens, cachedTokens, outputTokens int
	var cost float64

	// Extract token counts
	if v, ok := event["input_tokens"].(float64); ok {
		inputTokens = int(v)
	}
	if v, ok := event["cached_input_tokens"].(float64); ok {
		cachedTokens = int(v)
	}
	if v, ok := event["output_tokens"].(float64); ok {
		outputTokens = int(v)
	}

	// Extract cost if provided by Claude
	if v, ok := event["total_cost_usd"].(float64); ok {
		cost = v
	}

	return inputTokens, cachedTokens, outputTokens, cost, nil
}

// WriteStreamTo formats and writes events to a writer.
func WriteStreamTo(events <-chan agent.Event, writer io.Writer) error {
	state := &FormatterState{
		StartTime: time.Now(),
	}

	for event := range events {
		formatted := state.FormatEvent(event)
		if formatted != "" {
			if _, err := fmt.Fprintln(writer, formatted); err != nil {
				return err
			}
		}
	}

	return nil
}

// SimpleWriter is a simple event writer that formats events as JSON.
type SimpleWriter struct {
	writer io.Writer
}

// NewSimpleWriter creates a new simple event writer.
func NewSimpleWriter(w io.Writer) *SimpleWriter {
	return &SimpleWriter{writer: w}
}

// Write writes an event to the writer.
func (w *SimpleWriter) Write(event agent.Event) error {
	//nolint:musttag // agent.Event is defined in external package
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = w.writer.Write(data)
	if err != nil {
		return err
	}
	_, err = w.writer.Write([]byte("\n"))

	return err
}
