package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/configcheck"
	"github.com/valksor/kvelmo/pkg/settings"
)

var configCheckJSON bool

var configCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for config drift between global and project settings",
	Long:  "Compare project-level settings against global settings and report differences.",
	RunE:  runConfigCheck,
}

func init() {
	configCheckCmd.Flags().BoolVar(&configCheckJSON, "json", false, "Output as JSON")
	ConfigCmd.AddCommand(configCheckCmd)
}

func runConfigCheck(_ *cobra.Command, _ []string) error {
	effective, global, project, err := settings.LoadEffective("")
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	// Compare project vs global
	globalJSON, err := json.Marshal(global)
	if err != nil {
		return fmt.Errorf("marshal global: %w", err)
	}
	projectJSON, err := json.Marshal(project)
	if err != nil {
		return fmt.Errorf("marshal project: %w", err)
	}

	var globalMap, projectMap map[string]any
	if err := json.Unmarshal(globalJSON, &globalMap); err != nil {
		return fmt.Errorf("parse global: %w", err)
	}
	if err := json.Unmarshal(projectJSON, &projectMap); err != nil {
		return fmt.Errorf("parse project: %w", err)
	}

	drifts := configcheck.Check(globalMap, projectMap)

	if configCheckJSON {
		out, jsonErr := json.MarshalIndent(map[string]any{
			"drifts": drifts,
			"count":  len(drifts),
		}, "", "  ")
		if jsonErr != nil {
			return fmt.Errorf("format: %w", jsonErr)
		}
		fmt.Println(string(out))

		return nil
	}

	if len(drifts) == 0 {
		fmt.Println("No config drift detected. Project and global settings are aligned.")

		return nil
	}

	fmt.Printf("Config drift (%d difference(s)):\n\n", len(drifts))
	for _, d := range drifts {
		fmt.Printf("  %-40s\n", d.Path)
		fmt.Printf("    global:  %v\n", d.Expected)
		fmt.Printf("    project: %v\n", d.Actual)
	}

	// Suppress unused variable warning
	_ = effective

	return nil
}
