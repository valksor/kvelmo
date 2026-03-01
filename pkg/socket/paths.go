package socket

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/valksor/kvelmo/pkg/meta"
)

func BaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	return filepath.Join(home, meta.GlobalDir)
}

func GlobalSocketPath() string {
	return filepath.Join(BaseDir(), "global.sock")
}

func GlobalLockPath() string {
	return filepath.Join(BaseDir(), "global.lock")
}

func WorktreeSocketPath(worktreeDir string) string {
	absPath, err := filepath.Abs(worktreeDir)
	if err != nil {
		absPath = worktreeDir
	}

	hash := sha256.Sum256([]byte(absPath))
	hashStr := hex.EncodeToString(hash[:8])

	return filepath.Join(BaseDir(), "worktrees", hashStr+".sock")
}

func WorktreeIDFromPath(worktreeDir string) string {
	absPath, err := filepath.Abs(worktreeDir)
	if err != nil {
		absPath = worktreeDir
	}
	hash := sha256.Sum256([]byte(absPath))

	return hex.EncodeToString(hash[:8])
}

func SocketExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeSocket != 0
}

func EnsureDir() error {
	dirs := []string{
		BaseDir(),
		filepath.Join(BaseDir(), "worktrees"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	return nil
}

func AcquireGlobalLock(lockPath string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}

	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = file.Close()

		return nil, fmt.Errorf("acquire lock: %w", err)
	}

	release := func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}

	return release, nil
}
