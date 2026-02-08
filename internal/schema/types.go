// Package schema provides runtime schema generation from Go struct tags for settings UI.
package schema

// FieldType represents the type of a schema field for UI rendering.
type FieldType string

const (
	TypeString   FieldType = "string"
	TypeBoolean  FieldType = "boolean"
	TypeNumber   FieldType = "number"
	TypeSelect   FieldType = "select"
	TypeTextarea FieldType = "textarea"
	TypePassword FieldType = "password"
)

// SelectOption represents a choice in a select/dropdown field.
type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ValidationRules defines validation constraints for a field.
type ValidationRules struct {
	Required       bool   `json:"required,omitempty"`
	Min            *int   `json:"min,omitempty"`
	Max            *int   `json:"max,omitempty"`
	MaxLength      *int   `json:"maxLength,omitempty"`
	Pattern        string `json:"pattern,omitempty"`
	PatternMessage string `json:"patternMessage,omitempty"`
}

// Condition defines when a field should be visible.
type Condition struct {
	Field     string `json:"field"`               // Path to the controlling field
	Equals    any    `json:"equals,omitempty"`    // Show when field equals this value
	NotEquals any    `json:"notEquals,omitempty"` // Show when field does not equal this value
}

// Field represents a single configuration field in the schema.
type Field struct {
	Path        string           `json:"path"`                  // Dot-notation path: "git.commit_prefix"
	Type        FieldType        `json:"type"`                  // UI field type
	Label       string           `json:"label"`                 // Human-readable label
	Description string           `json:"description,omitempty"` // Help text
	Placeholder string           `json:"placeholder,omitempty"` // Input placeholder
	Default     any              `json:"default,omitempty"`     // Default value
	Options     []SelectOption   `json:"options,omitempty"`     // For select/multiselect types
	Validation  *ValidationRules `json:"validation,omitempty"`  // Validation constraints
	Sensitive   bool             `json:"sensitive,omitempty"`   // Mask in UI, protect in API
	ShowWhen    *Condition       `json:"showWhen,omitempty"`    // Conditional visibility
	Advanced    bool             `json:"advanced,omitempty"`    // Hide in simple mode
	Simple      bool             `json:"simple,omitempty"`      // Show in simple mode
}

// Section groups related fields together in the UI.
type Section struct {
	ID          string  `json:"id"`                    // Unique section identifier
	Title       string  `json:"title"`                 // Display title
	Description string  `json:"description,omitempty"` // Section description
	Icon        string  `json:"icon,omitempty"`        // Icon name (lucide-react)
	Category    string  `json:"category"`              // "core" | "providers" | "features"
	Fields      []Field `json:"fields"`                // Fields in this section
}

// Schema represents the complete settings schema.
type Schema struct {
	Version  string    `json:"version"`  // Schema version for compatibility
	Sections []Section `json:"sections"` // All settings sections
}

// SectionMeta holds metadata for a section that is not defined in struct tags.
type SectionMeta struct {
	Title       string
	Description string
	Icon        string
	Category    string // "core" | "providers" | "features"
}
