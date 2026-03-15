package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/cli"
	"github.com/valksor/kvelmo/pkg/socket"
)

var batchStateFilter string

var BatchCmd = &cobra.Command{
	Use:   "batch <action>",
	Short: "Run an action across all active projects",
	Long: `Execute an action on all active projects matching the optional state filter.

Actions:
  submit    Submit all matching tasks (creates PRs)
  abort     Abort all matching tasks
  reset     Reset all matching tasks
  stop      Stop all matching tasks

Examples:
  kvelmo batch submit --state reviewing   Submit all reviewed tasks
  kvelmo batch stop                       Stop all active tasks
  kvelmo batch abort --state failed       Abort all failed tasks`,
	Args: cobra.ExactArgs(1),
	RunE: runBatch,
}

func init() {
	BatchCmd.Flags().StringVar(&batchStateFilter, "state", "", "Only act on tasks in this state")
}

func runBatch(_ *cobra.Command, args []string) error {
	action := args[0]

	globalPath := socket.GlobalSocketPath()
	if !socket.SocketExists(globalPath) {
		return errors.New("global socket not running (run 'kvelmo serve' first)")
	}

	client, err := socket.NewClient(globalPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connect to global socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	params := map[string]any{
		"action": action,
	}
	if batchStateFilter != "" {
		params["filter"] = map[string]string{"state": batchStateFilter}
	}

	spinner := cli.NewSpinner(fmt.Sprintf("Running batch %s...", action))
	spinner.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "tasks.batch", params)
	if err != nil {
		spinner.Fail("Batch operation failed")

		return fmt.Errorf("batch call: %w", err)
	}

	var result struct {
		Action  string `json:"action"`
		Total   int    `json:"total"`
		Results []struct {
			Path    string `json:"path"`
			State   string `json:"state"`
			Success bool   `json:"success"`
			Error   string `json:"error,omitempty"`
		} `json:"results"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		spinner.Fail("Invalid response")

		return fmt.Errorf("parse result: %w", err)
	}

	if result.Total == 0 {
		spinner.Success("No matching tasks found")

		return nil
	}

	succeeded := 0
	for _, r := range result.Results {
		if r.Success {
			succeeded++
		}
	}

	spinner.Success(fmt.Sprintf("Batch %s: %d/%d succeeded", action, succeeded, result.Total))

	// Print details
	for _, r := range result.Results {
		status := cli.Green.Sprint("ok")
		if !r.Success {
			status = cli.Red.Sprint("fail")
		}
		detail := r.State
		if r.Error != "" {
			detail = r.Error
		}
		fmt.Printf("  [%s] %s (%s)\n", status, r.Path, detail)
	}

	return nil
}
