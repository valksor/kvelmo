package security

import (
	"os"
	"path/filepath"
)

// ProjectLanguage represents a detected programming language.
type ProjectLanguage string

const (
	LangGo         ProjectLanguage = "go"
	LangJavaScript ProjectLanguage = "javascript"
	LangTypeScript ProjectLanguage = "typescript"
	LangPython     ProjectLanguage = "python"
	LangPHP        ProjectLanguage = "php"
	LangRuby       ProjectLanguage = "ruby"
	LangRust       ProjectLanguage = "rust"
)

// String returns the string representation of the language.
func (l ProjectLanguage) String() string {
	return string(l)
}

// DisplayName returns a human-readable name for the language.
func (l ProjectLanguage) DisplayName() string {
	switch l {
	case LangGo:
		return "Go"
	case LangJavaScript:
		return "JavaScript"
	case LangTypeScript:
		return "TypeScript"
	case LangPython:
		return "Python"
	case LangPHP:
		return "PHP"
	case LangRuby:
		return "Ruby"
	case LangRust:
		return "Rust"
	default:
		return string(l)
	}
}

// ProjectInfo contains detected project information.
type ProjectInfo struct {
	// Detected languages in the project
	Languages []ProjectLanguage

	// Marker files detected (for more specific scanner recommendations)
	HasGoMod           bool // go.mod
	HasPackageJSON     bool // package.json
	HasPackageLockJSON bool // package-lock.json (npm)
	HasYarnLock        bool // yarn.lock
	HasTSConfig        bool // tsconfig.json
	HasPyProjectTOML   bool // pyproject.toml
	HasRequirementsTXT bool // requirements.txt
	HasSetupPy         bool // setup.py
	HasPipfile         bool // Pipfile (pipenv)
	HasComposerJSON    bool // composer.json (PHP)
	HasGemfile         bool // Gemfile (Ruby)
	HasCargoTOML       bool // Cargo.toml (Rust)
}

// HasLanguage returns true if the project includes the specified language.
func (p ProjectInfo) HasLanguage(lang ProjectLanguage) bool {
	for _, l := range p.Languages {
		if l == lang {
			return true
		}
	}

	return false
}

// HasAnyLanguage returns true if the project includes any of the specified languages.
func (p ProjectInfo) HasAnyLanguage(langs ...ProjectLanguage) bool {
	for _, lang := range langs {
		if p.HasLanguage(lang) {
			return true
		}
	}

	return false
}

// IsMultiLanguage returns true if multiple languages were detected.
func (p ProjectInfo) IsMultiLanguage() bool {
	return len(p.Languages) > 1
}

// DetectProject analyzes a directory and returns detected project information.
// It checks for common marker files to determine which languages and tools are used.
func DetectProject(dir string) ProjectInfo {
	info := ProjectInfo{
		Languages: make([]ProjectLanguage, 0),
	}

	// Check for Go (go.mod)
	if fileExists(filepath.Join(dir, "go.mod")) {
		info.HasGoMod = true
		info.Languages = append(info.Languages, LangGo)
	}

	// Check for JavaScript/TypeScript
	if fileExists(filepath.Join(dir, "package.json")) {
		info.HasPackageJSON = true
		// Check for TypeScript first (more specific)
		if fileExists(filepath.Join(dir, "tsconfig.json")) {
			info.HasTSConfig = true
			info.Languages = append(info.Languages, LangTypeScript)
		} else {
			info.Languages = append(info.Languages, LangJavaScript)
		}
	}

	// Check for lock files (useful for scanner recommendations)
	if fileExists(filepath.Join(dir, "package-lock.json")) {
		info.HasPackageLockJSON = true
	}
	if fileExists(filepath.Join(dir, "yarn.lock")) {
		info.HasYarnLock = true
	}

	// Check for Python
	hasPython := false
	if fileExists(filepath.Join(dir, "pyproject.toml")) {
		info.HasPyProjectTOML = true
		hasPython = true
	}
	if fileExists(filepath.Join(dir, "requirements.txt")) {
		info.HasRequirementsTXT = true
		hasPython = true
	}
	if fileExists(filepath.Join(dir, "setup.py")) {
		info.HasSetupPy = true
		hasPython = true
	}
	if fileExists(filepath.Join(dir, "Pipfile")) {
		info.HasPipfile = true
		hasPython = true
	}
	if hasPython {
		info.Languages = append(info.Languages, LangPython)
	}

	// Check for PHP (composer.json)
	if fileExists(filepath.Join(dir, "composer.json")) {
		info.HasComposerJSON = true
		info.Languages = append(info.Languages, LangPHP)
	}

	// Check for Ruby (Gemfile)
	if fileExists(filepath.Join(dir, "Gemfile")) {
		info.HasGemfile = true
		info.Languages = append(info.Languages, LangRuby)
	}

	// Check for Rust (Cargo.toml)
	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		info.HasCargoTOML = true
		info.Languages = append(info.Languages, LangRust)
	}

	return info
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// ScannerInfo describes a security scanner and its applicability.
type ScannerInfo struct {
	Name           string            // Scanner name (e.g., "gosec")
	DisplayName    string            // Human-readable name (e.g., "Gosec")
	Description    string            // Brief description
	Type           string            // "sast", "dependency", "secrets"
	Languages      []ProjectLanguage // Languages this scanner supports (empty = all)
	InstallCommand string            // Command to install the scanner
	Requires       string            // What marker file is required (e.g., "package-lock.json")
	AlwaysShow     bool              // Show regardless of detected languages (e.g., gitleaks)
}

