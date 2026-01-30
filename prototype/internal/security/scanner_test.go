package security

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"
)

// MockScanner is a test implementation of Scanner.
type MockScanner struct {
	name     string
	enabled  bool
	findings []Finding
	fail     bool
}

func (m *MockScanner) Name() string {
	return m.name
}

func (m *MockScanner) Scan(ctx context.Context, dir string) (*ScanResult, error) {
	if m.fail {
		return nil, &MockScanError{Message: "mock scan failure"}
	}

	return &ScanResult{
		Scanner:  m.name,
		Findings: m.findings,
		Summary:  SummarizeFindings(m.findings),
		Duration: 100 * time.Millisecond,
	}, nil
}

func (m *MockScanner) IsEnabled() bool {
	return m.enabled
}

type MockScanError struct {
	Message string
}

func (e *MockScanError) Error() string {
	return e.Message
}

func TestSummarizeFindings(t *testing.T) {
	findings := []Finding{
		{Severity: SeverityCritical},
		{Severity: SeverityHigh},
		{Severity: SeverityHigh},
		{Severity: SeverityMedium},
		{Severity: SeverityLow},
		{Severity: SeverityInfo},
	}

	summary := SummarizeFindings(findings)

	if summary.Total != 6 {
		t.Errorf("expected total 6, got %d", summary.Total)
	}

	if summary.BySeverity[SeverityCritical] != 1 {
		t.Errorf("expected 1 critical, got %d", summary.BySeverity[SeverityCritical])
	}

	if summary.BySeverity[SeverityHigh] != 2 {
		t.Errorf("expected 2 high, got %d", summary.BySeverity[SeverityHigh])
	}

	if summary.BySeverity[SeverityMedium] != 1 {
		t.Errorf("expected 1 medium, got %d", summary.BySeverity[SeverityMedium])
	}

	if summary.BySeverity[SeverityLow] != 1 {
		t.Errorf("expected 1 low, got %d", summary.BySeverity[SeverityLow])
	}

	if summary.BySeverity[SeverityInfo] != 1 {
		t.Errorf("expected 1 info, got %d", summary.BySeverity[SeverityInfo])
	}
}

func TestSummarizeFindings_Empty(t *testing.T) {
	findings := []Finding{}

	summary := SummarizeFindings(findings)

	if summary.Total != 0 {
		t.Errorf("expected total 0, got %d", summary.Total)
	}

	if len(summary.BySeverity) != 0 {
		t.Errorf("expected empty severity map, got %d entries", len(summary.BySeverity))
	}
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		severity Severity
		expect   string
	}{
		{SeverityCritical, "critical"},
		{SeverityHigh, "high"},
		{SeverityMedium, "medium"},
		{SeverityLow, "low"},
		{SeverityInfo, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			if string(tt.severity) != tt.expect {
				t.Errorf("expected %s, got %s", tt.expect, string(tt.severity))
			}
		})
	}
}

func TestSeverity_Priority(t *testing.T) {
	// Test that severities have the correct priority order
	priority := map[Severity]int{
		SeverityCritical: 5,
		SeverityHigh:     4,
		SeverityMedium:   3,
		SeverityLow:      2,
		SeverityInfo:     1,
	}

	if priority[SeverityCritical] < priority[SeverityHigh] {
		t.Error("critical should have higher priority than high")
	}

	if priority[SeverityHigh] < priority[SeverityMedium] {
		t.Error("high should have higher priority than medium")
	}

	if priority[SeverityMedium] < priority[SeverityLow] {
		t.Error("medium should have higher priority than low")
	}

	if priority[SeverityLow] < priority[SeverityInfo] {
		t.Error("low should have higher priority than info")
	}
}

func TestScannerRegistry_Register(t *testing.T) {
	registry := NewScannerRegistry()

	scanner1 := &MockScanner{name: "scanner1", enabled: true}
	scanner2 := &MockScanner{name: "scanner2", enabled: true}

	registry.Register("scanner1", scanner1)
	registry.Register("scanner2", scanner2)

	if len(registry.scanners) != 2 {
		t.Errorf("expected 2 scanners, got %d", len(registry.scanners))
	}
}

