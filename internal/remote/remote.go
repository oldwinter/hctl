package remote

import (
	"os"
	"strings"
	"unicode"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
)

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
	switch ctx.Kind {
	case config.KindLocal:
		if ctx.Home != "" {
			return Target{Name: nc.Name, FS: fsx.Local{}, Home: expand(ctx.Home)}, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return Target{}, err
		}
		return Target{Name: nc.Name, FS: fsx.Local{}, Home: home}, nil
	case config.KindSSH:
	default:
		return Target{}, exitcode.Errorf(exitcode.Usage, "context %q: unknown kind %q (want local|ssh)", nc.Name, ctx.Kind)
	}
	if config.Getenv("SSH") == "0" {
		return Target{}, exitcode.Errorf(exitcode.SSH, "ssh disabled (HCTL_SSH=0 or HARNESSCTL_SSH=0)")
	}
	sshTarget := ctx.Target()
	if sshTarget == "" {
		return Target{}, exitcode.Errorf(exitcode.SSH, "context %q: ssh target is empty", nc.Name)
	}
	if strings.HasPrefix(sshTarget, "-") || strings.IndexFunc(sshTarget, unicode.IsSpace) >= 0 {
		return Target{}, exitcode.Errorf(exitcode.Usage, "context %q: unsafe ssh target %q (want [user@]host)", nc.Name, sshTarget)
	}
	if strings.HasPrefix(ctx.Home, "~") {
		return Target{}, exitcode.Errorf(exitcode.Usage, "context %q: ssh home %q is not absolute (remote ~ expansion is unsupported)", nc.Name, ctx.Home)
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

func expand(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return strings.Replace(p, "~", home, 1)
		}
	}
	return p
}
