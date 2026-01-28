package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectLanguage_String(t *testing.T) {
	tests := []struct {
		lang     ProjectLanguage
		expected string
	}{
		{LangGo, "go"},
		{LangJavaScript, "javascript"},
		{LangTypeScript, "typescript"},
		{LangPython, "python"},
		{LangPHP, "php"},
		{LangRuby, "ruby"},
		{LangRust, "rust"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.lang.String() != tt.expected {
				t.Errorf("String() = %v, want %v", tt.lang.String(), tt.expected)
			}
		})
	}
}

func TestProjectLanguage_DisplayName(t *testing.T) {
	tests := []struct {
		lang     ProjectLanguage
		expected string
	}{
		{LangGo, "Go"},
		{LangJavaScript, "JavaScript"},
		{LangTypeScript, "TypeScript"},
		{LangPython, "Python"},
		{LangPHP, "PHP"},
		{LangRuby, "Ruby"},
		{LangRust, "Rust"},
		{ProjectLanguage("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.lang.DisplayName() != tt.expected {
				t.Errorf("DisplayName() = %v, want %v", tt.lang.DisplayName(), tt.expected)
			}
		})
	}
}

func TestProjectInfo_HasLanguage(t *testing.T) {
	info := ProjectInfo{
		Languages: []ProjectLanguage{LangGo, LangPython},
	}

	tests := []struct {
		lang     ProjectLanguage
		expected bool
	}{
		{LangGo, true},
		{LangPython, true},
		{LangJavaScript, false},
		{LangRust, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			if info.HasLanguage(tt.lang) != tt.expected {
				t.Errorf("HasLanguage(%v) = %v, want %v", tt.lang, info.HasLanguage(tt.lang), tt.expected)
			}
		})
	}
}

func TestProjectInfo_HasAnyLanguage(t *testing.T) {
	info := ProjectInfo{
		Languages: []ProjectLanguage{LangGo, LangPython},
	}

	tests := []struct {
		name     string
		langs    []ProjectLanguage
		expected bool
	}{
		{"has go", []ProjectLanguage{LangGo}, true},
		{"has python", []ProjectLanguage{LangPython}, true},
		{"has either go or javascript", []ProjectLanguage{LangGo, LangJavaScript}, true},
		{"has javascript or rust", []ProjectLanguage{LangJavaScript, LangRust}, false},
		{"empty list", []ProjectLanguage{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if info.HasAnyLanguage(tt.langs...) != tt.expected {
				t.Errorf("HasAnyLanguage(%v) = %v, want %v", tt.langs, info.HasAnyLanguage(tt.langs...), tt.expected)
			}
		})
	}
}

func TestProjectInfo_IsMultiLanguage(t *testing.T) {
	tests := []struct {
		name     string
		langs    []ProjectLanguage
		expected bool
	}{
		{"empty", []ProjectLanguage{}, false},
		{"single", []ProjectLanguage{LangGo}, false},
		{"double", []ProjectLanguage{LangGo, LangPython}, true},
		{"triple", []ProjectLanguage{LangGo, LangPython, LangJavaScript}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ProjectInfo{Languages: tt.langs}
			if info.IsMultiLanguage() != tt.expected {
				t.Errorf("IsMultiLanguage() = %v, want %v", info.IsMultiLanguage(), tt.expected)
			}
		})
	}
}

