package provider

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/valksor/go-toolkit/providerconfig"
)

// ValidationResult holds the result of configuration validation.
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// AddErrorf adds an error message to the validation result.
func (v *ValidationResult) AddErrorf(format string, args ...any) {
	v.Valid = false
	v.Errors = append(v.Errors, fmt.Sprintf(format, args...))
}

// Error returns a combined error message if validation failed.
func (v *ValidationResult) Error() error {
	if v.Valid {
		return nil
	}

	return fmt.Errorf("configuration validation failed: %s", strings.Join(v.Errors, "; "))
}

// Validator defines configuration validation rules for a provider.
type Validator struct {
	config       providerconfig.Config
	providerName string
	result       ValidationResult
}

// NewValidator creates a new validator for a provider.
func NewValidator(providerName string, cfg providerconfig.Config) *Validator {
	return &Validator{
		config:       cfg,
		providerName: providerName,
		result:       ValidationResult{Valid: true},
	}
}

// Validate runs all registered validations and returns the result.
func (v *Validator) Validate() *ValidationResult {
	return &v.result
}

// Required checks that a string field is not empty.
func (v *Validator) Required(field string) *Validator {
	if v.config.GetString(field) == "" {
		v.result.AddErrorf("%s: %s is required", v.providerName, field)
	}

	return v
}

// RequiredAny checks that at least one of the fields is not empty.
func (v *Validator) RequiredAny(fields ...string) *Validator {
	hasValue := false
	for _, field := range fields {
		if v.config.GetString(field) != "" {
			hasValue = true

			break
		}
	}
	if !hasValue {
		v.result.AddErrorf("%s: at least one of %s is required", v.providerName, strings.Join(fields, ", "))
	}

	return v
}

// RequiredTogether checks that if one field is set, all related fields must also be set.
func (v *Validator) RequiredTogether(fields ...string) *Validator {
	var hasValue, hasEmpty bool
	for _, field := range fields {
		val := v.config.GetString(field)
		if val != "" {
			hasValue = true
		} else {
			hasEmpty = true
		}
	}
	if hasValue && hasEmpty {
		v.result.AddErrorf("%s: %s must be specified together", v.providerName, strings.Join(fields, ", "))
	}

	return v
}

// URL checks that a field contains a valid URL.
func (v *Validator) URL(field string) *Validator {
	val := v.config.GetString(field)
	if val == "" {
		return v // Empty is OK, use Required() if needed
	}
	if _, err := url.Parse(val); err != nil {
		v.result.AddErrorf("%s: %s must be a valid URL: %v", v.providerName, field, err)
	}

	return v
}

// HTTPS checks that a field contains an HTTPS URL.
func (v *Validator) HTTPS(field string) *Validator {
	val := v.config.GetString(field)
	if val == "" {
		return v // Empty is OK
	}
	if !strings.HasPrefix(val, "https://") {
		v.result.AddErrorf("%s: %s must use HTTPS", v.providerName, field)
	}

	return v
}

// MinLength checks that a string field meets minimum length.
func (v *Validator) MinLength(field string, minLength int) *Validator {
	val := v.config.GetString(field)
	if val == "" {
		return v // Empty is OK
	}
	if len(val) < minLength {
		v.result.AddErrorf("%s: %s must be at least %d characters", v.providerName, field, minLength)
	}

	return v
}

// Match checks that a field matches a regular expression pattern.
func (v *Validator) Match(field, pattern string) *Validator {
	val := v.config.GetString(field)
	if val == "" {
		return v // Empty is OK
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		v.result.AddErrorf("%s: invalid pattern for field %s: %v", v.providerName, field, err)

		return v
	}
	if !re.MatchString(val) {
		v.result.AddErrorf("%s: %s has invalid format", v.providerName, field)
	}

	return v
}

// OneOf checks that a field's value is one of the allowed values.
func (v *Validator) OneOf(field string, allowedValues []string) *Validator {
	val := v.config.GetString(field)
	if val == "" {
		return v // Empty is OK
	}
	for _, allowed := range allowedValues {
		if val == allowed {
			return v
		}
	}
	v.result.AddErrorf("%s: %s must be one of: %s", v.providerName, field, strings.Join(allowedValues, ", "))

	return v
}

// Positive checks that an int field is positive (or zero).
func (v *Validator) Positive(field string) *Validator {
	val := v.config.GetInt(field)
	if val < 0 {
		v.result.AddErrorf("%s: %s must be positive or zero", v.providerName, field)
	}

	return v
}

// ValidateConfig is a convenience function for validating provider configuration.
func ValidateConfig(providerName string, cfg providerconfig.Config, validations func(*Validator)) error {
	v := NewValidator(providerName, cfg)
	validations(v)

	return v.Validate().Error()
}
