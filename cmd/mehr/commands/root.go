package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/config"
	"github.com/valksor/go-mehrhof/internal/disambiguate"
	"github.com/valksor/go-mehrhof/internal/help"
	"github.com/valksor/go-toolkit/display"
	"github.com/valksor/go-toolkit/log"
)

var (
	settings *config.Settings

	// Global flags.
	verbose bool
	noColor bool
	quiet   bool
)

var rootCmd = &cobra.Command{
	Use:   "mehr",
	Short: "AI-powered task automation",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Long: `Mehrhof is a CLI tool for AI-assisted task automation by Valksor.

It orchestrates AI agents to perform planning, implementation, and code review
workflows. Tasks can be sourced from files, directories, or external providers.

Quick Start:
  mehr start task.md     Start a task from a markdown file
  mehr plan              AI creates specifications
  mehr implement         AI implements the code
  mehr finish            Complete and merge/PR

For guidance:  mehr guide
For status:    mehr status
For full auto: mehr auto task.md`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load .env file FIRST, before anything else
		// This ensures env vars are available for all subsequent operations
		if err := config.LoadDotEnvFromCwd(); err != nil {
			// Log warning but don't fail - .env parsing errors should be reported
			// but shouldn't prevent the command from running
			fmt.Fprintf(os.Stderr, "warning: failed to load .mehrhof/.env: %v\n", err)
		}

		// Configure logging from CLI flag
		log.Configure(log.Options{
			Verbose: verbose,
		})

		// Initialize color output from CLI flag (also respects NO_COLOR env)
		display.InitColors(noColor)

		// Load settings (user preferences)
		var err error
		settings, err = config.LoadSettings()
		if err != nil {
			return fmt.Errorf("load settings: %w", err)
		}

		log.Debug("initialized", "verbose", verbose)

		// Async update check (non-blocking, doesn't slow startup)
		// Skip for the 'update' command itself to avoid redundant checks
		if cmd.Name() != "update" && shouldCheckForUpdates(settings) {
			go checkForUpdatesInBackground(cmd.Context())
		}

		return nil
	},
}

// Execute runs the root command with signal handling and command disambiguation.
func Execute() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Pre-process args to handle colon notation before Cobra sees them
	args := os.Args[1:]
	if len(args) > 0 {
		resolvedArgs, err := resolveCommandArgs(args)
		if err != nil {
			return err
		}
		if resolvedArgs != nil {
			rootCmd.SetArgs(resolvedArgs)

			return rootCmd.ExecuteContext(ctx)
		}
	}

	// Execute with Cobra's built-in prefix matching
	err := rootCmd.ExecuteContext(ctx)
	if err == nil {
		return nil
	}

	// Check if this is an ambiguous command error we can help with
	if shouldAttemptDisambiguation(err) && len(args) > 0 {
		resolvedArgs, disambigErr := attemptDisambiguation(args[0])
		if disambigErr == nil {
			rootCmd.SetArgs(append(resolvedArgs, args[1:]...))

			return rootCmd.ExecuteContext(ctx)
		}
		// If disambiguation was attempted (matches found), use its error
		// This handles: non-interactive mode, user cancelled selection, etc.
		// Only fall through to Cobra's error if no matches were found
		if !strings.Contains(disambigErr.Error(), "no commands match") {
			return disambigErr
		}
	}

	return err
}

// resolveCommandArgs handles colon notation (e.g., "c:v" -> "config validate").
// Returns nil if no transformation needed.
func resolveCommandArgs(args []string) ([]string, error) {
	if len(args) == 0 || !strings.Contains(args[0], ":") {
		return nil, nil
	}

	resolved, matches, err := disambiguate.ResolveColonPath(rootCmd, args[0])
	if err != nil {
		// Not a colon path or no match - let Cobra handle it
		if strings.Contains(err.Error(), "not a colon path") {
			return nil, nil
		}

		return nil, err
	}

	// Ambiguous match - need user selection
	if len(matches) > 0 {
		if !disambiguate.IsInteractive() {
			return nil, errors.New(disambiguate.FormatAmbiguousError(args[0], matches))
		}
		selected, err := disambiguate.SelectCommand(matches, args[0])
		if err != nil {
			return nil, err
		}
		resolved = selected.Path
	}

	// Replace colon path with resolved space-separated args
	return append(resolved, args[1:]...), nil
}

// shouldAttemptDisambiguation checks if we should try to disambiguate the error.
func shouldAttemptDisambiguation(err error) bool {
	errMsg := err.Error()

	return strings.Contains(errMsg, "unknown command")
}

// attemptDisambiguation tries to find matching commands for a prefix.
func attemptDisambiguation(prefix string) ([]string, error) {
	matches := disambiguate.FindPrefixMatches(rootCmd, prefix)

	if len(matches) == 0 {
		return nil, fmt.Errorf("no commands match prefix %q", prefix)
	}

	if len(matches) == 1 {
		// Single match - shouldn't happen often with EnablePrefixMatching
		// but handle it anyway
		return []string{matches[0].Command.Name()}, nil
	}

	// Multiple matches - interactive selection
	if !disambiguate.IsInteractive() {
		return nil, errors.New(disambiguate.FormatAmbiguousError(prefix, matches))
	}

	selected, err := disambiguate.SelectCommand(matches, prefix)
	if err != nil {
		return nil, err
	}

	return []string{selected.Command.Name()}, nil
}

func init() {
	// Enable Cobra's built-in prefix matching for unambiguous command prefixes
	cobra.EnablePrefixMatching = true

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")

	// Add command groups for better help organization
	rootCmd.AddGroup(&cobra.Group{
		ID:    "workflow",
		Title: "Workflow Commands:",
	}, &cobra.Group{
		ID:    "task",
		Title: "Task Commands:",
	}, &cobra.Group{
		ID:    "info",
		Title: "Information Commands:",
	}, &cobra.Group{
		ID:    "config",
		Title: "Configuration Commands:",
	}, &cobra.Group{
		ID:    "utility",
		Title: "Utility Commands:",
	})

	// Setup contextual help that shows available/unavailable commands
	help.SetupContextualHelp(rootCmd)
}

// GetSettings returns the loaded settings.
func GetSettings() *config.Settings {
	return settings
}
