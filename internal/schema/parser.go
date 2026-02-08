package schema

import (
	"strconv"
	"strings"
)

// TagValues represents parsed schema tag key-value pairs.
type TagValues map[string]string

// ParseSchemaTag parses a schema struct tag using semicolon-separated syntax.
//
// Format: schema:"key=value;key=value;flag"
//
// Examples:
//
//	schema:"label=Commit Prefix;desc=Pattern for messages;default=[{key}]"
//	schema:"label=Token;sensitive;advanced"
//	schema:"label=Auto Commit;desc=Commit after implementation;default=true"
//	schema:"label=Sign Commits;showWhen=git.auto_commit:true"
//
// Supported keys:
//   - label: Field label for UI
//   - desc: Field description/help text
//   - default: Default value
//   - placeholder: Input placeholder
//   - sensitive: Flag marking field as sensitive (tokens, passwords)
//   - advanced: Flag marking field as advanced (hidden in simple mode)
//   - showWhen: Conditional visibility (format: "path:value")
//   - min, max, maxlen: Numeric/string validation
//   - required: Validation flag
//   - pattern: Regex pattern for validation
//   - patternMsg: Human-readable pattern validation message
//   - options: Select options (format: "value1|value2|value3")
func ParseSchemaTag(tag string) TagValues {
	result := make(TagValues)
	if tag == "" {
		return result
	}

	// Split by semicolon
	pairs := strings.Split(tag, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Check for key=value or just key (flag)
		if idx := strings.Index(pair, "="); idx > 0 {
			key := strings.TrimSpace(pair[:idx])
			value := strings.TrimSpace(pair[idx+1:])
			result[key] = value
		} else {
			// Flag without value (e.g., "sensitive", "advanced", "required")
			result[pair] = "true"
		}
	}

	return result
}

// Get returns the value for a key, or empty string if not present.
func (t TagValues) Get(key string) string {
	return t[key]
}

// GetBool returns true if the key exists and is "true", otherwise false.
func (t TagValues) GetBool(key string) bool {
	return t[key] == "true"
}

// GetInt returns the integer value for a key, or nil if not present or invalid.
func (t TagValues) GetInt(key string) *int {
	v, ok := t[key]
	if !ok || v == "" {
		return nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}

	return &i
}

// Has returns true if the key exists.
func (t TagValues) Has(key string) bool {
	_, ok := t[key]

	return ok
}

// ParseShowWhen parses a showWhen value in the format "path:value".
func ParseShowWhen(value string) *Condition {
	if value == "" {
		return nil
	}

	// Format: "path:value" or "path:!value" for negation
	idx := strings.Index(value, ":")
	if idx <= 0 {
		return nil
	}

	field := strings.TrimSpace(value[:idx])
	condValue := strings.TrimSpace(value[idx+1:])

	if strings.HasPrefix(condValue, "!") {
		// Negation: showWhen=path:!value means show when NOT equal
		return &Condition{
			Field:     field,
			NotEquals: strings.TrimPrefix(condValue, "!"),
		}
	}

	// Parse boolean values
	var equalValue any = condValue
	switch condValue {
	case "true":
		equalValue = true
	case "false":
		equalValue = false
	}

	return &Condition{
		Field:  field,
		Equals: equalValue,
	}
}

// ParseOptions parses pipe-separated select options.
// Format: "option1|option2|option3" or "value:label|value:label".
func ParseOptions(value string) []SelectOption {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, "|")
	options := make([]SelectOption, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check for value:label format
		if idx := strings.Index(part, ":"); idx > 0 {
			options = append(options, SelectOption{
				Value: strings.TrimSpace(part[:idx]),
				Label: strings.TrimSpace(part[idx+1:]),
			})
		} else {
			// Value only, use as both value and label
			options = append(options, SelectOption{
				Value: part,
				Label: part,
			})
		}
	}

	return options
}
