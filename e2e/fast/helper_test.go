//go:build e2e_fast
// +build e2e_fast

package e2e_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Helper wraps testing.T and directory for running mehr commands.
type Helper struct {
	t        *testing.T
	dir      string
	lastOut  string
	lastExit int
}

// NewHelper creates a new Helper for the given test and directory.
func NewHelper(t *testing.T, dir string) *Helper {
	return &Helper{t: t, dir: dir}
}

// InitWithLocalConfig copies the local .mehrhof/config.yaml and .env into the test workspace.
func (h *Helper) InitWithLocalConfig() {
	localMehrhofDir := h.findLocalMehrhofDir()
	if localMehrhofDir == "" {
		h.t.Skip("local .mehrhof directory not found")
	}

	testConfigDir := filepath.Join(h.dir, ".mehrhof")
	if err := os.MkdirAll(testConfigDir, 0o755); err != nil {
		h.t.Fatalf("failed to create .mehrhof dir: %v", err)
	}

	localConfig := filepath.Join(localMehrhofDir, "config.yaml")
	testConfig := filepath.Join(testConfigDir, "config.yaml")
	if err := copyFile(localConfig, testConfig); err != nil {
		h.t.Fatalf("failed to copy config.yaml: %v", err)
	}

	localEnv := filepath.Join(localMehrhofDir, ".env")
	if _, err := os.Stat(localEnv); err == nil {
		testEnv := filepath.Join(testConfigDir, ".env")
		if err := copyFile(localEnv, testEnv); err != nil {
			h.t.Fatalf("failed to copy .env: %v", err)
		}
	}
}

func (h *Helper) findLocalMehrhofDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		mehrhofDir := filepath.Join(dir, ".mehrhof")
		if info, err := os.Stat(mehrhofDir); err == nil && info.IsDir() {
			return mehrhofDir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// Run executes a mehr command.
func (h *Helper) Run(args ...string) {
	cmd := exec.Command("mehr", args...)
	cmd.Dir = h.dir
	cmd.Env = append(os.Environ(), "ZAI_API_KEY="+os.Getenv("ZAI_API_KEY"))

	out, err := cmd.CombinedOutput()
	h.lastOut = string(out)

	if exitErr, ok := err.(*exec.ExitError); ok {
		h.lastExit = exitErr.ExitCode()
	} else if err != nil {
		h.lastExit = 1
		h.t.Fatalf("mehr %v: %v\noutput:\n%s", args, err, h.lastOut)
	} else {
		h.lastExit = 0
	}

	h.t.Logf("mehr %v\n%s", args, h.lastOut)
}

// RunWithTimeout executes a mehr command with a timeout.
func (h *Helper) RunWithTimeout(cmd string, timeout time.Duration, args ...string) {
	c := exec.Command("mehr", append([]string{cmd}, args...)...)
	c.Dir = h.dir
	c.Env = append(os.Environ(), "ZAI_API_KEY="+os.Getenv("ZAI_API_KEY"))

	done := make(chan error, 1)
	go func() {
		out, err := c.CombinedOutput()
		h.lastOut = string(out)
		done <- err
	}()

	select {
	case err := <-done:
		if exitErr, ok := err.(*exec.ExitError); ok {
			h.lastExit = exitErr.ExitCode()
		} else if err != nil {
			h.lastExit = 1
		} else {
			h.lastExit = 0
		}
	case <-time.After(timeout):
		c.Process.Kill()
		h.t.Fatalf("command timed out after %v\noutput:\n%s", timeout, h.lastOut)
	}
}

// AssertSuccess checks that the last command exited with code 0.
func (h *Helper) AssertSuccess() {
	if h.lastExit != 0 {
		h.t.Errorf("exit code %d, want 0\noutput:\n%s", h.lastExit, h.lastOut)
	}
}

// AssertOutputContains checks that the last output contains the given substring.
func (h *Helper) AssertOutputContains(s string) {
	if !contains(h.lastOut, s) {
		h.t.Errorf("output does not contain %q\n%s", s, h.lastOut)
	}
}

// AssertFileExists checks that at least one file matching the glob pattern exists.
func (h *Helper) AssertFileExists(pattern string) {
	matches, err := filepath.Glob(filepath.Join(h.dir, pattern))
	if err != nil {
		h.t.Fatalf("glob pattern %q invalid: %v", pattern, err)
	}
	if len(matches) == 0 {
		h.t.Errorf("no files matching %q in %s", pattern, h.dir)
	}
}

// AssertFileContains checks that a file contains the given content.
func (h *Helper) AssertFileContains(file, content string) {
	path := filepath.Join(h.dir, file)
	data, err := os.ReadFile(path)
	if err != nil {
		h.t.Fatalf("failed to read %s: %v", file, err)
	}
	if !contains(string(data), content) {
		h.t.Errorf("%s does not contain %q\ncontent:\n%s", file, content, string(data))
	}
}

// WriteTask writes a task file to the test directory.
func (h *Helper) WriteTask(name, content string) {
	path := filepath.Join(h.dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		h.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		h.t.Fatal(err)
	}
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

// indexOf returns the index of substr in s, or -1 if not found.
func indexOf(s, substr string) int {
	return strings.Index(s, substr)
}
