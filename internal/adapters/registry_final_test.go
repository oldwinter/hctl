package adapters

import (
	"errors"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

type errAd struct{}

func (errAd) Name() string          { return "errad" }
func (errAd) Aliases() []string     { return nil }
func (errAd) BinaryNames() []string { return nil }
func (errAd) ConfigRelPaths() []string {
	return nil
}
func (errAd) Read(home string) (model.Snapshot, error) { return model.Snapshot{}, errors.New("r") }
func (errAd) ReadFS(fsys fsx.FS, home string) (model.Snapshot, error) {
	return model.Snapshot{}, errors.New("rfs")
}

func TestReadOneFSError(t *testing.T) {
	if _, err := ReadOneFS(errAd{}, fsx.Local{}, t.TempDir()); err == nil {
		t.Fatal("expected")
	}
}
