package schema

import (
	"reflect"
	"testing"
)

func TestParseSchemaTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected TagValues
	}{
		{
			name: "simple key=value pairs",
			tag:  "label=Commit Prefix;desc=Help text",
			expected: TagValues{
				"label": "Commit Prefix",
				"desc":  "Help text",
			},
		},
		{
			name: "value with commas",
			tag:  "label=Pattern;desc=Use {key}, {type}, {slug}",
			expected: TagValues{
				"label": "Pattern",
				"desc":  "Use {key}, {type}, {slug}",
			},
		},
		{
			name: "flags without values",
			tag:  "label=Token;sensitive;advanced",
			expected: TagValues{
				"label":     "Token",
				"sensitive": "true",
				"advanced":  "true",
			},
		},
		{
			name: "mixed flags and values",
			tag:  "label=Password;desc=API key;sensitive;maxlen=100",
			expected: TagValues{
				"label":     "Password",
				"desc":      "API key",
				"sensitive": "true",
				"maxlen":    "100",
			},
		},
		{
			name: "value with equals sign",
			tag:  "label=Pattern;default=[{key}]",
			expected: TagValues{
				"label":   "Pattern",
				"default": "[{key}]",
			},
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: TagValues{},
		},
		{
			name: "whitespace handling",
			tag:  "  label = Spaced  ;  desc = More spaces  ",
			expected: TagValues{
				"label": "Spaced",
				"desc":  "More spaces",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSchemaTag(tt.tag)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d keys, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("key %q: expected %q, got %q", k, v, result[k])
				}
			}
		})
	}
}

func TestTagValuesGetBool(t *testing.T) {
	tags := TagValues{
		"enabled":  "true",
		"disabled": "false",
		"empty":    "",
	}

	if !tags.GetBool("enabled") {
		t.Error("expected GetBool('enabled') to be true")
	}
	if tags.GetBool("disabled") {
		t.Error("expected GetBool('disabled') to be false")
	}
	if tags.GetBool("empty") {
		t.Error("expected GetBool('empty') to be false")
	}
	if tags.GetBool("missing") {
		t.Error("expected GetBool('missing') to be false")
	}
}

func TestTagValuesGetInt(t *testing.T) {
	tags := TagValues{
		"valid":   "42",
		"invalid": "abc",
		"empty":   "",
	}

	if v := tags.GetInt("valid"); v == nil || *v != 42 {
		t.Errorf("expected GetInt('valid') to be 42, got %v", v)
	}
	if v := tags.GetInt("invalid"); v != nil {
		t.Errorf("expected GetInt('invalid') to be nil, got %v", *v)
	}
	if v := tags.GetInt("empty"); v != nil {
		t.Errorf("expected GetInt('empty') to be nil, got %v", *v)
	}
	if v := tags.GetInt("missing"); v != nil {
		t.Errorf("expected GetInt('missing') to be nil, got %v", *v)
	}
}

func TestParseShowWhen(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		expectedField  string
		expectedEquals any
		expectedNot    any
	}{
		{
			name:           "boolean true",
			value:          "git.auto_commit:true",
			expectedField:  "git.auto_commit",
			expectedEquals: true,
		},
		{
			name:           "boolean false",
			value:          "feature.enabled:false",
			expectedField:  "feature.enabled",
			expectedEquals: false,
		},
		{
			name:           "string value",
			value:          "provider:github",
			expectedField:  "provider",
			expectedEquals: "github",
		},
		{
			name:          "negation",
			value:         "mode:!disabled",
			expectedField: "mode",
			expectedNot:   "disabled",
		},
		{
			name:  "empty value",
			value: "",
		},
		{
			name:  "no colon",
			value: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseShowWhen(tt.value)

			if tt.expectedField == "" {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}

				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Field != tt.expectedField {
				t.Errorf("field: expected %q, got %q", tt.expectedField, result.Field)
			}

			if tt.expectedEquals != nil && result.Equals != tt.expectedEquals {
				t.Errorf("equals: expected %v, got %v", tt.expectedEquals, result.Equals)
			}

			if tt.expectedNot != nil && result.NotEquals != tt.expectedNot {
				t.Errorf("notEquals: expected %v, got %v", tt.expectedNot, result.NotEquals)
			}
		})
	}
}

