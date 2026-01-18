package security

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/valksor/go-mehrhof/internal/storage"
)

const (
	// maxDecompressedSize is the maximum decompressed size for archives (100MB).
	maxDecompressedSize = 100 * 1024 * 1024
)

// ToolSpec defines a security tool that can be downloaded.
type ToolSpec struct {
	Name          string // "gitleaks", "gosec", "govulncheck"
	Repository    string // "zricethezav/gitleaks"
	AssetPattern  string // "gitleaks_{version}_{os}_{arch}"
	BinaryName    string // "gitleaks"
	ChecksumsFile string // "gitleaks_{version}_checksums.txt"
}

// GitHubRelease represents a GitHub release.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// ToolStatus represents the status of a tool.
type ToolStatus struct {
	Installed bool
	Path      string
	Version   string
	Source    string // "path", "cache", "downloaded"
}

// ToolManager manages downloading and caching security tools.
type ToolManager struct {
	toolsDir     string
	autoDownload bool
	client       *http.Client
	mu           sync.RWMutex
	warnings     []string
}

// NewToolManager creates a new ToolManager.
func NewToolManager(toolsDir string, autoDownload bool) (*ToolManager, error) {
	if toolsDir == "" {
		// Use default location: ~/.valksor/mehrhof/tools/
		homeDir, err := storage.GetMehrhofHomeDir()
		if err != nil {
			// Fallback to explicit path
			home, _ := os.UserHomeDir()
			homeDir = filepath.Join(home, ".valksor", "mehrhof")
		}
		toolsDir = filepath.Join(homeDir, "tools")
	}

	return &ToolManager{
		toolsDir:     toolsDir,
		autoDownload: autoDownload,
		client:       &http.Client{},
		warnings:     make([]string, 0),
	}, nil
}

// EnsureTool ensures a tool is available, returning its path.
// It checks PATH first, then cache, then downloads if auto-download is enabled.
// Returns error if tool cannot be made available.
func (tm *ToolManager) EnsureTool(ctx context.Context, spec ToolSpec) (string, error) {
	// 1. Check if tool is in PATH
	if path, err := exec.LookPath(spec.BinaryName); err == nil {
		return path, nil
	}

	// 2. Check if tool is in cache
	cachedPath, err := tm.getCachedTool(spec)
	if err == nil {
		return cachedPath, nil
	}

	// 3. Download if auto-download is enabled
	if tm.autoDownload {
		tm.addWarning(spec.Name + " not found in PATH or cache, downloading...")

		return tm.downloadTool(ctx, spec)
	}

	// 4. Tool not available and auto-download disabled
	return "", fmt.Errorf("%s not installed. Install with: go install %s (or enable auto-download in config)", spec.Name, getInstallCommand(spec))
}

// getCachedTool checks if a tool is in the cache and returns its path.
func (tm *ToolManager) getCachedTool(spec ToolSpec) (string, error) {
	// List all directories in toolsDir
	entries, err := os.ReadDir(tm.toolsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("cache directory does not exist")
		}

		return "", fmt.Errorf("failed to read cache directory: %w", err)
	}

	// Look for matching tool directories
	// Pattern: {name}-{version}-{os}-{arch}
	prefix := spec.Name + "-"
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		if !strings.HasPrefix(dirName, prefix) {
			continue
		}

		// Check if binary exists in this directory
		binaryPath := filepath.Join(tm.toolsDir, dirName, spec.BinaryName)
		if runtime.GOOS == "windows" {
			binaryPath += ".exe"
		}

		if _, err := os.Stat(binaryPath); err == nil {
			return binaryPath, nil
		}
	}

	return "", errors.New("tool not found in cache")
}

