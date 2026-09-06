package cursor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestMetaWriteFields(t *testing.T) {
	a := Adapter{}
	_ = a.Aliases()
	_ = a.BinaryNames()
	_ = a.ConfigRelPaths()
	home := t.TempDir()
	paths, err := a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "p"})
	if err != nil || len(paths) != 1 {
		t.Fatal(err)
	}
	snap, err := a.ReadFS(fsx.Local{}, home)
	if err != nil || snap.DefaultModel != "m" {
		t.Fatalf("%#v %v", snap, err)
	}
	bad := t.TempDir()
	os.MkdirAll(filepath.Join(bad, ".cursor"), 0o755)
	os.WriteFile(filepath.Join(bad, ".cursor", "cli-config.json"), []byte("{"), 0o600)
	snap, err = a.ReadFS(fsx.Local{}, bad)
	if err != nil || snap.ParseError == "" {
		t.Fatalf("%#v %v", snap, err)
	}
	snap, _ = a.Read(t.TempDir())
	if snap.ConfigFound {
		t.Fatal("missing")
	}
}
