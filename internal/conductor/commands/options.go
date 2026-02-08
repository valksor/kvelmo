package commands

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// DecodeOptions decodes invocation options into a typed options struct.
func DecodeOptions[T any](inv Invocation) (T, error) {
	var out T
	if len(inv.Options) == 0 {
		return out, nil
	}

	data, err := json.Marshal(inv.Options)
	if err != nil {
		return out, fmt.Errorf("marshal options: %w", err)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, fmt.Errorf("decode options: %w", err)
	}

	return out, nil
}

// GetString gets a string option by key.
func GetString(opts map[string]any, key string) string {
	if opts == nil {
		return ""
	}
	value, ok := opts[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetBool gets a bool option by key.
func GetBool(opts map[string]any, key string) bool {
	if opts == nil {
		return false
	}
	value, ok := opts[key]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return false
		}

		return parsed
	default:
		return false
	}
}

// GetInt gets an int option by key.
func GetInt(opts map[string]any, key string) int {
	if opts == nil {
		return 0
	}
	value, ok := opts[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0
		}

		return int(i)
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}

		return i
	default:
		return 0
	}
}
