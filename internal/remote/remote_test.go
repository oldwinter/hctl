package remote

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
)

func named(name string, ctx config.Context) config.NamedContext {
	return config.NamedContext{Name: name, Context: ctx}
}

func TestDialHomeFlagWins(t *testing.T) {
	home := t.TempDir()
	tgt, err := Dial(named("box", config.Context{
		Kind: config.KindSSH,
		SSH:  "u@h",
	}), home)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tgt.FS.(fsx.Local); !ok {
		t.Fatalf("FS = %T, want fsx.Local", tgt.FS)
	}
	if tgt.Home != home {
		t.Fatalf("Home = %q, want %q", tgt.Home, home)
	}
}

func TestDialKindInference(t *testing.T) {
	t.Setenv("HCTL_SSH", "1")
	for _, tc := range []struct {
		name string
		ctx  config.Context
		ssh  bool
		want string
	}{
		{"ssh field", config.Context{SSH: "u@h", Home: "/r"}, true, "u@h"},
		{"host only", config.Context{Host: "h", Home: "/r"}, true, "h"},
		{"user+host", config.Context{User: "u", Host: "h", Home: "/r"}, true, "u@h"},
		{"neither", config.Context{Home: "/r"}, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tgt, err := Dial(named("c", tc.ctx), "")
			if err != nil {
				t.Fatal(err)
			}
			if tc.ssh {
				s, ok := tgt.FS.(fsx.SSH)
				if !ok {
					t.Fatalf("FS = %T, want fsx.SSH", tgt.FS)
				}
				if s.Target != tc.want {
					t.Fatalf("Target = %q, want %q", s.Target, tc.want)
				}
			} else if _, ok := tgt.FS.(fsx.Local); !ok {
				t.Fatalf("FS = %T, want fsx.Local", tgt.FS)
			}
			if tgt.Home != "/r" {
				t.Fatalf("Home = %q", tgt.Home)
			}
		})
	}
}

func TestDialUnknownKind(t *testing.T) {
	for _, kind := range []string{"bogus", "SSH", "ssh ", " local"} {
		_, err := Dial(named("c", config.Context{Kind: kind, Home: "/r"}), "")
		if exitcode.From(err) != exitcode.Usage {
			t.Fatalf("kind %q: err = %v (code %d), want usage error", kind, err, exitcode.From(err))
		}
	}
}

func TestDialLocalTildeHome(t *testing.T) {
	real, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no user home")
	}
	tgt, err := Dial(named("mba", config.Context{Kind: config.KindLocal, Home: "~/x"}), "")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(real, "x"); tgt.Home != want {
		t.Fatalf("Home = %q, want %q", tgt.Home, want)
	}
}

func TestDialSSHDisabled(t *testing.T) {
	t.Setenv("HCTL_SSH", "0")
	_, err := Dial(named("box", config.Context{
		Kind: config.KindSSH,
		SSH:  "u@h",
		Home: "/r",
	}), "")
	if exitcode.From(err) != exitcode.SSH {
		t.Fatalf("err = %v (code %d), want ssh error", err, exitcode.From(err))
	}
}

func TestDialSSHEmptyTarget(t *testing.T) {
	t.Setenv("HCTL_SSH", "1")
	_, err := Dial(named("box", config.Context{
		Kind: config.KindSSH,
		Home: "/r",
	}), "")
	if exitcode.From(err) != exitcode.SSH {
		t.Fatalf("err = %v (code %d), want ssh error", err, exitcode.From(err))
	}
}

func TestDialSSHUnsafeTarget(t *testing.T) {
	t.Setenv("HCTL_SSH", "1")
	for _, target := range []string{"-oProxyCommand=x", "-F", "u@h extra", "u@h\tx"} {
		_, err := Dial(named("box", config.Context{
			Kind: config.KindSSH,
			SSH:  target,
			Home: "/r",
		}), "")
		if exitcode.From(err) != exitcode.Usage {
			t.Fatalf("target %q: err = %v (code %d), want usage error", target, err, exitcode.From(err))
		}
	}
}

func TestDialSSHTildeHome(t *testing.T) {
	t.Setenv("HCTL_SSH", "1")
	_, err := Dial(named("box", config.Context{
		Kind: config.KindSSH,
		SSH:  "u@h",
		Home: "~/x",
	}), "")
	if exitcode.From(err) != exitcode.Usage {
		t.Fatalf("err = %v (code %d), want usage error", err, exitcode.From(err))
	}
}

func TestDialRejectsRelativeSSHHome(t *testing.T) {
	t.Setenv("HCTL_SSH", "1")
	_, err := Dial(named("box", config.Context{
		Kind: config.KindSSH,
		SSH:  "u@h",
		Home: "relative/home",
	}), "")
	if exitcode.From(err) != exitcode.Usage {
		t.Fatalf("err = %v (code %d), want usage error", err, exitcode.From(err))
	}
}
