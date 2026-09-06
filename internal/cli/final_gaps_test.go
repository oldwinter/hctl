package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestApplyEmptyHarnessContinue(t *testing.T) {
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	dir := t.TempDir()
	des := filepath.Join(dir, "d.toml")
	os.WriteFile(des, []byte("apiVersion = \"harnessctl/v1\"\n[harnesses.codex]\n[harnesses.claude]\nmodel = \"m\"\n"), 0o600)
	if _, err := run(t, "--home", home, "--config", cfg, "apply", "-f", des, "--dry-run"); err != nil {
		t.Fatal(err)
	}
}

func TestDiffErrorBranchesAndJSON(t *testing.T) {
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	home := testutil.Testdata(t, "home-a")
	homeB := testutil.Testdata(t, "home-b")
	if _, err := run(t, "--config", cfg, "diff", "--home-a", home, "--home-b", homeB, "pods", "codex"); err == nil {
		t.Fatal("unknown resource")
	}
	if _, err := run(t, "--config", cfg, "diff", "--home-a", home, "--home-b", homeB, "harness", "no-such"); err == nil {
		t.Fatal("byname")
	}
	out, err := run(t, "--config", cfg, "--json", "diff", "--home-a", home, "--home-b", homeB, "harness", "codex")
	if err != nil {
		t.Fatal(err, out)
	}
	loadConfigOverride = func(o *options) (*config.File, error) { return nil, errors.New("boom") }
	defer func() { loadConfigOverride = nil }()
	des := testutil.Testdata(t, "desired.toml")
	if _, err := run(t, "--config", cfg, "diff", "-f", des); err == nil {
		t.Fatal("openTarget")
	}
}

func TestDoctorTableWriteErr(t *testing.T) {
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	home := testutil.Testdata(t, "home-a")
	if err := runOut(t, errWriter{}, "--home", home, "--config", cfg, "doctor"); err == nil {
		t.Fatal("doctor table")
	}
}

func TestSyncFieldEmptyDefaultAndPrintRef(t *testing.T) {
	dir := t.TempDir()
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
	if _, err := run(t, "--config", cfgPath, "sync", "--from", "a", "--to", "b", "--harness", "codex", "--fields="); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "--config", cfgPath, "sync", "--from", "a", "--to", "b", "--harness", "codex", "--fields", "secret")
	t.Log(out, err)
}

func TestSyncApplyReportWriteErr(t *testing.T) {
	dir := t.TempDir()
	homeA := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	homeB := filepath.Join(dir, "hb")
	os.MkdirAll(homeB, 0o755)
	cfgPath := filepath.Join(dir, "c.yaml")
	f := config.Default()
	f.Contexts = []config.NamedContext{
		{Name: "a", Context: config.Context{Kind: config.KindLocal, Home: homeA}},
		{Name: "b", Context: config.Context{Kind: config.KindLocal, Home: homeB}},
	}
	f.CurrentContext = "a"
	config.Save(cfgPath, f)
	if err := runOut(t, errWriter{}, "--config", cfgPath, "sync", "--from", "a", "--to", "b", "--harness", "codex", "--fields", "model", "--dry-run"); err == nil {
		t.Fatal("apply report")
	}
}
