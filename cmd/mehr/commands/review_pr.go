package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/registration"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
)

var (
	reviewPRProvider       string
	reviewPRNumber         int
	reviewPRFormat         string
	reviewPRScope          string
	reviewPRAgent          string
	reviewPRToken          string
	reviewPRAcknowledge    bool
	reviewPRUpdateExisting bool
)

var reviewPRCmd = &cobra.Command{
	Use:   "pr",
	Short: "Review a pull/merge request",
	Long: `Review a pull request (GitHub) or merge request (GitLab) using AI agents.

The provider is auto-detected from git remote URL. Use --provider to override.

This is a standalone command that does not require an active task or workspace.
It works from just CLI flags + optional config, making it ideal for CI/CD.

Examples:
  mehr review pr --pr-number 123                           # Auto-detect provider
  mehr review pr --pr-number 123 --provider github          # Explicit provider
  mehr review pr --pr-number 123 --agent claude            # Use specific agent
  mehr review pr --pr-number 123 --scope compact           # Review scope
  mehr review pr --pr-number 123 --token "$GITHUB_TOKEN"   # CI with token`,
	RunE: runReviewPR,
}

func init() {
	reviewCmd.AddCommand(reviewPRCmd)

	reviewPRCmd.Flags().StringVar(&reviewPRProvider, "provider", "", "Provider (github, gitlab, bitbucket, azuredevops). Auto-detected from git remote if omitted.")
	reviewPRCmd.Flags().IntVar(&reviewPRNumber, "pr-number", 0, "PR/MR number (required)")
	reviewPRCmd.Flags().StringVar(&reviewPRFormat, "format", "summary", "Comment format: summary, line-comments")
	reviewPRCmd.Flags().StringVar(&reviewPRScope, "scope", "full", "Review scope: full, compact, files-changed")
	reviewPRCmd.Flags().StringVar(&reviewPRAgent, "agent", "", "Agent to use (built-in: claude; or custom agent from config)")
	reviewPRCmd.Flags().StringVar(&reviewPRToken, "token", "", "Auth token (optional). Overrides config/env vars. Use for CI: --token \"$GITHUB_TOKEN\"")
	reviewPRCmd.Flags().BoolVar(&reviewPRAcknowledge, "acknowledge-fixes", true, "Acknowledge when previously reported issues are fixed")
	reviewPRCmd.Flags().BoolVar(&reviewPRUpdateExisting, "update-existing", true, "Edit existing comment vs post new comment")
	_ = reviewPRCmd.MarkFlagRequired("pr-number")
}

func runReviewPR(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Validate PR number
	if reviewPRNumber <= 0 {
		return fmt.Errorf("pr-number must be positive (got %d)", reviewPRNumber)
	}

	// This is a STANDALONE command - no workspace, no task, no user input required
	// It must work from just CLI flags + optional config

	// 1. Try to load config from .mehrhof/config.yaml if it exists
	// Returns defaults if config doesn't exist (OK for CI with built-in agents)
	var cfg *storage.WorkspaceConfig
	cwd, err := os.Getwd()
	if err == nil {
		// Try to open workspace and load config
		workspace, err := storage.OpenWorkspace(ctx, cwd, nil)
		if err == nil && workspace.HasConfig() {
			cfg, err = workspace.LoadConfig()
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("load config: %w", err)
			}
		}
	}

	// 2. Validate agent selection
	agent := reviewPRAgent
	if agent == "" && cfg != nil && cfg.Agent.Default != "" {
		agent = cfg.Agent.Default
	}
	if agent == "" {
		agent = "claude" // Built-in default
	}

	// 3. Detect provider from git remote if not specified
	providerName := reviewPRProvider
	if providerName == "" {
		// Get git remote
		gitClient, err := vcs.New(ctx, cwd)
		if err != nil {
			return fmt.Errorf("initialize git: %w (use --provider to specify)", err)
		}

		remote, err := gitClient.RemoteURL(ctx, "origin")
		if err != nil {
			return fmt.Errorf("get git remote: %w (use --provider to specify)", err)
		}
		providerName = provider.DetectProviderFromURL(remote)
		if providerName == "" {
			return fmt.Errorf("could not detect provider from remote %s (use --provider to specify)", remote)
		}
	}

	// 4. Initialize lightweight conductor
	cond, err := initializeConductorForPRReview(ctx, cfg, conductor.WithVerbose(verbose))
	if err != nil {
		return err
	}

	// 5. Run PR review (fully automated, no user input)
	result, err := cond.RunPRReview(ctx, conductor.PRReviewOptions{
		Provider:         providerName,
		PRNumber:         reviewPRNumber,
		Format:           reviewPRFormat,
		Scope:            reviewPRScope,
		AgentName:        agent,
		AcknowledgeFixes: reviewPRAcknowledge,
		UpdateExisting:   reviewPRUpdateExisting,
		Token:            reviewPRToken,
	})
	if err != nil {
		return fmt.Errorf("PR review failed: %w", err)
	}

	// 6. Display results
	if result.Skipped {
		fmt.Printf("⏭️  Skipped: %s\n", result.Reason)

		return nil
	}

	fmt.Printf("✅ Review completed for PR #%d\n", reviewPRNumber)
	fmt.Printf("   Provider: %s\n", providerName)
	fmt.Printf("   Agent: %s\n", agent)
	fmt.Printf("   Comments posted: %d\n", result.CommentsPosted)
	if result.URL != "" {
		fmt.Printf("   URL: %s\n", result.URL)
	}

	return nil
}

// initializeConductorForPRReview creates a minimal conductor for PR review.
// Does NOT require workspace, active task, or user input.
func initializeConductorForPRReview(_ context.Context, _ *storage.WorkspaceConfig, opts ...conductor.Option) (*conductor.Conductor, error) {
	// Create a minimal conductor - config can be nil for PR review
	cond, err := conductor.New(opts...)
	if err != nil {
		return nil, err
	}

	// Register standard providers and agents
	registration.RegisterStandardProviders(cond)

	if err := registration.RegisterStandardAgents(cond); err != nil {
		return nil, fmt.Errorf("register agents: %w", err)
	}

	return cond, nil
}
