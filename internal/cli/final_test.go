package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/remote"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestFinalCLIGaps(t *testing.T) {
	if runtime.GOOS != "windows" {
		dir := t.TempDir()
		bad := filepath.Join(dir, "bad.yaml")
		os.WriteFile(bad, []byte("x: 1\n"), 0o000)
		defer os.Chmod(bad, 0o600)
		_, _ = run(t, "--config", bad, "config", "current-context")
		_, _ = run(t, "--config", bad, "config", "get-contexts")
		_, _ = run(t, "--config", bad, "config", "set-context", "x")
		_, _ = run(t, "--config", bad, "config", "use-context", "mba")
	}
	if runtime.GOOS != "windows" {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "c.yaml")
		config.Save(cfgPath, config.Default())
		os.Chmod(dir, 0o555)
		defer os.Chmod(dir, 0o755)
		_, _ = run(t, "--config", cfgPath, "config", "use-context", "mba")
		_, _ = run(t, "--config", cfgPath, "config", "set-context", "z", "--home", dir)
	}
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	old := remote.Dial
	defer func() { remote.Dial = old }()
	remote.Dial = func(nc config.NamedContext, homeFlag string) (fsx.FS, string, error) {
		return fsx.Local{}, "", nil
	}
	if _, err := run(t, "--config", cfg, "get", "harnesses"); err == nil {
		t.Fatal("empty home scan")
	}
	if _, err := run(t, "--config", cfg, "doctor"); err == nil {
		t.Fatal("doctor scan")
	}
	_, _ = run(t, "--config", cfg, "describe", "harness", "codex")
	remote.Dial = old
	if _, err := run(t, "--home", testutil.Testdata(t, "home-a"), "--config", cfg, "diff", "-f", filepath.Join(t.TempDir(), "no.toml")); err == nil {
		t.Fatal("diff file")
	}
	if runtime.GOOS != "windows" {
		bad := filepath.Join(t.TempDir(), "b.yaml")
		os.WriteFile(bad, []byte("x"), 0o000)
		defer os.Chmod(bad, 0o600)
		opts := &options{configPath: bad}
		_, _, _, _ = opts.openTarget()
	}
}

func TestApplyUnknownAndMutateFail(t *testing.T) {
	dir := t.TempDir()
	des := filepath.Join(dir, "d.toml")
	os.WriteFile(des, []byte("harnesses = { nope = { model = \"m\" } }\n"), 0o600)
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	if _, err := run(t, "--home", home, "--config", cfg, "apply", "-f", des, "--dry-run"); err == nil {
		t.Fatal("unknown harness")
	}
	des2 := filepath.Join(dir, "d2.toml")
	os.WriteFile(des2, []byte("harnesses = { claude = { provider = \"x\" } }\n"), 0o600)
	if _, err := run(t, "--home", home, "--config", cfg, "apply", "-f", des2, "--dry-run"); err == nil {
		t.Fatal("claude provider")
	}
	// doctor JSON + parse error exit already
	_, _ = run(t, "--home", home, "--config", cfg, "--json", "doctor")
}
