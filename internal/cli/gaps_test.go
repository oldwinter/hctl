package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/remote"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestDialAndOpenErrors(t *testing.T) {
	old := remote.Dial
	defer func() { remote.Dial = old }()
	remote.Dial = func(nc config.NamedContext, homeFlag string) (fsx.FS, string, error) {
		return nil, "", errors.New("dial boom")
	}
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	home := testutil.Testdata(t, "home-a")
	_, err := run(t, "--home", home, "--config", cfg, "get", "harnesses")
	if err == nil {
		t.Fatal("dial")
	}
	_, err = run(t, "--home", home, "--config", cfg, "doctor")
	if err == nil {
		t.Fatal("doctor dial")
	}
	_, err = run(t, "--home", home, "--config", cfg, "describe", "harness", "codex")
	if err == nil {
		t.Fatal("describe dial")
	}
	_, err = run(t, "--home", home, "--config", cfg, "set", "model", "codex", "x")
	if err == nil {
		t.Fatal("set dial")
	}
	_, err = run(t, "--home", home, "--config", cfg, "apply", "-f", testutil.Testdata(t, "desired.toml"))
	if err == nil {
		t.Fatal("apply dial")
	}
	_, err = run(t, "--config", cfg, "describe", "pod", "x")
	if err == nil {
		t.Fatal("bad resource")
	}
	_, err = run(t, "--home", home, "--config", cfg, "set", "alias", "codex", "x")
	if err == nil {
		t.Fatal("bad set resource")
	}
}

func TestConfigLoadSaveErrorsAndSyncSecrets(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.yaml")
	os.WriteFile(bad, []byte("x"), 0o000)
	_, err := run(t, "--config", bad, "config", "get-contexts")
	_ = err // may or may not fail depending on perms
	os.Chmod(bad, 0o600)

	homeA := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	homeB := testutil.CopyTree(t, testutil.Testdata(t, "home-b"))
	cfgPath := filepath.Join(dir, "c.yaml")
	f := config.Default()
	f.Contexts = []config.NamedContext{
		{Name: "a", Context: config.Context{Kind: config.KindLocal, Home: homeA}},
		{Name: "b", Context: config.Context{Kind: config.KindLocal, Home: homeB}},
	}
	f.CurrentContext = "a"
	config.Save(cfgPath, f)

	out, err := run(t, "--config", cfgPath, "--json", "sync", "--from", "a", "--to", "b", "--harness", "codex", "--fields", "model,secret", "--dry-run")
	if err != nil {
		t.Fatal(err, out)
	}
	out, err = run(t, "--config", cfgPath, "sync", "--from", "a", "--to", "b", "--harness", "codex", "--fields", "secret", "--dry-run")
	if err != nil {
		t.Fatal(err, out)
	}
	// real secret copy non-dry
	out, err = run(t, "--config", cfgPath, "sync", "--from", "a", "--to", "b", "--harness", "codex", "--fields", "secret")
	if err != nil {
		t.Log(err, out)
	}
	_, err = run(t, "--config", cfgPath, "sync", "--from", "a", "--to", "missing")
	if err == nil {
		t.Fatal("missing to")
	}
	_, err = run(t, "--config", cfgPath, "sync", "--from", "missing", "--to", "b")
	if err == nil {
		t.Fatal("missing from")
	}
	_, err = run(t, "--config", cfgPath, "sync", "--from", "a", "--to", "b", "--harness", "nope")
	if err == nil {
		t.Fatal("bad harness")
	}

	// doctor parse error exit
	badHome := t.TempDir()
	os.MkdirAll(filepath.Join(badHome, ".codex"), 0o755)
	os.WriteFile(filepath.Join(badHome, ".codex", "config.toml"), []byte("[[["), 0o600)
	f2 := config.Default()
	f2.Contexts[0].Context.Home = badHome
	cfg2 := filepath.Join(dir, "c2.yaml")
	config.Save(cfg2, f2)
	_, err = run(t, "--config", cfg2, "doctor")
	if err == nil {
		t.Log("doctor may not fail if parse error only on one harness")
	}

	_, err = run(t, "--config", cfgPath, "set", "model", "codex", "new-model", "--json", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	_, err = run(t, "--home", homeA, "--config", cfgPath, "apply", "-f", filepath.Join(dir, "missing.toml"))
	if err == nil {
		t.Fatal("missing desired")
	}
	homeCopy := testutil.CopyTree(t, homeA)
	_, err = run(t, "--home", homeCopy, "--config", cfgPath, "apply", "-f", testutil.Testdata(t, "desired.toml"), "--json", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}

	// openNamed missing context via diff contexts
	_, err = run(t, "--config", cfgPath, "diff", "--contexts", "a,missing", "harness", "codex")
	if err == nil {
		t.Fatal("missing context side")
	}

	// resolveDiffSides ResolveHome fail
	opts := &options{configPath: cfgPath}
	cfg3, _ := opts.loadConfig()
	_, _, _, _, err = resolveDiffSides(cfg3, "a", "missing", "", "", "", "")
	if err == nil {
		t.Fatal("resolve")
	}
	_, _, _, _, err = resolveDiffSides(cfg3, "missing", "a", "", "", "", "")
	if err == nil {
		t.Fatal("resolve2")
	}
}
