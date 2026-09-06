package remote

import (
	"errors"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/fsx"
)

func TestExpandAndSSHRemoteHome(t *testing.T) {
	old := userHomeDir
	defer func() { userHomeDir = old }()
	userHomeDir = func() (string, error) { return "", errors.New("no") }
	if expand("~/x") != "~/x" {
		t.Fatal(expand("~/x"))
	}
	_, _, err := DefaultDial(config.NamedContext{Name: "l", Context: config.Context{Kind: config.KindLocal}}, "")
	if err == nil {
		t.Fatal("expected home err")
	}
	// SSH with empty home triggers RemoteHome via real defaultRunner — avoid hang by setting home
	// Cover lines 45/48: empty target already covered; RemoteHome fail:
	t.Setenv("HARNESSCTL_SSH", "")
	// inject by dialing with home set is covered; for RemoteHome path use SSH with Run
	s := fsx.SSH{Target: "u@h", Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
		return nil, errors.New("ssh fail")
	}}
	_, err = s.RemoteHome()
	if err == nil {
		t.Fatal("remote home")
	}
	_ = s
}
