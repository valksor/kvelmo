// Package config provides configuration management utilities.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProviderMigration defines how to migrate a provider's config from .mehrhof/config.yaml to .crealfy/<provider>.yaml.
type ProviderMigration struct {
	// Name is the provider name (e.g., "wrike", "jira", "linear")
	Name string

	// YAMLKey is the key in .mehrhof/config.yaml (e.g., "wrike", "jira")
	YAMLKey string

	// FieldMap maps old field names to new field names.
	// If empty, fields are copied as-is.
	// Use empty string value to skip a field.
	FieldMap map[string]string

	// ExtraFields adds additional fields to the migrated config.
	ExtraFields map[string]string
}

// MigrateProviderConfig migrates a provider's config from .mehrhof/config.yaml to .crealfy/<provider>.yaml.
// Returns true if migration was performed, false if skipped (already exists or no old config).
func MigrateProviderConfig(workDir string, m ProviderMigration) (bool, error) {
	crealyfDir := filepath.Join(workDir, ".crealfy")
	newConfigPath := filepath.Join(crealyfDir, m.Name+".yaml")

	// Skip if new config already exists
	if _, err := os.Stat(newConfigPath); err == nil {
		return false, nil
	}

	// Load old config from .mehrhof/config.yaml
	oldConfigPath := filepath.Join(workDir, ".mehrhof", "config.yaml")
	oldData, err := os.ReadFile(oldConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // No old config to migrate
		}

		return false, fmt.Errorf("read old config: %w", err)
	}

	// Parse old config
	var oldConfig map[string]any
	if err := yaml.Unmarshal(oldData, &oldConfig); err != nil {
		return false, fmt.Errorf("parse old config: %w", err)
	}

	// Extract provider section
	providerSection, ok := oldConfig[m.YAMLKey]
	if !ok {
		return false, nil // Provider not configured in old config
	}

	providerMap, ok := providerSection.(map[string]any)
	if !ok {
		return false, nil // Invalid format
	}

	// Check if there's anything meaningful to migrate
	if len(providerMap) == 0 {
		return false, nil
	}

	// Build new config
	newConfig := make(map[string]any)

	// Apply field mappings
	for oldKey, value := range providerMap {
		newKey := oldKey
		if m.FieldMap != nil {
			if mapped, exists := m.FieldMap[oldKey]; exists {
				if mapped == "" {
					continue // Skip this field
				}
				newKey = mapped
			}
		}
		newConfig[newKey] = value
	}

	// Add extra fields
	for key, value := range m.ExtraFields {
		if _, exists := newConfig[key]; !exists {
			newConfig[key] = value
		}
	}

	// Skip if nothing to write
	if len(newConfig) == 0 {
		return false, nil
	}

	// Create .crealfy directory
	if err := os.MkdirAll(crealyfDir, 0o755); err != nil {
		return false, fmt.Errorf("create .crealfy directory: %w", err)
	}

	// Write new config
	newData, err := yaml.Marshal(newConfig)
	if err != nil {
		return false, fmt.Errorf("marshal new config: %w", err)
	}

	if err := os.WriteFile(newConfigPath, newData, 0o600); err != nil {
		return false, fmt.Errorf("write new config: %w", err)
	}

	slog.Info("Migrated provider config",
		"provider", m.Name,
		"from", oldConfigPath,
		"to", newConfigPath,
	)

	return true, nil
}

// StandardProviderMigrations returns migration definitions for all extractable providers.
func StandardProviderMigrations() []ProviderMigration {
	return []ProviderMigration{
		{
			Name:    "wrike",
			YAMLKey: "wrike",
			FieldMap: map[string]string{
				"token":   "token",
				"host":    "host",
				"space":   "space_id",
				"folder":  "folder_id",
				"project": "project_id",
			},
			ExtraFields: map[string]string{
				"host": "https://www.wrike.com/api/v4",
			},
		},
		// Add more providers as they get extracted:
		// {
		// 	Name:    "jira",
		// 	YAMLKey: "jira",
		// 	FieldMap: map[string]string{
		// 		"token":    "token",
		// 		"email":    "email",
		// 		"base_url": "host",
		// 		"project":  "project",
		// 	},
		// },
		// {
		// 	Name:    "linear",
		// 	YAMLKey: "linear",
		// 	FieldMap: map[string]string{
		// 		"token": "token",
		// 		"team":  "team",
		// 	},
		// },
	}
}

// MigrateAllProviders runs all standard provider migrations.
// Logs results but doesn't fail on individual migration errors.
func MigrateAllProviders(workDir string) {
	for _, m := range StandardProviderMigrations() {
		migrated, err := MigrateProviderConfig(workDir, m)
		if err != nil {
			slog.Warn("Provider config migration failed",
				"provider", m.Name,
				"error", err,
			)

			continue
		}
		if migrated {
			slog.Debug("Provider config migrated", "provider", m.Name)
		}
	}
}
