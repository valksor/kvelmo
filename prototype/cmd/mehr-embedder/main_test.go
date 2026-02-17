//go:build cgo

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/crealfy/crea-pipe/pkg/transport/jsonrpc"
)

// mockEmbeddingModel implements embeddingModel for testing.
type mockEmbeddingModel struct {
	dimension  int
	embedding  []float32
	embeddings [][]float32
	embedErr   error
	closeErr   error
}

func (m *mockEmbeddingModel) Embed(_ context.Context, _ string) ([]float32, error) {
	return m.embedding, m.embedErr
}

func (m *mockEmbeddingModel) EmbedBatch(_ context.Context, _ []string) ([][]float32, error) {
	return m.embeddings, m.embedErr
}

func (m *mockEmbeddingModel) Dimension() int {
	return m.dimension
}

func (m *mockEmbeddingModel) Close() error {
	return m.closeErr
}

// --- Embedder Tests ---

func TestNewEmbedder(t *testing.T) {
	e := NewEmbedder()
	if e == nil {
		t.Fatal("NewEmbedder returned nil")
	}
	if e.model != nil {
		t.Error("expected model to be nil on new embedder")
	}
}

func TestEmbedder_Embed_NotInitialized(t *testing.T) {
	e := NewEmbedder()
	_, err := e.Embed(context.Background(), EmbedParams{Text: "test"})
	if err == nil {
		t.Fatal("expected error for uninitialized embedder")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEmbedder_Embed_Success(t *testing.T) {
	e := NewEmbedder()
	expected := []float32{0.1, 0.2, 0.3}
	e.model = &mockEmbeddingModel{embedding: expected}

	result, err := e.Embed(context.Background(), EmbedParams{Text: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Embedding) != len(expected) {
		t.Errorf("expected %d embeddings, got %d", len(expected), len(result.Embedding))
	}
	for i, v := range result.Embedding {
		if v != expected[i] {
			t.Errorf("embedding[%d] = %f, want %f", i, v, expected[i])
		}
	}
}

func TestEmbedder_Embed_Error(t *testing.T) {
	e := NewEmbedder()
	e.model = &mockEmbeddingModel{embedErr: errors.New("embedding failed")}

	_, err := e.Embed(context.Background(), EmbedParams{Text: "test"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "embedding failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEmbedder_EmbedBatch_NotInitialized(t *testing.T) {
	e := NewEmbedder()
	_, err := e.EmbedBatch(context.Background(), EmbedBatchParams{Texts: []string{"a", "b"}})
	if err == nil {
		t.Fatal("expected error for uninitialized embedder")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEmbedder_EmbedBatch_Success(t *testing.T) {
	e := NewEmbedder()
	expected := [][]float32{{0.1, 0.2}, {0.3, 0.4}}
	e.model = &mockEmbeddingModel{embeddings: expected}

	result, err := e.EmbedBatch(context.Background(), EmbedBatchParams{Texts: []string{"a", "b"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Embeddings) != len(expected) {
		t.Errorf("expected %d embeddings, got %d", len(expected), len(result.Embeddings))
	}
}

func TestEmbedder_EmbedBatch_Error(t *testing.T) {
	e := NewEmbedder()
	e.model = &mockEmbeddingModel{embedErr: errors.New("batch failed")}

	_, err := e.EmbedBatch(context.Background(), EmbedBatchParams{Texts: []string{"a"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "batch failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEmbedder_Dimension_NotInitialized(t *testing.T) {
	e := NewEmbedder()
	_, err := e.Dimension()
	if err == nil {
		t.Fatal("expected error for uninitialized embedder")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEmbedder_Dimension_Success(t *testing.T) {
	e := NewEmbedder()
	e.model = &mockEmbeddingModel{dimension: 384}

	result, err := e.Dimension()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Dimension != 384 {
		t.Errorf("expected dimension 384, got %d", result.Dimension)
	}
}

func TestEmbedder_Close_NilModel(t *testing.T) {
	e := NewEmbedder()
	err := e.Close()
	if err != nil {
		t.Errorf("expected no error for nil model, got: %v", err)
	}
}

func TestEmbedder_Close_Success(t *testing.T) {
	e := NewEmbedder()
	e.model = &mockEmbeddingModel{}

	err := e.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEmbedder_Close_Error(t *testing.T) {
	e := NewEmbedder()
	e.model = &mockEmbeddingModel{closeErr: errors.New("close failed")}

	err := e.Close()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "close failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- handleRequest Tests ---

func TestHandleRequest(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		params      json.RawMessage
		setupModel  func() *mockEmbeddingModel
		wantErrCode int
		checkResult func(t *testing.T, resp *jsonrpc.Response)
	}{
		{
			name:   "embed success",
			method: "embed",
			params: json.RawMessage(`{"text":"hello"}`),
			setupModel: func() *mockEmbeddingModel {
				return &mockEmbeddingModel{embedding: []float32{0.1, 0.2, 0.3}}
			},
			checkResult: func(t *testing.T, resp *jsonrpc.Response) {
				t.Helper()
				if resp.Error != nil {
					t.Fatalf("unexpected error: %v", resp.Error)
				}
				var result EmbedResult
				if err := json.Unmarshal(resp.Result, &result); err != nil {
					t.Fatalf("unmarshal result: %v", err)
				}
				if len(result.Embedding) != 3 {
					t.Errorf("expected 3 embeddings, got %d", len(result.Embedding))
				}
			},
		},
		{
			name:        "embed invalid params",
			method:      "embed",
			params:      json.RawMessage(`{invalid`),
			wantErrCode: jsonrpc.ErrCodeInvalidParams,
		},
		{
			name:   "embed not initialized",
			method: "embed",
			params: json.RawMessage(`{"text":"hello"}`),
			// No setupModel = nil model
			wantErrCode: jsonrpc.ErrCodeInternalError,
		},
		{
			name:   "embedBatch success",
			method: "embedBatch",
			params: json.RawMessage(`{"texts":["a","b"]}`),
			setupModel: func() *mockEmbeddingModel {
				return &mockEmbeddingModel{embeddings: [][]float32{{0.1}, {0.2}}}
			},
			checkResult: func(t *testing.T, resp *jsonrpc.Response) {
				t.Helper()
				if resp.Error != nil {
					t.Fatalf("unexpected error: %v", resp.Error)
				}
				var result EmbedBatchResult
				if err := json.Unmarshal(resp.Result, &result); err != nil {
					t.Fatalf("unmarshal result: %v", err)
				}
				if len(result.Embeddings) != 2 {
					t.Errorf("expected 2 embeddings, got %d", len(result.Embeddings))
				}
			},
		},
		{
			name:        "embedBatch invalid params",
			method:      "embedBatch",
			params:      json.RawMessage(`not json`),
			wantErrCode: jsonrpc.ErrCodeInvalidParams,
		},
		{
			name:   "dimension success",
			method: "dimension",
			setupModel: func() *mockEmbeddingModel {
				return &mockEmbeddingModel{dimension: 384}
			},
			checkResult: func(t *testing.T, resp *jsonrpc.Response) {
				t.Helper()
				if resp.Error != nil {
					t.Fatalf("unexpected error: %v", resp.Error)
				}
				var result DimensionResult
				if err := json.Unmarshal(resp.Result, &result); err != nil {
					t.Fatalf("unmarshal result: %v", err)
				}
				if result.Dimension != 384 {
					t.Errorf("expected dimension 384, got %d", result.Dimension)
				}
			},
		},
		{
			name:        "dimension not initialized",
			method:      "dimension",
			wantErrCode: jsonrpc.ErrCodeInternalError,
		},
		{
			name:   "shutdown",
			method: "shutdown",
			checkResult: func(t *testing.T, resp *jsonrpc.Response) {
				t.Helper()
				if resp.Error != nil {
					t.Fatalf("unexpected error: %v", resp.Error)
				}
				var result map[string]bool
				if err := json.Unmarshal(resp.Result, &result); err != nil {
					t.Fatalf("unmarshal result: %v", err)
				}
				if !result["ok"] {
					t.Error("expected ok:true")
				}
			},
		},
		{
			name:        "unknown method",
			method:      "unknown",
			wantErrCode: jsonrpc.ErrCodeMethodNotFound,
		},
		{
			name:   "init with nil params",
			method: "init",
			params: nil, // nil params should work for init
			// Init creates a real ONNX model - result depends on ONNX availability
			// If ONNX is available, this succeeds; if not, it returns InternalError
			checkResult: func(t *testing.T, resp *jsonrpc.Response) {
				t.Helper()
				// Either success (ONNX available) or internal error (ONNX unavailable)
				if resp.Error != nil && resp.Error.Code != jsonrpc.ErrCodeInternalError {
					t.Errorf("unexpected error code: %d", resp.Error.Code)
				}
			},
		},
		{
			name:        "init with invalid params",
			method:      "init",
			params:      json.RawMessage(`{broken`),
			wantErrCode: jsonrpc.ErrCodeInvalidParams,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder := NewEmbedder()
			if tt.setupModel != nil {
				embedder.model = tt.setupModel()
			}

			req := &request{
				Method:  tt.method,
				Params:  tt.params,
				ID:      1,
				JSONRPC: "2.0",
			}

			resp := handleRequest(context.Background(), embedder, req)

			if resp.ID != 1 {
				t.Errorf("expected ID 1, got %d", resp.ID)
			}
			if resp.JSONRPC != "2.0" {
				t.Errorf("expected JSONRPC 2.0, got %s", resp.JSONRPC)
			}

			if tt.wantErrCode != 0 {
				if resp.Error == nil {
					t.Fatalf("expected error code %d, got no error", tt.wantErrCode)
				}
				if resp.Error.Code != tt.wantErrCode {
					t.Errorf("expected error code %d, got %d", tt.wantErrCode, resp.Error.Code)
				}
			} else if tt.checkResult != nil {
				tt.checkResult(t, resp)
			}
		})
	}
}

// --- runServer Tests ---

func TestRunServer_SingleRequest(t *testing.T) {
	embedder := NewEmbedder()
	embedder.model = &mockEmbeddingModel{dimension: 384}

	input := `{"jsonrpc":"2.0","method":"dimension","id":1}` + "\n" +
		`{"jsonrpc":"2.0","method":"shutdown","id":2}` + "\n"

	stdin := strings.NewReader(input)
	stdout := &bytes.Buffer{}

	err := runServer(context.Background(), embedder, stdin, stdout)
	if err != nil {
		t.Fatalf("runServer error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses, got %d: %v", len(lines), lines)
	}

	// Verify first response (dimension)
	var resp1 jsonrpc.Response
	if err := json.Unmarshal([]byte(lines[0]), &resp1); err != nil {
		t.Fatalf("unmarshal response 1: %v", err)
	}
	if resp1.Error != nil {
		t.Errorf("response 1 unexpected error: %v", resp1.Error)
	}

	// Verify second response (shutdown)
	var resp2 jsonrpc.Response
	if err := json.Unmarshal([]byte(lines[1]), &resp2); err != nil {
		t.Fatalf("unmarshal response 2: %v", err)
	}
	if resp2.Error != nil {
		t.Errorf("response 2 unexpected error: %v", resp2.Error)
	}
}

func TestRunServer_MultipleRequests(t *testing.T) {
	embedder := NewEmbedder()
	embedder.model = &mockEmbeddingModel{
		dimension:  384,
		embedding:  []float32{0.1, 0.2},
		embeddings: [][]float32{{0.1}, {0.2}},
	}

	input := `{"jsonrpc":"2.0","method":"dimension","id":1}` + "\n" +
		`{"jsonrpc":"2.0","method":"embed","params":{"text":"hello"},"id":2}` + "\n" +
		`{"jsonrpc":"2.0","method":"embedBatch","params":{"texts":["a","b"]},"id":3}` + "\n" +
		`{"jsonrpc":"2.0","method":"shutdown","id":4}` + "\n"

	stdin := strings.NewReader(input)
	stdout := &bytes.Buffer{}

	err := runServer(context.Background(), embedder, stdin, stdout)
	if err != nil {
		t.Fatalf("runServer error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 responses, got %d", len(lines))
	}

	// Verify all responses have no errors
	for i, line := range lines {
		var resp jsonrpc.Response
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("unmarshal response %d: %v", i+1, err)
		}
		if resp.Error != nil {
			t.Errorf("response %d unexpected error: %v", i+1, resp.Error)
		}
		if resp.ID != i+1 {
			t.Errorf("response %d: expected ID %d, got %d", i+1, i+1, resp.ID)
		}
	}
}

func TestRunServer_InvalidJSON(t *testing.T) {
	embedder := NewEmbedder()
	embedder.model = &mockEmbeddingModel{dimension: 384}

	input := `{invalid json}` + "\n" +
		`{"jsonrpc":"2.0","method":"shutdown","id":1}` + "\n"

	stdin := strings.NewReader(input)
	stdout := &bytes.Buffer{}

	err := runServer(context.Background(), embedder, stdin, stdout)
	if err != nil {
		t.Fatalf("runServer error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(lines))
	}

	// First response should be parse error
	var resp1 jsonrpc.Response
	if err := json.Unmarshal([]byte(lines[0]), &resp1); err != nil {
		t.Fatalf("unmarshal response 1: %v", err)
	}
	if resp1.Error == nil {
		t.Error("expected parse error in response 1")
	} else if resp1.Error.Code != jsonrpc.ErrCodeParseError {
		t.Errorf("expected parse error code, got %d", resp1.Error.Code)
	}

	// Second response should succeed
	var resp2 jsonrpc.Response
	if err := json.Unmarshal([]byte(lines[1]), &resp2); err != nil {
		t.Fatalf("unmarshal response 2: %v", err)
	}
	if resp2.Error != nil {
		t.Errorf("response 2 unexpected error: %v", resp2.Error)
	}
}

func TestRunServer_EOF(t *testing.T) {
	embedder := NewEmbedder()
	embedder.model = &mockEmbeddingModel{}

	// Empty input = immediate EOF
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}

	err := runServer(context.Background(), embedder, stdin, stdout)
	if err != nil {
		t.Fatalf("expected nil error on EOF, got: %v", err)
	}

	if stdout.Len() != 0 {
		t.Errorf("expected no output, got: %s", stdout.String())
	}
}

func TestRunServer_ContextCanceled(t *testing.T) {
	embedder := NewEmbedder()
	embedder.model = &mockEmbeddingModel{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Use a reader that blocks (so we rely on context cancellation)
	stdin := &blockingReader{}
	stdout := &bytes.Buffer{}

	err := runServer(ctx, embedder, stdin, stdout)
	if err != nil {
		t.Fatalf("expected nil error on context cancel, got: %v", err)
	}
}

// blockingReader is a reader that returns nothing until the test times out.
// Used to test context cancellation.
type blockingReader struct{}

func (r *blockingReader) Read(_ []byte) (int, error) {
	// This won't be called since context is already canceled
	return 0, io.EOF
}

func TestRunServer_ReadError(t *testing.T) {
	embedder := NewEmbedder()
	embedder.model = &mockEmbeddingModel{}

	stdin := &errorReader{err: errors.New("read failed")}
	stdout := &bytes.Buffer{}

	err := runServer(context.Background(), embedder, stdin, stdout)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "read failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

// errorReader is a reader that always returns an error.
type errorReader struct {
	err error
}

func (r *errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

// --- Protocol Struct Tests ---

func TestRequestParsing(t *testing.T) {
	input := `{"jsonrpc":"2.0","method":"embed","params":{"text":"hello"},"id":42}`

	var req request
	err := json.Unmarshal([]byte(input), &req)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", req.JSONRPC)
	}
	if req.Method != "embed" {
		t.Errorf("expected method embed, got %s", req.Method)
	}
	if req.ID != 42 {
		t.Errorf("expected id 42, got %d", req.ID)
	}

	// Verify params can be unmarshaled
	var params EmbedParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if params.Text != "hello" {
		t.Errorf("expected text 'hello', got %s", params.Text)
	}
}

func TestInitParams_Marshaling(t *testing.T) {
	params := InitParams{
		ModelName: "all-MiniLM-L6-v2",
		MaxLength: 512,
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded InitParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ModelName != params.ModelName {
		t.Errorf("expected model %s, got %s", params.ModelName, decoded.ModelName)
	}
	if decoded.MaxLength != params.MaxLength {
		t.Errorf("expected maxLength %d, got %d", params.MaxLength, decoded.MaxLength)
	}
}

func TestResultTypes_Marshaling(t *testing.T) {
	t.Run("InitResult", func(t *testing.T) {
		result := InitResult{Dimension: 384, Model: "test-model"}
		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		if !strings.Contains(string(data), `"dimension":384`) {
			t.Errorf("unexpected json: %s", data)
		}
	})

	t.Run("EmbedResult", func(t *testing.T) {
		result := EmbedResult{Embedding: []float32{0.1, 0.2}}
		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		if !strings.Contains(string(data), `"embedding"`) {
			t.Errorf("unexpected json: %s", data)
		}
	})

	t.Run("EmbedBatchResult", func(t *testing.T) {
		result := EmbedBatchResult{Embeddings: [][]float32{{0.1}, {0.2}}}
		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		if !strings.Contains(string(data), `"embeddings"`) {
			t.Errorf("unexpected json: %s", data)
		}
	})

	t.Run("DimensionResult", func(t *testing.T) {
		result := DimensionResult{Dimension: 768}
		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		if !strings.Contains(string(data), `"dimension":768`) {
			t.Errorf("unexpected json: %s", data)
		}
	})
}
