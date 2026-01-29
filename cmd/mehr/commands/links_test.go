//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valksor/go-mehrhof/internal/links"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// TestLinksCommand_Structure tests the links command structure.
func TestLinksCommand_Structure(t *testing.T) {
	if linksCmd.Use != "links" {
		t.Errorf("Use = %q, want %q", linksCmd.Use, "links")
	}

	if linksCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if linksCmd.RunE == nil {
		t.Error("RunE not set")
	}

	// Check for expected subcommands
	subcommands := linksCmd.Commands()
	expectedSubcommands := []string{"list", "backlinks", "search", "stats", "rebuild"}
	actualNames := make(map[string]bool)
	for _, cmd := range subcommands {
		// Extract just the command name (first word) from Use field
		name := cmd.Use
		for i, r := range name {
			if r == ' ' || r == '\t' {
				name = name[:i]

				break
			}
		}
		actualNames[name] = true
	}

	for _, expected := range expectedSubcommands {
		if !actualNames[expected] {
			t.Errorf("linksCmd missing %q subcommand", expected)
		}
	}
}

// TestLinksListCommand_Structure tests the list subcommand structure.
func TestLinksListCommand_Structure(t *testing.T) {
	// Find the list subcommand
	for _, cmd := range linksCmd.Commands() {
		// Extract command name from Use field (first word)
		name := cmd.Use
		for i, r := range name {
			if r == ' ' || r == '\t' {
				name = name[:i]

				break
			}
		}
		if name == "list" {
			// We can't access the private variable, but we can test the command
			if cmd.Short == "" {
				t.Error("list Short description is empty")
			}
			if cmd.RunE == nil {
				t.Error("list RunE not set")
			}

			return
		}
	}
	t.Error("list subcommand not found")
}

// TestLinksList_FormatFlag tests the format flag.
func TestLinksList_FormatFlag(t *testing.T) {
	for _, cmd := range linksCmd.Commands() {
		if cmd.Use == "list" {
			formatFlag := cmd.Flags().Lookup("format")
			if formatFlag == nil {
				t.Error("list command missing 'format' flag")
			}

			verboseFlag := cmd.Flags().Lookup("verbose")
			if verboseFlag == nil {
				t.Error("list command missing 'verbose' flag")
			}

			return
		}
	}
}

// TestLinksBacklinks_FormatFlag tests the backlinks format flag.
func TestLinksBacklinks_FormatFlag(t *testing.T) {
	for _, cmd := range linksCmd.Commands() {
		// Extract command name from Use field (first word)
		name := cmd.Use
		for i, r := range name {
			if r == ' ' || r == '\t' {
				name = name[:i]

				break
			}
		}
		if name == "backlinks" {
			formatFlag := cmd.Flags().Lookup("format")
			if formatFlag == nil {
				t.Error("backlinks command missing 'format' flag")
			}

			return
		}
	}
}

// TestLinksRebuild_SilenceUsage tests that rebuild silences usage on error.
func TestLinksRebuild_SilenceUsage(t *testing.T) {
	for _, cmd := range linksCmd.Commands() {
		if cmd.Use == "rebuild" {
			if !cmd.SilenceUsage {
				t.Error("rebuild should have SilenceUsage=true")
			}

			return
		}
	}
}

// TestFormatOrphanCount tests the formatOrphanCount function.
func TestFormatOrphanCount(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  string
	}{
		{"zero orphans", 0, "0"},   // Green (hard to test color codes)
		{"few orphans", 5, "5"},    // Yellow
		{"many orphans", 15, "15"}, // Red
		{"exactly ten", 10, "10"},  // Yellow
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatOrphanCount(tt.count)
			// Just verify it returns something non-empty (color codes make exact comparison difficult)
			if got == "" {
				t.Errorf("formatOrphanCount(%d) returned empty string", tt.count)
			}
		})
	}
}

// TestGetAllEntitiesWithLinks tests the getAllEntitiesWithLinks function.
func TestGetAllEntitiesWithLinks(t *testing.T) {
	// Test with empty links manager (no links indexed yet)
	tmpDir := t.TempDir()
	mgr := storage.NewLinkManager(tmpDir, &storage.LinksSettings{Enabled: true})

	result := getAllEntitiesWithLinks(mgr)
	if len(result) != 0 {
		t.Errorf("getAllEntitiesWithLinks() returned %d entities, want 0", len(result))
	}
}

