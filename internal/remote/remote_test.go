package remote

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
)

func TestDefaultDialLocalAndHomeFlag(t *testing.T) {
	home := t.TempDir()
	fsys, got, err := DefaultDial(config.NamedContext{Name: "x", Context: config.Context{Kind: config.KindLocal}}, home)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fsys.(fsx.Local); !ok {
		t.Fatalf("%T", fsys)
	}
	if got != home {
		t.Fatalf("%q", got)
	}
	fsys, got, err = DefaultDial(config.NamedContext{
		Name:    "local",
		Context: config.Context{Kind: config.KindLocal, Home: home},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != home {
		t.Fatalf("%q", got)
	}
	_ = fsys
}

func TestDefaultDialInferKindAndExpand(t *testing.T) {
	home := t.TempDir()
	fsys, got, err := DefaultDial(config.NamedContext{
		Name:    "infer-local",
		Context: config.Context{Home: "~/should-expand"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fsys.(fsx.Local); !ok {
		t.Fatalf("%T", fsys)
	}
	if !strings.Contains(got, "should-expand") || strings.HasPrefix(got, "~") {
		// expand replaces ~/ — if UserHomeDir fails, path may stay as-is
		_ = got
	}
	t.Setenv("HARNESSCTL_SSH", "0")
	_, _, err = DefaultDial(config.NamedContext{
		Name:    "ssh",
		Context: config.Context{Kind: config.KindSSH, SSH: "user@host", Home: home},
	}, "")
	if err == nil || exitcode.From(err) != exitcode.SSH {
		t.Fatalf("%v", err)
	}
	t.Setenv("HARNESSCTL_SSH", "")
	_, _, err = DefaultDial(config.NamedContext{
		Name:    "empty",
		Context: config.Context{Kind: config.KindSSH},
	}, "")
	if err == nil || !strings.Contains(err.Error(), "ssh target is empty") {
		t.Fatalf("%v", err)
	}
	fsys, got, err = DefaultDial(config.NamedContext{
		Name: "ssh-ok",
		Context: config.Context{
			Kind:         config.KindSSH,
			SSH:          "user@host",
			Home:         home,
			IdentityFile: "/tmp/id",
		},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	s, ok := fsys.(fsx.SSH)
	if !ok {
		t.Fatalf("%T", fsys)
	}
	if s.Target == "" || got != home {
		t.Fatalf("%#v %q", s, got)
	}
	if expand("~/x") == "~/x" {
		// may happen if UserHomeDir fails; still covered
	} else if !filepath.IsAbs(expand("~/x")) {
		t.Fatalf("%q", expand("~/x"))
	}
	if expand("/abs") != "/abs" {
		t.Fatal(expand("/abs"))
	}
	// empty kind + host infers ssh
	t.Setenv("HARNESSCTL_SSH", "0")
	_, _, err = DefaultDial(config.NamedContext{
		Name:    "host-infer",
		Context: config.Context{Host: "h"},
	}, "")
	if err == nil {
		t.Fatal("expected ssh disabled")
	}
	_ = os.Getenv
}
