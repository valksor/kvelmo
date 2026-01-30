package security

import "testing"

func TestNewBaseScanner(t *testing.T) {
	tests := []struct {
		name        string
		scannerName string
		enabled     bool
	}{
		{"enabled scanner", "gosec", true},
		{"disabled scanner", "bandit", false},
		{"empty name", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := NewBaseScanner(tt.scannerName, tt.enabled)

			if base.Name() != tt.scannerName {
				t.Errorf("Name() = %q, want %q", base.Name(), tt.scannerName)
			}
			if base.IsEnabled() != tt.enabled {
				t.Errorf("IsEnabled() = %v, want %v", base.IsEnabled(), tt.enabled)
			}
		})
	}
}

func TestBaseScanner_Name(t *testing.T) {
	base := NewBaseScanner("test-scanner", true)

	if got := base.Name(); got != "test-scanner" {
		t.Errorf("Name() = %q, want %q", got, "test-scanner")
	}
}

func TestBaseScanner_IsEnabled(t *testing.T) {
	enabled := NewBaseScanner("scanner", true)
	disabled := NewBaseScanner("scanner", false)

	if !enabled.IsEnabled() {
		t.Error("IsEnabled() = false for enabled scanner")
	}
	if disabled.IsEnabled() {
		t.Error("IsEnabled() = true for disabled scanner")
	}
}

func TestBaseScanner_SetEnabled(t *testing.T) {
	base := NewBaseScanner("scanner", true)

	if !base.IsEnabled() {
		t.Fatal("Initial state should be enabled")
	}

	base.SetEnabled(false)
	if base.IsEnabled() {
		t.Error("SetEnabled(false) did not disable scanner")
	}

	base.SetEnabled(true)
	if !base.IsEnabled() {
		t.Error("SetEnabled(true) did not enable scanner")
	}
}

func TestBaseScanner_ZeroValue(t *testing.T) {
	var base BaseScanner

	// Zero value should have empty name and be disabled
	if base.Name() != "" {
		t.Errorf("Zero value Name() = %q, want empty", base.Name())
	}
	if base.IsEnabled() {
		t.Error("Zero value IsEnabled() = true, want false")
	}
}

// ExampleEmbedding demonstrates how to embed BaseScanner in a concrete scanner.
type exampleScanner struct {
	BaseScanner

	customField string
}

func newExampleScanner(enabled bool, custom string) *exampleScanner {
	return &exampleScanner{
		BaseScanner: NewBaseScanner("example", enabled),
		customField: custom,
	}
}

func TestBaseScanner_Embedding(t *testing.T) {
	scanner := newExampleScanner(true, "custom-value")

	// BaseScanner methods should be accessible
	if scanner.Name() != "example" {
		t.Errorf("Name() = %q, want %q", scanner.Name(), "example")
	}
	if !scanner.IsEnabled() {
		t.Error("IsEnabled() = false, want true")
	}

	// Custom field should be accessible
	if scanner.customField != "custom-value" {
		t.Errorf("customField = %q, want %q", scanner.customField, "custom-value")
	}

	// SetEnabled should work
	scanner.SetEnabled(false)
	if scanner.IsEnabled() {
		t.Error("SetEnabled(false) did not work on embedded scanner")
	}
}
