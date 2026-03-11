package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/agent"
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

	// Run preflight checks for git and agent CLIs
	preflight := agent.RunPreflight()
	for _, c := range preflight.Checks {
		symbol := "✓"
		label := "installed"

		switch c.Status {
		case agent.CheckPassed:
			// default values are fine
		case agent.CheckFailed:
			symbol = "✗"
			label = "not found"
		case agent.CheckWarning:
			symbol = "⚠"
			label = "not found"
		}
		detail := ""
		if c.Status == agent.CheckPassed && c.Detail != "" {
			detail = " (" + c.Detail + ")"
		}
		// Map check names to display labels
		displayName := c.Name
		switch c.Name {
		case "claude":
			displayName = "Claude CLI"
		case "codex":
			displayName = "Codex CLI"
		case "git":
			displayName = "Git"
		}
		fmt.Printf("  %-14s %s %s%s\n", displayName+":", symbol, label, detail)
		if c.Fix != "" {
			issues = append(issues, c.Fix)
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
