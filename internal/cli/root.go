package cli

import "github.com/spf13/cobra"

// NewRootCommand creates the root command for the Higher CLI.
func NewRootCommand() *cobra.Command {
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

	return command
}
