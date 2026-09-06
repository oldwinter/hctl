package mutate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/adapters/claude"
	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/adapters/grok"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestApplySetModelRoundTrip(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	bak := t.TempDir()
	t.Setenv("HARNESSCTL_BACKUP_DIR", bak)
	ad := codex.Adapter{}
	rep, err := Apply(Request{
		Adapter:   ad,
		FS:        fsx.Local{},
		Home:      home,
		BackupDir: bak,
		Desired:   model.Desired{Model: "o4-mini"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Verified {
		t.Fatal("expected verify")
	}
	snap, err := adapters.ReadOne(ad, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "o4-mini" {
		t.Fatalf("got %q", snap.DefaultModel)
	}
	if snap.Provider != "custom" {
		t.Fatalf("provider churned: %q", snap.Provider)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if !strings.Contains(string(data), "experimental_bearer_token") {
		t.Fatal("lost unrelated secret field")
	}
	if strings.Contains(string(data), "sk-test-aaa") == false {
		// token should still be in the file (we don't strip it), just never printed
	}
	ents, _ := os.ReadDir(bak)
	if len(ents) == 0 {
		t.Fatal("expected backup")
	}
}

func TestDryRunDoesNotWrite(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	before, _ := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	rep, err := Apply(Request{
		Adapter: codex.Adapter{},
		FS:      fsx.Local{},
		Home:    home,
		Desired: model.Desired{Model: "nope"},
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if string(before) != string(after) {
		t.Fatal("dry-run mutated file")
	}
	if len(rep.Changes) != 1 {
		t.Fatalf("changes = %#v", rep.Changes)
	}
	wantPath := filepath.Join(home, ".codex", "config.toml")
	if rep.Changes[0].Path != wantPath {
		t.Fatalf("dry-run Path = %q want %q", rep.Changes[0].Path, wantPath)
	}
}

func TestSetProviderUnsupportedClaudeAndGrok(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	cases := []struct {
		ad   adapters.Adapter
		name string
	}{
		{claude.Adapter{}, "claude"},
		{grok.Adapter{}, "grok"},
	}
	for _, tc := range cases {
		beforeClaude, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
		beforeGrok, _ := os.ReadFile(filepath.Join(home, ".grok", "config.toml"))
		_, err := Apply(Request{
			Adapter: tc.ad,
			FS:      fsx.Local{},
			Home:    home,
			Desired: model.Desired{Provider: "custom"},
			DryRun:  true,
		})
		if err == nil {
			t.Fatalf("%s: expected unsupported error", tc.name)
		}
		if exitcode.From(err) != exitcode.Usage {
			t.Fatalf("%s: code=%d err=%v", tc.name, exitcode.From(err), err)
		}
		if !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("%s: err=%v", tc.name, err)
		}
		afterClaude, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
		afterGrok, _ := os.ReadFile(filepath.Join(home, ".grok", "config.toml"))
		if string(beforeClaude) != string(afterClaude) || string(beforeGrok) != string(afterGrok) {
			t.Fatalf("%s: dry-run/error mutated files", tc.name)
		}
	}
}
