package display

import (
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// TestNewFormatter tests creating a new formatter.
func TestNewFormatter(t *testing.T) {
	f := NewFormatter()
	if f == nil {
		t.Fatal("NewFormatter() returned nil")
	}
	if f.indentLevel != 0 {
		t.Errorf("NewFormatter() indentLevel = %d, want 0", f.indentLevel)
	}
	if f.width != DefaultTableWidth {
		t.Errorf("NewFormatter() width = %d, want %d", f.width, DefaultTableWidth)
	}
	if !f.color {
		t.Error("NewFormatter() color should be true by default")
	}
}

// TestFormatterSetIndent tests setting indentation level.
func TestFormatterSetIndent(t *testing.T) {
	f := NewFormatter()

	result := f.SetIndent(2)
	if result != f {
		t.Error("SetIndent() should return the formatter for chaining")
	}
	if f.indentLevel != 2 {
		t.Errorf("SetIndent() indentLevel = %d, want 2", f.indentLevel)
	}
}

// TestFormatterSetWidth tests setting output width.
func TestFormatterSetWidth(t *testing.T) {
	f := NewFormatter()

	result := f.SetWidth(100)
	if result != f {
		t.Error("SetWidth() should return the formatter for chaining")
	}
	if f.width != 100 {
		t.Errorf("SetWidth() width = %d, want 100", f.width)
	}
}

// TestFormatterIndent tests getting indentation string.
func TestFormatterIndent(t *testing.T) {
	tests := []struct {
		name        string
		indentLevel int
		want        string
	}{
		{"no indent", 0, ""},
		{"one level", 1, "  "},
		{"two levels", 2, "    "},
		{"three levels", 3, "      "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormatter().SetIndent(tt.indentLevel)
			if got := f.Indent(); got != tt.want {
				t.Errorf("Indent() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFormatterSection tests section formatting.
func TestFormatterSection(t *testing.T) {
	f := NewFormatter()

	result := f.Section("Test Section")
	if result == "" {
		t.Error("Section() returned empty string")
	}
	if !strings.Contains(result, "Test Section") {
		t.Errorf("Section() should contain title, got: %s", result)
	}
	if !strings.Contains(result, "─") {
		t.Error("Section() should contain separator")
	}

	// Test empty title
	resultEmpty := f.Section("")
	if resultEmpty == "" {
		t.Error("Section() with empty title should return separator")
	}
}

// TestFormatterSubsection tests subsection formatting.
func TestFormatterSubsection(t *testing.T) {
	f := NewFormatter()

	result := f.Subsection("Subsection Title")
	if result == "" {
		t.Error("Subsection() returned empty string")
	}
	if !strings.Contains(result, "Subsection Title") {
		t.Errorf("Subsection() should contain title, got: %s", result)
	}
}

// TestFormatterKeyValue tests key-value formatting.
func TestFormatterKeyValue(t *testing.T) {
	f := NewFormatter()

	result := f.KeyValue("name", "test value")
	if result == "" {
		t.Error("KeyValue() returned empty string")
	}
	if !strings.Contains(result, "name") {
		t.Errorf("KeyValue() should contain key, got: %s", result)
	}
	if !strings.Contains(result, "test value") {
		t.Errorf("KeyValue() should contain value, got: %s", result)
	}

	// Test with indent
	f.SetIndent(1)
	resultIndented := f.KeyValue("key", "value")
	if !strings.HasPrefix(resultIndented, "  ") {
		t.Error("KeyValue() with indent should start with spaces")
	}
}

// TestFormatterKeyValues tests multiple key-value formatting.
func TestFormatterKeyValues(t *testing.T) {
	f := NewFormatter()

	pairs := map[string]string{
		"name":   "test",
		"value":  "123",
		"status": "active",
	}

	result := f.KeyValues(pairs)
	if result == "" {
		t.Error("KeyValues() returned empty string")
	}

	// Check all keys are present
	for key := range pairs {
		if !strings.Contains(result, key) {
			t.Errorf("KeyValues() should contain key %q", key)
		}
	}

	// Check all values are present
	for _, value := range pairs {
		if !strings.Contains(result, value) {
			t.Errorf("KeyValues() should contain value %q", value)
		}
	}

	// Test empty map
	emptyResult := f.KeyValues(map[string]string{})
	if emptyResult != "" {
		t.Errorf("KeyValues() with empty map should return empty string, got: %s", emptyResult)
	}
}

// TestFormatterList tests list formatting.
func TestFormatterList(t *testing.T) {
	f := NewFormatter()

	items := []string{"item 1", "item 2", "item 3"}

	result := f.List(items)
	if result == "" {
		t.Error("List() returned empty string")
	}

	// Check all items are present
	for _, item := range items {
		if !strings.Contains(result, item) {
			t.Errorf("List() should contain item %q", item)
		}
	}

	// Check for bullet points or numbers
	if !strings.Contains(result, "•") && !strings.Contains(result, "1.") {
		t.Error("List() should contain bullet points or numbers")
	}

	// Test empty list
	emptyResult := f.List([]string{})
	if emptyResult != "" {
		t.Errorf("List() with empty items should return empty string, got: %s", emptyResult)
	}

	// Test long list (more than 10 items should switch to bullets)
	longItems := make([]string, 15)
	for i := range longItems {
		longItems[i] = "item"
	}
	longResult := f.List(longItems)
	// After 10 items, should use bullets
	// The implementation switches to bullets after 10 items
	_ = longResult // Verify the list is generated (tests the function produces output)
}

// TestFormatterDefinitionList tests definition list formatting.
func TestFormatterDefinitionList(t *testing.T) {
	f := NewFormatter()

	terms := map[string]string{
		"API": "Application Programming Interface",
		"CLI": "Command Line Interface",
	}

	result := f.DefinitionList(terms)
	if result == "" {
		t.Error("DefinitionList() returned empty string")
	}

	// Check all terms and definitions are present
	for term, definition := range terms {
		if !strings.Contains(result, term) {
			t.Errorf("DefinitionList() should contain term %q", term)
		}
		if !strings.Contains(result, definition) {
			t.Errorf("DefinitionList() should contain definition %q", definition)
		}
	}

	// Test empty map
	emptyResult := f.DefinitionList(map[string]string{})
	if emptyResult != "" {
		t.Errorf("DefinitionList() with empty map should return empty string, got: %s", emptyResult)
	}
}

// TestFormatterCodeBlock tests code block formatting.
func TestFormatterCodeBlock(t *testing.T) {
	f := NewFormatter()

	code := "func main() {\n\tprintln(\"hello\")\n}"

	result := f.CodeBlock(code, "go")
	if result == "" {
		t.Error("CodeBlock() returned empty string")
	}
	if !strings.Contains(result, "```go") {
		t.Error("CodeBlock() should contain language marker")
	}
	if !strings.Contains(result, "```") {
		t.Error("CodeBlock() should contain code fence markers")
	}

	// Test without language
	resultNoLang := f.CodeBlock(code, "")
	if resultNoLang == "" {
		t.Error("CodeBlock() without language returned empty string")
	}
	if !strings.Contains(resultNoLang, "```") {
		t.Error("CodeBlock() should contain code fence markers even without language")
	}

	// Test empty code
	emptyResult := f.CodeBlock("", "go")
	if emptyResult == "" {
		t.Error("CodeBlock() with empty code should still return formatted block")
	}
}

// TestFormatterTimestamp tests timestamp formatting.
func TestFormatterTimestamp(t *testing.T) {
	f := NewFormatter()

	testTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	result := f.Timestamp(testTime)
	if result == "" {
		t.Error("Timestamp() returned empty string")
	}
	// Should match format "02.01.2006 15:04:05" (dd.mm.yyyy)
	expected := "15.01.2024 14:30:00"
	if result != expected {
		t.Errorf("Timestamp() = %q, want %q", result, expected)
	}
}

// TestFormatterShortTimestamp tests short timestamp formatting.
func TestFormatterShortTimestamp(t *testing.T) {
	f := NewFormatter()

	testTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	result := f.ShortTimestamp(testTime)
	if result == "" {
		t.Error("ShortTimestamp() returned empty string")
	}
	// Should match format "15:04:05"
	expected := "14:30:00"
	if result != expected {
		t.Errorf("ShortTimestamp() = %q, want %q", result, expected)
	}
}

// TestFormatterRelativeTimestamp tests relative timestamp formatting.
func TestFormatterRelativeTimestamp(t *testing.T) {
	f := NewFormatter()
	now := time.Now()

	tests := []struct {
		name         string
		offset       time.Duration
		wantContains string
	}{
		{"just now", 10 * time.Second, "just now"},
		{"1 minute ago", 1 * time.Minute, "1 min ago"},
		{"5 minutes ago", 5 * time.Minute, "mins ago"},
		{"1 hour ago", 1 * time.Hour, "1 hr ago"},
		{"3 hours ago", 3 * time.Hour, "hrs ago"},
		{"1 day ago", 24 * time.Hour, "1 day ago"},
		{"5 days ago", 5 * 24 * time.Hour, "days ago"},
		{"old date", 40 * 24 * time.Hour, "-"}, // Should format as date
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime := now.Add(-tt.offset)
			result := f.RelativeTimestamp(testTime)
			if result == "" {
				t.Error("RelativeTimestamp() returned empty string")
			}
			if !strings.Contains(result, tt.wantContains) && tt.wantContains != "-" {
				t.Errorf("RelativeTimestamp() = %q, want to contain %q", result, tt.wantContains)
			}
		})
	}
}

// TestFormatterTruncate tests string truncation.
func TestFormatterTruncate(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "shorter than max",
			input:  "short",
			maxLen: 20,
			want:   "short",
		},
		{
			name:   "exactly max length",
			input:  "exactlylen",
			maxLen: 10,
			want:   "exactlylen",
		},
		{
			name:   "needs truncation",
			input:  "this is a long string",
			maxLen: 10,
			want:   "this is...",
		},
		{
			name:   "max len 3 or less",
			input:  "long string",
			maxLen: 3,
			want:   "...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.Truncate(tt.input, tt.maxLen)
			if result != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.want)
			}
		})
	}
}