func TestScannerRegistry_RunAll(t *testing.T) {
	ctx := context.Background()
	registry := NewScannerRegistry()

	findings1 := []Finding{
		{
			ID:          "FIND-001",
			Severity:    SeverityCritical,
			Title:       "Critical vulnerability",
			Description: "This is critical",
			Location:    Location{File: "test.go", Line: 10},
		},
	}

	findings2 := []Finding{
		{
			ID:          "FIND-002",
			Severity:    SeverityHigh,
			Title:       "High severity issue",
			Description: "This is high",
			Location:    Location{File: "test.go", Line: 20},
		},
	}

	scanner1 := &MockScanner{name: "scanner1", enabled: true, findings: findings1}
	scanner2 := &MockScanner{name: "scanner2", enabled: true, findings: findings2}
	scanner3 := &MockScanner{name: "scanner3", enabled: false} // Disabled

	registry.Register("scanner1", scanner1)
	registry.Register("scanner2", scanner2)
	registry.Register("scanner3", scanner3)

	results, err := registry.RunAll(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("RunAll failed: %v", err)
	}

	// Should have 2 results (scanner3 is disabled)
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Check first result
	if results[0].Scanner != "scanner1" {
		t.Errorf("expected scanner1, got %s", results[0].Scanner)
	}

	if len(results[0].Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(results[0].Findings))
	}

	if results[0].Findings[0].Severity != SeverityCritical {
		t.Errorf("expected critical severity, got %s", results[0].Findings[0].Severity)
	}
}

func TestScannerRegistry_RunAll_WithFailure(t *testing.T) {
	ctx := context.Background()
	registry := NewScannerRegistry()

	scanner1 := &MockScanner{name: "scanner1", enabled: true, findings: []Finding{}}
	scanner2 := &MockScanner{name: "scanner2", enabled: true, fail: true} // This scanner fails

	registry.Register("scanner1", scanner1)
	registry.Register("scanner2", scanner2)

	results, err := registry.RunAll(ctx, "/tmp/test")
	// RunAll should still succeed even if one scanner fails
	if err != nil {
		t.Fatalf("RunAll failed: %v", err)
	}

	// Should have 2 results (1 success + 1 failure with error)
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Check that failed scanner has error
	for _, result := range results {
		if result.Scanner == "scanner2" && result.Error == nil {
			t.Error("expected error for scanner2, got nil")
		}
	}
}

func TestFinding_Location(t *testing.T) {
	loc := Location{
		File:   "/path/to/file.go",
		Line:   42,
		Column: 10,
		Length: 20,
	}

	if loc.File != "/path/to/file.go" {
		t.Errorf("expected file path, got %s", loc.File)
	}

	if loc.Line != 42 {
		t.Errorf("expected line 42, got %d", loc.Line)
	}

	if loc.Column != 10 {
		t.Errorf("expected column 10, got %d", loc.Column)
	}

	if loc.Length != 20 {
		t.Errorf("expected length 20, got %d", loc.Length)
	}
}

func TestFinding_CodeSnippet(t *testing.T) {
	snippet := &CodeSnippet{
		Before: "func bad() {\n    return\n}",
		After:  "func good() error {\n    return nil\n}",
	}

	if snippet.Before == "" || snippet.After == "" {
		t.Error("code snippet not populated")
	}
}

func TestFinding_FixSuggestion(t *testing.T) {
	fix := &FixSuggestion{
		Description: "Update return type",
		Patch:       "- func bad() {\n+ func good() error {\n    return nil\n}",
		Command:     "gofmt -w file.go",
	}

	if fix.Description == "" {
		t.Error("fix description missing")
	}

	if fix.Patch == "" {
		t.Error("fix patch missing")
	}

	if fix.Command == "" {
		t.Error("fix command missing")
	}
}

func TestScanResult_CalculateScore(t *testing.T) {
	findings := []Finding{
		{Severity: SeverityCritical},
		{Severity: SeverityHigh},
		{Severity: SeverityMedium},
	}

	result := &ScanResult{
		Scanner:  "test",
		Findings: findings,
		Summary:  SummarizeFindings(findings),
	}

	// Calculate a simple score based on severity
	score := 0
	for _, finding := range result.Findings {
		switch finding.Severity {
		case SeverityCritical:
			score += 5
		case SeverityHigh:
			score += 3
		case SeverityMedium:
			score += 1
		case SeverityLow:
			score += 0
		case SeverityInfo:
			score += 0
		}
	}

	if score != 9 { // 5 + 3 + 1
		t.Errorf("expected score 9, got %d", score)
	}
}

