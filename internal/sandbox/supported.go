//go:build !linux && !darwin

package sandbox

import (
	"context"
	"fmt"
	"os/exec"
)

// UnsupportedSandbox represents a platform where sandboxing is not supported.
type UnsupportedSandbox struct {
	cfg *Config
}

// newPlatformSandbox creates an error for unsupported platforms.
func newPlatformSandbox(cfg *Config) (Sandbox, error) {
	return nil, fmt.Errorf("sandbox not supported on %s (supported: linux, darwin)", Platform())
}

// Prepare is a no-op for unsupported platforms.
func (s *UnsupportedSandbox) Prepare(ctx context.Context) error {
	return nil
}

// Cleanup is a no-op for unsupported platforms.
func (s *UnsupportedSandbox) Cleanup(ctx context.Context) error {
	return nil
}

// WrapCommand returns an error for unsupported platforms.
func (s *UnsupportedSandbox) WrapCommand(cmd *exec.Cmd) (*exec.Cmd, error) {
	return nil, fmt.Errorf("sandbox not supported on this platform")
}

// DefaultToolPaths returns empty slice on unsupported platforms.
func DefaultToolPaths() []string {
	return nil
}
