package cli

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/remote"
	"github.com/oldwinter/hctl/internal/testutil"
)

type errWriter struct{}

func (errWriter) Write(p []byte) (int, error) { return 0, errors.New("write boom") }

func runOut(t *testing.T, w interface{ Write([]byte) (int, error) }, args ...string) error {
	t.Helper()
	cmd := NewRoot()
	cmd.SetOut(w)
	cmd.SetErr(w)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func TestRenderAndSaveFailures(t *testing.T) {
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	home := testutil.Testdata(t, "home-a")
	if err := runOut(t, errWriter{}, "--home", home, "--config", cfg, "--json", "doctor"); err == nil {
		t.Fatal("doctor json")
	}
	if err := runOut(t, errWriter{}, "--home", home, "--config", cfg, "--json", "describe", "harness", "codex"); err == nil {
		t.Fatal("describe json")
	}
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	config.Save(cfgPath, config.Default())
	config.SetSaveOverride(func(string, *config.File) error { return errors.New("save boom") })
	defer config.SetSaveOverride(nil)
	if _, err := run(t, "--config", cfgPath, "config", "use-context", "mba"); err == nil {
		t.Fatal("use save")
	}
	if _, err := run(t, "--config", cfgPath, "config", "set-context", "mba", "--home", dir); err == nil {
		t.Fatal("set save")
	}
	// sync ReadOneFS / Apply errors via Dial boom after load
	config.SetSaveOverride(nil)
	old := remote.Dial
	defer func() { remote.Dial = old }()
	remote.Dial = func(nc config.NamedContext, homeFlag string) (fsx.FS, string, error) {
		return nil, "", errors.New("dial")
	}
	f := config.Default()
	f.Contexts = append(f.Contexts, config.NamedContext{Name: "b", Context: config.Context{Kind: config.KindLocal, Home: home}})
	cfg2 := filepath.Join(dir, "c2.yaml")
	config.Save(cfg2, f)
	if _, err := run(t, "--config", cfg2, "sync", "--from", "mba", "--to", "b"); err == nil {
		t.Fatal("sync dial")
	}
	if _, err := run(t, "--config", cfg2, "diff", "--contexts", "mba,b", "harness", "codex"); err == nil {
		t.Fatal("diff dial")
	}
}

type boomFS struct{ fsx.Local }

func (boomFS) ReadFile(name string) ([]byte, error) { return nil, errors.New("boom") }

func TestDialBoomFSErrors(t *testing.T) {
	old := remote.Dial
	defer func() { remote.Dial = old }()
	home := testutil.Testdata(t, "home-a")
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	remote.Dial = func(nc config.NamedContext, homeFlag string) (fsx.FS, string, error) {
		h := homeFlag
		if h == "" {
			h = home
		}
		return boomFS{}, h, nil
	}
	if _, err := run(t, "--home", home, "--config", cfg, "describe", "harness", "codex"); err == nil {
		t.Fatal("describe")
	}
	if _, err := run(t, "--home", home, "--config", cfg, "apply", "-f", testutil.Testdata(t, "desired.toml"), "--dry-run"); err == nil {
		t.Fatal("apply")
	}
	if _, err := run(t, "--home", home, "--config", cfg, "diff", "-f", testutil.Testdata(t, "desired.toml")); err == nil {
		t.Fatal("diff desired")
	}
	dir := t.TempDir()
	f := config.Default()
	f.Contexts = []config.NamedContext{
		{Name: "a", Context: config.Context{Kind: config.KindLocal, Home: home}},
		{Name: "b", Context: config.Context{Kind: config.KindLocal, Home: home}},
	}
	f.CurrentContext = "a"
	cfg2 := filepath.Join(dir, "c.yaml")
	config.Save(cfg2, f)
	if _, err := run(t, "--config", cfg2, "sync", "--from", "a", "--to", "b", "--harness", "codex", "--dry-run"); err == nil {
		t.Fatal("sync")
	}
	if _, err := run(t, "--config", cfg2, "diff", "--contexts", "a,b", "harness", "codex"); err == nil {
		t.Fatal("diff ctx")
	}
}
