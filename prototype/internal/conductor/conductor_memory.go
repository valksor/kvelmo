package conductor

import (
	"context"
	"fmt"

	"github.com/valksor/go-mehrhof/internal/memory"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// MemorySystem holds the memory system components.
type MemorySystem struct {
	memory  *memory.MemorySystem
	indexer *memory.Indexer
	tool    *memory.MemoryTool
	config  *storage.MemorySettings
}

// InitializeMemory initializes the memory system from workspace config.
func (c *Conductor) InitializeMemory(ctx context.Context) error {
	cfg, err := c.workspace.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg == nil || cfg.Memory == nil || !cfg.Memory.Enabled {
		return nil // Memory disabled
	}

	// Create embedding model using registry
	modelName := cfg.Memory.VectorDB.EmbeddingModel
	if modelName == "" {
		modelName = "default"
	}

	model, err := memory.CreateEmbedding(modelName, cfg.Memory.VectorDB)
	if err != nil {
		return fmt.Errorf("create embedding model %q: %w", modelName, err)
	}

	// Wire up progress feedback for embedder client (if using ONNX)
	if embedClient, ok := model.(*memory.EmbedderClient); ok {
		embedClient.SetEventPublisher(func(message string) {
			c.publishProgress(message, -1)
		})
	}

	// Create vector store
	var store memory.VectorStore

	connectionString := cfg.Memory.VectorDB.ConnectionString
	if connectionString == "" {
		connectionString = "./.mehrhof/vectors"
	}

	collection := cfg.Memory.VectorDB.Collection
	if collection == "" {
		collection = "mehr_task_memory"
	}

	switch cfg.Memory.VectorDB.Backend {
	case "chromadb", "":
		// Note: "chromadb" config value is kept for compatibility, but ChromaDBStore is a file-based implementation
		store, err = memory.NewChromaDBStore(connectionString, collection, model)
		if err != nil {
			return fmt.Errorf("create vector store: %w", err)
		}
	default:
		return fmt.Errorf("unsupported vector DB backend: %s", cfg.Memory.VectorDB.Backend)
	}

	// Create memory system - this can't fail
	mem := memory.NewMemorySystem(store, model)

	// Create indexer for automatic task indexing
	indexer := memory.NewIndexer(mem, c.workspace, c.git)

	// Create memory tool for agent integration
	tool := memory.NewMemoryTool(mem, indexer)

	// Store in conductor
	c.memory = &MemorySystem{
		memory:  mem,
		indexer: indexer,
		tool:    tool,
		config:  cfg.Memory,
	}

	c.publishProgress(fmt.Sprintf("Memory system initialized (backend: %s, model: %s)",
		cfg.Memory.VectorDB.Backend, cfg.Memory.VectorDB.EmbeddingModel), 100)

	return nil
}

// GetMemoryTool returns the memory tool for agent integration.
func (c *Conductor) GetMemoryTool() *memory.MemoryTool {
	if c.memory == nil {
		return nil
	}

	return c.memory.tool
}

// GetMemory returns the memory system.
func (c *Conductor) GetMemory() *memory.MemorySystem {
	if c.memory == nil {
		return nil
	}

	return c.memory.memory
}

// IndexCompletedTask indexes a completed task into memory.
func (c *Conductor) IndexCompletedTask(ctx context.Context) error {
	if c.memory == nil || !c.memory.config.Learning.AutoStore {
		return nil // Memory disabled or auto-store disabled
	}

	task := c.GetActiveTask()
	if task == nil {
		return nil
	}

	c.publishProgress(fmt.Sprintf("Indexing task %s into memory", task.ID), 50)

	if err := c.memory.indexer.IndexTask(ctx, task.ID); err != nil {
		return fmt.Errorf("index task: %w", err)
	}

	c.publishProgress("Task indexed successfully", 100)

	return nil
}

// GetMemoryContextForTask retrieves relevant memory context for the current task.
func (c *Conductor) GetMemoryContextForTask(ctx context.Context) (string, error) {
	if c.memory == nil {
		return "", nil // Memory disabled
	}

	work := c.GetTaskWork()
	if work == nil {
		return "", nil
	}

	c.publishProgress("Searching semantic memory...", -1)

	// Use memory tool to augment prompt
	context, err := c.memory.tool.AugmentPrompt(ctx, work.Metadata.Title, work.Metadata.ExternalKey)
	if err != nil {
		return "", fmt.Errorf("augment prompt with memory: %w", err)
	}

	return context, nil
}

// LearnFromCorrection stores a correction/fix as a solution.
func (c *Conductor) LearnFromCorrection(ctx context.Context, problem, solution string) error {
	if c.memory == nil || !c.memory.config.Learning.LearnFromCorrections {
		return nil // Memory disabled or learning disabled
	}

	task := c.GetActiveTask()
	if task == nil {
		return nil
	}

	c.publishProgress("Storing correction in memory for task "+task.ID, 50)

	if err := c.memory.tool.LearnFromCorrection(ctx, task.ID, problem, solution); err != nil {
		return fmt.Errorf("learn from correction: %w", err)
	}

	c.publishProgress("Correction stored in memory", 100)

	return nil
}

// GetSimilarTasks finds similar past tasks for the current task.
func (c *Conductor) GetSimilarTasks(ctx context.Context, limit int) ([]string, error) {
	if c.memory == nil {
		return nil, nil // Memory disabled
	}

	work := c.GetTaskWork()
	if work == nil {
		return nil, nil
	}

	c.publishProgress("Finding similar past tasks...", -1)

	// Build query from task title and external key
	query := fmt.Sprintf("%s %s", work.Metadata.Title, work.Metadata.ExternalKey)

	// Search for similar tasks
	results, err := c.memory.tool.SearchSimilarTasks(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search similar tasks: %w", err)
	}

	return results, nil
}
