package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

var CleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove stale socket files",
	Long: `Remove stale socket files that may be left over from crashed processes.

This command detects sockets that exist but aren't responding and removes them.
Use --dry-run to see what would be removed without actually deleting.
Use --force to skip confirmation prompts.`,
	RunE: runCleanup,
}

func init() {
	CleanupCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompts")
	CleanupCmd.Flags().Bool("dry-run", false, "Show what would be removed without deleting")
}

func runCleanup(cmd *cobra.Command, args []string) error {
	cleanupForce, _ := cmd.Flags().GetBool("force")
	cleanupDry, _ := cmd.Flags().GetBool("dry-run")
	var staleFiles []string

	// Check global socket
	globalPath := socket.GlobalSocketPath()
	globalSocketStale := isStaleSocket(globalPath)
	if globalSocketStale {
		staleFiles = append(staleFiles, globalPath)
	}

	// Check global lock (stale if socket doesn't exist or is unresponsive)
	lockPath := socket.GlobalLockPath()
	if (!socket.SocketExists(globalPath) || globalSocketStale) && fileExists(lockPath) {
		staleFiles = append(staleFiles, lockPath)
	}

	// Check worktree sockets
	worktreesDir := filepath.Join(socket.BaseDir(), "worktrees")
	if entries, err := os.ReadDir(worktreesDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sock" {
				sockPath := filepath.Join(worktreesDir, entry.Name())
				if isStaleSocket(sockPath) {
					staleFiles = append(staleFiles, sockPath)
				}
			}
		}
	}

	if len(staleFiles) == 0 {
		fmt.Println("No stale sockets found.")

		return nil
	}

	fmt.Printf("Found %d stale socket(s):\n", len(staleFiles))
	for _, f := range staleFiles {
		fmt.Printf("  • %s\n", f)
	}

	if cleanupDry {
		fmt.Println("\n(dry-run: no files removed)")

		return nil
	}

	if !cleanupForce {
		fmt.Print("\nRemove these files? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response) // EOF or error treated as decline
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")

			return nil
		}
	}

	removed := 0
	for _, f := range staleFiles {
		if err := os.Remove(f); err != nil {
			fmt.Printf("  ✗ Failed to remove %s: %v\n", f, err)
		} else {
			fmt.Printf("  ✓ Removed %s\n", f)
			removed++
		}
	}

	fmt.Printf("\nRemoved %d file(s).\n", removed)
	fmt.Printf("Run '%s serve' to start fresh.\n", meta.Name)

	return nil
}

// isStaleSocket checks if a socket file exists but isn't responding.
func isStaleSocket(path string) bool {
	if !socket.SocketExists(path) {
		return false
	}

	// Try to connect with short timeout
	client, err := socket.NewClient(path, socket.WithTimeout(500*time.Millisecond))
	if err != nil {
		// Socket exists but can't connect - it's stale
		return true
	}
	_ = client.Close()

	// Socket is responsive - not stale
	return false
}
