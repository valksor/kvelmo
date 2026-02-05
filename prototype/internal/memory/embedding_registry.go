package memory

import (
	"fmt"
	"sync"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// EmbeddingFactory is a function that creates an EmbeddingModel from config.
type EmbeddingFactory func(cfg storage.VectorDBSettings) (EmbeddingModel, error)

// EmbeddingRegistry manages embedding model factories.
type EmbeddingRegistry struct {
	mu        sync.RWMutex
	factories map[string]EmbeddingFactory
}

// NewEmbeddingRegistry creates a new embedding registry with default providers.
func NewEmbeddingRegistry() *EmbeddingRegistry {
	r := &EmbeddingRegistry{
		factories: make(map[string]EmbeddingFactory),
	}

	// Register built-in providers (errors ignored - these are guaranteed to succeed)
	_ = r.Register("default", newLocalHashEmbeddingFactory)
	_ = r.Register("hash", newLocalHashEmbeddingFactory)
	_ = r.Register("onnx", NewEmbedderClientFromConfig) // ONNX via sidecar

	return r
}

// Register adds an embedding factory to the registry.
func (r *EmbeddingRegistry) Register(name string, factory EmbeddingFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("embedding factory already registered: %s", name)
	}

	r.factories[name] = factory

	return nil
}

// Create instantiates an embedding model by name.
func (r *EmbeddingRegistry) Create(name string, cfg storage.VectorDBSettings) (EmbeddingModel, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown embedding model: %s (available: %v)", name, r.List())
	}

	return factory(cfg)
}

// List returns all registered embedding model names.
func (r *EmbeddingRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}

	return names
}

// Has checks if an embedding model is registered.
func (r *EmbeddingRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.factories[name]

	return ok
}

// newLocalHashEmbeddingFactory creates a LocalHashEmbedding from config.
func newLocalHashEmbeddingFactory(_ storage.VectorDBSettings) (EmbeddingModel, error) {
	return NewLocalHashEmbedding("", "default")
}

// DefaultRegistry is the global embedding registry instance.
var DefaultRegistry = NewEmbeddingRegistry()

// RegisterEmbedding registers an embedding factory with the default registry.
func RegisterEmbedding(name string, factory EmbeddingFactory) error {
	return DefaultRegistry.Register(name, factory)
}

// CreateEmbedding creates an embedding model from the default registry.
func CreateEmbedding(name string, cfg storage.VectorDBSettings) (EmbeddingModel, error) {
	return DefaultRegistry.Create(name, cfg)
}