func TestParseOptions(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []SelectOption
	}{
		{
			name:  "simple values",
			value: "small|medium|large",
			expected: []SelectOption{
				{Value: "small", Label: "small"},
				{Value: "medium", Label: "medium"},
				{Value: "large", Label: "large"},
			},
		},
		{
			name:  "value:label pairs",
			value: "s:Small|m:Medium|l:Large",
			expected: []SelectOption{
				{Value: "s", Label: "Small"},
				{Value: "m", Label: "Medium"},
				{Value: "l", Label: "Large"},
			},
		},
		{
			name:     "empty value",
			value:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseOptions(tt.value)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d options, got %d", len(tt.expected), len(result))

				return
			}

			for i, opt := range tt.expected {
				if result[i].Value != opt.Value {
					t.Errorf("option %d value: expected %q, got %q", i, opt.Value, result[i].Value)
				}
				if result[i].Label != opt.Label {
					t.Errorf("option %d label: expected %q, got %q", i, opt.Label, result[i].Label)
				}
			}
		})
	}
}

// Test struct for generator tests.
type TestGitSettings struct {
	CommitPrefix  string `yaml:"commit_prefix" json:"commit_prefix" schema:"label=Commit Prefix;desc=Pattern for commit messages;default=[{key}];maxlen=100"`
	BranchPattern string `yaml:"branch_pattern" json:"branch_pattern" schema:"label=Branch Pattern;desc=Pattern for branch names"`
	AutoCommit    bool   `yaml:"auto_commit" json:"auto_commit" schema:"label=Auto Commit;desc=Commit after implementation;default=true"`
	SignCommits   bool   `yaml:"sign_commits" json:"sign_commits" schema:"label=Sign Commits;showWhen=git.auto_commit:true"`
	Secret        string `yaml:"secret" json:"secret" schema:"label=API Secret;sensitive"`
	Advanced      string `yaml:"advanced" json:"advanced" schema:"label=Advanced Option;advanced"`
}

type TestConfig struct {
	Git     TestGitSettings  `yaml:"git" json:"git"`
	Pointer *TestGitSettings `yaml:"pointer" json:"pointer"`
}

func TestGenerateSimple(t *testing.T) {
	schema := Generate(reflect.TypeOf(TestGitSettings{}))

	if schema.Version != SchemaVersion {
		t.Errorf("expected version %q, got %q", SchemaVersion, schema.Version)
	}
}

func TestGenerateWithNestedStruct(t *testing.T) {
	schema := Generate(reflect.TypeOf(TestConfig{}))

	if len(schema.Sections) == 0 {
		t.Fatal("expected at least one section")
	}

	// Find git section
	var gitSection *Section
	for i := range schema.Sections {
		if schema.Sections[i].ID == "git" {
			gitSection = &schema.Sections[i]

			break
		}
	}

	if gitSection == nil {
		t.Fatal("expected git section")
	}

	if len(gitSection.Fields) < 6 {
		t.Errorf("expected at least 6 fields in git section, got %d", len(gitSection.Fields))
	}

	// Verify field properties
	for _, field := range gitSection.Fields {
		switch field.Path {
		case "git.commit_prefix":
			if field.Label != "Commit Prefix" {
				t.Errorf("commit_prefix label: expected 'Commit Prefix', got %q", field.Label)
			}
			if field.Type != TypeString {
				t.Errorf("commit_prefix type: expected string, got %q", field.Type)
			}
			if field.Validation == nil || field.Validation.MaxLength == nil || *field.Validation.MaxLength != 100 {
				t.Error("commit_prefix should have maxlen=100 validation")
			}
		case "git.auto_commit":
			if field.Type != TypeBoolean {
				t.Errorf("auto_commit type: expected boolean, got %q", field.Type)
			}
			if field.Default != true {
				t.Errorf("auto_commit default: expected true, got %v", field.Default)
			}
		case "git.sign_commits":
			if field.ShowWhen == nil {
				t.Error("sign_commits should have showWhen condition")
			} else if field.ShowWhen.Field != "git.auto_commit" {
				t.Errorf("sign_commits showWhen field: expected 'git.auto_commit', got %q", field.ShowWhen.Field)
			}
		case "git.secret":
			if !field.Sensitive {
				t.Error("secret should be marked sensitive")
			}
			if field.Type != TypePassword {
				t.Errorf("secret type: expected password, got %q", field.Type)
			}
		case "git.advanced":
			if !field.Advanced {
				t.Error("advanced should be marked advanced")
			}
		}
	}
}

