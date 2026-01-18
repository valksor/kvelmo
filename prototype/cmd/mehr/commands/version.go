package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/valksor/go-toolkit/version"
)

// Build-time variables set via ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		// Set version info for go-toolkit (works with test values)
		version.Set(Version, Commit, BuildTime)

		out := cmd.OutOrStdout()
		// Get version info from go-toolkit
		info := version.Info("mehr")
		// Insert "by Valksor" after the first line
		lines := info + "\n  by Valksor"
		_, _ = fmt.Fprintln(out, lines)
	},
}
