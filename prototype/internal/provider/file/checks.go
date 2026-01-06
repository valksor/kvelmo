package file

import (
	"github.com/valksor/go-mehrhof/internal/provider"
)

// Compile-time interface checks to ensure Provider implements declared capabilities.
var (
	_ provider.Reader      = (*Provider)(nil)
	_ provider.Identifier  = (*Provider)(nil)
	_ provider.Snapshotter = (*Provider)(nil)
)
