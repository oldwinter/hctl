package mutate

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

type boomFS2 struct {
	fsx.Local
	failAfterWrite bool
	nRead          int
}

func (b *boomFS2) ReadFile(name string) ([]byte, error) {
	b.nRead++
	if b.failAfterWrite && b.nRead > 3 {
		return nil, errors.New("reread boom")
	}
	return b.Local.ReadFile(name)
}

func TestApplyVerifyAndCopyGaps(t *testing.T) {
	a := codex.Adapter{}
	home := t.TempDir()
	_, _ = a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "custom"})
	bak := filepath.Join(t.TempDir(), "bak")
	// provider change ok
	_, err := Apply(Request{Adapter: a, FS: fsx.Local{}, Home: home, BackupDir: bak, Desired: model.Desired{Provider: "custom2"}})
	if err != nil {
		t.Fatal(err)
	}
	// secretref
	_, err = Apply(Request{Adapter: a, FS: fsx.Local{}, Home: home, BackupDir: bak, Desired: model.Desired{SecretRef: "K"}})
	if err != nil {
		t.Fatal(err)
	}
	// CopySecret peek error
	fs := &boomFS2{}
	fs.failAfterWrite = false
	// force peek fail with failRead on first read of peek - use boom from err_test
	type rf struct {
		fsx.Local
	}
	// WriteSecret only ref path in CopySecret when val empty and ref set — already
	src := t.TempDir()
	_, _ = a.WriteFields(fsx.Local{}, src, model.Desired{Model: "m", Provider: "custom", SecretRef: "ONLYREF"})
	// peek returns ref with empty value for env_key only configs
	sc, err := CopySecret(a, fsx.Local{}, src, fsx.Local{}, t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	_ = sc
}
