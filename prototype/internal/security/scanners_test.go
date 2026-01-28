package security

import (
	"testing"
)

// TestSemgrepScanner tests the Semgrep scanner.
func TestSemgrepScanner(t *testing.T) {
	t.Run("NewSemgrepScanner with nil config", func(t *testing.T) {
		scanner := NewSemgrepScanner(true, nil)
		if scanner == nil {
			t.Fatal("expected non-nil scanner")
		}
		if scanner.Name() != "semgrep" {
			t.Errorf("Name() = %s, want semgrep", scanner.Name())
		}
		if !scanner.IsEnabled() {
			t.Error("expected scanner to be enabled")
		}
		// Default config should be set
		if scanner.config.Config != "auto" {
			t.Errorf("expected default config 'auto', got %s", scanner.config.Config)
		}
	})

	t.Run("NewSemgrepScanner with custom config", func(t *testing.T) {
		config := &SemgrepConfig{
			Config:   "p/security-audit",
			Exclude:  []string{"vendor"},
			Severity: "error",
		}
		scanner := NewSemgrepScanner(false, config)
		if scanner.IsEnabled() {
			t.Error("expected scanner to be disabled")
		}
		if scanner.config.Config != "p/security-audit" {
			t.Errorf("config not preserved: %s", scanner.config.Config)
		}
	})

	t.Run("NewSemgrepScanner with empty config", func(t *testing.T) {
		config := &SemgrepConfig{Config: ""}
		scanner := NewSemgrepScanner(true, config)
		// Should default to "auto" when empty
		if scanner.config.Config != "auto" {
			t.Errorf("expected default config 'auto' for empty, got %s", scanner.config.Config)
		}
	})
}

func TestMapSemgrepSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected Severity
	}{
		{"error", SeverityHigh},
		{"ERROR", SeverityHigh},
		{"warning", SeverityMedium},
		{"WARNING", SeverityMedium},
		{"info", SeverityLow},
		{"INFO", SeverityLow},
		{"unknown", SeverityInfo},
		{"", SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapSemgrepSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("mapSemgrepSeverity(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestNpmAuditScanner tests the npm audit scanner.
func TestNpmAuditScanner(t *testing.T) {
	t.Run("NewNpmAuditScanner with nil config", func(t *testing.T) {
		scanner := NewNpmAuditScanner(true, nil)
		if scanner == nil {
			t.Fatal("expected non-nil scanner")
		}
		if scanner.Name() != "npm-audit" {
			t.Errorf("Name() = %s, want npm-audit", scanner.Name())
		}
		if !scanner.IsEnabled() {
			t.Error("expected scanner to be enabled")
		}
	})

	t.Run("NewNpmAuditScanner with custom config", func(t *testing.T) {
		config := &NpmAuditConfig{
			Level:      "high",
			Production: true,
		}
		scanner := NewNpmAuditScanner(true, config)
		if scanner.config.Level != "high" {
			t.Errorf("Level not preserved: %s", scanner.config.Level)
		}
		if !scanner.config.Production {
			t.Error("Production not preserved")
		}
	})
}

func TestMapNpmSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected Severity
	}{
		{"critical", SeverityCritical},
		{"CRITICAL", SeverityCritical},
		{"high", SeverityHigh},
		{"HIGH", SeverityHigh},
		{"moderate", SeverityMedium},
		{"MODERATE", SeverityMedium},
		{"low", SeverityLow},
		{"LOW", SeverityLow},
		{"info", SeverityInfo},
		{"INFO", SeverityInfo},
		{"unknown", SeverityInfo},
		{"", SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapNpmSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("mapNpmSeverity(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNpmAuditScanner_ShouldIgnore(t *testing.T) {
	config := &NpmAuditConfig{
		IgnoreAdvisories: []string{"lodash", "moment"},
	}
	scanner := NewNpmAuditScanner(true, config)

	tests := []struct {
		pkgName  string
		expected bool
	}{
		{"lodash", true},
		{"moment", true},
		{"express", false},
		{"react", false},
	}

	for _, tt := range tests {
		t.Run(tt.pkgName, func(t *testing.T) {
			result := scanner.shouldIgnore(tt.pkgName)
			if result != tt.expected {
				t.Errorf("shouldIgnore(%q) = %v, want %v", tt.pkgName, result, tt.expected)
			}
		})
	}
}

// TestESLintScanner tests the ESLint scanner.
func TestESLintScanner(t *testing.T) {
	t.Run("NewESLintScanner with nil config", func(t *testing.T) {
		scanner := NewESLintScanner(true, nil)
		if scanner == nil {
			t.Fatal("expected non-nil scanner")
		}
		if scanner.Name() != "eslint-security" {
			t.Errorf("Name() = %s, want eslint-security", scanner.Name())
		}
		// Default extensions should be set
		expectedExts := []string{".js", ".jsx", ".ts", ".tsx"}
		if len(scanner.config.Extensions) != len(expectedExts) {
			t.Errorf("expected %d default extensions, got %d", len(expectedExts), len(scanner.config.Extensions))
		}
	})

	t.Run("NewESLintScanner with custom extensions", func(t *testing.T) {
		config := &ESLintConfig{
			Extensions: []string{".mjs", ".cjs"},
		}
		scanner := NewESLintScanner(true, config)
		if len(scanner.config.Extensions) != 2 {
			t.Errorf("expected 2 extensions, got %d", len(scanner.config.Extensions))
		}
	})
}

func TestMapESLintSeverity(t *testing.T) {
	tests := []struct {
		input    int
		expected Severity
	}{
		{2, SeverityHigh},   // error
		{1, SeverityMedium}, // warning
		{0, SeverityInfo},   // off
		{-1, SeverityInfo},  // invalid
		{99, SeverityInfo},  // invalid
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.input)), func(t *testing.T) {
			result := mapESLintSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("mapESLintSeverity(%d) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsSecurityRule(t *testing.T) {
	tests := []struct {
		ruleID   string
		expected bool
	}{
		{"security/detect-eval-with-expression", true},
		{"security/detect-object-injection", true},
		{"no-eval", true},
		{"no-implied-eval", true},
		{"no-new-func", true},
		{"no-script-url", true},
		{"no-unsafe-innerhtml", true},
		{"no-unsafe-negation", true},
		{"no-unused-vars", false},
		{"prefer-const", false},
		{"semi", false},
	}

	for _, tt := range tests {
		t.Run(tt.ruleID, func(t *testing.T) {
			result := isSecurityRule(tt.ruleID)
			if result != tt.expected {
				t.Errorf("isSecurityRule(%q) = %v, want %v", tt.ruleID, result, tt.expected)
			}
		})
	}
}

// TestBanditScanner tests the Bandit scanner.
func TestBanditScanner(t *testing.T) {
	t.Run("NewBanditScanner with nil config", func(t *testing.T) {
		scanner := NewBanditScanner(true, nil)
		if scanner == nil {
			t.Fatal("expected non-nil scanner")
		}
		if scanner.Name() != "bandit" {
			t.Errorf("Name() = %s, want bandit", scanner.Name())
		}
	})

	t.Run("NewBanditScanner with custom config", func(t *testing.T) {
		config := &BanditConfig{
			Severity:   "high",
			Confidence: "medium",
			Exclude:    []string{"tests"},
			Skip:       []string{"B101"},
		}
		scanner := NewBanditScanner(true, config)
		if scanner.config.Severity != "high" {
			t.Errorf("Severity not preserved: %s", scanner.config.Severity)
		}
		if len(scanner.config.Skip) != 1 || scanner.config.Skip[0] != "B101" {
			t.Error("Skip not preserved")
		}
	})
}

func TestMapBanditSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected Severity
	}{
		{"HIGH", SeverityHigh},
		{"high", SeverityHigh},
		{"MEDIUM", SeverityMedium},
		{"medium", SeverityMedium},
		{"LOW", SeverityLow},
		{"low", SeverityLow},
		{"unknown", SeverityInfo},
		{"", SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapBanditSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("mapBanditSeverity(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestPipAuditScanner tests the pip-audit scanner.
func TestPipAuditScanner(t *testing.T) {
	t.Run("NewPipAuditScanner with nil config", func(t *testing.T) {
		scanner := NewPipAuditScanner(true, nil)
		if scanner == nil {
			t.Fatal("expected non-nil scanner")
		}
		if scanner.Name() != "pip-audit" {
			t.Errorf("Name() = %s, want pip-audit", scanner.Name())
		}
	})

	t.Run("NewPipAuditScanner with custom config", func(t *testing.T) {
		config := &PipAuditConfig{
			RequirementsFile: "requirements-dev.txt",
			Strict:           true,
			IgnoreVulns:      []string{"PYSEC-2023-1234"},
		}
		scanner := NewPipAuditScanner(true, config)
		if scanner.config.RequirementsFile != "requirements-dev.txt" {
			t.Errorf("RequirementsFile not preserved: %s", scanner.config.RequirementsFile)
		}
		if !scanner.config.Strict {
			t.Error("Strict not preserved")
		}
	})
}

func TestMapPipAuditSeverity(t *testing.T) {
	tests := []struct {
		vulnID   string
		expected Severity
	}{
		{"CVE-2024-1234", SeverityHigh},
		{"PYSEC-2023-5678", SeverityHigh},
		{"GHSA-xxxx-xxxx-xxxx", SeverityHigh},
		{"OTHER-1234", SeverityMedium},
		{"", SeverityMedium},
	}

	for _, tt := range tests {
		t.Run(tt.vulnID, func(t *testing.T) {
			result := mapPipAuditSeverity(tt.vulnID)
			if result != tt.expected {
				t.Errorf("mapPipAuditSeverity(%q) = %v, want %v", tt.vulnID, result, tt.expected)
			}
		})
	}
}

func TestExtractCVEFromID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CVE-2024-1234", "CVE-2024-1234"},
		{"CVE-2023-99999", "CVE-2023-99999"},
		{"PYSEC-2023-5678", ""},
		{"GHSA-xxxx-xxxx-xxxx", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractCVEFromID(tt.input)
			if result != tt.expected {
				t.Errorf("extractCVEFromID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGitleaksScanner tests the gitleaks scanner.
func TestGitleaksScanner(t *testing.T) {
	t.Run("NewGitleaksScanner with nil config", func(t *testing.T) {
		scanner := NewGitleaksScanner(true, nil)
		if scanner == nil {
			t.Fatal("expected non-nil scanner")
		}
		if scanner.Name() != "gitleaks" {
			t.Errorf("Name() = %s, want gitleaks", scanner.Name())
		}
	})

	t.Run("NewGitleaksScanner with custom config", func(t *testing.T) {
		config := &GitleaksConfig{
			MaxDepth: 100,
			Verbose:  true,
		}
		scanner := NewGitleaksScanner(true, config)
		if scanner.config.MaxDepth != 100 {
			t.Errorf("MaxDepth not preserved: %d", scanner.config.MaxDepth)
		}
	})
}

func TestMapGitleaksSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected Severity
	}{
		{"critical", SeverityCritical},
		{"CRITICAL", SeverityCritical},
		{"high", SeverityHigh},
		{"HIGH", SeverityHigh},
		{"medium", SeverityMedium},
		{"MEDIUM", SeverityMedium},
		{"low", SeverityLow},
		{"LOW", SeverityLow},
		{"unknown", SeverityInfo},
		{"", SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapGitleaksSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("mapGitleaksSeverity(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"short", "***"},                     // <= 8 chars
		{"12345678", "***"},                  // exactly 8 chars
		{"123456789", "1234...6789"},         // > 8 chars
		{"mySecretAPIKey123", "mySe...y123"}, // shows first 4 and last 4 chars
		{"", "***"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskSecret(tt.input)
			if result != tt.expected {
				t.Errorf("maskSecret(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGosecScanner tests the gosec scanner.
func TestGosecScanner(t *testing.T) {
	t.Run("NewGosecScanner with nil config", func(t *testing.T) {
		scanner := NewGosecScanner(true, nil)
		if scanner == nil {
			t.Fatal("expected non-nil scanner")
		}
		if scanner.Name() != "gosec" {
			t.Errorf("Name() = %s, want gosec", scanner.Name())
		}
	})

	t.Run("NewGosecScanner with custom config", func(t *testing.T) {
		config := &GosecConfig{
			Severity:   "high",
			Confidence: "medium",
			Exclude:    []string{"*_test.go"},
		}
		scanner := NewGosecScanner(true, config)
		if scanner.config.Severity != "high" {
			t.Errorf("Severity not preserved: %s", scanner.config.Severity)
		}
	})
}

// TestGovulncheckScanner tests the govulncheck scanner.
func TestGovulncheckScanner(t *testing.T) {
	t.Run("NewGovulncheckScanner", func(t *testing.T) {
		scanner := NewGovulncheckScanner(true)
		if scanner == nil {
			t.Fatal("expected non-nil scanner")
		}
		if scanner.Name() != "govulncheck" {
			t.Errorf("Name() = %s, want govulncheck", scanner.Name())
		}
		if !scanner.IsEnabled() {
			t.Error("expected scanner to be enabled")
		}
	})

	t.Run("NewGovulncheckScanner disabled", func(t *testing.T) {
		scanner := NewGovulncheckScanner(false)
		if scanner.IsEnabled() {
			t.Error("expected scanner to be disabled")
		}
	})
}