// AvailableScanners returns information about all available scanners.
func AvailableScanners() []ScannerInfo {
	return []ScannerInfo{
		// Cross-language
		{
			Name:           "semgrep",
			DisplayName:    "Semgrep",
			Description:    "Cross-language SAST scanner supporting 30+ languages",
			Type:           "sast",
			Languages:      nil, // Supports all
			InstallCommand: "pip install semgrep",
			AlwaysShow:     true,
		},
		{
			Name:           "gitleaks",
			DisplayName:    "Gitleaks",
			Description:    "Secret detection in code and git history",
			Type:           "secrets",
			Languages:      nil, // Supports all
			InstallCommand: "brew install gitleaks",
			AlwaysShow:     true,
		},
		// Go
		{
			Name:           "gosec",
			DisplayName:    "Gosec",
			Description:    "Security scanner for Go code",
			Type:           "sast",
			Languages:      []ProjectLanguage{LangGo},
			InstallCommand: "go install github.com/securego/gosec/v2/cmd/gosec@latest",
		},
		{
			Name:           "govulncheck",
			DisplayName:    "Govulncheck",
			Description:    "Vulnerability checker for Go dependencies",
			Type:           "dependency",
			Languages:      []ProjectLanguage{LangGo},
			InstallCommand: "go install golang.org/x/vuln/cmd/govulncheck@latest",
			Requires:       "go.mod",
		},
		// JavaScript/TypeScript
		{
			Name:           "npm-audit",
			DisplayName:    "npm audit",
			Description:    "Dependency vulnerability scanner for npm packages",
			Type:           "dependency",
			Languages:      []ProjectLanguage{LangJavaScript, LangTypeScript},
			InstallCommand: "Built-in to npm >= 6.0",
			Requires:       "package-lock.json",
		},
		{
			Name:           "eslint-security",
			DisplayName:    "ESLint Security",
			Description:    "Security rules for JavaScript/TypeScript code",
			Type:           "sast",
			Languages:      []ProjectLanguage{LangJavaScript, LangTypeScript},
			InstallCommand: "npm install eslint eslint-plugin-security",
		},
		// Python
		{
			Name:           "bandit",
			DisplayName:    "Bandit",
			Description:    "Security linter for Python code",
			Type:           "sast",
			Languages:      []ProjectLanguage{LangPython},
			InstallCommand: "pip install bandit",
		},
		{
			Name:           "pip-audit",
			DisplayName:    "pip-audit",
			Description:    "Vulnerability scanner for Python dependencies",
			Type:           "dependency",
			Languages:      []ProjectLanguage{LangPython},
			InstallCommand: "pip install pip-audit",
		},
	}
}

// GetApplicableScanners returns scanners that are applicable to the detected project.
func GetApplicableScanners(info ProjectInfo) []ScannerInfo {
	all := AvailableScanners()
	applicable := make([]ScannerInfo, 0)

	for _, scanner := range all {
		// Always show cross-language scanners
		if scanner.AlwaysShow {
			applicable = append(applicable, scanner)

			continue
		}

		// Check if scanner applies to any detected language
		for _, scannerLang := range scanner.Languages {
			if info.HasLanguage(scannerLang) {
				applicable = append(applicable, scanner)

				break
			}
		}
	}

	return applicable
}

// GetScannersByType returns applicable scanners filtered by type.
func GetScannersByType(info ProjectInfo, scannerType string) []ScannerInfo {
	applicable := GetApplicableScanners(info)
	filtered := make([]ScannerInfo, 0)

	for _, scanner := range applicable {
		if scanner.Type == scannerType {
			filtered = append(filtered, scanner)
		}
	}

	return filtered
}
