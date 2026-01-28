package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/security"
	"github.com/valksor/go-mehrhof/internal/storage"
)

var (
	scanDir       string
	scanScanners  []string
	scanOutput    string
	scanFormat    string
	scanFailLevel string
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Run security scans on codebase",
	Long: `Run security scanners to detect vulnerabilities, secrets, and compliance issues.

Scanners auto-detected by project type:
  Cross-language:  semgrep, gitleaks (secrets)
  Go:              gosec (SAST), govulncheck (dependencies)
  JavaScript/TS:   npm-audit (dependencies), eslint-security (SAST)
  Python:          bandit (SAST), pip-audit (dependencies)

By default, runs all applicable scanners for detected project languages.

Exit codes:
  0 - No findings or only below failure threshold
  1 - Critical findings (or configured failure threshold)
  2 - Scanner errors

Examples:
  mehr scan                           # Scan with auto-detected scanners
  mehr scan --scanners gosec,gitleaks # Run specific scanners
  mehr scan --scanners semgrep        # Run cross-language SAST
  mehr scan --dir ./src               # Scan specific directory
  mehr scan --format sarif            # Generate SARIF report
  mehr scan --output report.txt       # Save to file`,
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.GroupID = "utility" // Add to utility group

	scanCmd.Flags().StringVarP(&scanDir, "dir", "d", ".", "Directory to scan")
	scanCmd.Flags().StringSliceVarP(&scanScanners, "scanners", "s", []string{}, "Specific scanners to run (sast, secrets, dependencies)")
	scanCmd.Flags().StringVarP(&scanOutput, "output", "o", "", "Output file path")
	scanCmd.Flags().StringVar(&scanFormat, "format", "text", "Output format (text, sarif, json)")
	scanCmd.Flags().StringVar(&scanFailLevel, "fail-level", "critical", "Failure threshold (critical, high, medium, low, any)")
}