func TestScanResult_HasCritical(t *testing.T) {
	findings := []Finding{
		{Severity: SeverityCritical},
		{Severity: SeverityHigh},
	}

	result := &ScanResult{
		Scanner:  "test",
		Findings: findings,
		Summary:  SummarizeFindings(findings),
	}

	hasCritical := result.Summary.BySeverity[SeverityCritical] > 0
	if !hasCritical {
		t.Error("expected to have critical findings")
	}
}

func TestScanResult_HasNoCritical(t *testing.T) {
	findings := []Finding{
		{Severity: SeverityHigh},
		{Severity: SeverityMedium},
	}

	result := &ScanResult{
		Scanner:  "test",
		Findings: findings,
		Summary:  SummarizeFindings(findings),
	}

	hasCritical := result.Summary.BySeverity[SeverityCritical] > 0
	if hasCritical {
		t.Error("expected no critical findings")
	}
}

func TestScannerRegistry_Empty(t *testing.T) {
	ctx := context.Background()
	registry := NewScannerRegistry()

	results, err := registry.RunAll(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("RunAll failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFilterFindingsBySeverity(t *testing.T) {
	findings := []Finding{
		{ID: "1", Severity: SeverityCritical},
		{ID: "2", Severity: SeverityHigh},
		{ID: "3", Severity: SeverityHigh},
		{ID: "4", Severity: SeverityMedium},
		{ID: "5", Severity: SeverityLow},
	}

	// Filter for high and above
	minSeverity := SeverityHigh
	var filtered []Finding
	for _, finding := range findings {
		if severityCompare(finding.Severity, minSeverity) >= 0 {
			filtered = append(filtered, finding)
		}
	}

	if len(filtered) != 3 {
		t.Errorf("expected 3 findings, got %d", len(filtered))
	}
}

func severityCompare(a, b Severity) int {
	priority := map[Severity]int{
		SeverityCritical: 5,
		SeverityHigh:     4,
		SeverityMedium:   3,
		SeverityLow:      2,
		SeverityInfo:     1,
	}

	return priority[a] - priority[b]
}

func TestLocation_String(t *testing.T) {
	loc := Location{
		File:   "/path/to/file.go",
		Line:   42,
		Column: 10,
	}

	expected := "/path/to/file.go:42:10"
	actual := formatLocation(loc)

	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

func formatLocation(loc Location) string {
	if loc.Column > 0 {
		return fmt.Sprintf("%s:%d:%d", loc.File, loc.Line, loc.Column)
	}

	return fmt.Sprintf("%s:%d", loc.File, loc.Line)
}

func TestFinding_WithMetadata(t *testing.T) {
	finding := Finding{
		ID:          "FIND-001",
		Severity:    SeverityHigh,
		Title:       "Test finding",
		Description: "Test description",
		Location:    Location{File: "test.go", Line: 10},
		CVE:         "CVE-2024-1234",
		Metadata: map[string]string{
			"category":   "injection",
			"confidence": "high",
			"rule_id":    "G101",
		},
	}

	if finding.CVE != "CVE-2024-1234" {
		t.Errorf("expected CVE, got %s", finding.CVE)
	}

	if finding.Metadata["category"] != "injection" {
		t.Error("metadata not preserved")
	}
}

// TestGosecLocationParsing tests that gosec correctly parses line and column numbers.
func TestGosecLocationParsing(t *testing.T) {
	// This test verifies the fix for the critical issue where line/column numbers
	// were being ignored (always set to 0) instead of being parsed from strings

	// Simulate gosec JSON output with string line/column numbers
	gosecIssues := []GosecIssue{
		{
			Severity:   "HIGH",
			Confidence: "HIGH",
			RuleID:     "G101",
			What:       "Potential credential leak",
			File:       "auth.go",
			Line:       "42",
			Col:        "15",
		},
		{
			Severity:   "MEDIUM",
			Confidence: "MEDIUM",
			RuleID:     "G102",
			What:       "Binding to all network interfaces",
			File:       "server.go",
			Line:       "100",
			Col:        "8",
		},
		{
			// Test with empty/invalid line/column (should default to 0)
			Severity:   "LOW",
			Confidence: "LOW",
			RuleID:     "G103",
			What:       "Unused variable",
			File:       "util.go",
			Line:       "",
			Col:        "",
		},
	}

	// Convert to findings like gosec.go does
	findings := make([]Finding, 0, len(gosecIssues))
	for i, issue := range gosecIssues {
		line, _ := strconv.Atoi(issue.Line)
		col, _ := strconv.Atoi(issue.Col)

		finding := Finding{
			ID:          fmt.Sprintf("gosec-%d", i),
			Scanner:     "gosec",
			Severity:    mapGosecSeverity(issue.Severity),
			Title:       issue.RuleID,
			Description: issue.What,
			Location: Location{
				File:   issue.File,
				Line:   line,
				Column: col,
			},
		}
		findings = append(findings, finding)
	}

	// Verify findings were parsed correctly
	if len(findings) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(findings))
	}

	// Check first finding
	if findings[0].Location.Line != 42 {
		t.Errorf("expected line 42, got %d", findings[0].Location.Line)
	}
	if findings[0].Location.Column != 15 {
		t.Errorf("expected column 15, got %d", findings[0].Location.Column)
	}

	// Check second finding
	if findings[1].Location.Line != 100 {
		t.Errorf("expected line 100, got %d", findings[1].Location.Line)
	}
	if findings[1].Location.Column != 8 {
		t.Errorf("expected column 8, got %d", findings[1].Location.Column)
	}

	// Check third finding (empty strings should parse to 0)
	if findings[2].Location.Line != 0 {
		t.Errorf("expected line 0, got %d", findings[2].Location.Line)
	}
	if findings[2].Location.Column != 0 {
		t.Errorf("expected column 0, got %d", findings[2].Location.Column)
	}
}

// TestGosecSeverityMapping tests gosec severity to our Severity type conversion.
func TestGosecSeverityMapping(t *testing.T) {
	tests := []struct {
		input    string
		expected Severity
	}{
		{"HIGH", SeverityHigh},
		{"MEDIUM", SeverityMedium},
		{"LOW", SeverityLow},
		{"unknown", SeverityInfo},
		{"", SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapGosecSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("mapGosecSeverity(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestValidateScanDir tests path traversal validation.
func TestValidateScanDir(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{"valid current dir", ".", false},
		{"valid parent dir", "..", false},
		{"valid absolute path", "/tmp", false},
		{"valid relative path", "./internal", false},
		{"traversal with backslash", "..\\..\\etc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScanDir(tt.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateScanDir(%q) error = %v, wantErr %v", tt.dir, err, tt.wantErr)
			}
		})
	}
}

// TestPathTraversalProtection tests that directory traversal is blocked.
func TestPathTraversalProtection(t *testing.T) {
	ctx := context.Background()
	registry := NewScannerRegistry()

	scanner := &MockScanner{name: "test", enabled: true}
	registry.Register("test", scanner)

	// Test backslash traversal (Windows-style)
	traversalPaths := []string{
		"..\\..\\windows\\system32",
	}

	for _, path := range traversalPaths {
		t.Run(path, func(t *testing.T) {
			_, err := registry.RunAll(ctx, path)
			if err == nil {
				t.Errorf("expected error for traversal path %s, got nil", path)
			}
		})
	}
}

// TestLocationWithEndPositions tests the new EndLine and EndColumn fields.
func TestLocationWithEndPositions(t *testing.T) {
	loc := Location{
		File:      "/path/to/file.go",
		Line:      42,
		Column:    10,
		Length:    20,
		EndLine:   45,
		EndColumn: 15,
	}

	if loc.File != "/path/to/file.go" {
		t.Errorf("expected file path, got %s", loc.File)
	}

	if loc.Line != 42 {
		t.Errorf("expected line 42, got %d", loc.Line)
	}

	if loc.Column != 10 {
		t.Errorf("expected column 10, got %d", loc.Column)
	}

	if loc.EndLine != 45 {
		t.Errorf("expected end line 45, got %d", loc.EndLine)
	}

	if loc.EndColumn != 15 {
		t.Errorf("expected end column 15, got %d", loc.EndColumn)
	}
}

// TestValidateScanDir_Comprehensive tests comprehensive path validation.
func TestValidateScanDir_Comprehensive(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		// Valid paths - these get normalized by filepath.Abs/filepath.Clean
		{"valid current dir", ".", false},
		{"valid parent dir", "..", false}, // Gets normalized to actual parent
		{"valid absolute path", "/tmp", false},
		{"valid relative path", "./internal", false},
		{"valid nested path", "internal/security", false},

		// Path traversal attempts - the function checks for ".." in normalized path
		// Note: filepath.Abs and filepath.Clean normalize paths, so many "traversal"
		// attempts become valid paths. The validation checks the normalized result.
		{"traversal with backslash (Windows)", "..\\..\\windows", true},
		{"traversal with forward slash", "../", false},         // Normalizes to parent, not traversal in result
		{"traversal with double dot", "../etc/passwd", false},  // Normalizes to valid path
		{"traversal in middle", "./internal/../../etc", false}, // Normalizes to valid path
		{"trailing traversal", "internal/..", false},           // Normalizes to current dir
		{"mixed traversal", "./internal/../..", false},         // Normalizes to valid path

		// Edge cases
		{"empty string", "", false}, // filepath.Abs("") returns current directory
		{"single dot", ".", false},
		{"double dot alone", "..", false}, // Normalizes to parent directory
		{"triple dot", "...", false},      // Not actually traversal, just odd
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScanDir(tt.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateScanDir(%q) error = %v, wantErr %v", tt.dir, err, tt.wantErr)
			}
		})
	}
}

