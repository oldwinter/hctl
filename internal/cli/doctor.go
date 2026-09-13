package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/render"
)

func newDoctorCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check install, config, key, and onboarding status",
		Example: `  hctl doctor
  hctl describe harness codex`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, fsys, home, err := opts.openTarget()
			if err != nil {
				return writeErr(cmd, err)
			}
			snaps, err := adapters.Scan(fsys, home)
			if err != nil {
				return writeErr(cmd, err)
			}
			checks := adapters.Doctor(snaps)
			if opts.jsonOut {
				if err := render.JSON(cmd.OutOrStdout(), checks); err != nil {
					return err
				}
			} else if err := render.DoctorTable(cmd.OutOrStdout(), checks); err != nil {
				return err
			}
			for _, c := range checks {
				if c.Config == "error" {
					if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Next: hctl describe harness %s\n", c.Name); err != nil {
						return err
					}
					return exitcode.Errorf(exitcode.Parse, "doctor found parse errors")
				}
			}
			return nil
		},
	}
}
