package memory

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/valksor/go-mehrhof/internal/update"
	"github.com/valksor/go-toolkit/version"
)

const (
	// embedderBinaryName is the name of the embedder binary.
	embedderBinaryName = "mehr-embedder"

	// embedderInstallDir is the relative path within the mehrhof home directory.
	embedderInstallDir = "bin"

	// githubReleaseURLTemplate is the URL template for downloading release assets.
	// Uses the same release as the main mehr binary.
	githubReleaseURLTemplate = "https://github.com/valksor/go-mehrhof/releases/download/%s/%s"
)

// EmbedderDownloader handles downloading the mehr-embedder binary.
type EmbedderDownloader struct {
	downloader *update.Downloader
	installDir string
}

// NewEmbedderDownloader creates a new embedder downloader.
// If installDir is empty, defaults to ~/.valksor/mehrhof/bin/.
func NewEmbedderDownloader(installDir string) (*EmbedderDownloader, error) {
	if installDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		installDir = filepath.Join(homeDir, ".valksor", "mehrhof", embedderInstallDir)
	}

	return &EmbedderDownloader{
		downloader: update.NewDownloader(),
		installDir: installDir,
	}, nil
}

// EnsureEmbedder ensures the embedder binary is available.
// Downloads if not present or if version doesn't match.
// Returns the path to the embedder binary.
func (d *EmbedderDownloader) EnsureEmbedder(ctx context.Context) (string, error) {
	binPath := d.EmbedderPath()

	// Check if binary exists
	if _, err := os.Stat(binPath); err == nil {
		// Binary exists - for now, assume it's the correct version
		// In the future, could check version and upgrade if needed
		slog.Debug("embedder binary found", "path", binPath)

		return binPath, nil
	}

	// Download the embedder
	slog.Info("downloading mehr-embedder binary")
	if err := d.Download(ctx); err != nil {
		return "", err
	}

	return binPath, nil
}

// EmbedderPath returns the path where the embedder should be installed.
func (d *EmbedderDownloader) EmbedderPath() string {
	return filepath.Join(d.installDir, embedderBinaryName)
}

// Download downloads and installs the embedder binary.
func (d *EmbedderDownloader) Download(ctx context.Context) error {
	// Ensure install directory exists
	if err := os.MkdirAll(d.installDir, 0o755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}

	// Determine the asset name for this platform
	assetName := GetEmbedderAssetName()
	releaseTag := d.getReleaseTag()

	slog.Info("downloading embedder",
		"asset", assetName,
		"tag", releaseTag,
		"installDir", d.installDir)

	// Build URLs
	// The embedder uses individual .sha256 files (not the main checksums.txt)
	binaryURL := fmt.Sprintf(githubReleaseURLTemplate, releaseTag, assetName)
	checksumURL := fmt.Sprintf(githubReleaseURLTemplate, releaseTag, assetName+".sha256")

	// Download with checksum verification
	// Note: The embedder doesn't have a minisign signature (only the main checksums.txt is signed)
	tmpPath, err := d.downloader.DownloadWithChecksums(ctx, binaryURL, checksumURL, assetName)
	if err != nil {
		return fmt.Errorf("download embedder: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	slog.Debug("embedder downloaded and checksum verified")

	// Move to final location
	destPath := d.EmbedderPath()
	if err := d.installBinary(tmpPath, destPath); err != nil {
		return fmt.Errorf("install embedder: %w", err)
	}

	slog.Info("embedder installed", "path", destPath)

	return nil
}

// installBinary copies the binary from tmpPath to destPath and makes it executable.
func (d *EmbedderDownloader) installBinary(tmpPath, destPath string) error {
	// Remove existing binary if present
	_ = os.Remove(destPath)

	// Open source file
	src, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer func() { _ = src.Close() }()

	// Create destination file
	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer func() { _ = dst.Close() }()

	// Copy contents
	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(destPath)

		return fmt.Errorf("copy binary: %w", err)
	}

	return nil
}

// getReleaseTag returns the release tag to download from.
// Uses the current version if it's a release, otherwise uses "nightly".
func (d *EmbedderDownloader) getReleaseTag() string {
	v := version.Version
	if v == "" || v == "dev" || v == "nightly" {
		return "nightly"
	}
	// Check if it's a valid release tag (starts with v and has numbers)
	if len(v) > 1 && v[0] == 'v' {
		return v
	}
	// For any other format (like commit hashes), use nightly
	return "nightly"
}

// GetEmbedderAssetName returns the asset name for the current platform.
func GetEmbedderAssetName() string {
	return fmt.Sprintf("%s-%s-%s", embedderBinaryName, runtime.GOOS, runtime.GOARCH)
}

// IsEmbedderAvailable checks if the embedder binary is available for this platform.
func IsEmbedderAvailable() bool {
	return isEmbedderAvailable(runtime.GOOS, runtime.GOARCH)
}

func isEmbedderAvailable(goos, goarch string) bool {
	// Native embedder binaries are published for Linux/macOS on amd64/arm64.
	// Windows native binaries are not published; WSL uses Linux binaries.
	switch goos {
	case "linux", "darwin":
		return goarch == "amd64" || goarch == "arm64"
	default:
		return false
	}
}