func TestDetectProject(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name            string
		setupFiles      []string
		expectedLangs   []ProjectLanguage
		expectedGoMod   bool
		expectedPkgJSON bool
	}{
		{
			name:            "empty directory",
			setupFiles:      []string{},
			expectedLangs:   []ProjectLanguage{},
			expectedGoMod:   false,
			expectedPkgJSON: false,
		},
		{
			name:            "go project",
			setupFiles:      []string{"go.mod"},
			expectedLangs:   []ProjectLanguage{LangGo},
			expectedGoMod:   true,
			expectedPkgJSON: false,
		},
		{
			name:            "javascript project",
			setupFiles:      []string{"package.json"},
			expectedLangs:   []ProjectLanguage{LangJavaScript},
			expectedGoMod:   false,
			expectedPkgJSON: true,
		},
		{
			name:            "typescript project",
			setupFiles:      []string{"package.json", "tsconfig.json"},
			expectedLangs:   []ProjectLanguage{LangTypeScript},
			expectedGoMod:   false,
			expectedPkgJSON: true,
		},
		{
			name:            "python project with requirements",
			setupFiles:      []string{"requirements.txt"},
			expectedLangs:   []ProjectLanguage{LangPython},
			expectedGoMod:   false,
			expectedPkgJSON: false,
		},
		{
			name:            "python project with pyproject",
			setupFiles:      []string{"pyproject.toml"},
			expectedLangs:   []ProjectLanguage{LangPython},
			expectedGoMod:   false,
			expectedPkgJSON: false,
		},
		{
			name:            "multi-language project",
			setupFiles:      []string{"go.mod", "package.json", "requirements.txt"},
			expectedLangs:   []ProjectLanguage{LangGo, LangJavaScript, LangPython},
			expectedGoMod:   true,
			expectedPkgJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test subdirectory
			testDir := filepath.Join(tempDir, tt.name)
			if err := os.MkdirAll(testDir, 0o755); err != nil {
				t.Fatalf("failed to create test dir: %v", err)
			}

			// Create marker files
			for _, file := range tt.setupFiles {
				path := filepath.Join(testDir, file)
				if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
					t.Fatalf("failed to create file %s: %v", file, err)
				}
			}

			// Run detection
			info := DetectProject(testDir)

			// Verify languages
			if len(info.Languages) != len(tt.expectedLangs) {
				t.Errorf("detected %d languages, want %d: got %v, want %v",
					len(info.Languages), len(tt.expectedLangs), info.Languages, tt.expectedLangs)
			}

			for _, expectedLang := range tt.expectedLangs {
				if !info.HasLanguage(expectedLang) {
					t.Errorf("expected language %v not detected", expectedLang)
				}
			}

			// Verify marker files
			if info.HasGoMod != tt.expectedGoMod {
				t.Errorf("HasGoMod = %v, want %v", info.HasGoMod, tt.expectedGoMod)
			}
			if info.HasPackageJSON != tt.expectedPkgJSON {
				t.Errorf("HasPackageJSON = %v, want %v", info.HasPackageJSON, tt.expectedPkgJSON)
			}
		})
	}
}

func TestDetectProject_AllMarkerFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create all marker files
	files := []string{
		"go.mod",
		"package.json",
		"package-lock.json",
		"yarn.lock",
		"tsconfig.json",
		"pyproject.toml",
		"requirements.txt",
		"setup.py",
		"Pipfile",
		"composer.json",
		"Gemfile",
		"Cargo.toml",
	}

	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatalf("failed to create file %s: %v", file, err)
		}
	}

	info := DetectProject(tempDir)

	// All marker file flags should be true
	checks := []struct {
		name  string
		value bool
	}{
		{"HasGoMod", info.HasGoMod},
		{"HasPackageJSON", info.HasPackageJSON},
		{"HasPackageLockJSON", info.HasPackageLockJSON},
		{"HasYarnLock", info.HasYarnLock},
		{"HasTSConfig", info.HasTSConfig},
		{"HasPyProjectTOML", info.HasPyProjectTOML},
		{"HasRequirementsTXT", info.HasRequirementsTXT},
		{"HasSetupPy", info.HasSetupPy},
		{"HasPipfile", info.HasPipfile},
		{"HasComposerJSON", info.HasComposerJSON},
		{"HasGemfile", info.HasGemfile},
		{"HasCargoTOML", info.HasCargoTOML},
	}

	for _, check := range checks {
		if !check.value {
			t.Errorf("%s should be true", check.name)
		}
	}

	// Should have all languages except JavaScript (TypeScript takes precedence)
	expectedLangs := []ProjectLanguage{LangGo, LangTypeScript, LangPython, LangPHP, LangRuby, LangRust}
	for _, lang := range expectedLangs {
		if !info.HasLanguage(lang) {
			t.Errorf("expected language %v not detected", lang)
		}
	}

	// JavaScript should NOT be detected because TypeScript takes precedence
	if info.HasLanguage(LangJavaScript) {
		t.Error("JavaScript should not be detected when TypeScript is present")
	}
}

func TestAvailableScanners(t *testing.T) {
	scanners := AvailableScanners()

	if len(scanners) == 0 {
		t.Error("expected at least one scanner")
	}

	// Check that required scanners exist
	expectedScanners := []string{"semgrep", "gitleaks", "gosec", "govulncheck", "npm-audit", "eslint-security", "bandit", "pip-audit"}
	scannerNames := make(map[string]bool)
	for _, s := range scanners {
		scannerNames[s.Name] = true
	}

	for _, expected := range expectedScanners {
		if !scannerNames[expected] {
			t.Errorf("expected scanner %s not found", expected)
		}
	}
}

