// Package registration provides standard provider and agent registration functions.
// This is shared between the main conductor initialization and the MCP server.
package registration

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/crealfy/crea-wrike/wrike"
	"github.com/valksor/go-mehrhof/internal/agent/claude"
	"github.com/valksor/go-mehrhof/internal/agent/codex"
	"github.com/valksor/go-mehrhof/internal/agent/noop"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/provider"
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
	"github.com/valksor/go-mehrhof/internal/provider/youtrack"
	"github.com/valksor/go-toolkit/capability"
	"github.com/valksor/go-toolkit/providerconfig"
)

// RegisterStandardProviders registers all standard providers with the conductor's provider registry.
func RegisterStandardProviders(cond *conductor.Conductor) {
	registry := cond.GetProviderRegistry()

	file.Register(registry)
	directory.Register(registry)
	empty.Register(registry)
	github.Register(registry)
	gitlab.Register(registry)
	registerWrike(registry)
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

// registerWrike registers the Wrike provider from crea-wrike.
// The wrike provider self-discovers its configuration from .crealfy/wrike.yaml.
func registerWrike(r *provider.Registry) {
	_ = r.Register(provider.ProviderInfo{
		Name:         "wrike",
		Description:  "Wrike task management",
		Schemes:      []string{"wrike", "wk"},
		Priority:     20,
		Capabilities: capability.Infer(&wrike.Provider{}),
	}, func(ctx context.Context, _ providerconfig.Config) (any, error) {
		// wrike.New() self-discovers config from .crealfy/wrike.yaml
		return wrike.New(ctx)
	})
}

// RegisterStandardAgents registers all standard agents with the conductor's agent registry.
// Continues on error, collecting all errors and returning at the end.
// Some agents may be available even if others fail to register.
//
// When MEHR_TEST_MODE=1 is set, a noop agent is registered that's always available.
// This allows smoke tests to run in CI without requiring actual AI agents.
func RegisterStandardAgents(cond *conductor.Conductor) error {
	registry := cond.GetAgentRegistry()
	var errs []error

	// Register noop agent first when in test mode (ensures it's available as fallback)
	if os.Getenv("MEHR_TEST_MODE") == "1" {
		if err := noop.Register(registry); err != nil {
			errs = append(errs, fmt.Errorf("register noop agent: %w", err))
		}
	}

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
