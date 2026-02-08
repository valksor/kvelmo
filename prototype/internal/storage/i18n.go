package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// i18nDir is the subdirectory for i18n files in MehrhofHomeDir.
const i18nDir = "i18n"

// i18nProjectsDir is the subdirectory for project-specific overrides.
const i18nProjectsDir = "projects"

// i18nOverridesFile is the filename for override files.
const i18nOverridesFile = "overrides.json"

// I18nOverrides holds user customizations for translations.
// Supports two types of customization:
//   - Terminology: find/replace terms across all translations (case-insensitive)
//   - Keys: override specific translation keys per language
type I18nOverrides struct {
	// Terminology holds find/replace pairs applied across all translations.
	// Example: {"Task": "Ticket", "Workflow": "Pipeline"}
	Terminology map[string]string `json:"terminology"`

	// Keys holds direct translation key overrides per language.
	// Example: {"en": {"nav.dashboard": "Home"}}
	Keys map[string]map[string]string `json:"keys"`
}

// NewI18nOverrides creates an empty I18nOverrides struct.
func NewI18nOverrides() *I18nOverrides {
	return &I18nOverrides{
		Terminology: make(map[string]string),
		Keys:        make(map[string]map[string]string),
	}
}

// MergeI18nOverrides merges global and project overrides.
// Project overrides take precedence over global overrides.
func MergeI18nOverrides(global, project *I18nOverrides) *I18nOverrides {
	merged := NewI18nOverrides()

	// Copy global terminology
	if global != nil {
		for k, v := range global.Terminology {
			merged.Terminology[k] = v
		}
	}
	// Override with project terminology
	if project != nil {
		for k, v := range project.Terminology {
			merged.Terminology[k] = v
		}
	}

	// Merge keys per language
	if global != nil {
		for lang, keys := range global.Keys {
			if merged.Keys[lang] == nil {
				merged.Keys[lang] = make(map[string]string)
			}
			for k, v := range keys {
				merged.Keys[lang][k] = v
			}
		}
	}
	// Override with project keys
	if project != nil {
		for lang, keys := range project.Keys {
			if merged.Keys[lang] == nil {
				merged.Keys[lang] = make(map[string]string)
			}
			for k, v := range keys {
				merged.Keys[lang][k] = v
			}
		}
	}

	return merged
}

// GetI18nOverridesPath returns the path to the i18n overrides file.
// If projectName is empty, returns the global overrides path.
// Otherwise, returns the project-specific overrides path.
func GetI18nOverridesPath(projectName string) (string, error) {
	homeDir, err := GetMehrhofHomeDir()
	if err != nil {
		return "", err
	}

	if projectName == "" {
		// Global overrides: ~/.valksor/mehrhof/i18n/overrides.json
		return filepath.Join(homeDir, i18nDir, i18nOverridesFile), nil
	}

	// Project overrides: ~/.valksor/mehrhof/i18n/projects/<project-name>/overrides.json
	return filepath.Join(homeDir, i18nDir, i18nProjectsDir, projectName, i18nOverridesFile), nil
}

// LoadI18nOverrides loads i18n overrides from disk.
// If projectName is empty, loads global overrides.
// Returns empty overrides if file doesn't exist.
func LoadI18nOverrides(projectName string) (*I18nOverrides, error) {
	path, err := GetI18nOverridesPath(projectName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewI18nOverrides(), nil
		}

		return nil, err
	}

	var overrides I18nOverrides
	if err := json.Unmarshal(data, &overrides); err != nil {
		return nil, err
	}

	// Ensure maps are initialized
	if overrides.Terminology == nil {
		overrides.Terminology = make(map[string]string)
	}
	if overrides.Keys == nil {
		overrides.Keys = make(map[string]map[string]string)
	}

	return &overrides, nil
}

// SaveI18nOverrides saves i18n overrides to disk.
// If projectName is empty, saves to global overrides.
// Creates the directory structure if needed.
func SaveI18nOverrides(projectName string, overrides *I18nOverrides) error {
	path, err := GetI18nOverridesPath(projectName)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(overrides, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// LoadMergedI18nOverrides loads and merges global and project overrides.
// Returns the merged result where project overrides take precedence.
func LoadMergedI18nOverrides(projectName string) (*I18nOverrides, error) {
	global, err := LoadI18nOverrides("")
	if err != nil {
		return nil, err
	}

	if projectName == "" {
		return global, nil
	}

	project, err := LoadI18nOverrides(projectName)
	if err != nil {
		return nil, err
	}

	return MergeI18nOverrides(global, project), nil
}
