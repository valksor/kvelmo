package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/valksor/go-toolkit/paths"
)

func TestNewI18nOverrides(t *testing.T) {
	o := NewI18nOverrides()

	if o == nil {
		t.Fatal("NewI18nOverrides returned nil")
	}
	if o.Terminology == nil {
		t.Error("Terminology map should be initialized, got nil")
	}
	if o.Keys == nil {
		t.Error("Keys map should be initialized, got nil")
	}
	if len(o.Terminology) != 0 {
		t.Errorf("Terminology should be empty, got %d items", len(o.Terminology))
	}
	if len(o.Keys) != 0 {
		t.Errorf("Keys should be empty, got %d items", len(o.Keys))
	}
}

func TestMergeI18nOverrides(t *testing.T) {
	tests := []struct {
		name    string
		global  *I18nOverrides
		project *I18nOverrides
		want    func(*I18nOverrides) bool
	}{
		{
			name:    "nil global and nil project",
			global:  nil,
			project: nil,
			want: func(o *I18nOverrides) bool {
				return len(o.Terminology) == 0 && len(o.Keys) == 0
			},
		},
		{
			name: "global only",
			global: &I18nOverrides{
				Terminology: map[string]string{"Task": "Ticket"},
				Keys:        map[string]map[string]string{"en": {"nav.dashboard": "Home"}},
			},
			project: nil,
			want: func(o *I18nOverrides) bool {
				return o.Terminology["Task"] == "Ticket" &&
					o.Keys["en"]["nav.dashboard"] == "Home"
			},
		},
		{
			name:   "project only",
			global: nil,
			project: &I18nOverrides{
				Terminology: map[string]string{"Workflow": "Pipeline"},
				Keys:        map[string]map[string]string{"de": {"nav.settings": "Einstellungen"}},
			},
			want: func(o *I18nOverrides) bool {
				return o.Terminology["Workflow"] == "Pipeline" &&
					o.Keys["de"]["nav.settings"] == "Einstellungen"
			},
		},
		{
			name: "project overrides global same key",
			global: &I18nOverrides{
				Terminology: map[string]string{"Task": "Ticket"},
				Keys:        map[string]map[string]string{"en": {"nav.dashboard": "Home"}},
			},
			project: &I18nOverrides{
				Terminology: map[string]string{"Task": "Work Item"},
				Keys:        map[string]map[string]string{"en": {"nav.dashboard": "Dashboard"}},
			},
			want: func(o *I18nOverrides) bool {
				return o.Terminology["Task"] == "Work Item" &&
					o.Keys["en"]["nav.dashboard"] == "Dashboard"
			},
		},
		{
			name: "merge different keys",
			global: &I18nOverrides{
				Terminology: map[string]string{"Task": "Ticket"},
				Keys:        map[string]map[string]string{"en": {"nav.dashboard": "Home"}},
			},
			project: &I18nOverrides{
				Terminology: map[string]string{"Workflow": "Pipeline"},
				Keys:        map[string]map[string]string{"en": {"nav.settings": "Config"}},
			},
			want: func(o *I18nOverrides) bool {
				return o.Terminology["Task"] == "Ticket" &&
					o.Terminology["Workflow"] == "Pipeline" &&
					o.Keys["en"]["nav.dashboard"] == "Home" &&
					o.Keys["en"]["nav.settings"] == "Config"
			},
		},
		{
			name: "merge different languages",
			global: &I18nOverrides{
				Terminology: map[string]string{},
				Keys:        map[string]map[string]string{"en": {"nav.dashboard": "Dashboard"}},
			},
			project: &I18nOverrides{
				Terminology: map[string]string{},
				Keys:        map[string]map[string]string{"de": {"nav.dashboard": "Übersicht"}},
			},
			want: func(o *I18nOverrides) bool {
				return o.Keys["en"]["nav.dashboard"] == "Dashboard" &&
					o.Keys["de"]["nav.dashboard"] == "Übersicht"
			},
		},
		{
			name: "empty global with project",
			global: &I18nOverrides{
				Terminology: map[string]string{},
				Keys:        map[string]map[string]string{},
			},
			project: &I18nOverrides{
				Terminology: map[string]string{"Test": "Value"},
				Keys:        map[string]map[string]string{},
			},
			want: func(o *I18nOverrides) bool {
				return o.Terminology["Test"] == "Value"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeI18nOverrides(tt.global, tt.project)

			if result == nil {
				t.Fatal("MergeI18nOverrides returned nil")
			}
			if result.Terminology == nil {
				t.Error("Terminology map should be initialized")
			}
			if result.Keys == nil {
				t.Error("Keys map should be initialized")
			}
			if !tt.want(result) {
				t.Errorf("Merge result did not match expected, got: %+v", result)
			}
		})
	}
}

