package cli

import (
	"github.com/spf13/cobra"

	"github.com/oldwinter/harnessctl/internal/adapters"
	"github.com/oldwinter/harnessctl/internal/exitcode"
	"github.com/oldwinter/harnessctl/internal/fsx"
	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/mutate"
	"github.com/oldwinter/harnessctl/internal/render"
)

func newSetCmd(opts *options) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "set RESOURCE HARNESS VALUE",
		Short: "Set a harness field (model or provider) with backup and verify",
		Long: `Mutate one field on a harness config.

  harnessctl set model codex o4-mini --dry-run
  harnessctl set model codex o4-mini
  harnessctl set provider hermes custom

Writes are atomic (temp + rename). The previous file is copied to
~/.harnessctl/backups/<harness>-<timestamp>.bak (or $HARNESSCTL_BACKUP_DIR).
After write the snapshot is re-read and must match the intent.

Secrets are never printed. Unrelated keys are kept; see README for
format-preservation caveats (JSONC comments are dropped).`,
		Args:      cobra.ExactArgs(3),
		ValidArgs: []string{"model", "provider"},
		RunE: func(cmd *cobra.Command, args []string) error {
			var d model.Desired
			switch args[0] {
			case "model":
				d.Model = args[2]
			case "provider":
				d.Provider = args[2]
			default:
				return exitcode.Errorf(exitcode.Usage, "unknown set resource %q (want model|provider)", args[0])
			}
			ad, err := adapters.ByName(args[1])
			if err != nil {
				return err
			}
			_, fsys, home, err := opts.openTarget()
			if err != nil {
				return err
			}
			rep, err := mutate.Apply(mutate.Request{
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
			if opts.jsonOut {
				return render.JSON(cmd.OutOrStdout(), rep)
			}
			return render.ApplyReport(cmd.OutOrStdout(), rep)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the change without writing")
	return cmd
}

func (o *options) openTarget() (contextName string, fsys fsx.FS, home string, err error) {
	cfg, err := o.loadConfig()
	if err != nil {
		return "", nil, "", err
	}
	name, home, err := o.scanHome(cfg)
	if err != nil {
		return "", nil, "", err
	}
	return name, fsx.Local{}, home, nil
}
