package cli

import (
	"github.com/spf13/cobra"

	"github.com/oldwinter/harnessctl/internal/adapters"
	"github.com/oldwinter/harnessctl/internal/render"
)

func newDoctorCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check install, config, key, and onboarding status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := opts.loadConfig()
			if err != nil {
				return writeErr(cmd, err)
			}
			_, home, err := opts.scanHome(cfg)
			if err != nil {
				return writeErr(cmd, err)
			}
			snaps, err := adapters.Scan(home)
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
					return writeErr(cmd, errDoctor)
				}
			}
			return nil
		},
	}
}

var errDoctor = errString("doctor found parse errors")

type errString string

func (e errString) Error() string { return string(e) }
