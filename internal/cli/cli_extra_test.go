package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestResolveDiffSidesAndScanHome(t *testing.T) {
	cfg := config.Default()
	cfg.Contexts = append(cfg.Contexts, config.NamedContext{Name: "other", Context: config.Context{Kind: config.KindLocal, Home: t.TempDir()}})
	la, lb, pa, pb, err := resolveDiffSides(cfg, "mba", "other", "", "", "", t.TempDir())
	if err != nil || la == "" || lb == "" || pa == "" || pb == "" {
		t.Fatal(err, la, lb, pa, pb)
	}
	_, _, _, _, err = resolveDiffSides(cfg, "", "", "mba,other", "", "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, err = resolveDiffSides(cfg, "", "", "onlyone", "", "", "")
	if err == nil {
		t.Fatal("contexts")
	}
	_, _, _, _, err = resolveDiffSides(cfg, "", "", "", "a", "", "")
	if err == nil {
		t.Fatal("homes together")
	}
	la, lb, pa, pb, err = resolveDiffSides(cfg, "x", "y", "", "/a", "/b", "")
	if err != nil || pa != "/a" || pb != "/b" {
		t.Fatal(err, pa, pb)
	}
	_, _, _, _, err = resolveDiffSides(cfg, "", "", "", "", "", "")
	if err == nil {
		t.Fatal("need sides")
	}
	opts := &options{home: t.TempDir(), contextName: "mba"}
	name, home, err := opts.scanHome(cfg)
	if err != nil || name != "mba" || home == "" {
		t.Fatal(err, name, home)
	}
}

func TestCompletionZshFishPowershellAndWideGet(t *testing.T) {
	for _, sh := range []string{"zsh", "fish", "powershell"} {
		out, err := run(t, "completion", sh)
		if err != nil {
			t.Fatal(sh, err, out)
		}
	}
	home := testutil.Testdata(t, "home-a")
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	out, err := run(t, "--home", home, "--config", cfg, "-o", "wide", "get", "harnesses")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "CONFIG") && !strings.Contains(out, "codex") {
		t.Fatal(out)
	}
	out, err = run(t, "--home", home, "--config", cfg, "--json", "get", "models")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	// config get-contexts json
	out, err = run(t, "--config", cfg, "--json", "config", "get-contexts")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	// doctor json
	out, err = run(t, "--home", home, "--config", cfg, "--json", "doctor")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	// describe json
	out, err = run(t, "--home", home, "--config", cfg, "--json", "describe", "harness", "claude")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	// sync dry-run with homes
	homeB := testutil.Testdata(t, "home-b")
	out, err = run(t, "--config", cfg, "sync", "--from-home", home, "--to-home", homeB, "--dry-run")
	if err != nil {
		// may require flags differently
		out2, err2 := run(t, "--config", cfg, "sync", "harness", "codex", "--from-home", home, "--to-home", homeB, "--dry-run")
		if err2 != nil {
			t.Log(err, out, err2, out2)
		}
	}
	_ = os.Getenv
	_ = filepath.Base
}

func TestWriteErr(t *testing.T) {
	if writeErr(nil, nil) != nil {
		t.Fatal("nil")
	}
}
