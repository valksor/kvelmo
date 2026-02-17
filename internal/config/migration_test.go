package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMigrateProviderConfig(t *testing.T) {
	tests := []struct {
		name          string
		oldConfig     map[string]any
		migration     ProviderMigration
		wantMigrated  bool
		wantNewConfig map[string]any
	}{
		{
			name: "migrates wrike config",
			oldConfig: map[string]any{
				"wrike": map[string]any{
					"token":   "test-token",
					"host":    "https://custom.wrike.com/api/v4",
					"space":   "space123",
					"folder":  "folder456",
					"project": "project789",
				},
			},
			migration: ProviderMigration{
				Name:    "wrike",
				YAMLKey: "wrike",
				FieldMap: map[string]string{
					"token":   "token",
					"host":    "host",
					"space":   "space_id",
					"folder":  "folder_id",
					"project": "project_id",
				},
			},
			wantMigrated: true,
			wantNewConfig: map[string]any{
				"token":      "test-token",
				"host":       "https://custom.wrike.com/api/v4",
				"space_id":   "space123",
				"folder_id":  "folder456",
				"project_id": "project789",
			},
		},
		{
			name: "adds extra fields",
			oldConfig: map[string]any{
				"wrike": map[string]any{
					"token": "test-token",
				},
			},
			migration: ProviderMigration{
				Name:    "wrike",
				YAMLKey: "wrike",
				ExtraFields: map[string]string{
					"host": "https://www.wrike.com/api/v4",
				},
			},
			wantMigrated: true,
			wantNewConfig: map[string]any{
				"token": "test-token",
				"host":  "https://www.wrike.com/api/v4",
			},
		},
		{
			name:      "skips when no old config",
			oldConfig: nil,
			migration: ProviderMigration{
				Name:    "wrike",
				YAMLKey: "wrike",
			},
			wantMigrated: false,
		},
		{
			name: "skips when provider not in old config",
			oldConfig: map[string]any{
				"github": map[string]any{
					"token": "gh-token",
				},
			},
			migration: ProviderMigration{
				Name:    "wrike",
				YAMLKey: "wrike",
			},
			wantMigrated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Write old config if provided
			if tt.oldConfig != nil {
				mehrhofDir := filepath.Join(tmpDir, ".mehrhof")
				if err := os.MkdirAll(mehrhofDir, 0o755); err != nil {
					t.Fatal(err)
				}

				data, err := yaml.Marshal(tt.oldConfig)
				if err != nil {
					t.Fatal(err)
				}

				if err := os.WriteFile(filepath.Join(mehrhofDir, "config.yaml"), data, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			// Run migration
			migrated, err := MigrateProviderConfig(tmpDir, tt.migration)
			if err != nil {
				t.Fatalf("MigrateProviderConfig() error = %v", err)
			}

			if migrated != tt.wantMigrated {
				t.Errorf("MigrateProviderConfig() migrated = %v, want %v", migrated, tt.wantMigrated)
			}

			// Check new config if migration happened
			if tt.wantMigrated {
				newConfigPath := filepath.Join(tmpDir, ".crealfy", tt.migration.Name+".yaml")
				data, err := os.ReadFile(newConfigPath)
				if err != nil {
					t.Fatalf("Failed to read new config: %v", err)
				}

				var gotConfig map[string]any
				if err := yaml.Unmarshal(data, &gotConfig); err != nil {
					t.Fatalf("Failed to parse new config: %v", err)
				}

				for key, wantVal := range tt.wantNewConfig {
					if gotVal, ok := gotConfig[key]; !ok {
						t.Errorf("Missing key %q in new config", key)
					} else if gotVal != wantVal {
						t.Errorf("Key %q = %v, want %v", key, gotVal, wantVal)
					}
				}
			}
		})
	}
}

func TestMigrateProviderConfig_SkipsExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create old config
	mehrhofDir := filepath.Join(tmpDir, ".mehrhof")
	if err := os.MkdirAll(mehrhofDir, 0o755); err != nil {
		t.Fatal(err)
	}

	oldConfig := map[string]any{
		"wrike": map[string]any{
			"token": "old-token",
		},
	}
	data, _ := yaml.Marshal(oldConfig)
	if err := os.WriteFile(filepath.Join(mehrhofDir, "config.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create existing new config
	crealyfDir := filepath.Join(tmpDir, ".crealfy")
	if err := os.MkdirAll(crealyfDir, 0o755); err != nil {
		t.Fatal(err)
	}

	existingConfig := map[string]any{
		"token": "existing-token",
	}
	data, _ = yaml.Marshal(existingConfig)
	if err := os.WriteFile(filepath.Join(crealyfDir, "wrike.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Run migration
	migration := ProviderMigration{
		Name:    "wrike",
		YAMLKey: "wrike",
	}

	migrated, err := MigrateProviderConfig(tmpDir, migration)
	if err != nil {
		t.Fatalf("MigrateProviderConfig() error = %v", err)
	}

	if migrated {
		t.Error("Expected migration to be skipped when new config exists")
	}

	// Verify existing config wasn't overwritten
	data, _ = os.ReadFile(filepath.Join(crealyfDir, "wrike.yaml"))
	var gotConfig map[string]any
	_ = yaml.Unmarshal(data, &gotConfig)

	if gotConfig["token"] != "existing-token" {
		t.Error("Existing config was overwritten")
	}
}

func TestStandardProviderMigrations(t *testing.T) {
	migrations := StandardProviderMigrations()

	if len(migrations) == 0 {
		t.Error("Expected at least one migration definition")
	}

	// Verify wrike migration exists and has correct structure
	var wrikeMigration *ProviderMigration
	for i := range migrations {
		if migrations[i].Name == "wrike" {
			wrikeMigration = &migrations[i]

			break
		}
	}

	if wrikeMigration == nil {
		t.Fatal("Wrike migration not found")
	}

	if wrikeMigration.YAMLKey != "wrike" {
		t.Errorf("Wrike YAMLKey = %q, want %q", wrikeMigration.YAMLKey, "wrike")
	}

	// Check field mappings
	expectedMappings := map[string]string{
		"folder":  "folder_id",
		"project": "project_id",
		"space":   "space_id",
	}

	for oldKey, expectedNewKey := range expectedMappings {
		if newKey, ok := wrikeMigration.FieldMap[oldKey]; !ok {
			t.Errorf("Missing field mapping for %q", oldKey)
		} else if newKey != expectedNewKey {
			t.Errorf("Field mapping for %q = %q, want %q", oldKey, newKey, expectedNewKey)
		}
	}
}
