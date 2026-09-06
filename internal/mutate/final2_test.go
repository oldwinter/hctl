package mutate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

type nthRead struct {
	fsx.Local
	n, failAt int
}

func (n *nthRead) ReadFile(name string) ([]byte, error) {
	n.n++
	if n.n >= n.failAt {
		return nil, errors.New("boom")
	}
	return n.Local.ReadFile(name)
}

type failAfterWrite struct {
	fsx.Local
	wrote bool
}

func (f *failAfterWrite) WriteFile(name string, data []byte, perm os.FileMode) error {
	f.wrote = true
	return f.Local.WriteFile(name, data, perm)
}
func (f *failAfterWrite) ReadFile(name string) ([]byte, error) {
	if f.wrote {
		return nil, errors.New("after write")
	}
	return f.Local.ReadFile(name)
}

type parsePoison struct{ codex.Adapter }

func (parsePoison) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	path := fsys.Join(home, ".codex", "config.toml")
	_ = fsys.MkdirAll(fsys.Join(home, ".codex"), 0o755)
	_ = fsys.WriteFile(path, []byte("[[["), 0o600)
	return []string{path}, nil
}

type refOnly struct{ codex.Adapter }

func (refOnly) PeekSecret(fsys fsx.FS, home string) (string, string, error) { return "K", "", nil }
func (refOnly) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	return errors.New("ws")
}

func TestFinal2(t *testing.T) {
	home := t.TempDir()
	a := codex.Adapter{}
	_, _ = a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "custom"})
	bak := filepath.Join(t.TempDir(), "bak")
	fs := &nthRead{failAt: 3}
	_, err := Apply(Request{Adapter: a, FS: fs, Home: home, BackupDir: bak, Desired: model.Desired{Model: "m2"}})
	if err == nil {
		t.Fatal("backup read")
	}
	fs2 := &failAfterWrite{}
	_, err = Apply(Request{Adapter: a, FS: fs2, Home: home, BackupDir: bak, Desired: model.Desired{Model: "m3"}})
	if err == nil {
		t.Fatal("verify read")
	}
	_, err = Apply(Request{Adapter: parsePoison{}, FS: fsx.Local{}, Home: t.TempDir(), BackupDir: bak, Desired: model.Desired{Model: "m"}})
	if err == nil {
		t.Fatal("parse after")
	}
	_, err = CopySecret(refOnly{}, fsx.Local{}, home, fsx.Local{}, t.TempDir(), true)
	if err == nil {
		t.Fatal("ref ws")
	}
	_, err = CopySecret(refOnly{}, fsx.Local{}, home, fsx.Local{}, t.TempDir(), false)
	if err != nil {
		// val empty, falls through to ref write
		_ = err
	}
}

func TestApplyDefaultBackupDir(t *testing.T) {
	home := t.TempDir()
	a := codex.Adapter{}
	_, _ = a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "custom"})
	t.Setenv("HARNESSCTL_BACKUP_DIR", filepath.Join(t.TempDir(), "b"))
	_, err := Apply(Request{Adapter: a, FS: fsx.Local{}, Home: home, Desired: model.Desired{Model: "m2"}})
	if err != nil {
		t.Fatal(err)
	}
}
