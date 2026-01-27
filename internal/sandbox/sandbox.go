package sandbox

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
)

// Sandbox defines platform-specific isolation capabilities.
type Sandbox interface {
	// WrapCommand wraps an exec.Cmd with sandbox execution.
	// The returned command should execute the original command
	// within the isolated environment.
	WrapCommand(cmd *exec.Cmd) (*exec.Cmd, error)

	// Prepare sets up the sandbox environment before execution.
	// This is called before WrapCommand to create any necessary
	// temporary files, directories, or system resources.
	Prepare(ctx context.Context) error

	// Cleanup removes any temporary sandbox resources.
	// This is always called, even if the command fails.
	Cleanup(ctx context.Context) error
}

// New creates a platform-specific sandbox based on the current OS.
// Returns an error if sandboxing is not supported on this platform.
// When sandbox is disabled, returns (nil, nil) - caller should check cfg.Enabled first.
func New(cfg *Config) (Sandbox, error) {
	if !cfg.Enabled {
		return nil, nil //nolint:nilnil // Sandbox disabled, not an error
	}

	if cfg.ProjectDir == "" {
		return nil, errors.New("sandbox: project directory is required")
	}

	// Platform-specific implementations are selected via build tags
	return newPlatformSandbox(cfg)
}

// Supported returns true if the current platform supports sandboxing.
func Supported() bool {
	return runtime.GOOS == "linux" || runtime.GOOS == "darwin"
}

// Platform returns the current platform name.
func Platform() string {
	return runtime.GOOS
}
