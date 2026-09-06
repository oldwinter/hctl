package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print hctl version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			name := "hctl"
			if filepath.Base(os.Args[0]) == "harnessctl" {
				name = "harnessctl"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s version %s", name, Version)
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
