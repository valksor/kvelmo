package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewModelDownloader(t *testing.T) {
	tmpDir := t.TempDir()

	d, err := NewModelDownloader(tmpDir)
	if err != nil {
		t.Fatalf("NewModelDownloader: %v", err)
	}

	if d.cacheDir != tmpDir {
		t.Errorf("cacheDir: got %s, want %s", d.cacheDir, tmpDir)
	}
}

func TestNewModelDownloader_DefaultDir(t *testing.T) {
	d, err := NewModelDownloader("")
	if err != nil {
		t.Fatalf("NewModelDownloader: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".valksor", "mehrhof", "models")

	if d.cacheDir != expected {
		t.Errorf("cacheDir: got %s, want %s", d.cacheDir, expected)
	}
}

func TestModelDownloader_GetModelPath(t *testing.T) {
	tmpDir := t.TempDir()
	d, _ := NewModelDownloader(tmpDir)

	path := d.GetModelPath("test-model")
	expected := filepath.Join(tmpDir, "test-model")

	if path != expected {
		t.Errorf("GetModelPath: got %s, want %s", path, expected)
	}
}

func TestModelDownloader_IsModelCached(t *testing.T) {
	tmpDir := t.TempDir()
	d, _ := NewModelDownloader(tmpDir)

	// Not cached initially
	if d.IsModelCached("test-model") {
		t.Error("expected model to not be cached initially")
	}

	// Create model directory with model.onnx
	modelDir := filepath.Join(tmpDir, "test-model")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(modelDir, "model.onnx"), []byte("fake model"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Now should be cached
	if !d.IsModelCached("test-model") {
		t.Error("expected model to be cached after creating model.onnx")
	}
}

func TestModelDownloader_EnsureModel(t *testing.T) {
	// Create test server
	content := []byte("test model content")
	contentHash := sha256.Sum256(content)
	contentHashHex := hex.EncodeToString(contentHash[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/model.onnx":
			_, _ = w.Write(content)
		case "/tokenizer.json":
			_, _ = w.Write([]byte(`{"test": true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	d, _ := NewModelDownloader(tmpDir)

	model := ModelInfo{
		Name:      "test-model",
		Dimension: 384,
		Files: []ModelFile{
			{Name: "model.onnx", URL: server.URL + "/model.onnx", Size: int64(len(content))},
			{Name: "tokenizer.json", URL: server.URL + "/tokenizer.json", Size: 14},
		},
		Checksums: map[string]string{
			"model.onnx": contentHashHex,
		},
	}

	// Track progress
	var progressCalls int
	progress := func(p DownloadProgress) {
		progressCalls++
	}

	path, err := d.EnsureModel(context.Background(), model, progress)
	if err != nil {
		t.Fatalf("EnsureModel: %v", err)
	}

	// Verify path
	expected := filepath.Join(tmpDir, "test-model")
	if path != expected {
		t.Errorf("path: got %s, want %s", path, expected)
	}

	// Verify files exist
	if _, err := os.Stat(filepath.Join(path, "model.onnx")); err != nil {
		t.Errorf("model.onnx not found: %v", err)
	}

	if _, err := os.Stat(filepath.Join(path, "tokenizer.json")); err != nil {
		t.Errorf("tokenizer.json not found: %v", err)
	}

	// Verify progress was called
	if progressCalls == 0 {
		t.Error("expected progress callback to be called")
	}

	// Second call should use cache
	path2, err := d.EnsureModel(context.Background(), model, nil)
	if err != nil {
		t.Fatalf("EnsureModel (cached): %v", err)
	}

	if path2 != path {
		t.Errorf("cached path mismatch: got %s, want %s", path2, path)
	}
}

func TestModelDownloader_ChecksumFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("wrong content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	d, _ := NewModelDownloader(tmpDir)

	model := ModelInfo{
		Name: "test-model",
		Files: []ModelFile{
			{Name: "model.onnx", URL: server.URL + "/model.onnx", Size: 13},
		},
		Checksums: map[string]string{
			"model.onnx": "0000000000000000000000000000000000000000000000000000000000000000",
		},
	}

	_, err := d.EnsureModel(context.Background(), model, nil)
	if err == nil {
		t.Error("expected checksum error")
	}

	// Directory should be cleaned up
	if d.IsModelCached("test-model") {
		t.Error("model should not be cached after checksum failure")
	}
}

func TestModelDownloader_ClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	d, _ := NewModelDownloader(tmpDir)

	// Create fake model
	modelDir := filepath.Join(tmpDir, "test-model")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(modelDir, "model.onnx"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Clear specific model
	if err := d.ClearModelCache("test-model"); err != nil {
		t.Fatalf("ClearModelCache: %v", err)
	}

	if d.IsModelCached("test-model") {
		t.Error("model should not be cached after clear")
	}
}

func TestModelDownloader_ClearAllCache(t *testing.T) {
	tmpDir := t.TempDir()
	d, _ := NewModelDownloader(tmpDir)

	// Create fake models
	for _, name := range []string{"model1", "model2"} {
		modelDir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(modelDir, 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(modelDir, "model.onnx"), []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := d.ClearAllCache(); err != nil {
		t.Fatalf("ClearAllCache: %v", err)
	}

	if d.IsModelCached("model1") || d.IsModelCached("model2") {
		t.Error("models should not be cached after clear all")
	}
}

func TestModelDownloader_GetCacheSize(t *testing.T) {
	tmpDir := t.TempDir()
	d, _ := NewModelDownloader(tmpDir)

	// Initially empty
	size, err := d.GetCacheSize()
	if err != nil {
		t.Fatalf("GetCacheSize: %v", err)
	}

	if size != 0 {
		t.Errorf("expected size 0, got %d", size)
	}

	// Add some files
	modelDir := filepath.Join(tmpDir, "test-model")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := make([]byte, 1000)
	if err := os.WriteFile(filepath.Join(modelDir, "model.onnx"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	size, err = d.GetCacheSize()
	if err != nil {
		t.Fatalf("GetCacheSize: %v", err)
	}

	if size != 1000 {
		t.Errorf("expected size 1000, got %d", size)
	}
}

func TestGetModelInfo(t *testing.T) {
	// Known model
	info, err := GetModelInfo("all-MiniLM-L6-v2")
	if err != nil {
		t.Fatalf("GetModelInfo: %v", err)
	}

	if info.Dimension != 384 {
		t.Errorf("dimension: got %d, want 384", info.Dimension)
	}

	if len(info.Files) == 0 {
		t.Error("expected files")
	}

	// Unknown model
	_, err = GetModelInfo("unknown-model")
	if err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestListKnownModels(t *testing.T) {
	models := ListKnownModels()
	if len(models) == 0 {
		t.Error("expected at least one known model")
	}

	// Check all-MiniLM-L6-v2 is in list
	found := false
	for _, name := range models {
		if name == "all-MiniLM-L6-v2" {
			found = true

			break
		}
	}

	if !found {
		t.Error("all-MiniLM-L6-v2 not in known models")
	}
}