func TestGenerateWithPointerStruct(t *testing.T) {
	// Test that pointer structs are handled without panic
	schema := Generate(reflect.TypeOf(TestConfig{}))

	// The pointer section should also be generated
	var pointerSection *Section
	for i := range schema.Sections {
		if schema.Sections[i].ID == "pointer" {
			pointerSection = &schema.Sections[i]

			break
		}
	}

	if pointerSection == nil {
		t.Fatal("expected pointer section (from *TestGitSettings)")
	}

	if len(pointerSection.Fields) < 6 {
		t.Errorf("expected at least 6 fields in pointer section, got %d", len(pointerSection.Fields))
	}
}

func TestGenerateWithNilPointerType(t *testing.T) {
	// This should not panic - we use TypeOf, not ValueOf
	type ConfigWithNilPointer struct {
		Setting *struct {
			Value string `json:"value" schema:"label=Value"`
		} `json:"setting"`
	}

	// Should not panic
	schema := Generate(reflect.TypeOf(ConfigWithNilPointer{}))
	if schema == nil {
		t.Error("expected non-nil schema")
	}
}

func TestInferFieldType(t *testing.T) {
	tests := []struct {
		name     string
		goType   reflect.Type
		tags     TagValues
		expected FieldType
	}{
		{
			name:     "string",
			goType:   reflect.TypeOf(""),
			tags:     TagValues{},
			expected: TypeString,
		},
		{
			name:     "bool",
			goType:   reflect.TypeOf(true),
			tags:     TagValues{},
			expected: TypeBoolean,
		},
		{
			name:     "int",
			goType:   reflect.TypeOf(0),
			tags:     TagValues{},
			expected: TypeNumber,
		},
		{
			name:     "float64",
			goType:   reflect.TypeOf(0.0),
			tags:     TagValues{},
			expected: TypeNumber,
		},
		{
			name:     "sensitive string",
			goType:   reflect.TypeOf(""),
			tags:     TagValues{"sensitive": "true"},
			expected: TypePassword,
		},
		{
			name:     "with options",
			goType:   reflect.TypeOf(""),
			tags:     TagValues{"options": "a|b|c"},
			expected: TypeSelect,
		},
		{
			name:     "textarea",
			goType:   reflect.TypeOf(""),
			tags:     TagValues{"textarea": "true"},
			expected: TypeTextarea,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferFieldType(tt.goType, tt.tags)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetSectionMeta(t *testing.T) {
	// Known section
	meta := GetSectionMeta("git")
	if meta.Title != "Git" {
		t.Errorf("expected title 'Git', got %q", meta.Title)
	}
	if meta.Category != "core" {
		t.Errorf("expected category 'core', got %q", meta.Category)
	}

	// Unknown section
	meta = GetSectionMeta("unknown")
	if meta.Title != "Unknown" {
		t.Errorf("expected title 'Unknown', got %q", meta.Title)
	}
	if meta.Category != "features" {
		t.Errorf("expected default category 'features', got %q", meta.Category)
	}
}

func TestGenerateWithRealWorkspaceConfig(t *testing.T) {
	// Import the real WorkspaceConfig to verify schema generation works with it
	// This is a compile-time check that the schema tags are valid
	type GitSettings struct {
		CommitPrefix  string `yaml:"commit_prefix" json:"commit_prefix" schema:"label=Commit Prefix;desc=Pattern for commit messages. Use {key}, {type}, {slug};default=[{key}];maxlen=100"`
		BranchPattern string `yaml:"branch_pattern" json:"branch_pattern" schema:"label=Branch Pattern;desc=Pattern for branch names. Use {key}, {type}, {slug};default={type}/{key}--{slug}"`
		AutoCommit    bool   `yaml:"auto_commit" json:"auto_commit" schema:"label=Auto Commit;desc=Automatically commit after implementation;default=true"`
		SignCommits   bool   `yaml:"sign_commits" json:"sign_commits" schema:"label=Sign Commits;desc=GPG sign commits;default=false;showWhen=git.auto_commit:true"`
		StashOnStart  bool   `yaml:"stash_on_start" json:"stash_on_start" schema:"label=Stash on Start;desc=Auto-stash changes before creating task branch;default=false;advanced"`
		AutoPopStash  bool   `yaml:"auto_pop_stash" json:"auto_pop_stash" schema:"label=Auto Pop Stash;desc=Auto-pop stash after branch creation;default=true;showWhen=git.stash_on_start:true"`
		DefaultBranch string `yaml:"default_branch,omitempty" json:"default_branch,omitempty" schema:"label=Default Branch;desc=Override default branch detection (e.g., main, develop);placeholder=auto-detect"`
	}

	type TestWorkspaceConfig struct {
		Git GitSettings `yaml:"git" json:"git"`
	}

	schema := Generate(reflect.TypeOf(TestWorkspaceConfig{}))

	if len(schema.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(schema.Sections))
	}

	gitSection := schema.Sections[0]
	if gitSection.ID != "git" {
		t.Errorf("expected section ID 'git', got %q", gitSection.ID)
	}

	if len(gitSection.Fields) != 7 {
		t.Errorf("expected 7 fields, got %d", len(gitSection.Fields))
	}

	// Verify specific field properties
	fieldsByPath := make(map[string]Field)
	for _, f := range gitSection.Fields {
		fieldsByPath[f.Path] = f
	}

	// Test commit_prefix field
	if f, ok := fieldsByPath["git.commit_prefix"]; ok {
		if f.Label != "Commit Prefix" {
			t.Errorf("commit_prefix label: expected 'Commit Prefix', got %q", f.Label)
		}
		if f.Default != "[{key}]" {
			t.Errorf("commit_prefix default: expected '[{key}]', got %v", f.Default)
		}
		if f.Validation == nil || f.Validation.MaxLength == nil || *f.Validation.MaxLength != 100 {
			t.Error("commit_prefix should have maxlen=100 validation")
		}
	} else {
		t.Error("missing git.commit_prefix field")
	}

	// Test auto_commit field
	if f, ok := fieldsByPath["git.auto_commit"]; ok {
		if f.Type != TypeBoolean {
			t.Errorf("auto_commit type: expected boolean, got %q", f.Type)
		}
		if f.Default != true {
			t.Errorf("auto_commit default: expected true, got %v", f.Default)
		}
	} else {
		t.Error("missing git.auto_commit field")
	}

	// Test sign_commits field (has showWhen)
	if f, ok := fieldsByPath["git.sign_commits"]; ok {
		if f.ShowWhen == nil {
			t.Error("sign_commits should have showWhen condition")
		} else {
			if f.ShowWhen.Field != "git.auto_commit" {
				t.Errorf("sign_commits showWhen.field: expected 'git.auto_commit', got %q", f.ShowWhen.Field)
			}
			if f.ShowWhen.Equals != true {
				t.Errorf("sign_commits showWhen.equals: expected true, got %v", f.ShowWhen.Equals)
			}
		}
	} else {
		t.Error("missing git.sign_commits field")
	}

	// Test stash_on_start field (advanced)
	if f, ok := fieldsByPath["git.stash_on_start"]; ok {
		if !f.Advanced {
			t.Error("stash_on_start should be marked as advanced")
		}
	} else {
		t.Error("missing git.stash_on_start field")
	}

	// Test default_branch field (placeholder)
	if f, ok := fieldsByPath["git.default_branch"]; ok {
		if f.Placeholder != "auto-detect" {
			t.Errorf("default_branch placeholder: expected 'auto-detect', got %q", f.Placeholder)
		}
	} else {
		t.Error("missing git.default_branch field")
	}
}
