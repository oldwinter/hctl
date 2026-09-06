package mutate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestBackupLocalFailInApply(t *testing.T) {
	home := t.TempDir()
	a := codex.Adapter{}
	_, _ = a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "custom"})
	block := filepath.Join(t.TempDir(), "f")
	os.WriteFile(block, []byte("x"), 0o600)
	_, err := Apply(Request{Adapter: a, FS: fsx.Local{}, Home: home, BackupDir: filepath.Join(block, "sub"), Desired: model.Desired{Model: "m2"}})
	if err == nil {
		t.Fatal("expected backup fail")
	}
}
