package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/agent"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/settings"
	"github.com/valksor/kvelmo/pkg/socket"
)

var diagnoseJSON bool

var DiagnoseCmd = &cobra.Command{
	Use:     "diagnose",
	Aliases: []string{"diag"},
	Short:   "Check system setup and configuration",
	Long: `Diagnose checks that kvelmo is properly configured.

It verifies:
  - Git is installed
  - AI agent CLIs are available (claude, codex)
  - Global socket is running
  - Provider tokens are configured

Run this command to troubleshoot setup issues.`,
	RunE: runDiagnose,
}

func init() {
	DiagnoseCmd.Flags().BoolVar(&diagnoseJSON, "json", false, "Output raw JSON response")
}

type diagnoseCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Fix    string `json:"fix,omitempty"`
}

type diagnoseProvider struct {
	Name       string `json:"name"`
	Configured bool   `json:"configured"`
}

type diagnoseResult struct {
	Checks       []diagnoseCheck    `json:"checks"`
	GlobalSocket string             `json:"global_socket"`
	Providers    []diagnoseProvider `json:"providers"`
	Issues       []string           `json:"issues,omitempty"`
}

func runDiagnose(cmd *cobra.Command, args []string) error {
	var issues []string

	// Run preflight checks for git and agent CLIs
	preflight := agent.RunPreflight()

	var jsonChecks []diagnoseCheck

	for _, c := range preflight.Checks {
		jc := diagnoseCheck{
			Name:   c.Name,
			Status: string(c.Status),
			Detail: c.Detail,
			Fix:    c.Fix,
		}
		jsonChecks = append(jsonChecks, jc)
		if c.Fix != "" {
			issues = append(issues, c.Fix)
		}
	}

	// Check global socket
	globalPath := socket.GlobalSocketPath()
	socketStatus := "not_running"
	if socket.SocketExists(globalPath) {
		client, err := socket.NewClient(globalPath, socket.WithTimeout(500*time.Millisecond))
		if err == nil {
			_ = client.Close()
			socketStatus = "running"
		} else {
			socketStatus = "stale"
			issues = append(issues, "Remove stale socket: rm "+globalPath)
		}
	} else {
		issues = append(issues, fmt.Sprintf("Start server: %s serve", meta.Name))
	}

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

	var jsonProviders []diagnoseProvider
	for _, p := range providerChecks {
		configured := detectExistingToken(p.envVar, settings.ScopeGlobal, "") != nil
		jsonProviders = append(jsonProviders, diagnoseProvider{
			Name:       p.name,
			Configured: configured,
		})
	}

	if diagnoseJSON {
		result := diagnoseResult{
			Checks:       jsonChecks,
			GlobalSocket: socketStatus,
			Providers:    jsonProviders,
			Issues:       issues,
		}
		out, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Println(string(out))

		return nil
	}

	// Formatted output
	fmt.Println()
	fmt.Printf("  %s Diagnostics\n", meta.Name)
	fmt.Println("  ─────────────────────────────────────")
	fmt.Println()

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
	}

	switch socketStatus {
	case "running":
		fmt.Printf("  Global socket: ✓ running\n")
	case "stale":
		fmt.Printf("  Global socket: ⚠ stale (not responding)\n")
	default:
		fmt.Printf("  Global socket: ✗ not running\n")
	}

	fmt.Println()
	fmt.Println("  Providers:")

	for _, p := range providerChecks {
		if token := detectExistingToken(p.envVar, settings.ScopeGlobal, ""); token != nil {
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
