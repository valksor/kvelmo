package memory

import (
	"context"
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestEmbeddingRegistry_DefaultProviders(t *testing.T) {
	r := NewEmbeddingRegistry()

	// Should have default providers
	if !r.Has("default") {
		t.Error("expected 'default' provider to be registered")
	}

	if !r.Has("hash") {
		t.Error("expected 'hash' provider to be registered")
	}
}

func TestEmbeddingRegistry_Create(t *testing.T) {
	r := NewEmbeddingRegistry()
	cfg := storage.VectorDBSettings{}

	// Create default provider
	model, err := r.Create("default", cfg)
	if err != nil {
		t.Fatalf("Create default: %v", err)
	}

	// Verify it works
	embedding, err := model.Embed(context.Background(), "test text")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}

	if len(embedding) != model.Dimension() {
		t.Errorf("embedding dimension mismatch: got %d, want %d", len(embedding), model.Dimension())
	}
}

func TestEmbeddingRegistry_UnknownProvider(t *testing.T) {
	r := NewEmbeddingRegistry()
	cfg := storage.VectorDBSettings{}

	_, err := r.Create("nonexistent", cfg)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestEmbeddingRegistry_Register(t *testing.T) {
	r := NewEmbeddingRegistry()

	// Register a custom provider
	customFactory := func(_ storage.VectorDBSettings) (EmbeddingModel, error) {
		return NewSimpleEmbedding(256), nil
	}

	err := r.Register("custom", customFactory)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Should be in list
	if !r.Has("custom") {
		t.Error("expected 'custom' to be registered")
	}

	// Should be able to create
	model, err := r.Create("custom", storage.VectorDBSettings{})
	if err != nil {
		t.Fatalf("Create custom: %v", err)
	}

	if model.Dimension() != 256 {
		t.Errorf("custom dimension: got %d, want 256", model.Dimension())
	}
}

func TestEmbeddingRegistry_DuplicateRegister(t *testing.T) {
	r := NewEmbeddingRegistry()

	// Try to register over existing
	err := r.Register("default", func(_ storage.VectorDBSettings) (EmbeddingModel, error) {
		return NewSimpleEmbedding(256), nil
	})

	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestEmbeddingRegistry_List(t *testing.T) {
	r := NewEmbeddingRegistry()

	names := r.List()
	if len(names) < 2 {
		t.Errorf("expected at least 2 providers, got %d", len(names))
	}

	// Check default providers are in list
	hasDefault := false
	hasHash := false

	for _, name := range names {
		if name == "default" {
			hasDefault = true
		}

		if name == "hash" {
			hasHash = true
		}
	}

	if !hasDefault {
		t.Error("'default' not in list")
	}

	if !hasHash {
		t.Error("'hash' not in list")
	}
}

func TestDefaultRegistry(t *testing.T) {
	// Test global registry functions
	if !DefaultRegistry.Has("default") {
		t.Error("DefaultRegistry should have 'default' provider")
	}

	model, err := CreateEmbedding("default", storage.VectorDBSettings{})
	if err != nil {
		t.Fatalf("CreateEmbedding: %v", err)
	}

	if model == nil {
		t.Error("expected non-nil model")
	}
}
