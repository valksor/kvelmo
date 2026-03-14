package trace

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type ctxKey struct{}

// NewID generates a new correlation ID.
func NewID() string {
	return uuid.NewString()
}

// WithID returns a new context with the given correlation ID.
func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// ID returns the correlation ID from the context, or empty string if not set.
func ID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)

	return id
}

// SlogAttr returns a slog attribute with the correlation ID from the context.
// Returns an empty attribute if no correlation ID is set.
func SlogAttr(ctx context.Context) slog.Attr {
	id := ID(ctx)
	if id == "" {
		return slog.Attr{}
	}

	return slog.String("correlation_id", id)
}