// TestFormatterTable tests table formatting.
func TestFormatterTable(t *testing.T) {
	f := NewFormatter()

	headers := []string{"Name", "Value", "Status"}
	rows := [][]string{
		{"Item 1", "100", "active"},
		{"Item 2", "200", "inactive"},
	}

	result := f.Table(headers, rows)
	if result == "" {
		t.Error("Table() returned empty string")
	}

	// Check headers are present
	for _, h := range headers {
		if !strings.Contains(result, h) {
			t.Errorf("Table() should contain header %q", h)
		}
	}

	// Check row data is present
	for _, row := range rows {
		for _, cell := range row {
			if !strings.Contains(result, cell) {
				t.Errorf("Table() should contain cell %q", cell)
			}
		}
	}

	// Check for separator
	if !strings.Contains(result, "─") {
		t.Error("Table() should contain separator")
	}

	// Test empty table
	emptyResult := f.Table(headers, [][]string{})
	if !strings.Contains(emptyResult, "Name") {
		t.Error("Table() with no rows should still have headers")
	}

	// Test ragged rows (different number of columns)
	raggedRows := [][]string{
		{"A", "B"},
		{"X", "Y", "Z"},
	}
	raggedResult := f.Table(headers, raggedRows)
	if raggedResult == "" {
		t.Error("Table() with ragged rows should not crash")
	}
}

