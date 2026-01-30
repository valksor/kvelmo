package security

// BaseScanner provides common fields and methods for all security scanners.
// Embed this in concrete scanner types to avoid repeating the name/enabled boilerplate.
//
// Example usage:
//
//	type GosecScanner struct {
//	    BaseScanner
//	    config *GosecConfig
//	}
//
//	func NewGosecScanner(enabled bool, config *GosecConfig) *GosecScanner {
//	    return &GosecScanner{
//	        BaseScanner: NewBaseScanner("gosec", enabled),
//	        config:      config,
//	    }
//	}
type BaseScanner struct {
	name    string
	enabled bool
}

// NewBaseScanner creates a new BaseScanner with the given name and enabled state.
func NewBaseScanner(name string, enabled bool) BaseScanner {
	return BaseScanner{
		name:    name,
		enabled: enabled,
	}
}

// Name returns the name of the scanner.
func (b *BaseScanner) Name() string {
	return b.name
}

// IsEnabled returns whether the scanner is enabled.
func (b *BaseScanner) IsEnabled() bool {
	return b.enabled
}

// SetEnabled changes the enabled state of the scanner.
func (b *BaseScanner) SetEnabled(enabled bool) {
	b.enabled = enabled
}
