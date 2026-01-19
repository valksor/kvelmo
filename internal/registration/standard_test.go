package registration

import (
	"slices"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/helper_test"
)

// TestRegisterStandardProviders verifies that all standard providers are registered correctly.
func TestRegisterStandardProviders(t *testing.T) {
	tests := []struct {
		name             string
		providerName     string
		atLeastOneScheme string // Verify at least this scheme is registered
	}{
		{"file provider", "file", "file"},
		{"directory provider", "directory", "dir"},
		{"empty provider", "empty", "empty"},
		{"github provider", "github", "github"},
		{"gitlab provider", "gitlab", "gitlab"},
		{"wrike provider", "wrike", "wrike"},
		{"linear provider", "linear", "linear"},
		{"jira provider", "jira", "jira"},
		{"notion provider", "notion", "notion"},
		{"trello provider", "trello", "trello"},
		{"youtrack provider", "youtrack", "youtrack"},
		{"bitbucket provider", "bitbucket", "bitbucket"},
		{"asana provider", "asana", "asana"},
		{"clickup provider", "clickup", "clickup"},
		{"azuredevops provider", "azuredevops", "azdo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := helper_test.NewTestConductor(t)
			registry := cond.GetProviderRegistry()

			// Register standard providers
			RegisterStandardProviders(cond)

			// Verify provider is registered by name
			info, _, ok := registry.Get(tt.providerName)
			if !ok {
				t.Errorf("provider %q not registered", tt.providerName)

				return
			}

			// Verify at least the expected scheme is accessible
			if !slices.Contains(info.Schemes, tt.atLeastOneScheme) {
				t.Errorf("provider %q missing scheme %q; got schemes: %v", tt.providerName, tt.atLeastOneScheme, info.Schemes)
			}

			// Verify scheme is accessible via GetByScheme
			_, _, ok = registry.GetByScheme(tt.atLeastOneScheme)
			if !ok {
				t.Errorf("scheme %q not registered via GetByScheme", tt.atLeastOneScheme)
			}
		})
	}

	t.Run("all 15 providers registered", func(t *testing.T) {
		cond := helper_test.NewTestConductor(t)
		registry := cond.GetProviderRegistry()

		RegisterStandardProviders(cond)

		providers := registry.List()
		if len(providers) != 15 {
			t.Errorf("got %d providers, want 15", len(providers))
		}

		// Verify expected provider names are present
		expectedNames := []string{
			"file", "directory", "empty", "github", "gitlab", "wrike",
			"linear", "jira", "notion", "trello", "youtrack",
			"bitbucket", "asana", "clickup", "azuredevops",
		}
		actualNames := make([]string, len(providers))
		for i, p := range providers {
			actualNames[i] = p.Name
		}

		for _, name := range expectedNames {
			if !slices.Contains(actualNames, name) {
				t.Errorf("expected provider %q not found in %v", name, actualNames)
			}
		}
	})

	t.Run("all expected schemes accessible", func(t *testing.T) {
		cond := helper_test.NewTestConductor(t)
		registry := cond.GetProviderRegistry()

		RegisterStandardProviders(cond)

		// Verify common schemes are accessible
		expectedSchemes := []string{
			"file", "dir", "empty", "github", "gh", "gitlab", "gl",
			"wrike", "linear", "jira", "notion", "trello",
			"youtrack", "bitbucket", "asana", "clickup", "azdo", "azure",
		}

		for _, scheme := range expectedSchemes {
			_, _, ok := registry.GetByScheme(scheme)
			if !ok {
				t.Errorf("scheme %q not accessible via GetByScheme", scheme)
			}
		}
	})
}

// TestRegisterStandardAgents verifies agent registration behavior.
func TestRegisterStandardAgents(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		cond := helper_test.NewTestConductor(t)
		registry := cond.GetAgentRegistry()

		err := RegisterStandardAgents(cond)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}

		// Verify agents are registered
		agents := registry.List()
		if len(agents) == 0 {
			t.Error("no agents registered")
		}

		// Check for expected agents
		foundClaude := false
		foundCodex := false
		for _, name := range agents {
			if name == "claude" {
				foundClaude = true
			}
			if name == "codex" {
				foundCodex = true
			}
		}

		if !foundClaude {
			t.Error("claude agent not registered")
		}
		if !foundCodex {
			t.Error("codex agent not registered")
		}
	})

	t.Run("duplicate agent returns error", func(t *testing.T) {
		cond := helper_test.NewTestConductor(t)
		registry := cond.GetAgentRegistry()

		// Pre-register a mock agent with the same name as a standard agent
		mockAgent := helper_test.NewMockAgent("claude")
		if err := registry.Register(mockAgent); err != nil {
			t.Fatalf("failed to register mock agent: %v", err)
		}

		err := RegisterStandardAgents(cond)

		if err == nil {
			t.Error("expected error for duplicate agent, got nil")
		}

		// Verify error message contains expected text
		if !strings.Contains(err.Error(), "some agents failed to register") {
			t.Errorf("error message should contain 'some agents failed to register', got: %v", err)
		}

		// Verify some agents may still be registered (partial registration)
		agents := registry.List()
		if len(agents) == 0 {
			t.Error("expected some agents to be registered despite duplicate error")
		}
	})
}
