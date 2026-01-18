package log

import "log/slog"

// Domain-specific helpers for mehrhof

// TaskID is a helper for logging task IDs.
func TaskID(id string) slog.Attr {
	return slog.String("task_id", id)
}

// State is a helper for logging state transitions.
func State(from, to string) []slog.Attr {
	return []slog.Attr{
		slog.String("from", from),
		slog.String("to", to),
	}
}
