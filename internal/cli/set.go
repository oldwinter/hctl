package cli

import (
	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/mutate"
	"github.com/oldwinter/hctl/internal/remote"
	"github.com/oldwinter/hctl/internal/render"
)

func newSetCmd(opts *options) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "set RESOURCE HARNESS VALUE",
		Short: "Set a harness field (model or provider) with backup and verify",
		Long: `Mutate one field on a harness config.

  hctl set model codex o4-mini --dry-run
  hctl set model codex o4-mini
  hctl set provider hermes custom

Writes are atomic (temp + rename). Previous files are copied to unique,
source-identifying .bak files under backups/ beside the resolved config
(or $HCTL_BACKUP_DIR / $HARNESSCTL_BACKUP_DIR).
After write the snapshot is re-read and must match the intent.

set provider is unsupported for claude (implicit anthropic) and grok
(inferred from base_url). Those commands return a usage error even with
--dry-run — they do not silently no-op.

Secrets are never printed. Unrelated keys are kept; see README for
format-preservation caveats (JSONC comments are dropped).`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 3 {
				return exitcode.Errorf(exitcode.Usage, "set model|provider HARNESS VALUE (example: hctl set model codex o4-mini --dry-run)")
			}
			return nil
		},
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
				Ownership: opts.ownershipOptions(),
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
	return o.openNamed(cfg, o.activeContextName(cfg))
}

func (o *options) openNamed(cfg *config.File, name string) (string, fsx.FS, string, error) {
	nc, err := cfg.Get(name)
	if err != nil {
		return "", nil, "", err
	}
	t, err := remote.Dial(nc, o.home)
	if err != nil {
		return "", nil, "", err
	}
	if o.noProbe {
		t.FS = fsx.WithoutCommandProbes(t.FS)
	}
	return t.Name, t.FS, t.Home, nil
}
