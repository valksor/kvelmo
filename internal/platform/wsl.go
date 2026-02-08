// Package platform provides platform-specific detection utilities.
package platform

import (
	"bytes"
	"os"
	"sync"
)

var (
	isWSL     bool
	isWSLOnce sync.Once
)

// procVersionPath is the path to /proc/version. Exported for testing.
var procVersionPath = "/proc/version"

// IsWSL returns true if running inside Windows Subsystem for Linux.
// Result is cached after first call for performance.
func IsWSL() bool {
	isWSLOnce.Do(detectWSL)

	return isWSL
}

// detectWSL performs the actual WSL detection.
func detectWSL() {
	// Fast path: check WSL_DISTRO_NAME env var (set in all WSL versions)
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		isWSL = true

		return
	}

	// Fallback: check /proc/version for "microsoft" or "wsl" (case-insensitive)
	data, err := os.ReadFile(procVersionPath)
	if err != nil {
		// Not WSL if we can't read /proc/version (e.g., container, chroot, non-Linux)
		return
	}

	lower := bytes.ToLower(data)
	isWSL = bytes.Contains(lower, []byte("microsoft")) || bytes.Contains(lower, []byte("wsl"))
}

// ResetWSLDetection resets the cached WSL detection state.
// This is only intended for testing - do not use in production code.
func ResetWSLDetection() {
	isWSLOnce = sync.Once{}
	isWSL = false
}
