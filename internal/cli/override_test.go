package cli

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestLoadConfigOverrideErrors(t *testing.T) {
	defer func() { loadConfigOverride = nil }()
	loadConfigOverride = func(o *options) (*config.File, error) {
		return nil, errors.New("load boom")
	}
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	home := testutil.Testdata(t, "home-a")
	cmds := [][]string{
		{"--config", cfg, "config", "get-contexts"},
		{"--config", cfg, "config", "current-context"},
		{"--config", cfg, "config", "set-context", "x"},
		{"--config", cfg, "config", "use-context", "mba"},
		{"--home", home, "--config", cfg, "get", "harnesses"},
		{"--home", home, "--config", cfg, "doctor"},
		{"--home", home, "--config", cfg, "describe", "harness", "codex"},
		{"--home", home, "--config", cfg, "set", "model", "codex", "m"},
		{"--home", home, "--config", cfg, "apply", "-f", testutil.Testdata(t, "desired.toml")},
		{"--config", cfg, "diff", "--home-a", home, "--home-b", home, "harness", "codex"},
		{"--config", cfg, "sync", "--from", "mba", "--to", "mba"},
	}
	for _, args := range cmds {
		_, err := run(t, args...)
		if err == nil {
			t.Fatalf("expected err for %v", args)
		}
	}
	// set-context update existing + save ok path without override
	loadConfigOverride = nil
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	config.Save(cfgPath, config.Default())
	out, err := run(t, "--config", cfgPath, "config", "set-context", "mba", "--home", dir)
	if err != nil {
		t.Fatal(err, out)
	}
}
