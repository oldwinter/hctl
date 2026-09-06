package stub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
)

func TestStubAdapter(t *testing.T) {
	a := New("foo", []string{"foobin"}, []string{".foo/config.json"})
	if a.Name() != "foo" {
		t.Fatal(a.Name())
	}
	if a.Aliases() != nil {
		t.Fatal(a.Aliases())
	}
	if got := a.BinaryNames(); len(got) != 1 || got[0] != "foobin" {
		t.Fatalf("%v", got)
	}
	if got := a.ConfigRelPaths(); len(got) != 1 || got[0] != ".foo/config.json" {
		t.Fatalf("%v", got)
	}
	home := t.TempDir()
	snap, err := a.Read(home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.ConfigFound {
		t.Fatal("expected missing")
	}
	if err := os.MkdirAll(filepath.Join(home, ".foo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".foo", "config.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err = a.ReadFS(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.ConfigFound {
		t.Fatal("expected found")
	}
	found := false
	for _, n := range snap.Notes {
		if strings.Contains(n, "dedicated read adapter") {
			found = true
		}
	}
	if !found {
		t.Fatalf("%v", snap.Notes)
	}
}
