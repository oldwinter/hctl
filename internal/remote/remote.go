package remote

import (
	"os"
	"strings"

	"github.com/oldwinter/harnessctl/internal/config"
	"github.com/oldwinter/harnessctl/internal/exitcode"
	"github.com/oldwinter/harnessctl/internal/fsx"
)

// Dial opens a filesystem for a context. Tests may replace this.
var Dial = DefaultDial

// DefaultDial returns a local FS, or an OpenSSH-backed FS for kind=ssh.
func DefaultDial(nc config.NamedContext, homeFlag string) (fsx.FS, string, error) {
	if homeFlag != "" {
		return fsx.Local{}, homeFlag, nil
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
			return fsx.Local{}, expand(ctx.Home), nil
		}
		home, err := os.UserHomeDir()
		return fsx.Local{}, home, err
	}
	if os.Getenv("HARNESSCTL_SSH") == "0" {
		return nil, "", exitcode.Errorf(exitcode.SSH, "ssh disabled (HARNESSCTL_SSH=0)")
	}
	target := ctx.Target()
	if target == "" {
		return nil, "", exitcode.Errorf(exitcode.SSH, "context %q: ssh target is empty", nc.Name)
	}
	s := fsx.SSH{Target: target, Identity: ctx.IdentityFile}
	home := ctx.Home
	if home == "" {
		var err error
		home, err = s.RemoteHome()
		if err != nil {
			return nil, "", err
		}
	}
	return s, home, nil
}

func expand(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return strings.Replace(p, "~", home, 1)
		}
	}
	return p
}
