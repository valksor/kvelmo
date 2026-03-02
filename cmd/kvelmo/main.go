package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/cmd/kvelmo/commands"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/watchdog"
)

var rootCmd = &cobra.Command{
	Use:   "kvelmo",
	Short: "Task lifecycle orchestrator",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s %s (%s) built %s\n", meta.Name, meta.Version, meta.Commit, meta.BuildTime)
	},
}

func init() {
	rootCmd.Long = meta.Name + ": Socket-first task lifecycle orchestration for AI-assisted development.\n\nTask States:\n  None         No active task\n  Loaded       Task fetched, branch created\n  Planning     Agent generating specification\n  Planned      Specification complete\n  Implementing Agent writing code\n  Implemented  Code complete, ready for review\n  Optimizing   Agent improving code quality (optional)\n  Reviewing    Human review in progress\n  Submitted    PR created"

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(commands.ServeCmd)
	rootCmd.AddCommand(commands.StartCmd)
	rootCmd.AddCommand(commands.StatusCmd)
	rootCmd.AddCommand(commands.StopCmd)
	rootCmd.AddCommand(commands.ProjectsCmd)
	rootCmd.AddCommand(commands.WorkersCmd)
	rootCmd.AddCommand(commands.PlanCmd)
	rootCmd.AddCommand(commands.ImplementCmd)
	rootCmd.AddCommand(commands.SimplifyCmd)
	rootCmd.AddCommand(commands.OptimizeCmd)
	rootCmd.AddCommand(commands.ReviewCmd)
	rootCmd.AddCommand(commands.SubmitCmd)
	rootCmd.AddCommand(commands.FinishCmd)
	rootCmd.AddCommand(commands.RefreshCmd)
	rootCmd.AddCommand(commands.UndoCmd)
	rootCmd.AddCommand(commands.RedoCmd)
	rootCmd.AddCommand(commands.ConfigCmd)
	rootCmd.AddCommand(commands.CompletionCmd)
	rootCmd.AddCommand(commands.BrowserCmd)
	rootCmd.AddCommand(commands.AbortCmd)
	rootCmd.AddCommand(commands.ResetCmd)
	rootCmd.AddCommand(commands.ChatCmd)
	rootCmd.AddCommand(commands.CheckpointsCmd)
	rootCmd.AddCommand(commands.GitCmd)
	rootCmd.AddCommand(commands.ScreenshotsCmd)
	rootCmd.AddCommand(commands.MemoryCmd)

	// Core feature commands
	rootCmd.AddCommand(commands.AbandonCmd)
	rootCmd.AddCommand(commands.DeleteCmd)
	rootCmd.AddCommand(commands.UpdateCmd)
	rootCmd.AddCommand(commands.ListCmd)
	rootCmd.AddCommand(commands.FilesCmd)
	rootCmd.AddCommand(commands.BrowseCmd)
	rootCmd.AddCommand(commands.JobsCmd)
	rootCmd.AddCommand(commands.PipeCmd)
	rootCmd.AddCommand(commands.RecordingsCmd)
	rootCmd.AddCommand(commands.DiagnoseCmd)
	rootCmd.AddCommand(commands.CleanupCmd)

	// Provider commands (login, etc.)
	rootCmd.AddCommand(commands.GitHubCmd)
	rootCmd.AddCommand(commands.GitLabCmd)
	rootCmd.AddCommand(commands.LinearCmd)
	rootCmd.AddCommand(commands.WrikeCmd)

	// Start memory leak watchdog before every command.
	// Short-lived commands exit before the window fills; long-running ones
	// (serve, plan, implement, …) are monitored throughout their lifetime.
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		watchdog.Start(context.Background(), watchdog.DefaultConfig())
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
