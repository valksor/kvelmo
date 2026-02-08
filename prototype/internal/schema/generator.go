package schema

import (
	"reflect"
	"strconv"
	"strings"
)

// SchemaVersion is the current schema format version.
const SchemaVersion = "1.0"

// Generate creates a Schema from a Go struct type using reflection.
// It uses reflect.Type traversal (not Value) to safely handle nil pointer fields.
//
// Usage:
//
//	schema.Generate(reflect.TypeOf(storage.WorkspaceConfig{}))
func Generate(cfgType reflect.Type) *Schema {
	// Handle pointer types by getting underlying type
	if cfgType.Kind() == reflect.Ptr {
		cfgType = cfgType.Elem()
	}

	if cfgType.Kind() != reflect.Struct {
		return &Schema{Version: SchemaVersion}
	}

	sections := make(map[string]*Section)
	var sectionOrder []string

	// Traverse all fields in the config struct
	for i := range cfgType.NumField() {
		structField := cfgType.Field(i)
		processField(structField, "", sections, &sectionOrder)
	}

	// Build ordered sections list
	result := &Schema{
		Version:  SchemaVersion,
		Sections: make([]Section, 0, len(sectionOrder)),
	}

	for _, sectionID := range sectionOrder {
		if section, ok := sections[sectionID]; ok && len(section.Fields) > 0 {
			result.Sections = append(result.Sections, *section)
		}
	}

	return result
}

// processField recursively processes a struct field and adds it to the appropriate section.
func processField(structField reflect.StructField, pathPrefix string, sections map[string]*Section, sectionOrder *[]string) {
	// Skip unexported fields
	if !structField.IsExported() {
		return
	}

	// Get JSON tag for path building
	jsonTag := structField.Tag.Get("json")
	if jsonTag == "-" {
		return // Skip fields with json:"-"
	}

	// Extract field name from json tag
	fieldName := strings.Split(jsonTag, ",")[0]
	if fieldName == "" {
		fieldName = strings.ToLower(structField.Name)
	}

	// Build the full path
	var path string
	if pathPrefix == "" {
		path = fieldName
	} else {
		path = pathPrefix + "." + fieldName
	}

	// Get field type, handling pointers
	fieldType := structField.Type
	isPointer := fieldType.Kind() == reflect.Ptr
	if isPointer {
		fieldType = fieldType.Elem()
	}

	// Check for schema tag
	schemaTag := structField.Tag.Get("schema")

	// Handle nested structs
	if fieldType.Kind() == reflect.Struct {
		// Check if this is a struct with its own schema tags (leaf struct)
		// or a container struct that should be recursed into
		hasSchemaTag := schemaTag != ""

		if hasSchemaTag {
			// Treat as a single field (e.g., a nested config object)
			addFieldToSection(path, schemaTag, fieldType, sections, sectionOrder)
		} else {
			// Recurse into nested struct
			for j := range fieldType.NumField() {
				processField(fieldType.Field(j), path, sections, sectionOrder)
			}
		}

		return
	}

	// Handle map types (skip for now - needs custom renderers)
	if fieldType.Kind() == reflect.Map {
		return
	}

	// Handle slice types (skip for now - needs custom renderers)
	if fieldType.Kind() == reflect.Slice {
		return
	}

	// Process primitive fields with schema tags
	if schemaTag != "" {
		addFieldToSection(path, schemaTag, fieldType, sections, sectionOrder)
	}
}

// addFieldToSection parses the schema tag and adds a field to the appropriate section.
func addFieldToSection(path, schemaTag string, fieldType reflect.Type, sections map[string]*Section, sectionOrder *[]string) {
	// Parse the schema tag
	tags := ParseSchemaTag(schemaTag)

	// Skip fields without a label (not intended for UI)
	label := tags.Get("label")
	if label == "" {
		return
	}

	// Determine section from path prefix
	sectionID := strings.Split(path, ".")[0]

	// Create section if needed
	if _, ok := sections[sectionID]; !ok {
		meta := GetSectionMeta(sectionID)
		sections[sectionID] = &Section{
			ID:          sectionID,
			Title:       meta.Title,
			Description: meta.Description,
			Icon:        meta.Icon,
			Category:    meta.Category,
			Fields:      []Field{},
		}
		*sectionOrder = append(*sectionOrder, sectionID)
	}

	// Determine field type
	uiType := inferFieldType(fieldType, tags)

	// Build the field
	field := Field{
		Path:        path,
		Type:        uiType,
		Label:       label,
		Description: tags.Get("desc"),
		Placeholder: tags.Get("placeholder"),
		Sensitive:   tags.GetBool("sensitive"),
		Advanced:    tags.GetBool("advanced"),
		Simple:      tags.GetBool("simple"),
		ShowWhen:    ParseShowWhen(tags.Get("showWhen")),
		Options:     ParseOptions(tags.Get("options")),
	}

	// Parse default value based on type
	if defaultStr := tags.Get("default"); defaultStr != "" {
		field.Default = parseDefaultValue(defaultStr, fieldType)
	}

	// Build validation rules
	validation := buildValidation(tags)
	if validation != nil {
		field.Validation = validation
	}

	// Add field to section
	sections[sectionID].Fields = append(sections[sectionID].Fields, field)
}

// inferFieldType determines the UI field type from the Go type and tags.
func inferFieldType(goType reflect.Type, tags TagValues) FieldType {
	// Check for explicit type in tags
	if tags.Has("options") {
		return TypeSelect
	}
	if tags.GetBool("sensitive") {
		return TypePassword
	}
	if tags.GetBool("textarea") {
		return TypeTextarea
	}

	// Infer from Go type
	//nolint:exhaustive // Default case handles all other types as string
	switch goType.Kind() {
	case reflect.Bool:
		return TypeBoolean
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return TypeNumber
	default:
		return TypeString
	}
}

// parseDefaultValue converts a default value string to the appropriate Go type.
func parseDefaultValue(defaultStr string, goType reflect.Type) any {
	//nolint:exhaustive // Default case handles all other types as string
	switch goType.Kind() {
	case reflect.Bool:
		return defaultStr == "true"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v, err := strconv.ParseInt(defaultStr, 10, 64); err == nil {
			return v
		}

		return 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v, err := strconv.ParseUint(defaultStr, 10, 64); err == nil {
			return v
		}

		return 0
	case reflect.Float32, reflect.Float64:
		if v, err := strconv.ParseFloat(defaultStr, 64); err == nil {
			return v
		}

		return 0.0
	default:
		return defaultStr
	}
}

// buildValidation creates ValidationRules from tag values.
func buildValidation(tags TagValues) *ValidationRules {
	validation := &ValidationRules{
		Required:       tags.GetBool("required"),
		Min:            tags.GetInt("min"),
		Max:            tags.GetInt("max"),
		MaxLength:      tags.GetInt("maxlen"),
		Pattern:        tags.Get("pattern"),
		PatternMessage: tags.Get("patternMsg"),
	}

	// Return nil if no validation is specified
	if !validation.Required &&
		validation.Min == nil &&
		validation.Max == nil &&
		validation.MaxLength == nil &&
		validation.Pattern == "" {
		return nil
	}

	return validation
}
