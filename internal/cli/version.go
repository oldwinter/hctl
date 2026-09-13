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
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "hctl"
			if filepath.Base(os.Args[0]) == "harnessctl" {
				name = "harnessctl"
			}
			out := cmd.OutOrStdout()
			if _, err := fmt.Fprintf(out, "%s version %s\n", name, Version); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "commit: %s\n", Commit); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "built:  %s\n", Date); err != nil {
				return err
			}
			if Commit == "unknown" || Date == "unknown" {
				if _, err := fmt.Fprintln(out, "hint: just build injects commit and date; plain go install does not"); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
