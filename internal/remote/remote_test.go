package remote

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
)

func TestDefaultDialHomeFlagUsesLocalFS(t *testing.T) {
	home := t.TempDir()
	got, err := DefaultDial(config.NamedContext{
		Name:    "box",
		Context: config.Context{Kind: config.KindSSH, SSH: "user@box"},
	}, home)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.FS.(fsx.Local); !ok {
		t.Fatalf("FS type %T", got.FS)
	}
	if got.Home != home || got.Name != "box" {
		t.Fatalf("%#v", got)
	}
}

func TestDefaultDialLocalHomeAndTilde(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	got, err := DefaultDial(config.NamedContext{
		Name:    "mba",
		Context: config.Context{Kind: config.KindLocal, Home: "~/agent"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "agent")
	if got.Home != want {
		t.Fatalf("home=%q want %q", got.Home, want)
	}
}

func TestDefaultDialInfersSSHKind(t *testing.T) {
	t.Setenv("HCTL_SSH", "0")
	t.Setenv("HARNESSCTL_SSH", "")
	_, err := DefaultDial(config.NamedContext{
		Name:    "box",
		Context: config.Context{SSH: "user@box"},
	}, "")
	if exitcode.From(err) != exitcode.SSH {
		t.Fatalf("err=%v code=%d", err, exitcode.From(err))
	}
	if !strings.Contains(err.Error(), "ssh disabled") {
		t.Fatalf("%v", err)
	}
}

func TestDefaultDialEmptySSHTarget(t *testing.T) {
	t.Setenv("HCTL_SSH", "")
	t.Setenv("HARNESSCTL_SSH", "")
	_, err := DefaultDial(config.NamedContext{
		Name:    "box",
		Context: config.Context{Kind: config.KindSSH},
	}, "")
	if exitcode.From(err) != exitcode.SSH {
		t.Fatalf("err=%v code=%d", err, exitcode.From(err))
	}
	if !strings.Contains(err.Error(), "ssh target is empty") {
		t.Fatalf("%v", err)
	}
}

func TestExpandFailsClosedWhenHomeUnknown(t *testing.T) {
	old := userHomeDir
	userHomeDir = func() (string, error) { return "", errors.New("no home") }
	t.Cleanup(func() { userHomeDir = old })
	_, err := DefaultDial(config.NamedContext{
		Name:    "mba",
		Context: config.Context{Kind: config.KindLocal, Home: "~/x"},
	}, "")
	if err == nil {
		t.Fatal("expected expand error")
	}
}

func TestDefaultDialLocalUsesUserHome(t *testing.T) {
	want, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	got, err := DefaultDial(config.NamedContext{
		Name:    "mba",
		Context: config.Context{Kind: config.KindLocal},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Home != want {
		t.Fatalf("home=%q want %q", got.Home, want)
	}
}
