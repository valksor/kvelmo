package empty

import "github.com/valksor/go-mehrhof/internal/provider"

// Compile-time interface checks.
var (
	_ provider.Reader     = (*Provider)(nil)
	_ provider.Identifier = (*Provider)(nil)
)