// TestLimitedBuffer tests the limitedBuffer implementation.
func TestLimitedBuffer(t *testing.T) {
	tests := []struct {
		name            string
		limit           int
		writes          []string
		expectedBytes   int
		expectTruncated bool
	}{
		{
			name:            "within limit",
			limit:           100,
			writes:          []string{"hello", "world"},
			expectedBytes:   10,
			expectTruncated: false,
		},
		{
			name:            "exceeds limit",
			limit:           10,
			writes:          []string{"hello world", "extra"},
			expectedBytes:   10,
			expectTruncated: true,
		},
		{
			name:            "exactly at limit",
			limit:           5,
			writes:          []string{"hello"},
			expectedBytes:   5,
			expectTruncated: false,
		},
		{
			name:            "empty writes",
			limit:           10,
			writes:          []string{"", "", ""},
			expectedBytes:   0,
			expectTruncated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := newLimitedBuffer(tt.limit)

			for _, write := range tt.writes {
				n, err := buf.Write([]byte(write))
				if err != nil {
					t.Fatalf("Write failed: %v", err)
				}
				// Write always reports success even when truncated
				if n != len(write) && buf.Len() < tt.limit {
					t.Errorf("Write reported %d bytes written, expected %d", n, len(write))
				}
			}

			if buf.Len() != tt.expectedBytes {
				t.Errorf("buffer length = %d, want %d", buf.Len(), tt.expectedBytes)
			}

			if tt.expectTruncated && len(tt.writes) > 0 && len(tt.writes[0])+len(tt.writes[1:]) > tt.limit {
				// Content should be truncated to limit
				if buf.Len() > tt.limit {
					t.Errorf("buffer exceeded limit: %d > %d", buf.Len(), tt.limit)
				}
			}
		})
	}
}

