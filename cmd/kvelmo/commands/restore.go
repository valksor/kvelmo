package commands

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/valksor/kvelmo/pkg/cli"
	"github.com/valksor/kvelmo/pkg/paths"
)

const maxRestoreFileSize = 1 << 30 // 1 GB safety limit

var (
	restoreDryRun bool
	restoreTarget string
	restoreJSON   bool
)

var RestoreCmd = &cobra.Command{
	Use:   "restore <archive-path>",
	Short: "Restore kvelmo state from a backup archive",
	Long: `Restore kvelmo state from a previously created backup.

The archive is extracted to ~/.valksor/kvelmo/ by default.
Use --target to specify a different destination.
Use --dry-run to list archive contents without extracting.

Examples:
  kvelmo restore backup.tar.gz            # Restore to default location
  kvelmo restore backup.tar.gz --dry-run  # List contents only
  kvelmo restore backup.tar.gz --target /tmp/kvelmo  # Custom target`,
	Args: cobra.ExactArgs(1),
	RunE: runRestore,
}

func init() {
	RestoreCmd.Flags().BoolVar(&restoreDryRun, "dry-run", false, "List contents without extracting")
	RestoreCmd.Flags().StringVar(&restoreTarget, "target", "", "Target directory (default: ~/.valksor/kvelmo/)")
	RestoreCmd.Flags().BoolVar(&restoreJSON, "json", false, "Output as JSON")
}

type restoreOutput struct {
	Target  string   `json:"target"`
	Files   int      `json:"files"`
	Dirs    int      `json:"dirs"`
	Skipped int      `json:"skipped"`
	DryRun  bool     `json:"dry_run"`
	Entries []string `json:"entries,omitempty"`
}

func runRestore(cmd *cobra.Command, args []string) error {
	archivePath := args[0]

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("read gzip: %w", err)
	}
	defer func() { _ = gzReader.Close() }()

	tarReader := tar.NewReader(gzReader)

	targetDir := paths.Paths().BaseDir()
	if restoreTarget != "" {
		targetDir = restoreTarget
	}

	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}

	out := restoreOutput{
		Target: absTarget,
		DryRun: restoreDryRun,
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		// Safety: reject path traversal
		if hasDotDot(header.Name) {
			return fmt.Errorf("unsafe path in archive: %s", header.Name)
		}

		cleanName := filepath.Clean(header.Name)

		// Skip transient files
		if isTransientFile(cleanName) {
			out.Skipped++

			continue
		}

		// Skip symbolic links
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			out.Skipped++

			continue
		}

		// Skip oversized files
		if header.Size > maxRestoreFileSize {
			out.Skipped++
			if !cli.Quiet && !restoreJSON {
				fmt.Printf("  Skipped (>1GB): %s\n", cleanName)
			}

			continue
		}

		if restoreDryRun {
			out.Entries = append(out.Entries, cleanName)
			if header.Typeflag == tar.TypeDir {
				out.Dirs++
			} else {
				out.Files++
			}

			continue
		}

		destPath := filepath.Join(absTarget, cleanName)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0o750); err != nil {
				return fmt.Errorf("create directory %s: %w", cleanName, err)
			}
			out.Dirs++

		case tar.TypeReg:
			if err := restoreFile(tarReader, destPath, header, cleanName); err != nil {
				return err
			}
			out.Files++

		default:
			out.Skipped++
		}
	}

	return printRestoreOutput(out)
}

// restoreFile extracts a single regular file from the tar stream.
func restoreFile(tarReader *tar.Reader, destPath string, header *tar.Header, cleanName string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
		return fmt.Errorf("create parent dir for %s: %w", cleanName, err)
	}

	perm := filePermission(cleanName)

	outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("create file %s: %w", cleanName, err)
	}
	defer func() { _ = outFile.Close() }()

	// Use LimitReader as defense-in-depth (header.Size already checked)
	if _, err := io.Copy(outFile, io.LimitReader(tarReader, header.Size)); err != nil {
		return fmt.Errorf("extract %s: %w", cleanName, err)
	}

	return nil
}

// filePermission returns the appropriate permission for a file based on its name.
// Sensitive files (.env) get restrictive permissions.
func filePermission(name string) os.FileMode {
	base := filepath.Base(name)
	if base == ".env" || filepath.Ext(base) == ".env" {
		return 0o600
	}

	return 0o644
}

func printRestoreOutput(out restoreOutput) error {
	if restoreJSON {
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal json: %w", err)
		}
		fmt.Println(string(data))

		return nil
	}

	if out.DryRun {
		if !cli.Quiet {
			fmt.Printf("Archive contents (%d files, %d directories):\n", out.Files, out.Dirs)
			for _, entry := range out.Entries {
				fmt.Printf("  %s\n", entry)
			}
			if out.Skipped > 0 {
				fmt.Printf("\n%d entries would be skipped (transient/symlinks/oversized)\n", out.Skipped)
			}
		}

		return nil
	}

	if !cli.Quiet {
		fmt.Printf("Restored to: %s\n", out.Target)
		fmt.Printf("Files: %d, Directories: %d", out.Files, out.Dirs)
		if out.Skipped > 0 {
			fmt.Printf(", Skipped: %d", out.Skipped)
		}
		fmt.Println()
	}

	return nil
}