func runScan(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Validate output format
	validFormats := map[string]bool{"text": true, "sarif": true, "json": true}
	if !validFormats[scanFormat] {
		return fmt.Errorf("invalid format '%s': must be one of text, sarif, json", scanFormat)
	}

	// Validate fail level
	validLevels := map[string]bool{"critical": true, "high": true, "medium": true, "low": true, "any": true}
	if !validLevels[scanFailLevel] {
		return fmt.Errorf("invalid fail-level '%s': must be one of critical, high, medium, low, any", scanFailLevel)
	}

	// Initialize conductor
	opts := BuildConductorOptions(CommandOptions{
		Verbose: verbose,
	})
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	// Get workspace config
	ws := cond.GetWorkspace()
	if ws == nil {
		return errors.New("workspace not available")
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create scanner registry
	registry := security.NewScannerRegistry()

	// Initialize tool manager
	var toolMgr *security.ToolManager
	if cfg.Security != nil && cfg.Security.Tools != nil {
		// Get tools directory from config or use default
		toolsDir := cfg.Security.Tools.CacheDir
		autoDownload := cfg.Security.Tools.AutoDownload
		var err error
		toolMgr, err = security.NewToolManager(toolsDir, autoDownload)
		if err != nil {
			return fmt.Errorf("initialize tool manager: %w", err)
		}
	} else {
		// Use default (auto-download enabled)
		var err error
		toolMgr, err = security.NewToolManager("", true)
		if err != nil {
			return fmt.Errorf("initialize tool manager: %w", err)
		}
	}
	registry.SetToolManager(toolMgr)

	// Register scanners from config
	if cfg.Security != nil && cfg.Security.Enabled {
		registerScannersFromConfig(registry, cfg.Security)
	} else {
		// Register default scanners if no config
		registerDefaultScanners(registry)
	}

	// Determine scan directory
	targetDir := scanDir
	if targetDir == "." {
		// Use project root if available
		if root := ws.Root(); root != "" {
			targetDir = root
		}
	}

	// Validate directory path
	if err := validateScanDirectory(targetDir); err != nil {
		return fmt.Errorf("invalid scan directory: %w", err)
	}

	// Validate scanner names
	if len(scanScanners) > 0 {
		validScanners := map[string]bool{
			"gosec":           true,
			"gitleaks":        true,
			"govulncheck":     true,
			"semgrep":         true,
			"npm-audit":       true,
			"eslint-security": true,
			"bandit":          true,
			"pip-audit":       true,
		}
		for _, scanner := range scanScanners {
			if !validScanners[scanner] {
				return fmt.Errorf("invalid scanner '%s': must be one of gosec, gitleaks, govulncheck, semgrep, npm-audit, eslint-security, bandit, pip-audit", scanner)
			}
		}
	}

	// Run scanners
	fmt.Printf("Running security scans on: %s\n\n", targetDir)

	var results []*security.ScanResult
	if len(scanScanners) > 0 {
		results, err = registry.RunEnabled(ctx, targetDir, scanScanners)
	} else {
		results, err = registry.RunAll(ctx, targetDir)
	}

	if err != nil {
		fmt.Printf("Error running scanners: %v\n", err)
	}

	// Display tool manager warnings
	if toolMgr != nil && toolMgr.HasWarnings() {
		fmt.Fprintf(os.Stderr, "\n\u26A0 Warnings:\n")
		for _, warning := range toolMgr.GetWarnings() {
			fmt.Fprintf(os.Stderr, "  - %s\n", warning)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Generate output
	var output string
	switch scanFormat {
	case "sarif":
		report, err := security.GenerateSARIF(results)
		if err != nil {
			return fmt.Errorf("generate SARIF report: %w", err)
		}
		output = security.FormatSARIFReport(report)
	case "json":
		output = security.FormatJSONResults(results)
	default:
		output = security.FormatFindings(results)
	}

	// Save or print output
	if scanOutput != "" {
		if err := saveOutput(scanOutput, output); err != nil {
			return err
		}
		fmt.Printf("\nReport saved to: %s\n", scanOutput)
	} else {
		fmt.Println("\n" + output)
	}

	// Check for blocking findings
	failLevel := security.ParseSeverity(scanFailLevel)
	if security.ShouldBlock(results, failLevel) {
		blocking := security.GetBlockingFindings(results, failLevel)
		fmt.Printf("\n\u001B[31m\u26A0 Found %d finding(s) at or above '%s' severity\n\u001B[0m", len(blocking), failLevel)

		return fmt.Errorf("security scan failed: %d blocking finding(s)", len(blocking))
	}

	return nil
}

// registerScannersFromConfig registers scanners based on workspace config.
func registerScannersFromConfig(registry *security.ScannerRegistry, cfg *storage.SecuritySettings) {
	// Register SAST scanners
	if cfg.Scanners.SAST != nil && cfg.Scanners.SAST.Enabled {
		for _, tool := range cfg.Scanners.SAST.Tools {
			name, ok := tool["name"].(string)
			if !ok {
				continue
			}
			enabled, ok := tool["enabled"].(bool)
			if !ok {
				enabled = true // default to enabled
			}

			if name == "gosec" {
				gosecCfg := &security.GosecConfig{}
				if severity, ok := tool["severity"].(string); ok {
					gosecCfg.Severity = severity
				}
				if confidence, ok := tool["confidence"].(string); ok {
					gosecCfg.Confidence = confidence
				}
				registry.Register("gosec", security.NewGosecScanner(enabled, gosecCfg))
			}
		}
	}

	// Register secret scanners
	if cfg.Scanners.Secrets != nil && cfg.Scanners.Secrets.Enabled {
		for _, tool := range cfg.Scanners.Secrets.Tools {
			name, ok := tool["name"].(string)
			if !ok {
				continue
			}
			enabled, ok := tool["enabled"].(bool)
			if !ok {
				enabled = true // default to enabled
			}

			if name == "gitleaks" {
				gitleaksCfg := &security.GitleaksConfig{}
				if configPath, ok := tool["config_path"].(string); ok {
					gitleaksCfg.ConfigPath = configPath
				}
				if maxDepth, ok := tool["max_depth"].(int); ok {
					gitleaksCfg.MaxDepth = maxDepth
				}
				registry.Register("gitleaks", security.NewGitleaksScanner(enabled, gitleaksCfg))
			}
		}
	}

	// Register dependency scanners
	if cfg.Scanners.Dependencies != nil && cfg.Scanners.Dependencies.Enabled {
		for _, tool := range cfg.Scanners.Dependencies.Tools {
			name, ok := tool["name"].(string)
			if !ok {
				continue
			}
			enabled, ok := tool["enabled"].(bool)
			if !ok {
				enabled = true // default to enabled
			}

			if name == "govulncheck" {
				registry.Register("govulncheck", security.NewGovulncheckScanner(enabled))
			}
		}
	}
}

// registerDefaultScanners registers all scanners with default settings based on detected project.
func registerDefaultScanners(registry *security.ScannerRegistry) {
	// Enable all scanners by default - detect project and register appropriate scanners
	// Note: This is called when no security config is present, so we use default settings

	// Always register cross-language scanners
	registry.Register("gitleaks", security.NewGitleaksScanner(true, &security.GitleaksConfig{}))
	registry.Register("semgrep", security.NewSemgrepScanner(true, &security.SemgrepConfig{}))

	// Register Go scanners (will skip if not a Go project at scan time)
	registry.Register("gosec", security.NewGosecScanner(true, &security.GosecConfig{}))
	registry.Register("govulncheck", security.NewGovulncheckScanner(true))

	// Register JavaScript/TypeScript scanners
	registry.Register("npm-audit", security.NewNpmAuditScanner(true, &security.NpmAuditConfig{}))
	registry.Register("eslint-security", security.NewESLintScanner(true, &security.ESLintConfig{}))

	// Register Python scanners
	registry.Register("bandit", security.NewBanditScanner(true, &security.BanditConfig{}))
	registry.Register("pip-audit", security.NewPipAuditScanner(true, &security.PipAuditConfig{}))
}

// saveOutput saves output to file.
func saveOutput(path string, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// validateScanDirectory validates that the scan directory is safe to use.
func validateScanDirectory(dir string) error {
	// Convert to absolute path
	abs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid directory path: %w", err)
	}

	// Check if directory exists
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("cannot access directory: %w", err)
	}

	// Check if it's actually a directory
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", dir)
	}

	// Check for directory traversal attempts
	if strings.Contains(abs, "..") {
		return fmt.Errorf("directory traversal detected: %s", dir)
	}

	return nil
}
