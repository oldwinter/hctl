package remote

import (
	"fmt"
	"os"
	"strings"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
)

var userHomeDir = os.UserHomeDir

// Target is one opened context: a filesystem rooted at Home.
type Target struct {
	Name string
	FS   fsx.FS
	Home string
}

// Dial opens a filesystem for a context. Tests may replace this.
var Dial = DefaultDial

// DefaultDial is the only home resolver. --home / HCTL_HOME wins and
// allows fixture tests even for ssh.
func DefaultDial(nc config.NamedContext, homeFlag string) (Target, error) {
	if homeFlag != "" {
		return Target{Name: nc.Name, FS: fsx.Local{}, Home: homeFlag}, nil
	}
	ctx := nc.Context
	if ctx.Kind == "" {
		if ctx.SSH != "" || ctx.Host != "" {
			ctx.Kind = config.KindSSH
		} else {
			ctx.Kind = config.KindLocal
		}
	}
	if ctx.Kind != config.KindSSH {
		if ctx.Home != "" {
			home, err := expand(ctx.Home)
			if err != nil {
				return Target{}, err
			}
			return Target{Name: nc.Name, FS: fsx.Local{}, Home: home}, nil
		}
		home, err := userHomeDir()
		if err != nil {
			return Target{}, err
		}
		return Target{Name: nc.Name, FS: fsx.Local{}, Home: home}, nil
	}
	if config.Getenv("SSH") == "0" {
		return Target{}, exitcode.Errorf(exitcode.SSH, "ssh disabled (HCTL_SSH=0 or HARNESSCTL_SSH=0)")
	}
	sshTarget := ctx.Target()
	if sshTarget == "" {
		return Target{}, exitcode.Errorf(exitcode.SSH, "context %q: ssh target is empty", nc.Name)
	}
	s := fsx.SSH{Target: sshTarget, Identity: ctx.IdentityFile}
	home := ctx.Home
	if home == "" {
		var err error
		home, err = s.RemoteHome()
		if err != nil {
			return Target{}, err
		}
	}
	return Target{Name: nc.Name, FS: s, Home: home}, nil
}

func expand(p string) (string, error) {
	if !strings.HasPrefix(p, "~/") {
		return p, nil
	}
	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot expand %q: %w", p, err)
	}
	return strings.Replace(p, "~", home, 1), nil
}
