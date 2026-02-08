package schema

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// ValidationError represents a validation error for a specific field.
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Validate checks a config value against the schema and returns any validation errors.
func Validate(sch *Schema, cfg any) []ValidationError {
	var errors []ValidationError

	cfgValue := reflect.ValueOf(cfg)
	if cfgValue.Kind() == reflect.Ptr {
		cfgValue = cfgValue.Elem()
	}

	for _, section := range sch.Sections {
		for _, field := range section.Fields {
			if field.Validation == nil {
				continue
			}

			value := getValueByPath(cfgValue, field.Path)
			if err := validateField(field, value); err != nil {
				errors = append(errors, ValidationError{
					Path:    field.Path,
					Message: err.Error(),
				})
			}
		}
	}

	return errors
}

// validateField validates a single field value against its validation rules.
func validateField(field Field, value any) error {
	rules := field.Validation
	if rules == nil {
		return nil
	}

	// Check required
	if rules.Required && isEmpty(value) {
		return fmt.Errorf("%s is required", field.Label)
	}

	// Skip further validation if value is empty and not required
	if isEmpty(value) {
		return nil
	}

	// String validation
	if s, ok := value.(string); ok {
		if rules.MaxLength != nil && len(s) > *rules.MaxLength {
			return fmt.Errorf("%s must be at most %d characters", field.Label, *rules.MaxLength)
		}
		if rules.Pattern != "" {
			matched, err := regexp.MatchString(rules.Pattern, s)
			if err != nil || !matched {
				msg := rules.PatternMessage
				if msg == "" {
					msg = field.Label + " has an invalid format"
				}

				return errors.New(msg)
			}
		}
	}

	// Numeric validation
	if num, ok := toFloat64(value); ok {
		if rules.Min != nil && num < float64(*rules.Min) {
			return fmt.Errorf("%s must be at least %d", field.Label, *rules.Min)
		}
		if rules.Max != nil && num > float64(*rules.Max) {
			return fmt.Errorf("%s must be at most %d", field.Label, *rules.Max)
		}
	}

	return nil
}

// isEmpty checks if a value is considered empty.
func isEmpty(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	//nolint:exhaustive // Default case handles all other types as non-empty
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return false // Booleans are never "empty"
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// toFloat64 attempts to convert a value to float64.
func toFloat64(value any) (float64, bool) {
	v := reflect.ValueOf(value)
	//nolint:exhaustive // Only numeric types can be converted to float64
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	default:
		return 0, false
	}
}

// getValueByPath retrieves a value from a struct using dot-notation path.
func getValueByPath(v reflect.Value, path string) any {
	parts := strings.Split(path, ".")

	for _, part := range parts {
		if !v.IsValid() {
			return nil
		}

		// Handle pointer
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return nil
			}
			v = v.Elem()
		}

		// Handle struct
		if v.Kind() == reflect.Struct {
			v = v.FieldByNameFunc(func(name string) bool {
				// Match by json tag or field name (case-insensitive for json tag)
				field, ok := v.Type().FieldByName(name)
				if !ok {
					return false
				}
				jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]

				return jsonTag == part || strings.EqualFold(name, part)
			})

			continue
		}

		// Handle map
		if v.Kind() == reflect.Map {
			mapKey := reflect.ValueOf(part)
			v = v.MapIndex(mapKey)

			continue
		}

		return nil
	}

	if !v.IsValid() {
		return nil
	}

	return v.Interface()
}

// StripSensitiveFields clears sensitive field values from a config struct.
// This should be called before storing configs in global mode.
func StripSensitiveFields(sch *Schema, cfg any) {
	cfgValue := reflect.ValueOf(cfg)
	if cfgValue.Kind() == reflect.Ptr {
		cfgValue = cfgValue.Elem()
	}

	for _, section := range sch.Sections {
		for _, field := range section.Fields {
			if !field.Sensitive {
				continue
			}
			clearValueByPath(cfgValue, field.Path)
		}
	}
}

// clearValueByPath sets a field to its zero value using dot-notation path.
func clearValueByPath(v reflect.Value, path string) {
	parts := strings.Split(path, ".")

	for i, part := range parts {
		if !v.IsValid() || !v.CanSet() && i > 0 {
			return
		}

		// Handle pointer
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return
			}
			v = v.Elem()
		}

		// Handle struct
		if v.Kind() == reflect.Struct {
			fieldVal := v.FieldByNameFunc(func(name string) bool {
				field, ok := v.Type().FieldByName(name)
				if !ok {
					return false
				}
				jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]

				return jsonTag == part || strings.EqualFold(name, part)
			})

			if i == len(parts)-1 {
				// Last part - clear the value
				if fieldVal.CanSet() {
					fieldVal.Set(reflect.Zero(fieldVal.Type()))
				}

				return
			}

			v = fieldVal

			continue
		}

		return
	}
}

// GetSensitiveFields returns a list of paths for all sensitive fields.
func GetSensitiveFields(sch *Schema) []string {
	var paths []string
	for _, section := range sch.Sections {
		for _, field := range section.Fields {
			if field.Sensitive {
				paths = append(paths, field.Path)
			}
		}
	}

	return paths
}
