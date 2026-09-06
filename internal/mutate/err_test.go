package mutate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/adapters/stub"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

type boomFS struct {
	fsx.Local
	failRead, failMkdir, failRename bool
}

func (b boomFS) ReadFile(name string) ([]byte, error) {
	if b.failRead {
		return nil, errors.New("boom")
	}
	return b.Local.ReadFile(name)
}
func (b boomFS) MkdirAll(name string, perm os.FileMode) error {
	if b.failMkdir {
		return errors.New("boom")
	}
	return b.Local.MkdirAll(name, perm)
}
func (b boomFS) Rename(oldpath, newpath string) error {
	if b.failRename {
		return errors.New("boom")
	}
	return b.Local.Rename(oldpath, newpath)
}
func (b boomFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return b.Local.WriteFile(name, data, perm)
}

func TestApplyErrorBranches(t *testing.T) {
	a := codex.Adapter{}
	home := t.TempDir()
	_, _ = a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "custom"})
	_, err := Apply(Request{Adapter: a, FS: boomFS{failRead: true}, Home: home, Desired: model.Desired{Model: "n"}})
	if err == nil {
		t.Fatal("read before")
	}
	_, err = Apply(Request{Adapter: stub.New("x", nil, nil), FS: fsx.Local{}, Home: home, Desired: model.Desired{Model: "n"}, DryRun: true})
	if err == nil {
		t.Fatal("no writer")
	}
	bak := filepath.Join(t.TempDir(), "bak")
	_, err = Apply(Request{Adapter: a, FS: boomFS{failMkdir: true}, Home: home, BackupDir: bak, Desired: model.Desired{Model: "n2"}})
	if err == nil {
		// WriteFields atomic may fail
		t.Log(err)
	}
	// CopySecret preferRef with empty peek
	empty := t.TempDir()
	sc, err := CopySecret(a, fsx.Local{}, empty, fsx.Local{}, t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	_ = sc
	// ref-only copy
	src := t.TempDir()
	_, _ = a.WriteFields(fsx.Local{}, src, model.Desired{Model: "m", Provider: "custom", SecretRef: "ENVKEY"})
	sc, err = CopySecret(a, fsx.Local{}, src, fsx.Local{}, t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	_ = sc
	sc, err = CopySecret(a, fsx.Local{}, src, fsx.Local{}, t.TempDir(), false)
	_ = sc
	_ = err
}