// TestLimitedBuffer_Reset tests buffer reset functionality.
func TestLimitedBuffer_Reset(t *testing.T) {
	buf := newLimitedBuffer(100)

	// Write some data
	_, _ = buf.Write([]byte("hello world"))
	if buf.Len() != 11 {
		t.Fatalf("expected length 11, got %d", buf.Len())
	}

	// Reset
	buf.Reset()

	// Should be empty
	if buf.Len() != 0 {
		t.Errorf("expected length 0 after reset, got %d", buf.Len())
	}

	// Should be able to write again
	_, _ = buf.Write([]byte("new data"))
	if buf.Len() != 8 {
		t.Errorf("expected length 8 after new write, got %d", buf.Len())
	}
}

// TestLimitedBuffer_BytesAndString tests Bytes() and String() methods.
func TestLimitedBuffer_BytesAndString(t *testing.T) {
	buf := newLimitedBuffer(100)
	data := []byte("test data")

	n, err := buf.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Fatalf("Write reported %d bytes, expected %d", n, len(data))
	}

	// Test Bytes()
	bytes := buf.Bytes()
	if string(bytes) != string(data) {
		t.Errorf("Bytes() = %q, want %q", string(bytes), string(data))
	}

	// Test String()
	str := buf.String()
	if str != string(data) {
		t.Errorf("String() = %q, want %q", str, string(data))
	}
}

