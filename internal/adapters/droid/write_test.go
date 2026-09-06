package droid

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestMetaWritePeekSecret(t *testing.T) {
	a := Adapter{}
	_ = a.Aliases()
	_ = a.BinaryNames()
	_ = a.ConfigRelPaths()
	home := t.TempDir()
	paths, err := a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "p"})
	if err != nil || len(paths) != 1 {
		t.Fatal(err)
	}
	if err := a.WriteSecret(fsx.Local{}, home, "", "sk-test-droid"); err != nil {
		t.Fatal(err)
	}
	ref, val, err := a.PeekSecret(fsx.Local{}, home)
	if err != nil || val != "sk-test-droid" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	if err := a.WriteSecret(fsx.Local{}, home, "$DROID_KEY", ""); err != nil {
		t.Fatal(err)
	}
	ref, val, err = a.PeekSecret(fsx.Local{}, home)
	if err != nil || ref == "" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	empty := t.TempDir()
	_, _, err = a.PeekSecret(fsx.Local{}, empty)
	if err != nil {
		t.Fatal(err)
	}
	bad := t.TempDir()
	os.MkdirAll(filepath.Join(bad, ".factory"), 0o755)
	os.WriteFile(filepath.Join(bad, ".factory", "settings.json"), []byte("{"), 0o600)
	if _, _, err := a.PeekSecret(fsx.Local{}, bad); err == nil {
		t.Fatal("expected")
	}
	// Read parse error / missing
	snap, _ := a.Read(t.TempDir())
	if snap.ConfigFound {
		t.Fatal("missing")
	}
	home2 := t.TempDir()
	os.MkdirAll(filepath.Join(home2, ".factory"), 0o755)
	os.WriteFile(filepath.Join(home2, ".factory", "settings.json"), []byte(`{"model":"m","provider":"p","baseURL":"https://x.test","apiKey":"sk-test-x"}`), 0o600)
	snap, err = a.ReadFS(fsx.Local{}, home2)
	if err != nil || snap.SecretPresent != true {
		t.Fatalf("%#v %v", snap, err)
	}
}
