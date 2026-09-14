package cli

import (
	"time"

	"github.com/Wather17/higher/internal/storage"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	Store *storage.Store
	Now   func() time.Time
}

// NewRootCommand creates the root command for the Higher CLI.
func NewRootCommand(dependencies ...Dependencies) *cobra.Command {
	var dependency Dependencies
	if len(dependencies) > 0 {
		dependency = dependencies[0]
	}
	if dependency.Now == nil {
		dependency.Now = time.Now
	}

	command := &cobra.Command{
		Use:           "higher",
		Short:         "Gerencie suas finanças pessoais pelo terminal",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}
	command.AddCommand(newSubscriptionCommand(dependency))

	return command
}
