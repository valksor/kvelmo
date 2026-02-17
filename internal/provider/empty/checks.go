package empty

import "github.com/valksor/go-toolkit/workunit"

// Compile-time interface checks.
var (
	_ workunit.Reader     = (*Provider)(nil)
	_ workunit.Identifier = (*Provider)(nil)
)
