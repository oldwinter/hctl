package cursor

import (
	"errors"
	"os"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestAtomicWriteFail(t *testing.T) {
	old := fsx.AtomicWriteHook
	defer func() { fsx.AtomicWriteHook = old }()
	fsx.AtomicWriteHook = func(fsys fsx.FS, name string, data []byte, perm os.FileMode) error {
		return errors.New("aw")
	}
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, t.TempDir(), model.Desired{Model: "m"}); err == nil {
		t.Fatal("expected")
	}
}
