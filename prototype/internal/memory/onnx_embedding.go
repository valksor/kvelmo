//go:build cgo

package memory

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/yalue/onnxruntime_go"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// ONNXEmbedding implements EmbeddingModel using ONNX Runtime.
type ONNXEmbedding struct {
	mu        sync.Mutex
	session   *onnxruntime_go.DynamicAdvancedSession
	tokenizer *Tokenizer
	dimension int
	modelPath string
	maxLength int

	// Tensor shapes
	inputShape  onnxruntime_go.Shape
	outputShape onnxruntime_go.Shape

	// Session state
	initialized bool
}

// ONNXEmbeddingOptions configures the ONNX embedding model.
type ONNXEmbeddingOptions struct {
	ModelName string // Model name from KnownModels (default: "all-MiniLM-L6-v2")
	CachePath string // Custom cache path (default: ~/.valksor/mehrhof/models/)
	MaxLength int    // Max sequence length (default: 256)
}

// NewONNXEmbedding creates an ONNX embedding model.
// The model is loaded lazily on first use.
func NewONNXEmbedding(opts ONNXEmbeddingOptions) (*ONNXEmbedding, error) {
	if opts.ModelName == "" {
		opts.ModelName = "all-MiniLM-L6-v2"
	}

	if opts.MaxLength <= 0 {
		opts.MaxLength = 256
	}

	modelInfo, err := GetModelInfo(opts.ModelName)
	if err != nil {
		return nil, fmt.Errorf("get model info: %w", err)
	}

	return &ONNXEmbedding{
		dimension:   modelInfo.Dimension,
		maxLength:   opts.MaxLength,
		inputShape:  onnxruntime_go.NewShape(1, int64(opts.MaxLength)),
		outputShape: onnxruntime_go.NewShape(1, int64(opts.MaxLength), int64(modelInfo.Dimension)),
	}, nil
}

// NewONNXEmbeddingFromConfig creates an ONNX embedding from config settings.
func NewONNXEmbeddingFromConfig(cfg storage.VectorDBSettings) (EmbeddingModel, error) {
	opts := ONNXEmbeddingOptions{
		ModelName: cfg.ONNX.Model,
		CachePath: cfg.ONNX.CachePath,
		MaxLength: cfg.ONNX.MaxLength,
	}

	return NewONNXEmbedding(opts)
}

// ensureInitialized lazily initializes the model on first use.
func (e *ONNXEmbedding) ensureInitialized(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.initialized {
		return nil
	}

	// Get model info
	modelName := "all-MiniLM-L6-v2" // Default
	modelInfo, err := GetModelInfo(modelName)
	if err != nil {
		return fmt.Errorf("get model info: %w", err)
	}

	// Download model if needed
	downloader, err := NewModelDownloader("")
	if err != nil {
		return fmt.Errorf("create downloader: %w", err)
	}

	slog.Info("ensuring ONNX model is available", "model", modelName)

	modelPath, err := downloader.EnsureModel(ctx, modelInfo, func(p DownloadProgress) {
		slog.Debug("downloading model",
			"file", p.File,
			"progress", fmt.Sprintf("%.1f%%", p.Percent))
	})
	if err != nil {
		return fmt.Errorf("ensure model: %w", err)
	}

	e.modelPath = modelPath

	// Initialize ONNX Runtime
	if err := initONNXRuntime(); err != nil {
		return fmt.Errorf("init ONNX runtime: %w", err)
	}

	// Load tokenizer
	e.tokenizer, err = NewTokenizer(modelPath, e.maxLength)
	if err != nil {
		return fmt.Errorf("load tokenizer: %w", err)
	}

	// Create session
	onnxPath := filepath.Join(modelPath, "model.onnx")

	// Define input/output for dynamic session
	inputs := []string{"input_ids", "attention_mask", "token_type_ids"}
	outputs := []string{"last_hidden_state"}

	session, err := onnxruntime_go.NewDynamicAdvancedSession(
		onnxPath,
		inputs,
		outputs,
		nil, // Use default session options
	)
	if err != nil {
		return fmt.Errorf("create ONNX session: %w", err)
	}

	e.session = session
	e.initialized = true

	slog.Info("ONNX embedding model initialized",
		"model", modelName,
		"dimension", e.dimension,
		"maxLength", e.maxLength)

	return nil
}

// Embed generates an embedding for a single text.
func (e *ONNXEmbedding) Embed(ctx context.Context, text string) ([]float32, error) {
	if err := e.ensureInitialized(ctx); err != nil {
		return nil, err
	}

	embeddings, err := e.embedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts.
func (e *ONNXEmbedding) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if err := e.ensureInitialized(ctx); err != nil {
		return nil, err
	}

	return e.embedBatch(ctx, texts)
}

// embedBatch is the internal batch embedding implementation.
func (e *ONNXEmbedding) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	results := make([][]float32, len(texts))

	// Process one at a time to avoid batch dimension complexity
	for i, text := range texts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		embedding, err := e.embedSingle(text)
		if err != nil {
			return nil, fmt.Errorf("embed text %d: %w", i, err)
		}

		results[i] = embedding
	}

	return results, nil
}

