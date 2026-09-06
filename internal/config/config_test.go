package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingReturnsDefaultMBA(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if f.CurrentContext != "mba" {
		t.Fatalf("current = %q", f.CurrentContext)
	}
	if _, err := f.Get("mba"); err != nil {
		t.Fatal(err)
	}
}

func TestUseContextAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	f := Default()
	f.Contexts = append(f.Contexts, NamedContext{
		Name:    "box",
		Context: Context{Kind: KindSSH, SSH: "user@box", Home: "/home/user"},
	})
	if err := f.UseContext("box"); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %o", st.Mode().Perm())
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentContext != "box" {
		t.Fatalf("current = %q", got.CurrentContext)
	}
}

func TestResolveHomeSSHRequiresOverride(t *testing.T) {
	f := Default()
	f.Contexts = append(f.Contexts, NamedContext{
		Name:    "box",
		Context: Context{Kind: KindSSH, SSH: "user@box"},
	})
	if _, _, err := f.ResolveHome("box", ""); err == nil {
		t.Fatal("expected ssh not implemented")
	}
	_, home, err := f.ResolveHome("box", "/tmp/fixture")
	if err != nil {
		t.Fatal(err)
	}
	if home != "/tmp/fixture" {
		t.Fatalf("home = %q", home)
	}
}

func TestUnknownContext(t *testing.T) {
	if err := Default().UseContext("nope"); err == nil {
		t.Fatal("expected error")
	}
}
