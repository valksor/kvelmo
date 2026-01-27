// Package registration provides standard provider and agent registration functions.
// This is shared between the main conductor initialization and the MCP server.
package registration

import (
	"fmt"
	"log/slog"

	"github.com/valksor/go-mehrhof/internal/agent/claude"
	"github.com/valksor/go-mehrhof/internal/agent/codex"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/provider/asana"
	"github.com/valksor/go-mehrhof/internal/provider/azuredevops"
	"github.com/valksor/go-mehrhof/internal/provider/bitbucket"
	"github.com/valksor/go-mehrhof/internal/provider/clickup"
	"github.com/valksor/go-mehrhof/internal/provider/directory"
	"github.com/valksor/go-mehrhof/internal/provider/empty"
	"github.com/valksor/go-mehrhof/internal/provider/file"
	"github.com/valksor/go-mehrhof/internal/provider/github"
	"github.com/valksor/go-mehrhof/internal/provider/gitlab"
	"github.com/valksor/go-mehrhof/internal/provider/jira"
	"github.com/valksor/go-mehrhof/internal/provider/linear"
	"github.com/valksor/go-mehrhof/internal/provider/notion"
	"github.com/valksor/go-mehrhof/internal/provider/queue"
	"github.com/valksor/go-mehrhof/internal/provider/trello"
	"github.com/valksor/go-mehrhof/internal/provider/wrike"
	"github.com/valksor/go-mehrhof/internal/provider/youtrack"
)

// RegisterStandardProviders registers all standard providers with the conductor's provider registry.
func RegisterStandardProviders(cond *conductor.Conductor) {
	registry := cond.GetProviderRegistry()

	file.Register(registry)
	directory.Register(registry)
	empty.Register(registry)
	github.Register(registry)
	gitlab.Register(registry)
	wrike.Register(registry)
	linear.Register(registry)
	jira.Register(registry)
	notion.Register(registry)
	queue.Register(registry)
	trello.Register(registry)
	youtrack.Register(registry)
	bitbucket.Register(registry)
	asana.Register(registry)
	clickup.Register(registry)
	azuredevops.Register(registry)
}

// RegisterStandardAgents registers all standard agents with the conductor's agent registry.
// Continues on error, collecting all errors and returning at the end.
// Some agents may be available even if others fail to register.
func RegisterStandardAgents(cond *conductor.Conductor) error {
	registry := cond.GetAgentRegistry()
	var errs []error

	if err := claude.Register(registry); err != nil {
		errs = append(errs, fmt.Errorf("register claude agent: %w", err))
	}

	// Register Codex agent
	if err := codex.Register(registry); err != nil {
		errs = append(errs, fmt.Errorf("register codex agent: %w", err))
	}

	if len(errs) > 0 {
		// Log warnings but don't fail - some agents may be available
		for _, err := range errs {
			slog.Warn("Agent registration failed", "error", err)
		}

		return fmt.Errorf("some agents failed to register: %d errors", len(errs))
	}

	return nil
}