// embedSingle embeds a single text (must hold mu lock).
func (e *ONNXEmbedding) embedSingle(text string) ([]float32, error) {
	// Tokenize
	tokens := e.tokenizer.Encode(text)

	// Create input tensors
	inputIDs, err := onnxruntime_go.NewTensor(e.inputShape, tokens.InputIDs)
	if err != nil {
		return nil, fmt.Errorf("create input_ids tensor: %w", err)
	}
	defer func() { _ = inputIDs.Destroy() }()

	attentionMask, err := onnxruntime_go.NewTensor(e.inputShape, tokens.AttentionMask)
	if err != nil {
		return nil, fmt.Errorf("create attention_mask tensor: %w", err)
	}
	defer func() { _ = attentionMask.Destroy() }()

	tokenTypeIDs, err := onnxruntime_go.NewTensor(e.inputShape, tokens.TokenTypeIDs)
	if err != nil {
		return nil, fmt.Errorf("create token_type_ids tensor: %w", err)
	}
	defer func() { _ = tokenTypeIDs.Destroy() }()

	// Create output tensor
	outputTensor, err := onnxruntime_go.NewEmptyTensor[float32](e.outputShape)
	if err != nil {
		return nil, fmt.Errorf("create output tensor: %w", err)
	}
	defer func() { _ = outputTensor.Destroy() }()

	// Run inference
	err = e.session.Run(
		[]onnxruntime_go.ArbitraryTensor{inputIDs, attentionMask, tokenTypeIDs},
		[]onnxruntime_go.ArbitraryTensor{outputTensor},
	)
	if err != nil {
		return nil, fmt.Errorf("run inference: %w", err)
	}

	// Get output data
	output := outputTensor.GetData()

	// Mean pooling: average over non-padded tokens
	embedding := meanPooling(output, tokens.AttentionMask, e.maxLength, e.dimension)

	// L2 normalize
	embedding = l2Normalize(embedding)

	return embedding, nil
}

// Dimension returns the embedding dimension.
func (e *ONNXEmbedding) Dimension() int {
	return e.dimension
}

// Close releases ONNX session resources.
func (e *ONNXEmbedding) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.session != nil {
		if err := e.session.Destroy(); err != nil {
			return err
		}

		e.session = nil
	}

	e.initialized = false

	return nil
}

// meanPooling computes mean of token embeddings, weighted by attention mask.
func meanPooling(output []float32, attentionMask []int64, seqLen, hiddenDim int) []float32 {
	embedding := make([]float32, hiddenDim)

	// Count non-padded tokens
	var tokenCount float32

	for i := range seqLen {
		if attentionMask[i] == 1 {
			tokenCount++

			// Add this token's embedding
			for j := range hiddenDim {
				embedding[j] += output[i*hiddenDim+j]
			}
		}
	}

	// Average
	if tokenCount > 0 {
		for j := range hiddenDim {
			embedding[j] /= tokenCount
		}
	}

	return embedding
}

// l2Normalize normalizes a vector to unit length.
func l2Normalize(vec []float32) []float32 {
	var norm float32

	for _, v := range vec {
		norm += v * v
	}

	norm = float32(math.Sqrt(float64(norm)))

	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}

	return vec
}

// ONNX Runtime initialization (singleton).
var (
	ortInitOnce sync.Once
	errOrtInit  error
)

// initONNXRuntime initializes the ONNX Runtime library (once globally).
func initONNXRuntime() error {
	ortInitOnce.Do(func() {
		// Check if shared library is available
		errOrtInit = onnxruntime_go.InitializeEnvironment()
		if errOrtInit != nil {
			// Try to provide helpful error message
			errOrtInit = fmt.Errorf("ONNX Runtime initialization failed: %w. "+
				"Make sure libonnxruntime is installed (Linux: apt install libonnxruntime, "+
				"macOS: brew install onnxruntime, or set ORT_DYLIB_PATH)", errOrtInit)
		}
	})

	return errOrtInit
}

// IsONNXAvailable checks if ONNX Runtime is available on this system.
func IsONNXAvailable() bool {
	err := initONNXRuntime()

	return err == nil
}

// GetONNXError returns any ONNX initialization error.
func GetONNXError() error {
	return errOrtInit
}

// RegisterONNXEmbedding registers the ONNX embedding provider with the registry.
func RegisterONNXEmbedding() error {
	return RegisterEmbedding("onnx", NewONNXEmbeddingFromConfig)
}

func init() {
	// Check if ONNX runtime shared library exists before registering
	// This prevents errors when ONNX is not installed
	if ortLibPath := os.Getenv("ORT_DYLIB_PATH"); ortLibPath != "" {
		if _, err := os.Stat(ortLibPath); err == nil {
			_ = RegisterONNXEmbedding()
		}
	}
}
