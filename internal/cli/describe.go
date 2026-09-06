package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oldwinter/harnessctl/internal/adapters"
	"github.com/oldwinter/harnessctl/internal/render"
)

func newDescribeCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "describe RESOURCE NAME",
		Short: "Show details of a harness snapshot",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isHarnessResource(args[0]) {
				return writeErr(cmd, fmt.Errorf("unknown resource %q (want harness)", args[0]))
			}
			ctxName, fsys, home, err := opts.openTarget()
			if err != nil {
				return writeErr(cmd, err)
			}
			a, err := adapters.ByName(args[1])
			if err != nil {
				return writeErr(cmd, err)
			}
			snap, err := adapters.ReadOneFS(a, fsys, home)
			if err != nil {
				return writeErr(cmd, err)
			}
			if opts.jsonOut {
				return render.JSON(cmd.OutOrStdout(), snap)
			}
			return render.Describe(cmd.OutOrStdout(), ctxName, snap)
		},
	}
}

func isHarnessResource(s string) bool {
	s = strings.ToLower(s)
	return s == "harness" || s == "harnesses"
}
