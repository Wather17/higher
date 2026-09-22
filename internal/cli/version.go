package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Version, Commit, and BuildDate are set by release builds through ldflags.
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Exiba a versão instalada do Higher",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			version := Version
			if version != "dev" && !strings.HasPrefix(version, "v") {
				version = "v" + version
			}
			_, err := fmt.Fprintf(command.OutOrStdout(), "Higher %s (commit %s, build %s)\n", version, Commit, BuildDate)
			return err
		},
	}
}