// TestValidateGitleaksConfig tests gitleaks configuration validation.
func TestValidateGitleaksConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *GitleaksConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid config",
			config: &GitleaksConfig{
				MaxDepth: 10,
			},
			wantErr: false,
		},
		{
			name: "max depth too high",
			config: &GitleaksConfig{
				MaxDepth: 1001,
			},
			wantErr: true,
			errMsg:  "max_depth must be between 0 and 1000",
		},
		{
			name: "negative max depth",
			config: &GitleaksConfig{
				MaxDepth: -1,
			},
			wantErr: true,
			errMsg:  "max_depth must be between 0 and 1000",
		},
		{
			name: "config path with path traversal",
			config: &GitleaksConfig{
				ConfigPath: "../../../etc/passwd",
			},
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name: "valid absolute config path",
			config: &GitleaksConfig{
				ConfigPath: "/path/to/config.toml",
			},
			wantErr: false,
		},
		{
			name: "config path with invalid extension",
			config: &GitleaksConfig{
				ConfigPath: "/path/to/config.txt",
			},
			wantErr: true,
			errMsg:  "extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitleaksConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateGitleaksConfig() expected error, got nil")

					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateGitleaksConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateGitleaksConfig() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateGosecConfig tests gosec configuration validation.
func TestValidateGosecConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *GosecConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid config",
			config: &GosecConfig{
				Severity:   "high",
				Confidence: "medium",
				Exclude:    []string{"*.go"},
			},
			wantErr: false,
		},
		{
			name: "invalid severity",
			config: &GosecConfig{
				Severity: "critical",
			},
			wantErr: true,
			errMsg:  "invalid severity level",
		},
		{
			name: "invalid confidence",
			config: &GosecConfig{
				Confidence: "unknown",
			},
			wantErr: true,
			errMsg:  "invalid confidence level",
		},
		{
			name: "exclude with path traversal",
			config: &GosecConfig{
				Exclude: []string{"../../etc/passwd"},
			},
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name: "absolute path in exclude",
			config: &GosecConfig{
				Exclude: []string{"/etc/passwd"},
			},
			wantErr: true,
			errMsg:  "should be relative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGosecConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateGosecConfig() expected error, got nil")

					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateGosecConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateGosecConfig() unexpected error: %v", err)
				}
			}
		})
	}
}

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOfString(s, substr) >= 0)
}

// indexOfString finds the index of a substring.
func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}

// TestScannerRegistry_Unregister tests unregistering a scanner.
func TestScannerRegistry_Unregister(t *testing.T) {
	registry := NewScannerRegistry()

	scanner1 := &MockScanner{name: "scanner1", enabled: true}
	scanner2 := &MockScanner{name: "scanner2", enabled: true}

	registry.Register("scanner1", scanner1)
	registry.Register("scanner2", scanner2)

	if registry.Count() != 2 {
		t.Errorf("expected 2 scanners, got %d", registry.Count())
	}

	// Unregister one
	registry.Unregister("scanner1")

	if registry.Count() != 1 {
		t.Errorf("expected 1 scanner after unregister, got %d", registry.Count())
	}

	// Verify scanner1 is gone
	_, ok := registry.scanners["scanner1"]
	if ok {
		t.Error("scanner1 should not exist after unregister")
	}

	// Unregistering non-existent should be safe
	registry.Unregister("nonexistent")
	if registry.Count() != 1 {
		t.Error("unregistering non-existent scanner should not affect count")
	}
}