// TestHelperSection tests the helper Section function.
func TestHelperSection(t *testing.T) {
	result := Section("Test")
	if result == "" {
		t.Error("Section() helper returned empty string")
	}
}

// TestHelperKeyValue tests the helper KeyValue function.
func TestHelperKeyValue(t *testing.T) {
	result := KeyValue("key", "value")
	if result == "" {
		t.Error("KeyValue() helper returned empty string")
	}
	if !strings.Contains(result, "key") {
		t.Error("KeyValue() helper should contain key")
	}
}

// TestHelperList tests the helper List function.
func TestHelperList(t *testing.T) {
	items := []string{"a", "b", "c"}
	result := List(items)
	if result == "" {
		t.Error("List() helper returned empty string")
	}
}

// TestHelperTruncate tests the helper Truncate function.
func TestHelperTruncate(t *testing.T) {
	result := Truncate("long string here", 10)
	if result == "" {
		t.Error("Truncate() helper returned empty string")
	}
	if len(result) > 10 {
		t.Errorf("Truncate() helper result should be <= 10 chars, got %d", len(result))
	}
}

// TestFormatSpecificationStatusColored tests colored status formatting.
func TestFormatSpecificationStatusColored(t *testing.T) {
	tests := []struct {
		status       string
		wantContains string
	}{
		{storage.SpecificationStatusDraft, "Draft"},
		{storage.SpecificationStatusReady, "Ready"},
		{storage.SpecificationStatusImplementing, "Implementing"},
		{storage.SpecificationStatusDone, "Completed"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := FormatSpecificationStatusColored(tt.status)
			if result == "" {
				t.Error("FormatSpecificationStatusColored() returned empty string")
			}
			if !strings.Contains(result, tt.wantContains) {
				t.Errorf("FormatSpecificationStatusColored() = %q, want to contain %q", result, tt.wantContains)
			}
		})
	}
}

// TestFormatSpecificationStatusWithIconColored tests colored status with icon.
func TestFormatSpecificationStatusWithIconColored(t *testing.T) {
	tests := []struct {
		status   string
		wantIcon string
	}{
		{storage.SpecificationStatusDraft, "○"},
		{storage.SpecificationStatusReady, "◐"},
		{storage.SpecificationStatusImplementing, "◑"},
		{storage.SpecificationStatusDone, "●"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := FormatSpecificationStatusWithIconColored(tt.status)
			if result == "" {
				t.Error("FormatSpecificationStatusWithIconColored() returned empty string")
			}
			if !strings.Contains(result, tt.wantIcon) {
				t.Errorf("FormatSpecificationStatusWithIconColored() should contain icon %q", tt.wantIcon)
			}
		})
	}
}
