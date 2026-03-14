package trace_test

import (
	"context"
	"testing"

	"github.com/valksor/kvelmo/pkg/trace"
)

func TestNewID(t *testing.T) {
	id1 := trace.NewID()
	id2 := trace.NewID()

	if id1 == "" {
		t.Fatal("NewID returned empty string")
	}
	if id2 == "" {
		t.Fatal("NewID returned empty string")
	}
	if id1 == id2 {
		t.Fatalf("NewID returned duplicate IDs: %s", id1)
	}
}

func TestWithIDAndID(t *testing.T) {
	ctx := context.Background()
	want := "test-correlation-id-123"

	ctx = trace.WithID(ctx, want)
	got := trace.ID(ctx)

	if got != want {
		t.Fatalf("ID() = %q, want %q", got, want)
	}
}

func TestIDEmptyContext(t *testing.T) {
	ctx := context.Background()
	got := trace.ID(ctx)

	if got != "" {
		t.Fatalf("ID() on empty context = %q, want empty string", got)
	}
}

func TestSlogAttrWithID(t *testing.T) {
	ctx := trace.WithID(context.Background(), "abc-123")
	attr := trace.SlogAttr(ctx)

	if attr.Key != "correlation_id" {
		t.Fatalf("SlogAttr key = %q, want %q", attr.Key, "correlation_id")
	}
	if attr.Value.String() != "abc-123" {
		t.Fatalf("SlogAttr value = %q, want %q", attr.Value.String(), "abc-123")
	}
}

func TestSlogAttrWithoutID(t *testing.T) {
	ctx := context.Background()
	attr := trace.SlogAttr(ctx)

	if attr.Key != "" {
		t.Fatalf("SlogAttr on empty context: key = %q, want empty", attr.Key)
	}
}
