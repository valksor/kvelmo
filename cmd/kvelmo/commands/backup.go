package commands

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/kvelmo/pkg/cli"
	"github.com/valksor/kvelmo/pkg/paths"
)

var backupJSON bool

var BackupCmd = &cobra.Command{
	Use:   "backup [output-path]",
	Short: "Backup kvelmo state to a tar.gz archive",
	Long: `Create a compressed archive of all kvelmo state.

Includes configuration, task data, recordings, and memory.
Excludes transient files (sockets, locks).

Examples:
  kvelmo backup                        # Default: kvelmo-backup-<timestamp>.tar.gz
  kvelmo backup /tmp/my-backup.tar.gz  # Custom output path`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBackup,
}

func init() {
	BackupCmd.Flags().BoolVar(&backupJSON, "json", false, "Output as JSON")
}

type backupOutput struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	Files int    `json:"files"`
}

func runBackup(cmd *cobra.Command, args []string) error {
	baseDir := paths.Paths().BaseDir()

	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return fmt.Errorf("kvelmo directory does not exist: %s", baseDir)
	}

	outputPath := fmt.Sprintf("kvelmo-backup-%s.tar.gz", time.Now().Format("20060102-150405"))
	if len(args) > 0 {
		outputPath = args[0]
	}

	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	fileCount, err := writeBackupArchive(absOutput, baseDir)
	if err != nil {
		return err
	}

	stat, err := os.Stat(absOutput)
	if err != nil {
		return fmt.Errorf("stat archive: %w", err)
	}

	out := backupOutput{
		Path:  absOutput,
		Size:  stat.Size(),
		Files: fileCount,
	}

	if backupJSON {
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal json: %w", err)
		}
		fmt.Println(string(data))

		return nil
	}

	if !cli.Quiet {
		fmt.Printf("Backup created: %s\n", absOutput)
		fmt.Printf("Size: %s (%d files)\n", formatBytes(out.Size), out.Files)
	}

	return nil
}

// writeBackupArchive creates the tar.gz archive and returns the file count.
func writeBackupArchive(absOutput, baseDir string) (int, error) {
	outFile, err := os.Create(absOutput)
	if err != nil {
		return 0, fmt.Errorf("create archive: %w", err)
	}

	gzWriter := gzip.NewWriter(outFile)
	tarWriter := tar.NewWriter(gzWriter)

	fileCount := 0

	// Use os.Root for race-safe filesystem access (Go 1.24+)
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		closeWriters(tarWriter, gzWriter, outFile)
		_ = os.Remove(absOutput)

		return 0, fmt.Errorf("open root directory: %w", err)
	}
	defer func() { _ = root.Close() }()

	walkErr := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip symbolic links
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		if isTransientFile(path) {
			return nil
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return fmt.Errorf("compute relative path: %w", err)
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", relPath, err)
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("create tar header for %s: %w", relPath, err)
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("write tar header for %s: %w", relPath, err)
		}

		if d.IsDir() {
			return nil
		}

		f, err := root.Open(relPath)
		if err != nil {
			return fmt.Errorf("open %s: %w", relPath, err)
		}
		defer func() { _ = f.Close() }()

		if _, err := io.Copy(tarWriter, f); err != nil {
			return fmt.Errorf("write %s to archive: %w", relPath, err)
		}

		fileCount++

		return nil
	})
	if walkErr != nil {
		closeWriters(tarWriter, gzWriter, outFile)
		_ = os.Remove(absOutput)

		return 0, fmt.Errorf("walk kvelmo directory: %w", walkErr)
	}

	// Flush writers before stat
	if err := tarWriter.Close(); err != nil {
		_ = os.Remove(absOutput)

		return 0, fmt.Errorf("finalize tar: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		_ = os.Remove(absOutput)

		return 0, fmt.Errorf("finalize gzip: %w", err)
	}
	if err := outFile.Close(); err != nil {
		_ = os.Remove(absOutput)

		return 0, fmt.Errorf("close archive: %w", err)
	}

	return fileCount, nil
}

// closeWriters closes tar, gzip, and file writers, ignoring errors (used for cleanup on failure).
func closeWriters(tw *tar.Writer, gz *gzip.Writer, f *os.File) {
	_ = tw.Close()
	_ = gz.Close()
	_ = f.Close()
}

// isTransientFile returns true for files that should be excluded from backup/restore.
func isTransientFile(name string) bool {
	ext := filepath.Ext(name)

	return ext == ".sock" || ext == ".lock"
}

// hasDotDot checks if a path contains ".." components (path traversal).
func hasDotDot(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}

	return false
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	//nolint:mnd // Standard SI binary unit suffixes
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
