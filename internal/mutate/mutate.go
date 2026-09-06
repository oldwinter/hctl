package mutate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

// FieldWriter can mutate harness config fields.
type FieldWriter interface {
	WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error)
}

// SecretIO copies bearer tokens / env refs without logging values.
type SecretIO interface {
	PeekSecret(fsys fsx.FS, home string) (ref, value string, err error)
	WriteSecret(fsys fsx.FS, home, ref, value string) error
}

// Request is a single-harness apply.
type Request struct {
	Adapter   adapters.Adapter
	FS        fsx.FS
	Home      string
	BackupDir string
	Desired   model.Desired
	DryRun    bool
}

// Apply writes fields, backs up, and re-reads to verify.
func Apply(req Request) (model.ApplyReport, error) {
	rep := model.ApplyReport{DryRun: req.DryRun}
	if req.Desired.Empty() {
		return rep, nil
	}
	before, err := adapters.ReadOneFS(req.Adapter, req.FS, req.Home)
	if err != nil {
		return rep, err
	}
	if req.Desired.Model != "" && req.Desired.Model != before.DefaultModel {
		rep.Changes = append(rep.Changes, model.Change{Harness: before.Name, Field: "model", From: before.DefaultModel, To: req.Desired.Model})
	}
	if req.Desired.Provider != "" && req.Desired.Provider != before.Provider {
		rep.Changes = append(rep.Changes, model.Change{Harness: before.Name, Field: "provider", From: before.Provider, To: req.Desired.Provider})
	}
	if req.Desired.SecretRef != "" && req.Desired.SecretRef != before.SecretRef {
		rep.Changes = append(rep.Changes, model.Change{Harness: before.Name, Field: "secretRef", From: before.SecretRef, To: req.Desired.SecretRef})
	}
	if req.Desired.Provider != "" {
		if err := rejectUnsupportedProvider(req.Adapter.Name()); err != nil {
			return rep, err
		}
	}
	w, ok := req.Adapter.(FieldWriter)
	if !ok {
		return rep, exitcode.Errorf(exitcode.Usage, "harness %s does not support writes", req.Adapter.Name())
	}
	if req.DryRun {
		path := ""
		if len(before.ConfigPaths) > 0 {
			path = before.ConfigPaths[0]
		}
		for i := range rep.Changes {
			rep.Changes[i].Path = path
		}
		return rep, nil
	}
	if req.BackupDir == "" {
		req.BackupDir = DefaultBackupDir("")
	}
	for _, p := range before.ConfigPaths {
		data, err := fsx.ReadMaybe(req.FS, p)
		if err != nil {
			return rep, err
		}
		bak, err := fsx.BackupLocal(req.BackupDir, req.Adapter.Name(), data)
		if err != nil {
			return rep, err
		}
		if bak != "" {
			rep.Backups = append(rep.Backups, bak)
		}
	}
	paths, err := w.WriteFields(req.FS, req.Home, req.Desired)
	if err != nil {
		return rep, err
	}
	for i := range rep.Changes {
		if len(paths) > 0 {
			rep.Changes[i].Path = paths[0]
		}
	}
	after, err := adapters.ReadOneFS(req.Adapter, req.FS, req.Home)
	if err != nil {
		return rep, exitcode.Wrap(exitcode.Verify, err)
	}
	if after.ParseError != "" {
		return rep, exitcode.Errorf(exitcode.Parse, "%s: parse error after write: %s", after.Name, after.ParseError)
	}
	if req.Desired.Model != "" && after.DefaultModel != req.Desired.Model {
		return rep, exitcode.Errorf(exitcode.Verify, "%s: model verify failed: got %q want %q", after.Name, after.DefaultModel, req.Desired.Model)
	}
	if req.Desired.Provider != "" && after.Provider != req.Desired.Provider {
		return rep, exitcode.Errorf(exitcode.Verify, "%s: provider verify failed: got %q want %q", after.Name, after.Provider, req.Desired.Provider)
	}
	if req.Desired.SecretRef != "" && after.SecretRef != req.Desired.SecretRef {
		return rep, exitcode.Errorf(exitcode.Verify, "%s: secretRef verify failed: got %q want %q", after.Name, after.SecretRef, req.Desired.SecretRef)
	}
	rep.Verified = true
	return rep, nil
}

func rejectUnsupportedProvider(name string) error {
	switch name {
	case "claude":
		return exitcode.Errorf(exitcode.Usage, "set provider is unsupported for claude (provider is implicit anthropic); use set model")
	case "grok":
		return exitcode.Errorf(exitcode.Usage, "set provider is unsupported for grok (inferred from base_url); use set model")
	default:
		return nil
	}
}

// DefaultBackupDir is ~/.harnessctl/backups or $HARNESSCTL_BACKUP_DIR.
func DefaultBackupDir(configPath string) string {
	if d := strings.TrimSpace(os.Getenv("HARNESSCTL_BACKUP_DIR")); d != "" {
		return d
	}
	if configPath != "" {
		return filepath.Join(filepath.Dir(configPath), "backups")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".harnessctl", "backups")
	}
	return filepath.Join(home, ".harnessctl", "backups")
}

// CopySecret transfers a secret or env-ref from src to dst. Values are never returned.
func CopySecret(ad adapters.Adapter, srcFS fsx.FS, srcHome string, dstFS fsx.FS, dstHome string, preferRef bool) (model.SecretCopy, error) {
	out := model.SecretCopy{Harness: ad.Name(), Action: "none"}
	io, ok := ad.(SecretIO)
	if !ok {
		return out, fmt.Errorf("harness %s does not support secret copy", ad.Name())
	}
	ref, val, err := io.PeekSecret(srcFS, srcHome)
	if err != nil {
		return out, err
	}
	if r, ok := ad.(interface {
		ReadFS(fsys fsx.FS, home string) (model.Snapshot, error)
	}); ok {
		s, _ := r.ReadFS(srcFS, srcHome)
		out.From = s.SecretFingerprint
		d, _ := r.ReadFS(dstFS, dstHome)
		out.To = d.SecretFingerprint
	}
	if preferRef && ref != "" {
		if err := io.WriteSecret(dstFS, dstHome, ref, ""); err != nil {
			return out, err
		}
		out.Ref = ref
		out.Copied = true
		out.Action = "secret-ref"
		return out, nil
	}
	if val != "" {
		if err := io.WriteSecret(dstFS, dstHome, ref, val); err != nil {
			return out, err
		}
		out.Ref = ref
		out.Copied = true
		out.Action = "bearer"
		if r, ok := ad.(interface {
			ReadFS(fsys fsx.FS, home string) (model.Snapshot, error)
		}); ok {
			d, _ := r.ReadFS(dstFS, dstHome)
			out.To = d.SecretFingerprint
		}
		return out, nil
	}
	if ref != "" {
		if err := io.WriteSecret(dstFS, dstHome, ref, ""); err != nil {
			return out, err
		}
		out.Ref = ref
		out.Copied = true
		out.Action = "secret-ref"
	}
	return out, nil
}
