//go:build linux

package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/sys/unix"
)

// LinuxSandbox uses Linux user namespaces for unprivileged chroot.
type LinuxSandbox struct {
	cfg      *Config
	rootDir  string // Temporary sandbox root directory
	prepared bool
}

// newPlatformSandbox creates a Linux sandbox implementation.
func newPlatformSandbox(cfg *Config) (Sandbox, error) {
	if cfg.HomeDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
		cfg.HomeDir = homeDir
	}

	// Set default tmpdir if not specified
	if cfg.TmpDir == "" {
		cfg.TmpDir = filepath.Join(os.TempDir(), "mehrhof-sandbox-tmp")
	}

	return &LinuxSandbox{cfg: cfg}, nil
}

// Prepare sets up the sandbox root directory with necessary mounts.
func (s *LinuxSandbox) Prepare(ctx context.Context) error {
	if s.prepared {
		return nil
	}

	// Create temporary sandbox root
	tmpDir := os.TempDir()
	rootDir, err := os.MkdirTemp(tmpDir, "mehrhof-sandbox-")
	if err != nil {
		return fmt.Errorf("create sandbox root: %w", err)
	}
	s.rootDir = rootDir

	// Create directory structure
	dirs := []string{
		filepath.Join(rootDir, "tmp"),
		filepath.Join(rootDir, "dev"),
		filepath.Join(rootDir, "proc"),
		filepath.Join(rootDir, "workspace"),
		filepath.Join(rootDir, "home", "user"),
		filepath.Join(rootDir, "lib"),
		filepath.Join(rootDir, "lib64"),
		filepath.Join(rootDir, "usr", "lib"),
		filepath.Join(rootDir, "usr", "lib64"),
		filepath.Join(rootDir, "bin"),
		filepath.Join(rootDir, "usr", "bin"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Mount tmpfs on /tmp
	if err := unix.Mount("tmpfs", filepath.Join(rootDir, "tmp"), "tmpfs",
		unix.MS_NOSUID|unix.MS_NOEXEC|unix.MS_NODEV, ""); err != nil {
		return fmt.Errorf("mount tmpfs: %w", err)
	}

	// Mount /proc
	if err := unix.Mount("proc", filepath.Join(rootDir, "proc"), "proc", 0, ""); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	// Create device nodes
	devNodes := []struct {
		name  string
		mode  uint32
		major uint32
		minor uint32
	}{
		{"null", unix.S_IFCHR | 0o666, 1, 3},
		{"zero", unix.S_IFCHR | 0o666, 1, 5},
		{"random", unix.S_IFCHR | 0o666, 1, 8},
		{"urandom", unix.S_IFCHR | 0o666, 1, 9},
	}
	devDir := filepath.Join(rootDir, "dev")
	for _, dev := range devNodes {
		path := filepath.Join(devDir, dev.name)
		devNum := unix.Mkdev(dev.major, dev.minor)
		if err := unix.Mknod(path, dev.mode, int(devNum)); err != nil { //nolint:gosec //G115 devNum fits in int
			// Non-fatal, device may already exist
			_ = os.Remove(path)
			if err := unix.Mknod(path, dev.mode, int(devNum)); err != nil { //nolint:gosec //G115 devNum fits in int
				return fmt.Errorf("mknod %s: %w", dev.name, err)
			}
		}
	}

	// Bind mount project directory to /workspace
	projectBind := filepath.Join(rootDir, "workspace")
	if err := unix.Mount(s.cfg.ProjectDir, projectBind, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		return fmt.Errorf("bind mount project dir: %w", err)
	}

	// Bind mount .claude directory
	claudeSrc := filepath.Join(s.cfg.HomeDir, ".claude")
	claudeDst := filepath.Join(rootDir, "home", "user", ".claude")
	if _, err := os.Stat(claudeSrc); err == nil {
		if err := os.MkdirAll(claudeDst, 0o755); err != nil {
			return fmt.Errorf("create .claude directory: %w", err)
		}
		if err := unix.Mount(claudeSrc, claudeDst, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
			return fmt.Errorf("bind mount .claude: %w", err)
		}
	}

	// Bind mount system directories for shared libraries
	// This is needed for dynamically-linked binaries
	systemMounts := []struct {
		src      string
		dst      string
		required bool
	}{
		{"/lib", filepath.Join(rootDir, "lib"), true},
		{"/lib64", filepath.Join(rootDir, "lib64"), false},
		{"/usr/lib", filepath.Join(rootDir, "usr", "lib"), true},
		{"/usr/lib64", filepath.Join(rootDir, "usr", "lib64"), false},
		{"/bin", filepath.Join(rootDir, "bin"), false},
		{"/usr/bin", filepath.Join(rootDir, "usr", "bin"), false},
	}

	for _, m := range systemMounts {
		if _, err := os.Stat(m.src); err != nil {
			if m.required {
				return fmt.Errorf("system directory %s not found", m.src)
			}

			continue
		}
		if err := unix.Mount(m.src, m.dst, "", unix.MS_BIND|unix.MS_RDONLY|unix.MS_REC, ""); err != nil {
			// Non-fatal for optional mounts
			if m.required {
				return fmt.Errorf("bind mount %s: %w", m.src, err)
			}
		}
	}

	s.prepared = true

	return nil
}

// Cleanup removes the sandbox root directory.
func (s *LinuxSandbox) Cleanup(ctx context.Context) error {
	if s.rootDir != "" {
		// Unmount all mounts under root
		unmountAll(s.rootDir)
		// Remove the temporary directory
		if err := os.RemoveAll(s.rootDir); err != nil {
			return fmt.Errorf("remove sandbox root: %w", err)
		}
		s.rootDir = ""
		s.prepared = false
	}

	return nil
}

// WrapCommand wraps the command to execute inside the sandbox using unshare.
func (s *LinuxSandbox) WrapCommand(cmd *exec.Cmd) (*exec.Cmd, error) {
	if err := s.Prepare(context.Background()); err != nil {
		return nil, fmt.Errorf("prepare sandbox: %w", err)
	}

	// We need to use unshare to create a new user namespace
	// The actual pivot_root will happen in a child process

	// Build unshare command
	// CLONE_NEWUSER: Create new user namespace (unprivileged)
	// CLONE_NEWNS: Create new mount namespace
	// CLONE_NEWPID: Create new PID namespace (optional, adds isolation)
	// Note: We do NOT use CLONE_NEWNET because we need network access for LLM APIs

	// Build the pivot_root script
	pivotScript := s.buildPivotScript()

	// Execute the command via unshare + shell that does the pivot_root
	shellCmd := exec.CommandContext(context.Background(),
		"unshare",
		"--user", "--map-root-user", // Map current user to root in new namespace
		"--mount",      // New mount namespace
		"--mount-proc", // Remount /proc in new namespace
		"--pid",        // New PID namespace (optional)
		"--fork",       // Fork before executing
		"--pid",        // Fork again for PID namespace
		"sh", "-c", pivotScript+" "+cmd.Path+" "+stringsJoin(cmd.Args[1:], " "),
	)

	shellCmd.Stdin = cmd.Stdin
	shellCmd.Stdout = cmd.Stdout
	shellCmd.Stderr = cmd.Stderr
	shellCmd.Env = cmd.Env

	// Set working directory to sandbox workspace
	shellCmd.Dir = filepath.Join(s.rootDir, "workspace")

	return shellCmd, nil
}

// buildPivotScript builds a shell script that performs pivot_root.
func (s *LinuxSandbox) buildPivotScript() string {
	// This script will be executed with unshare (root in user namespace)
	return fmt.Sprintf(`
# Create put_old directory for pivot_root
mkdir -p %s/.pivot_root

# Pivot root to new root
pivot_root %s %s/.pivot_root || cd %s

# Change to new root
cd /

# Unmount old root
umount -l /.pivot_root
rmdir /.pivot_root

# Execute the command
`,
		s.rootDir, s.rootDir, s.rootDir, s.rootDir,
	)
}

// stringsJoin joins strings with a separator.
func stringsJoin(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	var resultSb251 strings.Builder
	for _, s := range strs[1:] {
		resultSb251.WriteString(sep + s)
	}
	result += resultSb251.String()

	return result
}

// unmountAll recursively unmounts all mounts under a directory.
func unmountAll(dir string) {
	// Unmount in reverse order of typical mount points
	// Start with deeper paths first
	mounts := []string{
		filepath.Join(dir, "home", "user", ".claude"),
		filepath.Join(dir, "workspace"),
		filepath.Join(dir, "usr", "lib64"),
		filepath.Join(dir, "usr", "lib"),
		filepath.Join(dir, "usr", "bin"),
		filepath.Join(dir, "lib64"),
		filepath.Join(dir, "lib"),
		filepath.Join(dir, "bin"),
		filepath.Join(dir, "proc"),
		filepath.Join(dir, "tmp"),
	}

	for _, m := range mounts {
		_ = unix.Unmount(m, 0) // Ignore errors
	}
}

// DefaultToolPaths returns common tool paths on Linux.
func DefaultToolPaths() []string {
	tools := []string{
		"/usr/bin/git",
		"/bin/git",
		"/usr/local/bin/git",
		"/usr/bin/node",
		"/usr/local/bin/node",
		"/usr/bin/python3",
		"/usr/bin/python",
		"/usr/local/bin/python3",
		"/usr/bin/go",
		"/usr/local/bin/go",
		"/usr/bin/golangci-lint",
		"/usr/local/bin/golangci-lint",
		"/usr/bin/npm",
		"/usr/local/bin/npm",
		"/usr/bin/npx",
		"/usr/local/bin/npx",
	}

	// Filter to only existing tools
	var result []string
	for _, tool := range tools {
		if _, err := os.Stat(tool); err == nil {
			if !slices.Contains(result, tool) {
				result = append(result, tool)
			}
		}
	}

	return result
}
