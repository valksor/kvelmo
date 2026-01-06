package browser

// NOTE: Registration will be handled through conductor options.
// This file is kept for potential future registry functionality.

// RegisterFunc is a function that can register browser capabilities.
type RegisterFunc func(interface{}) error

// DefaultRegister returns the default registration function.
func DefaultRegister() RegisterFunc {
	return func(registry interface{}) error {
		// Registration is handled through conductor options
		return nil
	}
}
