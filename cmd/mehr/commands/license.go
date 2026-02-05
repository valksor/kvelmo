package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/valksor/go-toolkit/display"
	"github.com/valksor/go-toolkit/licensing"
)

var (
	licenseJSON        bool
	licenseUnknownOnly bool
)

var licenseCmd = &cobra.Command{
	Use:     "license",
	Short:   "Display license information",
	GroupID: "info",
	Long: `Display license information for Mehrhof and its dependencies.

Examples:
  mehr license           # Show mehrhof license
  mehr license info      # Show all dependency licenses
  mehr license info --json      # Output as JSON
  mehr license info --unknown-only  # Show only unknown licenses`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(licensing.GetProjectLicense())

		return nil
	},
}

var licenseInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "List all dependency licenses",
	Long: `List all Go module dependencies with their detected SPDX license types.

The license detection uses github.com/google/go-licenses for accurate
SPDX license identification. Dependencies are scanned from the go.mod
file in the current directory.`,
	RunE: runLicenseInfo,
}

func init() {
	rootCmd.AddCommand(licenseCmd)
	licenseCmd.AddCommand(licenseInfoCmd)

	licenseInfoCmd.Flags().BoolVar(&licenseJSON, "json", false, "Output as JSON")
	licenseInfoCmd.Flags().BoolVar(&licenseUnknownOnly, "unknown-only", false, "Only show packages with unknown licenses")
}

func runLicenseInfo(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Get dependency licenses
	libs, err := licensing.GetDependencyLicenses(ctx, ".")
	if err != nil {
		return fmt.Errorf("get dependency licenses: %w", err)
	}

	// Filter for unknown-only if requested
	if licenseUnknownOnly {
		filtered := make([]licensing.PackageLicense, 0)
		for _, lib := range libs {
			if lib.Unknown {
				filtered = append(filtered, lib)
			}
		}
		libs = filtered
	}

	// JSON output
	if licenseJSON {
		type jsonLicense struct {
			Path    string `json:"path"`
			License string `json:"license"`
			Unknown bool   `json:"unknown"`
		}
		type jsonOutput struct {
			Licenses []jsonLicense `json:"licenses"`
			Count    int           `json:"count"`
		}

		out := jsonOutput{
			Licenses: make([]jsonLicense, len(libs)),
			Count:    len(libs),
		}
		for i, lib := range libs {
			out.Licenses[i] = jsonLicense{
				Path:    lib.Path,
				License: lib.License,
				Unknown: lib.Unknown,
			}
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(out)
	}

	// Text output
	if len(libs) == 0 {
		fmt.Println("No dependencies found.")

		return nil
	}

	fmt.Printf("Dependency Licenses (%d):\n", len(libs))
	fmt.Println()

	// Get terminal width for truncation
	_, width, _ := term.GetSize(int(os.Stdout.Fd()))
	maxPathLen := 50
	if width > 0 && width > 80 {
		maxPathLen = width - 40
	}

	for _, lib := range libs {
		path := lib.Path
		if len(path) > maxPathLen {
			path = path[:maxPathLen-3] + "..."
		}

		if lib.Unknown {
			fmt.Printf("  %s %s\n", display.Bold(path), display.Muted("("+lib.License+")"))
		} else {
			fmt.Printf("  %s %s\n", path, display.Muted(lib.License))
		}
	}

	// Count unknown licenses
	unknownCount := 0
	for _, lib := range libs {
		if lib.Unknown {
			unknownCount++
		}
	}
	if unknownCount > 0 {
		fmt.Printf("\n%d package(s) with unknown licenses.\n", unknownCount)
	}

	return nil
}