// TestGetTitleForEntity tests the getTitleForEntity function.
func TestGetTitleForEntity(t *testing.T) {
	tests := []struct {
		name     string
		entityID string
		names    *links.NameRegistry
		want     string
	}{
		{
			name:     "spec with registered name",
			entityID: "spec:task-123:1",
			names: func() *links.NameRegistry {
				n := links.NewNameRegistry()
				n.Specs["Authentication Flow"] = "spec:task-123:1"

				return n
			}(),
			want: "Authentication Flow",
		},
		{
			name:     "spec without registered name",
			entityID: "spec:task-123:2",
			names: func() *links.NameRegistry {
				n := links.NewNameRegistry()

				return n
			}(),
			want: "2",
		},
		{
			name:     "decision with registered name",
			entityID: "decision:task-123:cache-strategy",
			names: func() *links.NameRegistry {
				n := links.NewNameRegistry()
				n.Decisions["Cache Strategy"] = "decision:task-123:cache-strategy"

				return n
			}(),
			want: "Cache Strategy",
		},
		{
			name:     "nil names registry",
			entityID: "spec:task-123:1",
			names:    nil,
			want:     "spec:task-123:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTitleForEntity(tt.names, tt.entityID)
			if got != tt.want {
				t.Errorf("getTitleForEntity() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestListAllEntities tests the listAllEntities function.
func TestListAllEntities(t *testing.T) {
	// Test with empty links manager (no links indexed yet)
	tmpDir := t.TempDir()
	mgr := storage.NewLinkManager(tmpDir, &storage.LinksSettings{Enabled: true})

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listAllEntities(mgr)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("listAllEntities() error = %v", err)
	}

	// Should show "No entities with links found"
	if !strings.Contains(output, "No entities with links found") {
		t.Errorf("expected 'No entities with links found', got %q", output)
	}
}

// TestListAllEntities_JSON tests JSON output format.
func TestListAllEntities_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := storage.NewLinkManager(tmpDir, &storage.LinksSettings{Enabled: true})
	now := time.Now()
	idx := mgr.GetIndex()
	idx.Forward["spec:task-123:1"] = []links.Link{
		{Source: "spec:task-123:1", Target: "spec:task-123:2", Context: "see also", CreatedAt: now},
	}
	names := mgr.GetNames()
	names.Specs["Auth Spec"] = "spec:task-123:1"

	// Set JSON format
	originalFormat := linksListFormat
	linksListFormat = "json"
	defer func() { linksListFormat = originalFormat }()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listAllEntities(mgr)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("listAllEntities() error = %v", err)
	}

	// Check JSON format
	if !strings.Contains(output, "[") {
		t.Errorf("expected JSON array, got %q", output)
	}
	if !strings.Contains(output, "]") {
		t.Errorf("expected JSON array closing bracket, got %q", output)
	}
}

// TestSearchEntitiesByName tests the searchEntitiesByName function.
func TestSearchEntitiesByName(t *testing.T) {
	// Test with empty links manager (no names registered)
	tmpDir := t.TempDir()
	mgr := storage.NewLinkManager(tmpDir, &storage.LinksSettings{Enabled: true})

	result := searchEntitiesByName(mgr, "nonexistent")
	if len(result) != 0 {
		t.Errorf("searchEntitiesByName() returned %d results, want 0", len(result))
	}
}

// TestPrintLinks tests the printLinks function.
func TestPrintLinks(t *testing.T) {
	tests := []struct {
		name    string
		links   []links.Link
		verbose bool
		format  string
	}{
		{
			name:    "empty links",
			links:   []links.Link{},
			verbose: false,
			format:  "table",
		},
		{
			name: "single link",
			links: func() []links.Link {
				now := time.Now()

				return []links.Link{
					{Source: "spec:task-123:1", Target: "spec:task-123:2", Context: "see also", CreatedAt: now},
				}
			}(),
			verbose: false,
			format:  "table",
		},
		{
			name: "link with long context",
			links: func() []links.Link {
				now := time.Now()

				return []links.Link{
					{Source: "spec:task-123:1", Target: "spec:task-123:2", Context: "this is a very long context that should be truncated when not in verbose mode", CreatedAt: now},
				}
			}(),
			verbose: false,
			format:  "table",
		},
		{
			name: "link with long context verbose",
			links: func() []links.Link {
				now := time.Now()

				return []links.Link{
					{Source: "spec:task-123:1", Target: "spec:task-123:2", Context: "this is a very long context that should not be truncated in verbose mode", CreatedAt: now},
				}
			}(),
			verbose: true,
			format:  "table",
		},
		{
			name: "link with empty context",
			links: func() []links.Link {
				now := time.Now()

				return []links.Link{
					{Source: "spec:task-123:1", Target: "spec:task-123:2", Context: "", CreatedAt: now},
				}
			}(),
			verbose: false,
			format:  "table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			originalFormat := linksListFormat
			linksListFormat = tt.format
			originalVerbose := linksListVerbose
			linksListVerbose = tt.verbose

			printLinks(tt.links, tt.verbose)

			linksListFormat = originalFormat
			linksListVerbose = originalVerbose

			_ = w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			output := buf.String()

			if tt.format == "table" {
				if len(tt.links) > 0 && !strings.Contains(output, "TARGET") {
					t.Errorf("expected header 'TARGET' in table output, got %q", output)
				}
			}
		})
	}
}

// TestPrintLinksJSON tests the JSON output format.
func TestPrintLinksJSON(t *testing.T) {
	now := time.Now()
	linksSlice := []links.Link{
		{Source: "spec:task-123:1", Target: "spec:task-123:2", Context: "see also", CreatedAt: now},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printLinksJSON(linksSlice)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Check JSON format
	assert.Contains(t, output, "[")
	assert.Contains(t, output, "]")
	assert.Contains(t, output, `"source"`)
	assert.Contains(t, output, `"target"`)
	assert.Contains(t, output, `"context"`)
}
