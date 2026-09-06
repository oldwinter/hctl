package adapters

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

type boomFS struct{ fsx.Local }

func (boomFS) ReadFile(name string) ([]byte, error) { return nil, errors.New("boom") }

func TestScanFSReadError(t *testing.T) {
	if _, err := ScanFS(boomFS{}, t.TempDir()); err == nil {
		t.Fatal("expected")
	}
}

type bare2 struct{}

func (bare2) Name() string                             { return "bare2" }
func (bare2) Aliases() []string                        { return nil }
func (bare2) BinaryNames() []string                    { return nil }
func (bare2) ConfigRelPaths() []string                 { return []string{".bare2/x"} }
func (bare2) Read(home string) (model.Snapshot, error) { return model.Snapshot{}, nil }

func TestReadOneFSConfigFoundViaExists(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".bare2"), 0o755)
	os.WriteFile(filepath.Join(home, ".bare2", "x"), []byte("{}"), 0o600)
	snap, err := ReadOneFS(bare2{}, fsx.Local{}, home)
	if err != nil || !snap.ConfigFound {
		t.Fatalf("%#v %v", snap, err)
	}
}