// downloadTool downloads a tool from GitHub releases.
func (tm *ToolManager) downloadTool(ctx context.Context, spec ToolSpec) (string, error) {
	// Get latest release info
	release, err := tm.getLatestRelease(ctx, spec.Repository)
	if err != nil {
		tm.addWarning(fmt.Sprintf("failed to fetch %s release info: %v", spec.Name, err))

		return "", fmt.Errorf("failed to fetch release info: %w", err)
	}

	version := strings.TrimPrefix(release.TagName, "v")

	// Find the appropriate asset for current platform
	assetURL, assetName, err := tm.findAssetForPlatform(release, spec)
	if err != nil {
		tm.addWarning(fmt.Sprintf("no %s binary available for %s/%s: %v", spec.Name, runtime.GOOS, runtime.GOARCH, err))

		return "", fmt.Errorf("no binary available for platform: %w", err)
	}

	// Create cache directory for this version
	cacheDir := filepath.Join(tm.toolsDir, fmt.Sprintf("%s-%s-%s-%s", spec.Name, version, runtime.GOOS, runtime.GOARCH))
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Download the asset
	tm.addWarning(fmt.Sprintf("Downloading %s %s for %s/%s...", spec.Name, version, runtime.GOOS, runtime.GOARCH))

	downloadPath := filepath.Join(cacheDir, assetName)
	if err := tm.downloadAsset(ctx, assetURL, downloadPath); err != nil {
		return "", fmt.Errorf("failed to download asset: %w", err)
	}

	// Extract if it's a tar.gz
	if strings.HasSuffix(assetName, ".tar.gz") {
		binaryPath, err := tm.extractTarGz(downloadPath, cacheDir, spec.BinaryName)
		if err != nil {
			return "", fmt.Errorf("failed to extract archive: %w", err)
		}

		// Make executable
		if err := os.Chmod(binaryPath, 0o755); err != nil {
			return "", fmt.Errorf("failed to make binary executable: %w", err)
		}

		// Clean up downloaded archive
		_ = os.Remove(downloadPath)

		return binaryPath, nil
	}

	// Otherwise, assume it's the binary itself
	binaryPath := downloadPath
	if runtime.GOOS == "windows" && !strings.HasSuffix(binaryPath, ".exe") {
		newPath := binaryPath + ".exe"
		if err := os.Rename(binaryPath, newPath); err != nil {
			return "", fmt.Errorf("failed to rename binary: %w", err)
		}
		binaryPath = newPath
	}

	// Make executable
	if err := os.Chmod(binaryPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	return binaryPath, nil
}

// getLatestRelease fetches the latest release from GitHub.
func (tm *ToolManager) getLatestRelease(ctx context.Context, repo string) (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := tm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// findAssetForPlatform finds the appropriate asset for the current platform.
func (tm *ToolManager) findAssetForPlatform(release *GitHubRelease, spec ToolSpec) (string, string, error) {
	// Build expected asset name pattern
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Normalize arch names
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	}

	// Try different patterns
	patterns := []string{
		fmt.Sprintf("%s_%s_%s-%s", spec.Name, release.TagName, os, arch),
		fmt.Sprintf("%s_%s_%s", spec.Name, os, arch),
		fmt.Sprintf("%s-%s-%s", spec.Name, os, arch),
	}

	for _, asset := range release.Assets {
		assetName := asset.Name

		// Check if any pattern matches
		for _, pattern := range patterns {
			if strings.Contains(strings.ToLower(assetName), strings.ToLower(pattern)) {
				return asset.URL, assetName, nil
			}
		}

		// Fallback: check if asset name contains OS and arch
		if strings.Contains(strings.ToLower(assetName), os) &&
			strings.Contains(strings.ToLower(assetName), runtime.GOARCH) {
			return asset.URL, assetName, nil
		}
	}

	return "", "", fmt.Errorf("no matching asset found for platform %s/%s", runtime.GOOS, runtime.GOARCH)
}

// downloadAsset downloads a file from a URL.
func (tm *ToolManager) downloadAsset(ctx context.Context, url, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := tm.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Create file
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	// Download with progress could be added here
	_, err = io.Copy(f, resp.Body)

	return err
}

// extractTarGz extracts a tar.gz archive and returns the path to the specified binary.
func (tm *ToolManager) extractTarGz(archivePath, destDir, binaryName string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)

	var decompressedSize int64

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Check for decompression bomb
		if header.Size > 0 && decompressedSize+header.Size > maxDecompressedSize {
			return "", fmt.Errorf("decompressed size exceeds maximum allowed size of %d bytes", maxDecompressedSize)
		}

		targetPath := filepath.Join(destDir, filepath.Base(header.Name))

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return "", err
			}
		case tar.TypeReg:
			// Extract file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return "", err
			}

			// Use limited reader to prevent decompression bomb
			// Limit individual file size to prevent DoS
			maxFileSize := int64(10 * 1024 * 1024) // 10MB per file
			limitedReader := io.LimitReader(tr, maxFileSize)
			n, copyErr := io.Copy(outFile, limitedReader)
			decompressedSize += n
			if copyErr != nil {
				_ = outFile.Close()

				return "", copyErr
			}

			if err := outFile.Close(); err != nil {
				return "", err
			}

			// Check if this is the binary we're looking for
			if filepath.Base(header.Name) == binaryName {
				return targetPath, nil
			}
		}
	}

	return "", fmt.Errorf("binary %s not found in archive", binaryName)
}

// addWarning adds a warning message.
func (tm *ToolManager) addWarning(msg string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.warnings = append(tm.warnings, msg)
}

// GetWarnings returns all warnings.
func (tm *ToolManager) GetWarnings() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.warnings
}

// HasWarnings returns true if there are any warnings.
func (tm *ToolManager) HasWarnings() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return len(tm.warnings) > 0
}

// ClearWarnings clears all warnings.
func (tm *ToolManager) ClearWarnings() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.warnings = make([]string, 0)
}

// getInstallCommand returns the go install command for a tool.
func getInstallCommand(spec ToolSpec) string {
	switch spec.Name {
	case "gitleaks":
		return "github.com/zricethezav/gitleaks/v8/cmd/gitleaks@latest"
	case "gosec":
		return "github.com/securego/gosec/v2/cmd/gosec@latest"
	case "govulncheck":
		return "golang.org/x/vuln/cmd/govulncheck@latest"
	default:
		return spec.Repository
	}
}
