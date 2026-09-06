package mutate

import (
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/adapters/claude"
	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/adapters/stub"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestDefaultBackupCopySecretApply(t *testing.T) {
	t.Setenv("HARNESSCTL_BACKUP_DIR", filepath.Join(t.TempDir(), "b"))
	if DefaultBackupDir("") == "" {
		t.Fatal("backup")
	}
	t.Setenv("HARNESSCTL_BACKUP_DIR", "")
	_ = DefaultBackupDir("/cfg/config.yaml")
	_ = DefaultBackupDir("")
	if err := rejectUnsupportedProvider("claude"); err == nil {
		t.Fatal("claude")
	}
	if err := rejectUnsupportedProvider("grok"); err == nil {
		t.Fatal("grok")
	}
	if err := rejectUnsupportedProvider("codex"); err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	dst := t.TempDir()
	a := codex.Adapter{}
	_, err := a.WriteFields(fsx.Local{}, src, model.Desired{Model: "m", Provider: "custom"})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.WriteSecret(fsx.Local{}, src, "K", "sk-test-copy"); err != nil {
		t.Fatal(err)
	}
	sc, err := CopySecret(a, fsx.Local{}, src, fsx.Local{}, dst, false)
	if err != nil || !sc.Copied {
		t.Fatalf("%#v %v", sc, err)
	}
	sc, err = CopySecret(a, fsx.Local{}, src, fsx.Local{}, dst, true)
	if err != nil {
		t.Fatal(err)
	}
	_ = sc
	_, err = CopySecret(stub.New("x", nil, nil), fsx.Local{}, src, fsx.Local{}, dst, false)
	if err == nil {
		t.Fatal("stub")
	}
	// empty apply
	rep, err := Apply(Request{Adapter: a, FS: fsx.Local{}, Home: src, Desired: model.Desired{}})
	if err != nil || len(rep.Changes) != 0 {
		t.Fatal(err, rep)
	}
	// dry-run
	rep, err = Apply(Request{Adapter: a, FS: fsx.Local{}, Home: src, Desired: model.Desired{Model: "m2"}, DryRun: true})
	if err != nil || !rep.DryRun {
		t.Fatal(err, rep)
	}
	// real apply
	bak := filepath.Join(t.TempDir(), "bak")
	rep, err = Apply(Request{Adapter: a, FS: fsx.Local{}, Home: src, BackupDir: bak, Desired: model.Desired{Model: "m3", Provider: "custom"}})
	if err != nil || !rep.Verified {
		t.Fatalf("%#v %v", rep, err)
	}
	// claude provider reject via Apply
	chome := t.TempDir()
	_, err = Apply(Request{Adapter: claude.Adapter{}, FS: fsx.Local{}, Home: chome, Desired: model.Desired{Provider: "x"}, DryRun: true})
	if err == nil {
		t.Fatal("expected")
	}
}