func TestGetI18nOverridesPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	tests := []struct {
		name        string
		projectName string
		wantSuffix  string
	}{
		{
			name:        "global path",
			projectName: "",
			wantSuffix:  filepath.Join("i18n", "overrides.json"),
		},
		{
			name:        "project path",
			projectName: "my-project",
			wantSuffix:  filepath.Join("i18n", "projects", "my-project", "overrides.json"),
		},
		{
			name:        "project with special chars",
			projectName: "github.com-user-repo",
			wantSuffix:  filepath.Join("i18n", "projects", "github.com-user-repo", "overrides.json"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := GetI18nOverridesPath(tt.projectName)
			if err != nil {
				t.Fatalf("GetI18nOverridesPath error: %v", err)
			}

			if !filepath.IsAbs(path) {
				t.Errorf("Path should be absolute, got: %s", path)
			}

			expectedSuffix := filepath.Join("mehrhof", tt.wantSuffix)
			if !i18nPathContains(path, expectedSuffix) {
				t.Errorf("Path = %s, want suffix %s", path, expectedSuffix)
			}
		})
	}
}

func TestLoadI18nOverrides(t *testing.T) {
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	t.Run("file does not exist returns empty", func(t *testing.T) {
		overrides, err := LoadI18nOverrides("")
		if err != nil {
			t.Fatalf("LoadI18nOverrides error: %v", err)
		}
		if overrides == nil {
			t.Fatal("Expected non-nil overrides")
		}
		if len(overrides.Terminology) != 0 {
			t.Errorf("Expected empty Terminology, got %d items", len(overrides.Terminology))
		}
		if len(overrides.Keys) != 0 {
			t.Errorf("Expected empty Keys, got %d items", len(overrides.Keys))
		}
	})

	t.Run("valid JSON loads correctly", func(t *testing.T) {
		// Create valid JSON file
		path, _ := GetI18nOverridesPath("")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll error: %v", err)
		}

		data := `{
			"terminology": {"Task": "Ticket"},
			"keys": {"en": {"nav.dashboard": "Home"}}
		}`
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		overrides, err := LoadI18nOverrides("")
		if err != nil {
			t.Fatalf("LoadI18nOverrides error: %v", err)
		}
		if overrides.Terminology["Task"] != "Ticket" {
			t.Errorf("Terminology[Task] = %q, want %q", overrides.Terminology["Task"], "Ticket")
		}
		if overrides.Keys["en"]["nav.dashboard"] != "Home" {
			t.Errorf("Keys[en][nav.dashboard] = %q, want %q", overrides.Keys["en"]["nav.dashboard"], "Home")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		homeDir2 := t.TempDir()
		t.Cleanup(paths.SetHomeDirForTesting(homeDir2))

		path, _ := GetI18nOverridesPath("")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll error: %v", err)
		}
		if err := os.WriteFile(path, []byte("{invalid json}"), 0o644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		_, err := LoadI18nOverrides("")
		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}
	})

	t.Run("partial JSON initializes missing maps", func(t *testing.T) {
		homeDir3 := t.TempDir()
		t.Cleanup(paths.SetHomeDirForTesting(homeDir3))

		path, _ := GetI18nOverridesPath("")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll error: %v", err)
		}

		// JSON with only terminology, no keys
		data := `{"terminology": {"Task": "Ticket"}}`
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		overrides, err := LoadI18nOverrides("")
		if err != nil {
			t.Fatalf("LoadI18nOverrides error: %v", err)
		}
		if overrides.Terminology == nil {
			t.Error("Terminology should be initialized")
		}
		if overrides.Keys == nil {
			t.Error("Keys should be initialized even when not in JSON")
		}
		if overrides.Terminology["Task"] != "Ticket" {
			t.Errorf("Terminology[Task] = %q, want %q", overrides.Terminology["Task"], "Ticket")
		}
	})
}

func TestSaveI18nOverrides(t *testing.T) {
	t.Run("creates directory and saves file", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Cleanup(paths.SetHomeDirForTesting(homeDir))

		overrides := &I18nOverrides{
			Terminology: map[string]string{"Task": "Ticket"},
			Keys:        map[string]map[string]string{"en": {"nav.dashboard": "Home"}},
		}

		err := SaveI18nOverrides("", overrides)
		if err != nil {
			t.Fatalf("SaveI18nOverrides error: %v", err)
		}

		path, _ := GetI18nOverridesPath("")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("File should exist after save")
		}

		// Verify content
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile error: %v", err)
		}

		var loaded I18nOverrides
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if loaded.Terminology["Task"] != "Ticket" {
			t.Errorf("Saved Terminology[Task] = %q, want %q", loaded.Terminology["Task"], "Ticket")
		}
	})

	t.Run("creates project directory", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Cleanup(paths.SetHomeDirForTesting(homeDir))

		overrides := &I18nOverrides{
			Terminology: map[string]string{"Workflow": "Pipeline"},
			Keys:        map[string]map[string]string{},
		}

		err := SaveI18nOverrides("test-project", overrides)
		if err != nil {
			t.Fatalf("SaveI18nOverrides error: %v", err)
		}

		path, _ := GetI18nOverridesPath("test-project")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("Project file should exist after save")
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Cleanup(paths.SetHomeDirForTesting(homeDir))

		// First save
		first := &I18nOverrides{
			Terminology: map[string]string{"Task": "Old"},
			Keys:        map[string]map[string]string{},
		}
		if err := SaveI18nOverrides("", first); err != nil {
			t.Fatalf("First save error: %v", err)
		}

		// Second save
		second := &I18nOverrides{
			Terminology: map[string]string{"Task": "New"},
			Keys:        map[string]map[string]string{},
		}
		if err := SaveI18nOverrides("", second); err != nil {
			t.Fatalf("Second save error: %v", err)
		}

		// Verify overwritten
		loaded, err := LoadI18nOverrides("")
		if err != nil {
			t.Fatalf("LoadI18nOverrides error: %v", err)
		}
		if loaded.Terminology["Task"] != "New" {
			t.Errorf("Terminology[Task] = %q, want %q", loaded.Terminology["Task"], "New")
		}
	})

	t.Run("uses 2-space indentation", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Cleanup(paths.SetHomeDirForTesting(homeDir))

		overrides := &I18nOverrides{
			Terminology: map[string]string{"Task": "Ticket"},
			Keys:        map[string]map[string]string{},
		}
		if err := SaveI18nOverrides("", overrides); err != nil {
			t.Fatalf("SaveI18nOverrides error: %v", err)
		}

		path, _ := GetI18nOverridesPath("")
		data, _ := os.ReadFile(path)

		// Check for 2-space indentation (not tabs)
		content := string(data)
		if !i18nStringContains(content, "  \"terminology\"") {
			t.Error("Expected 2-space indentation in saved JSON")
		}
	})
}