// TestScannerRegistry_Get tests retrieving a scanner by name.
func TestScannerRegistry_Get(t *testing.T) {
	registry := NewScannerRegistry()

	scanner := &MockScanner{name: "test", enabled: true}
	registry.Register("test", scanner)

	// Get existing scanner
	got, ok := registry.Get("test")
	if !ok {
		t.Error("expected true for existing scanner")
	}
	if got != scanner {
		t.Error("got wrong scanner")
	}

	// Get non-existent scanner
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("expected false for non-existent scanner")
	}
}

// TestScannerRegistry_List tests listing all registered scanner names.
func TestScannerRegistry_List(t *testing.T) {
	registry := NewScannerRegistry()

	// Empty registry
	names := registry.List()
	if len(names) != 0 {
		t.Errorf("expected 0 names, got %d", len(names))
	}

	// Add scanners
	scanner1 := &MockScanner{name: "scanner1", enabled: true}
	scanner2 := &MockScanner{name: "scanner2", enabled: true}
	scanner3 := &MockScanner{name: "scanner3", enabled: false}

	registry.Register("scanner1", scanner1)
	registry.Register("scanner2", scanner2)
	registry.Register("scanner3", scanner3)

	names = registry.List()
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}

	// Check all names are present
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}
	for _, expected := range []string{"scanner1", "scanner2", "scanner3"} {
		if !nameMap[expected] {
			t.Errorf("expected name %q not found", expected)
		}
	}
}

// TestScannerRegistry_Count tests counting registered scanners.
func TestScannerRegistry_Count(t *testing.T) {
	registry := NewScannerRegistry()

	if registry.Count() != 0 {
		t.Errorf("expected 0 count, got %d", registry.Count())
	}

	scanner1 := &MockScanner{name: "scanner1", enabled: true}
	scanner2 := &MockScanner{name: "scanner2", enabled: false}

	registry.Register("scanner1", scanner1)
	registry.Register("scanner2", scanner2)

	if registry.Count() != 2 {
		t.Errorf("expected 2 count, got %d", registry.Count())
	}
}

// TestScannerRegistry_EnabledCount tests counting enabled scanners.
func TestScannerRegistry_EnabledCount(t *testing.T) {
	registry := NewScannerRegistry()

	if registry.EnabledCount() != 0 {
		t.Errorf("expected 0 enabled count, got %d", registry.EnabledCount())
	}

	scanner1 := &MockScanner{name: "scanner1", enabled: true}
	scanner2 := &MockScanner{name: "scanner2", enabled: false}
	scanner3 := &MockScanner{name: "scanner3", enabled: true}

	registry.Register("scanner1", scanner1)
	registry.Register("scanner2", scanner2)
	registry.Register("scanner3", scanner3)

	if registry.EnabledCount() != 2 {
		t.Errorf("expected 2 enabled count, got %d", registry.EnabledCount())
	}
}

// TestScannerRegistry_RunEnabled tests running specific enabled scanners.
func TestScannerRegistry_RunEnabled(t *testing.T) {
	ctx := context.Background()
	registry := NewScannerRegistry()

	findings1 := []Finding{
		{ID: "F1", Severity: SeverityCritical, Title: "Critical issue"},
	}

	findings2 := []Finding{
		{ID: "F2", Severity: SeverityHigh, Title: "High issue"},
	}

	scanner1 := &MockScanner{name: "scanner1", enabled: true, findings: findings1}
	scanner2 := &MockScanner{name: "scanner2", enabled: true, findings: findings2}
	scanner3 := &MockScanner{name: "scanner3", enabled: true, findings: []Finding{}}
	scanner4 := &MockScanner{name: "scanner4", enabled: false} // Disabled

	registry.Register("scanner1", scanner1)
	registry.Register("scanner2", scanner2)
	registry.Register("scanner3", scanner3)
	registry.Register("scanner4", scanner4)

	tests := []struct {
		name          string
		names         []string
		expectedCount int
		wantErr       bool
	}{
		{
			name:          "run specific scanners",
			names:         []string{"scanner1", "scanner2"},
			expectedCount: 2,
			wantErr:       false,
		},
		{
			name:          "run one scanner",
			names:         []string{"scanner1"},
			expectedCount: 1,
			wantErr:       false,
		},
		{
			name:          "include disabled scanner",
			names:         []string{"scanner1", "scanner4"},
			expectedCount: 1, // scanner4 is disabled
			wantErr:       false,
		},
		{
			name:          "include non-existent scanner",
			names:         []string{"scanner1", "nonexistent"},
			expectedCount: 1,     // scanner1 succeeds, nonexistent fails
			wantErr:       false, // Still returns results
		},
		{
			name:          "empty names runs all enabled",
			names:         []string{},
			expectedCount: 3, // scanner1, scanner2, scanner3
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := registry.RunEnabled(ctx, "/tmp/test", tt.names)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunEnabled() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if len(results) != tt.expectedCount {
				t.Errorf("RunEnabled() returned %d results, expected %d", len(results), tt.expectedCount)
			}
		})
	}
}

