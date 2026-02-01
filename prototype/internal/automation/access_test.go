package automation

import (
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestNewAccessFilter(t *testing.T) {
	cfg := &storage.AutomationAccessControlConfig{
		Mode:      "allowlist",
		Allowlist: []string{"trusted-org", "trusted-user"},
		Blocklist: []string{"blocked-org"},
	}

	filter := NewAccessFilter(cfg)

	if filter == nil {
		t.Fatal("Expected filter to be created")
	}
}

func TestAccessFilter_AllowedMode(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *storage.AutomationAccessControlConfig
		user     *UserInfo
		repo     *RepositoryInfo
		expected bool
	}{
		{
			name: "all_mode_allows_everything",
			cfg: &storage.AutomationAccessControlConfig{
				Mode: "all",
			},
			user: &UserInfo{
				Login: "any-user",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner: "any-owner",
			},
			expected: true,
		},
		{
			name: "all_mode_blocks_bots_by_default",
			cfg: &storage.AutomationAccessControlConfig{
				Mode:      "all",
				AllowBots: false,
			},
			user: &UserInfo{
				Login: "dependabot[bot]",
				Type:  "Bot",
			},
			repo: &RepositoryInfo{
				Owner: "any-owner",
			},
			expected: false,
		},
		{
			name: "all_mode_allows_bots_when_enabled",
			cfg: &storage.AutomationAccessControlConfig{
				Mode:      "all",
				AllowBots: true,
			},
			user: &UserInfo{
				Login: "dependabot[bot]",
				Type:  "Bot",
			},
			repo: &RepositoryInfo{
				Owner: "any-owner",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewAccessFilter(tt.cfg)
			allowed, _ := filter.IsAllowed(tt.user, tt.repo)
			if allowed != tt.expected {
				t.Errorf("IsAllowed() = %v, want %v", allowed, tt.expected)
			}
		})
	}
}

func TestAccessFilter_AllowlistMode(t *testing.T) {
	cfg := &storage.AutomationAccessControlConfig{
		Mode:      "allowlist",
		Allowlist: []string{"trusted-org", "trusted-user", "myorg/*"},
	}

	filter := NewAccessFilter(cfg)

	tests := []struct {
		name     string
		user     *UserInfo
		repo     *RepositoryInfo
		expected bool
	}{
		{
			name: "allowed_owner",
			user: &UserInfo{
				Login: "some-user",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner: "trusted-org",
			},
			expected: true,
		},
		{
			name: "allowed_user",
			user: &UserInfo{
				Login: "trusted-user",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner: "random-org",
			},
			expected: true,
		},
		{
			name: "wildcard_match",
			user: &UserInfo{
				Login: "any-user",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner:    "myorg",
				FullName: "myorg/repo",
			},
			expected: true,
		},
		{
			name: "not_allowed",
			user: &UserInfo{
				Login: "untrusted-user",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner:    "untrusted-org",
				FullName: "untrusted-org/repo",
			},
			expected: false,
		},
		{
			name: "bot_blocked",
			user: &UserInfo{
				Login: "github-actions[bot]",
				Type:  "Bot",
			},
			repo: &RepositoryInfo{
				Owner: "trusted-org",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, _ := filter.IsAllowed(tt.user, tt.repo)
			if allowed != tt.expected {
				t.Errorf("IsAllowed() = %v, want %v", allowed, tt.expected)
			}
		})
	}
}

func TestAccessFilter_BlocklistMode(t *testing.T) {
	cfg := &storage.AutomationAccessControlConfig{
		Mode:      "blocklist",
		Blocklist: []string{"blocked-org", "spammer", "evil-*"},
	}

	filter := NewAccessFilter(cfg)

	tests := []struct {
		name     string
		user     *UserInfo
		repo     *RepositoryInfo
		expected bool
	}{
		{
			name: "not_blocked",
			user: &UserInfo{
				Login: "good-user",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner:    "good-org",
				FullName: "good-org/repo",
			},
			expected: true,
		},
		{
			name: "blocked_owner",
			user: &UserInfo{
				Login: "some-user",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner:    "blocked-org",
				FullName: "blocked-org/repo",
			},
			expected: false,
		},
		{
			name: "blocked_user",
			user: &UserInfo{
				Login: "spammer",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner:    "good-org",
				FullName: "good-org/repo",
			},
			expected: false,
		},
		{
			name: "wildcard_blocked",
			user: &UserInfo{
				Login: "good-user",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner:    "evil-corp",
				FullName: "evil-corp/repo",
			},
			expected: false,
		},
		{
			name: "bot_always_blocked",
			user: &UserInfo{
				Login: "renovate[bot]",
				Type:  "Bot",
			},
			repo: &RepositoryInfo{
				Owner:    "good-org",
				FullName: "good-org/repo",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, _ := filter.IsAllowed(tt.user, tt.repo)
			if allowed != tt.expected {
				t.Errorf("IsAllowed() = %v, want %v", allowed, tt.expected)
			}
		})
	}
}

