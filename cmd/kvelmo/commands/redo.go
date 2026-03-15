package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/socket"
)

var RedoCmd = &cobra.Command{
	Use:     "redo",
	Aliases: []string{"r"},
	Short: "Redo to the next checkpoint",
	Long: `Restores the working directory to the next checkpoint in the redo stack.
Only available after using 'undo'.`,
	RunE: runRedo,
}

func init() {
	RedoCmd.Flags().IntP("steps", "n", 1, "Number of checkpoints to redo")
}

func runRedo(cmd *cobra.Command, args []string) error {
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
	result, err := client.Call(ctx, "redo", params)
	if err != nil {
		return fmt.Errorf("redo: %w", err)
	}

	fmt.Printf("Redo: %v\n", result)

	return nil
}
