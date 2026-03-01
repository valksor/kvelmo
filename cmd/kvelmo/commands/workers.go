package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

var WorkersCmd = &cobra.Command{
	Use:   "workers",
	Short: "List worker pool status",
	Long:  "Query the global socket for worker pool status.",
	RunE:  runWorkers,
}

func runWorkers(cmd *cobra.Command, args []string) error {
	globalPath := socket.GlobalSocketPath()

	if !socket.SocketExists(globalPath) {
		return errors.New("global socket not running\nRun '" + meta.Name + " start' first")
	}

	client, err := socket.NewClient(globalPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connect to global socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "workers.list", nil)
	if err != nil {
		return fmt.Errorf("list workers: %w", err)
	}

	var result WorkersListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse workers: %w", err)
	}

	if len(result.Workers) == 0 {
		fmt.Println("No workers running")
		fmt.Printf("\nPool: %d/%d workers, %d queued jobs\n",
			result.Stats.WorkingWorkers,
			result.Stats.TotalWorkers,
			result.Stats.QueuedJobs)

		return nil
	}

	fmt.Println("Workers:")
	fmt.Println("--------")
	for _, w := range result.Workers {
		status := "○"
		if w.Status == "working" {
			status = "●"
		}
		fmt.Printf("  %s %s", status, w.ID)
		if w.CurrentJob != "" {
			fmt.Printf(" → %s", w.CurrentJob)
		}
		fmt.Println()
	}

	fmt.Printf("\nPool: %d/%d active, %d queued\n",
		result.Stats.WorkingWorkers,
		result.Stats.TotalWorkers,
		result.Stats.QueuedJobs)

	return nil
}

type WorkerInfo struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	CurrentJob string `json:"current_job,omitempty"`
}

type WorkersStats struct {
	TotalWorkers     int `json:"total_workers"`
	AvailableWorkers int `json:"available_workers"`
	WorkingWorkers   int `json:"working_workers"`
	QueuedJobs       int `json:"queued_jobs"`
}

type WorkersListResult struct {
	Workers []WorkerInfo `json:"workers"`
	Stats   WorkersStats `json:"stats"`
}
