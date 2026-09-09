package cli

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/desired"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/mutate"
	"github.com/oldwinter/hctl/internal/render"
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
			names := make([]string, 0, len(want.Harnesses))
			for name := range want.Harnesses {
				names = append(names, name)
			}
			sort.Strings(names)
			plans := make([]mutate.Plan, 0, len(names))
			for _, name := range names {
				d := want.Harnesses[name]
				if d.Empty() {
					continue
				}
				ad, err := adapters.ByName(name)
				if err != nil {
					return err
				}
				plan, err := mutate.Preflight(mutate.Request{
					Adapter:   ad,
					FS:        fsys,
					Home:      home,
					BackupDir: mutate.DefaultBackupDir(opts.configPath),
					Desired:   d,
					DryRun:    dryRun,
					Ownership: opts.ownershipOptions(),
				})
				if err != nil {
					return err
				}
				plans = append(plans, plan)
			}
			rep := model.ApplyReport{DryRun: dryRun}
			for _, plan := range plans {
				one, err := mutate.ApplyPrepared(plan)
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
