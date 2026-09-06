package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigHelpers(t *testing.T) {
	c := Context{SSH: "u@h"}
	if c.Target() != "u@h" {
		t.Fatal(c.Target())
	}
	c2 := Context{User: "u", Host: "h"}
	if c2.Target() != "u@h" {
		t.Fatal(c2.Target())
	}
	if (Context{Host: "h"}).Target() != "h" {
		t.Fatal("host")
	}
	t.Setenv("HARNESSCTL_CONFIG", "/tmp/custom-hctl.yaml")
	if DefaultPath() != "/tmp/custom-hctl.yaml" {
		t.Fatal(DefaultPath())
	}
	t.Setenv("HARNESSCTL_CONFIG", "")
	_ = DefaultPath()
	f := Default()
	cur, err := f.Current()
	if err != nil || cur.Name != "mba" {
		t.Fatal(err, cur)
	}
	path := filepath.Join(t.TempDir(), "c.yaml")
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	_, home, err := loaded.ResolveHome("mba", "/tmp/home")
	if err != nil || home != "/tmp/home" {
		t.Fatal(err, home)
	}
	f.Contexts = append(f.Contexts, NamedContext{Name: "box", Context: Context{Kind: KindSSH, Home: "/remote"}})
	_, home, err = f.ResolveHome("box", "")
	if err != nil || home != "/remote" {
		t.Fatal(err, home)
	}
	f.Contexts[1].Context.Home = ""
	_, _, err = f.ResolveHome("box", "")
	if err == nil {
		t.Fatal("expected ssh not implemented")
	}
	f.Contexts = append(f.Contexts, NamedContext{Name: "loc", Context: Context{Kind: KindLocal, Home: "~/x"}})
	_, home, err = f.ResolveHome("loc", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = home
	_ = expandHome("/abs")
	_ = expandHome("~/rel")
	empty := &File{}
	if _, err := empty.Current(); err == nil {
		t.Fatal("empty current")
	}
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	d, err := Load(missing)
	if err != nil || d.CurrentContext == "" {
		t.Fatal(err, d)
	}
	bad := filepath.Join(t.TempDir(), "bad.yaml")
	os.WriteFile(bad, []byte(":\n:"), 0o600)
	if _, err := Load(bad); err == nil {
		t.Fatal("expected parse err")
	}
}
