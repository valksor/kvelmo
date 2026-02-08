package memory

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/jsonrpc"
)

const (
	// embedderStopTimeout is the maximum time to wait for the embedder to stop gracefully.
	embedderStopTimeout = 10 * time.Second
	// embedderStartupTimeout is the maximum time to wait for the embedder to start.
	embedderStartupTimeout = 30 * time.Second
)

// EmbedderClient implements EmbeddingModel by communicating with mehr-embedder sidecar.
type EmbedderClient struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Reader
	pending   map[int64]chan *rpcResponse
	reqID     atomic.Int64
	dimension int
	model     string
	binPath   string

	// Process state
	started bool
	done    chan struct{}
	err     error
}

// rpcRequest is a JSON-RPC request.
type rpcRequest struct {
	Params  any    `json:"params,omitempty"`
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      int64  `json:"id"`
}

// rpcResponse is a JSON-RPC response.
type rpcResponse struct {
	Error   *jsonrpc.RPCError `json:"error,omitempty"`
	Result  json.RawMessage   `json:"result,omitempty"`
	JSONRPC string            `json:"jsonrpc"`
	ID      int64             `json:"id"`
}

// EmbedderClientOptions configures the embedder client.
type EmbedderClientOptions struct {
	BinPath   string // Path to mehr-embedder binary
	ModelName string // Model name to use (default: all-MiniLM-L6-v2)
	MaxLength int    // Max sequence length (default: 256)
}

// NewEmbedderClient creates a new embedder client.
// The embedder process is started lazily on first use.
func NewEmbedderClient(opts EmbedderClientOptions) *EmbedderClient {
	return &EmbedderClient{
		binPath: opts.BinPath,
		model:   opts.ModelName,
		pending: make(map[int64]chan *rpcResponse),
		done:    make(chan struct{}),
	}
}

// ensureRunning starts the embedder process if not already running.
func (c *EmbedderClient) ensureRunning(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return nil
	}

	// Find the embedder binary
	binPath := c.binPath
	if binPath == "" {
		// Look in common locations
		binPath = c.findEmbedderBinary()
		if binPath == "" {
			// Try to download it
			var err error
			binPath, err = c.downloadEmbedder(ctx)
			if err != nil {
				return fmt.Errorf("embedder not available: %w", err)
			}
		}
	}

	slog.Debug("starting embedder process", "path", binPath)

	c.cmd = exec.CommandContext(ctx, binPath)

	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("create stdin pipe: %w", err)
	}
	c.stdin = stdin

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()

		return fmt.Errorf("create stdout pipe: %w", err)
	}
	c.stdout = bufio.NewReader(stdout)

	// Stderr goes to slog
	c.cmd.Stderr = &slogWriter{level: slog.LevelDebug, prefix: "embedder"}

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start embedder: %w", err)
	}

	c.started = true

	// Start response reader
	go c.readResponses()

	// Initialize the embedder
	initCtx, cancel := context.WithTimeout(ctx, embedderStartupTimeout)
	defer cancel()

	initParams := map[string]any{}
	if c.model != "" {
		initParams["modelName"] = c.model
	}

	result, err := c.call(initCtx, "init", initParams)
	if err != nil {
		//nolint:contextcheck // Fresh context for cleanup; parent context may be cancelled
		_ = c.stopLocked(context.Background())

		return fmt.Errorf("init embedder: %w", err)
	}

	var initResult struct {
		Dimension int    `json:"dimension"`
		Model     string `json:"model"`
	}
	if err := json.Unmarshal(result, &initResult); err != nil {
		//nolint:contextcheck // Fresh context for cleanup; parent context may be cancelled
		_ = c.stopLocked(context.Background())

		return fmt.Errorf("parse init result: %w", err)
	}

	c.dimension = initResult.Dimension
	c.model = initResult.Model

	slog.Info("embedder started",
		"model", c.model,
		"dimension", c.dimension)

	return nil
}

// findEmbedderBinary looks for mehr-embedder in common locations.
func (c *EmbedderClient) findEmbedderBinary() string {
	// Check common locations
	candidates := []string{
		// Same directory as current executable
		filepath.Join(filepath.Dir(os.Args[0]), "mehr-embedder"),
		// Build directory
		"./build/mehr-embedder",
		// User's PATH
		"mehr-embedder",
	}

	// Add ~/.valksor/mehrhof/bin/ candidate if home dir is available
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, ".valksor", "mehrhof", "bin", "mehr-embedder"))
	}

	for _, path := range candidates {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}

	return ""
}