func TestGetApplicableScanners(t *testing.T) {
	tests := []struct {
		name                string
		info                ProjectInfo
		expectedContains    []string
		expectedNotContains []string
	}{
		{
			name: "go project",
			info: ProjectInfo{
				Languages: []ProjectLanguage{LangGo},
				HasGoMod:  true,
			},
			expectedContains:    []string{"semgrep", "gitleaks", "gosec", "govulncheck"},
			expectedNotContains: []string{"npm-audit", "bandit", "pip-audit"},
		},
		{
			name: "javascript project",
			info: ProjectInfo{
				Languages:      []ProjectLanguage{LangJavaScript},
				HasPackageJSON: true,
			},
			expectedContains:    []string{"semgrep", "gitleaks", "npm-audit", "eslint-security"},
			expectedNotContains: []string{"gosec", "bandit"},
		},
		{
			name: "python project",
			info: ProjectInfo{
				Languages:          []ProjectLanguage{LangPython},
				HasRequirementsTXT: true,
			},
			expectedContains:    []string{"semgrep", "gitleaks", "bandit", "pip-audit"},
			expectedNotContains: []string{"gosec", "npm-audit"},
		},
		{
			name: "empty project",
			info: ProjectInfo{
				Languages: []ProjectLanguage{},
			},
			// Cross-language scanners should still be shown
			expectedContains:    []string{"semgrep", "gitleaks"},
			expectedNotContains: []string{"gosec", "npm-audit", "bandit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanners := GetApplicableScanners(tt.info)
			scannerNames := make(map[string]bool)
			for _, s := range scanners {
				scannerNames[s.Name] = true
			}

			for _, expected := range tt.expectedContains {
				if !scannerNames[expected] {
					t.Errorf("expected scanner %s not found", expected)
				}
			}

			for _, notExpected := range tt.expectedNotContains {
				if scannerNames[notExpected] {
					t.Errorf("unexpected scanner %s found", notExpected)
				}
			}
		})
	}
}

func TestGetScannersByType(t *testing.T) {
	info := ProjectInfo{
		Languages: []ProjectLanguage{LangGo, LangPython},
		HasGoMod:  true,
	}

	tests := []struct {
		scannerType string
		minCount    int
	}{
		{"sast", 3},       // semgrep, gosec, bandit
		{"dependency", 2}, // govulncheck, pip-audit
		{"secrets", 1},    // gitleaks
	}

	for _, tt := range tests {
		t.Run(tt.scannerType, func(t *testing.T) {
			scanners := GetScannersByType(info, tt.scannerType)
			if len(scanners) < tt.minCount {
				t.Errorf("expected at least %d %s scanners, got %d", tt.minCount, tt.scannerType, len(scanners))
			}

			// Verify all returned scanners are of the correct type
			for _, s := range scanners {
				if s.Type != tt.scannerType {
					t.Errorf("scanner %s has type %s, expected %s", s.Name, s.Type, tt.scannerType)
				}
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file
	filePath := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a directory
	dirPath := filepath.Join(tempDir, "testdir")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing file", filePath, true},
		{"directory", dirPath, false},
		{"non-existent", filepath.Join(tempDir, "nonexistent"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if fileExists(tt.path) != tt.expected {
				t.Errorf("fileExists(%s) = %v, want %v", tt.path, fileExists(tt.path), tt.expected)
			}
		})
	}
}

func TestScannerInfo_Fields(t *testing.T) {
	scanners := AvailableScanners()

	for _, s := range scanners {
		t.Run(s.Name, func(t *testing.T) {
			// Every scanner should have required fields
			if s.Name == "" {
				t.Error("scanner Name is empty")
			}
			if s.DisplayName == "" {
				t.Error("scanner DisplayName is empty")
			}
			if s.Description == "" {
				t.Error("scanner Description is empty")
			}
			if s.Type == "" {
				t.Error("scanner Type is empty")
			}
			if s.InstallCommand == "" {
				t.Error("scanner InstallCommand is empty")
			}

			// Type should be one of known types
			validTypes := map[string]bool{
				"sast":       true,
				"dependency": true,
				"secrets":    true,
			}
			if !validTypes[s.Type] {
				t.Errorf("scanner Type %s is not valid", s.Type)
			}
		})
	}
}
