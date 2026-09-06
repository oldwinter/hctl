package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEdges(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	os.WriteFile(p, []byte("apiVersion: x\nkind: y\ncontexts: []\n"), 0o600)
	f, err := Load(p)
	if err != nil || f.CurrentContext != "mba" {
		t.Fatal(err, f)
	}
	os.WriteFile(p, []byte("contexts:\n  - name: s\n    context:\n      ssh: u@h\n  - name: l\n    context: {}\n"), 0o600)
	f, err = Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if f.Contexts[0].Context.Kind != KindSSH || f.Contexts[1].Context.Kind != KindLocal {
		t.Fatal(f.Contexts)
	}
	if f.CurrentContext != "s" {
		t.Fatal(f.CurrentContext)
	}
	// ResolveHome local without home uses UserHomeDir
	_, home, err := f.ResolveHome("l", "")
	if err != nil || home == "" {
		t.Fatal(err, home)
	}
}
