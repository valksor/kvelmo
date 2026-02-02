package file

import (
	"reflect"
	"testing"
)

// TestKnownFrontmatterKeysComplete ensures knownFrontmatterKeys stays in sync
// with the Frontmatter struct. If a new field with a yaml tag is added to
// Frontmatter but not to knownFrontmatterKeys, this test fails.
func TestKnownFrontmatterKeysComplete(t *testing.T) {
	rt := reflect.TypeOf(Frontmatter{})

	for i := range rt.NumField() {
		field := rt.Field(i)
		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue // Skip untagged or excluded fields (e.g., Extra)
		}

		// Strip ",omitempty" and other options
		key := yamlTag
		if idx := len(key); idx > 0 {
			for j, c := range key {
				if c == ',' {
					key = key[:j]

					break
				}
			}
		}

		if !knownFrontmatterKeys[key] {
			t.Errorf("Frontmatter field %q (yaml:%q) not in knownFrontmatterKeys — add %q to the set", field.Name, yamlTag, key)
		}
	}

	// Also check reverse: no stale keys in knownFrontmatterKeys
	yamlKeys := make(map[string]bool)
	for i := range reflect.TypeOf(Frontmatter{}).NumField() {
		field := reflect.TypeOf(Frontmatter{}).Field(i)
		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}
		key := yamlTag
		for j, c := range key {
			if c == ',' {
				key = key[:j]

				break
			}
		}
		yamlKeys[key] = true
	}

	for key := range knownFrontmatterKeys {
		if !yamlKeys[key] {
			t.Errorf("knownFrontmatterKeys contains %q but no matching Frontmatter field — remove it", key)
		}
	}
}

func TestExtractExtraFrontmatter(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]any
		want   map[string]any
	}{
		{
			name:   "only known keys returns nil",
			fields: map[string]any{"title": "Test", "priority": "high"},
			want:   nil,
		},
		{
			name:   "extra keys extracted",
			fields: map[string]any{"title": "Test", "custom_field": "value", "deadline": "2026-03-01"},
			want:   map[string]any{"custom_field": "value", "deadline": "2026-03-01"},
		},
		{
			name:   "empty map returns nil",
			fields: map[string]any{},
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractExtraFrontmatter(tt.fields)
			if tt.want == nil {
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}

				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("got %d keys, want %d", len(got), len(tt.want))

				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("got[%q] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestParseMarkdown_ExtraFrontmatter(t *testing.T) {
	content := `---
title: My Task
priority: high
custom_field: hello
reference_url: "https://example.com/task/42"
---
# Body content`

	parsed, err := ParseMarkdown(content, "fallback")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if parsed.Frontmatter == nil {
		t.Fatal("Frontmatter is nil")
	}
	if parsed.Frontmatter.Title != "My Task" {
		t.Errorf("Title = %q, want %q", parsed.Frontmatter.Title, "My Task")
	}
	if parsed.Frontmatter.Extra == nil {
		t.Fatal("Extra is nil")
	}
	if parsed.Frontmatter.Extra["custom_field"] != "hello" {
		t.Errorf("Extra[custom_field] = %v, want %q", parsed.Frontmatter.Extra["custom_field"], "hello")
	}
	if parsed.Frontmatter.Extra["reference_url"] != "https://example.com/task/42" {
		t.Errorf("Extra[reference_url] = %v, want %q", parsed.Frontmatter.Extra["reference_url"], "https://example.com/task/42")
	}
	// Known fields should NOT appear in Extra
	if _, exists := parsed.Frontmatter.Extra["title"]; exists {
		t.Error("known field 'title' should not be in Extra")
	}
}
