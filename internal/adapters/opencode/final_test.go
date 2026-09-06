package opencode

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

type readBoom struct{ fsx.Local }

func (readBoom) ReadFile(name string) ([]byte, error) { return nil, errors.New("read boom") }

func TestFinalGaps(t *testing.T) {
	a := Adapter{}
	home := t.TempDir()
	if _, err := a.WriteFields(readBoom{}, home, model.Desired{Model: "openai/m"}); err == nil {
		t.Fatal("read")
	}
	if err := a.WriteSecret(readBoom{}, home, "", "sk-test-x"); err == nil {
		t.Fatal("ws")
	}
	// empty provider nodes / options
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{"model":"x/y","provider":{"x":{}}}`), 0o600)
	_, _, _ = a.PeekSecret(fsx.Local{}, home)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{"model":"x/y","provider":{"x":{"options":{}}}}`), 0o600)
	_, _, _ = a.PeekSecret(fsx.Local{}, home)
	old := fsx.AtomicWriteHook
	defer func() { fsx.AtomicWriteHook = old }()
	fsx.AtomicWriteHook = func(fsys fsx.FS, name string, data []byte, perm os.FileMode) error {
		return errors.New("aw")
	}
	if _, err := a.WriteFields(fsx.Local{}, t.TempDir(), model.Desired{Model: "openai/m"}); err == nil {
		t.Fatal("aw")
	}
}
