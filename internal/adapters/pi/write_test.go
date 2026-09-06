package pi

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
	_, err := a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.WriteSecret(fsx.Local{}, home, "", "sk-test-pi"); err != nil {
		t.Fatal(err)
	}
	ref, val, err := a.PeekSecret(fsx.Local{}, home)
	if err != nil || val != "sk-test-pi" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	if err := a.WriteSecret(fsx.Local{}, home, "$PI_KEY", ""); err != nil {
		t.Fatal(err)
	}
	ref, _, err = a.PeekSecret(fsx.Local{}, home)
	if err != nil || ref == "" {
		t.Fatal(ref, err)
	}
	empty := t.TempDir()
	_, _, _ = a.PeekSecret(fsx.Local{}, empty)
	bad := t.TempDir()
	os.MkdirAll(filepath.Join(bad, ".pi", "agent"), 0o755)
	os.WriteFile(filepath.Join(bad, ".pi", "agent", "auth.json"), []byte("{"), 0o600)
	if _, _, err := a.PeekSecret(fsx.Local{}, bad); err == nil {
		t.Fatal("expected")
	}
	// auth only found
	home2 := t.TempDir()
	os.MkdirAll(filepath.Join(home2, ".pi", "agent"), 0o755)
	os.WriteFile(filepath.Join(home2, ".pi", "agent", "auth.json"), []byte(`{"accessToken":"sk-test-tok"}`), 0o600)
	snap, err := a.ReadFS(fsx.Local{}, home2)
	if err != nil || !snap.ConfigFound {
		t.Fatalf("%#v %v", snap, err)
	}
	_, val, err = a.PeekSecret(fsx.Local{}, home2)
	if err != nil || val != "sk-test-tok" {
		t.Fatalf("%q %v", val, err)
	}
}