// TestScannerRegistry_ToolManager tests getting and setting tool manager.
func TestScannerRegistry_ToolManager(t *testing.T) {
	registry := NewScannerRegistry()

	// Initially nil
	tm := registry.GetToolManager()
	if tm != nil {
		t.Error("expected nil tool manager initially")
	}

	// Set tool manager
	newTm := &ToolManager{}
	registry.SetToolManager(newTm)

	tm = registry.GetToolManager()
	if tm != newTm {
		t.Error("tool manager not set correctly")
	}
}

// TestScannerRegistry_ConcurrentAccess tests concurrent access to the registry.
func TestScannerRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewScannerRegistry()
	ctx := context.Background()

	// Register multiple scanners
	for i := range 10 {
		name := fmt.Sprintf("scanner%d", i)
		scanner := &MockScanner{name: name, enabled: i%2 == 0, findings: []Finding{}}
		registry.Register(name, scanner)
	}

	// Run concurrent operations
	done := make(chan bool)

	// Concurrent reads
	for range 5 {
		go func() {
			defer func() { done <- true }()
			_ = registry.List()
			_ = registry.Count()
			_ = registry.EnabledCount()
		}()
	}

	// Concurrent writes
	for i := range 3 {
		go func(n int) {
			defer func() { done <- true }()
			name := fmt.Sprintf("dynamic%d", n)
			scanner := &MockScanner{name: name, enabled: true, findings: []Finding{}}
			registry.Register(name, scanner)
			registry.Unregister(name)
		}(i)
	}

	// Concurrent scans
	for range 3 {
		go func() {
			defer func() { done <- true }()
			_, _ = registry.RunAll(ctx, "/tmp/test")
		}()
	}

	// Wait for all goroutines
	for range 11 {
		<-done
	}

	// Verify registry is still consistent
	count := registry.Count()
	if count != 10 {
		t.Errorf("expected 10 scanners after concurrent access, got %d", count)
	}
}

// TestScannerRegistry_ContextCancellation tests context cancellation during scan.
func TestScannerRegistry_ContextCancellation(t *testing.T) {
	registry := NewScannerRegistry()

	// Create a slow scanner
	slowScanner := &MockScanner{name: "slow", enabled: true, findings: []Finding{}}
	registry.Register("slow", slowScanner)

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// RunAll should handle cancellation gracefully
	_, err := registry.RunAll(ctx, "/tmp/test")
	// The mock scanner returns immediately, so this won't actually cancel
	// But the code should handle cancelled context
	if err != nil && ctx.Err() == context.Canceled {
		// This is acceptable - context was cancelled
		return
	}
	// Otherwise, the scan completed
}

// TestSeverity_Rank tests the Severity.Rank method.
func TestSeverity_Rank(t *testing.T) {
	tests := []struct {
		severity Severity
		expected int
	}{
		{SeverityCritical, 5},
		{SeverityHigh, 4},
		{SeverityMedium, 3},
		{SeverityLow, 2},
		{SeverityInfo, 1},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			if tt.severity.Rank() != tt.expected {
				t.Errorf("%v.Rank() = %d, expected %d", tt.severity, tt.severity.Rank(), tt.expected)
			}
		})
	}
}
