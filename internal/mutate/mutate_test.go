package mutate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/adapters/claude"
	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/adapters/grok"
	"github.com/oldwinter/hctl/internal/adapters/pi"
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
	path := filepath.Join(home, ".codex", "config.toml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
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
	snap, err := adapters.ReadOne(ad, fsx.Local{}, home)
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
	if !strings.Contains(string(data), "sk-test-aaa") {
		t.Fatal("lost bearer token (must stay in file, never printed)")
	}
	if len(rep.Backups) != 1 {
		t.Fatalf("expected one backup, got %d", len(rep.Backups))
	}
	backup, err := os.ReadFile(rep.Backups[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != string(before) {
		t.Fatal("backup does not contain the original config bytes")
	}
	if len(rep.Changes) != 1 || rep.Changes[0].Path != path {
		t.Fatalf("expected change path %q, got %#v", path, rep.Changes)
	}
}

func TestBackupFailureMustLeaveDestinationUnchanged(t *testing.T) {
	for _, tc := range []struct {
		name    string
		model   string
		dryRun  bool
		wantErr bool
	}{
		{name: "changed", model: "new", wantErr: true},
		{name: "dry-run", model: "new", dryRun: true},
		{name: "identical", model: "old"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(home, ".codex", "config.toml")
			before := []byte("model = \"old\"\n")
			if err := fsx.AtomicWrite(fsx.Local{}, path, before, 0o600); err != nil {
				t.Fatal(err)
			}
			blocker := filepath.Join(home, "backup-blocker")
			if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
				t.Fatal(err)
			}
			rep, err := Apply(Request{
				Adapter:   codex.Adapter{},
				FS:        fsx.Local{},
				Home:      home,
				BackupDir: filepath.Join(blocker, "backups"),
				Desired:   model.Desired{Model: tc.model},
				DryRun:    tc.dryRun,
			})
			if (err != nil) != tc.wantErr {
				t.Fatalf("Apply error = %v, want error = %v", err, tc.wantErr)
			}
			if len(rep.Backups) != 0 || rep.Verified {
				t.Fatalf("unexpected backup or verification: %#v", rep)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(before) {
				t.Fatal("destination changed without a backup")
			}
		})
	}
}

type failingRenameFS struct {
	fsx.Local
	path string
	err  error
}

func (f failingRenameFS) Rename(oldpath, newpath string) error {
	if newpath == f.path {
		return f.err
	}
	return f.Local.Rename(oldpath, newpath)
}

func TestApplyBacksUpBeforePartialWriteError(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	backupDir := t.TempDir()
	paths := []string{
		filepath.Join(home, ".pi", "agent", "settings.json"),
		filepath.Join(home, ".pi", "agent", "models.json"),
		filepath.Join(home, ".pi", "agent", "auth.json"),
	}
	before := make([][]byte, len(paths))
	for i, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		before[i] = data
	}
	writeErr := errors.New("injected settings rename failure")
	rep, err := Apply(Request{
		Adapter:   pi.Adapter{},
		FS:        failingRenameFS{path: paths[0], err: writeErr},
		Home:      home,
		BackupDir: backupDir,
		Desired:   model.Desired{Model: "new", SecretRef: "PI_TEST_KEY"},
	})
	if !errors.Is(err, writeErr) {
		t.Fatalf("expected write error, got %v", err)
	}
	if rep.Verified {
		t.Fatal("failed write must not be verified")
	}
	for i, path := range paths {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if changed := string(after) != string(before[i]); changed != (i == 2) {
			t.Fatalf("expected only auth.json to change, file %s changed = %v", filepath.Base(path), changed)
		}
	}
	if len(rep.Backups) != len(paths) {
		t.Fatalf("expected all %d originals backed up before the write error, got %d", len(paths), len(rep.Backups))
	}
	for i, path := range rep.Backups {
		backup, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(backup) != string(before[i]) {
			t.Fatalf("backup does not contain original %s bytes", filepath.Base(paths[i]))
		}
	}
}

func TestApplyIdempotentSkipsBackupAndWrite(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	bak := t.TempDir()
	ad := codex.Adapter{}
	req := Request{
		Adapter:   ad,
		FS:        fsx.Local{},
		Home:      home,
		BackupDir: bak,
		Desired:   model.Desired{Model: "o4-mini"},
	}
	if _, err := Apply(req); err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(bak)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) == 0 {
		t.Fatal("expected first apply to backup")
	}
	before, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Apply(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Changes) != 0 || len(rep.Backups) != 0 || rep.Verified {
		t.Fatalf("converged apply should no-op: %#v", rep)
	}
	after, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("idempotent apply rewrote config")
	}
	ents2, err := os.ReadDir(bak)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents2) != len(ents) {
		t.Fatalf("idempotent apply created extra backups: %d -> %d", len(ents), len(ents2))
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

func TestApplyClaudeBacksUpExistingConfigFiles(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	bak := t.TempDir()
	rep, err := Apply(Request{
		Adapter:   claude.Adapter{},
		FS:        fsx.Local{},
		Home:      home,
		BackupDir: bak,
		Desired:   model.Desired{Model: "claude-opus-4"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Backups) != 2 || !strings.Contains(rep.Backups[0], "settings.json") || !strings.Contains(rep.Backups[1], ".claude.json") {
		t.Fatalf("backups=%v", rep.Backups)
	}
}

func TestPreflightRejectsParseErrorBeforeBackupOrWrite(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	backupDir := t.TempDir()
	settingsPath := filepath.Join(home, ".pi", "agent", "settings.json")
	authPath := filepath.Join(home, ".pi", "agent", "auth.json")
	before, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authPath, []byte(`{"sub2api":`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = Apply(Request{
		Adapter:   pi.Adapter{},
		FS:        fsx.Local{},
		Home:      home,
		BackupDir: backupDir,
		Desired:   model.Desired{Model: "must-not-be-written"},
	})
	if err == nil || exitcode.From(err) != exitcode.Parse || strings.Contains(err.Error(), `{"sub2api":`) {
		t.Fatalf("expected sanitized parse error, got %v", err)
	}
	after, readErr := os.ReadFile(settingsPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(before) {
		t.Fatal("settings changed after parse-error preflight")
	}
	entries, readErr := os.ReadDir(backupDir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("preflight created backups: %v", entries)
	}
}
