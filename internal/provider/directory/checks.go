package directory

import (
	"github.com/valksor/go-toolkit/snapshot"
	"github.com/valksor/go-toolkit/workunit"
)

// Compile-time interface checks to ensure Provider implements declared capabilities.
var (
	_ workunit.Reader      = (*Provider)(nil)
	_ workunit.Identifier  = (*Provider)(nil)
	_ snapshot.Snapshotter = (*Provider)(nil)
)
