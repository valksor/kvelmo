package commands

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/settings"
	"github.com/valksor/kvelmo/pkg/socket"
)

var DiagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Check system setup and configuration",
	Long: `Diagnose checks that kvelmo is properly configured.

It verifies:
  - Git is installed
  - AI agent CLIs are available (claude, codex)
  - Global socket is running
  - Provider tokens are configured

Run this command to troubleshoot setup issues.`,
	RunE: runDiagnose,
}

func runDiagnose(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Printf("  %s Diagnostics\n", meta.Name)
	fmt.Println("  ─────────────────────────────────────")
	fmt.Println()

	var issues []string

	// Check git
	if gitVersion, err := getCommandVersion("git", "--version"); err == nil {
		fmt.Printf("  Git:           ✓ installed (%s)\n", gitVersion)
	} else {
		fmt.Printf("  Git:           ✗ not found\n")
		issues = append(issues, "Install git: https://git-scm.com/downloads")
	}

	// Check Claude CLI
	claudeAvailable := false
	if claudeVersion, err := getCommandVersion("claude", "--version"); err == nil {
		fmt.Printf("  Claude CLI:    ✓ installed (%s)\n", claudeVersion)
		claudeAvailable = true
	} else {
		fmt.Printf("  Claude CLI:    ✗ not found\n")
		issues = append(issues, "Install Claude CLI: https://docs.anthropic.com/en/docs/claude-code/getting-started")
	}

	// Check Codex CLI
	if codexVersion, err := getCommandVersion("codex", "--version"); err == nil {
		fmt.Printf("  Codex CLI:     ✓ installed (%s)\n", codexVersion)
	} else {
		fmt.Printf("  Codex CLI:     ✗ not found\n")
		// Only flag as issue if Claude is also missing (need at least one agent)
		if !claudeAvailable {
			issues = append(issues, "Install at least one AI agent (Claude or Codex)")
		}
	}

	// Check global socket
	globalPath := socket.GlobalSocketPath()
	if socket.SocketExists(globalPath) {
		// Try to connect to verify it's responsive
		client, err := socket.NewClient(globalPath, socket.WithTimeout(500*time.Millisecond))
		if err == nil {
			_ = client.Close()
			fmt.Printf("  Global socket: ✓ running\n")
		} else {
			fmt.Printf("  Global socket: ⚠ stale (not responding)\n")
			issues = append(issues, "Remove stale socket: rm "+globalPath)
		}
	} else {
		fmt.Printf("  Global socket: ✗ not running\n")
		issues = append(issues, fmt.Sprintf("Start server: %s serve", meta.Name))
	}

	fmt.Println()
	fmt.Println("  Providers:")

	// Check provider tokens
	providerChecks := []struct {
		name   string
		envVar string
	}{
		{"GitHub", "GITHUB_TOKEN"},
		{"GitLab", "GITLAB_TOKEN"},
		{"Linear", "LINEAR_TOKEN"},
		{"Wrike", "WRIKE_TOKEN"},
	}

	for _, p := range providerChecks {
		if token := detectExistingToken(p.envVar, settings.ScopeGlobal, ""); token != nil {
			// Mask token - only show last 4 chars for verification
			masked := "****"
			if len(token.Value) > 4 {
				masked = "****" + token.Value[len(token.Value)-4:]
			}
			fmt.Printf("    %-8s ✓ configured (%s)\n", p.name+":", masked)
		} else {
			fmt.Printf("    %-8s ✗ not configured\n", p.name+":")
		}
	}

	fmt.Println()

	// Print issues and next steps
	if len(issues) > 0 {
		fmt.Println("  Next steps:")
		for _, issue := range issues {
			fmt.Printf("    • %s\n", issue)
		}
		fmt.Println()
	} else {
		fmt.Println("  ✓ All checks passed!")
		fmt.Println()
	}

	return nil
}

// getCommandVersion runs a command and extracts a version string from output.
func getCommandVersion(name string, args ...string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Extract version from output (first line, cleaned up)
	version := strings.TrimSpace(string(output))
	if idx := strings.Index(version, "\n"); idx > 0 {
		version = version[:idx]
	}

	// Try to extract just the version number
	version = extractVersion(version)

	return version, nil
}

// extractVersion attempts to extract a version number from a string.
func extractVersion(s string) string {
	// Common patterns: "git version 2.43.0", "claude 1.2.3", etc.
	s = strings.TrimSpace(s)

	// Remove common prefixes
	prefixes := []string{"git version ", "claude ", "codex "}
	for _, p := range prefixes {
		if strings.HasPrefix(strings.ToLower(s), strings.ToLower(p)) {
			s = strings.TrimSpace(s[len(p):])

			break
		}
	}

	// Take first word if it looks like a version
	if idx := strings.Index(s, " "); idx > 0 {
		candidate := s[:idx]
		if looksLikeVersion(candidate) {
			return candidate
		}
	}

	// Return as-is if short enough
	if len(s) <= 20 {
		return s
	}

	return s[:20] + "..."
}

// looksLikeVersion checks if a string looks like a version number.
func looksLikeVersion(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Version usually starts with a digit
	return s[0] >= '0' && s[0] <= '9'
}
