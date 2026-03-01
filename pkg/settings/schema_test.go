package settings

import (
	"reflect"
	"testing"
)

func TestGenerateSchema_NotNil(t *testing.T) {
	schema := GenerateSchema()
	if schema == nil {
		t.Fatal("GenerateSchema() = nil")
	}
	if schema.Version != SchemaVersion {
		t.Errorf("Version = %q, want %q", schema.Version, SchemaVersion)
	}
	if len(schema.Sections) == 0 {
		t.Error("GenerateSchema() returned empty sections")
	}
}

func TestGenerateSchema_HasExpectedSections(t *testing.T) {
	schema := GenerateSchema()

	sectionIDs := map[string]bool{}
	for _, s := range schema.Sections {
		sectionIDs[s.ID] = true
	}

	for _, want := range []string{"agent", "providers", "git", "workers"} {
		if !sectionIDs[want] {
			t.Errorf("missing section %q", want)
		}
	}
}

func TestGenerateSchema_SectionsHaveFields(t *testing.T) {
	schema := GenerateSchema()

	for _, section := range schema.Sections {
		if len(section.Fields) == 0 {
			t.Errorf("section %q has no fields", section.ID)
		}
	}
}

func TestGenerateSchema_SensitiveFieldsHavePassword(t *testing.T) {
	schema := GenerateSchema()

	sensitiveFound := false
	for _, section := range schema.Sections {
		for _, field := range section.Fields {
			if field.Sensitive {
				sensitiveFound = true
				if field.Type != TypePassword {
					t.Errorf("field %q is sensitive but type=%q, want password", field.Path, field.Type)
				}
			}
		}
	}

	if !sensitiveFound {
		t.Error("GenerateSchema() found no sensitive fields (expected github/gitlab/wrike tokens)")
	}
}

func TestGenerateSchema_BooleanFieldsHaveCorrectType(t *testing.T) {
	schema := GenerateSchema()

	boolPaths := map[string]bool{
		"git.create_branch":    true,
		"git.auto_commit":      true,
		"git.sign_commits":     true,
		"git.allow_pr_comment": true,
	}

	for _, section := range schema.Sections {
		for _, field := range section.Fields {
			if boolPaths[field.Path] {
				if field.Type != TypeBoolean {
					t.Errorf("field %q: type=%q, want boolean", field.Path, field.Type)
				}
				delete(boolPaths, field.Path)
			}
		}
	}

	for path := range boolPaths {
		t.Errorf("boolean field %q not found in schema", path)
	}
}

func TestGenerateSchema_NumberFieldsHaveCorrectType(t *testing.T) {
	schema := GenerateSchema()

	for _, section := range schema.Sections {
		for _, field := range section.Fields {
			if field.Path == "workers.max" {
				if field.Type != TypeNumber {
					t.Errorf("workers.max type=%q, want number", field.Type)
				}

				return
			}
		}
	}
	t.Error("workers.max field not found in schema")
}

func TestGenerateSchema_SensitiveFieldsHaveEnvVar(t *testing.T) {
	schema := GenerateSchema()

	for _, section := range schema.Sections {
		for _, field := range section.Fields {
			if field.Sensitive && field.EnvVar == "" {
				t.Errorf("field %q is sensitive but has no EnvVar set", field.Path)
			}
		}
	}
}

func TestGenerateSchemaWithCustomAgents_AddsOptions(t *testing.T) {
	s := &Settings{
		CustomAgents: map[string]CustomAgent{
			"my-bot": {Extends: "claude", Description: "My custom bot"},
		},
	}

	schema := GenerateSchemaWithCustomAgents(s)

	found := false
	for _, section := range schema.Sections {
		if section.ID != "agent" {
			continue
		}
		for _, field := range section.Fields {
			if field.Path != "agent.default" {
				continue
			}
			for _, opt := range field.Options {
				if opt.Value == "my-bot" {
					found = true
				}
			}
		}
	}

	if !found {
		t.Error("GenerateSchemaWithCustomAgents() did not add my-bot to agent.default options")
	}
}

func TestGenerateSchemaWithCustomAgents_NoAgents(t *testing.T) {
	s := &Settings{}
	schema := GenerateSchemaWithCustomAgents(s)

	// Should return normal schema without modification
	if schema == nil {
		t.Fatal("GenerateSchemaWithCustomAgents() = nil")
	}
	if len(schema.Sections) == 0 {
		t.Error("schema has no sections")
	}
}

func TestGenerateSchemaWithCustomAgents_NilSettings(t *testing.T) {
	schema := GenerateSchemaWithCustomAgents(nil)
	if schema == nil {
		t.Fatal("GenerateSchemaWithCustomAgents(nil) = nil")
	}
}

func TestParseSchemaTag_KeyValue(t *testing.T) {
	tags := parseSchemaTag("label=My Label;desc=Description text;type=select")

	if tags["label"] != "My Label" {
		t.Errorf("label = %q, want My Label", tags["label"])
	}
	if tags["desc"] != "Description text" {
		t.Errorf("desc = %q, want Description text", tags["desc"])
	}
	if tags["type"] != "select" {
		t.Errorf("type = %q, want select", tags["type"])
	}
}