func TestLoadMergedI18nOverrides(t *testing.T) {
	t.Run("empty project returns global only", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Cleanup(paths.SetHomeDirForTesting(homeDir))

		// Save global overrides
		global := &I18nOverrides{
			Terminology: map[string]string{"Task": "Ticket"},
			Keys:        map[string]map[string]string{},
		}
		if err := SaveI18nOverrides("", global); err != nil {
			t.Fatalf("SaveI18nOverrides error: %v", err)
		}

		merged, err := LoadMergedI18nOverrides("")
		if err != nil {
			t.Fatalf("LoadMergedI18nOverrides error: %v", err)
		}
		if merged.Terminology["Task"] != "Ticket" {
			t.Errorf("Terminology[Task] = %q, want %q", merged.Terminology["Task"], "Ticket")
		}
	})

	t.Run("merges global and project", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Cleanup(paths.SetHomeDirForTesting(homeDir))

		// Save global
		global := &I18nOverrides{
			Terminology: map[string]string{"Task": "Ticket", "Workflow": "Process"},
			Keys:        map[string]map[string]string{"en": {"nav.dashboard": "Home"}},
		}
		if err := SaveI18nOverrides("", global); err != nil {
			t.Fatalf("SaveI18nOverrides global error: %v", err)
		}

		// Save project (overrides Task, adds new key)
		project := &I18nOverrides{
			Terminology: map[string]string{"Task": "Work Item"},
			Keys:        map[string]map[string]string{"en": {"nav.settings": "Config"}},
		}
		if err := SaveI18nOverrides("test-project", project); err != nil {
			t.Fatalf("SaveI18nOverrides project error: %v", err)
		}

		merged, err := LoadMergedI18nOverrides("test-project")
		if err != nil {
			t.Fatalf("LoadMergedI18nOverrides error: %v", err)
		}

		// Project overrides global
		if merged.Terminology["Task"] != "Work Item" {
			t.Errorf("Terminology[Task] = %q, want %q", merged.Terminology["Task"], "Work Item")
		}
		// Global preserved
		if merged.Terminology["Workflow"] != "Process" {
			t.Errorf("Terminology[Workflow] = %q, want %q", merged.Terminology["Workflow"], "Process")
		}
		// Both keys present
		if merged.Keys["en"]["nav.dashboard"] != "Home" {
			t.Errorf("Keys[en][nav.dashboard] = %q, want %q", merged.Keys["en"]["nav.dashboard"], "Home")
		}
		if merged.Keys["en"]["nav.settings"] != "Config" {
			t.Errorf("Keys[en][nav.settings] = %q, want %q", merged.Keys["en"]["nav.settings"], "Config")
		}
	})

	t.Run("handles missing files gracefully", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Cleanup(paths.SetHomeDirForTesting(homeDir))

		// No files exist
		merged, err := LoadMergedI18nOverrides("nonexistent-project")
		if err != nil {
			t.Fatalf("LoadMergedI18nOverrides error: %v", err)
		}
		if merged == nil {
			t.Fatal("Expected non-nil result")
		}
		if len(merged.Terminology) != 0 {
			t.Errorf("Expected empty Terminology, got %d items", len(merged.Terminology))
		}
	})
}

// Helper functions

func i18nPathContains(path, suffix string) bool {
	// Normalize path separators and check if suffix is in path
	normalizedPath := filepath.ToSlash(path)
	normalizedSuffix := filepath.ToSlash(suffix)

	for i := 0; i <= len(normalizedPath)-len(normalizedSuffix); i++ {
		if normalizedPath[i:i+len(normalizedSuffix)] == normalizedSuffix {
			return true
		}
	}

	return false
}

func i18nStringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