// downloadEmbedder downloads the embedder binary from GitHub releases.
func (c *EmbedderClient) downloadEmbedder(ctx context.Context) (string, error) {
	// Check if platform is supported
	if !IsEmbedderAvailable() {
		return "", fmt.Errorf("ONNX embedder not available for %s/%s; use hash embeddings instead",
			runtime.GOOS, runtime.GOARCH)
	}

	slog.Info("downloading ONNX embedder binary (this may take a moment)...")

	downloader, err := NewEmbedderDownloader("")
	if err != nil {
		return "", err
	}

	return downloader.EnsureEmbedder(ctx)
}

// readResponses reads JSON-RPC responses from stdout.
func (c *EmbedderClient) readResponses() {
	defer close(c.done)

	for {
		line, err := c.stdout.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				c.err = fmt.Errorf("read stdout: %w", err)
			}
			// Close all pending requests
			c.mu.Lock()
			for id, ch := range c.pending {
				close(ch)
				delete(c.pending, id)
			}
			c.mu.Unlock()

			return
		}

		var resp rpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue // Skip malformed lines
		}

		c.mu.Lock()
		if ch, ok := c.pending[resp.ID]; ok {
			ch <- &resp
			delete(c.pending, resp.ID)
		}
		c.mu.Unlock()
	}
}

// call sends a JSON-RPC request and waits for a response.
func (c *EmbedderClient) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.reqID.Add(1)
	ch := make(chan *rpcResponse, 1)

	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}

	data, err := json.Marshal(req)
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()

		return nil, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	c.mu.Lock()
	_, err = c.stdin.Write(data)
	c.mu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()

		return nil, fmt.Errorf("write request: %w", err)
	}

	select {
	case resp, ok := <-ch:
		if !ok {
			return nil, errors.New("embedder process closed")
		}
		if resp.Error != nil {
			return nil, resp.Error
		}

		return resp.Result, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()

		return nil, ctx.Err()
	}
}

// Embed generates an embedding for a single text.
func (c *EmbedderClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if err := c.ensureRunning(ctx); err != nil {
		return nil, err
	}

	result, err := c.call(ctx, "embed", map[string]string{"text": text})
	if err != nil {
		return nil, err
	}

	var embedResult struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.Unmarshal(result, &embedResult); err != nil {
		return nil, fmt.Errorf("parse embed result: %w", err)
	}

	return embedResult.Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts.
func (c *EmbedderClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if err := c.ensureRunning(ctx); err != nil {
		return nil, err
	}

	result, err := c.call(ctx, "embedBatch", map[string][]string{"texts": texts})
	if err != nil {
		return nil, err
	}

	var batchResult struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.Unmarshal(result, &batchResult); err != nil {
		return nil, fmt.Errorf("parse embedBatch result: %w", err)
	}

	return batchResult.Embeddings, nil
}

// Dimension returns the embedding dimension.
func (c *EmbedderClient) Dimension() int {
	return c.dimension
}

// Close stops the embedder process.
func (c *EmbedderClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.stopLocked(context.Background())
}

// stopLocked stops the embedder process (must hold mu).
func (c *EmbedderClient) stopLocked(ctx context.Context) error {
	if !c.started {
		return nil
	}

	// Send shutdown request
	shutdownCtx, cancel := context.WithTimeout(ctx, embedderStopTimeout)
	defer cancel()

	// Temporarily release lock for shutdown call
	c.mu.Unlock()
	_, _ = c.call(shutdownCtx, "shutdown", nil)
	c.mu.Lock()

	// Close stdin
	if c.stdin != nil {
		_ = c.stdin.Close()
	}

	// Wait for process
	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	timer := time.NewTimer(embedderStopTimeout)
	defer timer.Stop()

	select {
	case err := <-done:
		c.started = false

		return err
	case <-timer.C:
		_ = c.cmd.Process.Kill()
		c.started = false

		return <-done
	}
}

// slogWriter adapts slog for use as an io.Writer.
type slogWriter struct {
	level  slog.Level
	prefix string
}

func (w *slogWriter) Write(p []byte) (int, error) {
	slog.Log(context.Background(), w.level, string(p), "source", w.prefix)

	return len(p), nil
}

// NewEmbedderClientFromConfig creates an EmbedderClient from config settings.
func NewEmbedderClientFromConfig(cfg storage.VectorDBSettings) (EmbeddingModel, error) {
	opts := EmbedderClientOptions{
		ModelName: cfg.ONNX.Model,
	}

	client := NewEmbedderClient(opts)

	// Lazy initialization - don't start the process yet
	return client, nil
}
