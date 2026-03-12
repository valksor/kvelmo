package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show task artifacts (specs, plans)",
	Long: `Display specifications or plans for the current task.

Subcommands:
  show spec    Show all specification files
  show plan    Show the plan (alias for show spec)`,
}

var showSpecCmd = &cobra.Command{
	Use:   "spec",
	Short: "Show specification files for the current task",
	RunE:  runShowSpec,
}

var showPlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Show the plan for the current task",
	Long:  "Plans are stored as specification files. This is an alias for 'show spec'.",
	RunE:  runShowPlan,
}

func init() {
	ShowCmd.PersistentFlags().Bool("json", false, "Output raw JSON response")

	ShowCmd.AddCommand(showSpecCmd)
	ShowCmd.AddCommand(showPlanCmd)
}

func runShowSpec(cmd *cobra.Command, _ []string) error {
	return showArtifact(cmd, "show.spec")
}

func runShowPlan(cmd *cobra.Command, _ []string) error {
	return showArtifact(cmd, "show.plan")
}

func showArtifact(cmd *cobra.Command, method string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	wtPath := socket.WorktreeSocketPath(cwd)
	if !socket.SocketExists(wtPath) {
		return fmt.Errorf("no worktree socket running for %s\nRun '%s start' first", cwd, meta.Name)
	}

	client, err := socket.NewClient(wtPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connect to worktree socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, method, nil)
	if err != nil {
		return fmt.Errorf("%s call: %w", method, err)
	}

	jsonFlag, _ := cmd.Flags().GetBool("json")
	if jsonFlag {
		var pretty interface{}
		if jsonErr := json.Unmarshal(resp.Result, &pretty); jsonErr != nil {
			fmt.Println(string(resp.Result))

			return nil
		}
		out, jsonErr := json.MarshalIndent(pretty, "", "  ")
		if jsonErr != nil {
			fmt.Println(string(resp.Result))

			return nil
		}
		fmt.Println(string(out))

		return nil
	}

	var result socket.ShowSpecResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if len(result.Specifications) == 0 {
		fmt.Println("No specifications found for the current task.")

		return nil
	}

	for i, spec := range result.Specifications {
		if i > 0 {
			fmt.Print("\n---\n\n")
		}
		fmt.Printf("# %s\n\n", spec.Path)
		fmt.Println(spec.Content)
	}

	return nil
}
