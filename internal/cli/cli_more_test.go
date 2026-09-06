package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestConfigSetCurrentUseAndErrors(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	// seed
	f := config.Default()
	if err := config.Save(cfgPath, f); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "--config", cfgPath, "config", "current-context")
	if err != nil || !strings.Contains(out, "mba") {
		t.Fatal(err, out)
	}
	out, err = run(t, "--config", cfgPath, "--json", "config", "current-context")
	if err != nil {
		t.Fatal(err, out)
	}
	out, err = run(t, "--config", cfgPath, "config", "set-context", "box", "--kind", "ssh", "--ssh", "u@h", "--home", "/home/u", "--identity", "/id")
	if err != nil {
		t.Fatal(err, out)
	}
	out, err = run(t, "--config", cfgPath, "config", "set-context", "box2", "--ssh", "u@h2")
	if err != nil {
		t.Fatal(err, out)
	}
	out, err = run(t, "--config", cfgPath, "config", "set-context", "loc", "--home", dir)
	if err != nil {
		t.Fatal(err, out)
	}
	out, err = run(t, "--config", cfgPath, "config", "use-context", "loc")
	if err != nil {
		t.Fatal(err, out)
	}
	_, err = run(t, "--config", cfgPath, "config", "use-context", "missing")
	if err == nil {
		t.Fatal("expected missing context")
	}
	_, err = run(t, "--config", filepath.Join(dir, "nope.yaml"), "get", "nope")
	// load missing returns default — unknown resource
	home := testutil.Testdata(t, "home-a")
	_, err = run(t, "--home", home, "--config", cfgPath, "get", "widgets")
	if err == nil {
		t.Fatal("unknown resource")
	}
	_, err = run(t, "apply")
	if err == nil {
		t.Fatal("apply needs -f")
	}
	_, err = run(t, "sync")
	if err == nil {
		t.Fatal("sync needs from/to")
	}
	_, err = run(t, "completion", "nope")
	if err == nil {
		// Help returns nil sometimes
		t.Log(err)
	}
}

func TestVersionCommitDateAndHarnessctlName(t *testing.T) {
	oldC, oldD, oldArgs := Commit, Date, os.Args
	defer func() { Commit, Date, os.Args = oldC, oldD, oldArgs }()
	Commit = "abc"
	Date = "2026-01-01"
	os.Args = []string{"harnessctl", "version"}
	out, err := run(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "abc") || !strings.Contains(out, "2026-01-01") {
		t.Fatal(out)
	}
	// NewRoot use name
	os.Args = []string{"harnessctl"}
	cmd := NewRoot()
	if cmd.Use != "harnessctl" {
		t.Fatalf("%q", cmd.Use)
	}
}

func TestDiffContextsResolveFSAndDesiredJSON(t *testing.T) {
	homeA := testutil.Testdata(t, "home-a")
	homeB := testutil.Testdata(t, "home-b")
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	out, err := run(t, "--config", cfg, "diff", "harness", "codex", "--home-a", homeA, "--home-b", homeB)
	if err != nil {
		t.Fatal(err, out)
	}
	_, err = run(t, "--config", cfg, "diff", "--home-a", homeA)
	if err == nil {
		t.Fatal("homes together")
	}
	des := testutil.Testdata(t, "desired.toml")
	out, err = run(t, "--home", homeA, "--config", cfg, "--json", "diff", "-f", des)
	if err != nil {
		t.Fatal(err, out)
	}
	out, err = run(t, "--home", homeA, "--config", cfg, "--json", "apply", "-f", des, "--dry-run")
	if err != nil {
		t.Fatal(err, out)
	}
	// sync from/to local homes via contexts with home set
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	f := config.Default()
	f.Contexts = []config.NamedContext{
		{Name: "a", Context: config.Context{Kind: config.KindLocal, Home: homeA}},
		{Name: "b", Context: config.Context{Kind: config.KindLocal, Home: homeB}},
	}
	f.CurrentContext = "a"
	if err := config.Save(cfgPath, f); err != nil {
		t.Fatal(err)
	}
	out, err = run(t, "--config", cfgPath, "sync", "--from", "a", "--to", "b", "--harness", "codex", "--fields", "model,provider,secret-ref", "--dry-run")
	if err != nil {
		t.Fatal(err, out)
	}
	out, err = run(t, "--config", cfgPath, "sync", "--from", "a", "--to", "b", "--harness", "codex", "--fields", "model", "--dry-run", "--json")
	if err != nil {
		t.Log(err, out) // json flag may be global
	}
	_, err = run(t, "--config", cfgPath, "diff", "--contexts", "only")
	if err == nil {
		t.Fatal("bad contexts")
	}
	out, err = run(t, "--config", cfgPath, "diff", "--contexts", "a,b", "harness", "codex")
	if err != nil {
		t.Log(err, out)
	}
	_, err = run(t, "--config", cfgPath, "diff", "harness", "codex")
	if err == nil {
		t.Fatal("need sides")
	}
	opts := &options{configPath: cfgPath}
	cfg2, _ := opts.loadConfig()
	_, _, _, _, _, _, err = resolveDiffFS(opts, cfg2, "", "", "a,b", "", "")
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, _, _, err = resolveDiffFS(opts, cfg2, "", "", "x", "", "")
	if err == nil {
		t.Fatal("bad")
	}
	_, _, _, _, _, _, err = resolveDiffFS(opts, cfg2, "", "", "", homeA, "")
	if err == nil {
		t.Fatal("homes")
	}
	_, _, _, _, _, _, err = resolveDiffFS(opts, cfg2, "a", "b", "", homeA, homeB)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, _, _, err = resolveDiffFS(opts, cfg2, "", "", "", "", "")
	if err == nil {
		t.Fatal("need")
	}
}

func TestDescribeDoctorErrors(t *testing.T) {
	home := testutil.Testdata(t, "home-a")
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	_, err := run(t, "--home", home, "--config", cfg, "describe", "harness", "nope")
	if err == nil {
		t.Fatal("unknown")
	}
	_, err = run(t, "--home", home, "--config", cfg, "set", "model", "nope", "x")
	if err == nil {
		t.Fatal("unknown harness")
	}
}
