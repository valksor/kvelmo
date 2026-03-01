package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/socket"
)

var UndoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Undo to the previous checkpoint",
	Long: `Reverts the working directory to the previous checkpoint.
Checkpoints are created after each agent operation (plan, implement).
Use 'redo' to restore undone checkpoints.`,
	RunE: runUndo,
}

func init() {
	UndoCmd.Flags().IntP("steps", "n", 1, "Number of checkpoints to undo")
}

func runUndo(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	socketPath := socket.WorktreeSocketPath(cwd)

	client, err := socket.NewClient(socketPath, socket.WithTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("connect to socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	steps, _ := cmd.Flags().GetInt("steps")

	params := map[string]any{
		"steps": steps,
	}

	ctx := context.Background()
	result, err := client.Call(ctx, "undo", params)
	if err != nil {
		return fmt.Errorf("undo: %w", err)
	}

	fmt.Printf("Undo: %v\n", result)

	return nil
}
