//go:build cgo

// mehr-embedder is a sidecar binary that provides ONNX embedding capabilities
// over JSON-RPC via stdio. It is downloaded and spawned by mehr when ONNX
// embeddings are requested.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/crealfy/crea-pipe/pkg/transport/jsonrpc"
	"github.com/valksor/go-mehrhof/internal/memory"
)

// Protocol types for the embedder.

// embeddingModel defines the interface for embedding models.
// This is satisfied by *memory.ONNXEmbedding but allows mocking in tests.
type embeddingModel interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
	Close() error
}

// InitParams contains parameters for the init method.
type InitParams struct {
	ModelName string `json:"modelName,omitempty"`
	MaxLength int    `json:"maxLength,omitempty"`
}

// InitResult contains the result of the init method.
type InitResult struct {
	Dimension int    `json:"dimension"`
	Model     string `json:"model"`
}

// EmbedParams contains parameters for the embed method.
type EmbedParams struct {
	Text string `json:"text"`
}

// EmbedResult contains the result of the embed method.
type EmbedResult struct {
	Embedding []float32 `json:"embedding"`
}

// EmbedBatchParams contains parameters for the embedBatch method.
type EmbedBatchParams struct {
	Texts []string `json:"texts"`
}

// EmbedBatchResult contains the result of the embedBatch method.
type EmbedBatchResult struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// DimensionResult contains the result of the dimension method.
type DimensionResult struct {
	Dimension int `json:"dimension"`
}

// request is a JSON-RPC request with raw params for later unmarshaling.
type request struct {
	Params  json.RawMessage `json:"params,omitempty"`
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	ID      int             `json:"id"`
}

// Embedder wraps the ONNX embedding model with JSON-RPC handlers.
type Embedder struct {
	model embeddingModel
}

// NewEmbedder creates a new embedder instance.
func NewEmbedder() *Embedder {
	return &Embedder{}
}

// Init initializes the ONNX embedding model.
func (e *Embedder) Init(_ context.Context, params InitParams) (*InitResult, error) {
	opts := memory.ONNXEmbeddingOptions{
		ModelName: params.ModelName,
		MaxLength: params.MaxLength,
	}

	model, err := memory.NewONNXEmbedding(opts)
	if err != nil {
		return nil, fmt.Errorf("create ONNX embedding: %w", err)
	}

	e.model = model

	// Get model info for the response
	modelName := opts.ModelName
	if modelName == "" {
		modelName = "all-MiniLM-L6-v2"
	}

	return &InitResult{
		Dimension: model.Dimension(),
		Model:     modelName,
	}, nil
}

// Embed generates an embedding for a single text.
func (e *Embedder) Embed(ctx context.Context, params EmbedParams) (*EmbedResult, error) {
	if e.model == nil {
		return nil, errors.New("embedder not initialized")
	}

	embedding, err := e.model.Embed(ctx, params.Text)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}

	return &EmbedResult{Embedding: embedding}, nil
}

// EmbedBatch generates embeddings for multiple texts.
func (e *Embedder) EmbedBatch(ctx context.Context, params EmbedBatchParams) (*EmbedBatchResult, error) {
	if e.model == nil {
		return nil, errors.New("embedder not initialized")
	}

	embeddings, err := e.model.EmbedBatch(ctx, params.Texts)
	if err != nil {
		return nil, fmt.Errorf("embed batch: %w", err)
	}

	return &EmbedBatchResult{Embeddings: embeddings}, nil
}

// Dimension returns the embedding dimension.
func (e *Embedder) Dimension() (*DimensionResult, error) {
	if e.model == nil {
		return nil, errors.New("embedder not initialized")
	}

	return &DimensionResult{Dimension: e.model.Dimension()}, nil
}

// Close releases resources.
func (e *Embedder) Close() error {
	if e.model != nil {
		return e.model.Close()
	}

	return nil
}

func main() {
	exitCode := run()
	os.Exit(exitCode)
}

func run() int {
	// Set up logging to stderr (stdout is for JSON-RPC)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("mehr-embedder starting")

	// Handle shutdown signals
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	embedder := NewEmbedder()
	defer func() {
		if err := embedder.Close(); err != nil {
			slog.Error("failed to close embedder", "error", err)
		}
	}()

	// Run the JSON-RPC server
	if err := runServer(ctx, embedder, os.Stdin, os.Stdout); err != nil {
		slog.Error("server error", "error", err)

		return 1
	}

	slog.Info("mehr-embedder stopped")

	return 0
}

// runServer runs the JSON-RPC server over stdio.
func runServer(ctx context.Context, embedder *Embedder, r io.Reader, w io.Writer) error {
	reader := bufio.NewReader(r)
	encoder := json.NewEncoder(w)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Read a line from stdin
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return fmt.Errorf("read stdin: %w", err)
		}

		// Parse the request
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := jsonrpc.Response{
				JSONRPC: "2.0",
				Error: &jsonrpc.RPCError{
					Code:    jsonrpc.ErrCodeParseError,
					Message: "Parse error",
				},
			}
			if encErr := encoder.Encode(resp); encErr != nil {
				slog.Error("failed to encode error response", "error", encErr)
			}

			continue
		}

		// Handle the request
		resp := handleRequest(ctx, embedder, &req)
		if err := encoder.Encode(resp); err != nil {
			slog.Error("failed to encode response", "error", err)
		}

		// Check for shutdown
		if req.Method == "shutdown" {
			return nil
		}
	}
}

// handleRequest dispatches a JSON-RPC request to the appropriate handler.
func handleRequest(ctx context.Context, embedder *Embedder, req *request) *jsonrpc.Response {
	resp := &jsonrpc.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	var result any
	var err error

	switch req.Method {
	case "init":
		var params InitParams
		if req.Params != nil {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				resp.Error = &jsonrpc.RPCError{
					Code:    jsonrpc.ErrCodeInvalidParams,
					Message: fmt.Sprintf("invalid params: %v", err),
				}

				return resp
			}
		}
		result, err = embedder.Init(ctx, params)

	case "embed":
		var params EmbedParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &jsonrpc.RPCError{
				Code:    jsonrpc.ErrCodeInvalidParams,
				Message: fmt.Sprintf("invalid params: %v", err),
			}

			return resp
		}
		result, err = embedder.Embed(ctx, params)

	case "embedBatch":
		var params EmbedBatchParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &jsonrpc.RPCError{
				Code:    jsonrpc.ErrCodeInvalidParams,
				Message: fmt.Sprintf("invalid params: %v", err),
			}

			return resp
		}
		result, err = embedder.EmbedBatch(ctx, params)

	case "dimension":
		result, err = embedder.Dimension()

	case "shutdown":
		// Acknowledged, server will exit after sending response
		result = map[string]bool{"ok": true}

	default:
		resp.Error = &jsonrpc.RPCError{
			Code:    jsonrpc.ErrCodeMethodNotFound,
			Message: "method not found: " + req.Method,
		}

		return resp
	}

	if err != nil {
		resp.Error = &jsonrpc.RPCError{
			Code:    jsonrpc.ErrCodeInternalError,
			Message: err.Error(),
		}

		return resp
	}

	// Marshal the result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		resp.Error = &jsonrpc.RPCError{
			Code:    jsonrpc.ErrCodeInternalError,
			Message: fmt.Sprintf("marshal result: %v", err),
		}

		return resp
	}

	resp.Result = resultJSON

	return resp
}
