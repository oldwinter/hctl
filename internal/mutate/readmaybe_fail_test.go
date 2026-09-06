package mutate

import (
	"errors"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

type onlyRead struct{}

func (onlyRead) Name() string             { return "onlyread" }
func (onlyRead) Aliases() []string        { return nil }
func (onlyRead) BinaryNames() []string    { return nil }
func (onlyRead) ConfigRelPaths() []string { return nil }
func (onlyRead) Read(home string) (model.Snapshot, error) {
	return model.Snapshot{Name: "onlyread", ConfigPaths: []string{"blocked"}}, nil
}
func (onlyRead) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	return nil, nil
}

type errFS struct{ fsx.Local }

func (errFS) ReadFile(string) ([]byte, error) { return nil, errors.New("read boom") }

func TestApplyReadMaybeFail(t *testing.T) {
	_, err := Apply(Request{
		Adapter:   onlyRead{},
		FS:        errFS{},
		Home:      t.TempDir(),
		Desired:   model.Desired{Model: "m"},
		BackupDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected")
	}
}
