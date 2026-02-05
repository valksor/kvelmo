package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ModelInfo describes a downloadable embedding model.
type ModelInfo struct {
	Name        string            // Model name (e.g., "all-MiniLM-L6-v2")
	Dimension   int               // Embedding dimension
	Files       []ModelFile       // Files to download
	Description string            // Human-readable description
	Size        int64             // Total size in bytes (approximate)
	Checksums   map[string]string // SHA256 checksums by filename
}

// ModelFile describes a single file to download for a model.
type ModelFile struct {
	Name string // Filename (e.g., "model.onnx")
	URL  string // Download URL
	Size int64  // Size in bytes (approximate)
}

// DownloadProgress reports download progress.
type DownloadProgress struct {
	File           string  // Current file being downloaded
	BytesCompleted int64   // Bytes downloaded so far
	BytesTotal     int64   // Total bytes to download
	Percent        float64 // Progress percentage (0-100)
}

// ProgressCallback is called during download to report progress.
type ProgressCallback func(DownloadProgress)

// ModelDownloader handles downloading and caching embedding models.
type ModelDownloader struct {
	cacheDir   string
	httpClient *http.Client
}

// NewModelDownloader creates a downloader with the specified cache directory.
// If cacheDir is empty, uses ~/.valksor/mehrhof/models/.
func NewModelDownloader(cacheDir string) (*ModelDownloader, error) {
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}

		cacheDir = filepath.Join(home, ".valksor", "mehrhof", "models")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	return &ModelDownloader{
		cacheDir: cacheDir,
		httpClient: &http.Client{
			Timeout: 30 * time.Minute, // Large models may take time
		},
	}, nil
}

// GetModelPath returns the path where a model is/would be cached.
func (d *ModelDownloader) GetModelPath(modelName string) string {
	return filepath.Join(d.cacheDir, modelName)
}

// IsModelCached checks if a model is already downloaded.
func (d *ModelDownloader) IsModelCached(modelName string) bool {
	modelPath := d.GetModelPath(modelName)

	// Check if model.onnx exists (the main model file)
	onnxPath := filepath.Join(modelPath, "model.onnx")
	_, err := os.Stat(onnxPath)

	return err == nil
}

// EnsureModel downloads the model if not cached, returns path to model directory.
func (d *ModelDownloader) EnsureModel(ctx context.Context, model ModelInfo, progress ProgressCallback) (string, error) {
	modelPath := d.GetModelPath(model.Name)

	// Check if already cached
	if d.IsModelCached(model.Name) {
		// Verify checksums if provided
		if err := d.verifyChecksums(modelPath, model.Checksums); err != nil {
			// Checksums failed, re-download
			if err := os.RemoveAll(modelPath); err != nil {
				return "", fmt.Errorf("remove corrupted model: %w", err)
			}
		} else {
			return modelPath, nil
		}
	}

	// Create model directory
	if err := os.MkdirAll(modelPath, 0o755); err != nil {
		return "", fmt.Errorf("create model dir: %w", err)
	}

	// Download all files
	for _, file := range model.Files {
		if err := d.downloadFile(ctx, file, modelPath, progress); err != nil {
			// Clean up on failure
			_ = os.RemoveAll(modelPath)

			return "", fmt.Errorf("download %s: %w", file.Name, err)
		}
	}

	// Verify checksums after download
	if err := d.verifyChecksums(modelPath, model.Checksums); err != nil {
		_ = os.RemoveAll(modelPath)

		return "", fmt.Errorf("checksum verification failed: %w", err)
	}

	return modelPath, nil
}

// downloadFile downloads a single file to the model directory.
func (d *ModelDownloader) downloadFile(ctx context.Context, file ModelFile, modelPath string, progress ProgressCallback) error {
	filePath := filepath.Join(modelPath, file.Name)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, file.URL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "mehrhof-model-downloader/1.0")

	// Execute request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Get content length
	contentLength := resp.ContentLength
	if contentLength < 0 {
		contentLength = file.Size // Use estimated size if not provided
	}

	// Create output file
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = out.Close() }()

	// Download with progress
	var downloaded int64
	var reader io.Reader = resp.Body

	if progress != nil {
		reader = &progressReader{
			reader: resp.Body,
			onProgress: func(n int64) {
				downloaded += n
				progress(DownloadProgress{
					File:           file.Name,
					BytesCompleted: downloaded,
					BytesTotal:     contentLength,
					Percent:        float64(downloaded) / float64(contentLength) * 100,
				})
			},
		}
	}

	if _, err := io.Copy(out, reader); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// verifyChecksums checks SHA256 checksums for downloaded files.
