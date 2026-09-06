package claude

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
	if _, err := a.WriteFields(readBoom{}, home, model.Desired{Model: "m"}); err == nil {
		t.Fatal("read")
	}
	if err := a.WriteSecret(readBoom{}, home, "", "sk-test-x"); err == nil {
		t.Fatal("ws read")
	}
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{"env":{"OTHER":"x"}}`), 0o600)
	ref, val, err := a.PeekSecret(fsx.Local{}, home)
	if err != nil || ref != "" || val != "" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	old := fsx.AtomicWriteHook
	defer func() { fsx.AtomicWriteHook = old }()
	fsx.AtomicWriteHook = func(fsys fsx.FS, name string, data []byte, perm os.FileMode) error {
		return errors.New("aw")
	}
	if _, err := a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m2"}); err == nil {
		t.Fatal("aw")
	}
}
