package automation

import (
	"log/slog"
	"slices"
	"strings"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// AccessFilter enforces access control rules for automation.
type AccessFilter struct {
	config *storage.AutomationAccessControlConfig
}

// NewAccessFilter creates a new access filter with the given configuration.
func NewAccessFilter(cfg *storage.AutomationAccessControlConfig) *AccessFilter {
	if cfg == nil {
		cfg = &storage.AutomationAccessControlConfig{
			Mode: "all",
		}
	}

	return &AccessFilter{config: cfg}
}

// IsAllowed checks if a user/repository combination is allowed to trigger automation.
// Returns (allowed, reason).
func (f *AccessFilter) IsAllowed(user *UserInfo, repo *RepositoryInfo) (bool, string) {
	if f.config == nil {
		return true, ""
	}

	// Check bot restriction.
	if !f.config.AllowBots && f.isBot(user) {
		return false, "bot accounts are not allowed"
	}

	// Check org requirement.
	if f.config.RequireOrg && !f.isOrgMember(user, repo) {
		return false, "user is not an organization member"
	}

	// Apply mode-specific rules.
	switch strings.ToLower(f.config.Mode) {
	case "allowlist":
		return f.checkAllowlist(user, repo)
	case "blocklist":
		return f.checkBlocklist(user, repo)
	case "all", "":
		// In "all" mode, still apply blocklist if present.
		if len(f.config.Blocklist) > 0 {
			return f.checkBlocklist(user, repo)
		}

		return true, ""
	default:
		return true, ""
	}
}

// checkAllowlist returns true if user/repo is in the allowlist.
func (f *AccessFilter) checkAllowlist(user *UserInfo, repo *RepositoryInfo) (bool, string) {
	if len(f.config.Allowlist) == 0 {
		return false, "allowlist is empty"
	}

	// Check if user is in allowlist.
	if f.matchesAny(user.Login, f.config.Allowlist) {
		return true, ""
	}

	// Check if repo owner is in allowlist.
	if f.matchesAny(repo.Owner, f.config.Allowlist) {
		return true, ""
	}

	// Check if full repo name is in allowlist.
	if f.matchesAny(repo.FullName, f.config.Allowlist) {
		return true, ""
	}

	return false, "user or repository not in allowlist"
}

// checkBlocklist returns false if user/repo is in the blocklist.
func (f *AccessFilter) checkBlocklist(user *UserInfo, repo *RepositoryInfo) (bool, string) {
	if len(f.config.Blocklist) == 0 {
		return true, ""
	}

	// Check if user is in blocklist.
	if f.matchesAny(user.Login, f.config.Blocklist) {
		return false, "user is in blocklist"
	}

	// Check if repo owner is in blocklist.
	if f.matchesAny(repo.Owner, f.config.Blocklist) {
		return false, "repository owner is in blocklist"
	}

	// Check if full repo name is in blocklist.
	if f.matchesAny(repo.FullName, f.config.Blocklist) {
		return false, "repository is in blocklist"
	}

	return true, ""
}

// matchesAny checks if value matches any pattern in the list.
// Supports exact match and wildcard prefix/suffix matching.
func (f *AccessFilter) matchesAny(value string, patterns []string) bool {
	value = strings.ToLower(value)
	for _, pattern := range patterns {
		pattern = strings.ToLower(pattern)
		if f.matches(value, pattern) {
			return true
		}
	}

	return false
}

// matches checks if a value matches a pattern.
// Supports:
// - Exact match: "username"
// - Prefix wildcard: "*-bot" matches "deploy-bot"
// - Suffix wildcard: "org/*" matches "org/repo".
func (f *AccessFilter) matches(value, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Prefix wildcard: *suffix
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(value, pattern[1:])
	}

	// Suffix wildcard: prefix*
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(value, pattern[:len(pattern)-1])
	}

	// Exact match.
	return value == pattern
}

// isBot determines if a user is a bot account.
func (f *AccessFilter) isBot(user *UserInfo) bool {
	if user == nil {
		return false
	}

	// Check user type.
	if strings.EqualFold(user.Type, "Bot") {
		return true
	}

	// Check common bot naming patterns.
	login := strings.ToLower(user.Login)
	botPatterns := []string{
		"[bot]",
		"-bot",
		"_bot",
		"bot-",
		"bot_",
	}

	for _, pattern := range botPatterns {
		if strings.Contains(login, pattern) {
			return true
		}
	}

	// Check for specific known bots.
	knownBots := []string{
		"dependabot",
		"renovate",
		"greenkeeper",
		"codecov",
		"snyk",
		"github-actions",
		"gitlab-ci",
	}

	return slices.Contains(knownBots, login)
}

// isOrgMember checks if user is a member of the repository's organization.
// Note: This is a basic check based on repo owner. Full org membership
// verification would require an API call.
func (f *AccessFilter) isOrgMember(user *UserInfo, repo *RepositoryInfo) bool {
	if user == nil || repo == nil {
		return false
	}

	// Simple check: user login matches repo owner.
	// This covers personal repos and org repos where user is the owner.
	// Full org membership would require GitHub/GitLab API verification.
	isOwner := strings.EqualFold(user.Login, repo.Owner)
	if !isOwner {
		slog.Debug("org membership check is approximate (owner-only match)",
			"user", user.Login,
			"repo_owner", repo.Owner,
		)
	}

	return isOwner
}
