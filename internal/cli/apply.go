package cli

import (
	"github.com/spf13/cobra"

	"github.com/oldwinter/harnessctl/internal/adapters"
	"github.com/oldwinter/harnessctl/internal/desired"
	"github.com/oldwinter/harnessctl/internal/exitcode"
	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/mutate"
	"github.com/oldwinter/harnessctl/internal/render"
)

func newApplyCmd(opts *options) *cobra.Command {
	var (
		file   string
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a desired-state file to the current context",
		Long: `Read desired.toml / desired.yaml and set each listed harness field.

  harnessctl apply -f testdata/desired.toml --dry-run
  harnessctl apply -f testdata/desired.toml
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return exitcode.Errorf(exitcode.Usage, "apply requires -f FILE")
			}
			want, err := desired.Load(file)
			if err != nil {
				return err
			}
			_, fsys, home, err := opts.openTarget()
			if err != nil {
				return err
			}
			rep := model.ApplyReport{DryRun: dryRun}
			for name, d := range want.Harnesses {
				if d.Empty() {
					continue
				}
				ad, err := adapters.ByName(name)
				if err != nil {
					return err
				}
				one, err := mutate.Apply(mutate.Request{
					Adapter:   ad,
					FS:        fsys,
					Home:      home,
					BackupDir: mutate.DefaultBackupDir(opts.configPath),
					Desired:   d,
					DryRun:    dryRun,
				})
				if err != nil {
					return err
				}
				rep.Changes = append(rep.Changes, one.Changes...)
				rep.Backups = append(rep.Backups, one.Backups...)
				rep.Verified = rep.Verified || one.Verified
			}
			if opts.jsonOut {
				return render.JSON(cmd.OutOrStdout(), rep)
			}
			return render.ApplyReport(cmd.OutOrStdout(), rep)
		},
	}
	cmd.Flags().StringVarP(&file, "filename", "f", "", "desired-state file (TOML or YAML)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show changes without writing")
	return cmd
}
