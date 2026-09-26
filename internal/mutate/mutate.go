package mutate

import (
	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/ownership"
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

type desiredValidator interface {
	ValidateDesired(fsys fsx.FS, home string, d model.Desired) error
}

type secretWriteValidator interface {
	ValidateSecretWrite(fsys fsx.FS, home, provider string) error
}

// Request is a single-harness apply.
type Request struct {
	Adapter   adapters.Adapter
	FS        fsx.FS
	Home      string
	BackupDir string
	Desired   model.Desired
	DryRun    bool
	Ownership ownership.Options
}

// Plan is a fully validated field mutation. Creating a plan is read-only.
type Plan struct {
	req    Request
	report model.ApplyReport
	before model.Snapshot
	writer FieldWriter
}

// Preflight validates one field mutation without creating backups or writing.
func Preflight(req Request) (Plan, error) {
	rep := model.ApplyReport{DryRun: req.DryRun}
	plan := Plan{req: req, report: rep}
	if req.Desired.Empty() {
		return plan, nil
	}
	before, err := adapters.ReadOne(req.Adapter, req.FS, req.Home)
	if err != nil {
		return plan, err
	}
	if before.ParseError != "" {
		return plan, snapshotParseError(before, "destination")
	}
	plan.before = before
	rep.Changes = model.ChangesFromDesired(before.Name, before, req.Desired)
	if err := ownership.Check(req.FS, req.Home, before.ConfigPaths, req.Ownership); err != nil {
		return plan, err
	}
	if validator, ok := req.Adapter.(desiredValidator); ok {
		if err := validator.ValidateDesired(req.FS, req.Home, req.Desired); err != nil {
			return plan, err
		}
	}
	w, ok := req.Adapter.(FieldWriter)
	if !ok {
		return plan, exitcode.Errorf(exitcode.Usage, "harness %s does not support writes", req.Adapter.Name())
	}
	plan.report = rep
	plan.writer = w
	return plan, nil
}

// Apply preflights, backs up, writes fields, and re-reads to verify.
func Apply(req Request) (model.ApplyReport, error) {
	plan, err := Preflight(req)
	if err != nil {
		return plan.report, err
	}
	return ApplyPrepared(plan)
}

// ApplyPrepared executes a Plan returned by Preflight.
func ApplyPrepared(plan Plan) (model.ApplyReport, error) {
	req := plan.req
	rep := plan.report
	if len(rep.Changes) == 0 {
		return rep, nil
	}
	if req.DryRun {
		path := ""
		if len(plan.before.ConfigPaths) > 0 {
			path = plan.before.ConfigPaths[0]
		}
		for i := range rep.Changes {
			rep.Changes[i].Path = path
		}
		return rep, nil
	}
	if req.BackupDir == "" {
		req.BackupDir = DefaultBackupDir("")
	}
	// Back up every known config before invoking a writer that may partially
	// mutate files and return an error without reporting their paths.
	for _, p := range plan.before.ConfigPaths {
		data, err := fsx.ReadMaybe(req.FS, p)
		if err != nil {
			return rep, err
		}
		bak, err := fsx.BackupLocal(req.BackupDir, req.Adapter.Name(), p, data)
		if err != nil {
			return rep, err
		}
		if bak != "" {
			rep.Backups = append(rep.Backups, bak)
		}
	}
	paths, err := plan.writer.WriteFields(req.FS, req.Home, req.Desired)
	if err != nil {
		return rep, err
	}
	for i := range rep.Changes {
		if len(paths) > 0 {
			rep.Changes[i].Path = paths[0]
		}
	}
	after, err := adapters.ReadOne(req.Adapter, req.FS, req.Home)
	if err != nil {
		return rep, exitcode.Wrap(exitcode.Verify, err)
	}
	if after.ParseError != "" {
		return rep, exitcode.Errorf(exitcode.Parse, "%s: parse error after write: %s", after.Name, after.ParseError)
	}
	if leftover := model.ChangesFromDesired(after.Name, after, req.Desired); len(leftover) > 0 {
		c := leftover[0]
		return rep, exitcode.Errorf(exitcode.Verify, "%s: %s verify failed: got %q want %q", after.Name, c.Field, c.From, c.To)
	}
	rep.Verified = true
	return rep, nil
}

// DefaultBackupDir is $HCTL_BACKUP_DIR, else $HARNESSCTL_BACKUP_DIR, else
// backups/ beside the resolved config file.
func DefaultBackupDir(configPath string) string {
	return config.BackupDir(configPath)
}

// SecretRequest describes a destination-guarded secret transfer.
type SecretRequest struct {
	Adapter   adapters.Adapter
	SrcFS     fsx.FS
	SrcHome   string
	DstFS     fsx.FS
	DstHome   string
	PreferRef bool
	Provider  string
	DryRun    bool
	BackupDir string
	Ownership ownership.Options
}

// SecretPlan holds a validated secret transfer. Secret values remain private.
type SecretPlan struct {
	req              SecretRequest
	io               SecretIO
	ref              string
	value            string
	out              model.SecretCopy
	destinationPaths []string
}

// PreflightSecret validates and reads a transfer without mutating its destination.
func PreflightSecret(req SecretRequest) (SecretPlan, error) {
	plan := SecretPlan{req: req, out: model.SecretCopy{Harness: req.Adapter.Name(), Action: "none"}}
	io, ok := req.Adapter.(SecretIO)
	if !ok {
		return plan, exitcode.Errorf(exitcode.Usage, "harness %s does not support secret copy", req.Adapter.Name())
	}
	dst, err := adapters.ReadOne(req.Adapter, req.DstFS, req.DstHome)
	if err != nil {
		return plan, err
	}
	if dst.ParseError != "" {
		return plan, snapshotParseError(dst, "destination")
	}
	if err := ownership.Check(req.DstFS, req.DstHome, dst.ConfigPaths, req.Ownership); err != nil {
		return plan, err
	}
	if validator, ok := req.Adapter.(secretWriteValidator); ok {
		if err := validator.ValidateSecretWrite(req.DstFS, req.DstHome, req.Provider); err != nil {
			return plan, err
		}
	}
	src, err := adapters.ReadOne(req.Adapter, req.SrcFS, req.SrcHome)
	if err != nil {
		return plan, err
	}
	if src.ParseError != "" {
		return plan, snapshotParseError(src, "source")
	}
	ref, value, err := io.PeekSecret(req.SrcFS, req.SrcHome)
	if err != nil {
		return plan, err
	}
	plan.io = io
	plan.ref = ref
	plan.value = value
	plan.out.From = src.SecretFingerprint
	plan.out.To = dst.SecretFingerprint
	plan.out.Ref = ref
	plan.destinationPaths = dst.ConfigPaths
	switch {
	case req.PreferRef && ref != "":
		plan.out.Action = "secret-ref"
	case value != "":
		plan.out.Action = "bearer"
	case ref != "":
		plan.out.Action = "secret-ref"
	}
	return plan, nil
}

func snapshotParseError(snap model.Snapshot, side string) error {
	return exitcode.Errorf(exitcode.Parse, "%s %s config has a parse error", snap.Name, side)
}

// ApplyPreparedSecret executes a SecretPlan returned by PreflightSecret.
func ApplyPreparedSecret(plan SecretPlan) (model.SecretCopy, error) {
	out := plan.out
	if plan.req.DryRun || out.Action == "none" {
		return out, nil
	}
	backupDir := plan.req.BackupDir
	if backupDir == "" {
		backupDir = DefaultBackupDir("")
	}
	for _, path := range plan.destinationPaths {
		data, err := fsx.ReadMaybe(plan.req.DstFS, path)
		if err != nil {
			return out, err
		}
		backup, err := fsx.BackupLocal(backupDir, plan.req.Adapter.Name(), path, data)
		if err != nil {
			return out, err
		}
		if backup != "" {
			out.Backups = append(out.Backups, backup)
		}
	}
	ref, value := plan.ref, plan.value
	if plan.req.PreferRef && ref != "" {
		value = ""
	}
	if err := plan.io.WriteSecret(plan.req.DstFS, plan.req.DstHome, ref, value); err != nil {
		return out, err
	}
	after, err := adapters.ReadOne(plan.req.Adapter, plan.req.DstFS, plan.req.DstHome)
	if err != nil {
		return out, exitcode.Wrap(exitcode.Verify, err)
	}
	if out.Action == "secret-ref" && after.SecretRef != ref {
		return out, exitcode.Errorf(exitcode.Verify, "%s: secretRef verify failed", plan.req.Adapter.Name())
	}
	if out.Action == "bearer" && out.From != "" && after.SecretFingerprint != out.From {
		return out, exitcode.Errorf(exitcode.Verify, "%s: secret verify failed", plan.req.Adapter.Name())
	}
	out.To = after.SecretFingerprint
	out.Copied = true
	return out, nil
}

// CopySecret transfers a secret or env-ref from src to dst. Values are never returned.
func CopySecret(ad adapters.Adapter, srcFS fsx.FS, srcHome string, dstFS fsx.FS, dstHome string, preferRef bool) (model.SecretCopy, error) {
	plan, err := PreflightSecret(SecretRequest{
		Adapter: ad, SrcFS: srcFS, SrcHome: srcHome, DstFS: dstFS, DstHome: dstHome, PreferRef: preferRef,
	})
	if err != nil {
		return plan.out, err
	}
	return ApplyPreparedSecret(plan)
}
