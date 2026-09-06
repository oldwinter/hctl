package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print harnessctl version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "harnessctl version %s", Version)
			if Commit != "" && Commit != "unknown" {
				fmt.Fprintf(cmd.OutOrStdout(), " (%s)", Commit)
			}
			if Date != "" && Date != "unknown" {
				fmt.Fprintf(cmd.OutOrStdout(), " %s", Date)
			}
			fmt.Fprintln(cmd.OutOrStdout())
		},
	}
}
