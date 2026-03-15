package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/cli"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

var planForce bool

var PlanCmd = &cobra.Command{
	Use:     "plan",
	Aliases: []string{"pl"},
	Short:   "Start planning phase for current task",
	Long:    "Submit a planning job to the worker pool for the current task.",
	RunE:    runPlan,
}

func init() {
	PlanCmd.Flags().BoolVar(&planForce, "force", false, "Re-run planning even if already planned")
}

func runPlan(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	wtPath := socket.WorktreeSocketPath(cwd)

	if !socket.SocketExists(wtPath) {
		return errors.New("no worktree socket running\nRun '" + meta.Name + " start' first")
	}

	spinner := cli.NewSpinner("Submitting plan job...")
	spinner.Start()

	client, err := socket.NewClient(wtPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		spinner.Fail("Connection failed")

		return fmt.Errorf("connect to worktree socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := map[string]any{
		"force": planForce,
	}

	// Submit plan job
	resp, err := client.Call(ctx, "plan", params)
	if err != nil {
		spinner.Fail("Plan submission failed")

		return fmt.Errorf("plan call: %w", err)
	}

	var result PlanResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		spinner.Fail("Invalid response")

		return fmt.Errorf("parse result: %w", err)
	}

	spinner.Success("Planning job submitted: " + result.JobID)
	fmt.Println("Use '" + meta.Name + " status' to check progress")

	return nil
}

type PlanResult struct {
	JobID string `json:"job_id"`
}
