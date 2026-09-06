package cursor

import (
	"errors"
	"os"
	"testing"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestProbeLoginNoBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	ok, note := probeLogin()
	if ok || note != "" {
		t.Fatalf("%v %q", ok, note)
	}
}

func TestWriteFieldsAtomicFail(t *testing.T) {
	prev := fsx.AtomicWriteHook
	t.Cleanup(func() { fsx.AtomicWriteHook = prev })
	fsx.AtomicWriteHook = func(fsys fsx.FS, path string, data []byte, mode os.FileMode) error {
		return errors.New("aw")
	}
	home := t.TempDir()
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{Model: "m"}); err == nil {
		t.Fatal("expected")
	}
	_ = edit.Format("")
}

func TestWriteFieldsProviderSetFail(t *testing.T) {
	prev := edit.SetJSONHook
	t.Cleanup(func() { edit.SetJSONHook = prev })
	edit.SetJSONHook = func(src []byte, path []string, value string) ([]byte, error) {
		if len(path) > 0 && path[0] == "provider" {
			return nil, errors.New("set provider")
		}
		return prev(src, path, value)
	}
	home := t.TempDir()
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{Provider: "p"}); err == nil {
		t.Fatal("expected")
	}
}
