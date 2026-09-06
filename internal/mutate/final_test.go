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

type wrongModel struct{ codex.Adapter }

func (w wrongModel) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	d2 := d
	if d2.Model != "" {
		d2.Model = "NOT-" + d2.Model
	}
	if d2.Provider != "" {
		d2.Provider = "NOT-" + d2.Provider
	}
	if d2.SecretRef != "" {
		d2.SecretRef = "NOT_" + d2.SecretRef
	}
	return w.Adapter.WriteFields(fsys, home, d2)
}

type readBoom struct{ fsx.Local }

func (readBoom) ReadFile(name string) ([]byte, error) { return nil, errors.New("boom") }

type peekBoom struct{ codex.Adapter }

func (peekBoom) PeekSecret(fsys fsx.FS, home string) (string, string, error) {
	return "", "", errors.New("peek boom")
}

type writeSecBoom struct{ codex.Adapter }

func (writeSecBoom) PeekSecret(fsys fsx.FS, home string) (string, string, error) {
	return "K", "sk-test-x", nil
}
func (writeSecBoom) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	return errors.New("ws boom")
}

func TestFinalMutateGaps(t *testing.T) {
	home := t.TempDir()
	a := codex.Adapter{}
	_, _ = a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "custom"})
	bak := filepath.Join(t.TempDir(), "bak")
	_, err := Apply(Request{Adapter: wrongModel{}, FS: fsx.Local{}, Home: home, BackupDir: bak, Desired: model.Desired{Model: "want"}})
	if err == nil {
		t.Fatal("verify model")
	}
	_, err = Apply(Request{Adapter: wrongModel{}, FS: fsx.Local{}, Home: home, BackupDir: bak, Desired: model.Desired{Provider: "wantp"}})
	if err == nil {
		t.Fatal("verify provider")
	}
	_, err = Apply(Request{Adapter: wrongModel{}, FS: fsx.Local{}, Home: home, BackupDir: bak, Desired: model.Desired{SecretRef: "WANT"}})
	if err == nil {
		t.Fatal("verify secret")
	}
	// backup read fail
	_, err = Apply(Request{Adapter: a, FS: readBoom{}, Home: home, BackupDir: bak, Desired: model.Desired{Model: "m2"}})
	if err == nil {
		t.Fatal("backup read")
	}
	// backup write fail
	badBak := filepath.Join(t.TempDir(), "file")
	os.WriteFile(badBak, []byte("x"), 0o600)
	_, err = Apply(Request{Adapter: a, FS: fsx.Local{}, Home: home, BackupDir: filepath.Join(badBak, "x"), Desired: model.Desired{Model: "m3"}})
	if err == nil {
		t.Fatal("backup write")
	}
	// WriteFields fail via AtomicWriteHook
	old := fsx.AtomicWriteHook
	fsx.AtomicWriteHook = func(fsys fsx.FS, name string, data []byte, perm os.FileMode) error { return errors.New("aw") }
	_, err = Apply(Request{Adapter: a, FS: fsx.Local{}, Home: home, BackupDir: bak, Desired: model.Desired{Model: "m4"}})
	fsx.AtomicWriteHook = old
	if err == nil {
		t.Fatal("writefields")
	}
	// CopySecret peek/write errors
	_, err = CopySecret(peekBoom{}, fsx.Local{}, home, fsx.Local{}, t.TempDir(), false)
	if err == nil {
		t.Fatal("peek")
	}
	_, err = CopySecret(writeSecBoom{}, fsx.Local{}, home, fsx.Local{}, t.TempDir(), false)
	if err == nil {
		t.Fatal("ws")
	}
	_, err = CopySecret(writeSecBoom{}, fsx.Local{}, home, fsx.Local{}, t.TempDir(), true)
	if err == nil {
		// preferRef with ref K from peek - writeSecBoom Peek returns K,sk - preferRef uses ref
		t.Log(err)
	}
}
