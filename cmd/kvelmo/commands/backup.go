package commands

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/valksor/kvelmo/pkg/backup"
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

func runBackup(_ *cobra.Command, args []string) error {
	baseDir := paths.Paths().BaseDir()

	outputPath := ""
	if len(args) > 0 {
		outputPath = args[0]
	}

	result, err := backup.Create(baseDir, outputPath)
	if err != nil {
		return err
	}

	if backupJSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal json: %w", err)
		}
		fmt.Println(string(data))

		return nil
	}

	if !cli.Quiet {
		fmt.Printf("Backup created: %s\n", result.Path)
		fmt.Printf("Size: %s (%d files)\n", backup.FormatBytes(result.Size), result.Files)
	}

	return nil
}

// isTransientFile returns true for files that should be excluded from backup/restore.
func isTransientFile(name string) bool {
	return backup.IsTransientFile(name)
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
