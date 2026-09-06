package remote

import (
	"testing"

	"github.com/oldwinter/hctl/internal/config"
)

func TestDefaultDialUserHomeAndSSHHome(t *testing.T) {
	fsys, home, err := DefaultDial(config.NamedContext{Name: "x", Context: config.Context{Kind: config.KindLocal}}, "")
	if err != nil || home == "" {
		t.Fatal(err, home)
	}
	_ = fsys
	t.Setenv("HARNESSCTL_SSH", "")
	fsys, home, err = DefaultDial(config.NamedContext{
		Name:    "ssh",
		Context: config.Context{Kind: config.KindSSH, User: "u", Host: "h", Home: t.TempDir()},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	_ = fsys
	_ = home
}