func TestParseSchemaTag_Flags(t *testing.T) {
	tags := parseSchemaTag("label=Token;sensitive;required;advanced")

	if tags["sensitive"] != "true" {
		t.Errorf("sensitive = %q, want true", tags["sensitive"])
	}
	if tags["required"] != "true" {
		t.Errorf("required = %q, want true", tags["required"])
	}
	if tags["advanced"] != "true" {
		t.Errorf("advanced = %q, want true", tags["advanced"])
	}
}

func TestParseSchemaTag_Empty(t *testing.T) {
	tags := parseSchemaTag("")
	if len(tags) != 0 {
		t.Errorf("parseSchemaTag(\"\") = %v, want empty map", tags)
	}
}

func TestParseShowWhen_Equals(t *testing.T) {
	cond := parseShowWhen("agent.default:claude")
	if cond == nil {
		t.Fatal("parseShowWhen() = nil")
	}
	if cond.Field != "agent.default" {
		t.Errorf("Field = %q, want agent.default", cond.Field)
	}
	if cond.Equals != "claude" {
		t.Errorf("Equals = %v, want claude", cond.Equals)
	}
}

func TestParseShowWhen_NotEquals(t *testing.T) {
	cond := parseShowWhen("agent.default:!codex")
	if cond == nil {
		t.Fatal("parseShowWhen() = nil")
	}
	if cond.NotEquals != "codex" {
		t.Errorf("NotEquals = %q, want codex", cond.NotEquals)
	}
}

func TestParseShowWhen_BoolTrue(t *testing.T) {
	cond := parseShowWhen("git.auto_commit:true")
	if cond == nil {
		t.Fatal("parseShowWhen() = nil")
	}
	if cond.Equals != true {
		t.Errorf("Equals = %v (%T), want bool true", cond.Equals, cond.Equals)
	}
}

func TestParseShowWhen_Empty(t *testing.T) {
	if parseShowWhen("") != nil {
		t.Error("parseShowWhen(\"\") should return nil")
	}
}

func TestParseShowWhen_NoColon(t *testing.T) {
	if parseShowWhen("nodivider") != nil {
		t.Error("parseShowWhen() with no colon should return nil")
	}
}

func TestParseOptions_Simple(t *testing.T) {
	opts := parseOptions("claude|codex")
	if len(opts) != 2 {
		t.Fatalf("parseOptions() len = %d, want 2", len(opts))
	}
	if opts[0].Value != "claude" {
		t.Errorf("opts[0].Value = %q, want claude", opts[0].Value)
	}
	if opts[1].Value != "codex" {
		t.Errorf("opts[1].Value = %q, want codex", opts[1].Value)
	}
}

func TestParseOptions_ValueLabel(t *testing.T) {
	opts := parseOptions("gh:GitHub|gl:GitLab")
	if len(opts) != 2 {
		t.Fatalf("parseOptions() len = %d, want 2", len(opts))
	}
	if opts[0].Value != "gh" || opts[0].Label != "GitHub" {
		t.Errorf("opts[0] = {%q, %q}, want {gh, GitHub}", opts[0].Value, opts[0].Label)
	}
}

func TestParseOptions_Empty(t *testing.T) {
	if parseOptions("") != nil {
		t.Error("parseOptions(\"\") should return nil")
	}
}

func TestParseDefaultValue_Uint(t *testing.T) {
	result := parseDefaultValue("42", reflect.TypeOf(uint(0)))
	if v, ok := result.(uint64); !ok || v != 42 {
		t.Errorf("parseDefaultValue(uint) = %v (%T), want uint64(42)", result, result)
	}
}

func TestParseDefaultValue_Float(t *testing.T) {
	result := parseDefaultValue("3.14", reflect.TypeOf(float64(0)))
	if v, ok := result.(float64); !ok || v != 3.14 {
		t.Errorf("parseDefaultValue(float) = %v (%T), want float64(3.14)", result, result)
	}
}

func TestParseDefaultValue_InvalidInt(t *testing.T) {
	result := parseDefaultValue("notanumber", reflect.TypeOf(int(0)))
	// Falls through to return 0 (int)
	if result != 0 {
		t.Errorf("parseDefaultValue(invalid int) = %v, want 0", result)
	}
}

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct{ in, want string }{
		{"hello", "Hello"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := capitalizeFirst(tt.in); got != tt.want {
			t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGenerate_PointerType(t *testing.T) {
	// Generate should handle pointer types
	schema := Generate(reflect.TypeOf(&Settings{}))
	if schema == nil {
		t.Fatal("Generate(pointer) = nil")
	}
	if len(schema.Sections) == 0 {
		t.Error("Generate(pointer) returned empty schema")
	}
}

func TestGenerate_NonStruct(t *testing.T) {
	// Non-struct types should return empty schema
	schema := Generate(reflect.TypeOf("string"))
	if schema == nil {
		t.Fatal("Generate(string) = nil")
	}
	if len(schema.Sections) != 0 {
		t.Errorf("Generate(string) sections = %d, want 0", len(schema.Sections))
	}
}
