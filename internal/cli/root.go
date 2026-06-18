// internal/cli/root.go
package cli

import (
	"github.com/jiahongchen/race-results/internal/provider"
	"github.com/spf13/cobra"
)

// NewRoot builds the root command with all subcommands wired to reg.
func NewRoot(reg *provider.Registry) *cobra.Command {
	root := &cobra.Command{
		Use:           "race-results",
		Short:         "Look up a runner's race result by race name + bib",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newLookupCmd(reg))
	return root
}
