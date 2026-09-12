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
		Short: "Print hctl version, commit, and build date",
		Long: `Print version metadata.

just build / just release inject Version, Commit, and Date via -ldflags
(see justfile). Plain go build / go install leaves commit and date as "unknown".`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			name := "hctl"
			if filepath.Base(os.Args[0]) == "harnessctl" {
				name = "harnessctl"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s version %s\n", name, Version)
			fmt.Fprintf(cmd.OutOrStdout(), "commit: %s\n", Commit)
			fmt.Fprintf(cmd.OutOrStdout(), "built:  %s\n", Date)
			if Commit == "unknown" || Date == "unknown" {
				fmt.Fprintln(cmd.OutOrStdout(), "hint: just build injects commit and date; plain go install does not")
			}
		},
	}
}
