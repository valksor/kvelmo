// Package token provides shared token resolution utilities for providers.
package token

import (
	"errors"
)

// ErrNoToken is returned when no token can be resolved.
var ErrNoToken = errors.New("no token found")

// ResolverConfig defines the token sources for a provider.
type ResolverConfig struct {
	// ProviderName is the provider name for error messages.
	// Should be in uppercase (e.g., "GITHUB", "NOTION", "JIRA").
	ProviderName string

	// ConfigToken is the value from the configuration file (config.yaml).
	// This value has already been expanded with environment variables
	// if it uses ${VAR} syntax (e.g., "${GITHUB_TOKEN}").
	ConfigToken string

	// OptionalCLIFallback is an optional function to get a token from a CLI tool.
	// Examples: gh CLI auth token, az account get-access-token, etc.
	// Only used if ConfigToken is empty.
	OptionalCLIFallback func() string
}

// ResolveToken resolves a provider token from config and optional CLI fallback.
//
// Priority order:
//  1. ConfigToken (from config.yaml, already expanded with ${VAR} values)
//  2. OptionalCLIFallback result (e.g., gh CLI auth token)
//
// Note: Environment variable checking has been removed - config.yaml is the
// source of truth. Use ${VAR} syntax in config.yaml to reference environment
// variables.
//
// Returns ErrNoToken if no token is found.
func ResolveToken(cfg ResolverConfig) (string, error) {
	// 1. Check config token (already expanded with env vars)
	if cfg.ConfigToken != "" {
		return cfg.ConfigToken, nil
	}

	// 2. Try CLI fallback
	if cfg.OptionalCLIFallback != nil {
		if token := cfg.OptionalCLIFallback(); token != "" {
			return token, nil
		}
	}

	return "", ErrNoToken
}

// Config creates a ResolverConfig with the given provider name and config token.
func Config(providerName, configToken string) ResolverConfig {
	return ResolverConfig{
		ProviderName: providerName,
		ConfigToken:  configToken,
	}
}

// WithCLIFallback sets the optional CLI fallback function.
func (cfg ResolverConfig) WithCLIFallback(fallback func() string) ResolverConfig {
	cfg.OptionalCLIFallback = fallback

	return cfg
}

// MustResolveToken is like ResolveToken but panics on error.
// Useful for package-level initialization or when token is required.
func MustResolveToken(cfg ResolverConfig) string {
	token, err := ResolveToken(cfg)
	if err != nil {
		provider := cfg.ProviderName
		if provider == "" {
			provider = "provider"
		}
		panic(provider + ": " + ErrNoToken.Error() + " (configure token in config.yaml)")
	}

	return token
}