func (d *ModelDownloader) verifyChecksums(modelPath string, checksums map[string]string) error {
	if len(checksums) == 0 {
		return nil // No checksums to verify
	}

	for filename, expected := range checksums {
		filePath := filepath.Join(modelPath, filename)

		actual, err := fileChecksum(filePath)
		if err != nil {
			return fmt.Errorf("checksum %s: %w", filename, err)
		}

		if !strings.EqualFold(actual, expected) {
			return fmt.Errorf("checksum mismatch for %s: got %s, want %s", filename, actual, expected)
		}
	}

	return nil
}

// fileChecksum computes SHA256 checksum of a file.
func fileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// progressReader wraps an io.Reader to report progress.
type progressReader struct {
	reader     io.Reader
	onProgress func(int64)
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 && r.onProgress != nil {
		r.onProgress(int64(n))
	}

	return n, err
}

// KnownModels contains information about supported embedding models.
var KnownModels = map[string]ModelInfo{
	"all-MiniLM-L6-v2": {
		Name:        "all-MiniLM-L6-v2",
		Dimension:   384,
		Description: "Fast, lightweight model with good quality (22MB)",
		Size:        23_000_000, // ~22MB total
		Files: []ModelFile{
			{
				Name: "model.onnx",
				URL:  "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx",
				Size: 22_700_000,
			},
			{
				Name: "tokenizer.json",
				URL:  "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/tokenizer.json",
				Size: 712_000,
			},
			{
				Name: "config.json",
				URL:  "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/config.json",
				Size: 600,
			},
		},
		Checksums: map[string]string{
			// Note: These are placeholder checksums - will update with actual values
			// after verifying downloads from HuggingFace
		},
	},
	"all-MiniLM-L12-v2": {
		Name:        "all-MiniLM-L12-v2",
		Dimension:   384,
		Description: "Higher quality model with more layers (33MB)",
		Size:        34_000_000,
		Files: []ModelFile{
			{
				Name: "model.onnx",
				URL:  "https://huggingface.co/sentence-transformers/all-MiniLM-L12-v2/resolve/main/onnx/model.onnx",
				Size: 33_400_000,
			},
			{
				Name: "tokenizer.json",
				URL:  "https://huggingface.co/sentence-transformers/all-MiniLM-L12-v2/resolve/main/tokenizer.json",
				Size: 712_000,
			},
			{
				Name: "config.json",
				URL:  "https://huggingface.co/sentence-transformers/all-MiniLM-L12-v2/resolve/main/config.json",
				Size: 600,
			},
		},
		Checksums: map[string]string{},
	},
}

// GetModelInfo returns information about a known model.
func GetModelInfo(name string) (ModelInfo, error) {
	info, ok := KnownModels[name]
	if !ok {
		return ModelInfo{}, fmt.Errorf("unknown model: %s", name)
	}

	return info, nil
}

// ListKnownModels returns names of all known models.
func ListKnownModels() []string {
	names := make([]string, 0, len(KnownModels))
	for name := range KnownModels {
		names = append(names, name)
	}

	return names
}

// ClearModelCache removes a cached model.
func (d *ModelDownloader) ClearModelCache(modelName string) error {
	modelPath := d.GetModelPath(modelName)

	if err := os.RemoveAll(modelPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove model cache: %w", err)
	}

	return nil
}

// ClearAllCache removes all cached models.
func (d *ModelDownloader) ClearAllCache() error {
	entries, err := os.ReadDir(d.cacheDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("read cache dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if err := os.RemoveAll(filepath.Join(d.cacheDir, entry.Name())); err != nil {
				return fmt.Errorf("remove %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// GetCacheSize returns the total size of cached models in bytes.
func (d *ModelDownloader) GetCacheSize() (int64, error) {
	var total int64

	err := filepath.Walk(d.cacheDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			total += info.Size()
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}

		return 0, err
	}

	return total, nil
}
