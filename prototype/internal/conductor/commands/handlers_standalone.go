package commands

import (
	"context"
	"errors"

	"github.com/valksor/go-mehrhof/internal/conductor"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "standalone-review",
			Description:  "Perform standalone code review without an active task",
			Category:     "workflow",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleStandaloneReview,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "standalone-simplify",
			Description:  "Perform standalone code simplification without an active task",
			Category:     "workflow",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleStandaloneSimplify,
	})
}

// handleStandaloneReview performs a code review on uncommitted or branch changes.
func handleStandaloneReview(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	opts, err := buildReviewOpts(inv)
	if err != nil {
		return nil, err
	}

	result, err := cond.ReviewStandalone(ctx, opts)
	if err != nil {
		return NewResult("Review failed").WithData(map[string]any{ //nolint:nilerr // Return error details as data
			"success": false,
			"error":   err.Error(),
		}), nil
	}

	return NewResult("Review complete").WithData(buildReviewResponseData(result)), nil
}

// handleStandaloneSimplify performs code simplification on uncommitted or branch changes.
func handleStandaloneSimplify(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	opts, err := buildSimplifyOpts(inv)
	if err != nil {
		return nil, err
	}

	result, err := cond.SimplifyStandalone(ctx, opts)
	if err != nil {
		return NewResult("Simplify failed").WithData(map[string]any{ //nolint:nilerr // Return error details as data
			"success": false,
			"error":   err.Error(),
		}), nil
	}

	return NewResult("Simplify complete").WithData(buildSimplifyResponseData(result)), nil
}

func buildDiffOpts(inv Invocation) conductor.StandaloneDiffOptions {
	mode := GetString(inv.Options, "mode")
	if mode == "" {
		mode = "uncommitted"
	}

	contextLines := GetInt(inv.Options, "context")

	// Extract files slice from options.
	var files []string
	if raw, ok := inv.Options["files"]; ok {
		if slice, ok := raw.([]any); ok {
			for _, v := range slice {
				if s, ok := v.(string); ok {
					files = append(files, s)
				}
			}
		}
	}

	return conductor.StandaloneDiffOptions{
		Mode:       mapDiffMode(mode),
		BaseBranch: GetString(inv.Options, "base_branch"),
		Range:      GetString(inv.Options, "range"),
		Files:      files,
		Context:    contextLines,
	}
}

func buildReviewOpts(inv Invocation) (conductor.StandaloneReviewOptions, error) {
	if inv.Options == nil {
		return conductor.StandaloneReviewOptions{}, errors.New("options required")
	}

	diffOpts := buildDiffOpts(inv)
	applyFixes := GetBool(inv.Options, "apply_fixes")
	createCheckpoint := GetBool(inv.Options, "create_checkpoint")

	// Default checkpoint to true when applying fixes.
	if applyFixes && !createCheckpoint {
		createCheckpoint = true
	}

	return conductor.StandaloneReviewOptions{
		StandaloneDiffOptions: diffOpts,
		Agent:                 GetString(inv.Options, "agent"),
		ApplyFixes:            applyFixes,
		CreateCheckpoint:      createCheckpoint,
	}, nil
}

func buildSimplifyOpts(inv Invocation) (conductor.StandaloneSimplifyOptions, error) {
	if inv.Options == nil {
		return conductor.StandaloneSimplifyOptions{}, errors.New("options required")
	}

	diffOpts := buildDiffOpts(inv)

	return conductor.StandaloneSimplifyOptions{
		StandaloneDiffOptions: diffOpts,
		Agent:                 GetString(inv.Options, "agent"),
		CreateCheckpoint:      GetBool(inv.Options, "create_checkpoint"),
	}, nil
}

func buildReviewResponseData(result *conductor.StandaloneReviewResult) map[string]any {
	resp := map[string]any{
		"success": true,
		"verdict": result.Verdict,
		"summary": result.Summary,
	}

	if len(result.Issues) > 0 {
		issues := make([]map[string]any, 0, len(result.Issues))
		for _, issue := range result.Issues {
			issues = append(issues, map[string]any{
				"severity":    issue.Severity,
				"category":    issue.Category,
				"file":        issue.File,
				"line":        issue.Line,
				"description": issue.Message,
			})
		}
		resp["issues"] = issues
	}

	if len(result.Changes) > 0 {
		changes := make([]map[string]any, 0, len(result.Changes))
		for _, change := range result.Changes {
			changes = append(changes, map[string]any{
				"path":      change.Path,
				"operation": string(change.Operation),
			})
		}
		resp["changes"] = changes
	}

	if result.Usage != nil {
		resp["usage"] = map[string]any{
			"input_tokens":  result.Usage.InputTokens,
			"output_tokens": result.Usage.OutputTokens,
			"cached_tokens": result.Usage.CachedTokens,
			"cost_usd":      result.Usage.CostUSD,
		}
	}

	return resp
}

func buildSimplifyResponseData(result *conductor.StandaloneSimplifyResult) map[string]any {
	resp := map[string]any{
		"success": true,
		"summary": result.Summary,
	}

	if len(result.Changes) > 0 {
		changes := make([]map[string]any, 0, len(result.Changes))
		for _, change := range result.Changes {
			changes = append(changes, map[string]any{
				"path":      change.Path,
				"operation": string(change.Operation),
			})
		}
		resp["changes"] = changes
	}

	if result.Usage != nil {
		resp["usage"] = map[string]any{
			"input_tokens":  result.Usage.InputTokens,
			"output_tokens": result.Usage.OutputTokens,
			"cached_tokens": result.Usage.CachedTokens,
			"cost_usd":      result.Usage.CostUSD,
		}
	}

	return resp
}

// mapDiffMode converts string mode to conductor.StandaloneDiffMode.
func mapDiffMode(mode string) conductor.StandaloneDiffMode {
	switch mode {
	case "uncommitted":
		return conductor.DiffModeUncommitted
	case "branch":
		return conductor.DiffModeBranch
	case "range":
		return conductor.DiffModeRange
	case "files":
		return conductor.DiffModeFiles
	default:
		return conductor.DiffModeUncommitted
	}
}
