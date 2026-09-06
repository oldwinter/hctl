package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/mutate"
	"github.com/oldwinter/hctl/internal/render"
	"github.com/oldwinter/hctl/internal/secret"
)

func newSyncCmd(opts *options) *cobra.Command {
	var (
		from, to string
		harness  string
		fields   string
		dryRun   bool
	)
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Copy selected harness fields from one context to another",
		Long: `Sync model/provider/secret-ref (and optionally bearer tokens) across contexts.

  harnessctl sync --from mba --to box --harness codex,claude --dry-run
  harnessctl sync --from mba --to box --harness codex --fields model,provider,secret-ref

Bearer copy (--fields …,secret) transfers key bytes over the filesystem/SSH
without logging them. Only fingerprints are shown. Prefer secret-ref.

Risk: copying a bearer token duplicates a credential. Rotate if a host is untrusted.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" || to == "" {
				return exitcode.Errorf(exitcode.Usage, "sync requires --from and --to")
			}
			cfg, err := opts.loadConfig()
			if err != nil {
				return err
			}
			_, srcFS, srcHome, err := opts.openNamed(cfg, from)
			if err != nil {
				return err
			}
			_, dstFS, dstHome, err := opts.openNamed(cfg, to)
			if err != nil {
				return err
			}
			names := adapters.Names()
			if harness != "" {
				names = splitCSV(harness)
			}
			fieldSet := map[string]bool{}
			if fields == "" {
				fields = "model,provider"
			}
			for _, f := range splitCSV(fields) {
				fieldSet[f] = true
			}
			rep := model.ApplyReport{DryRun: dryRun}
			var copies []model.SecretCopy
			for _, name := range names {
				ad, err := adapters.ByName(name)
				if err != nil {
					return err
				}
				src, err := adapters.ReadOneFS(ad, srcFS, srcHome)
				if err != nil {
					return err
				}
				d := model.Desired{}
				if fieldSet["model"] {
					d.Model = src.DefaultModel
				}
				if fieldSet["provider"] {
					d.Provider = src.Provider
				}
				if fieldSet["secret-ref"] {
					d.SecretRef = src.SecretRef
				}
				if !d.Empty() {
					one, err := mutate.Apply(mutate.Request{
						Adapter:   ad,
						FS:        dstFS,
						Home:      dstHome,
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
				if fieldSet["secret"] && !dryRun {
					cp, err := mutate.CopySecret(ad, srcFS, srcHome, dstFS, dstHome, false)
					if err != nil {
						return err
					}
					copies = append(copies, cp)
				} else if fieldSet["secret"] && dryRun {
					copies = append(copies, model.SecretCopy{Harness: name, Action: "bearer", From: src.SecretFingerprint})
				}
			}
			type out struct {
				model.ApplyReport
				Secrets []model.SecretCopy `json:"secrets,omitempty"`
			}
			payload := out{ApplyReport: rep, Secrets: copies}
			if opts.jsonOut {
				return render.JSON(cmd.OutOrStdout(), payload)
			}
			if err := render.ApplyReport(cmd.OutOrStdout(), rep); err != nil {
				return err
			}
			for _, c := range copies {
				line := "secret " + c.Harness + " action=" + c.Action
				if c.From != "" {
					line += " from=sha256:" + c.From
				}
				if c.To != "" {
					line += " to=sha256:" + c.To
				}
				if c.Ref != "" {
					line += " ref=" + c.Ref
				}
				cmd.Println(secret.Redact(line))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "source context")
	cmd.Flags().StringVar(&to, "to", "", "destination context")
	cmd.Flags().StringVar(&harness, "harness", "", "comma-separated harness names (default: all writable)")
	cmd.Flags().StringVar(&fields, "fields", "model,provider", "model,provider,secret-ref,secret")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show changes without writing")
	return cmd
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
