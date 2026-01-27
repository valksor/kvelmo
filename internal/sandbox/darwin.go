//go:build darwin

package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// DarwinSandbox uses macOS sandbox-exec for isolation.
type DarwinSandbox struct {
	cfg     *Config
	profile string // Generated SBPL profile
}

// newPlatformSandbox creates a macOS sandbox implementation.
func newPlatformSandbox(cfg *Config) (Sandbox, error) {
	if cfg.HomeDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
		cfg.HomeDir = homeDir
	}

	return &DarwinSandbox{cfg: cfg}, nil
}

// Prepare generates the SBPL profile and writes it to a temp file.
func (s *DarwinSandbox) Prepare(ctx context.Context) error {
	s.profile = s.buildSBPLProfile()
	return nil
}

// Cleanup removes any temporary resources.
func (s *DarwinSandbox) Cleanup(ctx context.Context) error {
	// No temp files to clean up - profile is inline
	return nil
}

// WrapCommand wraps the command with sandbox-exec.
func (s *DarwinSandbox) WrapCommand(cmd *exec.Cmd) (*exec.Cmd, error) {
	// If a custom profile is provided, write it to a temp file
	profilePath := ""
	if s.cfg.Profile != "" {
		var err error
		profilePath, err = s.writeProfileFile(s.cfg.Profile)
		if err != nil {
			return nil, fmt.Errorf("write custom profile: %w", err)
		}
	} else {
		// Use inline profile
		profilePath = s.profile
	}

	// Build sandbox-exec command
	newCmd := exec.Command("sandbox-exec")
	if profilePath != "" {
		newCmd.Args = []string{"sandbox-exec", "-f", profilePath}
	} else {
		newCmd.Args = []string{"sandbox-exec", "-p", s.buildSBPLProfile()}
	}
	newCmd.Args = append(newCmd.Args, cmd.Path)
	newCmd.Args = append(newCmd.Args, cmd.Args[1:]...)

	// Copy I/O, environment, and directory
	newCmd.Stdin = cmd.Stdin
	newCmd.Stdout = cmd.Stdout
	newCmd.Stderr = cmd.Stderr
	newCmd.Dir = cmd.Dir
	newCmd.Env = cmd.Env

	return newCmd, nil
}

// buildSBPLProfile generates an SBPL (Scheme-like) profile for sandbox-exec.
func (s *DarwinSandbox) buildSBPLProfile() string {
	var b strings.Builder

	b.WriteString("(version 1)\n")
	b.WriteString("(deny default)\n")

	// File system access
	projectDir := s.cfg.ProjectDir
	homeDir := s.cfg.HomeDir
	claudeDir := filepath.Join(homeDir, ".claude")

	b.WriteString(";; File system access\n")
	b.WriteString(fmt.Sprintf("(allow file-read* file-write-create file-write-data\n"+
		"    (literal \"/tmp\")\n"+
		"    (literal \"%s\")\n"+
		"    (literal \"%s\")\n"+
		"    (subpath \"%s\")\n"+
		"    (subpath \"%s\")\n"+
		"    (subpath \"/tmp\"))\n",
		sanitizePath(projectDir),
		sanitizePath(claudeDir),
		sanitizePath(projectDir),
		sanitizePath(claudeDir),
	))

	// Shared libraries
	b.WriteString(";; Shared libraries\n")
	b.WriteString("(allow file-read* (regex #\"\\.dylib$\"))\n")
	b.WriteString("(allow file-read* (regex #\"\\.so$\"))\n")

	// Device files
	b.WriteString(";; Device files\n")
	b.WriteString("(allow file-read* (literal \"/dev/null\"))\n")
	b.WriteString("(allow file-read* (literal \"/dev/zero\"))\n")
	b.WriteString("(allow file-read* (literal \"/dev/random\"))\n")
	b.WriteString("(allow file-read* (literal \"/dev/urandom\"))\n")

	// Network access (required for LLM API calls)
	if s.cfg.Network {
		b.WriteString(";; Network (required for LLM API calls)\n")
		b.WriteString("(allow network* (remote unix))\n")
		b.WriteString("(allow network-outbound (remote tcp))\n")
		b.WriteString("(allow network-outbound (remote udp))\n")
		b.WriteString("(allow dns)\n")
	}

	// Process execution - common tools
	b.WriteString(";; Process execution\n")
	s.writeExecRules(&b)

	return b.String()
}

// writeExecRules writes SBPL process execution rules for common tools.
func (s *DarwinSandbox) writeExecRules(b *strings.Builder) {
	// Standard shell
	b.WriteString("(allow process-exec (literal \"/bin/bash\"))\n")
	b.WriteString("(allow process-exec (literal \"/bin/sh\"))\n")
	b.WriteString("(allow process-exec (literal \"/bin/zsh\"))\n")
	b.WriteString("(allow process-exec (literal \"/usr/bin/env\"))\n")

	// Common locations
	b.WriteString("(allow process-exec (regex #\"^/usr/local/bin/\"))\n")
	b.WriteString("(allow process-exec (regex #\"^/usr/bin/\"))\n")
	b.WriteString("(allow process-exec (regex #\"^/bin/\"))\n")
	b.WriteString("(allow process-exec (regex #\"^/opt/homebrew/bin/\"))\n")

	// Common tools
	commonTools := []string{
		"/usr/local/bin/git", "/usr/bin/git",
		"/usr/local/bin/node", "/usr/bin/node",
		"/usr/local/bin/npm", "/usr/bin/npm",
		"/usr/local/bin/python3", "/usr/bin/python3", "/usr/local/bin/python", "/usr/bin/python",
		"/usr/local/bin/go", "/usr/bin/go",
		"/usr/local/bin/golangci-lint",
	}
	for _, tool := range commonTools {
		if _, err := os.Stat(tool); err == nil {
			b.WriteString(fmt.Sprintf("(allow process-exec (literal \"%s\"))\n", tool))
		}
	}

	// Custom tools from config
	for _, tool := range s.cfg.Tools {
		b.WriteString(fmt.Sprintf("(allow process-exec (literal \"%s\"))\n", tool))
	}
}

// writeProfileFile writes the profile to a temporary file.
func (s *DarwinSandbox) writeProfileFile(profile string) (string, error) {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "mehrhof-sandbox.sb")
	if err := os.WriteFile(tmpFile, []byte(profile), 0o644); err != nil {
		return "", fmt.Errorf("write profile file: %w", err)
	}
	return tmpFile, nil
}

// sanitizePath escapes a path for use in SBPL.
func sanitizePath(p string) string {
	// Escape backslashes and quotes
	p = strings.ReplaceAll(p, "\\", "\\\\")
	p = strings.ReplaceAll(p, "\"", "\\\"")
	return p
}

// DefaultToolPaths returns common tool paths on macOS.
func DefaultToolPaths() []string {
	tools := []string{
		"/usr/bin/git",
		"/usr/local/bin/git",
		"/opt/homebrew/bin/git",
		"/usr/bin/node",
		"/usr/local/bin/node",
		"/opt/homebrew/bin/node",
		"/usr/bin/python3",
		"/usr/local/bin/python3",
		"/opt/homebrew/bin/python3",
		"/usr/bin/go",
		"/usr/local/bin/go",
		"/opt/homebrew/bin/go",
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