func TestAccessFilter_CombinedMode(t *testing.T) {
	// Both allowlist and blocklist - blocklist takes precedence in "all" mode with blocklist.
	cfg := &storage.AutomationAccessControlConfig{
		Mode:      "all",
		Blocklist: []string{"blocked-user"},
	}

	filter := NewAccessFilter(cfg)

	tests := []struct {
		name     string
		user     *UserInfo
		repo     *RepositoryInfo
		expected bool
	}{
		{
			name: "not_blocked",
			user: &UserInfo{
				Login: "allowed-user",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner: "org-b",
			},
			expected: true,
		},
		{
			name: "blocked",
			user: &UserInfo{
				Login: "blocked-user",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner: "org-a",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, _ := filter.IsAllowed(tt.user, tt.repo)
			if allowed != tt.expected {
				t.Errorf("IsAllowed() = %v, want %v", allowed, tt.expected)
			}
		})
	}
}

func TestAccessFilter_NilConfig(t *testing.T) {
	filter := NewAccessFilter(nil)

	user := &UserInfo{
		Login: "any-user",
		Type:  "User",
	}
	repo := &RepositoryInfo{
		Owner: "any-owner",
	}

	// With nil config, should allow (default to "all" mode).
	allowed, _ := filter.IsAllowed(user, repo)
	if !allowed {
		t.Error("Expected event to be allowed with nil config")
	}
}

func TestAccessFilter_IsBot(t *testing.T) {
	tests := []struct {
		name     string
		user     *UserInfo
		expected bool
	}{
		{
			name:     "bot_type",
			user:     &UserInfo{Login: "some-bot", Type: "Bot"},
			expected: true,
		},
		{
			name:     "bot_suffix",
			user:     &UserInfo{Login: "dependabot[bot]", Type: "User"},
			expected: true,
		},
		{
			name:     "github_actions",
			user:     &UserInfo{Login: "github-actions[bot]", Type: ""},
			expected: true,
		},
		{
			name:     "renovate_bot",
			user:     &UserInfo{Login: "renovate", Type: ""},
			expected: true,
		},
		{
			name:     "normal_user",
			user:     &UserInfo{Login: "john-doe", Type: "User"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use internal method through IsAllowed with AllowBots = false.
			cfg := &storage.AutomationAccessControlConfig{
				Mode:      "all",
				AllowBots: false,
			}
			f := NewAccessFilter(cfg)
			repo := &RepositoryInfo{Owner: "test"}

			allowed, _ := f.IsAllowed(tt.user, repo)
			isBot := !allowed

			if isBot != tt.expected {
				t.Errorf("isBot() = %v, want %v", isBot, tt.expected)
			}
		})
	}
}

func TestAccessFilter_MatchPattern(t *testing.T) {
	// Test specific patterns without the wildcard "*" that matches everything.
	cfg := &storage.AutomationAccessControlConfig{
		Mode:      "allowlist",
		Allowlist: []string{"foo", "foo*", "*bar"},
	}
	f := NewAccessFilter(cfg)

	tests := []struct {
		name     string
		user     string
		expected bool
	}{
		{
			name:     "exact_match",
			user:     "foo",
			expected: true,
		},
		{
			name:     "no_match",
			user:     "baz",
			expected: false,
		},
		{
			name:     "wildcard_suffix",
			user:     "foobar",
			expected: true,
		},
		{
			name:     "wildcard_prefix_match",
			user:     "testbar",
			expected: true, // matches *bar
		},
		{
			name:     "case_insensitive",
			user:     "FOO",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &UserInfo{Login: tt.user, Type: "User"}
			repo := &RepositoryInfo{Owner: "other"}

			allowed, _ := f.IsAllowed(user, repo)
			if allowed != tt.expected {
				t.Errorf("matchPattern for user %q = %v, want %v", tt.user, allowed, tt.expected)
			}
		})
	}
}

func TestAccessFilter_RequireOrg(t *testing.T) {
	cfg := &storage.AutomationAccessControlConfig{
		Mode:       "all",
		RequireOrg: true,
	}

	filter := NewAccessFilter(cfg)

	tests := []struct {
		name     string
		user     *UserInfo
		repo     *RepositoryInfo
		expected bool
	}{
		{
			name: "owner_matches_user",
			user: &UserInfo{
				Login: "owner",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner: "owner",
			},
			expected: true,
		},
		{
			name: "owner_does_not_match",
			user: &UserInfo{
				Login: "random-user",
				Type:  "User",
			},
			repo: &RepositoryInfo{
				Owner: "org",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, _ := filter.IsAllowed(tt.user, tt.repo)
			if allowed != tt.expected {
				t.Errorf("IsAllowed() = %v, want %v", allowed, tt.expected)
			}
		})
	}
}
