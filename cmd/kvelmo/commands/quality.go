package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

// QualityCmd is the parent command for quality gate controls.
var QualityCmd = &cobra.Command{
	Use:   "quality",
	Short: "Quality gate controls",
	Long:  "Commands for interacting with the quality gate during task review.",
}

var qualityRespondCmd = &cobra.Command{
	Use:   "respond",
	Short: "Answer a pending quality gate prompt",
	Long: `Answer a pending quality gate prompt by providing a yes/no response.

The prompt ID is shown in 'kvelmo status' when a quality gate question is waiting.`,
	RunE: runQualityRespond,
}

func init() {
	qualityRespondCmd.Flags().String("prompt-id", "", "Prompt ID to respond to (required)")
	qualityRespondCmd.Flags().Bool("yes", false, "Answer yes")
	qualityRespondCmd.Flags().Bool("no", false, "Answer no")
	_ = qualityRespondCmd.MarkFlagRequired("prompt-id")
	QualityCmd.AddCommand(qualityRespondCmd)
}

func runQualityRespond(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	socketPath := socket.WorktreeSocketPath(cwd)
	if !socket.SocketExists(socketPath) {
		return errors.New("no worktree socket running\nRun '" + meta.Name + " start' first")
	}

	client, err := socket.NewClient(socketPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connect to socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	promptID, _ := cmd.Flags().GetString("prompt-id")
	yes, _ := cmd.Flags().GetBool("yes")
	no, _ := cmd.Flags().GetBool("no")

	if !yes && !no {
		return errors.New("must specify --yes or --no")
	}
	if yes && no {
		return errors.New("cannot specify both --yes and --no")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Call(ctx, "quality.respond", map[string]any{
		"prompt_id": promptID,
		"answer":    yes,
	})
	if err != nil {
		return fmt.Errorf("quality respond: %w", err)
	}

	if yes {
		fmt.Println("Answered: yes")
	} else {
		fmt.Println("Answered: no")
	}

	return nil
}
