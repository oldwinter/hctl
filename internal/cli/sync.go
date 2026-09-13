package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/mutate"
	"github.com/oldwinter/hctl/internal/render"
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

  hctl sync --from mba --to box --harness codex,claude --dry-run
  hctl sync --from mba --to box --harness codex --fields model,provider,secret-ref

Bearer copy (--fields …,secret) transfers key bytes over the filesystem/SSH
without logging them. Only fingerprints are shown. Prefer secret-ref.

Risk: copying a bearer token duplicates a credential. Rotate if a host is untrusted.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" || to == "" {
				return exitcode.Errorf(exitcode.Usage, "sync requires --from and --to (example: hctl sync --from mba --to box --harness codex --dry-run)")
			}
			cfg, err := opts.loadConfig()
			if err != nil {
				return err
			}
			_, srcFS, srcHome, err := opts.openNamed(cfg, from)
			if err != nil {
				return wrapMissingContext(err, opts.configPath)
			}
			_, dstFS, dstHome, err := opts.openNamed(cfg, to)
			if err != nil {
				return wrapMissingContext(err, opts.configPath)
			}
			names := adapters.Names()
			if harness != "" {
				names = splitCSV(harness)
			}
			batch := len(names) > 1
			var skips []string
			fieldSet := map[string]bool{}
			if fields == "" {
				fields = "model,provider"
			}
			for _, f := range splitCSV(fields) {
				if !model.KnownSyncField(f) {
					return exitcode.Errorf(exitcode.Usage, "unknown sync field %q", f)
				}
				fieldSet[f] = true
			}
			type prepared struct {
				field     mutate.Plan
				hasField  bool
				secret    mutate.SecretPlan
				hasSecret bool
			}
			preparedBatch := make([]prepared, 0, len(names))
			for _, name := range names {
				ad, err := adapters.ByName(name)
				if err != nil {
					return err
				}
				src, err := adapters.ReadOne(ad, srcFS, srcHome)
				if err != nil {
					return err
				}
				if src.ParseError != "" {
					return exitcode.Errorf(exitcode.Parse, "%s source config has a parse error", src.Name)
				}
				var project []string
				for _, field := range []string{"model", "provider", "secret-ref"} {
					if fieldSet[field] {
						project = append(project, field)
					}
				}
				project = adapters.FilterDesiredFields(ad, project)
				d := model.Project(src, project...)
				entry := prepared{}
				if !d.Empty() {
					entry.field, err = mutate.Preflight(mutate.Request{
						Adapter:   ad,
						FS:        dstFS,
						Home:      dstHome,
						BackupDir: mutate.DefaultBackupDir(opts.configPath),
						Desired:   d,
						DryRun:    dryRun,
						Ownership: opts.ownershipOptions(),
					})
					if err != nil {
						if batch && exitcode.From(err) == exitcode.Usage {
							skips = append(skips, fmt.Sprintf("skipped %s: %s", ad.Name(), err.Error()))
							continue
						}
						return err
					}
					entry.hasField = true
				}
				if fieldSet["secret"] {
					entry.secret, err = mutate.PreflightSecret(mutate.SecretRequest{
						Adapter:   ad,
						SrcFS:     srcFS,
						SrcHome:   srcHome,
						DstFS:     dstFS,
						DstHome:   dstHome,
						Provider:  d.Provider,
						DryRun:    dryRun,
						BackupDir: mutate.DefaultBackupDir(opts.configPath),
						Ownership: opts.ownershipOptions(),
					})
					if err != nil {
						if batch && exitcode.From(err) == exitcode.Usage {
							skips = append(skips, fmt.Sprintf("skipped %s: %s", ad.Name(), err.Error()))
							continue
						}
						return err
					}
					entry.hasSecret = true
				}
				preparedBatch = append(preparedBatch, entry)
			}
			rep := model.ApplyReport{DryRun: dryRun, Notes: skips}
			var copies []model.SecretCopy
			for _, entry := range preparedBatch {
				if entry.hasField {
					one, err := mutate.ApplyPrepared(entry.field)
					if err != nil {
						return err
					}
					rep.Changes = append(rep.Changes, one.Changes...)
					rep.Backups = append(rep.Backups, one.Backups...)
					rep.Verified = rep.Verified || one.Verified
				}
				if entry.hasSecret {
					cp, err := mutate.ApplyPreparedSecret(entry.secret)
					if err != nil {
						return err
					}
					copies = append(copies, cp)
					rep.Backups = append(rep.Backups, cp.Backups...)
					if cp.Copied {
						rep.Verified = true
					}
				}
			}
			rep.Secrets = copies
			if opts.wantJSON() {
				return render.JSON(cmd.OutOrStdout(), rep)
			}
			return render.ApplyReport(cmd.OutOrStdout(), rep)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "source context")
	cmd.Flags().StringVar(&to, "to", "", "destination context")
	cmd.Flags().StringVar(&harness, "harness", "", "comma-separated harness names (default: all writable)")
	cmd.Flags().StringVar(&fields, "fields", "model,provider", "model,provider,secret-ref,secret")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show changes without writing")
	return cmd
}

func wrapMissingContext(err error, configPath string) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("%w; add it with hctl config set-context NAME (see %s)", err, configPath)
	}
	return err
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
